package predator

import (
	"image"

	"github.com/Sternrassler/evolution/sim/entity"
	"github.com/Sternrassler/evolution/sim/world"
)

// State hält alle Felder, die der Predator-Tick benötigt (SoA-kompatibel).
// Value-Type: stack-allokiert, 0 allocs im Hot-Path (ADR-011).
//
// ReproThreshold und ReproReserve kommen aus config.PredatorConfig — kein
// config-Import nötig, da die Partition diese Werte beim State-Aufbau füllt.
type State struct {
	Idx            int32
	X, Y           int32
	Energy         float32
	Genes          [entity.NumGenes]float32
	ReproThreshold float32 // aus config.PredatorConfig.ReproThreshold
	ReproReserve   float32 // aus config.PredatorConfig.ReproReserve
	MaxSight       int32   // aus config.PredatorConfig.MaxSight (Räuber-spezifisch, > Herbivore-Sicht)
}

// Tick führt den Predator-Schritt aus: Jagd, Random Walk, Reproduktion.
// Kein Zeiger-Receiver, kein Interface → stack-allokiert, 0 allocs (ADR-011).
//
// Phase-1-Kontrakt (ADR-005): nur lesen (ctx, s) + Events schreiben (out).
// Keine Weltmutation.
//
// Events die geschrieben werden können:
//
//	EventDie       — Energie ≤ 0 nach Basiskosten
//	EventAttack    — Beute in Sichtweite; Value = float32(target SoA-Index) für Phase-2-Auflösung
//	EventMove      — kein Ziel in Sichtweite → Random Walk
//	EventReproduce — Energie ≥ ReproThreshold
func Tick(s State, ctx world.WorldContext, out *entity.EventBuffer) {
	pos := image.Pt(int(s.X), int(s.Y))
	energy := s.Energy
	rng := ctx.Rand()

	speedGene := s.Genes[entity.GeneSpeed]
	aggressionGene := s.Genes[entity.GeneAggression]

	// Energiekosten pro Tick (Räuber teurer als Herbivore: höherer Körperaufwand)
	// Herbivore: 0.5 + speed×0.1 — Räuber: 0.8 + speed×0.15
	baseCost := float32(0.8) + speedGene*0.15
	energy -= baseCost

	// Tod durch Energiemangel
	if energy <= 0 {
		out.Append(entity.Event{Type: entity.EventDie, AgentIdx: s.Idx, TargetPos: pos})
		return
	}

	// Jagdradius: GeneAggression skaliert den Sichtradius [1, Predator.MaxSight]
	// Predatoren haben größere Sichtweite als Herbivore (s.MaxSight > GlobalMaxSight).
	sightRadius := max(1, int(aggressionGene*float32(s.MaxSight)+0.5))

	// Beute suchen — IndividualsNear gibt SoA-Indizes zurück (zero-alloc via ctx-internen Buffer)
	nearby := ctx.IndividualsNear(pos, sightRadius)

	// Jagdversuch: GeneAggression bestimmt Erfolgswahrscheinlichkeit.
	// Hohe Aggression → häufigere Kills; niedrige Aggression → Wanderer, der verhungert.
	// Evolutionsdruck: nur Räuber mit ausreichend hoher Aggression überleben.
	// Bei Misserfolg oder keiner Beute in Sichtweite: Random Walk.
	if len(nearby) > 0 && rng.Float64() < float64(aggressionGene) {
		// Jagd erfolgreich: Angriff auf zufällige Beute in der Nähe.
		// Value = float32(target SoA-Index): Phase 2 löst Energie-Transfer auf (Issue #7)
		targetIdx := nearby[rng.Intn(len(nearby))]
		out.Append(entity.Event{
			Type:      entity.EventAttack,
			AgentIdx:  s.Idx,
			TargetPos: pos,
			Value:     float32(targetIdx),
		})
	} else {
		// Keine Beute in Sichtweite oder Jagd gescheitert → Random Walk
		maxStep := max(1, int(speedGene+0.5))
		dx := rng.Intn(2*maxStep+1) - maxStep
		dy := rng.Intn(2*maxStep+1) - maxStep
		targetPos := image.Pt(pos.X+dx, pos.Y+dy)
		if ctx.TileAt(targetPos).IsWalkable() && targetPos != pos {
			out.Append(entity.Event{Type: entity.EventMove, AgentIdx: s.Idx, TargetPos: targetPos})
		}
	}

	// Reproduktion wenn Energie die Schwelle überschreitet
	if energy >= s.ReproThreshold {
		out.Append(entity.Event{Type: entity.EventReproduce, AgentIdx: s.Idx, TargetPos: pos})
	}
}
