package partition_test

import (
	"image"
	"testing"

	"pgregory.net/rapid"

	"github.com/Sternrassler/evolution/sim/entity"
	"github.com/Sternrassler/evolution/sim/partition"
	"github.com/Sternrassler/evolution/testworld"
)

// makeIndividual ist ein Hilfshelfer für Tests.
func makeIndividual(id uint64, x, y int, energy float32) entity.Individual {
	genes := [entity.NumGenes]float32{1.0, 2.0, 1.0}
	return entity.NewIndividual(id, image.Pt(x, y), genes, energy)
}

// TestNewPartition prüft die korrekte Initialisierung der Arrays.
func TestNewPartition(t *testing.T) {
	p := partition.NewPartition(100, 0, 20)

	if cap(p.X) != 100 {
		t.Errorf("cap(X): want 100, got %d", cap(p.X))
	}
	if cap(p.Y) != 100 {
		t.Errorf("cap(Y): want 100, got %d", cap(p.Y))
	}
	if cap(p.Energy) != 100 {
		t.Errorf("cap(Energy): want 100, got %d", cap(p.Energy))
	}
	if cap(p.Age) != 100 {
		t.Errorf("cap(Age): want 100, got %d", cap(p.Age))
	}
	if cap(p.Alive) != 100 {
		t.Errorf("cap(Alive): want 100, got %d", cap(p.Alive))
	}
	if cap(p.Genes) != 100 {
		t.Errorf("cap(Genes): want 100, got %d", cap(p.Genes))
	}
	if cap(p.IDs) != 100 {
		t.Errorf("cap(IDs): want 100, got %d", cap(p.IDs))
	}
	if p.StartRow != 0 {
		t.Errorf("StartRow: want 0, got %d", p.StartRow)
	}
	if p.EndRow != 20 {
		t.Errorf("EndRow: want 20, got %d", p.EndRow)
	}
	if p.Len != 0 {
		t.Errorf("Len: want 0, got %d", p.Len)
	}
}

// TestAddIndividual_Basic prüft das erste hinzugefügte Individuum.
func TestAddIndividual_Basic(t *testing.T) {
	p := partition.NewPartition(100, 0, 20)
	ind := makeIndividual(42, 5, 10, 50.0)

	idx := p.AddIndividual(ind)

	if idx != 0 {
		t.Errorf("first index: want 0, got %d", idx)
	}
	if p.Len != 1 {
		t.Errorf("Len: want 1, got %d", p.Len)
	}
	if p.X[0] != 5 {
		t.Errorf("X[0]: want 5, got %d", p.X[0])
	}
	if p.Y[0] != 10 {
		t.Errorf("Y[0]: want 10, got %d", p.Y[0])
	}
	if p.Energy[0] != 50.0 {
		t.Errorf("Energy[0]: want 50.0, got %f", p.Energy[0])
	}
	if !p.Alive[0] {
		t.Error("Alive[0]: want true, got false")
	}
	if p.IDs[0] != 42 {
		t.Errorf("IDs[0]: want 42, got %d", p.IDs[0])
	}
}

// TestAddIndividual_FreeListReuse prüft dass tote Slots wiederverwendet werden.
func TestAddIndividual_FreeListReuse(t *testing.T) {
	p := partition.NewPartition(100, 0, 20)

	// Zwei Individuen hinzufügen
	idx0 := p.AddIndividual(makeIndividual(1, 1, 1, 10.0))
	idx1 := p.AddIndividual(makeIndividual(2, 2, 2, 20.0))

	if idx0 != 0 || idx1 != 1 {
		t.Fatalf("unexpected initial indices: %d, %d", idx0, idx1)
	}
	if p.Len != 2 {
		t.Fatalf("Len before dead: want 2, got %d", p.Len)
	}

	// Erstes töten
	p.MarkDead(idx0)

	if p.Len != 2 {
		t.Errorf("Len after MarkDead: want 2 (unchanged), got %d", p.Len)
	}

	// Neues Individuum hinzufügen — soll Slot 0 wiederverwenden
	idx2 := p.AddIndividual(makeIndividual(3, 3, 3, 30.0))

	if idx2 != 0 {
		t.Errorf("reused index: want 0, got %d", idx2)
	}
	if p.Len != 2 {
		t.Errorf("Len after reuse: want 2 (unchanged), got %d", p.Len)
	}
	if p.IDs[0] != 3 {
		t.Errorf("IDs[0] after reuse: want 3, got %d", p.IDs[0])
	}
	if !p.Alive[0] {
		t.Error("Alive[0] after reuse: want true, got false")
	}
	// idx1 bleibt unberührt
	if p.IDs[1] != 2 {
		t.Errorf("IDs[1]: want 2 (unchanged), got %d", p.IDs[1])
	}
}

