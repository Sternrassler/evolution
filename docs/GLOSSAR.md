# Glossar — Evolution Simulation

Kurzdefinitionen der zentralen Begriffe. Für Implementierungsdetails: `ARCHITECTURE.md`.

---

## Agent

Interface mit einer Methode: `Tick(ctx WorldContext, out *EventBuffer)`.
Einzige Verhaltensschnittstelle für simulierbare Akteure. Im MVP implementiert
von `Individual`. Ab Stufe 2 auch von `Predator`. Der Agent liest die Welt
read-only über `WorldContext` und schreibt Absichten als Events in `EventBuffer`.
Direkte Weltmutation ist in Phase 1 verboten.

---

## AoS (Array of Structs)

Datenorganisation, bei der Felder gebündelt in einem Struct liegen:
`[]Individual` — jedes Element enthält alle Felder eines Individuums zusammen.
Verwendet in `WorldSnapshot` und `sim/entity` (öffentliche API).
Gegenüber SoA cache-ungünstiger für Hot-Path-Iterationen über einzelne Felder,
aber leichter lesbar und als API-Typ geeignet. Siehe auch: **SoA**, **SoA/AoS-Grenze**.

---

## EventBuffer

Konkreter Struct (kein Interface) für zero-alloc Event-Sammlung in Phase 1.
Ein Buffer pro `Partition`, pre-allokiert mit `cap = MaxEventsPerTick`.
Wird zu Beginn jedes Ticks via `Reset()` geleert (setzt `len` auf 0, behält `cap`).
Der Agent ruft `Append(Event)` auf; der Koordinator liest `Events()` in Phase 2.
Konkreter Typ statt Interface ermöglicht Inlining im Hot-Path.

---

## GeneDef

Config-Struct mit Metadaten zu einem Gen: `Key GeneKey`, `Min`, `Max float32`,
`MutationRate float32`. Wird in `Config.GeneDefinitions []GeneDef` gespeichert.
Kein Func-Feld, keine Registry — Effect-Logik liegt als `switch/case` auf
`GeneKey` im Tick-Code (ADR-007).

---

## GeneKey

Integer-Konstante in `sim/entity/gene.go`, die ein Gen identifiziert (z.B.
`Speed`, `Sight`, `Efficiency`). Index in das `Genes [NumGenes]float32`-Array
eines Individuums. Neues Gen = neue Konstante + `NumGenes` erhöhen.

---

## Ghost-Row

Read-only Kopie von K Randzeilen einer `Partition`, die in den Ghost-Buffer
der Nachbarpartition kopiert wird. Ermöglicht Phase-1-Goroutinen, Nachbardaten
zu lesen ohne Data Race. K = `max(Config.MaxSpeedRange, Config.MaxSightRange)`.
Ghost-Rows sind maximal 1 Tick alt (ADR-006).

---

## Individual

Konkreter Struct in `sim/entity`. AoS-Repräsentation eines simulierten Lebewesens:
`ID uint64`, `Pos image.Point`, `Energy float32`, `Age int`,
`Genes [NumGenes]float32`, `alive bool`. Implementiert das `Agent`-Interface.
Öffentliche API-Darstellung; interne Darstellung in `sim/partition` ist SoA.

---

## Partition

Horizontaler Streifen der Spielwelt (N Zeilen). Besitzt SoA-Hot-Arrays,
Ghost-Rows und einen `EventBuffer`. Wird in Phase 1 von genau einer Goroutine
bearbeitet. Individuen, die die Partitionsgrenze überschreiten
(Boundary-Crosser), werden in Phase 2 reassigniert.

---

## Phase 1 / Phase 2

**Phase 1 (parallel):** Alle Partition-Goroutinen rufen `agent.Tick()` auf.
Lesen die Welt read-only (eigene SoA-Arrays + Ghost-Rows + Spatial-Grid).
Schreiben Absichten als Events in den Partition-eigenen `EventBuffer`.
Keine Weltmutation erlaubt.

