# Changelog

Alle wesentlichen Änderungen an diesem Projekt werden hier dokumentiert.
Format basiert auf [Keep a Changelog](https://keepachangelog.com/de/1.1.0/).

## [Unreleased]

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

## [0.1.0] — 2026-03-17

### Added

- Projektspezifikation: `CONCEPT.md`, `ARCHITECTURE.md`, `ROADMAP.md`
- 7 Architecture Decision Records (`docs/adr/ADR-001` bis `ADR-007`)
- Serena-Projektkonfiguration (`.serena/project.yml`)
- MIT-Lizenz
- README mit Projektübersicht und Meilenstein-Tabelle
- `CHANGELOG.md`