// TestMarkDead prüft dass MarkDead korrekt Alive und FreeList setzt.
func TestMarkDead(t *testing.T) {
	p := partition.NewPartition(100, 0, 20)
	p.AddIndividual(makeIndividual(1, 1, 1, 10.0))
	p.AddIndividual(makeIndividual(2, 2, 2, 20.0))

	p.MarkDead(0)

	if p.Alive[0] {
		t.Error("Alive[0]: want false after MarkDead, got true")
	}
	if p.Alive[1] != true {
		t.Error("Alive[1]: should be unaffected")
	}
	if len(p.FreeList) != 1 || p.FreeList[0] != 0 {
		t.Errorf("FreeList: want [0], got %v", p.FreeList)
	}
}

// TestLiveCount prüft die korrekte Anzahl lebender Individuen.
func TestLiveCount(t *testing.T) {
	p := partition.NewPartition(100, 0, 20)

	if p.LiveCount() != 0 {
		t.Errorf("empty: want 0, got %d", p.LiveCount())
	}

	p.AddIndividual(makeIndividual(1, 1, 1, 10.0))
	p.AddIndividual(makeIndividual(2, 2, 2, 20.0))
	p.AddIndividual(makeIndividual(3, 3, 3, 30.0))

	if p.LiveCount() != 3 {
		t.Errorf("3 added: want 3, got %d", p.LiveCount())
	}

	p.MarkDead(1)

	if p.LiveCount() != 2 {
		t.Errorf("1 dead: want 2, got %d", p.LiveCount())
	}

	p.MarkDead(0)
	p.MarkDead(2)

	if p.LiveCount() != 0 {
		t.Errorf("all dead: want 0, got %d", p.LiveCount())
	}
}

// TestToIndividuals prüft SoA→AoS-Konvertierung.
func TestToIndividuals(t *testing.T) {
	p := partition.NewPartition(100, 0, 20)

	ind0 := makeIndividual(10, 5, 6, 50.0)
	ind1 := makeIndividual(20, 7, 8, 60.0)
	ind2 := makeIndividual(30, 9, 10, 70.0)

	p.AddIndividual(ind0)
	p.AddIndividual(ind1)
	p.AddIndividual(ind2)

	// ind1 töten
	p.MarkDead(1)

	result := p.ToIndividuals()

	if len(result) != 2 {
		t.Fatalf("ToIndividuals: want 2 individuals, got %d", len(result))
	}

	// IDs der lebenden prüfen
	ids := map[uint64]bool{}
	for _, ind := range result {
		ids[ind.ID] = true
		if !ind.IsAlive() {
			t.Errorf("ToIndividuals: individual %d should be alive", ind.ID)
		}
	}

	if !ids[10] {
		t.Error("ToIndividuals: missing individual with ID 10")
	}
	if !ids[30] {
		t.Error("ToIndividuals: missing individual with ID 30")
	}
	if ids[20] {
		t.Error("ToIndividuals: dead individual with ID 20 should not appear")
	}

	// Position und Energie für ind0 prüfen
	for _, ind := range result {
		if ind.ID == 10 {
			if ind.Pos != image.Pt(5, 6) {
				t.Errorf("ind0.Pos: want (5,6), got %v", ind.Pos)
			}
			if ind.Energy != 50.0 {
				t.Errorf("ind0.Energy: want 50.0, got %f", ind.Energy)
			}
		}
	}
}

