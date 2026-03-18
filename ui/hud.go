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

// SidebarWidth ist die Breite der rechten Seitenleiste in Pixeln.
const SidebarWidth = 200

// ChartHeight ist die Höhe des Verlaufsdiagramms unterhalb der Karte.
const ChartHeight = 160

const (
	maxHistory = 400
	sidebarPad = 6
	lineH      = 14
	swatchSize = 10
)

// historyBuffer ist ein Ringpuffer für drei Zeitreihen.
type historyBuffer struct {
	pop    [maxHistory]float32
	food   [maxHistory]float32
	desert [maxHistory]float32
	n      int
}

func (h *historyBuffer) push(pop, food, desert float32) {
	i := h.n % maxHistory
	h.pop[i] = pop
	h.food[i] = food
	h.desert[i] = desert
	h.n++
}

func (h *historyBuffer) count() int {
	if h.n < maxHistory {
		return h.n
	}
	return maxHistory
}

func (h *historyBuffer) at(i int) (pop, food, desert float32) {
	count := h.count()
	idx := (h.n - count + i) % maxHistory
	return h.pop[idx], h.food[idx], h.desert[idx]
}

// HUD verwaltet Seitenleiste und Verlaufsdiagramm.
type HUD struct {
	mapW, mapH int
	hist       historyBuffer
}

func NewHUD(mapW, mapH int) *HUD { return &HUD{mapW: mapW, mapH: mapH} }

// Draw zeichnet Seitenleiste und Diagramm.
func (h *HUD) Draw(screen *ebiten.Image, snap *sim.WorldSnapshot, cfg config.Config) {
	if snap == nil {
		return
	}

	land := snap.Stats.LandTiles
	if land == 0 {
		land = 1
	}
	maxPop := cfg.MaxPopulation
	if maxPop == 0 {
		maxPop = 1
	}
	h.hist.push(
		float32(snap.Stats.Population)/float32(maxPop)*100,
		snap.Stats.AvgFoodPct,
		float32(snap.Stats.DesertTiles)/float32(land)*100,
	)

	h.drawSidebar(screen, snap, cfg)
	h.drawBottomChart(screen)
}

func (h *HUD) drawSidebar(screen *ebiten.Image, snap *sim.WorldSnapshot, cfg config.Config) {
	sx := float32(h.mapW)
	sh := float32(screen.Bounds().Dy())

	vector.FillRect(screen, sx, 0, SidebarWidth, sh, color.RGBA{20, 20, 20, 255}, false)
	vector.FillRect(screen, sx, 0, 1, sh, color.RGBA{80, 80, 80, 255}, false)

	tx := h.mapW + sidebarPad
	ty := sidebarPad

	ty = h.drawStats(screen, tx, ty, snap)
	ty = drawSep(screen, h.mapW, ty)
	ty = drawLegendSection(screen, tx, ty)
	ty = drawSep(screen, h.mapW, ty)
	drawParamsSection(screen, tx, ty, cfg)
}

func (h *HUD) drawStats(screen *ebiten.Image, tx, ty int, snap *sim.WorldSnapshot) int {
	avgSpeed, avgSight, avgEff := avgGenes(snap.Individuals)
	lines := []string{
		"Statistik:",
		fmt.Sprintf("Tick:  %d", snap.Tick),
		fmt.Sprintf("Pop:   %d", snap.Stats.Population),
		fmt.Sprintf("Geb: %d  Tod: %d", snap.Stats.Births, snap.Stats.Deaths),
		fmt.Sprintf("ØSpd:%.2f ØSgt:%.2f", avgSpeed, avgSight),
		fmt.Sprintf("ØEff:%.2f", avgEff),
	}
	for _, l := range lines {
		ebitenutil.DebugPrintAt(screen, l, tx, ty)
		ty += lineH
	}
	return ty
}

func drawLegendSection(screen *ebiten.Image, tx, ty int) int {
	ebitenutil.DebugPrintAt(screen, "Gelände:", tx, ty)
	ty += lineH
	for _, b := range []struct {
		label string
		c     color.RGBA
	}{
		{"Wiese", color.RGBA{40, 160, 30, 255}},
		{"Wüste", color.RGBA{200, 180, 100, 255}},
		{"Wasser", color.RGBA{30, 80, 160, 255}},
	} {
		vector.FillRect(screen, float32(tx), float32(ty), swatchSize, swatchSize, b.c, false)
		ebitenutil.DebugPrintAt(screen, b.label, tx+swatchSize+4, ty)
		ty += lineH
	}
	ty += 4
	ebitenutil.DebugPrintAt(screen, "Individuen (RGB=Gen):", tx, ty)
	ty += lineH
	for _, g := range []struct {
		label string
		c     color.RGBA
	}{
		{"Rot   = Speed", color.RGBA{220, 60, 60, 255}},
		{"Grün  = Sight", color.RGBA{60, 220, 60, 255}},
		{"Blau  = Effiz.", color.RGBA{60, 60, 220, 255}},
	} {
		vector.FillRect(screen, float32(tx), float32(ty), swatchSize, swatchSize, g.c, false)
		ebitenutil.DebugPrintAt(screen, g.label, tx+swatchSize+4, ty)
		ty += lineH
	}
	return ty
}

