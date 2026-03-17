package ui

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
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
	return nil
}
