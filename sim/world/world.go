package world

import (
	"image"

	"github.com/Sternrassler/evolution/sim/entity"
)

// BiomeType klassifiziert den Geländetyp einer Tile.
type BiomeType uint8

const (
	BiomeWater  BiomeType = iota // nicht begehbar, kein Nahrungswachstum
	BiomeMeadow                  // normales Nahrungswachstum
	BiomeDesert                  // langsames Nahrungswachstum
)

// Tile ist eine Zelle in der Simulationswelt.
type Tile struct {
	Biome   BiomeType
	Food    float32
	FoodMax float32
}

// IsWalkable gibt zurück, ob ein Individuum diese Tile betreten darf.
func (t Tile) IsWalkable() bool { return t.Biome != BiomeWater }

// Grid hält das 2D-Feld aller Tiles (row-major: Tiles[y*Width+x]).
type Grid struct {
	Tiles  []Tile
	Width  int
	Height int
}

// NewGrid erstellt ein leeres Grid der angegebenen Größe.
func NewGrid(width, height int) *Grid {
	return &Grid{
		Tiles:  make([]Tile, width*height),
		Width:  width,
		Height: height,
	}
}

// At gibt einen Zeiger auf die Tile an Position (x, y) zurück.
// Panic bei Out-of-Bounds (nur in Debug-Pfaden aufrufen).
func (g *Grid) At(x, y int) *Tile {
	return &g.Tiles[y*g.Width+x]
}

// InBounds prüft ob (x,y) innerhalb des Grids liegt.
func (g *Grid) InBounds(x, y int) bool {
	return x >= 0 && x < g.Width && y >= 0 && y < g.Height
}

// ApplyRegrowth wächst Nahrung auf allen nicht-Wasser-Tiles nach.
// meadowRate und desertRate sind Anteile von FoodMax pro Tick (aus config.Config).
// Gibt die gesamte gewachsene Energie zurück (für TickStats.EnergyRegrown).
// Diese Methode mutiert g.Tiles — nur in Phase 2 aufrufen.
func (g *Grid) ApplyRegrowth(meadowRate, desertRate float32) float32 {
	var total float32
	for i := range g.Tiles {
		t := &g.Tiles[i]
		if t.Biome == BiomeWater {
			continue
		}
		var rate float32
		switch t.Biome {
		case BiomeMeadow:
			rate = meadowRate
		case BiomeDesert:
			rate = desertRate
		}
		if rate == 0 || t.Food >= t.FoodMax {
			continue
		}
		delta := rate * t.FoodMax
		newFood := t.Food + delta
		if newFood > t.FoodMax {
			delta = t.FoodMax - t.Food
			newFood = t.FoodMax
		}
		t.Food = newFood
		total += delta
	}
	return total
}

// WorldContext ist die read-only Weltansicht für Phase-1-Agenten.
// Wird von partition und testworld implementiert.
// RandSource ist entity.RandSource (kein extra Import nötig, da entity bereits importiert).
type WorldContext interface {
	TileAt(p image.Point) Tile
	IndividualsNear(p image.Point, radius int) []int32 // SoA-Indizes, zero-alloc
	Rand() entity.RandSource
	MutationRate() float32
	ReproductionThreshold() float32
	MaxSpeed() float32
	MaxSight() float32
}
