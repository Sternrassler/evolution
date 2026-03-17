package testworld

import (
	"image"
	"math/rand"

	"github.com/Sternrassler/evolution/config"
	"github.com/Sternrassler/evolution/sim/entity"
	"github.com/Sternrassler/evolution/sim/world"
)

// Builder baut eine kleine echte WorldContext-Implementierung für Tests.
// Keine Mocks — echte Semantik.
type Builder struct {
	width, height int
	tiles         []world.Tile
	individuals   []entity.Individual
	rng           entity.RandSource
	cfg           config.Config
}

// New erstellt einen Builder für eine width×height-Welt
// (alle Tiles: BiomeMeadow, Food=FoodMax=10).
func New(width, height int) *Builder {
	tiles := make([]world.Tile, width*height)
	for i := range tiles {
		tiles[i] = world.Tile{
			Biome:   world.BiomeMeadow,
			Food:    10,
			FoodMax: 10,
		}
	}
	return &Builder{
		width:  width,
		height: height,
		tiles:  tiles,
		cfg:    config.DefaultConfig(),
	}
}

// WithTile setzt die Tile an Position (x,y).
func (b *Builder) WithTile(x, y int, t world.Tile) *Builder {
	if x >= 0 && x < b.width && y >= 0 && y < b.height {
		b.tiles[y*b.width+x] = t
	}
	return b
}

// WithIndividual fügt ein Individuum hinzu.
func (b *Builder) WithIndividual(ind entity.Individual) *Builder {
	b.individuals = append(b.individuals, ind)
	return b
}

// WithRng setzt die Zufallsquelle (Default: rand.New(rand.NewSource(42))).
func (b *Builder) WithRng(rng entity.RandSource) *Builder {
	b.rng = rng
	return b
}

// WithConfig setzt Config-Werte (für MutationRate etc.).
func (b *Builder) WithConfig(cfg config.Config) *Builder {
	b.cfg = cfg
	return b
}

// Build erstellt die WorldContext-Implementierung.
func (b *Builder) Build() *WorldCtx {
	rng := b.rng
	if rng == nil {
		rng = rand.New(rand.NewSource(42))
	}

	grid := world.NewGrid(b.width, b.height)
	copy(grid.Tiles, b.tiles)

	individuals := make([]entity.Individual, len(b.individuals))
	copy(individuals, b.individuals)

	return &WorldCtx{
		grid:        grid,
		individuals: individuals,
		rng:         rng,
		cfg:         b.cfg,
		nearBuf:     make([]int32, 0, 64),
	}
}

// WorldCtx ist eine echte, leichtgewichtige WorldContext-Implementierung für Tests.
type WorldCtx struct {
	grid        *world.Grid
	individuals []entity.Individual
	rng         entity.RandSource
	cfg         config.Config
	nearBuf     []int32 // reused buffer für IndividualsNear (zero-alloc)
}

// TileAt gibt die Tile an Position p zurück.
// Gibt BiomeMeadow-Tile zurück wenn out of bounds (safe fallback).
func (w *WorldCtx) TileAt(p image.Point) world.Tile {
	if !w.grid.InBounds(p.X, p.Y) {
		return world.Tile{
			Biome:   world.BiomeMeadow,
			Food:    10,
			FoodMax: 10,
		}
	}
	return *w.grid.At(p.X, p.Y)
}

// IndividualsNear gibt die Indizes aller Individuen zurück, deren euklidische
// Distanz zu p ≤ radius ist. Verwendet w.nearBuf als Output (zero-alloc für
// den Test-Kontext). Einfache lineare Suche — kein Performance-Anspruch.
func (w *WorldCtx) IndividualsNear(p image.Point, radius int) []int32 {
	w.nearBuf = w.nearBuf[:0]
	r2 := float64(radius * radius)
	for i, ind := range w.individuals {
		dx := float64(ind.Pos.X - p.X)
		dy := float64(ind.Pos.Y - p.Y)
		if dx*dx+dy*dy <= r2 {
			w.nearBuf = append(w.nearBuf, int32(i))
		}
	}
	return w.nearBuf
}

// Rand gibt die injizierte Zufallsquelle zurück.
func (w *WorldCtx) Rand() entity.RandSource { return w.rng }

// MutationRate gibt die Mutationsrate zurück.
// Verwendet GeneDefinitions[0].MutationRate oder 0.1 als Fallback.
func (w *WorldCtx) MutationRate() float32 {
	if len(w.cfg.GeneDefinitions) > 0 {
		return w.cfg.GeneDefinitions[0].MutationRate
	}
	return 0.1
}

// ReproductionThreshold gibt den Schwellwert für Reproduktion zurück.
func (w *WorldCtx) ReproductionThreshold() float32 {
	return w.cfg.ReproductionThreshold
}

// MaxSpeed gibt den maximalen Geschwindigkeitsbereich zurück.
func (w *WorldCtx) MaxSpeed() float32 {
	return float32(w.cfg.MaxSpeedRange)
}

// MaxSight gibt den maximalen Sichtbereich zurück.
func (w *WorldCtx) MaxSight() float32 {
	return float32(w.cfg.MaxSightRange)
}

