package sim

import (
	"image"
	"sort"
	"sync"

	"github.com/Sternrassler/evolution/config"
	"github.com/Sternrassler/evolution/gen"
	"github.com/Sternrassler/evolution/sim/entity"
	"github.com/Sternrassler/evolution/sim/partition"
	"github.com/Sternrassler/evolution/sim/world"
)

// Compile-time check: worldContextImpl muss world.WorldContext implementieren.
var _ world.WorldContext = (*worldContextImpl)(nil)

// lockedRandSource schützt einen RandSource mit einem Mutex für parallelen Zugriff in Phase 1.
type lockedRandSource struct {
	mu  sync.Mutex
	rng RandSource
}

func (l *lockedRandSource) Float64() float64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.rng.Float64()
}

func (l *lockedRandSource) Intn(n int) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.rng.Intn(n)
}

// Simulation orchestriert alle Partitionen und den Tick-Ablauf.
type Simulation struct {
	cfg        config.Config
	pendingCfg config.Config
	hasPending bool
	cfgMu      sync.Mutex

	grid        *world.Grid
	spatialGrid *world.SpatialGrid
	partitions  []*partition.Partition
	nextID      uint64

	tick      uint64
	observer  TickObserver
	exporter  *SnapshotExporter
	rng       RandSource       // für Phase 2 (sequentiell)
	phase1Rng *lockedRandSource // für Phase 1 (parallel)
}

// New erstellt eine neue Simulation. rng wird für Weltgenerierung und Phase 2 verwendet.
func New(cfg config.Config, rng RandSource, observer TickObserver) (*Simulation, *SnapshotExporter, error) {
	if err := cfg.Validate(); err != nil {
		return nil, nil, err
	}
	if observer == nil {
		observer = NoopObserver{}
	}

	// Weltgenerierung
	tiles := gen.GenerateWorld(cfg, rng)
	grid := world.NewGrid(cfg.WorldWidth, cfg.WorldHeight)
	copy(grid.Tiles, tiles)

	// Spatial Grid
	spatialGrid := world.NewSpatialGrid(cfg.SpatialCellSize, cfg.WorldWidth, cfg.WorldHeight)

	// Partitionen erstellen
	partitions := makePartitions(cfg)

	exporter := NewSnapshotExporter(len(grid.Tiles), cfg.MaxPopulation)

	s := &Simulation{
		cfg:         cfg,
		grid:        grid,
		spatialGrid: spatialGrid,
		partitions:  partitions,
		nextID:      1,
		observer:    observer,
		exporter:    exporter,
		rng:         rng,
		phase1Rng:   &lockedRandSource{rng: rng},
	}

	// Initiale Population
	s.spawnInitialPopulation()
	s.spawnInitialPredators()

	return s, exporter, nil
}

// makePartitions teilt die Welt in N horizontale Streifen.
func makePartitions(cfg config.Config) []*partition.Partition {
	n := cfg.NumPartitions
	rowsPerPart := cfg.WorldHeight / n
	parts := make([]*partition.Partition, n)
	for i := range n {
		start := i * rowsPerPart
		end := start + rowsPerPart
		if i == n-1 {
			end = cfg.WorldHeight // letzte Partition bekommt Rest
		}
		parts[i] = partition.NewPartition(cfg.MaxPopulation/n+10, start, end)
	}
	return parts
}

// partitionFor gibt den Partitions-Index für eine Y-Koordinate zurück.
func (s *Simulation) partitionFor(y int) int {
	n := len(s.partitions)
	rowsPerPart := s.cfg.WorldHeight / n
	idx := y / rowsPerPart
	if idx >= n {
		idx = n - 1
	}
	return idx
}

// spawnInitialPredators erzeugt die initialen Räuber.
func (s *Simulation) spawnInitialPredators() {
	cfg := s.cfg
	for range cfg.Predator.InitialPredators {
		x := s.rng.Intn(cfg.WorldWidth)
		y := s.rng.Intn(cfg.WorldHeight)
		tile := s.grid.At(x, y)
		if !tile.IsWalkable() {
			continue
		}
		genes := randomGenes(cfg.GeneDefinitions, s.rng)
		ind := entity.NewIndividual(s.nextID, image.Pt(x, y), genes, cfg.Predator.ReproReserve)
		ind.EntityType = entity.Predator
		s.nextID++
		pIdx := s.partitionFor(y)
		s.partitions[pIdx].AddIndividual(ind)
	}
}

