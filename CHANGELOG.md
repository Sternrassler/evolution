# Changelog

Alle wesentlichen Änderungen an diesem Projekt werden hier dokumentiert.
Format basiert auf [Keep a Changelog](https://keepachangelog.com/de/1.1.0/).

## [Unreleased]

### Added

- `README.md`: vollständig überarbeitet — Build-Anleitung, Steuerung, Ansichtsübersicht, aktueller Meilenstein-Status
- `CONTRIBUTING.md`: für externe Beitragende überarbeitet — Schnelleinstieg, Merge-Kriterien, Anleitungen für neue Gene und ADRs
- `.github/ISSUE_TEMPLATE/bug_report.md` + `feature_request.md`: strukturierte Issue-Vorlagen
- `.github/PULL_REQUEST_TEMPLATE.md`: PR-Checkliste mit CI, CHANGELOG, Import-Prüfung
- `CLAUDE.md`: öffentlicher Hinweis auf KI-Arbeitsweise ergänzt

- `docs/adr/ADR-010`: ViewMode — umschaltbare Kartenansichten statt Zoom-Dispatch; ADR-009 mit Nachtrag versehen
- `docs/GLOSSAR.md`: neue Einträge `ViewMode` und `Verwüstung`; `TickStats`-Eintrag um `AvgFoodPct`, `DesertTiles`, `LandTiles` erweitert
- `docs/ROADMAP.md`: Status auf MVP vollständig (M0–M10 ✅) aktualisiert; Post-MVP-Erweiterungen dokumentiert
- `docs/ARCHITECTURE.md`: `TickStats`-Klassendiagramm, `Game`-Struct (Sektion 7) und `render`-Package-Beschreibung aktualisiert

- Vier Kartenansichten, umschaltbar per Taste 1–4:
  - `1` Biom: Geländetyp + Nahrungsfüllstand mit Individuen-Punkten (Standard)
  - `2` Dichte: Populationsdichte pro Tile als Heatmap (schwarz → rot → orange → gelb)
  - `3` Genotyp: Durchschnittsgene aller Individuen pro Tile als RGB (R=Speed, G=Sight, B=Effizienz)
  - `4` Nahrung: Nahrungsfüllstand biomunabhängig (grau → grün)
- `render/viewmode.go`: `ViewMode`-Typ mit `ViewBiom`/`ViewDichte`/`ViewGenotyp`/`ViewNahrung` und `ViewName()` (kein Ebiten-Build-Tag, headless-kompatibel)
- `render/renderer.go`: `RenderToBuffer(snap, mode ViewMode)` — dispatcht auf `renderTiles+renderIndividuals`, `renderDichte`, `renderGenotyp`, `renderNahrung`; pre-allokierte Hilfs-Buffer (`densityBuf`, `geneSumBuf`, `geneCountBuf`) in Renderer-Struct
- `render/color.go`: `DensityColor(count, maxCount)` (Heatmap schwarz→gelb), `FoodOnlyColor(biome, food, foodMax)` (biomunabhängiger Füllstand)
- `ui/hud.go`: Ansichts-Schalter in Seitenleiste zeigt aktive Ansicht hervorgehoben; Legende passt sich an aktive Ansicht an
- `ui/input.go`: Tasten 1–4 schalten `g.viewMode`

### Changed

- Diagramm: Zeitfenster wächst jetzt dynamisch mit der Simulationszeit — der gesamte Verlauf seit Start wird angezeigt. Historydaten in unbegrenzt wachsendem Slice; beim Zeichnen Downsampling auf Chartbreite (gleichmäßige Indexverteilung).

### Added

- `docs/REGELKREISE.md`: fachliche und mathematische Beschreibung aller Regelkreise (Energie, Nahrung, Verwüstung) inkl. Gleichgewichtsbedingungen, Wechselwirkungen und Parametertabelle

### Fixed

- Diagramm: Nahrungskurve zeigte "Tiles mit Food > 0" (blieb nahe 100% trotz Verwüstung); ersetzt durch durchschnittlichen Füllstand `Food/FoodMax × 100` — fällt jetzt korrekt wenn Tiles verarmen

### Changed

- Diagramm-Kurven auf gemeinsame 0–100%-Achse umgestellt: Population (% von MaxPop), Nahrung (% der Land-Tiles mit Food > 0), Wüste (% der Land-Tiles); Gitternetz bei 25/50/75%
- `sim/snapshot.go`: `TotalFood float32` → `FoodTiles int` + `LandTiles int` (Tile-Zähler statt Summe)

### Added

- `config/config.go`: `DesertifyThreshold` (0.05) und `RecoverThreshold` (0.50) — steuern dynamische Verwüstung und Erholung von Biomen
- `sim/world/world.go`: `ApplyDesertification(desertifyThreshold, recoverThreshold float32) int` — wandelt Wiesen bei Nahrungsmangel in Wüste um und Wüsten bei ausreichend Nahrung zurück; gibt Anzahl der Wüsten-Tiles zurück
- `sim/sim.go`: `Step()` ruft nach `ApplyRegrowth` automatisch `ApplyDesertification` auf
- `ui/hud.go`: Verlaufsdiagramm unterhalb der Karte (`ChartHeight = 160`); dritte Kurve zeigt Wüsten-Tile-Anzahl (orange-braun) statt Wüstennahrung; Parameter-Panel zeigt Verwüstungs- und Erholungsschwellen
- `ui/hud.go`: `ChartHeight`-Konstante (exportiert) für Layout-Berechnungen in `game.go` und `main.go`

### Changed

- `sim/snapshot.go`: `TickStats.DesertFood float32` ersetzt durch `DesertTiles int`
- `ui/hud.go`: `NewHUD(mapW int)` → `NewHUD(mapW, mapH int)`; Chart aus Seitenleiste entfernt und als eigenständiger Block unterhalb der Karte neu implementiert
- `ui/game.go`: `Layout()` gibt `h + ChartHeight` zurück; `NewHUD` erhält `mapH`
- `cmd/evolution/main.go`: `SetWindowSize` berücksichtigt `ChartHeight`

- `ui/hud.go`: Parameter-Panel unten links — zeigt BaseEnergyCost, Repro-Schwelle/-Reserve, Regrowth-Raten und Max-Population
- `config/config.go`: `RegrowthMeadow` (0.002) und `RegrowthDesert` (0.0005) als konfigurierbare Felder; `ApplyRegrowth` verwendet diese Werte statt interner Konstanten

### Changed

- Regrowth-Raten drastisch reduziert: Wiese 0.05→0.002, Wüste 0.01→0.0005 — Nahrung wächst jetzt wesentlich langsamer als eine mittelgroße Population frisst
- `sim/world.ApplyRegrowth()` nimmt nun `meadowRate, desertRate float32` als Parameter (aus `config.Config`)

### Fixed

- Energie-Drain fehlte: `applyPhase2` in `sim/sim.go` schreibt nun `BaseEnergyCost + speedGene*0.1` zurück auf SoA-Arrays; Individuen sterben jetzt korrekt an Energiemangel
- `BaseEnergyCost` von 1.0 auf 3.0 erhöht, dann nach Beobachtung auf 0.5 korrigiert (3.0 ließ Population auf ~25 kollabieren)
- **Race Condition auf `s.rng`**: Phase-1-Goroutinen teilten sich den RNG ohne Mutex → korrupte Positionen, alle sammelten sich in der linken oberen Ecke. Behoben mit `lockedRandSource` (Mutex-Wrapper) für Phase 1; Phase 2 verwendet weiterhin den ungeschützten `s.rng` (sequentiell)

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
