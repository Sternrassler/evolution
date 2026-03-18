package render

import (
	"github.com/Sternrassler/evolution/config"
	"github.com/Sternrassler/evolution/sim/entity"
	"github.com/Sternrassler/evolution/sim/world"
)

// BiomeColor gibt die RGBA-Grundfarbe für einen Biom-Typ zurück.
func BiomeColor(biome world.BiomeType, food, foodMax float32) (r, g, b uint8) {
	switch biome {
	case world.BiomeMeadow:
		// Grün, dunkler wenn weniger Nahrung
		intensity := uint8(60 + 100*food/max(foodMax, 0.001))
		return 40, intensity, 30
	case world.BiomeDesert:
		// Sandgelb, dunkler bei weniger Nahrung
		intensity := uint8(140 + 60*food/max(foodMax, 0.001))
		return intensity, intensity - 20, 60
	default: // BiomeWater
		return 30, 80, 160
	}
}

// GeneColor kodiert Genotyp als RGB:
// R = Speed normiert [Min,Max] → [0,255]
// G = Sight normiert [Min,Max] → [0,255]
// B = Efficiency normiert [Min,Max] → [0,255]
func GeneColor(genes [entity.NumGenes]float32, defs []config.GeneDef) (r, g, b uint8) {
	if len(defs) < 3 {
		return 200, 100, 100 // Fallback
	}
	r = normalizeGene(genes[entity.GeneSpeed], defs[entity.GeneSpeed])
	g = normalizeGene(genes[entity.GeneSight], defs[entity.GeneSight])
	b = normalizeGene(genes[entity.GeneEfficiency], defs[entity.GeneEfficiency])
	return
}

// DensityColor gibt eine Heatmap-Farbe für eine Populationsdichte zurück.
// 0 = fast schwarz, maxCount = hellgelb.
func DensityColor(count, maxCount int) (r, g, b uint8) {
	if count == 0 || maxCount == 0 {
		return 10, 10, 15
	}
	t := float32(count) / float32(maxCount)
	if t > 1 {
		t = 1
	}
	// schwarz → dunkelrot → orange → hellgelb
	switch {
	case t < 0.33:
		s := t / 0.33
		return uint8(180 * s), 0, 0
	case t < 0.66:
		s := (t - 0.33) / 0.33
		return 180 + uint8(75*s), uint8(80*s), 0
	default:
		s := (t - 0.66) / 0.34
		return 255, 80 + uint8(175*s), uint8(100*s)
	}
}

// FoodOnlyColor zeigt den Nahrungsfüllstand unabhängig vom Biom.
// Wasser bleibt blau, Land: dunkelgrau (leer) → hellgrün (voll).
func FoodOnlyColor(biome world.BiomeType, food, foodMax float32) (r, g, b uint8) {
	if biome == world.BiomeWater {
		return 30, 80, 160
	}
	if foodMax <= 0 {
		return 20, 20, 20
	}
	t := food / foodMax
	if t > 1 {
		t = 1
	}
	return uint8(20 + 30*t), uint8(20 + 200*t), uint8(20 + 20*t)
}

func normalizeGene(val float32, def config.GeneDef) uint8 {
	span := def.Max - def.Min
	if span <= 0 {
		return 128
	}
	norm := (val - def.Min) / span
	if norm < 0 {
		norm = 0
	}
	if norm > 1 {
		norm = 1
	}
	return uint8(norm * 255)
}
