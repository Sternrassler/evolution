# ADR-004: `RandSource`-Interface-Injection statt `math/rand`

- **Datum:** 2026-03-17
- **Status:** Accepted

---

## Kontext

Die Simulation enthält stochastische Prozesse in jedem Tick:

- Bewegung (zufällige Richtung wenn keine Nahrung sichtbar)
- Mutation (ob ein Gen mutiert, wie stark)
- Reproduktions-Timing
- Weltgenerierung (Biom-Verteilung, Cellular Automaton Seed)

Zwei Zufallsquellen stehen zur Wahl:

**Paketeigener globaler State (`math/rand`):**
```go
// Irgendwo in sim/partition/worker.go
dx := rand.Intn(3) - 1  // -1, 0, oder 1
```
Einfach, null Setup. Aber: globaler State, nicht thread-safe ohne Lock,
nicht deterministisch über Runs, nicht injizierbar in Tests.

**Injiziertes Interface:**
```go
type RandSource interface {
    Float64() float64
    Intn(n int) int
}
// Aufruf:
dx := ctx.Rand().Intn(3) - 1
```
Mehr Setup, aber vollständig deterministisch und testbar.

Kernproblem: Die Simulation soll **reproduzierbar** sein — gleicher Seed,
gleiche Welt, identisches Ergebnis nach N Ticks. Das ist die Grundlage von
CI Gate 3 (Determinismus-Test) und für Bugrepros unerlässlich.

---

## Entscheidung

**`RandSource` ist ein Interface, das überall injiziert wird.**

Definition in `sim/` (wird von `config/` ebenfalls re-exportiert für `gen`):

```go
type RandSource interface {
    Float64() float64
    Intn(n int) int
}
```

Regeln:
1. Kein `math/rand`-Direktaufruf in `sim/`, `sim/partition/`, `sim/world/`, `gen/`
2. `RandSource` wird über `WorldContext.Rand()` an Agents durchgereicht
3. `GenerateWorld(cfg Config, rng RandSource)` — pure function, RNG als Parameter
4. CI Gate 2 (`check_global_rand.go`) prüft Einhaltung automatisch

**Konkrete Implementierungen:**
- Produktiv: `rand.New(rand.NewSource(seed))` wrapped in einem Adapter-Struct
- Tests: Deterministisch geseedeter `rand.New(rand.NewSource(42))`
- Property-Tests: `rapid`-internes RNG (wird automatisch geshrunk)

**Pro Partition ein `RandSource`:** Phase 1 läuft parallel. Jede Partition bekommt
eine eigene RNG-Instanz (von der übergeordneten geseedeten Quelle abgeleitet),
um Lock-Contention zu vermeiden und Determinismus zu erhalten.

---

## Konsequenzen

**Positiv:**
- CI Gate 3 möglich: `TestDeterminism` mit `-count=2` beweist identischen Hash
- Bugs reproduzierbar: Seed aus Log → exakte Wiederholung
- Tests kontrollieren Zufallszahlen exakt → keine flaky tests durch Entropie
- `rapid` shrinking funktioniert: RNG ist injiziert, rapid kann minimale
  Gegenbeispiele suchen
- Thread-safe: keine geteilten RNG-Instanzen zwischen Goroutinen

**Negativ:**
- Jede Funktion im Hot-Path braucht einen `ctx`-Parameter (oder direkt `rng`)
- Neuer Entwickler muss die Konvention kennen; Verstoß erst bei CI Gate 2 sichtbar
- Leicht mehr Boilerplate in Tests (RNG initialisieren)

**Durchsetzung:**
- `tools/check_global_rand.go`: AST-Walk, sucht `"math/rand"` in Import-Statements
  von Dateien unter `sim/`; gibt Exit-Code 1 bei Treffer
- Läuft in jedem CI-Durchlauf (Gate 2)

---

## Verworfene Alternativen

### A: `math/rand` mit globalem Lock

`sync.Mutex` um jeden `rand.Float64()`-Aufruf. Thread-safe, aber:
Lock-Contention in Phase 1 (N Goroutinen konkurrieren), nicht deterministisch
über Runs (OS-Scheduling bestimmt Reihenfolge). Abgelehnt.

### B: `math/rand/v2` mit `rand.New(rand.NewPCG(seed1, seed2))`

Moderner, besserer Algorithmus, thread-sicher per Instanz. Wäre ebenfalls
korrekt — das Interface-Design in dieser ADR ist mit v2 kompatibel. Der
Adapter-Struct wrapped schlicht eine v2-Instanz. Keine architektonische
Auswirkung; Implementierungsdetail.

### C: Zufallszahlen-Seed in Config, keine Interface-Injection

`Config.Seed` + globaler `var globalRNG = rand.New(...)`. Deterministisch,
aber nicht thread-safe in Phase 1 und nicht injizierbar für unit tests.
Das Interface ist der entscheidende Unterschied.