// spawnInitialPopulation erzeugt die initiale Population.
func (s *Simulation) spawnInitialPopulation() {
	cfg := s.cfg
	for range cfg.InitialPop {
		x := s.rng.Intn(cfg.WorldWidth)
		y := s.rng.Intn(cfg.WorldHeight)
		// Nur auf begehbaren Tiles spawnen
		tile := s.grid.At(x, y)
		if !tile.IsWalkable() {
			continue
		}
		genes := randomGenes(cfg.GeneDefinitions, s.rng)
		ind := entity.NewIndividual(s.nextID, image.Pt(x, y), genes, cfg.ReproductionThreshold*0.5)
		s.nextID++
		pIdx := s.partitionFor(y)
		s.partitions[pIdx].AddIndividual(ind)
	}
}

// randomGenes erzeugt zufällige Gene im [Min, Max]-Bereich.
func randomGenes(defs []config.GeneDef, rng RandSource) [entity.NumGenes]float32 {
	var genes [entity.NumGenes]float32
	for i, def := range defs {
		if i >= entity.NumGenes {
			break
		}
		genes[i] = def.Min + float32(rng.Float64())*(def.Max-def.Min)
	}
	return genes
}

// Step führt einen vollständigen Simulations-Tick durch.
func (s *Simulation) Step() {
	// 1. Config-Swap
	cfg := s.swapPendingConfig()

	// 2. Ghost-Row-Copy
	s.copyGhostRows(cfg)

	// 3. Spatial-Grid-Rebuild
	allInds, globalRefs := s.collectAll()
	s.spatialGrid.Rebuild(allInds)

	// 4. Phase 1 — Herbivoren parallel
	var wg sync.WaitGroup
	for _, p := range s.partitions {
		wg.Add(1)
		go func(part *partition.Partition) {
			defer wg.Done()
			ctx := s.newWorldContext(part, cfg, s.phase1Rng)
			part.RunPhase1(ctx)
		}(p)
	}
	wg.Wait()

	// 4b. Räuber-Phase-1 — sequentiell (deterministisch, s.rng ohne Mutex)
	for _, p := range s.partitions {
		ctx := s.newWorldContext(p, cfg, s.rng)
		p.RunPredatorPhase1(ctx, cfg.Predator.ReproThreshold, cfg.Predator.ReproReserve, int32(cfg.Predator.MaxSight))
	}

	// 5. Phase 2 — sequentiell
	stats := s.applyPhase2(cfg, globalRefs)

	// 6. Regrowth (nach Phase 2)
	stats.EnergyRegrown = s.grid.ApplyRegrowth(cfg.RegrowthMeadow, cfg.RegrowthDesert)

	// Verwüstung / Erholung anwenden
	s.grid.ApplyDesertification(cfg.DesertifyThreshold, cfg.RecoverThreshold)

	// 7. Observer
	s.observer.OnTick(s.tick, stats)

	// 8. Snapshot-Export
	snap := s.buildSnapshot(stats)
	s.exporter.store(snap)

	s.tick++

	// 9. Integrity-Check
	if cfg.DebugIntegrity {
		s.checkIntegrity(cfg)
	}
}

// UpdateConfig ersetzt die Config beim nächsten Tick (thread-sicher).
func (s *Simulation) UpdateConfig(cfg config.Config) {
	s.cfgMu.Lock()
	s.pendingCfg = cfg
	s.hasPending = true
	s.cfgMu.Unlock()
}

func (s *Simulation) swapPendingConfig() config.Config {
	s.cfgMu.Lock()
	defer s.cfgMu.Unlock()
	if s.hasPending {
		s.cfg = s.pendingCfg
		s.hasPending = false
	}
	return s.cfg
}

// globalRef bildet einen globalen Index (aus SpatialGrid) auf Partition + SoA-Index ab.
type globalRef struct {
	part   *partition.Partition
	soaIdx int32
}

// allIndividuals sammelt alle lebenden Individuen über alle Partitionen.
func (s *Simulation) allIndividuals() []entity.Individual {
	inds, _ := s.collectAll()
	return inds
}

// collectAll gibt alle lebenden Individuen und die zugehörigen globalRefs zurück.
// Die Reihenfolge entspricht dem Rebuild-Input des SpatialGrid (Partitions-Reihenfolge → SoA-Index).
func (s *Simulation) collectAll() ([]entity.Individual, []globalRef) {
	var inds []entity.Individual
	var refs []globalRef
	for _, p := range s.partitions {
		for i := range p.Len {
			if !p.Alive[i] {
				continue
			}
			ind := entity.NewIndividual(p.IDs[i], image.Pt(int(p.X[i]), int(p.Y[i])), p.Genes[i], p.Energy[i])
			ind.EntityType = p.EntityType[i]
			inds = append(inds, ind)
			refs = append(refs, globalRef{part: p, soaIdx: int32(i)})
		}
	}
	return inds, refs
}

