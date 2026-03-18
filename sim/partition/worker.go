package partition

import (
	"image"

	"github.com/Sternrassler/evolution/sim/entity"
	"github.com/Sternrassler/evolution/sim/predator"
	"github.com/Sternrassler/evolution/sim/world"
)

// RunPhase1 iteriert alle lebenden Herbivoren und führt ihre Tick-Logik aus.
// Schreibt Events in p.Buf (nach Reset). KEINE Weltmutation.
// Räuber werden NICHT verarbeitet — sie laufen sequentiell in RunPredatorPhase1.
func (p *Partition) RunPhase1(ctx world.WorldContext) {
	p.Buf.Reset()
	for i := int32(0); i < int32(p.Len); i++ {
		if !p.Alive[i] || p.EntityType[i] == entity.Predator {
			continue
		}
		a := agent{idx: i, p: p}
		a.tick(ctx, &p.Buf)
	}
}

// RunPredatorPhase1 verarbeitet alle lebenden Räuber sequentiell.
// Hängt Events an p.Buf an (kein Reset — herbivore Events bleiben erhalten).
// Wird nach RunPhase1 aus dem Simulations-Koordinator aufgerufen (sequentiell, deterministisch).
// predReproThreshold, predReproReserve und predMaxSight kommen aus config.PredatorConfig.
func (p *Partition) RunPredatorPhase1(ctx world.WorldContext, predReproThreshold, predReproReserve float32, predMaxSight int32) {
	for i := int32(0); i < int32(p.Len); i++ {
		if !p.Alive[i] || p.EntityType[i] != entity.Predator {
			continue
		}
		s := predator.State{
			Idx:            i,
			X:              p.X[i],
			Y:              p.Y[i],
			Energy:         p.Energy[i],
			Genes:          p.Genes[i],
			ReproThreshold: predReproThreshold,
			ReproReserve:   predReproReserve,
			MaxSight:       predMaxSight,
		}
		predator.Tick(s, ctx, &p.Buf)
	}
}

// agent wraps einen SoA-Slot für die Tick-Logik (kein Alloc — Stack-allokiert).
type agent struct {
	idx int32
	p   *Partition
}

// tick implementiert die Verhaltenslogik eines Individuums für einen Tick.
// Logik basiert auf Genen:
//   - GeneSpeed: bestimmt Bewegungswahrscheinlichkeit und max. Schrittweite
//   - GeneSight: bestimmt Suchradius für Nahrung und Partner
//   - GeneEfficiency: bestimmt Energiegewinn beim Essen
//
// Events die geschrieben werden können: EventMove, EventEat, EventReproduce, EventDie
//
// Energie-Logik:
//   - Jeder Tick kostet Basisbetrag (abhängig von GeneSpeed)
//   - Essen: Nimmt Nahrung von Tile, gewinnt Energie (× GeneEfficiency)
//   - Reproduktion: Wenn Energy >= ReproductionThreshold, schreibt EventReproduce
//   - Tod: Wenn Energy <= 0 nach Kosten, schreibt EventDie
func (a agent) tick(ctx world.WorldContext, buf *entity.EventBuffer) {
	p := a.p
	i := a.idx

	pos := image.Pt(int(p.X[i]), int(p.Y[i]))
	energy := p.Energy[i]
	genes := p.Genes[i]
	rng := ctx.Rand()

	speedGene := genes[entity.GeneSpeed]
	sightGene := genes[entity.GeneSight]
	efficiencyGene := genes[entity.GeneEfficiency]

	// Energiekosten pro Tick (Basiskosten + Speed-Malus)
	baseCost := float32(0.5) + speedGene*0.1
	energy -= baseCost

	// Tod durch Energiemangel
	if energy <= 0 {
		buf.Append(entity.Event{Type: entity.EventDie, AgentIdx: i, TargetPos: pos})
		return
	}

	// Maximale Schrittweite aus Speed-Gen
	maxStep := max(1, int(speedGene+0.5))

	// Sichtradius aus Sight-Gen
	sightRadius := max(1, int(sightGene+0.5))

	// Nahrungssuche: beste nahegelegene Tile finden
	bestFood := float32(0)
	bestPos := pos
	for dy := -sightRadius; dy <= sightRadius; dy++ {
		for dx := -sightRadius; dx <= sightRadius; dx++ {
			np := image.Pt(pos.X+dx, pos.Y+dy)
			tile := ctx.TileAt(np)
			if tile.IsWalkable() && tile.Food > bestFood {
				bestFood = tile.Food
				bestPos = np
			}
		}
	}

	// Bewege Richtung beste Nahrung (oder zufällig wenn keine)
	var targetPos image.Point
	if bestFood > 0 {
		// Schritt in Richtung bestPos
		dx := clampStep(bestPos.X-pos.X, maxStep)
		dy := clampStep(bestPos.Y-pos.Y, maxStep)
		targetPos = image.Pt(pos.X+dx, pos.Y+dy)
	} else {
		// Zufällige Bewegung
		dx := rng.Intn(2*maxStep+1) - maxStep
		dy := rng.Intn(2*maxStep+1) - maxStep
		targetPos = image.Pt(pos.X+dx, pos.Y+dy)
	}

	// Bewegungs-Event
	targetTile := ctx.TileAt(targetPos)
	if targetTile.IsWalkable() && targetPos != pos {
		buf.Append(entity.Event{
			Type:      entity.EventMove,
			AgentIdx:  i,
			TargetPos: targetPos,
		})
	} else {
		targetPos = pos // bleibt stehen
	}

	// Essen: wenn Nahrung auf Zielpos
	eatTile := ctx.TileAt(targetPos)
	if eatTile.Food > 0 {
		eaten := eatTile.Food * 0.5 // nimmt 50% der verfügbaren Nahrung
		if eaten > 2.0 {
			eaten = 2.0
		}
		gain := eaten * efficiencyGene
		energy += gain
		buf.Append(entity.Event{
			Type:      entity.EventEat,
			AgentIdx:  i,
			TargetPos: targetPos,
			Value:     eaten,
		})
	}

	// Reproduktion
	if energy >= ctx.ReproductionThreshold() {
		buf.Append(entity.Event{
			Type:      entity.EventReproduce,
			AgentIdx:  i,
			TargetPos: targetPos,
		})
	}
}

// clampStep begrenzt einen Delta-Wert auf [-maxStep, maxStep].
func clampStep(delta, maxStep int) int {
	if delta > maxStep {
		return maxStep
	}
	if delta < -maxStep {
		return -maxStep
	}
	return delta
}
