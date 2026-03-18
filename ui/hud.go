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

const (
	maxHistory = 400
	sidebarPad = 6
	lineH      = 14
	swatchSize = 10
)

// historyBuffer ist ein Ringpuffer für drei Zeitreihen.
type historyBuffer struct {
	pop   [maxHistory]float32
	food  [maxHistory]float32
	dfood [maxHistory]float32
	n     int
}

func (h *historyBuffer) push(pop, food, dfood float32) {
	i := h.n % maxHistory
	h.pop[i] = pop
	h.food[i] = food
	h.dfood[i] = dfood
	h.n++
}

func (h *historyBuffer) count() int {
	if h.n < maxHistory {
		return h.n
	}
	return maxHistory
}

func (h *historyBuffer) at(i int) (pop, food, dfood float32) {
	count := h.count()
	idx := (h.n - count + i) % maxHistory
	return h.pop[idx], h.food[idx], h.dfood[idx]
}

// HUD verwaltet die Seitenleiste mit Stats, Legende, Parametern und Verlaufsdiagramm.
type HUD struct {
	mapW int
	hist historyBuffer
}

func NewHUD(mapW int) *HUD { return &HUD{mapW: mapW} }

// Draw zeichnet die gesamte Seitenleiste.
func (h *HUD) Draw(screen *ebiten.Image, snap *sim.WorldSnapshot, cfg config.Config) {
	if snap == nil {
		return
	}

	h.hist.push(
		float32(snap.Stats.Population),
		snap.Stats.TotalFood,
		snap.Stats.DesertFood,
	)

	sx := float32(h.mapW)
	sh := float32(screen.Bounds().Dy())

	// Seitenleisten-Hintergrund
	vector.FillRect(screen, sx, 0, SidebarWidth, sh, color.RGBA{20, 20, 20, 255}, false)
	// Trennlinie zur Karte
	vector.FillRect(screen, sx, 0, 1, sh, color.RGBA{80, 80, 80, 255}, false)

	tx := h.mapW + sidebarPad
	ty := sidebarPad

	ty = h.drawStats(screen, tx, ty, snap)
	ty = drawSep(screen, h.mapW, ty)
	ty = drawLegendSection(screen, tx, ty)
	ty = drawSep(screen, h.mapW, ty)
	ty = drawParamsSection(screen, tx, ty, cfg)
	ty = drawSep(screen, h.mapW, ty)
	h.drawChart(screen, tx, ty)
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
	biomes := []struct {
		label string
		c     color.RGBA
	}{
		{"Wiese", color.RGBA{40, 160, 30, 255}},
		{"Wüste", color.RGBA{200, 180, 100, 255}},
		{"Wasser", color.RGBA{30, 80, 160, 255}},
	}
	for _, b := range biomes {
		vector.FillRect(screen, float32(tx), float32(ty), swatchSize, swatchSize, b.c, false)
		ebitenutil.DebugPrintAt(screen, b.label, tx+swatchSize+4, ty)
		ty += lineH
	}
	ty += 4
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
		vector.FillRect(screen, float32(tx), float32(ty), swatchSize, swatchSize, g.c, false)
		ebitenutil.DebugPrintAt(screen, g.label, tx+swatchSize+4, ty)
		ty += lineH
	}
	return ty
}

func drawParamsSection(screen *ebiten.Image, tx, ty int, cfg config.Config) int {
	ebitenutil.DebugPrintAt(screen, "Parameter:", tx, ty)
	ty += lineH
	lines := []string{
		fmt.Sprintf("Energie-Kosten: %.2f", cfg.BaseEnergyCost),
		fmt.Sprintf("Repro-Schwelle: %.0f E", cfg.ReproductionThreshold),
		fmt.Sprintf("Repro-Reserve:  %.0f E", cfg.ReproductionReserve),
		fmt.Sprintf("Nachwachs Wiese:%.4f", cfg.RegrowthMeadow),
		fmt.Sprintf("Nachwachs Wüste:%.4f", cfg.RegrowthDesert),
		fmt.Sprintf("Max-Population: %d", cfg.MaxPopulation),
	}
	for _, l := range lines {
		ebitenutil.DebugPrintAt(screen, l, tx, ty)
		ty += lineH
	}
	return ty
}

// drawSep zeichnet eine horizontale Trennlinie und gibt das neue ty zurück.
func drawSep(screen *ebiten.Image, mapW, ty int) int {
	ty += 3
	vector.FillRect(screen,
		float32(mapW+sidebarPad), float32(ty),
		float32(SidebarWidth-2*sidebarPad), 1,
		color.RGBA{70, 70, 70, 255}, false,
	)
	ty += 4
	return ty
}

// drawChart zeichnet das Verlaufsdiagramm ab ty.
func (h *HUD) drawChart(screen *ebiten.Image, tx, ty int) {
	ebitenutil.DebugPrintAt(screen, "Verlauf:", tx, ty)
	ty += lineH

	const (
		chartW = SidebarWidth - 2*sidebarPad
		chartH = 220
	)

	cx := float32(tx)
	cy := float32(ty)
	vector.FillRect(screen, cx, cy, chartW, chartH, color.RGBA{10, 10, 10, 255}, false)

	count := h.hist.count()
	if count < 2 {
		return
	}

	// Sichtbare Punkte: maximal chartW
	pts := count
	if pts > chartW {
		pts = int(chartW)
	}
	startIdx := count - pts

	// Maxima für Normalisierung
	var maxPop, maxFood, maxDFood float32 = 1, 1, 1
	for i := range pts {
		p, f, d := h.hist.at(startIdx + i)
		if p > maxPop {
			maxPop = p
		}
		if f > maxFood {
			maxFood = f
		}
		if d > maxDFood {
			maxDFood = d
		}
	}

	yOf := func(val, maxVal float32) float32 {
		if maxVal <= 0 {
			return cy + chartH - 1
		}
		return cy + float32(chartH) - 1 - (val/maxVal)*float32(chartH-2)
	}

	xStep := float32(chartW) / float32(pts-1)

	for i := 1; i < pts; i++ {
		p0, f0, d0 := h.hist.at(startIdx + i - 1)
		p1, f1, d1 := h.hist.at(startIdx + i)
		x0 := cx + float32(i-1)*xStep
		x1 := cx + float32(i)*xStep

		vector.StrokeLine(screen, x0, yOf(p0, maxPop), x1, yOf(p1, maxPop), 1,
			color.RGBA{220, 200, 50, 255}, false) // Population: gelb
		vector.StrokeLine(screen, x0, yOf(f0, maxFood), x1, yOf(f1, maxFood), 1,
			color.RGBA{50, 200, 50, 255}, false) // Nahrung gesamt: grün
		vector.StrokeLine(screen, x0, yOf(d0, maxDFood), x1, yOf(d1, maxDFood), 1,
			color.RGBA{200, 130, 30, 255}, false) // Wüstennahrung: orange
	}

	ty += chartH + 4
	for _, e := range []struct {
		c color.RGBA
		l string
	}{
		{color.RGBA{220, 200, 50, 255}, "Population"},
		{color.RGBA{50, 200, 50, 255}, "Nahrung ges."},
		{color.RGBA{200, 130, 30, 255}, "Wüstennahrung"},
	} {
		vector.FillRect(screen, float32(tx), float32(ty), swatchSize, swatchSize, e.c, false)
		ebitenutil.DebugPrintAt(screen, e.l, tx+swatchSize+4, ty)
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