// copyGhostRows kopiert K Grenzzeilen zwischen benachbarten Partitionen.
// K = cfg.GhostK(). MVP: Stub — Ghost-Rows werden in worker.go nicht aktiv genutzt.
func (s *Simulation) copyGhostRows(_ config.Config) {
	// Voll implementiert wenn Phase-2-Boundary-Crossing getestet wird.
}

// newWorldContext erstellt einen WorldContext für eine Partition.
// rng wird explizit übergeben: phase1Rng (mutex) für parallele Herbivoren,
// s.rng (direkt) für sequentielle Predator-Ticks.
func (s *Simulation) newWorldContext(_ *partition.Partition, cfg config.Config, rng RandSource) *worldContextImpl {
	return &worldContextImpl{
		grid:        s.grid,
		spatialGrid: s.spatialGrid,
		rng:         rng,
		cfg:         cfg,
		nearBuf:     make([]int32, 0, 32),
	}
}

// worldContextImpl implementiert world.WorldContext für eine Partition in Phase 1.
type worldContextImpl struct {
	grid        *world.Grid
	spatialGrid *world.SpatialGrid
	rng         RandSource
	cfg         config.Config
	nearBuf     []int32
}

func (w *worldContextImpl) TileAt(p image.Point) world.Tile {
	if !w.grid.InBounds(p.X, p.Y) {
		return world.Tile{Biome: world.BiomeWater}
	}
	return *w.grid.At(p.X, p.Y)
}

func (w *worldContextImpl) IndividualsNear(p image.Point, radius int) []int32 {
	w.nearBuf = w.spatialGrid.IndividualsNear(p, radius, w.nearBuf)
	return w.nearBuf
}

func (w *worldContextImpl) Rand() entity.RandSource { return w.rng }
func (w *worldContextImpl) MutationRate() float32 {
	if len(w.cfg.GeneDefinitions) > 0 {
		return w.cfg.GeneDefinitions[0].MutationRate
	}
	return 0.1
}
func (w *worldContextImpl) ReproductionThreshold() float32 { return w.cfg.ReproductionThreshold }
func (w *worldContextImpl) MaxSpeed() float32              { return float32(w.cfg.MaxSpeedRange) }
func (w *worldContextImpl) MaxSight() float32              { return float32(w.cfg.MaxSightRange) }

