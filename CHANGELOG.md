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

## [0.1.0] — 2026-03-17

### Added
- Projektspezifikation: `CONCEPT.md`, `ARCHITECTURE.md`, `ROADMAP.md`
- 7 Architecture Decision Records (`docs/adr/ADR-001` bis `ADR-007`)
- Serena-Projektkonfiguration (`.serena/project.yml`)
- MIT-Lizenz
- README mit Projektübersicht und Meilenstein-Tabelle
- `CHANGELOG.md`
