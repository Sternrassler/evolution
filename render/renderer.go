//go:build !noebiten

package render

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/Sternrassler/evolution/config"
	"github.com/Sternrassler/evolution/sim"
	"github.com/Sternrassler/evolution/sim/world"
)

const defaultTileSize = 4

// Renderer verwaltet den Pixel-Buffer und rendert WorldSnapshots.
// RenderToBuffer ist zero-alloc nach Initialisierung.
type Renderer struct {
	pixelBuf  []byte         // RGBA, pre-allokiert: width*height*tileSize^2*4
	offscreen *ebiten.Image
	tileSize  int
	width     int // in Tiles
	height    int // in Tiles
	cfg       config.Config
}

// NewRenderer erstellt einen Renderer mit pre-allokiertem Pixel-Buffer.
func NewRenderer(cfg config.Config, tileSize int) *Renderer {
	if tileSize <= 0 {
		tileSize = defaultTileSize
	}
	pw := cfg.WorldWidth * tileSize
	ph := cfg.WorldHeight * tileSize
	return &Renderer{
		pixelBuf:  make([]byte, pw*ph*4),
		offscreen: ebiten.NewImage(pw, ph),
		tileSize:  tileSize,
		width:     cfg.WorldWidth,
		height:    cfg.WorldHeight,
		cfg:       cfg,
	}
}

// RenderToBuffer schreibt den WorldSnapshot in den Pixel-Buffer.
// Zero-alloc nach Initialisierung.
func (r *Renderer) RenderToBuffer(snap *sim.WorldSnapshot) {
	if snap == nil {
		return
	}
	r.renderTiles(snap.Tiles)
	r.renderIndividuals(snap)
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
			// Alle Pixel dieser Tile füllen
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
		ir, ig, ib := GeneColor(ind.Genes, defs)
		if ts >= 4 {
			// Mittelpunkt der Tile setzen
			cx := x*ts + ts/2
			cy := y*ts + ts/2
			pIdx := (cy*pw + cx) * 4
			if pIdx+3 < len(r.pixelBuf) {
				r.pixelBuf[pIdx] = ir
				r.pixelBuf[pIdx+1] = ig
				r.pixelBuf[pIdx+2] = ib
				r.pixelBuf[pIdx+3] = 255
			}
		} else {
			// Klein: gesamte Tile überschreiben
			pIdx := (y*ts*pw + x*ts) * 4
			if pIdx+3 < len(r.pixelBuf) {
				r.pixelBuf[pIdx] = ir
				r.pixelBuf[pIdx+1] = ig
				r.pixelBuf[pIdx+2] = ib
				r.pixelBuf[pIdx+3] = 255
			}
		}
	}
}

// DrawBuffer schreibt den Pixel-Buffer auf den Screen. Immer aufgerufen.
func (r *Renderer) DrawBuffer(screen *ebiten.Image) {
	r.offscreen.WritePixels(r.pixelBuf)
	screen.DrawImage(r.offscreen, nil)
}

// ScreenSize gibt die Pixel-Dimensionen des Render-Outputs zurück.
func (r *Renderer) ScreenSize() (int, int) {
	return r.width * r.tileSize, r.height * r.tileSize
}
