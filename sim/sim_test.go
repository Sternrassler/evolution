package sim

import (
	mrand "math/rand"
	"testing"

	"pgregory.net/rapid"

	"github.com/Sternrassler/evolution/config"
	"github.com/Sternrassler/evolution/sim/entity"
)

// testRng ist ein deterministischer RNG für Tests.
type testRng struct{ r *mrand.Rand }

func (t *testRng) Float64() float64 { return t.r.Float64() }
func (t *testRng) Intn(n int) int   { return t.r.Intn(n) }

func newTestRng(seed int64) *testRng {
	return &testRng{r: mrand.New(mrand.NewSource(seed))} //nolint:gosec
}

func testConfig() config.Config {
	cfg := config.DefaultConfig()
	cfg.WorldWidth = 50
	cfg.WorldHeight = 50
	cfg.InitialPop = 50
	cfg.MaxPopulation = 500
	cfg.NumPartitions = 2
	return cfg
}

// TestNew_Valid prüft dass New() mit valider Config keinen Fehler liefert.
func TestNew_Valid(t *testing.T) {
	cfg := testConfig()
	rng := newTestRng(42)
	sim, exp, err := New(cfg, rng, nil)
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	if sim == nil {
		t.Fatal("New() returned nil simulation")
	}
	if exp == nil {
		t.Fatal("New() returned nil exporter")
	}
}

// TestStep_PopulationNotZero prüft dass nach 10 Ticks noch Individuen leben.
func TestStep_PopulationNotZero(t *testing.T) {
	cfg := testConfig()
	rng := newTestRng(42)
	sim, _, err := New(cfg, rng, nil)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	for range 10 {
		sim.Step()
	}
	pop := sim.totalPopulation()
	if pop == 0 {
		t.Error("population is zero after 10 ticks")
	}
}

// TestDeterminism prüft CI Gate 3: Gleicher Seed → identischer Hash nach 50 Ticks.
func TestDeterminism(t *testing.T) {
	cfg := testConfig()

	runSim := func(seed int64) uint64 {
		rng := newTestRng(seed)
		sim, exp, err := New(cfg, rng, nil)
		if err != nil {
			t.Fatalf("New() error: %v", err)
		}
		for range 50 {
			sim.Step()
		}
		snap := exp.Load()
		return snap.Hash()
	}

	hash1 := runSim(12345)
	hash2 := runSim(12345)

	if hash1 != hash2 {
		t.Errorf("determinism violation: hash1=%d hash2=%d", hash1, hash2)
	}

	// Verschiedene Seeds sollen unterschiedliche Hashes erzeugen (Sanity-Check)
	hash3 := runSim(99999)
	if hash1 == hash3 {
		t.Log("warning: different seeds produced same hash (unlikely but possible)")
	}
}

// TestSnapshotExporter_LockFree prüft dass Load() nach store() einen validen Snapshot liefert.
func TestSnapshotExporter_LockFree(t *testing.T) {
	exp := NewSnapshotExporter(100, 50)

	snap := WorldSnapshot{
		Tick: 42,
		Stats: TickStats{
			Population: 10,
			Births:     2,
			Deaths:     1,
		},
	}
	exp.store(snap)

	loaded := exp.Load()
	if loaded == nil {
		t.Fatal("Load() returned nil")
	}
	if loaded.Tick != 42 {
		t.Errorf("expected Tick=42, got %d", loaded.Tick)
	}
	if loaded.Stats.Population != 10 {
		t.Errorf("expected Population=10, got %d", loaded.Stats.Population)
	}
}

// TestMutateBounds prüft dass mutierte Gene stets in [Min, Max] liegen.
func TestMutateBounds(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		seed := rapid.Int64().Draw(rt, "seed")
		rng := newTestRng(seed)
		cfg := testConfig()

		var parent [entity.NumGenes]float32
		for i, def := range cfg.GeneDefinitions {
			if i >= entity.NumGenes {
				break
			}
			parent[i] = def.Min + float32(rng.Float64())*(def.Max-def.Min)
		}

		// Mehrfach mutieren
		child := parent
		for range 100 {
			child = mutateGenes(child, cfg.GeneDefinitions, rng)
		}

		for i, def := range cfg.GeneDefinitions {
			if i >= entity.NumGenes {
				break
			}
			if child[i] < def.Min || child[i] > def.Max {
				rt.Errorf("Gene[%d]=%f out of bounds [%f, %f]", i, child[i], def.Min, def.Max)
			}
		}
	})
}

// TestPopulationCap prüft dass die Population MaxPopulation nie überschreitet.
func TestPopulationCap(t *testing.T) {
	cfg := testConfig()
	rng := newTestRng(42)
	sim, _, err := New(cfg, rng, nil)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	for tick := range 100 {
		sim.Step()
		pop := sim.totalPopulation()
		if pop > cfg.MaxPopulation {
			t.Errorf("tick %d: population %d exceeds MaxPopulation %d", tick, pop, cfg.MaxPopulation)
		}
	}
}
