package sim

// TickObserver empfängt Statistiken nach jedem Tick.
// Verwendung: Test-Seam, HUD-Updates, Logging.
type TickObserver interface {
	OnTick(tick uint64, stats TickStats)
}

// NoopObserver verwirft alle Tick-Daten (Default).
type NoopObserver struct{}

func (NoopObserver) OnTick(uint64, TickStats) {}
