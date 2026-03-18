package predator_test

import (
	"image"
	"testing"

	"github.com/Sternrassler/evolution/sim/entity"
	"github.com/Sternrassler/evolution/sim/predator"
	"github.com/Sternrassler/evolution/testworld"
)

// defaultState liefert einen State mit sinnvollen Standardwerten für Tests.
func defaultState(idx int32, x, y int32, energy float32) predator.State {
	return predator.State{
		Idx:    idx,
		X:      x,
		Y:      y,
		Energy: energy,
		Genes: [entity.NumGenes]float32{
			entity.GeneSpeed:      1.0,
			entity.GeneSight:      5.0,
			entity.GeneEfficiency: 1.0,
			entity.GeneAggression: 0.5,
		},
		ReproThreshold: 120.0,
		ReproReserve:   60.0,
	}
}

func TestTick_DiesWhenEnergyLow(t *testing.T) {
	ctx := testworld.New(10, 10).Build()
	buf := entity.NewEventBuffer(8)

	// Energie so niedrig, dass baseCost (0.8 + 1.0×0.15 = 0.95) sie auf ≤ 0 bringt
	s := defaultState(0, 5, 5, 0.5)
	predator.Tick(s, ctx, &buf)

	if buf.Len() != 1 {
		t.Fatalf("Len() = %d, want 1 (nur EventDie)", buf.Len())
	}
	if buf.Events()[0].Type != entity.EventDie {
		t.Errorf("Events()[0].Type = %d, want EventDie (%d)", buf.Events()[0].Type, entity.EventDie)
	}
}

func TestTick_AttacksWhenNearby(t *testing.T) {
	// Individuum in Sichtweite platzieren
	genes := [entity.NumGenes]float32{}
	target := entity.NewIndividual(99, image.Pt(6, 5), genes, 50.0)

	ctx := testworld.New(20, 20).
		WithIndividual(target).
		Build()

	buf := entity.NewEventBuffer(8)
	s := defaultState(0, 5, 5, 50.0)
	predator.Tick(s, ctx, &buf)

	// Mindestens ein Event muss EventAttack sein
	var found bool
	for _, e := range buf.Events() {
		if e.Type == entity.EventAttack {
			found = true
			if e.AgentIdx != 0 {
				t.Errorf("AgentIdx = %d, want 0 (Predator-Idx)", e.AgentIdx)
			}
			// Value enthält den SoA-Index des Ziels (0, da erstes Individuum)
			if int(e.Value) != 0 {
				t.Errorf("Value = %f, want 0 (target SoA-Index)", e.Value)
			}
		}
	}
	if !found {
		t.Error("kein EventAttack, obwohl Individuum in Sichtweite")
	}
}

func TestTick_RandomWalkWhenNoTarget(t *testing.T) {
	// Keine Individuen in der Welt → Random Walk
	ctx := testworld.New(20, 20).Build()
	buf := entity.NewEventBuffer(8)

	s := defaultState(0, 10, 10, 50.0)
	predator.Tick(s, ctx, &buf)

	for _, e := range buf.Events() {
		if e.Type == entity.EventAttack {
			t.Error("EventAttack ohne Beute in Sichtweite")
		}
	}
	// EventMove oder kein Event (wenn Zufallsschritt auf aktuelle Pos fällt — unwahrscheinlich)
	for _, e := range buf.Events() {
		if e.Type != entity.EventMove && e.Type != entity.EventReproduce {
			t.Errorf("unerwartetes Event: %d", e.Type)
		}
	}
}

func TestTick_ReproducesWhenHighEnergy(t *testing.T) {
	// Keine Beute → kein EventAttack; Energie >> ReproThreshold
	ctx := testworld.New(20, 20).Build()
	buf := entity.NewEventBuffer(8)

	s := defaultState(0, 10, 10, 200.0) // weit über ReproThreshold=120
	predator.Tick(s, ctx, &buf)

	var found bool
	for _, e := range buf.Events() {
		if e.Type == entity.EventReproduce {
			found = true
			if e.AgentIdx != 0 {
				t.Errorf("AgentIdx = %d, want 0", e.AgentIdx)
			}
		}
	}
	if !found {
		t.Error("kein EventReproduce bei Energy=200 >> ReproThreshold=120")
	}
}

func TestTick_NoReproduceBelowThreshold(t *testing.T) {
	ctx := testworld.New(20, 20).Build()
	buf := entity.NewEventBuffer(8)

	s := defaultState(0, 10, 10, 50.0) // unter ReproThreshold=120
	predator.Tick(s, ctx, &buf)

	for _, e := range buf.Events() {
		if e.Type == entity.EventReproduce {
			t.Error("EventReproduce bei Energy=50 < ReproThreshold=120")
		}
	}
}

func TestTick_ZeroAlloc(t *testing.T) {
	genes := [entity.NumGenes]float32{}
	target := entity.NewIndividual(1, image.Pt(6, 5), genes, 50.0)
	ctx := testworld.New(20, 20).WithIndividual(target).Build()
	buf := entity.NewEventBuffer(100)
	s := defaultState(0, 5, 5, 50.0)

	allocs := testing.AllocsPerRun(100, func() {
		buf.Reset()
		predator.Tick(s, ctx, &buf)
	})

	if allocs != 0 {
		t.Errorf("AllocsPerRun = %f, want 0 (zero-alloc Hot-Path)", allocs)
	}
}
