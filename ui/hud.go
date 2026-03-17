package ui

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"

	"github.com/Sternrassler/evolution/sim"
	"github.com/Sternrassler/evolution/sim/entity"
)

// HUD zeigt Simulations-Statistiken im Fenster an.
type HUD struct{}

func NewHUD() *HUD { return &HUD{} }

// Draw zeichnet das HUD auf den Screen.
func (h *HUD) Draw(screen *ebiten.Image, snap *sim.WorldSnapshot) {
	if snap == nil {
		return
	}

	// Durchschnittsgene berechnen
	avgSpeed, avgSight, avgEff := avgGenes(snap.Individuals)

	text := fmt.Sprintf(
		"Tick: %d  Pop: %d  Births: %d  Deaths: %d\nØSpeed:%.2f  ØSight:%.2f  ØEffic:%.2f",
		snap.Tick, snap.Stats.Population, snap.Stats.Births, snap.Stats.Deaths,
		avgSpeed, avgSight, avgEff,
	)
	ebitenutil.DebugPrint(screen, text)
}

func avgGenes(inds []entity.Individual) (speed, sight, eff float32) {
	if len(inds) == 0 {
		return
	}
	for _, ind := range inds {
		speed += ind.Genes[entity.GeneSpeed]
		sight += ind.Genes[entity.GeneSight]
		eff += ind.Genes[entity.GeneEfficiency]
	}
	n := float32(len(inds))
	return speed / n, sight / n, eff / n
}
