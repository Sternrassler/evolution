// Package gen implementiert die prozedurale Weltgenerierung.
// Kein ebiten-Import (CI Gate 1), kein direktes math/rand (CI Gate 2).
package gen

import (
	"github.com/Sternrassler/evolution/config"
	"github.com/Sternrassler/evolution/sim/world"
)

// TileSource ist das Interface für austauschbare Weltgeneratoren.
// EditorSource (Stufe 4) implementiert dasselbe Interface.
type TileSource interface {
	Generate(cfg config.Config, rng config.RandSource) []world.Tile
}

// GenerateWorld ist eine pure function — kein globaler State, kein Side-Effect.
// Kein math/rand direkt — rng wird injiziert (CI Gate 2).
func GenerateWorld(cfg config.Config, rng config.RandSource) []world.Tile {
	src := ProceduralSource{}
	return src.Generate(cfg, rng)
}

// ProceduralSource: Cellular-Automaton-basierte Weltgenerierung.
type ProceduralSource struct{}

// Generate implementiert TileSource.
// Algorithmus:
//  1. Zufällige Biom-Belegung: 60% Wiese, 20% Wüste, 20% Wasser
//  2. 3 Iterationen Cellular Automaton (Majority-Rule, 3×3-Nachbarschaft)
//  3. Nahrungswerte für non-Water-Tiles auf FoodMax initialisieren
//
// FoodMax-Werte: Meadow=10.0, Desert=3.0, Water=0.0
func (p ProceduralSource) Generate(cfg config.Config, rng config.RandSource) []world.Tile {
	w, h := cfg.WorldWidth, cfg.WorldHeight
	tiles := make([]world.Tile, w*h)

	// Schritt 1: Zufällige Belegung
	for i := range tiles {
		r := rng.Float64()
		switch {
		case r < 0.60:
			tiles[i].Biome = world.BiomeMeadow
		case r < 0.80:
			tiles[i].Biome = world.BiomeDesert
		default:
			tiles[i].Biome = world.BiomeWater
		}
	}

	// Schritt 2: 3 Cellular-Automaton-Iterationen
	buf := make([]world.Tile, w*h)
	for range 3 {
		for y := range h {
			for x := range w {
				buf[y*w+x].Biome = majorityBiome(tiles, x, y, w, h)
			}
		}
		tiles, buf = buf, tiles
	}

	// Schritt 3: FoodMax und Food setzen
	for i := range tiles {
		switch tiles[i].Biome {
		case world.BiomeMeadow:
			tiles[i].FoodMax = 10.0
			tiles[i].Food = 10.0
		case world.BiomeDesert:
			tiles[i].FoodMax = 3.0
			tiles[i].Food = 3.0
		case world.BiomeWater:
			tiles[i].FoodMax = 0.0
			tiles[i].Food = 0.0
		}
	}
	return tiles
}

// majorityBiome bestimmt das dominante Biom in der 3×3-Nachbarschaft von (x,y).
func majorityBiome(tiles []world.Tile, x, y, w, h int) world.BiomeType {
	counts := [3]int{} // indexed by BiomeType
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			nx, ny := x+dx, y+dy
			if nx < 0 || nx >= w || ny < 0 || ny >= h {
				// Rand-Behandlung: Wasser außerhalb
				counts[world.BiomeWater]++
				continue
			}
			counts[tiles[ny*w+nx].Biome]++
		}
	}
	// Mehrheit bestimmen
	best := world.BiomeType(0)
	for i := 1; i < len(counts); i++ {
		if counts[i] > counts[best] {
			best = world.BiomeType(i)
		}
	}
	return best
}
