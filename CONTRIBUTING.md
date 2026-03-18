# Beitragen zur Evolution Simulation

Danke für dein Interesse! Beiträge sind willkommen — egal ob Bugfix, neues Feature
oder Verbesserung der Dokumentation.

---

## Schnelleinstieg

```bash
git clone https://github.com/Sternrassler/evolution
cd evolution
make test-sim   # Tests ohne X11 (schnell, CI-kompatibel)
make build      # Binary bauen (braucht X11-Entwicklungsheader)
```

Benötigt: Go 1.22+, GCC, X11-Bibliotheken (siehe README.md).

---

## Vor dem Loslegen

**Kleine Änderungen** (Tippfehler, Doku, offensichtliche Bugs): direkt einen PR öffnen.

**Größere Änderungen** (neues Feature, Architekturänderung, neues Gen): bitte erst ein
[Issue](../../issues) öffnen und die Idee kurz beschreiben. Das spart Arbeit auf
beiden Seiten, falls die Richtung nicht passt.

---

## Branching

```
main            ← immer lauffähig, alle CI-Gates grün
feature/MX-*    ← Meilenstein-Feature, z.B. feature/M11-raeuber
fix/*           ← Bugfix
docs/*          ← nur Dokumentation
```

Kein Commit direkt auf `main` für Code-Änderungen. Dokumentations-Commits
sind auf `main` erlaubt.

---

## Commit-Konventionen

Format: `<typ>(<scope>): <was in Präsens>`

| Typ | Wann |
|---|---|
| `feat` | Neues Feature |
| `fix` | Bugfix |
| `test` | Tests hinzugefügt oder korrigiert |
| `refactor` | Kein Verhalten geändert, kein Bug gefixt |
| `docs` | Nur Dokumentation |
| `ci` | CI-Gates, Makefile, Tooling |
| `chore` | Abhängigkeiten, go.mod, .gitignore |

Beispiele:
```
feat(sim/entity): Individual-Struct und GeneKey-Konstanten
fix(gen): RandSource-Injection in GenerateWorld
docs: ADR-010 für ViewMode-Entscheidung
```

**Pflicht vor jedem Commit:** `CHANGELOG.md` im Abschnitt `[Unreleased]` aktualisieren.

---

## CI — alle Gates müssen grün sein

```bash
make ci        # Gates + Tests + Race-Detector
make test-sim  # schnelle Tests ohne X11
```

| Gate | Prüft |
|---|---|
| Gate 1 | Kein `ebiten`-Import in `sim/`, `gen/`, `config/` |
| Gate 2 | Kein `math/rand` direkt in `sim/` |
| Gate 3 | Determinismus: gleicher Seed → identischer Snapshot-Hash |
| Gate 4 | `go test -race ./sim/...` — keine Data Races |
| Gate 5 | Allokations-Budget: >50% Regression = Fail |

---

## Code-Konventionen

Die vollständigen Regeln stehen in [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md).
Die häufigsten Fallen:

| Falsch | Richtig |
|---|---|
| `rand.Float64()` in `sim/` | `ctx.Rand().Float64()` |
| `import "ebiten"` in `sim/` | Nicht erlaubt — CI schlägt fehl |
| `new(EventBuffer)` im Hot-Path | Partition-eigenen Buffer wiederverwenden |
| Weltmutation in Phase 1 | Nur Events in Buffer schreiben |
| Mocks für `WorldContext` | `testworld.New(w,h).Build()` verwenden |

**Neues Gen hinzufügen:**
1. `sim/entity/gene.go`: neue `GeneKey`-Konstante, `NumGenes` erhöhen
2. `config/config.go`: neuen `GeneDef{...}` in `GeneDefinitions`
3. Agent-Tick-Code: neuen `case`-Branch
4. Kein `RegisterGeneEffect()`, kein Func-Feld in `GeneDef`

---

## Pull Requests

- Titel folgt der Commit-Konvention: `feat(sim): ...`
- Beschreibung enthält: Was ändert sich? Warum? Welcher Meilenstein aus [`docs/ROADMAP.md`](docs/ROADMAP.md)?
- Hot-Path betroffen? → Benchmark-Ergebnisse angeben (`go test -bench -benchmem`)
- Neue nicht-offensichtliche Entscheidung? → ADR in `docs/adr/` anlegen

**Merge-Kriterien:**
- CI grün (alle aktiven Gates)
- `CHANGELOG.md` aktualisiert
- Import-Richtung gegen [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) geprüft

---

## Fragen?

Einfach ein Issue öffnen — kein Problem zu klein.
