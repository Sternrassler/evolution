# Evolution Simulation — Implementierungs-Roadmap

> Basis: [CONCEPT.md](./CONCEPT.md) + [ARCHITECTURE.md](./ARCHITECTURE.md)
> Stand: 2026-03-18 · Status: M0–M11 ✅, Post-MVP Erweiterungen teilweise umgesetzt

---

## Voraussetzungen & Erste Schritte

### Go-Modul initialisieren

```bash
go mod init github.com/user/evolution
go get github.com/hajimehoshi/ebiten/v2
go get pgregory.net/rapid
go get github.com/BurntSushi/toml   # Config-Loader
```

### Verzeichnisstruktur anlegen

```
evolution/
├── cmd/
│   └── evolution/
│       └── main.go
├── config/
│   ├── config.go
│   └── config_test.go
├── gen/
│   ├── gen.go
│   └── gen_test.go
├── render/
│   ├── renderer.go
│   └── renderer_test.go
├── sim/
│   ├── sim.go
│   ├── sim_test.go
│   ├── entity/
│   │   ├── individual.go
│   │   ├── gene.go
│   │   ├── event.go
│   │   └── entity_test.go
│   ├── partition/
│   │   ├── partition.go
│   │   └── partition_test.go
│   ├── testutil/
│   │   └── testutil.go
│   └── world/
│       ├── world.go
│       └── world_test.go
├── testworld/
│   └── testworld.go
├── ui/
│   ├── game.go
│   └── game_test.go
├── tools/
│   ├── check_global_rand.go
│   └── check_ebiten_imports.go
├── .github/
│   └── workflows/
│       └── ci.yml
├── Makefile
├── go.mod
├── go.sum
├── docs/
│   ├── CONCEPT.md
│   ├── ARCHITECTURE.md
│   └── ROADMAP.md
```

### Empfohlene Werkzeuge

| Werkzeug | Zweck | Installation |
|---|---|---|
| `golangci-lint` | Statische Analyse | `go install github.com/golangci-lint/golangci-lint/cmd/golangci-lint@latest` |
| `pprof` | Performance-Profiling | in Go-Stdlib enthalten |
| `benchstat` | Benchmark-Vergleich | `go install golang.org/x/perf/cmd/benchstat@latest` |
| `go test -race` | Data-Race-Detektion | in Go-Toolchain enthalten |
| `xxhash` | Schneller Hash für Snapshots | `go get github.com/cespare/xxhash/v2` |

---

## Meilenstein 0 — CI-Grundgerüst

> Ziel: Alle CI-Gates aufstellen, bevor eine einzige Simulation-Zeile geschrieben wird.
> Kein Go-Package-Code außer den Tools selbst.

### Dateien

| Datei | Inhalt |
|---|---|
| `.github/workflows/ci.yml` | GitHub Actions Workflow mit allen 5 Gates |
| `tools/check_global_rand.go` | AST-basierter Checker: kein `math/rand` in `sim/` |
| `tools/check_ebiten_imports.go` | Import-Checker: kein `ebiten` in `sim/`, `gen/`, `config/` |
| `Makefile` | Targets: `test`, `race`, `bench`, `lint`, `ci` |

### `.github/workflows/ci.yml`

```yaml
name: CI

on:
  push:
  pull_request:

jobs:
  ci:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      # Gate 1: Import-Check — kein ebiten in sim/, gen/, config/
      - name: Gate 1 – Import-Check
        run: go run ./tools/check_ebiten_imports.go ./...

      # Gate 2: Global-rand-Check — kein math/rand in sim/
      - name: Gate 2 – Global-rand-Check
        run: go run ./tools/check_global_rand.go ./sim/...

      # Gate 3: Determinismus — erst grün ab M7
      - name: Gate 3 – Determinismus
        run: go test -run TestDeterminism -count=2 ./sim/...

      # Gate 4: Race Detector
      - name: Gate 4 – Race Detector
        run: go test -race ./sim/...

      # Gate 5: Allokations-Budget — erst grün ab M6
      - name: Gate 5 – Allokations-Budget
        run: go test -run='^$' -bench=BenchmarkPhase1 -benchmem ./sim/partition/...

      - name: Lint
        run: golangci-lint run ./...

      - name: Tests
        run: go test ./...
```

> **Hinweis:** Gates 3 und 5 schlagen initial fehl — das ist erwartet. Beide werden erst grün, wenn die jeweiligen Meilensteine fertig sind. Gates 1 und 2 sind sofort grün (kein zu prüfender Code).

### `tools/check_global_rand.go`

```go
//go:build ignore

// check_global_rand prüft, dass kein Code in den angegebenen Packages
// math/rand direkt importiert. Nur RandSource-Injection ist erlaubt.
//
// Aufruf: go run tools/check_global_rand.go ./sim/...
package main

import (
    "fmt"
    "go/parser"
    "go/token"
    "os"
    "path/filepath"
    "strings"
)

func main() {
    // Alle .go-Dateien in sim/ rekursiv prüfen
    // Ausnahme: _test.go-Dateien in testutil/ (dürfen rand.New verwenden)
    // Fehler: import "math/rand" ohne /v2 (nur rand.New(rand.NewSource(...)) via RandSource)
    // Gibt Exit-Code 1 bei Verletzungen aus
}
```

### `Makefile`

```makefile
.PHONY: test race bench lint ci

test:
	go test ./...

race:
	go test -race ./...

bench:
	go test -run='^$$' -bench=. -benchmem ./...

lint:
	golangci-lint run ./...

ci: lint test race
	go run ./tools/check_ebiten_imports.go ./...
	go run ./tools/check_global_rand.go ./sim/...
```

### Akzeptanzkriterien M0

- [x] `make ci` läuft durch (Gates 3 und 5 dürfen noch fehlschlagen — `exit 0` via `|| true`)
- [x] Gate 1 ist grün (kein Code → nichts zu verletzen)
- [x] Gate 2 ist grün (kein Code → nichts zu verletzen)
- [x] Gate 4 ist grün (kein Code → kein Race)
- [x] GitHub Actions Workflow ist sichtbar im Repository

---

## Meilenstein 1 — `sim/entity` (Leaf-Package)

> **Abhängigkeiten:** keine
> **Warum zuerst:** Wird von `sim/partition`, `sim/world`, `gen`, `sim` importiert. Zirkuläre Imports entstehen, wenn dieser Typ woanders liegt.

### Dateien

| Datei | Inhalt |
|---|---|
| `sim/entity/gene.go` | `GeneKey`, `NumGenes`, Konstanten |
| `sim/entity/individual.go` | `Individual`-Struct (AoS), Konstruktor |
| `sim/entity/event.go` | `Event`-Struct, `EventType`-Enum, `EventBuffer` |
| `sim/entity/entity_test.go` | Tests für alle Typen |

### Implementierungsdetails

**`gene.go`:**

```go
package entity

type GeneKey int

const (
    GeneSpeed      GeneKey = 0
    GeneSight      GeneKey = 1
    GeneEfficiency GeneKey = 2
    NumGenes               = 3  // für Stufe 2 erhöhen → neuen case-Branch hinzufügen
)
```

**`individual.go`:**

```go
package entity

import "image"

type Individual struct {
    ID     uint64
    Pos    image.Point
    Energy float32
    Age    int
    Genes  [NumGenes]float32
    alive  bool
}

func NewIndividual(id uint64, pos image.Point, genes [NumGenes]float32, energy float32) Individual {
    return Individual{ID: id, Pos: pos, Genes: genes, Energy: energy, alive: true}
}

func (ind *Individual) IsAlive() bool { return ind.alive }
func (ind *Individual) Kill()         { ind.alive = false }
```

**`event.go`:**

