package world

import (
	"image"

	"github.com/Sternrassler/evolution/sim/entity"
)

// SpatialGrid: Räumliche Datenstruktur für schnelle Nachbarschaftssuche.
// CellSize-basierte Bucket-Map. Pre-allokiert, O(n) Rebuild pro Tick.
type SpatialGrid struct {
	buckets  [][]int32 // Entity-SoA-Indizes pro Zelle
	cellSize int
	cols     int // Anzahl Spalten (worldWidth / cellSize, aufgerundet)
	rows     int // Anzahl Zeilen (worldHeight / cellSize, aufgerundet)
}

// NewSpatialGrid erstellt ein SpatialGrid. Parameter statt *config.Config.
func NewSpatialGrid(cellSize, worldWidth, worldHeight int) *SpatialGrid {
	cols := (worldWidth + cellSize - 1) / cellSize
	rows := (worldHeight + cellSize - 1) / cellSize
	buckets := make([][]int32, cols*rows)
	for i := range buckets {
		buckets[i] = make([]int32, 0, 8) // typische Dichte
	}
	return &SpatialGrid{
		buckets:  buckets,
		cellSize: cellSize,
		cols:     cols,
		rows:     rows,
	}
}

// Rebuild baut den Grid vollständig neu — O(n), einmal pro Tick.
// Alle bisherigen Bucket-Inhalte werden geleert (len=0, cap bleibt).
func (sg *SpatialGrid) Rebuild(individuals []entity.Individual) {
	// Buckets leeren (kein alloc)
	for i := range sg.buckets {
		sg.buckets[i] = sg.buckets[i][:0]
	}
	for i, ind := range individuals {
		if !ind.IsAlive() {
			continue
		}
		bIdx := sg.bucketIdx(ind.Pos.X, ind.Pos.Y)
		if bIdx < 0 {
			continue
		}
		sg.buckets[bIdx] = append(sg.buckets[bIdx], int32(i))
	}
}

// IndividualsNear gibt SoA-Indizes aller Individuen im Radius zurück.
// out wird als Ergebnis-Buffer wiederverwendet (zero-alloc wenn cap ausreicht).
// Gibt nil zurück wenn keine Individuen gefunden.
func (sg *SpatialGrid) IndividualsNear(p image.Point, radius int, out []int32) []int32 {
	out = out[:0]
	minCellX := (p.X - radius) / sg.cellSize
	maxCellX := (p.X + radius) / sg.cellSize
	minCellY := (p.Y - radius) / sg.cellSize
	maxCellY := (p.Y + radius) / sg.cellSize

	// Clamp auf Grid-Grenzen
	if minCellX < 0 {
		minCellX = 0
	}
	if maxCellX >= sg.cols {
		maxCellX = sg.cols - 1
	}
	if minCellY < 0 {
		minCellY = 0
	}
	if maxCellY >= sg.rows {
		maxCellY = sg.rows - 1
	}

	for cy := minCellY; cy <= maxCellY; cy++ {
		for cx := minCellX; cx <= maxCellX; cx++ {
			bucket := sg.buckets[cy*sg.cols+cx]
			for _, idx := range bucket {
				// Für MVP: Alle in den Zellen innerhalb des Bounding-Box zurückgeben.
				// Exakte Distanzprüfung übernimmt der Agent in Phase 1.
				out = append(out, idx)
			}
		}
	}
	return out
}

func (sg *SpatialGrid) bucketIdx(x, y int) int {
	cx := x / sg.cellSize
	cy := y / sg.cellSize
	if cx < 0 || cx >= sg.cols || cy < 0 || cy >= sg.rows {
		return -1
	}
	return cy*sg.cols + cx
}
