# ADR-008: Tick-Loop-Steuerung — synchrones Step() in Update()

- **Datum:** 2026-03-17
- **Status:** Accepted

---

## Kontext

Die Simulation muss steuerbar sein:

- **Geschwindigkeit:** 0–60 TPS (Ticks per Second), konfigurierbar zur Laufzeit
- **Pause / Weiter:** Simulation einfrieren, Rendering läuft weiter (letzter Snapshot bleibt sichtbar)
- **Next Step:** Bei Geschwindigkeit 0 einen einzelnen Tick manuell auslösen
- **Deterministik:** Tick-Sequenz darf nicht von Frame-Rate abhängen

Zwei grundsätzliche Architekturansätze wurden diskutiert:

**A — Synchron in `Update()`:** `sim.Step()` wird direkt in `Game.Update()` aufgerufen.
Ebiten steuert die Aufrufrate via `ebiten.SetTPS()`.

**B — Eigene Goroutine:** Eine dedizierte `sim`-Goroutine läuft in ihrer eigenen Schleife
mit `time.Ticker`. `Update()` und `Draw()` kommunizieren mit ihr via Channels oder
`atomic.Pointer`.

---

## Entscheidung

**`sim.Step()` wird synchron in `Game.Update()` aufgerufen. Keine separate Sim-Goroutine.**

```go
func (g *Game) Update() error {
    if g.paused {
        return nil  // Rendering läuft weiter, letzter Snapshot sichtbar
    }
    g.sim.Step()
    return nil
}
```

**Geschwindigkeitssteuerung** via `ebiten.SetTPS(n)`:

```go
// Geschwindigkeitsregler (0–60 TPS)
func (g *Game) SetSpeed(tps int) {
    if tps == 0 {
        g.paused = true
        return
    }
    g.paused = false
    ebiten.SetTPS(tps)
}
```

Ebiten ruft `Update()` exakt `tps`-mal pro Sekunde auf — keine eigene Timer-Logik nötig.
`Draw()` läuft davon unabhängig bei 60 FPS.

**Pause / Weiter:**

```go
func (g *Game) TogglePause() {
    g.paused = !g.paused
}
```

Kein `ebiten.SetTPS(0)` für Pause — `SetTPS(0)` bedeutet unbegrenzt, nicht gestoppt.
Das `paused`-Flag in `Game` ist einfacher und expliziter.

**Next Step (bei Pause):**

```go
func (g *Game) NextStep() {
    if g.paused {
        g.sim.Step()
    }
}
```

Einmalig aus dem Input-Handler aufgerufen; kein Threading-Problem, da alles in
der Ebiten-Game-Goroutine läuft.

---

## Konsequenzen

**Positiv:**
- Ebiten übernimmt die gesamte Timing-Logik — kein manueller `time.Ticker`
- `Update()` und `Draw()` laufen in derselben Goroutine — kein Mutex für `Game`-State nötig
- `paused`-Flag ist trivial zu testen (kein Channel-Handshake)
- Tick-Zähler in `WorldSnapshot.Tick` ist die einzige Wahrheitsquelle für Tempo
- Determinismus bleibt erhalten: gleiche Tick-Sequenz unabhängig von FPS

**Negativ:**
- `sim.Step()` darf nicht blockieren — ein langsamer Tick verzögert `Update()` und
  damit den gesamten Ebiten-Loop. Gegenmittel: Allokations-Budget-Benchmark in CI
- Maximale Sim-Geschwindigkeit ist an Ebiten-TPS gebunden (Maximum: `ebiten.SetTPS(ebiten.SyncWithFPS)` für unlimitiert)
- Bei TPS > 60: `Update()` wird öfter als `Draw()` aufgerufen — mehrere Ticks pro Frame.
  Der Dirty-Flag-Mechanismus in `Draw()` (ADR-003) behandelt das korrekt

**Invariante:**
- `sim.Step()` darf nur aus `Game.Update()` aufgerufen werden — niemals aus einer
  anderen Goroutine. Verletzung: Data Race auf `Simulation`-internem State.

---

## Verworfene Alternativen

### A: Eigene Sim-Goroutine mit `time.Ticker`

```go
go func() {
    ticker := time.NewTicker(time.Second / time.Duration(tps))
    for range ticker.C {
        sim.Step()
        exporter.Store(snap)
    }
}()
```

Problem: Zwei Goroutinen für `sim.Step()` (Sim-Goroutine) und `Game.Update()`
(Ebiten-Game-Loop). Benötigt Synchronisation für `Game`-State (paused-Flag,
Config-Swap). `SnapshotExporter` (ADR-003) löst nur den `Draw()`-Konflikt —
`Update()` vs. Sim-Goroutine ist ein neues Problem.
Zusätzliche Komplexität ohne messbaren Gewinn bei synchronem Design. Abgelehnt.

### B: `ebiten.SetTPS(0)` für Pause

`SetTPS(0)` setzt die TPS auf unbegrenzt (so schnell wie möglich), nicht auf null.
Pause-Semantik über TPS ist damit nicht ausdrückbar. Abgelehnt.

### C: Channel-basierte Step-Anfragen

`Update()` sendet `stepRequest` auf Channel, Sim-Goroutine antwortet nach Step().
Maximale Komplexität für minimalen Nutzen. Sinnvoll nur wenn `sim.Step()`
substanziell länger als ein Frame dauert — das ist der Fehlerfall, nicht der
Normalfall. Als Optimierungspfad dokumentiert, nicht für MVP.
