# Contributing

> Aktuell Solo-Projekt. Dieses Dokument definiert die eigenen Arbeitsregeln
> und dient als Referenz bei späteren Beiträgen Dritter.

---

## Voraussetzungen

```bash
go mod tidy          # Abhängigkeiten aktualisieren
make ci              # Alle Gates müssen grün sein (Gates 3+5 bis M6/M7 via || true)
```

Benötigt: Go 1.22+, `make`.

---

## Branching

```
main          ← immer lauffähig, alle CI-Gates grün
feature/MX-*  ← Meilenstein-Feature, z.B. feature/M1-entity
fix/*         ← Bugfix außerhalb eines Meilensteins
```

Kein Commit direkt auf `main` für Features ab M1. Einzel-Commits für
Dokumentations-/Konfigurationsänderungen sind auf `main` erlaubt.

---

## Commit-Konventionen

Format: `<type>(<scope>): <was in Präsens>`

| Typ | Wann |
|---|---|
| `feat` | Neues Feature (neue Funktion, neues Package) |
| `fix` | Bugfix |
| `test` | Tests hinzugefügt oder korrigiert |
| `refactor` | Kein Verhalten geändert, kein Bug gefixt |
| `docs` | Nur Dokumentation |
| `ci` | CI-Gates, Makefile, Tooling |
| `chore` | Abhängigkeiten, go.mod, .gitignore |

Beispiele:
```
feat(sim/entity): Individual-Struct und GeneKey-Konstanten
test(sim/partition): Property-Tests für Mutations-Bounds (rapid)
fix(gen): RandSource-Injection in GenerateWorld
ci: Gate 1 – ebiten-Import-Check via check_ebiten_imports.go
```

**Pflicht vor jedem Commit:** `CHANGELOG.md` aktualisieren (Unreleased-Abschnitt).

---

## Code-Konventionen

Alle Konventionen sind verbindlich in `CLAUDE.md` und `docs/ARCHITECTURE.md` dokumentiert.
Kurzfassung der häufigsten Fehler:

| Falsch | Richtig |
|---|---|
| `rand.Float64()` in sim/ | `ctx.Rand().Float64()` |
| `import "github.com/hajimehoshi/ebiten/v2"` in sim/ | Nicht erlaubt |
| `new(EventBuffer)` im Hot-Path | Partition-eigenen Buffer übergeben |
| Weltmutation in Phase 1 | Nur Events in Buffer schreiben |
| Mocks für `WorldContext` | `testworld.New(w,h).Build()` |

Gate 1 und Gate 2 in `make ci` schlagen bei Verletzung fehl.

---

## Meilenstein-Checkliste

Vor dem Merge/Commit eines Meilensteins:

- [ ] `make ci` grün (alle aktiven Gates)
- [ ] `CHANGELOG.md` aktualisiert
- [ ] Neue Packages: Import-Richtung gegen `ARCHITECTURE.md` geprüft
- [ ] Neue Gene: Schritte in `CLAUDE.md` § "Gen-System erweitern" befolgt
- [ ] Neue ADRs für nicht-offensichtliche Entscheidungen in `docs/adr/`

---

## Pull Requests

Titel: gleiche Konvention wie Commits.
Beschreibung: Verweis auf Meilenstein aus `docs/ROADMAP.md`, betroffene ADRs, messbare
Allokations-Budgets falls Hot-Path betroffen.
