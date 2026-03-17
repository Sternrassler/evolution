# CLAUDE.md — Evolution Simulation

## Projektstatus

Reine Spezifikation, **0% implementiert**. Alle Entscheidungen liegen in:

- `docs/CONCEPT.md` — fachliche Anforderungen
- `docs/ARCHITECTURE.md` — technische Architektur (maßgeblich)
- `docs/ROADMAP.md` — Implementierungsreihenfolge M0–M14
- `docs/adr/` — ADRs für nicht-offensichtliche Entscheidungen

---

## Package-Struktur (Abhängigkeitsrichtung)

```
cmd/evolution
  └── ui ──────────────── render
        └── sim ──────── sim/partition ── sim/entity  ← Leaf
              └── sim/world ──────────── sim/entity
gen ──── sim/world
config ─ (keine Projekt-Imports)

testworld, sim/testutil  ← nur in _test.go importieren
```

**Harte Grenzen — niemals verletzen:**

- `sim/`, `sim/partition/`, `sim/world/`, `sim/entity/`, `gen/` importieren **kein** `ebiten`
- `render/` importiert kein `ui/`
- `config/` importiert nichts aus dem Projekt
- `sim/entity/` importiert keine anderen `sim/`-Packages

---

## Kritische Konventionen

### RandSource — überall injiziert

Kein `rand.Float64()` oder `rand.Intn()` direkt in `sim/`, `gen/`:

```go
// ✗ Falsch
dx := rand.Intn(3) - 1

// ✓ Richtig
dx := ctx.Rand().Intn(3) - 1
```

CI Gate 2 (`tools/check_global_rand.go`) schlägt bei Verletzung fehl. → ADR-004

### SoA/AoS-Grenze

- **SoA** nur in `sim/partition` (intern, Hot-Path)
- **AoS** `entity.Individual` in `WorldSnapshot` (öffentliche API)
- Konvertierung ausschließlich in `Partition.ToIndividuals()` und `testutil.BuildPartition()`

→ ADR-002

### EventBuffer

- Ein `EventBuffer` pro Partition, pre-allokiert
- `Reset()` vor jedem `RunPhase1()`-Aufruf
- Kein `new(EventBuffer)` im Hot-Path — immer den Partition-eigenen Buffer übergeben

### Phase 1 / Phase 2

- Phase 1: **nur lesen** (Grid, GhostRows, WorldContext) + Events in Buffer schreiben
- Phase 2: **sequentiell**, Events anwenden, Konflikte lösen
- Keine Weltmutation in Phase 1 — kein Exception

→ ADR-005, ADR-006

---

## CI Gates

| Gate | Was | Wann grün |
|---|---|---|
| 1 | Kein `ebiten` in `sim/`, `gen/`, `config/` | sofort (M0) |
| 2 | Kein `math/rand` direkt in `sim/` | sofort (M0) |
| 3 | `TestDeterminism -count=2` | ab M7 |
| 4 | `go test -race ./sim/...` | ab M6 |
| 5 | Allokations-Budget-Benchmark | ab M6 |

`make ci` läuft durch (Gates 3+5 sind bis M6/M7 via `|| true` überbrückt).

---

## Test-Konventionen

- **Keine Mocks** für `WorldContext` — `testworld.New(w,h).Build()` liefert echte Implementierung
- `testutil.BuildPartition([]Individual, cfg)` für lesbare Partition-Tests (AoS→SoA)
- `testutil.HashSnapshot(snap)` für Determinismus-Assertions
- `TickObserver`-Recorder als Test-Seam für Stats-Assertions
- Property-Tests mit `pgregory.net/rapid` für: Mutations-Bounds, Energieerhaltung,
  Spatial-Konsistenz, Phase-2-Idempotenz

---

## Allokations-Ziele (Hot-Path)

| Funktion | Ziel |
|---|---|
| `partition.RunPhase1()` | 0 allocs |
| `agent.Tick()` | 0 allocs |
| `SnapshotExporter.Load/Store()` | 0 allocs |
| `SpatialGrid.Rebuild()` | 0 allocs |
| `render.RenderToBuffer()` | 0 allocs |

Regressions-Schwelle in CI: >50% Verschlechterung gegenüber Baseline = Fail.

---

## Gen-System erweitern (Stufe 2+)

1. `sim/entity/gene.go`: neue `GeneKey`-Konstante, `NumGenes` erhöhen
2. `config`: neuen `GeneDef{...}` in `GeneDefinitions`
3. Agent-Tick-Code: neuen `case NewGene:` Branch
4. Kein `RegisterGeneEffect()`, kein Func-Feld in `GeneDef` → ADR-007

---

## Meilenstein-Reihenfolge (Kurzform)

```
M0 CI → M1 entity → M2 config → M3 world → M4 gen →
M5 testutil → M6 partition → M7 sim → M8 render →
M9 ui → M10 cmd  ← MVP
M11 Räuber → M12 Umwelt → M13 Editor → M14 Detail
```

Vollständige Details: `docs/ROADMAP.md`
