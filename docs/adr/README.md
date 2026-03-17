# Architecture Decision Records

Entscheidungen mit langfristiger Architektur-Wirkung, die im Code selbst
nicht sichtbar sind. Neue ADRs hier eintragen.

| # | Titel | Status | Thema |
|---|---|---|---|
| [ADR-001](ADR-001-sim-entity-leaf-package.md) | `sim/entity` als Leaf-Package | Accepted | Package-Topologie, Circular Imports |
| [ADR-002](ADR-002-soa-aos-dualitaet.md) | SoA in `sim/partition`, AoS in `sim/entity` | Accepted | Datenstruktur, Cache-Performance |
| [ADR-003](ADR-003-snapshot-sync-atomic-pointer.md) | `atomic.Pointer` + 2-Buffer-Pool | Accepted | Concurrency, Update/Draw-Sync |
| [ADR-004](ADR-004-randsource-injection.md) | `RandSource`-Interface-Injection | Accepted | Determinismus, Testbarkeit |
| [ADR-005](ADR-005-phase2-sequentiell.md) | Phase 2 sequentiell im MVP | Accepted | Performance, Parallelismus |
| [ADR-006](ADR-006-ghost-rows-partition-grenzen.md) | Ghost-Rows, 1-Tick-Latenz akzeptiert | Accepted | Partitionierung, Parallelismus |
| [ADR-007](ADR-007-switch-case-statt-gene-registry.md) | `switch/case` statt Registry | Accepted | Hot-Path, Erweiterbarkeit |

## Format

```
# ADR-NNN: Titel

- Datum: YYYY-MM-DD
- Status: Proposed | Accepted | Deprecated | Superseded by ADR-NNN

## Kontext
## Entscheidung
## Konsequenzen
## Verworfene Alternativen
```
