# ADR-002: SoA in `sim/partition`, AoS in `sim/entity`

- **Datum:** 2026-03-17
- **Status:** Accepted

---

## Kontext

Die Simulation iteriert pro Tick über alle lebenden Individuen in einer Partition
(Phase 1: `RunPhase1()`). Bei bis zu 10 000 Individuen auf einer 200×200-Welt mit
mehreren Partitionen ist Cache-Effizienz im Hot-Path entscheidend für die angestrebten
≥20 Ticks/Sekunde.

Zwei etablierte Speicherlayouts stehen zur Wahl:

**Array of Structs (AoS):**
```
[{X, Y, Energy, Age, Genes}, {X, Y, Energy, Age, Genes}, ...]
```
Ein Individuum liegt zusammenhängend im Speicher. Zugriff auf alle Felder eines
Individuums: ein Cache-Line-Load. Iteration über ein einzelnes Feld (z.B. nur
`Energy`): jeder Zugriff lädt irrelevante Nachbarfelder mit → Cache-Thrashing.

**Struct of Arrays (SoA):**
```
X:      [x0, x1, x2, ...]
Y:      [y0, y1, y2, ...]
Energy: [e0, e1, e2, ...]
Genes:  [[g0...], [g1...], ...]
```
Iteration über ein Feld: maximal cache-freundlich. Nachteil: Zugriff auf mehrere
Felder eines Individuums erfordert mehrere Array-Lookups.

`Phase 1` liest sequentiell `Energy`, `Genes`, `X`, `Y` — typischerweise nicht alle
gleichzeitig pro Individuum, sondern feldweise im Algorithmus. SoA ist hier überlegen.

Gleichzeitig muss `WorldSnapshot` (gelesen von `render/`) ergonomisch traversierbar
sein. `render` iteriert pro Pixel über alle Individuen und greift auf Pos + Genes zu —
AoS ist hier ausreichend, da beide Felder gemeinsam benötigt werden.

---

## Entscheidung

**SoA intern in `sim/partition`** für den Simulations-Hot-Path:

```go
type Partition struct {
    X      []int32
    Y      []int32
    Energy []float32
    Age    []int32
    Alive  []bool
    Genes  [][NumGenes]float32
    IDs    []uint64
    // ...
}
```

**AoS in `sim/entity.Individual`** für die öffentliche API und den `WorldSnapshot`:

```go
type Individual struct {
    ID     uint64
    Pos    image.Point
    Energy float32
    Age    int
    Genes  [NumGenes]float32
    alive  bool
}
```

**Explizite Konvertierungsgrenze:**
- `Partition.ToIndividuals()` — SoA → AoS beim Snapshot-Export (einmal pro Tick)
- `sim/testutil.BuildPartition()` — AoS → SoA in Tests

Die Konvertierungskosten beim Snapshot-Export betragen bei 10 000 Individuen
~Mikrosekunden und fallen in Phase 2 (sequentiell, unkritisch).

---

## Konsequenzen

**Positiv:**
- `RunPhase1()` ist zero-alloc, cache-freundlich (CI Gate 5 erzwingt das)
- Öffentliche API (`WorldSnapshot`) bleibt ergonomisch
- Klare Trennung: Hot-Path-Optimierung intern, externe Konsumenten unberührt
- `render/` muss keine SoA-Arrays kennen

**Negativ:**
- Zwei Darstellungen desselben Konzepts erhöhen die kognitive Last
- Konvertierungsschritt in `ToIndividuals()` muss korrekt gehalten werden
- Tests für `sim/partition` brauchen `BuildPartition()` als AoS→SoA-Helfer
  (sonst manuelles SoA-Befüllen in jedem Test)

**Messbare Verpflichtung:**
- Benchmark `BenchmarkPhase1` mit `ReportAllocs()` in CI (Gate 5)
- Regressions-Schwelle: >50% Allokationszunahme = Fail

---

## Verworfene Alternativen

### A: Nur AoS überall

Einfacher, aber `RunPhase1()` mit AoS erzeugt Cache-Misses bei feldweiser
Iteration. Bei 10k Individuen und 20 TPS bedeutet das messbare Performanceverluste.
Benchmark zuerst? Ja — aber SoA ist bei diesem Zugriffsmuster so etabliert, dass
der Messung vorzugreifen vertretbar ist. Kann jederzeit auf AoS zurückgebaut werden
(Änderung nur in `sim/partition` intern).

### B: Nur SoA überall (auch im `WorldSnapshot`)

`render/` würde SoA-Arrays direkt traversieren. Das koppelt `render/` an die
interne Partition-Struktur und macht den Code schwer lesbar. `WorldSnapshot` soll
eine stabile, einfache API sein — SoA ist dort kein Gewinn.

### C: Unsafe-Pointer-Tricks für In-Place-Uminterpretation

Kein explizites Konvertieren, stattdessen Speicher-Layout so gestalten, dass
AoS und SoA dieselbe Speicherregion interpretieren. Vermeidet Konvertierungskosten,
aber: undefined behavior territory, nicht portabel, nicht wartbar. Abgelehnt.
