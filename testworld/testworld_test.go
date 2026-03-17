package testworld_test

import (
	"image"
	"math/rand"
	"testing"

	"github.com/Sternrassler/evolution/sim/entity"
	"github.com/Sternrassler/evolution/sim/world"
	"github.com/Sternrassler/evolution/testworld"
)

// TestWorldCtx_ImplementsInterface ist ein Compile-time-Check.
var _ world.WorldContext = (*testworld.WorldCtx)(nil)

// TestNew_DefaultTiles prüft, dass alle Tiles BiomeMeadow mit Food=10 sind.
func TestNew_DefaultTiles(t *testing.T) {
	ctx := testworld.New(5, 5).Build()
	for y := range 5 {
		for x := range 5 {
			tile := ctx.TileAt(image.Pt(x, y))
			if tile.Biome != world.BiomeMeadow {
				t.Errorf("TileAt(%d,%d).Biome = %v, want BiomeMeadow", x, y, tile.Biome)
			}
			if tile.Food != 10 {
				t.Errorf("TileAt(%d,%d).Food = %v, want 10", x, y, tile.Food)
			}
			if tile.FoodMax != 10 {
				t.Errorf("TileAt(%d,%d).FoodMax = %v, want 10", x, y, tile.FoodMax)
			}
		}
	}
}

// TestWithTile prüft, dass eine gesetzte Tile korrekt via TileAt zurückgegeben wird.
func TestWithTile(t *testing.T) {
	customTile := world.Tile{Biome: world.BiomeDesert, Food: 3, FoodMax: 5}
	ctx := testworld.New(10, 10).
		WithTile(2, 3, customTile).
		Build()

	got := ctx.TileAt(image.Pt(2, 3))
	if got != customTile {
		t.Errorf("TileAt(2,3) = %+v, want %+v", got, customTile)
	}

	// Andere Tiles bleiben unverändert
	other := ctx.TileAt(image.Pt(0, 0))
	if other.Biome != world.BiomeMeadow {
		t.Errorf("TileAt(0,0).Biome = %v, want BiomeMeadow", other.Biome)
	}
}

// TestWithIndividual_IndividualsNear prüft, dass ein Individuum im Radius gefunden wird
// und eines außerhalb nicht.
func TestWithIndividual_IndividualsNear(t *testing.T) {
	indInside := entity.NewIndividual(1, image.Pt(5, 5), [entity.NumGenes]float32{}, 50)
	indOutside := entity.NewIndividual(2, image.Pt(20, 20), [entity.NumGenes]float32{}, 50)

	ctx := testworld.New(30, 30).
		WithIndividual(indInside).
		WithIndividual(indOutside).
		Build()

	// Suche mit Radius 3 um (5,5): indInside (Distanz 0) soll gefunden werden,
	// indOutside (Distanz ~21) nicht.
	near := ctx.IndividualsNear(image.Pt(5, 5), 3)
	if len(near) != 1 {
		t.Fatalf("IndividualsNear returned %d results, want 1", len(near))
	}
	if near[0] != 0 {
		t.Errorf("IndividualsNear returned index %d, want 0 (indInside)", near[0])
	}

	// Individuum genau am Rand des Radius
	indEdge := entity.NewIndividual(3, image.Pt(8, 5), [entity.NumGenes]float32{}, 50)
	ctx2 := testworld.New(30, 30).
		WithIndividual(indEdge).
		Build()

	nearEdge := ctx2.IndividualsNear(image.Pt(5, 5), 3)
	if len(nearEdge) != 1 {
		t.Errorf("IndividualsNear mit Distanz=3, radius=3: got %d results, want 1", len(nearEdge))
	}

	// Individuum knapp außerhalb
	indJustOut := entity.NewIndividual(4, image.Pt(9, 5), [entity.NumGenes]float32{}, 50)
	ctx3 := testworld.New(30, 30).
		WithIndividual(indJustOut).
		Build()

	nearOut := ctx3.IndividualsNear(image.Pt(5, 5), 3)
	if len(nearOut) != 0 {
		t.Errorf("IndividualsNear mit Distanz=4, radius=3: got %d results, want 0", len(nearOut))
	}
}

// TestRand_Deterministic prüft, dass gleicher Seed → gleiche Zufallsfolge liefert.
func TestRand_Deterministic(t *testing.T) {
	rng1 := rand.New(rand.NewSource(99))
	rng2 := rand.New(rand.NewSource(99))

	ctx1 := testworld.New(10, 10).WithRng(rng1).Build()
	ctx2 := testworld.New(10, 10).WithRng(rng2).Build()

	const n = 20
	for i := range n {
		v1 := ctx1.Rand().Float64()
		v2 := ctx2.Rand().Float64()
		if v1 != v2 {
			t.Errorf("Rand().Float64() [%d]: got %v and %v, want equal", i, v1, v2)
		}
	}
}

// TestTileAt_OutOfBounds prüft den sicheren Fallback bei Out-of-bounds-Zugriff.
func TestTileAt_OutOfBounds(t *testing.T) {
	ctx := testworld.New(5, 5).Build()
	tile := ctx.TileAt(image.Pt(100, 100))
	if tile.Biome != world.BiomeMeadow {
		t.Errorf("TileAt out-of-bounds Biome = %v, want BiomeMeadow", tile.Biome)
	}
}
