# ADR-011: Predator-Agent-Architektur

- **Datum:** 2026-03-18
- **Status:** Accepted

---

## Kontext

M11 führt Räuber (Predatoren) als zweiten Agenten-Typ in die Simulation ein. Drei
Implementierungsoptionen wurden evaluiert. Die Entscheidung hat direkte Auswirkungen
auf die Zero-Alloc-Garantie von `BenchmarkRunPhase1` (CI Gate 5).

### Option A — `sim/predator`-Package mit Value-Type-Funktion

Ein separates Package `sim/predator` enthält einen schlanken `State`-Value-Type und
eine Top-Level-Funktion `Tick(s State, ctx world.WorldContext, out *entity.EventBuffer)`.
Dispatch in `sim/partition/worker.go` via `EntityType`-Switch.

### Option B — `EntityType`-Switch im bestehenden `agent.tick()`

Der bestehende Agent-Tick-Code erhält einen Switch auf `EntityType` und führt
Predator-Logik direkt im Herbivoren-Tick aus.

### Option C — Interface-Dispatch im Hot-Path

Ein `Agent`-Interface (`Tick(ctx, out)`) wird definiert; `agent` und `Predator`
implementieren es. `sim/partition` speichert Interface-Werte und ruft `Tick` polymorphisch auf.

---

## Entscheidung

**Option A — modifiziert: `sim/predator`-Package mit `State`-Value-Type.**

```go
// sim/predator/predator.go
package predator

import (
    "github.com/Sternrassler/evolution/sim/entity"
    "github.com/Sternrassler/evolution/sim/world"
)

// State hält nur die Felder, die der Predator-Tick benötigt (SoA-kompatibel).
type State struct {
    Idx    int32
    X, Y   int32
    Energy float32
    Genes  [entity.NumGenes]float32
}

// Tick führt den Predator-Schritt aus: Jagd, Random Walk, Reproduktion.
// Kein Zeiger-Receiver, kein Interface → stack-allokiert, 0 allocs.
func Tick(s State, ctx world.WorldContext, out *entity.EventBuffer) {
    // Jagdverhalten: IndividualsNear → EventAttack
    // Random Walk wenn keine Beute in Sichtweite
    // Reproduktion mit GeneAggression-Einfluss
}
```

Dispatch in `sim/partition/worker.go`:

```go
switch entityType {
case entity.Herbivore:
    agent.Tick(idx, ctx, &p.Buf)
case entity.Predator:
    s := predator.State{
        Idx:    i,
        X:      p.X[i],
        Y:      p.Y[i],
        Energy: p.Energy[i],
        Genes:  p.Genes[i],
    }
    predator.Tick(s, ctx, &p.Buf)
}
```

---

## Konsequenzen

**Positiv:**

- **0 allocs im Hot-Path** — `predator.State` ist ein Value-Type, wird vom Go-Compiler
  stack-allokiert; kein Interface-Boxen, kein Heap-Escape
- **SoA/AoS-Grenze gewahrt** (ADR-002) — `predator.State` liest direkt aus den SoA-Arrays
  der Partition; kein AoS-`Individual` im Hot-Path
- **`sim/predator` unabhängig testbar** — keine Abhängigkeit auf `sim/partition`;
  Tests rufen `predator.Tick(state, ctx, buf)` direkt auf
- **Herbivore-Logik unberührt** — bestehender `agent.Tick`-Code wird nicht verändert
- **Kein zirkulärer Import** — `sim/predator` importiert nur `sim/entity` + `sim/world`
  (Leaf-Packages); `sim/partition` importiert `sim/predator` für den Dispatch

**Negativ / Anpassungsbedarf:**

- Issue #5 (`sim/predator`): Die ursprünglich in ROADMAP.md skizzierte Signatur
  `func (p *Predator) Tick(ctx, out)` wird zu `predator.Tick(State, ctx, out)` —
  Anpassung der Issue-Beschreibung nötig
- `sim/partition/worker.go` erhält einen EntityType-Switch — minimale Kopplung an
  Predator-Existenz, aber kein Open/Closed-Verstoß da Typen endlich und bekannt sind

---

## Verworfene Alternativen

### Option B — Switch im bestehenden `agent.tick()`

Verstößt gegen das Single-Responsibility-Prinzip: `agent.tick()` wird für Herbivoren-
und Predator-Logik zuständig. Schlechtere Testbarkeit, kein separates Package.
Schwerer erweiterbar wenn weitere Typen (z. B. Scavenger in M14) hinzukommen.

### Option C — Interface `Agent` mit `Tick(ctx, out)`

Interface-Werte in Go haben eine Größe von 16 Bytes (Typ-Zeiger + Daten-Zeiger).
`agent{idx int32, p *Partition}` ist 12 Bytes — beim Boxen in ein Interface wird der
Wert auf dem Heap allokiert. Damit verletzt Option C die `BenchmarkRunPhase1`
0-allocs/op-Garantie (CI Gate 5). Außerdem: Interface-Dispatch verhindert Inlining
im Hot-Path.

---

## Importgraph (nach ADR-011)

```
cmd/evolution
  └── ui ──────────── render
        └── sim ──── sim/partition ── sim/predator ── sim/entity  ← Leaf
                  └──────────────── sim/world ────── sim/entity
gen ──── sim/world
config ─ (keine Projekt-Imports)
```

`sim/predator` ist ein neues Blatt zwischen `sim/partition` und `sim/entity`/`sim/world`.
Kein Zirkel, SoA-Grenze gewahrt.
