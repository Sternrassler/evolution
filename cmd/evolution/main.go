package main

import (
	"log"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/Sternrassler/evolution/config"
	"github.com/Sternrassler/evolution/render"
	"github.com/Sternrassler/evolution/sim"
	"github.com/Sternrassler/evolution/ui"
)

type randSource struct{ r *rand.Rand }

func (rs *randSource) Float64() float64 { return rs.r.Float64() }
func (rs *randSource) Intn(n int) int   { return rs.r.Intn(n) }

func main() {
	cfg := config.DefaultConfig()
	if err := cfg.Validate(); err != nil {
		log.Fatal("Ungültige Config:", err)
	}

	rng := &randSource{r: rand.New(rand.NewSource(42))}

	simulation, exporter, err := sim.New(cfg, rng, nil)
	if err != nil {
		log.Fatal("Simulation-Initialisierung fehlgeschlagen:", err)
	}

	renderer := render.NewRenderer(cfg, ui.TileSize)
	game := ui.NewGame(simulation, exporter, renderer, cfg)

	w, h := renderer.ScreenSize()
	ebiten.SetWindowTitle("Evolution Simulation")
	ebiten.SetWindowSize(w+ui.SidebarWidth, h)
	ebiten.SetTPS(20) // 20 Ticks pro Sekunde

	if err := ebiten.RunGame(game); err != nil && err != ebiten.Termination {
		log.Fatal(err)
	}
}
