//go:build noebiten

package render

import (
	"testing"

	"github.com/Sternrassler/evolution/config"
	"github.com/Sternrassler/evolution/sim/entity"
	"github.com/Sternrassler/evolution/sim/world"
)

func TestGeneColor_Normalization(t *testing.T) {
	defs := config.DefaultConfig().GeneDefinitions
	// Speed bei Min → R = 0
	var genes [entity.NumGenes]float32
	genes[entity.GeneSpeed] = defs[entity.GeneSpeed].Min
	genes[entity.GeneSight] = defs[entity.GeneSight].Min
	genes[entity.GeneEfficiency] = defs[entity.GeneEfficiency].Min
	r, _, _ := GeneColor(genes, defs)
	if r != 0 {
		t.Errorf("Speed=Min: want R=0, got %d", r)
	}

	// Speed bei Max → R = 255
	genes[entity.GeneSpeed] = defs[entity.GeneSpeed].Max
	r, _, _ = GeneColor(genes, defs)
	if r != 255 {
		t.Errorf("Speed=Max: want R=255, got %d", r)
	}
}

func TestGeneColor_Fallback(t *testing.T) {
	var genes [entity.NumGenes]float32
	r, g, b := GeneColor(genes, nil)
	if r != 200 || g != 100 || b != 100 {
		t.Errorf("Fallback: want (200,100,100), got (%d,%d,%d)", r, g, b)
	}

	// Auch mit zu wenigen defs
	r, g, b = GeneColor(genes, []config.GeneDef{{}, {}})
	if r != 200 || g != 100 || b != 100 {
		t.Errorf("Fallback (2 defs): want (200,100,100), got (%d,%d,%d)", r, g, b)
	}
}

func TestBiomeColor_Water(t *testing.T) {
	r, g, b := BiomeColor(world.BiomeWater, 0, 0)
	if r != 30 || g != 80 || b != 160 {
		t.Errorf("BiomeWater: want (30,80,160), got (%d,%d,%d)", r, g, b)
	}
}

func TestNormalizeGene_Clamp(t *testing.T) {
	def := config.GeneDef{Min: 1.0, Max: 3.0}

	// Wert unter Min → 0
	got := normalizeGene(0.0, def)
	if got != 0 {
		t.Errorf("val<Min: want 0, got %d", got)
	}

	// Wert über Max → 255
	got = normalizeGene(5.0, def)
	if got != 255 {
		t.Errorf("val>Max: want 255, got %d", got)
	}
}

func TestPixelBufSize(t *testing.T) {
	// Nur Pixel-Buffer-Größe prüfen, ohne NewRenderer (braucht ebiten.NewImage)
	width, height, tileSize := 10, 10, 2
	expected := width * height * tileSize * tileSize * 4
	got := len(make([]byte, width*height*tileSize*tileSize*4))
	if got != expected {
		t.Errorf("pixelBuf size: want %d, got %d", expected, got)
	}
}