// TestRunPhase1_NoMutation prüft dass RunPhase1 die SoA-Arrays nicht mutiert.
func TestRunPhase1_NoMutation(t *testing.T) {
	ctx := testworld.New(20, 20).Build()
	p := partition.NewPartition(100, 0, 20)

	// 5 Individuen mit genug Energie (damit sie nicht sofort sterben)
	for i := range 5 {
		genes := [entity.NumGenes]float32{1.0, 2.0, 1.0}
		ind := entity.NewIndividual(uint64(i+1), image.Pt(5+i, 5), genes, 80.0)
		p.AddIndividual(ind)
	}

	// Snapshot VOR Phase 1
	snapX := make([]int32, p.Len)
	snapY := make([]int32, p.Len)
	snapEnergy := make([]float32, p.Len)
	snapAlive := make([]bool, p.Len)
	snapGenes := make([][entity.NumGenes]float32, p.Len)
	copy(snapX, p.X[:p.Len])
	copy(snapY, p.Y[:p.Len])
	copy(snapEnergy, p.Energy[:p.Len])
	copy(snapAlive, p.Alive[:p.Len])
	copy(snapGenes, p.Genes[:p.Len])

	p.RunPhase1(ctx)

	// Snapshot NACH Phase 1 — muss identisch sein
	for i := range p.Len {
		if p.X[i] != snapX[i] {
			t.Errorf("X[%d] mutated: was %d, now %d", i, snapX[i], p.X[i])
		}
		if p.Y[i] != snapY[i] {
			t.Errorf("Y[%d] mutated: was %d, now %d", i, snapY[i], p.Y[i])
		}
		if p.Energy[i] != snapEnergy[i] {
			t.Errorf("Energy[%d] mutated: was %f, now %f", i, snapEnergy[i], p.Energy[i])
		}
		if p.Alive[i] != snapAlive[i] {
			t.Errorf("Alive[%d] mutated: was %v, now %v", i, snapAlive[i], p.Alive[i])
		}
		if p.Genes[i] != snapGenes[i] {
			t.Errorf("Genes[%d] mutated", i)
		}
	}

	// Buf.Len() darf sich geändert haben (Events wurden geschrieben)
	// Mindestens 1 Event wird erwartet (Move oder Eat)
	if p.Buf.Len() == 0 {
		t.Error("RunPhase1: expected at least 1 event in buffer, got 0")
	}
}

// BenchmarkRunPhase1 misst Allokationen im Hot-Path.
func BenchmarkRunPhase1(b *testing.B) {
	ctx := testworld.New(50, 50).Build()
	p := partition.NewPartition(200, 0, 50)

	for i := range 100 {
		genes := [entity.NumGenes]float32{1.5, 3.0, 1.2}
		ind := entity.NewIndividual(uint64(i+1), image.Pt((i%50)+1, (i/50)+1), genes, 80.0)
		p.AddIndividual(ind)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		p.RunPhase1(ctx)
	}
}

// TestAddRemoveProperty prüft via Property-Test dass LiveCount immer korrekt ist.
func TestAddRemoveProperty(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		maxPop := rapid.IntRange(10, 200).Draw(rt, "maxPop")
		p := partition.NewPartition(maxPop, 0, 20)

		numOps := rapid.IntRange(1, maxPop).Draw(rt, "numOps")
		expectedLive := 0

		// Phase 1: nur hinzufügen
		for i := range numOps {
			genes := [entity.NumGenes]float32{1.0, 2.0, 1.0}
			ind := entity.NewIndividual(uint64(i+1), image.Pt(i%20, i/20), genes, 50.0)
			p.AddIndividual(ind)
			expectedLive++
		}

		if p.LiveCount() != expectedLive {
			rt.Fatalf("after adds: LiveCount=%d, want %d", p.LiveCount(), expectedLive)
		}

		// Phase 2: zufällig töten
		numToKill := rapid.IntRange(0, numOps).Draw(rt, "numToKill")
		killed := map[int32]bool{}
		for range numToKill {
			idx := int32(rapid.IntRange(0, numOps-1).Draw(rt, "killIdx"))
			if !killed[idx] && p.Alive[idx] {
				p.MarkDead(idx)
				killed[idx] = true
				expectedLive--
			}
		}

		if p.LiveCount() != expectedLive {
			rt.Fatalf("after kills: LiveCount=%d, want %d", p.LiveCount(), expectedLive)
		}

		// Phase 3: FreeList-Reuse — neue Individuen hinzufügen
		numToAdd := rapid.IntRange(0, len(killed)).Draw(rt, "numToAdd")
		for i := range numToAdd {
			genes := [entity.NumGenes]float32{1.0, 2.0, 1.0}
			ind := entity.NewIndividual(uint64(numOps+i+1), image.Pt(i%20, i/20), genes, 50.0)
			p.AddIndividual(ind)
			expectedLive++
		}

		if p.LiveCount() != expectedLive {
			rt.Fatalf("after reuse: LiveCount=%d, want %d", p.LiveCount(), expectedLive)
		}
	})
}
