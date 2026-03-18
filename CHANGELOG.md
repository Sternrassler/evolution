# Changelog

Alle wesentlichen Änderungen an diesem Projekt werden hier dokumentiert.
Format basiert auf [Keep a Changelog](https://keepachangelog.com/de/1.1.0/).

## [Unreleased]

### Fixed

- Energie-Drain fehlte: `applyPhase2` in `sim/sim.go` schreibt nun `BaseEnergyCost + speedGene*0.1` zurück auf SoA-Arrays; Individuen sterben jetzt korrekt an Energiemangel
- `BaseEnergyCost` von 1.0 auf 3.0 erhöht, damit der Selektionsdruck spürbar bleibt

### Added

- `ui/hud.go`: Farblegende unten rechts — Geländetypen (Wiese/Wüste/Wasser) und Gen-Bedeutung (Rot=Speed, Grün=Sight, Blau=Effizienz) mit farbigen Swatches (`vector.FillRect`)
- `Makefile`: `build`- und `run`-Targets mit X11-Linker-Workaround (`/tmp/extralibs`)

### Changed

- `CONCEPT.md`, `ARCHITECTURE.md`, `ROADMAP.md` nach `docs/` verschoben; alle internen Links aktualisiert

### Added

- `CONTRIBUTING.md`: Branching, Commit-Konventionen, Code-Konventionen, Meilenstein-Checkliste
- `docs/adr/ADR-008`: Tick-Loop-Steuerung — synchrones `Step()` in `Update()`, Pause/Speed-API
- `docs/adr/ADR-009`: Rendering-Strategie — Pixel-Buffer + Zoom-abhängige Darstellung
- `docs/GLOSSAR.md`: Definitionen aller zentralen Begriffe
- **M0 CI-Grundgerüst**: `tools/check_global_rand.go` (Gate 2), `tools/check_ebiten_imports.go` (Gate 1), `.github/workflows/ci.yml`, `Makefile`
- **M1** `sim/entity`: `GeneKey`/`NumGenes`, `Individual` (AoS), `Event`/`EventType`/`EventBuffer` — alle Tests grün, zero-alloc im Hot-Path bestätigt
- **M2** `config`: `Config`-Struct mit TOML-Tags, `GeneDef`, `DefaultConfig()`, `GhostK()`, `Validate()` — alle Tests grün inkl. Property-Test (rapid)
- **M3** `sim/world`: `BiomeType`, `Tile`, `Grid`, `ApplyRegrowth()`, `SpatialGrid` (Flat-Bucket, O(n) Rebuild, zero-alloc `IndividualsNear`), `WorldContext`-Interface — alle Tests grün inkl. Property-Test (Energieerhaltung, rapid)
- **M4** `gen`: `GenerateWorld()`, `TileSource`-Interface, `ProceduralSource` mit Cellular-Automaton (3 Iterationen, Majority-Rule, Rand-Behandlung via Wasser) — alle Tests grün inkl. Property-Tests (Food-Invariante, Biom-Verteilung, rapid); deterministische Ausgabe bei gleichem Seed bestätigt
- **M5 (partial)** `testworld`: `Builder` + `WorldCtx` — echte `WorldContext`-Implementierung für Tests (kein Mock), Builder-API (`New`, `WithTile`, `WithIndividual`, `WithRng`, `WithConfig`), `IndividualsNear` mit linearem Scan, deterministischer Default-RNG (Seed 42), Out-of-bounds-Fallback in `TileAt` — alle Tests grün
- **M6** `sim/partition` + `sim/testutil.BuildPartition`: `Partition`-Struct (SoA-Hot-Arrays, FreeList, pre-allokierter `EventBuffer`), `GhostRow`, `AddIndividual`/`MarkDead`/`LiveCount`/`ToIndividuals` (SoA→AoS), `RunPhase1` mit internem `agent`-Typ (Bewegung, Nahrungssuche, Essen, Reproduktion, Tod), `clampStep`-Hilfsfunktion; `sim/testutil.BuildPartition` (AoS→SoA für Tests); `BenchmarkRunPhase1` bestätigt 0 allocs/op; `go test -race` grün
- **M7** `sim`: `Simulation` (Koordinator), `Step()` mit Phase-1-Parallelisierung (WaitGroup) und Phase-2-Sequenz (Die→Move→Eat→Reproduce), Boundary-Crossing, `SnapshotExporter` (2-Buffer-Pool, `atomic.Pointer`, lock-frei), `WorldSnapshot.Hash()` (FNV-1a, deterministisch), `worldContextImpl` (implementiert `world.WorldContext`), `mutateGenes`/`clamp32`, `TickStats`, `TickObserver`/`NoopObserver`; `sim/testutil.HashSnapshot` hinzugefügt; CI Gate 3 (Determinismus) grün; `go test -race ./sim/...` grün
- **M8** `render`: `color.go` (`BiomeColor`, `GeneColor`, `normalizeGene`), `renderer.go` (`Renderer` mit pre-allokiertem Pixel-Buffer, `RenderToBuffer`, `DrawBuffer`, `ScreenSize`); Tests für Farb-Normalisierung und Biom-Farben grün (`go test -tags noebiten ./render/...`); `ebiten`-Import hinter `//go:build !noebiten`-Tag für headless-CI-Kompatibilität
- **M9** `ui`: `game.go` (`Game`-Struct, implementiert `ebiten.Game`-Interface, synchrones `Step()` in `Update()` per ADR-008), `hud.go` (`HUD` mit `ebitenutil.DebugPrint`, Tick-/Populationsstatistiken und Durchschnittsgene), `input.go` (`InputHandler` mit Space=Pause, ArrowRight=NextStep, Escape=Beenden via `inpututil.IsKeyJustPressed`)
- **M10** `cmd/evolution`: `main.go` (MVP-Binary-Einstiegspunkt: `DefaultConfig`, seeded RandSource, `sim.New`, `render.NewRenderer`, `ui.NewGame`, `ebiten.RunGame` mit 20 TPS); `go build ./cmd/evolution/` grün

## [0.1.0] — 2026-03-17

### Added

- Projektspezifikation: `CONCEPT.md`, `ARCHITECTURE.md`, `ROADMAP.md`
- 7 Architecture Decision Records (`docs/adr/ADR-001` bis `ADR-007`)
- Serena-Projektkonfiguration (`.serena/project.yml`)
- MIT-Lizenz
- README mit Projektübersicht und Meilenstein-Tabelle
- `CHANGELOG.md`
