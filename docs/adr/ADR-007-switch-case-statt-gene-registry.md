# ADR-007: `switch/case` auf `GeneKey` statt `RegisterGeneEffect()`-Registry

- **Datum:** 2026-03-17
- **Status:** Accepted

---

## Kontext

Jedes Individuum trägt Gene, deren Werte sein Verhalten pro Tick beeinflussen:
`GeneSpeed` bestimmt die maximale Bewegungsgeschwindigkeit, `GeneSight` den
Sichtradius, `GeneEfficiency` die Energieausbeute. Mit Stufe 2 kommt
`GeneAggression` hinzu.

Die Effect-Logik muss irgendwo von `GeneKey` + Wert auf ein Simulationsverhalten
abgebildet werden. Zwei grundlegende Ansätze:

**Registry-Pattern:**
```go
// In gen-Setup-Code:
RegisterGeneEffect(GeneSpeed, func(ind *Individual, val float32) {
    ind.maxSpeed = val
})

// Im Tick-Code:
for _, def := range cfg.GeneDefinitions {
    geneEffects[def.Key](ind, ind.Genes[def.Key])
}
```

**switch/case:**
```go
// Im Tick-Code direkt:
speed := ind.Genes[entity.GeneSpeed]
sight := int(ind.Genes[entity.GeneSight])
// ... speed und sight direkt verwenden
```

---

## Entscheidung

**`switch/case` auf `GeneKey` — keine Registry, kein Func-Feld in `GeneDef`.**

Effect-Logik lebt als `switch/case` im Agent-Tick-Code. Neues Gen hinzufügen:

1. `sim/entity/gene.go`: Neue `GeneKey`-Konstante, `NumGenes` erhöhen
2. `config`: Neuen `GeneDef{Key: NewGene, Min: ..., Max: ..., MutationRate: ...}` in `GeneDefinitions`
3. Agent-Tick-Code: Neuen `case NewGene:` Branch

Kein `RegisterGeneEffect()`, kein Func-Feld in `GeneDef`, kein globaler State.

---

## Konsequenzen

**Positiv:**
- **Kein Funktionszeiger im Hot-Path** — Go-Compiler kann `switch/case` auf
  `GeneKey` (kleiner int) als Jump-Table optimieren; direkte Func-Pointer-Calls
  verhindern Inlining und erzwingen indirekte Sprünge
- **Kein globaler State** — Registry wäre ein globales `map[GeneKey]func(...)`.
  Global State ist nicht thread-safe ohne Mutex und erschwert parallele Tests
- **Deterministisch** — Reihenfolge der Gene-Verarbeitung durch Array-Index definiert,
  nicht durch Registrierungs-Reihenfolge oder Map-Iteration (Go-Maps sind non-deterministisch)
- **Testbar** — Neues Gen: neuer Test für den neuen `case`-Branch, kein Mock-Registry nötig
- **Compiler-sichtbar** — `switch/case` mit konstantem Typ: Exhaustiveness via `golangci-lint`
  prüfbar; Registry-Aufruf ist zur Compile-Zeit opak

**Negativ:**
- Agent-Tick-Funktion wächst mit jedem neuen Gen (mehr Cases)
- Neue Gene erfordern Änderungen im Tick-Code (nicht rein additiv)
- Für Plugin-Architekturen (Nutzer-definierte Gene zur Laufzeit) ungeeignet —
  das ist aber explizit nicht im Scope dieser Simulation

**Skalierungsgrenze:** Bei >20 Genen (aktuell 3, absehbar ≤6) sollte die
switch/case-Logik in eine separate `applyGenes(ind, ctx)`-Funktion extrahiert werden.
Keine Architekturänderung nötig, nur Refactoring.

---

## Verworfene Alternativen

### A: Func-Feld in `GeneDef`

```go
type GeneDef struct {
    Key    GeneKey
    Effect func(ind *Individual, val float32)
    // ...
}
```

Scheinbar elegant, aber: Func-Feld bedeutet Interface-Dispatch im Hot-Path
(`agent.Tick()` wird 10 000× pro Tick aufgerufen). Go-Compiler kann Func-Pointer
nicht inlinen. Benchmark-Unterschied: direkte Feldnutzung ~0 ns, Func-Indirektion
~2–5 ns pro Gen — bei 3 Genen und 10k Individuen: ~150 μs extra pro Tick.

### B: `RegisterGeneEffect(key GeneKey, fn func(...))`

Globale Registry, aufgerufen in `init()` oder `main()`. Flexibel für Plugin-Szenarien.
Probleme: globaler State, nicht thread-safe, Go-Maps non-deterministisch iterierbar,
Func-Pointer-Overhead wie in A. Außerdem: Testaufrufe müssen Registry-State
bereinigen — fehleranfällig bei Parallel-Tests (`t.Parallel()`).

### C: Interface pro Gen

```go
type GeneEffect interface {
    Apply(ind *Individual, val float32)
}
```

Maximale Typsicherheit. Overhead: Interface-Dispatch (Vtable-Lookup) pro Gen pro
Individuum pro Tick. Schlimmer als Func-Pointer da zusätzlich Pointer-Indirektion
für Vtable. Für Hot-Path ungeeignet.
