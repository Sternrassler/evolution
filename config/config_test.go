package config

import (
	"testing"

	"pgregory.net/rapid"

	"github.com/Sternrassler/evolution/sim/entity"
)

func TestDefaultConfigValid(t *testing.T) {
	cfg := DefaultConfig()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("DefaultConfig().Validate() = %v, want nil", err)
	}
}

func TestGhostK(t *testing.T) {
	// MaxSpeedRange > MaxSightRange → gibt MaxSpeedRange zurück
	cfg := DefaultConfig()
	cfg.MaxSpeedRange = 8
	cfg.MaxSightRange = 5
	if got := cfg.GhostK(); got != 8 {
		t.Errorf("GhostK() = %d, want 8 (MaxSpeedRange dominiert)", got)
	}

	// MaxSightRange > MaxSpeedRange → gibt MaxSightRange zurück
	cfg.MaxSpeedRange = 3
	cfg.MaxSightRange = 10
	if got := cfg.GhostK(); got != 10 {
		t.Errorf("GhostK() = %d, want 10 (MaxSightRange dominiert)", got)
	}

	// Gleichstand → gibt den gemeinsamen Wert zurück
	cfg.MaxSpeedRange = 7
	cfg.MaxSightRange = 7
	if got := cfg.GhostK(); got != 7 {
		t.Errorf("GhostK() = %d, want 7 (Gleichstand)", got)
	}
}

func TestValidate_ZeroPopulation(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MaxPopulation = 0
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() = nil, want error für MaxPopulation=0")
	}
}

func TestValidate_InitialPopExceedsMax(t *testing.T) {
	cfg := DefaultConfig()
	cfg.InitialPop = cfg.MaxPopulation + 1
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() = nil, want error für InitialPop > MaxPopulation")
	}
}

func TestValidate_TooManyPartitions(t *testing.T) {
	cfg := DefaultConfig()
	// Mit WorldHeight=200, K=10 → 2*K=20 → minHeight=WorldHeight/NumPartitions muss >= 20
	// NumPartitions=11 → 200/11=18 < 20 → Fehler
	cfg.WorldHeight = 200
	cfg.MaxSpeedRange = 5
	cfg.MaxSightRange = 10
	cfg.NumPartitions = 11
	if err := cfg.Validate(); err == nil {
		t.Fatalf("Validate() = nil, want error für zu viele Partitionen (NumPartitions=%d, WorldHeight=%d, K=%d)",
			cfg.NumPartitions, cfg.WorldHeight, cfg.GhostK())
	}
}

func TestValidate_InvalidGeneCount(t *testing.T) {
	cfg := DefaultConfig()
	// Eine GeneDef entfernen → falsche Anzahl
	cfg.GeneDefinitions = cfg.GeneDefinitions[:entity.NumGenes-1]
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() = nil, want error für falsche Anzahl GeneDefinitions")
	}
}

func TestValidate_InvalidGeneRange(t *testing.T) {
	cfg := DefaultConfig()
	// Min >= Max
	cfg.GeneDefinitions[0].Min = 5.0
	cfg.GeneDefinitions[0].Max = 5.0
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() = nil, want error für Min >= Max")
	}

	cfg2 := DefaultConfig()
	cfg2.GeneDefinitions[0].Min = 6.0
	cfg2.GeneDefinitions[0].Max = 3.0
	if err := cfg2.Validate(); err == nil {
		t.Fatal("Validate() = nil, want error für Min > Max")
	}
}

func TestValidate_InvalidMutationRate(t *testing.T) {
	cfg := DefaultConfig()
	cfg.GeneDefinitions[0].MutationRate = 1.5
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() = nil, want error für MutationRate > 1.0")
	}

	cfg2 := DefaultConfig()
	cfg2.GeneDefinitions[0].MutationRate = -0.1
	if err := cfg2.Validate(); err == nil {
		t.Fatal("Validate() = nil, want error für MutationRate < 0")
	}
}

// TestValidate_ValidConfigs ist ein Property-Test: zufällig erzeugte, strukturell valide
// Configs sollen Validate() == nil liefern.
func TestValidate_ValidConfigs(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		// Welt-Dimensionen: zwischen 100 und 500
		worldWidth := rapid.IntRange(100, 500).Draw(rt, "worldWidth")
		worldHeight := rapid.IntRange(100, 500).Draw(rt, "worldHeight")

		// Ghost-K-Parameter: 1..10
		maxSpeed := rapid.IntRange(1, 10).Draw(rt, "maxSpeed")
		maxSight := rapid.IntRange(1, 10).Draw(rt, "maxSight")
		k := max(maxSpeed, maxSight)

		// NumPartitions so wählen, dass WorldHeight/NumPartitions >= 2*K
		maxPartitions := worldHeight / (2 * k)
		if maxPartitions < 1 {
			maxPartitions = 1
			// In diesem Fall muss worldHeight >= 2*k gelten — überspringen wenn nicht erfüllt
			if worldHeight < 2*k {
				t.Skip("worldHeight zu klein für k, überspringen")
			}
		}
		numPartitions := rapid.IntRange(1, maxPartitions).Draw(rt, "numPartitions")

		maxPop := rapid.IntRange(1, 100000).Draw(rt, "maxPop")
		initialPop := rapid.IntRange(0, maxPop).Draw(rt, "initialPop")

		spatialCellSize := rapid.IntRange(1, 20).Draw(rt, "spatialCellSize")
		tps := rapid.IntRange(1, 120).Draw(rt, "tps")

		baseEnergy := rapid.Float32Range(0.01, 10.0).Draw(rt, "baseEnergy")
		repThreshold := rapid.Float32Range(1.0, 1000.0).Draw(rt, "repThreshold")
		repReserve := rapid.Float32Range(0.1, repThreshold-0.1).Draw(rt, "repReserve")

		// GeneDefs: NumGenes viele, jeweils valide
		geneDefs := make([]GeneDef, entity.NumGenes)
		for i := range geneDefs {
			minVal := rapid.Float32Range(0.1, 4.9).Draw(rt, "geneMin")
			maxVal := rapid.Float32Range(minVal+0.1, 10.0).Draw(rt, "geneMax")
			mutRate := rapid.Float32Range(0.0, 1.0).Draw(rt, "mutRate")
			mutStep := rapid.Float32Range(0.01, 1.0).Draw(rt, "mutStep")
			geneDefs[i] = GeneDef{
				Key:          entity.GeneKey(i),
				Min:          minVal,
				Max:          maxVal,
				MutationRate: mutRate,
				MutationStep: mutStep,
			}
		}

		cfg := Config{
			WorldWidth:            worldWidth,
			WorldHeight:           worldHeight,
			NumPartitions:         numPartitions,
			MaxPopulation:         maxPop,
			InitialPop:            initialPop,
			TicksPerSecond:        tps,
			DebugIntegrity:        false,
			SpatialCellSize:       spatialCellSize,
			MaxSpeedRange:         maxSpeed,
			MaxSightRange:         maxSight,
			BaseEnergyCost:        baseEnergy,
			ReproductionThreshold: repThreshold,
			ReproductionReserve:   repReserve,
			GeneDefinitions:       geneDefs,
		}

		if err := cfg.Validate(); err != nil {
			rt.Fatalf("Validate() = %v für valide Config: %+v", err, cfg)
		}
	})
}
