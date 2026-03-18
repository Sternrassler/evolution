package ui

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/Sternrassler/evolution/render"
)

// InputHandler verarbeitet Tastatureingaben.
type InputHandler struct {
	Paused   bool
	StepOnce bool
}

// Process verarbeitet alle Eingaben für diesen Frame.
func (h *InputHandler) Process(g *Game) error {
	// Space → Pause togglen
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		h.Paused = !h.Paused
	}
	// Right-Arrow → Next Step (nur wenn paused)
	if h.Paused && inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) {
		h.StepOnce = true
	}
	// Escape → Beenden
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return ebiten.Termination
	}
	// 1–4 → Ansicht wechseln
	for i, key := range []ebiten.Key{ebiten.Key1, ebiten.Key2, ebiten.Key3, ebiten.Key4} {
		if inpututil.IsKeyJustPressed(key) {
			g.viewMode = render.ViewMode(i + 1)
		}
	}
	return nil
}
