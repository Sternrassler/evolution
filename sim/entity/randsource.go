package entity

// RandSource ist die einzige erlaubte Qufalle für Zufallszahlen in sim/ und gen/.
// Kein direktes math/rand — immer über injizierte RandSource (CI Gate 2).
type RandSource interface {
	Float64() float64
	Intn(n int) int
}