func drawParamsSection(screen *ebiten.Image, tx, ty int, cfg config.Config) int {
	ebitenutil.DebugPrintAt(screen, "Parameter:", tx, ty)
	ty += lineH
	for _, l := range []string{
		fmt.Sprintf("Energie-Kosten: %.2f", cfg.BaseEnergyCost),
		fmt.Sprintf("Repro-Schwelle: %.0f E", cfg.ReproductionThreshold),
		fmt.Sprintf("Repro-Reserve:  %.0f E", cfg.ReproductionReserve),
		fmt.Sprintf("Nachwachs Wiese:%.4f", cfg.RegrowthMeadow),
		fmt.Sprintf("Nachwachs Wüste:%.4f", cfg.RegrowthDesert),
		fmt.Sprintf("Verwüstung ab:  %.0f%%", cfg.DesertifyThreshold*100),
		fmt.Sprintf("Erholung ab:    %.0f%%", cfg.RecoverThreshold*100),
		fmt.Sprintf("Max-Population: %d", cfg.MaxPopulation),
	} {
		ebitenutil.DebugPrintAt(screen, l, tx, ty)
		ty += lineH
	}
	return ty
}

func drawSep(screen *ebiten.Image, mapW, ty int) int {
	ty += 3
	vector.FillRect(screen,
		float32(mapW+sidebarPad), float32(ty),
		float32(SidebarWidth-2*sidebarPad), 1,
		color.RGBA{70, 70, 70, 255}, false,
	)
	return ty + 4
}

// drawBottomChart zeichnet das Verlaufsdiagramm unterhalb der Karte.
func (h *HUD) drawBottomChart(screen *ebiten.Image) {
	cx := float32(0)
	cy := float32(h.mapH)
	cw := float32(h.mapW)
	ch := float32(ChartHeight)

	// Hintergrund
	vector.FillRect(screen, cx, cy, cw, ch, color.RGBA{15, 15, 15, 255}, false)
	// Trennlinie zur Karte
	vector.FillRect(screen, cx, cy, cw, 1, color.RGBA{80, 80, 80, 255}, false)

	const legendH = 3*lineH + 4
	const pad = 4
	plotH := ch - float32(legendH) - float32(lineH) - float32(2*pad) // Platz für Header + Legende

	// Header
	ebitenutil.DebugPrintAt(screen, "Verlauf in % (Pop/MaxPop · Nahrung/Land · Wüste/Land):", pad, h.mapH+pad)

	plotY := cy + float32(lineH) + float32(pad)

	// Plot-Hintergrund
	vector.FillRect(screen, cx+float32(pad), plotY, cw-float32(2*pad), plotH, color.RGBA{8, 8, 8, 255}, false)

	count := h.hist.count()
	if count < 2 {
		h.drawChartLegend(screen, pad, int(plotY+plotH)+pad)
		return
	}

	pts := count
	plotWi := int(cw) - 2*pad
	if pts > plotWi {
		pts = plotWi
	}
	startIdx := count - pts

	// Feste 0–100%-Achse
	yOf := func(pct float32) float32 {
		return plotY + plotH - 1 - (pct/100)*(plotH-2)
	}

	// Gitternetz bei 25%, 50%, 75%
	for _, pct := range []float32{25, 50, 75} {
		gy := yOf(pct)
		vector.StrokeLine(screen, cx+float32(pad), gy, cx+cw-float32(pad), gy, 1,
			color.RGBA{45, 45, 45, 255}, false)
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%d%%", int(pct)), int(cx)+1, int(gy)-lineH/2)
	}

	xStep := (cw - float32(2*pad)) / float32(pts-1)

	for i := 1; i < pts; i++ {
		p0, f0, d0 := h.hist.at(startIdx + i - 1)
		p1, f1, d1 := h.hist.at(startIdx + i)
		x0 := cx + float32(pad) + float32(i-1)*xStep
		x1 := cx + float32(pad) + float32(i)*xStep

		vector.StrokeLine(screen, x0, yOf(p0), x1, yOf(p1), 1,
			color.RGBA{220, 200, 50, 255}, false)
		vector.StrokeLine(screen, x0, yOf(f0), x1, yOf(f1), 1,
			color.RGBA{50, 200, 50, 255}, false)
		vector.StrokeLine(screen, x0, yOf(d0), x1, yOf(d1), 1,
			color.RGBA{200, 130, 30, 255}, false)
	}

	h.drawChartLegend(screen, pad, int(plotY+plotH)+pad)
}

func (h *HUD) drawChartLegend(screen *ebiten.Image, pad, ty int) {
	for _, e := range []struct {
		c color.RGBA
		l string
	}{
		{color.RGBA{220, 200, 50, 255}, "Population (% von Max)"},
		{color.RGBA{50, 200, 50, 255}, "Nahrung (Ø Füllstand %)"},
		{color.RGBA{200, 130, 30, 255}, "Wüste (% der Land-Tiles)"},
	} {
		vector.FillRect(screen, float32(pad), float32(ty), swatchSize, swatchSize, e.c, false)
		ebitenutil.DebugPrintAt(screen, e.l, pad+swatchSize+4, ty)
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
