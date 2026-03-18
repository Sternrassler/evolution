package world

import (
	"image"
	"testing"

	"github.com/Sternrassler/evolution/sim/entity"
	"pgregory.net/rapid"
)

// ---------------------------------------------------------------------------
// Tile / Grid Tests
// ---------------------------------------------------------------------------

func TestTileWalkable(t *testing.T) {
	if (Tile{Biome: BiomeWater}).IsWalkable() {
		t.Error("BiomeWater sollte nicht begehbar sein")
	}
	if !(Tile{Biome: BiomeMeadow}).IsWalkable() {
		t.Error("BiomeMeadow sollte begehbar sein")
	}
	if !(Tile{Biome: BiomeDesert}).IsWalkable() {
		t.Error("BiomeDesert sollte begehbar sein")
	}
}

func TestNewGrid(t *testing.T) {
	g := NewGrid(10, 5)
	if g.Width != 10 || g.Height != 5 {
		t.Fatalf("erwartete Width=10, Height=5, got %d/%d", g.Width, g.Height)
	}
	if len(g.Tiles) != 50 {
		t.Fatalf("erwartete 50 Tiles, got %d", len(g.Tiles))
	}
	for i, tile := range g.Tiles {
		if tile.Biome != BiomeWater || tile.Food != 0 || tile.FoodMax != 0 {
			t.Fatalf("Tile[%d] sollte zero-value (BiomeWater) sein", i)
		}
	}
}

func TestGridAt(t *testing.T) {
	g := NewGrid(4, 4)
	// Setze eine Tile und prüfe ob At() dieselbe Adresse liefert
	g.Tiles[1*4+2] = Tile{Biome: BiomeMeadow, Food: 3.0, FoodMax: 10.0}
	tile := g.At(2, 1)
	if tile.Biome != BiomeMeadow || tile.Food != 3.0 || tile.FoodMax != 10.0 {
		t.Errorf("At(2,1) lieferte falsche Tile: %+v", tile)
	}
	// Mutation über At() wirkt sich auf Tiles-Slice aus
	tile.Food = 5.0
	if g.Tiles[1*4+2].Food != 5.0 {
		t.Error("At() sollte Pointer auf Tiles-Element zurückgeben")
	}
}

func TestApplyRegrowth_Budget(t *testing.T) {
	g := NewGrid(3, 1)
	g.Tiles[0] = Tile{Biome: BiomeMeadow, Food: 0.0, FoodMax: 10.0}
	g.Tiles[1] = Tile{Biome: BiomeDesert, Food: 5.0, FoodMax: 20.0}
	g.Tiles[2] = Tile{Biome: BiomeMeadow, Food: 8.0, FoodMax: 10.0}

	foodBefore := make([]float32, len(g.Tiles))
	for i, tile := range g.Tiles {
		foodBefore[i] = tile.Food
	}

	returned := g.ApplyRegrowth(0.05, 0.01)

	var actualDelta float32
	for i, tile := range g.Tiles {
		actualDelta += tile.Food - foodBefore[i]
	}

	// Erlauben kleinen float32-Rundungsfehler
	diff := returned - actualDelta
	if diff < 0 {
		diff = -diff
	}
	if diff > 1e-4 {
		t.Errorf("ApplyRegrowth() returned=%f, tatsächliches Delta=%f", returned, actualDelta)
	}
}

func TestApplyRegrowth_Capped(t *testing.T) {
	g := NewGrid(2, 1)
	g.Tiles[0] = Tile{Biome: BiomeMeadow, Food: 9.9, FoodMax: 10.0}
	g.Tiles[1] = Tile{Biome: BiomeDesert, Food: 19.8, FoodMax: 20.0}

	g.ApplyRegrowth(0.05, 0.01)

	for i, tile := range g.Tiles {
		if tile.Food > tile.FoodMax {
			t.Errorf("Tile[%d]: Food=%f überschreitet FoodMax=%f", i, tile.Food, tile.FoodMax)
		}
	}
}

func TestApplyRegrowth_WaterSkipped(t *testing.T) {
	g := NewGrid(1, 1)
	g.Tiles[0] = Tile{Biome: BiomeWater, Food: 0.0, FoodMax: 10.0}

	returned := g.ApplyRegrowth(0.05, 0.01)

	if returned != 0.0 {
		t.Errorf("Wasser-Tile sollte kein Wachstum haben, got %f", returned)
	}
	if g.Tiles[0].Food != 0.0 {
		t.Errorf("Wasser-Tile Food sollte 0 bleiben, got %f", g.Tiles[0].Food)
	}
}