```go
package entity

type EventType uint8

const (
    EventMove       EventType = iota
    EventEat
    EventReproduce
    EventDie
)

type Event struct {
    Type      EventType
    AgentIdx  int32   // SoA-Index im Partition-Array
    TargetPos image.Point
    Value     float32 // Energie-Delta, Gen-Wert etc.
}

type EventBuffer struct {
    events []Event
}

func NewEventBuffer(cap int) EventBuffer {
    return EventBuffer{events: make([]Event, 0, cap)}
}

func (b *EventBuffer) Append(e Event) { b.events = append(b.events, e) }
func (b *EventBuffer) Reset()         { b.events = b.events[:0] }
func (b *EventBuffer) Len() int       { return len(b.events) }
func (b *EventBuffer) Events() []Event { return b.events }
```

### Tests

```go
// TestEventBufferZeroAlloc: testing.AllocsPerRun(100, ...) == 0 nach Prä-Allokation
// TestIndividualLiveness: Kill() → IsAlive() == false
// TestGeneKeyConstants: NumGenes == 3, Konstanten korrekt
// Property (rapid): EventBuffer.Append/Reset — Len() immer konsistent
```

### Akzeptanzkriterien M1

- [x] `sim/entity` hat **null Imports** auf andere `sim/`-Packages (CI Gate 1 prüft dies)
- [x] `go test ./sim/entity/...` grün
- [x] `EventBuffer.Append()` ist zero-alloc nach Pre-Allokation (Benchmark)
- [x] Kein `math/rand` in `sim/entity/` (CI Gate 2)

---

## Meilenstein 2 — `config`

> **Abhängigkeiten:** keine (nur Go-Stdlib + optionales TOML-Paket)

### Dateien

| Datei | Inhalt |
|---|---|
| `config/config.go` | `Config`-Struct, `DefaultConfig()`, `Validate()` |
| `config/loader.go` | TOML-Loader (optional, MVP: nur Defaults) |
| `config/config_test.go` | Tests |

### Implementierungsdetails

```go
package config

import "github.com/user/evolution/sim/entity"

type Config struct {
    // Welt
    WorldWidth      int     `toml:"world_width"`      // Default: 200
    WorldHeight     int     `toml:"world_height"`     // Default: 200
    NumPartitions   int     `toml:"num_partitions"`   // Default: runtime.GOMAXPROCS(0)

    // Simulation
    MaxPopulation   int     `toml:"max_population"`   // Default: 10000
    InitialPop      int     `toml:"initial_pop"`      // Default: 500
    TicksPerSecond  int     `toml:"ticks_per_second"` // Default: 20
    DebugIntegrity  bool    `toml:"debug_integrity"`  // Default: false

    // Spatial Grid
    SpatialCellSize int     `toml:"spatial_cell_size"` // Default: MaxSightRange

    // Gen-Grenzen (für Ghost-Row-Berechnung)
    MaxSpeedRange   int     `toml:"max_speed_range"`  // Default: 5
    MaxSightRange   int     `toml:"max_sight_range"`  // Default: 10

    // Energie
    BaseEnergyCost         float32 `toml:"base_energy_cost"`
    ReproductionThreshold  float32 `toml:"reproduction_threshold"`
    ReproductionReserve    float32 `toml:"reproduction_reserve"`

    // Gene-Definitionen
    GeneDefinitions []GeneDef `toml:"gene_definitions"`
}

type GeneDef struct {
    Key          entity.GeneKey `toml:"key"`
    Min          float32        `toml:"min"`
    Max          float32        `toml:"max"`
    MutationRate float32        `toml:"mutation_rate"`
    MutationStep float32        `toml:"mutation_step"`
}

func DefaultConfig() Config { ... }

// Validate prüft Konsistenz: WorldHeight / NumPartitions >= 2 * GhostK()
// Gibt error zurück, kein panic
func (c *Config) Validate() error { ... }

// GhostK berechnet K = max(MaxSpeedRange, MaxSightRange)
func (c *Config) GhostK() int { ... }
```

### Tests

```go
// TestDefaultConfigValid: DefaultConfig().Validate() == nil
// TestValidate_TooManyPartitions: Fehler wenn Partitionen zu viele für Ghost-Rows
// TestValidate_ZeroPopulation: Fehler bei MaxPopulation == 0
// Property (rapid): Zufällige valide Config → Validate() == nil
```

### Akzeptanzkriterien M2

- [x] `config` hat **keine Projekt-Imports** (reine Stdlib)
- [x] `DefaultConfig().Validate()` ist nil
- [x] `go test ./config/...` grün
- [x] `GhostK()` korrekt berechnet

---

## Meilenstein 3 — `sim/world`

> **Abhängigkeiten:** M1 (`sim/entity`), M2 (`config`)

### Dateien

| Datei | Inhalt |
|---|---|
| `sim/world/world.go` | `Tile`, `BiomeType`, `Grid`, `ApplyRegrowth()` |
| `sim/world/spatial.go` | `SpatialGrid`, `IndividualsNear()` |
| `sim/world/world_test.go` | Tests |

### Implementierungsdetails

```go
package world

import (
    "github.com/user/evolution/sim/entity"
    "github.com/user/evolution/config"
)

type BiomeType uint8

const (
    BiomeWater  BiomeType = iota
    BiomeMeadow
    BiomeDesert
)

// Regrowth-Rate pro Biom und Tick
var BiomeRegrowthRate = map[BiomeType]float32{
    BiomeMeadow: 0.05,
    BiomeDesert: 0.01,
    BiomeWater:  0.0,
}

type Tile struct {
    Biome   BiomeType
    Food    float32
    FoodMax float32
}

func (t *Tile) IsWalkable() bool { return t.Biome != BiomeWater }

type Grid struct {
    Tiles  []Tile
    Width  int
    Height int
}

func NewGrid(width, height int) *Grid { ... }
func (g *Grid) At(x, y int) *Tile    { return &g.Tiles[y*g.Width+x] }

// ApplyRegrowth wächst Nahrung nach; gibt gesamte gewachsene Energie zurück
// (für Energieerhaltungs-Invariante in TickStats.EnergyRegrown)
func (g *Grid) ApplyRegrowth() float32 { ... }

// SpatialGrid: Flat Bucket-Array, pre-allokiert, O(n) Rebuild
type SpatialGrid struct {
    buckets  [][]int32  // Entity-Indizes pro Zelle
    cellSize int
    width    int
    height   int
}

func NewSpatialGrid(cfg *config.Config) *SpatialGrid { ... }

// Rebuild baut den Grid vollständig neu — O(n), einmal pro Tick
func (sg *SpatialGrid) Rebuild(individuals []entity.Individual) { ... }

// IndividualsNear gibt SoA-Indizes im Sichtradius zurück — zero-alloc (reused slice)
func (sg *SpatialGrid) IndividualsNear(p image.Point, radius int, out []int32) []int32 { ... }
```

### Tests

```go
// TestRegrowthEnergyBudget: Summe ApplyRegrowth() == tatsächliche FoodDelta-Summe
// TestSpatialGridRebuild: Alle Individuen auffindbar nach Rebuild
// TestSpatialGridNear_Empty: Zero-Result bei leerer Welt
// TestTileWalkable: Wasser nicht begehbar, Wiese begehbar
// Property (rapid): IndividualsNear — niemals außerhalb Radius
```

### Akzeptanzkriterien M3

- [x] `go test ./sim/world/...` grün
- [x] `ApplyRegrowth()` gibt korrekte Energie-Summe zurück
- [x] `SpatialGrid.IndividualsNear()` ist zero-alloc (Benchmark)
- [x] Kein `ebiten`-Import (CI Gate 1)

---

## Meilenstein 4 — `gen`

> **Abhängigkeiten:** M2 (`config`), M3 (`sim/world`)

### Dateien

| Datei | Inhalt |
|---|---|
| `gen/gen.go` | `GenerateWorld()`, `ProceduralSource`, Cellular Automaton |
| `gen/gen_test.go` | Tests |

### Implementierungsdetails

