//go:build !noebiten

package render

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/Sternrassler/evolution/config"
	"github.com/Sternrassler/evolution/sim"
	"github.com/Sternrassler/evolution/sim/entity"
	"github.com/Sternrassler/evolution/sim/world"
)

const defaultTileSize = 4

// Renderer verwaltet den Pixel-Buffer und rendert WorldSnapshots.
// RenderToBuffer ist zero-alloc nach Initialisierung.
type Renderer struct {
	pixelBuf     []byte
	offscreen    *ebiten.Image
	tileSize     int
	width        int
	height       int
	cfg          config.Config
	densityBuf   []int
	geneSumBuf   []float32 // width*height*NumGenes
	geneCountBuf []int
}

// NewRenderer erstellt einen Renderer mit pre-allokierten Buffern.
func NewRenderer(cfg config.Config, tileSize int) *Renderer {
	if tileSize <= 0 {
		tileSize = defaultTileSize
	}
	pw := cfg.WorldWidth * tileSize
	ph := cfg.WorldHeight * tileSize
	n := cfg.WorldWidth * cfg.WorldHeight
	return &Renderer{
		pixelBuf:     make([]byte, pw*ph*4),
		offscreen:    ebiten.NewImage(pw, ph),
		tileSize:     tileSize,
		width:        cfg.WorldWidth,
		height:       cfg.WorldHeight,
		cfg:          cfg,
		densityBuf:   make([]int, n),
		geneSumBuf:   make([]float32, n*entity.NumGenes),
		geneCountBuf: make([]int, n),
	}
}

// RenderToBuffer schreibt den WorldSnapshot in den Pixel-Buffer.
func (r *Renderer) RenderToBuffer(snap *sim.WorldSnapshot, mode ViewMode) {
	if snap == nil {
		return
	}
	switch mode {
	case ViewDichte:
		r.renderDichte(snap)
	case ViewGenotyp:
		r.renderGenotyp(snap)
	case ViewNahrung:
		r.renderNahrung(snap.Tiles)
	default: // ViewBiom
		r.renderTiles(snap.Tiles)
		r.renderIndividuals(snap)
	}
}

func (r *Renderer) renderTiles(tiles []world.Tile) {
	ts := r.tileSize
	pw := r.width * ts
	for y := range r.height {
		for x := range r.width {
			idx := y*r.width + x
			if idx >= len(tiles) {
				break
			}
			t := tiles[idx]
			tr, tg, tb := BiomeColor(t.Biome, t.Food, t.FoodMax)
			for py := range ts {
				for px := range ts {
					pIdx := ((y*ts+py)*pw + (x*ts + px)) * 4
					r.pixelBuf[pIdx] = tr
					r.pixelBuf[pIdx+1] = tg
					r.pixelBuf[pIdx+2] = tb
					r.pixelBuf[pIdx+3] = 255
				}
			}
		}
	}
}

func (r *Renderer) renderIndividuals(snap *sim.WorldSnapshot) {
	ts := r.tileSize
	pw := r.width * ts
	defs := r.cfg.GeneDefinitions
	for _, ind := range snap.Individuals {
		x, y := ind.Pos.X, ind.Pos.Y
		if x < 0 || x >= r.width || y < 0 || y >= r.height {
			continue
		}
		var ir, ig, ib uint8
		if ind.EntityType == entity.Predator {
			ir, ig, ib = 255, 60, 60
		} else {
			ir, ig, ib = GeneColor(ind.Genes, defs)
		}
		cx := x*ts + ts/2
		cy := y*ts + ts/2
		pIdx := (cy*pw + cx) * 4
		if pIdx+3 < len(r.pixelBuf) {
			r.pixelBuf[pIdx] = ir
			r.pixelBuf[pIdx+1] = ig
			r.pixelBuf[pIdx+2] = ib
			r.pixelBuf[pIdx+3] = 255
		}
	}
}

func (r *Renderer) renderDichte(snap *sim.WorldSnapshot) {
	for i := range r.densityBuf {
		r.densityBuf[i] = 0
	}
	for _, ind := range snap.Individuals {
		x, y := ind.Pos.X, ind.Pos.Y
		if x >= 0 && x < r.width && y >= 0 && y < r.height {
			r.densityBuf[y*r.width+x]++
		}
	}
	maxD := 1
	for _, d := range r.densityBuf {
		if d > maxD {
			maxD = d
		}
	}
	ts := r.tileSize
	pw := r.width * ts
	for y := range r.height {
		for x := range r.width {
			d := r.densityBuf[y*r.width+x]
			tr, tg, tb := DensityColor(d, maxD)
			for py := range ts {
				for px := range ts {
					pIdx := ((y*ts+py)*pw + (x*ts + px)) * 4
					r.pixelBuf[pIdx] = tr
					r.pixelBuf[pIdx+1] = tg
					r.pixelBuf[pIdx+2] = tb
					r.pixelBuf[pIdx+3] = 255
				}
			}
		}
	}
}

