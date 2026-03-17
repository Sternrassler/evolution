# ADR-006: Ghost-Rows für Partitionsgrenzen, 1-Tick-Latenz akzeptiert

- **Datum:** 2026-03-17
- **Status:** Accepted

---

## Kontext

Die Welt ist in N horizontale Streifen (Partitionen) aufgeteilt. Jede Partition
wird von einer Goroutine in Phase 1 bearbeitet. Individuen nahe einer Partitionsgrenze
müssen aber Nahrung und andere Individuen auch **jenseits** der Grenze wahrnehmen
können — bis zu einem Radius K Tiles (K = `max(MaxSpeedRange, MaxSightRange)`).

Problem: In Phase 1 laufen alle Goroutinen parallel. Goroutine 0 darf den State
von Partition 1 nicht lesen, während Goroutine 1 ihn liest (Data Race). Ein
globaler Mutex würde die Parallelisierung zunichtemachen.

Drei Ansätze wurden diskutiert:

1. **Ghost-Rows:** Vor Phase 1 kopiert der Koordinator K Randzeilen jeder Partition
   in Read-Only-Puffer der Nachbarpartitionen. Phase 1 liest nur den eigenen State
   + Ghost-Puffer — keine geteilten mutablen Referenzen.

2. **Hold-Logik:** Individuen an Grenzen werden nicht in Phase 1 verarbeitet,
   sondern in Phase 2 (sequentiell) nachgeholt.

3. **Shared Memory + RWLock:** Grenzregionen werden unter Lock von beiden
   Partitionen gelesen.

---

## Entscheidung

**Ghost-Row-Ansatz mit bewusst akzeptierter 1-Tick-Latenz.**

**Mechanismus:**

```
Vor Phase 1 (sequentiell):
  Partition[i].GhostBottom = copy(Partition[i+1].Rows[0..K-1])
  Partition[i+1].GhostTop  = copy(Partition[i].Rows[EndRow-K..EndRow-1])

Phase 1 (parallel):
  Goroutine i liest:
  - Eigene Partition (mutable, aber nur diese Goroutine liest sie)
  - GhostTop / GhostBottom (read-only Kopien, kein Data Race)
```

**K-Berechnung:** `K = max(Config.MaxSpeedRange, Config.MaxSightRange)` — statisch
aus Config, bekannt vor Simulationsstart. `config.Validate()` prüft:
`WorldHeight / NumPartitions >= 2 * K`.

**1-Tick-Latenz:** Ghost-Rows sind Kopien vom Ende des **vorherigen** Ticks.
Ein Individuum nahe der Grenze sieht seine Nachbarn um einen Tick verzögert.
Diese Latenz ist bewusst akzeptiert.

**Warum akzeptabel?** Individuen treffen probabilistische Entscheidungen
(Nahrungssuche, Reproduktion). Eine 1-Tick-Latenz von ~50ms bei 20 TPS
ist nicht wahrnehmbar und verändert keine Simulationsinvarianten
(Energieerhaltung, Populations-Balance). Der Determinismus ist erhalten,
weil Ghost-Rows identisch sind bei gleichem Seed.

---

## Konsequenzen

**Positiv:**
- Phase 1 ist vollständig parallelisierbar — keine Locks zwischen Goroutinen
- `RunPhase1()` liest nur eigenen zusammenhängenden Speicher → cache-friendly
- Einfach zu testen: `PartitionIntegrityChecker` prüft Ghost-Row-Frische
- Statisches K aus Config → keine Laufzeitüberraschungen

**Negativ:**
- Speicheroverhead: 2 × K Zeilen pro Partitionsgrenze als Kopie
  Bei K=10, 200 Tiles Breite, 4 Feldern à 4 Byte: ~32 KB pro Grenze — vernachlässigbar
- Ghost-Row-Copy ist sequentiell und kostet O(K × BreitePartition) pro Tick
- 1-Tick-Latenz an Grenzen: Individuen reagieren leicht verzögert auf Nachbarn
  jenseits der Grenze — akzeptierter Simulationsartefakt
- `MinPartitionHeight = 2 * K` begrenzt maximale Partitionsanzahl:
  Bei K=10, WorldHeight=200: max. 10 Partitionen

**Invariante:**
- Ghost-Rows sind niemals älter als 1 Tick
- `PartitionIntegrityChecker` (aktiviert via `Config.DebugIntegrity`) prüft dies
  nach jedem Tick

---

## Verworfene Alternativen

### A: Hold-Logik (Grenz-Individuen in Phase 2)

Individuen innerhalb K Tiles einer Partitionsgrenze werden nicht in Phase 1
verarbeitet. Phase 2 holt sie sequentiell nach.

Problem: Bei K=10 und schmalen Partitionen kann bis zu 20/H der Population
"held" sein — das untergräbt den Parallelisierungsgewinn erheblich. Außerdem
kompliziert sich Phase 2 stark (welche Individuen sind held? Wechseln sie
zwischen Ticks?). Abgelehnt.

### B: Shared Memory + `sync.RWMutex` für Grenzbereiche

Goroutinen locken die Grenzregion per RWLock.
Problem: Lock-Contention an jedem Tick zwischen N Goroutinen-Paaren.
Bei 4 Partitionen: 3 Grenzen × 2 Seiten = 6 Lock-Operationen pro Tick.
Overhead oft größer als Parallelisierungsgewinn. Abgelehnt.

### C: Feinere Ghost-Rows (nur die tatsächlich relevanten Tiles)

Statt K vollständige Zeilen zu kopieren, nur Tiles kopieren, auf die tatsächlich
zugegriffen wird (Spatial-Grid-gesteuert). Reduziert Ghost-Row-Speicher und
Copy-Overhead. Komplexität deutlich höher (dynamische Menge, volatile zwischen Ticks).
Als Optimierungspfad dokumentiert, nicht für MVP.