```go
package gen

import (
    "github.com/user/evolution/config"
    "github.com/user/evolution/sim/world"
)

// TileSource ist das Interface für austauschbare Weltgeneratoren
type TileSource interface {
    Generate(cfg config.Config, rng config.RandSource) []world.Tile
}

// ProceduralSource: Cellular-Automaton-basierte Generierung
type ProceduralSource struct{}

// GenerateWorld ist eine pure function — kein globaler State, kein Side-Effect
// Algorithmus:
// 1. Zufällige Biom-Belegung nach Wahrscheinlichkeiten aus Config
// 2. N Iterationen Cellular Automaton (Majority-Rule, 4er-Nachbarschaft)
// 3. Nahrungswerte auf FoodMax initialisieren
// 4. Wasserrand-Glättung (optional)
func GenerateWorld(cfg config.Config, rng config.RandSource) []world.Tile {
    src := ProceduralSource{}
    return src.Generate(cfg, rng)
}

func (p ProceduralSource) Generate(cfg config.Config, rng config.RandSource) []world.Tile { ... }
```

### Cellular Automaton Details

```
Algorithmus (3 Iterationen):
1. Seed: jede Zelle → BiomeType mit Wahrscheinlichkeit (60% Wiese, 20% Wüste, 20% Wasser)
2. Pro Iteration: jede Zelle → BiomeType der Mehrheit in 3×3-Nachbarschaft
3. Küstenglättung: Wasser-Tiles mit >3 Land-Nachbarn → nächstes Land-Biom
```

### Tests

```go
// TestGenerateWorld_Dimensions: Länge == Width*Height
// TestGenerateWorld_ValidBiomes: Alle Biome in BiomeType-Range
// TestGenerateWorld_FoodInit: Food == FoodMax für alle non-Water-Tiles
// TestGenerateWorld_Determinism: Gleicher Seed → identisches []Tile
// Property (rapid): Biom-Verteilung innerhalb erwarteter Bänder (±20% der Config-Wahrscheinlichkeiten)
// Property (rapid): Kein Tile mit Food > FoodMax
```

### Akzeptanzkriterien M4

- [x] `go test ./gen/...` grün
- [x] `GenerateWorld()` ist deterministisch bei gleichem Seed
- [x] Kein globaler `math/rand` (CI Gate 2)
- [x] Kein `ebiten`-Import (CI Gate 1)

---

## Meilenstein 5 — Test-Infrastruktur (`testworld` + `sim/testutil`)

> **Abhängigkeiten:** M1–M4
> **Warum jetzt:** Die nächsten Meilensteine (M6–M9) sind complex und brauchen saubere Testhelfer. Keine Mocks — echte Semantik.

### Dateien

| Datei | Inhalt |
|---|---|
| `testworld/testworld.go` | `WorldContextBuilder`, `WorldContext`-Implementierung |
| `sim/testutil/testutil.go` | `BuildPartition()`, `HashSnapshot()` |

### Implementierungsdetails

**`testworld/testworld.go`:**

```go
package testworld

import (
    "image"
    "github.com/user/evolution/sim/entity"
    "github.com/user/evolution/sim/world"
    "github.com/user/evolution/config"
)

// WorldContextBuilder baut eine kleine, reale WorldContext-Implementierung
// für Tests. Keine Mocks — echte WorldContext-Semantik.
type WorldContextBuilder struct {
    width, height int
    tiles         []world.Tile
    individuals   []entity.Individual
    rng           config.RandSource
    cfg           config.Config
}

func New(width, height int) *WorldContextBuilder { ... }
func (b *WorldContextBuilder) WithTile(x, y int, t world.Tile) *WorldContextBuilder { ... }
func (b *WorldContextBuilder) WithIndividual(ind entity.Individual) *WorldContextBuilder { ... }
func (b *WorldContextBuilder) WithRng(rng config.RandSource) *WorldContextBuilder { ... }
func (b *WorldContextBuilder) Build() *WorldContext { ... }

// WorldContext implementiert sim.WorldContext (das Interface aus sim/)
type WorldContext struct { ... }

func (w *WorldContext) TileAt(p image.Point) world.Tile                    { ... }
func (w *WorldContext) IndividualsNear(p image.Point, r int) []int32       { ... }
func (w *WorldContext) Rand() config.RandSource                            { ... }
func (w *WorldContext) MutationRate() float32                              { ... }
func (w *WorldContext) ReproductionThreshold() float32                     { ... }
func (w *WorldContext) MaxSpeed() float32                                  { ... }
func (w *WorldContext) MaxSight() float32                                  { ... }
```

**`sim/testutil/testutil.go`:**

```go
package testutil

import (
    "github.com/user/evolution/sim/entity"
    "github.com/user/evolution/sim/partition"
    "github.com/user/evolution/sim"
)

// BuildPartition konvertiert []Individual (AoS) → *Partition (SoA)
// Macht Tests lesbar: kein manuelles SoA-Befüllen in Testcode
func BuildPartition(individuals []entity.Individual, cfg config.Config) *partition.Partition { ... }

// HashSnapshot berechnet FNV-1a/xxhash über geordnete Snapshot-Felder
// Keine Maps — deterministisch, geordnete Slices
func HashSnapshot(snap *sim.WorldSnapshot) uint64 { ... }
```

### Akzeptanzkriterien M5

- [x] `testworld.New(10,10).WithTile(0,0,...).Build()` kompiliert und läuft
- [x] `BuildPartition()` konvertiert korrekt AoS→SoA (Unit-Tests)
- [x] `HashSnapshot()` ist deterministisch (gleiche Eingabe → gleicher Hash)
- [x] Keine Test-Packages importieren `ebiten`

---

## Meilenstein 6 — `sim/partition`

> **Abhängigkeiten:** M1 (`sim/entity`), M2 (`config`), M3 (`sim/world`), M5 (`sim/testutil`)

### Dateien

| Datei | Inhalt |
|---|---|
| `sim/partition/partition.go` | `Partition`-Struct (SoA), `GhostRow`, FreeList, Worker |
| `sim/partition/worker.go` | Phase-1-Worker-Goroutine |
| `sim/partition/partition_test.go` | Tests |

### Implementierungsdetails

**SoA-Struct:**

```go
package partition

import (
    "github.com/user/evolution/sim/entity"
    "github.com/user/evolution/config"
)

type GhostRow struct {
    X      []int32
    Y      []int32
    Energy []float32
    Genes  [][entity.NumGenes]float32
}

type Partition struct {
    // SoA-Hot-Arrays — zusammenhängender Speicher, cache-freundlich
    X      []int32
    Y      []int32
    Energy []float32
    Age    []int32
    Alive  []bool
    Genes  [][entity.NumGenes]float32

    // ID-Tracking
    IDs    []uint64

    // Management
    FreeList []int32      // Indizes toter Slots zur Wiederverwendung
    Buf      entity.EventBuffer  // pre-allokiert, ein Buffer pro Partition
    Len      int          // aktuelle Anzahl lebender Slots (inkl. freier)

    // Ghost-Rows (read-only für Phase 1)
    GhostTop    []GhostRow
    GhostBottom []GhostRow

    // Partition-Grenzen in der Gesamtwelt
    StartRow int
    EndRow   int
}

// NewPartition allokiert alle Arrays einmalig mit cap = MaxPopulation
func NewPartition(cfg config.Config, startRow, endRow int) *Partition { ... }

// AddIndividual: FreeList-Reuse oder append
func (p *Partition) AddIndividual(ind entity.Individual) int32 { ... }

// MarkDead: setzt Alive[i] = false, fügt i in FreeList ein
func (p *Partition) MarkDead(i int32) { ... }

// ToIndividuals: SoA→AoS-Konvertierung für Snapshot-Export
func (p *Partition) ToIndividuals() []entity.Individual { ... }
```

**Phase-1-Worker:**

