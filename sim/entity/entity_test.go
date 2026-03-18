package entity

import (
	"image"
	"testing"

	"pgregory.net/rapid"
)

func TestGeneKeyConstants(t *testing.T) {
	if NumGenes != 4 {
		t.Errorf("NumGenes = %d, want 4", NumGenes)
	}
	if GeneSpeed != 0 {
		t.Errorf("GeneSpeed = %d, want 0", GeneSpeed)
	}
	if GeneSight != 1 {
		t.Errorf("GeneSight = %d, want 1", GeneSight)
	}
	if GeneEfficiency != 2 {
		t.Errorf("GeneEfficiency = %d, want 2", GeneEfficiency)
	}
	if GeneAggression != 3 {
		t.Errorf("GeneAggression = %d, want 3", GeneAggression)
	}
}

func TestNewIndividual(t *testing.T) {
	genes := [NumGenes]float32{0.5, 1.0, 0.8}
	pos := image.Point{X: 3, Y: 7}
	ind := NewIndividual(42, pos, genes, 100.0)

	if ind.ID != 42 {
		t.Errorf("ID = %d, want 42", ind.ID)
	}
	if ind.Pos != pos {
		t.Errorf("Pos = %v, want %v", ind.Pos, pos)
	}
	if ind.Energy != 100.0 {
		t.Errorf("Energy = %f, want 100.0", ind.Energy)
	}
	if ind.Genes != genes {
		t.Errorf("Genes = %v, want %v", ind.Genes, genes)
	}
	if !ind.IsAlive() {
		t.Error("newly created Individual should be alive")
	}
}

func TestIndividualLiveness(t *testing.T) {
	genes := [NumGenes]float32{}
	ind := NewIndividual(1, image.Point{}, genes, 50.0)

	if !ind.IsAlive() {
		t.Error("Individual should be alive before Kill()")
	}
	ind.Kill()
	if ind.IsAlive() {
		t.Error("Individual should not be alive after Kill()")
	}
}

func TestEventBufferAppendReset(t *testing.T) {
	buf := NewEventBuffer(8)

	if buf.Len() != 0 {
		t.Errorf("Len after creation = %d, want 0", buf.Len())
	}

	e := Event{Type: EventMove, AgentIdx: 0, TargetPos: image.Point{X: 1, Y: 2}, Value: 0.5}
	buf.Append(e)
	buf.Append(e)

	if buf.Len() != 2 {
		t.Errorf("Len after 2 Appends = %d, want 2", buf.Len())
	}

	buf.Reset()

	if buf.Len() != 0 {
		t.Errorf("Len after Reset = %d, want 0", buf.Len())
	}
}

func TestEventBufferZeroAlloc(t *testing.T) {
	buf := NewEventBuffer(100)
	e := Event{Type: EventEat, AgentIdx: 1, TargetPos: image.Point{X: 5, Y: 5}, Value: 10.0}

	allocs := testing.AllocsPerRun(100, func() {
		buf.Append(e)
		buf.Reset()
	})

	if allocs != 0 {
		t.Errorf("AllocsPerRun = %f, want 0 (zero-alloc)", allocs)
	}
}

func TestEventBufferEvents(t *testing.T) {
	buf := NewEventBuffer(4)

	e1 := Event{Type: EventMove, AgentIdx: 0, TargetPos: image.Point{X: 1, Y: 0}, Value: 0.0}
	e2 := Event{Type: EventDie, AgentIdx: 1, TargetPos: image.Point{X: 2, Y: 3}, Value: -1.0}

	buf.Append(e1)
	buf.Append(e2)

	events := buf.Events()
	if len(events) != 2 {
		t.Fatalf("len(Events()) = %d, want 2", len(events))
	}
	if events[0] != e1 {
		t.Errorf("events[0] = %v, want %v", events[0], e1)
	}
	if events[1] != e2 {
		t.Errorf("events[1] = %v, want %v", events[1], e2)
	}
}

func TestEventBufferProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		capacity := rapid.IntRange(1, 64).Draw(t, "capacity")
		buf := NewEventBuffer(capacity)

		ops := rapid.SliceOf(rapid.Bool()).Draw(t, "ops") // true=Append, false=Reset

		expected := 0
		for _, isAppend := range ops {
			if isAppend {
				e := Event{
					Type:      EventType(rapid.IntRange(0, 3).Draw(t, "eventType")),
					AgentIdx:  int32(rapid.Int32Range(0, 100).Draw(t, "agentIdx")),
					TargetPos: image.Point{X: rapid.IntRange(0, 99).Draw(t, "x"), Y: rapid.IntRange(0, 99).Draw(t, "y")},
					Value:     float32(rapid.Float32().Draw(t, "value")),
				}
				buf.Append(e)
				expected++
			} else {
				buf.Reset()
				expected = 0
			}

			if buf.Len() != expected {
				t.Fatalf("Len() = %d, want %d", buf.Len(), expected)
			}
		}

		// After reset, always 0
		buf.Reset()
		if buf.Len() != 0 {
			t.Fatalf("Len() after final Reset = %d, want 0", buf.Len())
		}
	})
}
