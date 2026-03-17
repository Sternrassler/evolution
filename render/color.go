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