```go
// RunPhase1 iteriert über alle lebenden Individuen in der Partition
// und ruft agent.Tick(ctx, &p.Buf) auf.
// Liest nur: GhostRows, Grid-Tiles (via WorldContext)
// Schreibt nur: p.Buf (EventBuffer)
// KEINE Weltmutation in Phase 1
func (p *Partition) RunPhase1(ctx WorldContext) {
    p.Buf.Reset()
    for i := int32(0); i < int32(p.Len); i++ {
        if !p.Alive[i] {
            continue
        }
        ind := p.indAt(i)  // AoS-View für Agent-Interface
        ind.Tick(ctx, &p.Buf)
    }
}
```

### Tests

```go
// TestPartitionAddRemove: FreeList-Reuse korrekt
// TestPhase1_NoMutation: Welt-State nach Phase 1 unverändert
// TestPhase1_ZeroAlloc: Benchmark — 0 Allocs pro RunPhase1()
// TestGhostRowCopy: Grenzwerte korrekt kopiert
// Property (rapid): AddIndividual/MarkDead — Len immer konsistent
// Property (rapid): Kein lebender Slot mit ungültigen Koordinaten
```

### Akzeptanzkriterien M6

- [x] `RunPhase1()` ist zero-alloc (CI Gate 5 beginnt hier zu messen)
- [x] `go test -race ./sim/partition/...` grün (CI Gate 4)
- [x] FreeList-Reuse korrekt: Kein Wachstum bei Geburt-nach-Tod
- [x] SoA→AoS-Konvertierung korrekt

---

## Meilenstein 7 — `sim` (Koordinator)

> **Abhängigkeiten:** M1–M6
> **Kritisch:** Hier kommen alle Teile zusammen. Phase 2, SnapshotExporter, TickObserver.

### Dateien

| Datei | Inhalt |
|---|---|
| `sim/sim.go` | `Simulation`, `Step()`, `WorldContext`-Interface |
| `sim/snapshot.go` | `WorldSnapshot`, `SnapshotExporter`, `TickStats` |
| `sim/observer.go` | `TickObserver`-Interface, `NoopObserver` |
| `sim/agent.go` | `Agent`-Interface, `RandSource`-Interface |
| `sim/sim_test.go` | Integrations- und Simulations-Tests |

### Implementierungsdetails

**Interfaces:**

```go
package sim

import (
    "image"
    "github.com/user/evolution/sim/entity"
    "github.com/user/evolution/sim/world"
)

type RandSource interface {
    Float64() float64
    Intn(n int) int
}

type Agent interface {
    Tick(ctx WorldContext, out *entity.EventBuffer)
}

type WorldContext interface {
    TileAt(p image.Point) world.Tile
    IndividualsNear(p image.Point, radius int) []int32
    Rand() RandSource
    MutationRate() float32
    ReproductionThreshold() float32
    MaxSpeed() float32
    MaxSight() float32
}

type TickObserver interface {
    OnTick(tick uint64, stats TickStats)
}
```

**SnapshotExporter:**

```go
type SnapshotExporter struct {
    pool     [2]WorldSnapshot
    current  atomic.Pointer[WorldSnapshot]
    writeIdx int  // nur von Update()-Goroutine (Step()) beschrieben
}

func (e *SnapshotExporter) Store(snap *WorldSnapshot) {
    next := &e.pool[1-e.writeIdx]
    // Befülle next ...
    e.current.Store(next)
    e.writeIdx = 1 - e.writeIdx
}

// Load ist lock-frei — kann von Draw()-Goroutine ohne Mutex aufgerufen werden
func (e *SnapshotExporter) Load() *WorldSnapshot {
    return e.current.Load()
}
```

**Tick-Ablauf in `Step()`:**

```go
func (s *Simulation) Step() {
    // 1. Config-Snapshot (Pending-Swap)
    cfg := s.swapPendingConfig()

    // 2. Ghost-Row-Copy (K Grenzzeilen)
    s.copyGhostRows(cfg)

    // 3. Spatial-Grid-Rebuild O(n)
    s.spatialGrid.Rebuild(s.allIndividuals())

    // 4. Phase 1 — parallel
    var wg sync.WaitGroup
    for _, p := range s.partitions {
        wg.Add(1)
        go func(part *partition.Partition) {
            defer wg.Done()
            part.RunPhase1(s.contextFor(part, cfg))
        }(p)
    }
    wg.Wait()

    // 5. Phase 2 — sequentiell
    stats := s.applyPhase2(cfg)

    // 6. Observer
    s.observer.OnTick(s.tick, stats)

    // 7. Snapshot-Export
    s.exporter.Store(s.buildSnapshot(stats))

    s.tick++

    // 8. Integrity-Check (wenn DebugIntegrity)
    if cfg.DebugIntegrity {
        s.checkIntegrity()
    }
}
```

**Mutations-Logik (in Phase 2 bei EventReproduce):**

```go
// mutateGenes: Klont Eltern-Gene, appliziert Gauss-Störung, clampt auf [Min, Max]
func mutateGenes(parent [entity.NumGenes]float32, geneDefs []config.GeneDef, rng RandSource) [entity.NumGenes]float32 {
    child := parent
    for i, def := range geneDefs {
        if rng.Float64() < float64(def.MutationRate) {
            delta := float32(rng.Float64()*2-1) * def.MutationStep
            child[i] = clamp(parent[i]+delta, def.Min, def.Max)
        }
    }
    return child
}
```

### Tests

```go
// TestDeterminism: Gleicher Seed → identischer HashSnapshot nach 100 Ticks (2x)
//   → CI Gate 3 wird hier grün
// TestEnergyConservation: ΔEnergie_Individuen + ΔEnergie_Tiles + Energie_Tote == Regrowth
// TestNoRaceCondition: -race flag (CI Gate 4)
// TestBoundaryHandling: Individuum wechselt Partition → korrekt reassigniert
// TestPopulationCap: MaxPopulation nie überschritten
// TestMutateBounds (rapid): Genes[i] ∈ [Min, Max] nach Mutation
// TestReproductionConflict: Niedrigere ID gewinnt bei gleichzeitiger Reproduktion
// TestSnapshotImmutable: Keine Mutation exportierter Snapshots
```

### Akzeptanzkriterien M7

- [x] CI Gate 3 (Determinismus) ist jetzt **grün**
- [x] CI Gate 4 (Race Detector) ist grün
- [ ] Energieerhaltungs-Invariante: `ΔE_ind + ΔE_tiles + E_tote == E_regrown` über 100 Ticks
- [x] `SnapshotExporter.Load()` ist lock-frei (kein Mutex)
- [x] `go test ./sim/...` grün

---

## Meilenstein 8 — `render`

> **Abhängigkeiten:** M7 (`sim`), M1 (`sim/entity`), M3 (`sim/world`)

### Dateien

| Datei | Inhalt |
|---|---|
| `render/renderer.go` | `Renderer`, Pixel-Buffer, `WritePixels`-Pipeline |
| `render/color.go` | Gen-Farb-Kodierung, Biom-Farben |
| `render/renderer_test.go` | Tests |

### Implementierungsdetails

```go
package render

import (
    "github.com/hajimehoshi/ebiten/v2"
    "github.com/user/evolution/sim"
    "github.com/user/evolution/sim/world"
    "github.com/user/evolution/sim/entity"
)

type Renderer struct {
    pixelBuf  []byte   // RGBA, pre-allokiert: Width * Height * TileSize^2 * 4
    offscreen *ebiten.Image
    tileSize  int
    width     int
    height    int
}

func NewRenderer(width, height, tileSize int) *Renderer { ... }

// RenderToBuffer schreibt WorldSnapshot in den Pixel-Buffer
// Biom-Farben für Tiles, Genotyp-Farben für Individuen
// Nur aufgerufen wenn snap.Tick != lastTick (Dirty-Flag in Game.Draw)
func (r *Renderer) RenderToBuffer(snap *sim.WorldSnapshot) {
    r.renderTiles(snap.Tiles)
    r.renderIndividuals(snap.Individuals)
}

// DrawBuffer schreibt den Pixel-Buffer auf den Screen
// Immer aufgerufen, auch wenn Buffer nicht neu
func (r *Renderer) DrawBuffer(screen *ebiten.Image) {
    r.offscreen.WritePixels(r.pixelBuf)
    screen.DrawImage(r.offscreen, nil)
}
```