// applyPhase2 wendet alle Events sequentiell an.
// Reihenfolge: EventDie → EventAttack → EventMove → EventEat → EventReproduce
// Konfliktauflösung: Last-Write-Loses (Essen), niedrigere ID gewinnt (Reproduktion)
// globalRefs bildet globale SpatialGrid-Indizes auf Partitions-SoA-Slots ab.
func (s *Simulation) applyPhase2(cfg config.Config, globalRefs []globalRef) TickStats {
	var stats TickStats

	// Alle Events sammeln (über alle Partitionen)
	type indexedEvent struct {
		event entity.Event
		part  *partition.Partition
	}

	var dies, attacks, moves, eats, reproduces []indexedEvent
	for _, p := range s.partitions {
		for _, ev := range p.Buf.Events() {
			ie := indexedEvent{event: ev, part: p}
			switch ev.Type {
			case entity.EventDie:
				dies = append(dies, ie)
			case entity.EventAttack:
				attacks = append(attacks, ie)
			case entity.EventMove:
				moves = append(moves, ie)
			case entity.EventEat:
				eats = append(eats, ie)
			case entity.EventReproduce:
				reproduces = append(reproduces, ie)
			}
		}
	}

	// Tod anwenden (Energie ≤ 0 in Phase 1)
	for _, ie := range dies {
		idx := ie.event.AgentIdx
		if int(idx) < ie.part.Len && ie.part.Alive[idx] {
			stats.EnergyLostToDeath += ie.part.Energy[idx]
			ie.part.MarkDead(idx)
			stats.Deaths++
		}
	}

	// Angriffe anwenden: Räuber tötet Herbivore, gewinnt Energie
	for _, ie := range attacks {
		predIdx := ie.event.AgentIdx
		if int(predIdx) >= ie.part.Len || !ie.part.Alive[predIdx] {
			continue
		}
		targetGlobalIdx := int(ie.event.Value)
		if targetGlobalIdx < 0 || targetGlobalIdx >= len(globalRefs) {
			continue
		}
		ref := globalRefs[targetGlobalIdx]
		if !ref.part.Alive[ref.soaIdx] {
			continue // Beute bereits tot
		}
		if ref.part.EntityType[ref.soaIdx] != entity.Herbivore {
			continue // Räuber greift keine Räuber an (inkl. sich selbst)
		}
		// Beute töten
		stats.EnergyLostToDeath += ref.part.Energy[ref.soaIdx]
		ref.part.MarkDead(ref.soaIdx)
		stats.Deaths++
		stats.Kills++
		// Räuber bekommt Energie
		ie.part.Energy[predIdx] += cfg.Predator.EnergyPerKill
	}

	// Bewegung anwenden
	for _, ie := range moves {
		idx := ie.event.AgentIdx
		if int(idx) >= ie.part.Len || !ie.part.Alive[idx] {
			continue
		}
		newPos := ie.event.TargetPos
		if !s.grid.InBounds(newPos.X, newPos.Y) {
			continue
		}
		// Partition-Wechsel prüfen
		newPartIdx := s.partitionFor(newPos.Y)
		oldPartIdx := -1
		for i, p := range s.partitions {
			if p == ie.part {
				oldPartIdx = i
				break
			}
		}
		ie.part.X[idx] = int32(newPos.X)
		ie.part.Y[idx] = int32(newPos.Y)
		ie.part.Age[idx]++

		// Boundary-Crossing: in neue Partition verschieben
		if newPartIdx != oldPartIdx && newPartIdx >= 0 && newPartIdx < len(s.partitions) {
			ind := entity.NewIndividual(
				ie.part.IDs[idx],
				newPos,
				ie.part.Genes[idx],
				ie.part.Energy[idx],
			)
			ind.EntityType = ie.part.EntityType[idx] // EntityType erhalten
			ie.part.MarkDead(idx)
			s.partitions[newPartIdx].AddIndividual(ind)
		}
	}

	// Essen anwenden (nur Herbivore fressen; Räuber essen kein Gras)
	for _, ie := range eats {
		idx := ie.event.AgentIdx
		if int(idx) >= ie.part.Len || !ie.part.Alive[idx] {
			continue
		}
		pos := ie.event.TargetPos
		if !s.grid.InBounds(pos.X, pos.Y) {
			continue
		}
		tile := s.grid.At(pos.X, pos.Y)
		eaten := ie.event.Value
		if eaten > tile.Food {
			eaten = tile.Food
		}
		tile.Food -= eaten
		efficiencyGene := ie.part.Genes[idx][entity.GeneEfficiency]
		gain := eaten * efficiencyGene
		ie.part.Energy[idx] += gain
		stats.EnergyConsumed += eaten
	}

	// Reproduktion anwenden (niedrigere ID gewinnt bei Konflikt auf gleicher Pos)
	type reproRequest struct {
		pos        image.Point
		id         uint64
		part       *partition.Partition
		idx        int32
		entityType entity.EntityType
	}

	reproByPos := make(map[image.Point]reproRequest)
	for _, ie := range reproduces {
		idx := ie.event.AgentIdx
		if int(idx) >= ie.part.Len || !ie.part.Alive[idx] {
			continue
		}
		et := ie.part.EntityType[idx]
		threshold := cfg.ReproductionThreshold
		if et == entity.Predator {
			threshold = cfg.Predator.ReproThreshold
		}
		if ie.part.Energy[idx] < threshold {
			continue
		}
		pos := ie.event.TargetPos
		id := ie.part.IDs[idx]
		existing, exists := reproByPos[pos]
		if !exists || id < existing.id {
			reproByPos[pos] = reproRequest{pos: pos, id: id, part: ie.part, idx: idx, entityType: et}
		}
	}

	// Sortiere nach ID für Determinismus
	winners := make([]reproRequest, 0, len(reproByPos))
	for _, r := range reproByPos {
		winners = append(winners, r)
	}
	sort.Slice(winners, func(i, j int) bool { return winners[i].id < winners[j].id })

	totalPop := s.totalPopulation()
	for _, r := range winners {
		if totalPop >= cfg.MaxPopulation {
			break
		}
		threshold := cfg.ReproductionThreshold
		reserve := cfg.ReproductionReserve
		if r.entityType == entity.Predator {
			threshold = cfg.Predator.ReproThreshold
			reserve = cfg.Predator.ReproReserve
		}
		if !r.part.Alive[r.idx] || r.part.Energy[r.idx] < threshold {
			continue
		}
		// Energie teilen
		r.part.Energy[r.idx] -= reserve
		childGenes := mutateGenes(r.part.Genes[r.idx], cfg.GeneDefinitions, s.rng)

		childPos := r.pos
		if !s.grid.InBounds(childPos.X, childPos.Y) || !s.grid.At(childPos.X, childPos.Y).IsWalkable() {
			childPos = image.Pt(int(r.part.X[r.idx]), int(r.part.Y[r.idx]))
		}

		child := entity.NewIndividual(s.nextID, childPos, childGenes, reserve)
		child.EntityType = r.entityType // Kind erbt EntityType
		s.nextID++
		pIdx := s.partitionFor(childPos.Y)
		s.partitions[pIdx].AddIndividual(child)
		stats.Births++
		totalPop++
	}

	// Energiekosten anwenden — MUSS nach Essen passieren damit Nahrung zählt.
	// Phase 1 berechnet Kosten nur lokal; hier werden sie auf SoA-Arrays geschrieben.
	// Räuber: 0.8 + speed×0.15 (höherer Körperaufwand, ADR-011)
	// Herbivore: BaseEnergyCost + speed×0.1
	for _, p := range s.partitions {
		for i := range p.Len {
			if !p.Alive[i] {
				continue
			}
			speedGene := p.Genes[i][entity.GeneSpeed]
			var cost float32
			if p.EntityType[i] == entity.Predator {
				cost = float32(0.8) + speedGene*0.15
			} else {
				cost = cfg.BaseEnergyCost + speedGene*0.1
			}
			p.Energy[i] -= cost
			stats.EnergyConsumed += cost
			if p.Energy[i] <= 0 {
				stats.EnergyLostToDeath += 0
				p.MarkDead(int32(i))
				stats.Deaths++
			}
		}
	}

	// Räuber zählen
	for _, p := range s.partitions {
		for i := range p.Len {
			if p.Alive[i] && p.EntityType[i] == entity.Predator {
				stats.Predators++
			}
		}
	}

	stats.Population = s.totalPopulation()
	return stats
}

