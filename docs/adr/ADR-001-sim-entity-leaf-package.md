# ADR-001: `sim/entity` als Leaf-Package

- **Datum:** 2026-03-17
- **Status:** Accepted

---

## Kontext

Die Simulation benötigt einen gemeinsamen Datentyp `Individual`, der von mehreren Packages
gleichzeitig verwendet wird:

- `sim/partition` — speichert Individuen in SoA-Arrays
- `sim/world` — prüft, ob ein Tile von einem Individuum belegt ist
- `sim` — exportiert Individuen im `WorldSnapshot`

Läge `Individual` direkt im Package `sim`, entstünde ein zirkulärer Import:

```
sim → sim/partition → sim   ✗ (zirkulär)
sim → sim/world     → sim   ✗ (zirkulär)
```

Go verbietet zirkuläre Imports zur Compile-Zeit. Es braucht ein Package, das von allen
anderen importiert werden kann, ohne selbst eines von ihnen zu importieren.

Weitere Anforderung: Mit Stufe 2 (Räuber & Beute) soll ein zweiter Akteur `Predator`
hinzukommen, der denselben `Agent`-Interface-Vertrag erfüllt. Der Typname `individual`
wäre dann zu eng.

---

## Entscheidung

`Individual`, `GeneKey`, `GeneDef`, `Event` und `EventBuffer` werden in ein eigenes
Package `sim/entity` ausgelagert.

Regeln für dieses Package:
1. **Null Imports** auf andere `sim/`-Packages — keine Ausnahmen
2. Nur Go-Stdlib (`image`, grundlegende Typen)
3. Kein Verhalten außer dem, das direkt zu den Datentypen gehört
   (`IsAlive()`, `Kill()`, `EventBuffer.Append()` etc.)
4. CI Gate 1 (`check_ebiten_imports.go`) prüft: kein `ebiten`-Import hier

Der Name `entity` statt `individual` ist bewusst generisch gewählt, sodass
`Predator` und künftige Akteure ohne Umbenennung in dasselbe Package passen.

---

## Konsequenzen

**Positiv:**
- Kein zirkulärer Import möglich — Compile-Time-Garantie
- Einheitlicher Einstiegspunkt für alle Akteur-Typen (Herbivore, Predator, ...)
- Leaf-Package ist einfach zu testen (keine Abhängigkeiten zu mocken)
- Klare Ownership: Datenstrukturen hier, Verhalten in den jeweiligen Packages

**Negativ:**
- Ein zusätzliches Package in der Hierarchie
- Neue Entwickler suchen `Individual` zunächst in `sim/` und finden es nicht

**Folge-Entscheidungen:**
- `sim/testutil.BuildPartition()` konvertiert `[]entity.Individual` → `*partition.Partition`
  (AoS→SoA) — der Konvertierungsschritt ist eine direkte Konsequenz dieser Trennung
- Jede neue Entitätsart (Stufe 2+) bekommt ihren Struct in `sim/entity/`

---

## Verworfene Alternativen

### A: `Individual` in `sim/`

Führt zu zirkulären Imports sobald `sim/partition` oder `sim/world` den Typ braucht.
Go-Compiler verweigert das. Nicht umsetzbar.

### B: `Individual` in `sim/world`

`sim/partition` müsste `sim/world` importieren, was semantisch falsch ist
(`partition` ist kein Konsument der Welt, sondern ein paralleler Simulationsarbeiter).
Erzeugt falsche konzeptuelle Kopplung.

### C: Separates Top-Level-Package `entity/`

Möglich, aber `sim/entity` macht die Zugehörigkeit zur Simulations-Domäne
explizit. Ein reines `entity/`-Package auf Top-Level würde suggerieren, dass
`render/` oder `ui/` dieselben Typen direkt verwenden (sollten sie nicht).