**Gen-Farb-Kodierung:**

```go
// GeneColor kodiert Genotyp als RGB:
// R = Speed normiert auf [0, 255]
// G = Sight normiert auf [0, 255]
// B = Efficiency normiert auf [0, 255]
func GeneColor(genes [entity.NumGenes]float32, defs []config.GeneDef) (r, g, b uint8) {
    r = normalizeGene(genes[entity.GeneSpeed], defs[entity.GeneSpeed])
    g = normalizeGene(genes[entity.GeneSight], defs[entity.GeneSight])
    b = normalizeGene(genes[entity.GeneEfficiency], defs[entity.GeneEfficiency])
    return
}

// BiomColors: Wasser=Blau, Wiese=Grün, Wüste=Sandgelb
var BiomColors = map[world.BiomeType][3]uint8{
    world.BiomeWater:  {64, 128, 200},
    world.BiomeMeadow: {80, 160, 60},
    world.BiomeDesert: {200, 180, 100},
}
```

**Zoom-abhängige Darstellung:**

```
Zoom nah   (TileSize >= 8):  Individuen als Symbol (Rechteck mit Rand)
Zoom mittel (TileSize >= 4): Individuen als farbiger Punkt (1 Pixel pro Tile)
Zoom weit  (TileSize < 4):   Individuen als Häufungspunkt (Farbmischung)
```

### Tests

```go
// TestRenderToBuffer_NoAlloc: Benchmark — 0 Allocs pro RenderToBuffer()
// TestGeneColor_Normalization: Grenzwerte Min/Max → 0/255
// TestBiomColors_Coverage: Alle BiomeTypes haben eine Farbe
// TestPixelBufSize: len(pixelBuf) == Width*Height*TileSize^2*4
```

> **Hinweis:** Ebiten-Tests laufen nicht in CI headless. `RenderToBuffer` ist ohne Ebiten testbar (nur Pixel-Buffer-Schreiblogik). `DrawBuffer` nur in manuellen Tests / Smoke-Tests.

### Akzeptanzkriterien M8

- [ ] `RenderToBuffer()` ist zero-alloc (Benchmark)
- [x] Gen-Farb-Normalisierung korrekt für alle Grenzwerte
- [x] `go test ./render/...` grün (ohne Headless-Display)
- [x] **Kein `sim/`-Package importiert `ebiten`** (CI Gate 1 — schon längst grün)

---

## Meilenstein 9 — `ui`

> **Abhängigkeiten:** M7 (`sim`), M8 (`render`), M2 (`config`)

### Dateien

| Datei | Inhalt |
|---|---|
| `ui/game.go` | `Game`-Struct, Ebiten-Game-Interface |
| `ui/hud.go` | HUD-Rendering (Statistik-Panel) |
| `ui/input.go` | Input-Handler (Geschwindigkeit, Pause, Next Step) |

### Implementierungsdetails

```go
package ui

import (
    "github.com/hajimehoshi/ebiten/v2"
    "github.com/user/evolution/sim"
    "github.com/user/evolution/render"
    "github.com/user/evolution/config"
)

type Game struct {
    sim      *sim.Simulation
    exporter *sim.SnapshotExporter
    renderer *render.Renderer
    hud      *HUD
    input    *InputHandler
    lastTick uint64
    paused   bool
    cfg      config.Config
}

func (g *Game) Update() error {
    g.input.Process(g)  // Geschwindigkeit, Pause, Next Step
    if !g.paused {
        g.sim.Step()
    }
    return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
    snap := g.exporter.Load()
    if snap != nil && snap.Tick != g.lastTick {
        g.renderer.RenderToBuffer(snap)  // nur bei neuem Tick
        g.lastTick = snap.Tick
    }
    g.renderer.DrawBuffer(screen)  // immer
    g.hud.Draw(screen, snap)       // Statistik-Overlay
}

func (g *Game) Layout(ow, oh int) (int, int) {
    return g.cfg.WorldWidth * TileSize, g.cfg.WorldHeight * TileSize
}
```

**HUD (Statistik-Panel):**

```
┌─────────────────────────┐
│ Tick:    12345           │
│ Pop:     487             │
│ Ø Speed: 2.34            │
│ Ø Sight: 6.12            │
│ Ø Effic: 1.05            │
│ [Pause] [Next] [1x][5x] │
└─────────────────────────┘
```

**Input-Handler:**

```go
type InputHandler struct{}

func (h *InputHandler) Process(g *Game) {
    // Space       → Toggle Pause
    // Right-Arrow → Next Step (wenn paused)
    // + / -       → Geschwindigkeit ±1 TPS
    // Escape      → Beenden
}
```

### Akzeptanzkriterien M9

- [x] Fenster öffnet sich und zeigt simulierte Welt
- [x] HUD zeigt korrekte Tick- und Populationszahlen
- [x] Pause/Weiter funktioniert
- [x] Next-Step sichtbar wenn paused
- [x] Fenster schließt sauber mit Escape / Window-Close

---

## Meilenstein 10 — `cmd/evolution` (Binary-Einstiegspunkt)

> **Abhängigkeiten:** M2, M4, M7, M9
> **Letzter Meilenstein Stufe 1 — MVP vollständig.**

### Dateien

| Datei | Inhalt |
|---|---|
| `cmd/evolution/main.go` | Alles verdrahten, Ebiten-Run |

### Implementierungsdetails

```go
package main

import (
    "log"
    "github.com/hajimehoshi/ebiten/v2"
    "github.com/user/evolution/config"
    "github.com/user/evolution/gen"
    "github.com/user/evolution/sim"
    "github.com/user/evolution/render"
    "github.com/user/evolution/ui"
)

func main() {
    // 1. Config laden (DefaultConfig + optionale TOML-Datei)
    cfg := config.DefaultConfig()
    if err := cfg.Validate(); err != nil {
        log.Fatal("Invalid config:", err)
    }

    // 2. RandSource initialisieren (seeded für Determinismus)
    rng := newSeededRand(cfg.Seed)

    // 3. Welt generieren
    tiles := gen.GenerateWorld(cfg, rng)

    // 4. Simulation initialisieren
    simulation := sim.New(cfg, tiles, rng)

    // 5. Renderer und Game erstellen
    renderer := render.NewRenderer(cfg.WorldWidth, cfg.WorldHeight, TileSize)
    game := ui.NewGame(simulation, renderer, cfg)

    // 6. Ebiten starten
    ebiten.SetWindowTitle("Evolution Simulation")
    ebiten.SetWindowSize(cfg.WorldWidth*TileSize, cfg.WorldHeight*TileSize)
    if err := ebiten.RunGame(game); err != nil {
        log.Fatal(err)
    }
}
```

### Akzeptanzkriterien M10 — MVP vollständig

- [x] `go build ./cmd/evolution/` kompiliert ohne Fehler
- [x] `./evolution` startet, zeigt bunte Welt
- [x] Evolution sichtbar: Farbveränderung der Population über Zeit
- [x] Alle 5 CI-Gates sind grün
- [x] `go test -race ./...` grün
- [x] Population stirbt nicht sofort aus (Smoke-Test: 1000 Ticks, Population > 0)

---

## Post-MVP Erweiterungen — außerplanmäßig umgesetzt

> Diese Features wurden nach M10 (MVP) implementiert, sind aber kein Bestandteil
> der ursprünglichen Stufe-1-Roadmap. Sie gehören thematisch zu Stufe 2/3.

