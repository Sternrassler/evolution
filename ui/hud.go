package ui

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/Sternrassler/evolution/config"
	"github.com/Sternrassler/evolution/sim"
	"github.com/Sternrassler/evolution/sim/entity"
)

// HUD zeigt Simulations-Statistiken und eine Farblegende im Fenster an.
type HUD struct{}

func NewHUD() *HUD { return &HUD{} }

// Draw zeichnet das HUD auf den Screen.
func (h *HUD) Draw(screen *ebiten.Image, snap *sim.WorldSnapshot, cfg config.Config) {
	if snap == nil {
		return
	}

	avgSpeed, avgSight, avgEff := avgGenes(snap.Individuals)

	text := fmt.Sprintf(
		"Tick: %d  Pop: %d  Births: %d  Deaths: %d\nØSpeed:%.2f  ØSight:%.2f  ØEffic:%.2f",
		snap.Tick, snap.Stats.Population, snap.Stats.Births, snap.Stats.Deaths,
		avgSpeed, avgSight, avgEff,
	)
	ebitenutil.DebugPrint(screen, text)

	drawLegend(screen)
	drawParamsPanel(screen, cfg)
}

// drawLegend zeichnet eine Farblegende unten rechts.
func drawLegend(screen *ebiten.Image) {
	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()

	const (
		boxW    = 160
		boxH    = 130
		padding = 6
		swatch  = 10 // Größe der Farbquadrate
		lineH   = 14
	)

	x0 := float32(sw - boxW - 4)
	y0 := float32(sh - boxH - 4)

	// Hintergrund-Box (halbtransparent schwarz)
	vector.FillRect(screen, x0, y0, boxW, boxH, color.RGBA{0, 0, 0, 180}, false)

	tx := int(x0) + padding
	ty := int(y0) + padding

	// --- Biome ---
	ebitenutil.DebugPrintAt(screen, "Gelände:", tx, ty)
	ty += lineH

	biomes := []struct {
		label string
		c     color.RGBA
	}{
		{"Wiese", color.RGBA{40, 160, 30, 255}},
		{"Wüste", color.RGBA{200, 180, 100, 255}},
		{"Wasser", color.RGBA{30, 80, 160, 255}},
	}
	for _, b := range biomes {
		vector.FillRect(screen, float32(tx), float32(ty), swatch, swatch, b.c, false)
		ebitenutil.DebugPrintAt(screen, b.label, tx+swatch+4, ty)
		ty += lineH
	}

	ty += 4 // Abstand

	// --- Individuen (Genfarben) ---
	ebitenutil.DebugPrintAt(screen, "Individuen (RGB=Gen):", tx, ty)
	ty += lineH

	genes := []struct {
		label string
		c     color.RGBA
	}{
		{"Rot   = Speed", color.RGBA{220, 60, 60, 255}},
		{"Grün  = Sight", color.RGBA{60, 220, 60, 255}},
		{"Blau  = Effiz.", color.RGBA{60, 60, 220, 255}},
	}
	for _, g := range genes {
		vector.FillRect(screen, float32(tx), float32(ty), swatch, swatch, g.c, false)
		ebitenutil.DebugPrintAt(screen, g.label, tx+swatch+4, ty)
		ty += lineH
	}
}

// drawParamsPanel zeichnet eine Simulations-Parameter-Übersicht unten links.
func drawParamsPanel(screen *ebiten.Image, cfg config.Config) {
	const (
		boxW    = 190
		boxH    = 120
		padding = 6
		lineH   = 14
	)

	x0 := float32(4)
	y0 := float32(screen.Bounds().Dy() - boxH - 4)

	vector.FillRect(screen, x0, y0, boxW, boxH, color.RGBA{0, 0, 0, 180}, false)

	tx := int(x0) + padding
	ty := int(y0) + padding

	ebitenutil.DebugPrintAt(screen, "Parameter:", tx, ty)
	ty += lineH

	lines := []string{
		fmt.Sprintf("Energie-Kosten:  %.2f/Tick", cfg.BaseEnergyCost),
		fmt.Sprintf("Repro-Schwelle:  %.0f E", cfg.ReproductionThreshold),
		fmt.Sprintf("Repro-Reserve:   %.0f E", cfg.ReproductionReserve),
		fmt.Sprintf("Nachwachs Wiese: %.4f", cfg.RegrowthMeadow),
		fmt.Sprintf("Nachwachs Wüste: %.4f", cfg.RegrowthDesert),
		fmt.Sprintf("Max-Population:  %d", cfg.MaxPopulation),
	}
	for _, line := range lines {
		ebitenutil.DebugPrintAt(screen, line, tx, ty)
		ty += lineH
	}
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
