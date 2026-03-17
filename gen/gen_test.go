package gen_test

import (
	mrand "math/rand"
	"reflect"
	"testing"

	"pgregory.net/rapid"

	"github.com/Sternrassler/evolution/config"
	"github.com/Sternrassler/evolution/gen"
	"github.com/Sternrassler/evolution/sim/world"
)

// deterministicRng wraps math/rand.New für Tests — NUR in _test.go erlaubt.
type deterministicRng struct{ r *mrand.Rand }

func (d *deterministicRng) Float64() float64 { return d.r.Float64() }
func (d *deterministicRng) Intn(n int) int   { return d.r.Intn(n) }

func newRng(seed int64) *deterministicRng {
	return &deterministicRng{r: mrand.New(mrand.NewSource(seed))} //nolint:gosec
}

func smallCfg() config.Config {
	cfg := config.DefaultConfig()
	cfg.WorldWidth = 40
	cfg.WorldHeight = 40
	return cfg
}

// TestGenerateWorld_Dimensions: len(tiles) == cfg.WorldWidth * cfg.WorldHeight
func TestGenerateWorld_Dimensions(t *testing.T) {
	cfg := smallCfg()
	tiles := gen.GenerateWorld(cfg, newRng(42))
	want := cfg.WorldWidth * cfg.WorldHeight
	if len(tiles) != want {
		t.Errorf("len(tiles) = %d, want %d", len(tiles), want)
	}
}

// TestGenerateWorld_ValidBiomes: Alle tile.Biome in {BiomeWater, BiomeMeadow, BiomeDesert}
func TestGenerateWorld_ValidBiomes(t *testing.T) {
	cfg := smallCfg()
	tiles := gen.GenerateWorld(cfg, newRng(42))
	for i, tile := range tiles {
		switch tile.Biome {
		case world.BiomeWater, world.BiomeMeadow, world.BiomeDesert:
			// valid
		default:
			t.Errorf("tiles[%d].Biome = %d, kein gültiges Biom", i, tile.Biome)
		}
	}
}

// TestGenerateWorld_FoodInit: tile.Food == tile.FoodMax für alle Tiles (Startzustand)
func TestGenerateWorld_FoodInit(t *testing.T) {
	cfg := smallCfg()
	tiles := gen.GenerateWorld(cfg, newRng(42))
	for i, tile := range tiles {
		if tile.Food != tile.FoodMax {
			t.Errorf("tiles[%d]: Food=%f != FoodMax=%f", i, tile.Food, tile.FoodMax)
		}
	}
}

// TestGenerateWorld_NoFoodOnWater: Wasser-Tiles haben Food == 0 und FoodMax == 0
func TestGenerateWorld_NoFoodOnWater(t *testing.T) {
	cfg := smallCfg()
	tiles := gen.GenerateWorld(cfg, newRng(42))
	for i, tile := range tiles {
		if tile.Biome == world.BiomeWater {
			if tile.Food != 0 {
				t.Errorf("tiles[%d] (Water): Food=%f, want 0", i, tile.Food)
			}
			if tile.FoodMax != 0 {
				t.Errorf("tiles[%d] (Water): FoodMax=%f, want 0", i, tile.FoodMax)
			}
		}
	}
}

// TestGenerateWorld_Determinism: Gleicher Seed → identisches []world.Tile
func TestGenerateWorld_Determinism(t *testing.T) {
	cfg := smallCfg()
	tiles1 := gen.GenerateWorld(cfg, newRng(12345))
	tiles2 := gen.GenerateWorld(cfg, newRng(12345))
	if !reflect.DeepEqual(tiles1, tiles2) {
		t.Error("GenerateWorld ist nicht deterministisch: gleicher Seed liefert unterschiedliche Ergebnisse")
	}
}

// TestGenerateWorld_DifferentSeeds: Verschiedene Seeds → unterschiedliche Welten
func TestGenerateWorld_DifferentSeeds(t *testing.T) {
	cfg := smallCfg()
	tiles1 := gen.GenerateWorld(cfg, newRng(1))
	tiles2 := gen.GenerateWorld(cfg, newRng(2))
	if reflect.DeepEqual(tiles1, tiles2) {
		t.Error("GenerateWorld liefert bei verschiedenen Seeds identische Ergebnisse — unwahrscheinlich, aber möglich")
	}
}

// TestGenerateWorld_FoodInvariant (rapid): tile.Food <= tile.FoodMax für alle Tiles
func TestGenerateWorld_FoodInvariant(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		seed := rapid.Int64().Draw(rt, "seed")
		w := rapid.IntRange(5, 30).Draw(rt, "width")
		h := rapid.IntRange(5, 30).Draw(rt, "height")

		cfg := config.DefaultConfig()
		cfg.WorldWidth = w
		cfg.WorldHeight = h
		// NumPartitions muss kompatibel mit Weltgröße sein
		cfg.NumPartitions = 1

		tiles := gen.GenerateWorld(cfg, newRng(seed))
		for i, tile := range tiles {
			if tile.Food > tile.FoodMax {
				rt.Errorf("tiles[%d]: Food=%f > FoodMax=%f", i, tile.Food, tile.FoodMax)
			}
		}
	})
}

// TestGenerateWorld_BiomeDistribution (rapid): grobe Biom-Verteilungsprüfung nach CA.
// Nach dem Cellular Automaton kann die Verteilung von der ursprünglichen 60/20/20 abweichen,
// aber keine Biom-Klasse sollte komplett verschwinden oder dominieren (>95%).
func TestGenerateWorld_BiomeDistribution(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		seed := rapid.Int64().Draw(rt, "seed")

		cfg := config.DefaultConfig()
		cfg.WorldWidth = 60
		cfg.WorldHeight = 60
		cfg.NumPartitions = 1

		tiles := gen.GenerateWorld(cfg, newRng(seed))
		total := len(tiles)

		counts := [3]int{}
		for _, tile := range tiles {
			counts[tile.Biome]++
		}

		// Keine Biom-Klasse sollte mehr als 95% der Fläche belegen
		for b := range 3 {
			frac := float64(counts[b]) / float64(total)
			if frac > 0.95 {
				rt.Errorf("Biom %d belegt %.1f%% der Fläche — zu dominant", b, frac*100)
			}
		}
	})
}