**Phase 2 (sequentiell):** Der Koordinator wendet alle Events an, löst
Konflikte auf (Nahrung: last-write-loses; Reproduktion: niedrigere ID gewinnt),
reassigniert Boundary-Crosser, aggregiert `TickStats`, ruft `TickObserver` auf,
exportiert `WorldSnapshot` via `SnapshotExporter` (ADR-005).

---

## RandSource

Interface mit `Float64() float64` und `Intn(n int) int`. Einzige erlaubte
Zufallsquelle in `sim/`, `sim/partition/`, `sim/world/`, `gen/`.
Wird überall injiziert — kein `rand.Float64()` direkt (ADR-004).
CI Gate 2 (`check_global_rand.go`) schlägt bei Verletzung fehl.

---

## SnapshotExporter

Struct in `sim/` mit 2-Buffer-Pool und `atomic.Pointer[WorldSnapshot]`.
`sim.Step()` schreibt immer in den inaktiven Buffer, dann `atomic.Store()`.
`Game.Draw()` ruft `Load()` auf — lock-frei, kein Mutex, zero-alloc (ADR-003).

---

## SoA (Struct of Arrays)

Datenorganisation, bei der gleichartige Felder in separaten Slices liegen:
`X []int32`, `Y []int32`, `Energy []float32` statt `[]Individual`.
Verwendet intern in `sim/partition` für den Hot-Path. Cache-freundlich für
Iterationen über einzelne Felder (z.B. alle Energiewerte). Siehe auch: **AoS**.

---

## SoA/AoS-Grenze

Die explizite Konvertierungsgrenze zwischen interner SoA-Darstellung
(in `sim/partition`) und externer AoS-Darstellung (in `WorldSnapshot`).
Konvertierung findet ausschließlich in `Partition.ToIndividuals()` und
`testutil.BuildPartition()` statt. Nirgendwo sonst wird zwischen den
Darstellungen gewechselt (ADR-002).

---

## Spatial-Grid

Flat-Bucket-Array für räumliche Näherungsabfragen (`IndividualsNear`).
CellSize = `Config.SpatialCellSize` (Default: `Config.MaxSightRange`).
Vollständiger O(n)-Rebuild einmal pro Tick vor Phase 1. Pre-allokiert,
zero-alloc im Rebuild.

---

## TickObserver

Interface mit `OnTick(tick uint64, stats TickStats)`. Wird einmal pro Tick
nach Phase 2 aufgerufen. Dient als Test-Seam für Statistik-Assertions
(Recorder-Implementierung in Tests). Im MVP: HUD-Update und Statistik-Logging.

---

## TickStats

Wert-Struct mit aggregierten Statistiken eines Ticks: `Population int`,
`Births int`, `Deaths int`, `EnergyConsumed float32`, `EnergyLostToDeath float32`,
`EnergyRegrown float32`. Teil von `WorldSnapshot`. Basis für die
Energieerhaltungs-Invariante: `ΔEnergie_Individuen + ΔEnergie_Tiles + Energie_Tote = Regrowth`.

---

## TileSource

Interface mit `Generate(cfg Config, rng RandSource) []Tile`. Austauschbare
Karten-Quelle. `ProceduralSource` (MVP, Cellular Automaton) und künftiger
`EditorSource` (Stufe 4) implementieren dieses Interface.

---

## WorldContext

Interface, das jedem `Agent` in Phase 1 übergeben wird. Scoped read-only
Weltansicht: `TileAt`, `IndividualsNear`, `Rand()`, sowie simulationsrelevante
Config-Parameter als Methoden. Exponiert nicht die vollständige `Config` —
Agents sehen nur was sie brauchen. Echte Implementierung via `testworld`-Package
in Tests (keine Mocks).

---

## WorldSnapshot

Konkreter Struct, der den vollständigen Zustand nach einem Tick beschreibt:
`Tiles []Tile`, `Individuals []Individual`, `Tick uint64`, `Stats TickStats`.
Immutable nach `atomic.Store()` — kein Mutieren exportierter Snapshots.
Slices werden mit `cap = MaxPopulation` einmalig allokiert; danach nur
`len`-Anpassung. Producer: `sim.Step()`. Consumer: `render.Renderer`.