func (r *Renderer) renderGenotyp(snap *sim.WorldSnapshot) {
	n := r.width * r.height
	for i := range r.geneCountBuf {
		r.geneCountBuf[i] = 0
	}
	for i := range r.geneSumBuf {
		r.geneSumBuf[i] = 0
	}
	for _, ind := range snap.Individuals {
		x, y := ind.Pos.X, ind.Pos.Y
		if x < 0 || x >= r.width || y < 0 || y >= r.height {
			continue
		}
		if ind.EntityType == entity.Predator {
			continue // Räuber nicht in Gen-Durchschnitt einbeziehen
		}
		base := (y*r.width + x) * entity.NumGenes
		for g := range entity.NumGenes {
			r.geneSumBuf[base+g] += ind.Genes[g]
		}
		r.geneCountBuf[y*r.width+x]++
	}
	ts := r.tileSize
	pw := r.width * ts
	defs := r.cfg.GeneDefinitions
	for i := range n {
		y := i / r.width
		x := i % r.width
		var ir, ig, ib uint8
		if cnt := r.geneCountBuf[i]; cnt > 0 {
			base := i * entity.NumGenes
			var avg [entity.NumGenes]float32
			for g := range entity.NumGenes {
				avg[g] = r.geneSumBuf[base+g] / float32(cnt)
			}
			ir, ig, ib = GeneColor(avg, defs)
		} else {
			ir, ig, ib = 15, 15, 15
		}
		for py := range ts {
			for px := range ts {
				pIdx := ((y*ts+py)*pw + (x*ts + px)) * 4
				r.pixelBuf[pIdx] = ir
				r.pixelBuf[pIdx+1] = ig
				r.pixelBuf[pIdx+2] = ib
				r.pixelBuf[pIdx+3] = 255
			}
		}
	}
	// Räuber als rote Mittelpixel überlagern
	r.renderIndividualsPredatorOnly(snap, pw)
}

func (r *Renderer) renderIndividualsPredatorOnly(snap *sim.WorldSnapshot, pw int) {
	ts := r.tileSize
	for _, ind := range snap.Individuals {
		if ind.EntityType != entity.Predator {
			continue
		}
		x, y := ind.Pos.X, ind.Pos.Y
		if x < 0 || x >= r.width || y < 0 || y >= r.height {
			continue
		}
		cx := x*ts + ts/2
		cy := y*ts + ts/2
		pIdx := (cy*pw + cx) * 4
		if pIdx+3 < len(r.pixelBuf) {
			r.pixelBuf[pIdx] = 255
			r.pixelBuf[pIdx+1] = 255
			r.pixelBuf[pIdx+2] = 255
			r.pixelBuf[pIdx+3] = 255
		}
	}
}

func (r *Renderer) renderNahrung(tiles []world.Tile) {
	ts := r.tileSize
	pw := r.width * ts
	for y := range r.height {
		for x := range r.width {
			idx := y*r.width + x
			if idx >= len(tiles) {
				break
			}
			t := tiles[idx]
			tr, tg, tb := FoodOnlyColor(t.Biome, t.Food, t.FoodMax)
			for py := range ts {
				for px := range ts {
					pIdx := ((y*ts+py)*pw + (x*ts + px)) * 4
					r.pixelBuf[pIdx] = tr
					r.pixelBuf[pIdx+1] = tg
					r.pixelBuf[pIdx+2] = tb
					r.pixelBuf[pIdx+3] = 255
				}
			}
		}
	}
}

// DrawBuffer schreibt den Pixel-Buffer auf den Screen.
func (r *Renderer) DrawBuffer(screen *ebiten.Image) {
	r.offscreen.WritePixels(r.pixelBuf)
	screen.DrawImage(r.offscreen, nil)
}

// ScreenSize gibt die Pixel-Dimensionen des Render-Outputs zurück.
func (r *Renderer) ScreenSize() (int, int) {
	return r.width * r.tileSize, r.height * r.tileSize
}