// ---------------------------------------------------------------------------
// SpatialGrid Tests
// ---------------------------------------------------------------------------

func makeIndividual(id uint64, x, y int) entity.Individual {
	return entity.NewIndividual(id, image.Pt(x, y), [entity.NumGenes]float32{}, 1.0)
}

func TestSpatialGridRebuild(t *testing.T) {
	sg := NewSpatialGrid(10, 100, 100)

	inds := []entity.Individual{
		makeIndividual(1, 5, 5),
		makeIndividual(2, 15, 15),
		makeIndividual(3, 55, 55),
	}
	sg.Rebuild(inds)

	out := make([]int32, 0, 16)

	// Alle drei Individuen müssen in einem ausreichend großen Radius auffindbar sein
	out = sg.IndividualsNear(image.Pt(5, 5), 100, out)
	found := make(map[int32]bool)
	for _, idx := range out {
		found[idx] = true
	}
	for _, want := range []int32{0, 1, 2} {
		if !found[want] {
			t.Errorf("Individuum mit Index %d nicht gefunden", want)
		}
	}
}

func TestSpatialGridNear_Empty(t *testing.T) {
	sg := NewSpatialGrid(10, 100, 100)
	sg.Rebuild(nil)

	out := make([]int32, 0, 8)
	result := sg.IndividualsNear(image.Pt(50, 50), 10, out)
	if len(result) != 0 {
		t.Errorf("leere Welt: erwartete leeres Ergebnis, got %v", result)
	}
}

func TestSpatialGridNear_Radius(t *testing.T) {
	// cellSize=10, world 100x100 → 10x10 Zellen
	sg := NewSpatialGrid(10, 100, 100)

	inds := []entity.Individual{
		makeIndividual(1, 5, 5),   // Zelle (0,0)
		makeIndividual(2, 95, 95), // Zelle (9,9)
	}
	sg.Rebuild(inds)

	out := make([]int32, 0, 8)
	// Suche bei (5,5) mit Radius 5 → nur Zelle (0,0) innerhalb BBox
	result := sg.IndividualsNear(image.Pt(5, 5), 5, out)

	for _, idx := range result {
		if idx == 1 { // Index von Individuum bei (95,95)
			t.Error("Individuum bei (95,95) sollte bei Suche nahe (5,5) mit Radius 5 nicht erscheinen")
		}
	}
}

// ---------------------------------------------------------------------------
// Property Tests (rapid)
// ---------------------------------------------------------------------------

func TestApplyRegrowth_Property(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		width := rapid.IntRange(1, 10).Draw(rt, "width")
		height := rapid.IntRange(1, 10).Draw(rt, "height")
		g := NewGrid(width, height)

		for i := range g.Tiles {
			biome := BiomeType(rapid.IntRange(0, 2).Draw(rt, "biome"))
			foodMax := rapid.Float32Range(0, 100).Draw(rt, "foodMax")
			food := rapid.Float32Range(0, float32(foodMax)).Draw(rt, "food")
			g.Tiles[i] = Tile{Biome: biome, Food: food, FoodMax: foodMax}
		}

		foodBefore := make([]float32, len(g.Tiles))
		for i, tile := range g.Tiles {
			foodBefore[i] = tile.Food
		}

		returned := g.ApplyRegrowth(0.05, 0.01)

		// Invariante: kein Free Energy — returned darf maximal der tatsächlichen Zunahme entsprechen
		var maxPossible float32
		for i, tile := range g.Tiles {
			delta := tile.Food - foodBefore[i]
			if delta < 0 {
				rt.Fatalf("Tile[%d]: Nahrung abgenommen: vorher=%f nachher=%f", i, foodBefore[i], tile.Food)
			}
			maxPossible += delta
		}

		diff := returned - maxPossible
		if diff < 0 {
			diff = -diff
		}
		if diff > 1e-3 {
			rt.Fatalf("returned=%f weicht von tatsächlichem Delta=%f ab (diff=%f)", returned, maxPossible, diff)
		}

		// Invariante: Food überschreitet niemals FoodMax
		for i, tile := range g.Tiles {
			if tile.Food > tile.FoodMax+1e-5 {
				rt.Fatalf("Tile[%d]: Food=%f > FoodMax=%f", i, tile.Food, tile.FoodMax)
			}
		}
	})
}