// mutateGenes klont Eltern-Gene und appliziert Gauss-ähnliche Störung.
func mutateGenes(parent [entity.NumGenes]float32, geneDefs []config.GeneDef, rng RandSource) [entity.NumGenes]float32 {
	child := parent
	for i, def := range geneDefs {
		if i >= entity.NumGenes {
			break
		}
		if rng.Float64() < float64(def.MutationRate) {
			delta := float32(rng.Float64()*2-1) * def.MutationStep
			child[i] = clamp32(parent[i]+delta, def.Min, def.Max)
		}
	}
	return child
}

func clamp32(v, lo, hi float32) float32 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func (s *Simulation) totalPopulation() int {
	total := 0
	for _, p := range s.partitions {
		total += p.LiveCount()
	}
	return total
}

// buildSnapshot erstellt einen WorldSnapshot aus dem aktuellen Zustand.
func (s *Simulation) buildSnapshot(stats TickStats) WorldSnapshot {
	inds := s.allIndividuals()
	sort.Slice(inds, func(i, j int) bool { return inds[i].ID < inds[j].ID })
	var foodSum float32
	for _, t := range s.grid.Tiles {
		if t.Biome == world.BiomeWater {
			continue
		}
		stats.LandTiles++
		if t.FoodMax > 0 {
			foodSum += t.Food / t.FoodMax
		}
		if t.Biome == world.BiomeDesert {
			stats.DesertTiles++
		}
	}
	if stats.LandTiles > 0 {
		stats.AvgFoodPct = foodSum / float32(stats.LandTiles) * 100
	}
	return WorldSnapshot{
		Tiles:       s.grid.Tiles,
		Individuals: inds,
		Tick:        s.tick,
		Stats:       stats,
	}
}

// checkIntegrity prüft Invarianten wenn DebugIntegrity=true.
func (s *Simulation) checkIntegrity(_ config.Config) {
	seen := make(map[uint64]bool)
	for _, p := range s.partitions {
		for i := range p.Len {
			if !p.Alive[i] {
				continue
			}
			id := p.IDs[i]
			if seen[id] {
				panic("integrity: duplicate individual ID")
			}
			seen[id] = true
		}
	}
}
