# ADR-005: Phase 2 bleibt sequentiell im MVP

- **Datum:** 2026-03-17
- **Status:** Accepted

---

## Kontext

Der Tick-Ablauf ist in zwei Phasen unterteilt:

**Phase 1 (parallel):** Jede Partition-Goroutine ruft `agent.Tick()` auf,
liest die Welt read-only und schreibt Events in einen pre-allozierten `EventBuffer`.
Parallelisierung ist trivial und sicher, weil keine shared mutable state existiert.

**Phase 2 (sequentiell):** Der Koordinator wendet alle Events aus allen Partitionen an:
- Bewegungen durchführen, Energiebilanz berechnen
- Konflikte auflösen (Nahrung: last-write-loses; Reproduktion: niedrigere ID gewinnt)
- Boundary-Crosser reassignieren (Individuen, die Partitionsgrenzen überschritten haben)
- `TickStats` aggregieren

Phase 2 hat potenziell parallelisierbare Teile: Intra-Partition-Events könnten
unabhängig voneinander parallel angewendet werden (jede Partition mutiert nur
ihren eigenen State). Cross-Partition-Events (Boundary-Crosser, Nahrungskonflikte
an Grenzen) bleiben sequentiell.

Die Frage: Soll Phase 2 bereits im MVP parallelisiert werden?

---

## Entscheidung

**Phase 2 bleibt im MVP vollständig sequentiell.**

Dokumentierter Optimierungspfad (in `sim/sim.go` als Kommentar):

```go
// Phase 2: Events anwenden (sequentiell)
// Optimierungspfad: Wenn pprof zeigt Phase 2 > 30% der Tick-Zeit,
// Intra-Partition-Events parallelisieren. Cross-Partition-Events
// (Boundary-Crosser, Grenzkonflikte) bleiben sequentiell.
// Stand MVP: Keine vorzeitige Optimierung.
stats := s.applyPhase2(cfg)
```

Auslösepunkt für Parallelisierung: `pprof` zeigt Phase 2 >30% der Gesamttick-Zeit
bei der Ziel-Konfiguration (200×200, 5000 Individuen, 20 TPS).

---

## Konsequenzen

**Positiv:**
- Einfacherer Code in Phase 2 — kein Partitions-Locking für Intra-Partition-State
- Konfliktauflösung deterministisch und einfach testbar
- Boundary-Crosser-Logik ohne concurrent map access
- Schnellere Implementierung im MVP — Phase 2 Parallelisierung ist komplex
  (Lock-Granularität, Deadlock-Risiko zwischen Partitionen bei Grenzfällen)
- Korrekt nach Messung optimieren statt spekulativ

**Negativ:**
- Bei hoher Individuenzahl (>5000) könnte Phase 2 zum Bottleneck werden
- Parallelisierung später nachrüsten erfordert Refactoring von `applyPhase2()`

**Messbare Grenze:**
Phase 2 besteht typischerweise aus O(n) Event-Applies. Bei 5000 Individuen mit
durchschnittlich 2 Events/Tick = 10 000 Event-Applies. Benchmark baseline wird
in M7 gemessen. Wenn Phase 1 (parallel, ~200μs) vs. Phase 2 (sequentiell, ~?μs)
im Verhältnis >3:1 steht, ist Optimierung sinnlos.

---

## Verworfene Alternativen

### A: Phase 2 sofort parallelisieren

Intra-Partition-Events parallel, Cross-Partition-Events sequentiell.
Problem: Boundary-Crosser-Erkennung braucht Zugriff auf Nachbarpartitionen.
Nahrungskonflikte an Partitionsgrenzen brauchen Koordination. Das Locking-Schema
ist nicht trivial. Bei N=4 Partitionen kaum messbarer Gewinn bei O(n)-Phase-2.

### B: Work-Stealing-Queue für Phase 2

Jede Partition processed ihre eigenen Events, Boundary-Events in shared Queue.
Maximale Parallelisierung, aber: erhebliche Komplexität, potenzielle Deadlocks,
schwer zu testen. Optimierungspfad nur wenn A sich als unzureichend erweist.

### C: Kein Profiling, intuitiv parallelisieren

"Phase 2 könnte langsam werden, also jetzt optimieren."
Lehrbuchfall für premature optimization. Die Latenz von Phase 1 (echte
parallele Arbeit) dominiert fast sicher über Phase 2 (sequentielle Apply-Logik).
Messung zuerst.
