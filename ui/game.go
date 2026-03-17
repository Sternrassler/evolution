package ui

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/Sternrassler/evolution/config"
	"github.com/Sternrassler/evolution/render"
	"github.com/Sternrassler/evolution/sim"
)

// TileSize ist die Pixel-Größe einer Tile im Fenster.
const TileSize = 4

// Game implementiert das ebiten.Game-Interface.
type Game struct {
	simulation *sim.Simulation
	exporter   *sim.SnapshotExporter
	renderer   *render.Renderer
	hud        *HUD
	input      *InputHandler
	lastTick   uint64
	cfg        config.Config
}

// NewGame erstellt ein neues Game.
func NewGame(simulation *sim.Simulation, exporter *sim.SnapshotExporter, renderer *render.Renderer, cfg config.Config) *Game {
	return &Game{
		simulation: simulation,
		exporter:   exporter,
		renderer:   renderer,
		hud:        NewHUD(),
		input:      &InputHandler{},
		cfg:        cfg,
	}
}

// Update wird von Ebiten einmal pro Frame aufgerufen (60 FPS).
// Führt genau einen Sim-Tick aus (per ADR-008: synchrones Step() in Update()).
func (g *Game) Update() error {
	if err := g.input.Process(g); err != nil {
		return err
	}
	if !g.input.Paused {
		g.simulation.Step()
	} else if g.input.StepOnce {
		g.simulation.Step()
		g.input.StepOnce = false
	}
	return nil
}

// Draw wird von Ebiten einmal pro Frame aufgerufen.
func (g *Game) Draw(screen *ebiten.Image) {
	snap := g.exporter.Load()
	if snap != nil && snap.Tick != g.lastTick {
		g.renderer.RenderToBuffer(snap)
		g.lastTick = snap.Tick
	}
	g.renderer.DrawBuffer(screen)
	if snap != nil {
		g.hud.Draw(screen, snap)
	}
}

// Layout gibt die logische Fenstergröße zurück.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return g.renderer.ScreenSize()
}
