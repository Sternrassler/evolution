# ADR-003: `atomic.Pointer` + 2-Buffer-Pool für Update/Draw-Synchronisation

- **Datum:** 2026-03-17
- **Status:** Accepted

---

## Kontext

Ebiten ruft `Game.Update()` und `Game.Draw()` in **derselben Goroutine** auf,
jedoch in separaten Phasen des Game-Loop-Ticks. Die Simulation (`sim.Step()`)
läuft in `Update()`, das Rendering liest den `WorldSnapshot` in `Draw()`.

Das Problem entsteht nicht durch parallele Goroutinen, sondern durch die
Anforderung, dass `Draw()` einen **konsistenten, vollständigen Snapshot** sehen
soll — und nicht ein halb-geschriebenes Ergebnis eines laufenden `Step()`.

Zusätzlich: `Draw()` wird bei 60 FPS aufgerufen, `Step()` nur bei 20 TPS.
`Draw()` soll nicht blockieren, wenn kein neuer Tick vorliegt.

Anforderungen:
1. `Draw()` liest immer einen vollständigen, konsistenten Snapshot
2. Kein Mutex in `Draw()` (lock-frei, kein Frame-Drop-Risiko)
3. Kein GC-Druck durch neue Snapshot-Allokationen pro Tick
4. `Step()` kann den nächsten Snapshot schreiben, während `Draw()` den letzten liest

---

## Entscheidung

**`atomic.Pointer[WorldSnapshot]` + 2-Buffer-Pool:**

```go
type SnapshotExporter struct {
    pool     [2]WorldSnapshot
    current  atomic.Pointer[WorldSnapshot]
    writeIdx int  // nur von Step() (Update-Goroutine) beschrieben
}
```

- `Step()` befüllt immer `pool[1-writeIdx]` (den "inaktiven" Buffer)
- Nach vollständiger Befüllung: `current.Store(&pool[1-writeIdx])`
- `writeIdx` wird getauscht
- `Draw()` ruft `current.Load()` — lock-frei, kein Mutex

**Happens-Before-Garantie:** Go Memory Model garantiert, dass alle Schreibvorgänge
vor `atomic.Store()` nach dem zugehörigen `atomic.Load()` sichtbar sind.
Kein manueller Memory-Fence nötig.

**Dirty-Flag in `Game.Draw()`:**
```go
snap := g.exporter.Load()
if snap != nil && snap.Tick != g.lastTick {
    g.renderer.RenderToBuffer(snap)  // nur bei neuem Tick
    g.lastTick = snap.Tick
}
g.renderer.DrawBuffer(screen)  // immer
```
Spart ~66% redundante `WritePixels`-Calls bei 60 FPS / 20 TPS.

**Slices im WorldSnapshot** werden mit `cap = MaxPopulation` einmalig allokiert.
Pro Tick nur `len`-Anpassung — kein GC-Druck.

---

## Konsequenzen

**Positiv:**
- `Draw()` blockiert nie auf einem Mutex — keine Frame-Drops durch Simulation
- Zero-GC pro Tick (keine neuen Snapshot-Allokationen)
- Happens-Before durch Go Memory Model formal garantiert — kein undefiniertes Verhalten
- Einfach zu verstehen: zwei Buffer, immer in den inaktiven schreiben

**Negativ:**
- Maximal 1 Tick Latenz zwischen Simulation und Rendering
  (Draw sieht ggf. einen Tick alten Snapshot — bei 20 TPS: ~50ms, nicht wahrnehmbar)
- `writeIdx` darf nur von einer Goroutine (Update) beschrieben werden —
  implizite Regel, nicht durch das Typsystem erzwungen
- 2 × `WorldSnapshot` im Heap dauerhaft allokiert (cap = MaxPopulation)

**Invariante die eingehalten werden muss:**
- Nach `atomic.Store()` darf `pool[writeIdx]` (der gerade gespeicherte Buffer)
  **nicht mehr mutiert werden**, bis er beim nächsten Überschreiben wieder
  der "inaktive" Buffer ist. Verletzung führt zu sichtbarem Tearing in `Draw()`.

---

## Verworfene Alternativen

### A: `sync.Mutex` in `SnapshotExporter`

Einfachste Lösung. `Draw()` lockt, liest Snapshot, unlockt.
Problem: Bei langen `Step()`-Calls (Spike durch großes Event in Phase 2)
blockiert `Draw()` — Frame-Drop. Ebiten ist frame-sensitiv.
Abgelehnt wegen latency jitter.

### B: `chan *WorldSnapshot` (unbuffered)

`Step()` sendet, `Draw()` empfängt. Zu starke Kopplung: wenn `Draw()` nicht
rechtzeitig empfängt, blockiert `Step()`. Channel-Semantik falsch für diesen
Anwendungsfall (1 Producer, 1 Consumer, aber nicht synchron).

### C: Snapshot direkt auf `Game`-Struct (kein Exporter)

`Game` hält `lastSnap WorldSnapshot` als Feld. `Step()` schreibt direkt.
Problem: Kein Schutz vor halb-geschriebenem State. Da Update/Draw in
derselben Goroutine laufen, ist das tatsächlich sicher — aber der Exporter
kapselt die Sync-Logik und macht sie explizit testbar. Bei zukünftiger
Parallelisierung wäre ein nacktes Feld sofort falsch.

### D: `sync/atomic.Value`

Vorgänger von `atomic.Pointer`, erfordert Type-Assertion, weniger typsicher.
`atomic.Pointer[T]` (seit Go 1.19) ist der idiomatische Ersatz.