### Dynamische Verwüstung (`sim/world`, `config`, `sim`)

- `ApplyDesertification(desertifyThreshold, recoverThreshold float32) int` in `sim/world/world.go`
- Wiesen-Tiles mit `Food/FoodMax < DesertifyThreshold` werden zu Wüste
- Wüsten-Tiles mit `Food/FoodMax > RecoverThreshold` erholen sich zurück zu Wiese
- Hysterese durch zwei verschiedene Schwellwerte verhindert Flackern
- Neue Config-Felder: `DesertifyThreshold = 0.05`, `RecoverThreshold = 0.50`
- `sim.Step()` ruft `ApplyDesertification` nach `ApplyRegrowth` auf

### Konfigurierbare Regrowth-Raten (`config`)

- `RegrowthMeadow = 0.002`, `RegrowthDesert = 0.0005` als eigene Config-Felder
- Nahrung wächst bewusst wesentlich langsamer als eine mittelgroße Population frisst

### Verlaufsdiagramm (`ui/hud.go`)

- Diagramm unterhalb der Karte (`ChartHeight = 160 px`)
- Zeigt gesamten Simulationsverlauf seit Start (dynamisch wachsender Slice, Downsampling auf Chartbreite)
- Drei Kurven auf gemeinsamer 0–100%-Achse: Population (% von MaxPop), Nahrung (Ø Füllstand), Wüste (% der Land-Tiles)
- Gitternetz bei 25 / 50 / 75 %

### Rechte Seitenleiste (`ui/hud.go`, `ui/game.go`)

- `SidebarWidth = 200 px` neben der Karte
- Statistik-Panel, Ansichts-Schalter, Legende, Parameter-Panel

### Vier Ansichtsmodi (`render/viewmode.go`, `render/renderer.go`, `ui/input.go`)

- Taste **1** Biom — Geländetyp + Nahrungsfüllstand + Individuen-Punkte
- Taste **2** Dichte — Populationsdichte pro Tile als Heatmap
- Taste **3** Genotyp — Ø Gene pro Tile als RGB (R=Speed, G=Sight, B=Effizienz)
- Taste **4** Nahrung — Nahrungsfüllstand biomunabhängig
- Seitenleiste zeigt aktiven Modus hervorgehoben, Legende passt sich an (ADR-010)

### Bugfixes

- Race Condition auf `s.rng`: `lockedRandSource` (Mutex-Wrapper) für Phase-1-Goroutinen
- `BaseEnergyCost` von 3.0 auf 0.5 korrigiert (Population kollabierte bei zu hohen Kosten)
- Energie-Drain fehlte: `applyPhase2` schreibt nun `BaseEnergyCost + speedGene×0.1` zurück

---

## Meilenstein 11 — Stufe 2: Räuber & Beute

