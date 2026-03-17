package sim

import (
	"github.com/Sternrassler/evolution/sim/entity"
)

// RandSource ist die Zufallsquelle für die Simulation.
// Identische Methoden wie entity.RandSource (strukturell kompatibel).
type RandSource interface {
	Float64() float64
	Intn(n int) int
}

// Agent ist das Interface für simulierbare Akteure (MVP: Individual, Stufe 2: Predator).
// Hinweis: entity.Individual implementiert Agent NICHT direkt — partition.agent ist der interne Wrapper.
// Dieses Interface dient als Dokumentation und für externe Erweiterungen.
type Agent interface {
	Tick(ctx any, out *entity.EventBuffer)
}