> **Abhängigkeiten:** M10 (MVP vollständig)
> **GitHub:** [Milestone M11](https://github.com/Sternrassler/evolution/milestone/2)

### Issues

| # | Titel | Abhängigkeiten |
|---|---|---|
| [#18](https://github.com/Sternrassler/evolution/issues/18) | ADR-011 — Predator-Agent-Architektur → [ADR-011](adr/ADR-011-predator-agent-architektur.md) ✅ | — |
| [#3](https://github.com/Sternrassler/evolution/issues/3) | feat(entity): GeneAggression + EntityType ✅ | Blocked by #18 |
| [#4](https://github.com/Sternrassler/evolution/issues/4) | feat(entity): EventAttack + EventFlee ✅ | Blocked by #18 |
| [#6](https://github.com/Sternrassler/evolution/issues/6) | feat(config): PredatorConfig-Felder ✅ | Blocked by #18 |
| [#5](https://github.com/Sternrassler/evolution/issues/5) | feat(predator): Predator-Agent implementieren ✅ | #18, #3, #4 |
| [#7](https://github.com/Sternrassler/evolution/issues/7) | feat(sim): Räuber-Beute-Integration ✅ | #3, #4, #5, #6 |

### Neue Dateien / Änderungen

| Datei | Änderung |
|---|---|
| `sim/entity/gene.go` | `GeneAggression GeneKey = 3`, `NumGenes = 4` |
| `sim/entity/individual.go` | Neues Feld `EntityType` (Herbivore/Predator) |
| `sim/entity/event.go` | Neuer `EventAttack` |
| `sim/predator/predator.go` | `State`-Value-Type + `Tick(State, ctx, out)`-Funktion (ADR-011) |
| `config/config.go` | `PredatorConfig` mit `InitialPredators`, `EnergyPerKill`, `ReproThreshold`, `ReproReserve`, `MaxSight` |
| `sim/partition/worker.go` | `RunPredatorPhase1()` — Räuber-Tick sequentiell |
| `sim/sim.go` | `applyPhase2`: `EventAttack`-Auflösung, EntityType-Guard |
| `render/renderer.go` | Räuber-Darstellung rot (255,60,60) in Biom-View |
| `ui/hud.go` | Räuber-Statistik, 4. Chart-Linie (rot), Legende |

### Implementierungsdetails

**Predator als Agent:**

```go
package predator

// Predator implementiert das sim.Agent-Interface — Tick-Loop unverändert
type Predator struct {
    entity.Individual
}

func (p *Predator) Tick(ctx sim.WorldContext, out *entity.EventBuffer) {
    // Jagdverhalten:
    // 1. IndividualsNear nach Herbivoren suchen
    // 2. Nahe Herbivore angreifen (EventAttack)
    // 3. Zu wenig Nahrung → random walk
    // 4. Energie > Schwelle → Reproduktion (mit Aggression-Gen)
}
```

**GeneAggression-Effekt (in Individual.Tick):**

```go
case entity.GeneAggression:
    // Fluchttendenz: bei Aggression < 0.5 → Flucht-Verhalten wenn Räuber nah
    // Kampftendenz: bei Aggression > 0.5 → Verteidigung (für Stufe 2 Herbivore)
```

**Neue Events:**

```go
EventAttack  EventType = 4   // Räuber greift Herbivore an
EventFlee    EventType = 5   // Herbivore flieht
```

### Akzeptanzkriterien M11

- [x] Räuber-Population entsteht und überlebt
- [x] Räuber-Beute-Dynamik sichtbar (Lotka-Volterra-ähnliche Schwingung)
- [x] Herbivore-`GeneAggression` evolviert sichtbar unter Räuber-Druck
- [x] `go test -race ./...` grün
- [x] Alle CI-Gates weiterhin grün

---

## Meilenstein 12 — Stufe 3: Umweltbedingungen

> **Abhängigkeiten:** M11
> **GitHub:** [Milestone M12](https://github.com/Sternrassler/evolution/milestone/1)

### Issues

| # | Titel | Abhängigkeiten |
|---|---|---|
| [#10](https://github.com/Sternrassler/evolution/issues/10) | feat(config): SeasonConfig + CatastropheConfig | — |
| [#8](https://github.com/Sternrassler/evolution/issues/8) | feat(world): AdvanceEnvironment() — Saison-Zyklus | #10 |
| [#9](https://github.com/Sternrassler/evolution/issues/9) | feat(world): Katastrophen (Dürre, Flut, Seuche) | #10 |
| [#11](https://github.com/Sternrassler/evolution/issues/11) | feat(world): Feuchtigkeitsgradient — Nachwuchsrate in Wassernähe | — |

### Neue Dateien / Änderungen

| Datei | Änderung |
|---|---|
| `sim/world/world.go` | `AdvanceEnvironment()`, `Season`-Typ |
| `sim/world/world.go` | `Tile.FoodGrowthRate` als Funktion von `(biome, season)` |
| `sim/world/catastrophe.go` | `Catastrophe`-Typen, `ApplyCatastrophe()` |
| `config/config.go` | `SeasonConfig`, `CatastropheConfig` |

### Implementierungsdetails

```go
type Season uint8

const (
    SeasonSpring Season = iota
    SeasonSummer
    SeasonAutumn
    SeasonWinter
)

// AdvanceEnvironment wird vor Phase 1 aufgerufen
// Berechnet aktuelle Season aus Tick-Zähler
// Passt FoodGrowthRate für alle Tiles an
func (g *Grid) AdvanceEnvironment(tick uint64, cfg config.SeasonConfig) {
    season := seasonFromTick(tick, cfg.TicksPerSeason)
    for i := range g.Tiles {
        g.Tiles[i].FoodGrowthRate = biomeSeasonRate(g.Tiles[i].Biome, season)
    }
}

// Katastrophen (zufällig ausgelöst via Config-Wahrscheinlichkeit)
type CatastropheType uint8

const (
    CatastropheDrought CatastropheType = iota  // Nahrung halbieren
    CatastropheFlood                           // Wasser breitet sich aus
    CatastropheDisease                         // 20% Population stirbt
)
```

### Akzeptanzkriterien M12

- [ ] Saisonale Schwankungen sichtbar in Populationsgröße
- [ ] Katastrophen auslösbar via Konfiguration
- [ ] Energieerhaltungs-Invariante weiterhin gültig (AdvanceEnvironment ändert Rates, nicht absolute Energie)
- [ ] `go test ./...` grün

---

## Meilenstein 13 — Stufe 4: Karten-Editor

> **Abhängigkeiten:** M12
> **GitHub:** [Milestone M13](https://github.com/Sternrassler/evolution/milestone/3)

### Issues

| # | Titel | Abhängigkeiten |
|---|---|---|
| [#19](https://github.com/Sternrassler/evolution/issues/19) | ADR-012 — Editor-Modus State-Modellierung | — |
| [#12](https://github.com/Sternrassler/evolution/issues/12) | feat(gen): EditorSource implementiert TileSource | #19 |
| [#14](https://github.com/Sternrassler/evolution/issues/14) | feat(ui): Taste E togglet Editor-Modus | #19, #12 |
| [#13](https://github.com/Sternrassler/evolution/issues/13) | feat(ui): Editor-Modus mit Maus-Interaktion | #19, #12, #14 |

### Neue Dateien / Änderungen

| Datei | Änderung |
|---|---|
| `gen/editor.go` | `EditorSource` implementiert `TileSource`-Interface |
| `ui/editor.go` | Editor-Modus, Maus-Interaktion, Biom-Pinsel |
| `ui/game.go` | Toggle Editor-Modus (Taste E) |

### Implementierungsdetails

```go
// EditorSource implementiert TileSource — austauschbar mit ProceduralSource
// Keine Architektur-Änderung nötig
type EditorSource struct {
    tiles  []world.Tile
    width  int
    height int
}

func (e *EditorSource) Generate(cfg config.Config, rng config.RandSource) []world.Tile {
    return e.tiles  // gibt manuell gesetzte Tiles zurück
}

func (e *EditorSource) SetTile(x, y int, biome world.BiomeType) {
    e.tiles[y*e.width+x] = world.Tile{Biome: biome, FoodMax: biomeDefaultFood(biome)}
}
```

**UI Maus-Interaktion:**

```
Linke Maustaste + Drag  → Biom malen
Rechte Maustaste        → Biom-Auswahl-Dropdown
R                       → Simulation neu starten mit aktueller Karte
```

### Akzeptanzkriterien M13

- [ ] Editor-Modus öffnet sich mit Taste E
- [ ] Biome malbar, Simulation restartbar mit neuer Karte
- [ ] `ProceduralSource` und `EditorSource` austauschbar ohne Architektur-Änderung
- [ ] `go test ./...` grün

---

## Meilenstein 14 — Stufe 5: Detailansicht

> **Abhängigkeiten:** M13
> **GitHub:** [Milestone M14](https://github.com/Sternrassler/evolution/milestone/4)

### Issues

| # | Titel | Abhängigkeiten |
|---|---|---|
| [#15](https://github.com/Sternrassler/evolution/issues/15) | feat(sim): LineageTracker als TickObserver | — |
| [#16](https://github.com/Sternrassler/evolution/issues/16) | feat(ui): Inspector-Panel (Stammbaum + Genverlauf) | #15 |
| [#17](https://github.com/Sternrassler/evolution/issues/17) | feat(ui): Klick auf Individuum öffnet Inspector | #15, #16 |

### Neue Dateien / Änderungen

| Datei | Änderung |
|---|---|
| `sim/lineage.go` | `LineageTracker` — implementiert `TickObserver` |
| `ui/inspector.go` | `Inspector`-Panel, Stammbaum-Darstellung |
| `ui/game.go` | Klick auf Individuum → Inspector öffnen |

### Implementierungsdetails

```go
// LineageTracker: separater Concern via TickObserver
// Konsumiert Birth/Death-Events, baut Stammbaum auf
// Kein ParentID-Hack in Gene-Metadaten
type LineageTracker struct {
    parents  map[uint64]uint64  // ID → ParentID
    births   map[uint64]uint64  // ID → Tick
    deaths   map[uint64]uint64  // ID → Tick
    geneMaps map[uint64][entity.NumGenes]float32
}

func (lt *LineageTracker) OnTick(tick uint64, stats sim.TickStats) { ... }

// Inspector: zeigt Stammbaum, Genverlauf, Lebensgeschichte
// Zugriff via sim.World.Inspector(id) — read-only
type Inspector struct {
    tracker *LineageTracker
}

func (i *Inspector) DrawLineage(screen *ebiten.Image, id uint64, depth int) { ... }
func (i *Inspector) DrawGeneHistory(screen *ebiten.Image, id uint64) { ... }
```

**Zoom-abhängige Darstellung (erweitert):**

```
Klick auf Individuum (TileSize >= 4) → Inspector-Panel öffnet sich
Inspector zeigt:
  - Aktuelle Gene (numerisch + Farbbalken)
  - Alter (Ticks)
  - Energie-Level
  - Stammbaum (3 Generationen rückwärts)
```

### Akzeptanzkriterien M14

- [ ] Klick auf Individuum öffnet Inspector
- [ ] Stammbaum wird korrekt angezeigt (min. 3 Generationen)
- [ ] Genverlauf über Zeit sichtbar
- [ ] `LineageTracker` fügt keine Overhead-Allokationen im Hot-Path hinzu (separater Observer)
- [ ] `go test ./...` grün

---

## Anhang A — Meilenstein-Abhängigkeitsgraph

```
M0 (CI)
│
M1 (sim/entity) ──────────────────────────────────────────────┐
│                                                              │
M2 (config) ──────────────────────────────────────────────────┤
│                                                              │
M3 (sim/world) ←── M1, M2 ────────────────────────────────────┤
│                                                              │
M4 (gen) ←── M2, M3                                           │
│                                                              │
M5 (testutil) ←── M1, M2, M3, M4                              │
│                                                              │
M6 (sim/partition) ←── M1, M2, M3, M5                         │
│                                                              │
M7 (sim) ←── M1, M2, M3, M4, M5, M6 ← ALLE CI-GATES GRÜN     │
│                                                              │
M8 (render) ←── M1, M3, M7 ←─────────────────────────────────┘
│
M9 (ui) ←── M2, M7, M8
│
M10 (cmd/evolution) ←── M2, M4, M7, M9  ← MVP VOLLSTÄNDIG
│
M11 (Räuber) ←── M10
│
M12 (Umwelt) ←── M11
│
M13 (Editor) ←── M12
│
M14 (Detail) ←── M13
```

**Kein Zyklus im Graph.** Topologische Ordnung: M0 → M1 → M2 → M3 → M4 → M5 → M6 → M7 → M8 → M9 → M10 → M11 → M12 → M13 → M14.

---

## Anhang B — Testpyramide-Zielverteilung

```
         ▲
        /S\   Simulations-Tests ~10%
       /---\  Full-Run-Szenarien, Emergenz,
      / INT \  Determinismus-Regression
     /-------\
    /   25%   \ Integrations-Tests
   /  Partition-\ Sync, Energie-Erhaltung,
  / Sync·Entity  \ Entity-Lifecycle
 /   Lifecycle   \
/─────────────────\
      65%          Unit-Tests
  Pure Funktionen  Genetik, Energie, Bewegung
  rapid-Properties Nahrungsverbrauch, Bounds
```

**Verteilungsziele pro Meilenstein:**

| Meilenstein | Unit | Integration | Simulation |
|---|---|---|---|
| M1 (entity) | 100% | — | — |
| M2 (config) | 100% | — | — |
| M3 (world) | 90% | 10% | — |
| M4 (gen) | 100% | — | — |
| M5 (testutil) | 100% | — | — |
| M6 (partition) | 70% | 30% | — |
| M7 (sim) | 40% | 40% | 20% |
| M8–M10 | 60% | 25% | 15% |

---

## Anhang C — Rapid Property-Tests Übersicht

| Property | Package | Beschreibung |
|---|---|---|
| `PropMutationBounds` | `sim` | `Genes[i] ∈ [Min, Max]` nach beliebig vielen Mutationen |
| `PropEnergyConservation` | `sim` | `ΔE_ind + ΔE_tiles + E_tote == E_regrown` über beliebige Tick-Sequenzen |
| `PropSpatialConsistency` | `sim/world` | Kein Individuum auf Wasser-Tile nach einem Tick |
| `PropPopulationMonotony` | `sim` | Bei `FoodMax = ∞` sinkt Population nie (kein Verhungern) |
| `PropPhase2Idempotency` | `sim` | Zweimaliges Anwenden derselben Events ändert nichts |
| `PropGenerateWorldBiomDist` | `gen` | Biom-Verteilung innerhalb ±20% der Config-Wahrscheinlichkeiten |
| `PropFoodNeverNegative` | `sim/world` | `Tile.Food >= 0` nach beliebig vielen `ApplyRegrowth()` + Eat-Events |
| `PropPartitionConsistency` | `sim/partition` | Jede ID existiert genau einmal über alle Partitionen |
| `PropGhostRowFreshness` | `sim/partition` | Ghost-Row-Daten == echte Nachbarpartitions-Daten nach Copy |

---

## Anhang D — Allokations-Budgets

| Funktion | Ziel Allocs | Enforcement | Meilenstein |
|---|---|---|---|
| `partition.RunPhase1()` | 0 | Benchmark + 50%-Regressions-Gate | M6 |
| `agent.Tick()` | 0 | Benchmark + 50%-Regressions-Gate | M7 |
| `WorldSnapshot`-Export | 0 | Benchmark + 50%-Regressions-Gate | M7 |
| `SpatialGrid.Rebuild()` | 0 | Benchmark + 50%-Regressions-Gate | M3 |
| `WorldSnapshot.Hash()` | 0 | Benchmark + 50%-Regressions-Gate | M5 |
| `Phase 2 Event-Apply` | ≤ Births | Wird bei FreeList-Reuse 0 | M7 |
| `render.RenderToBuffer()` | 0 | Benchmark + 50%-Regressions-Gate | M8 |
| `GenerateWorld()` | unbegrenzt | Einmaliger Aufruf — kein Gate | M4 |

**Enforcement-Strategie:** Benchmarks mit `testing.B.ReportAllocs()`. Bei >50% Allokations-Regression im Vergleich zur Baseline: CI Gate 5 schlägt fehl. Kein absolut harter `AllocsPerRun == 0`-Gate — robuster gegen Go-Compiler-Updates.

---

## Anhang E — Häufige Stolperfallen

### 1. Circular Imports

**Problem:** `sim` → `sim/partition` → `sim` (zirkulär).

**Lösung:** `sim/entity` als Leaf-Package. Beide Packages importieren `entity`, nicht gegenseitig.

**Regel:** Wenn Package A ein Typ aus Package B braucht und B auch A importiert → Typ in ein drittes Package auslagern.

### 2. Globaler `math/rand` vergessen

**Problem:** `rand.Float64()` direkt aufgerufen → nicht deterministisch, nicht testbar.

**Erkennung:** CI Gate 2 (`check_global_rand.go`) schlägt fehl.

**Lösung:** Immer `RandSource`-Interface injizieren. Niemals `rand.Float64()` in `sim/`-Packages.

### 3. SoA/AoS-Grenze verschwimmt

**Problem:** `Individual`-Pointer aus `sim/entity` direkt in SoA-Arrays — Aliasing, Cache-Misses.

**Regel:** SoA intern in `sim/partition`, AoS `Individual` nur im `WorldSnapshot`. Konvertierung explizit in `ToIndividuals()` und `BuildPartition()`.

### 4. `WorldSnapshot` nach Export mutiert

**Problem:** `SnapshotExporter.Store()` aufgerufen, dann Slice weiter befüllt → Draw() sieht inkonsistenten State.

**Regel:** Nach `atomic.Store()` niemals mehr in den gespeicherten Snapshot schreiben. 2-Buffer-Pool stellt sicher, dass der alte Buffer nicht mehr beschrieben wird, bevor der nächste Tick ihn übernimmt.

### 5. Ghost-Row-K falsch berechnet

**Problem:** `K < max(MaxSpeedRange, MaxSightRange)` → Individuen sehen Daten jenseits der Ghost-Zone → falsche Entscheidungen.

**Regel:** `K = max(MaxSpeedRange, MaxSightRange)`. `config.Validate()` prüft `WorldHeight / NumPartitions >= 2 * K`.

### 6. Phase 2 parallelisiert bevor profiled

**Problem:** Vorzeitige Optimierung — Phase 2 ist oft <30% der Tick-Zeit.

**Regel:** Phase 2 bleibt sequentiell im MVP. Optimierungsschwelle: pprof zeigt Phase 2 >30% → dann parallelisieren. Dokumentierter Pfad in `sim/sim.go`.

### 7. `GeneDef` als globale Registry

**Problem:** `RegisterGeneEffect(key, func)` → globaler State, nicht testbar, Func-Pointer verhindert Inlining.

**Lösung:** `switch/case` auf `GeneKey` im Tick-Code. Neue Gene: neue Konstante + neuer case-Branch + `GeneDef` in Config. Keine Registry.

### 8. `testworld` mockt `WorldContext`

**Problem:** Mock gibt fest kodierte Werte zurück → Tests prüfen Implementierungs-Details, nicht Semantik.

**Regel:** `testworld`-Package baut echte, leichtgewichtige `WorldContext`-Implementierung. Tests prüfen echte Semantik. Keine `interface{...}`-Mocks.

### 9. `atomic.Pointer` ohne Happens-Before

**Problem:** Schreibvorgänge vor `Store()` werden nach `Load()` nicht sichtbar, weil kein Memory-Fence.

**Tatsache:** `atomic.Pointer.Store()` in Go garantiert Happens-Before für alle vorherigen Schreibvorgänge. Dokumentiert, kein manueller Fence nötig.

### 10. Ebiten-Tests in CI headless

**Problem:** Tests die `ebiten.RunGame()` aufrufen, schlagen fehl ohne Display.

**Lösung:** Render-Logik in `RenderToBuffer()` (nur Pixel-Buffer, kein Ebiten) und `DrawBuffer()` (Ebiten) aufteilen. Nur `RenderToBuffer()` in Unit-Tests testen. `DrawBuffer()` nur in manuellen Smoke-Tests.

---

*Dieses Dokument ist die verbindliche Implementierungs-Roadmap. Alle Architektur-Entscheidungen sind in [ARCHITECTURE.md](./ARCHITECTURE.md) begründet.*
