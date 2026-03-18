package sim

import (
	"encoding/binary"
	"hash/fnv"
	"math"
	"sync/atomic"

	"github.com/Sternrassler/evolution/sim/entity"
	"github.com/Sternrassler/evolution/sim/world"
)

// TickStats enthält aggregierte Statistiken eines Ticks.
type TickStats struct {
	Population        int
	Births            int
	Deaths            int
	EnergyConsumed    float32
	EnergyLostToDeath float32
	EnergyRegrown     float32
	FoodTiles   int // Tiles mit Food > 0
	DesertTiles int // Wüsten-Tiles nach Verwüstung/Erholung
	LandTiles   int // Nicht-Wasser-Tiles gesamt (konstant)
}

// WorldSnapshot ist ein immutabler Zustand der Welt nach einem Tick.
// Kein Mutex — nach atomic.Store() nicht mehr mutieren.
type WorldSnapshot struct {
	Tiles       []world.Tile
	Individuals []entity.Individual
	Tick        uint64
	Stats       TickStats
}

// Hash berechnet einen deterministischen FNV-1a-Hash über alle Snapshot-Felder.
// Geordnete Iteration — keine Maps, 100% deterministisch.
func (s *WorldSnapshot) Hash() uint64 {
	h := fnv.New64a()
	var buf [8]byte
	// Tick
	binary.LittleEndian.PutUint64(buf[:], s.Tick)
	h.Write(buf[:])
	// Tiles
	for _, t := range s.Tiles {
		buf[0] = byte(t.Biome)
		h.Write(buf[:1])
		binary.LittleEndian.PutUint32(buf[:4], math.Float32bits(t.Food))
		h.Write(buf[:4])
	}
	// Individuals (sortiert nach ID für Determinismus — Annahme: bereits sortiert via buildSnapshot)
	for _, ind := range s.Individuals {
		binary.LittleEndian.PutUint64(buf[:], ind.ID)
		h.Write(buf[:])
		binary.LittleEndian.PutUint32(buf[:4], uint32(ind.Pos.X))
		h.Write(buf[:4])
		binary.LittleEndian.PutUint32(buf[:4], uint32(ind.Pos.Y))
		h.Write(buf[:4])
		binary.LittleEndian.PutUint32(buf[:4], math.Float32bits(ind.Energy))
		h.Write(buf[:4])
	}
	return h.Sum64()
}

// SnapshotExporter: 2-Buffer-Pool + atomic.Pointer — lock-frei, zero-alloc nach Init.
type SnapshotExporter struct {
	pool     [2]WorldSnapshot
	current  atomic.Pointer[WorldSnapshot]
	writeIdx int // nur von Step()-Goroutine beschrieben
}

// NewSnapshotExporter initialisiert die Buffer mit pre-allozierten Slices.
func NewSnapshotExporter(tileCount, maxPop int) *SnapshotExporter {
	e := &SnapshotExporter{}
	e.pool[0].Tiles = make([]world.Tile, tileCount)
	e.pool[0].Individuals = make([]entity.Individual, 0, maxPop)
	e.pool[1].Tiles = make([]world.Tile, tileCount)
	e.pool[1].Individuals = make([]entity.Individual, 0, maxPop)
	e.current.Store(&e.pool[0])
	return e
}

// store kopiert snap-Daten in den inaktiven Buffer und macht ihn aktiv.
// Nur von Step() aufgerufen (single-writer).
func (e *SnapshotExporter) store(snap WorldSnapshot) {
	next := &e.pool[1-e.writeIdx]
	next.Tick = snap.Tick
	next.Stats = snap.Stats
	// Tiles kopieren (len bleibt gleich)
	copy(next.Tiles, snap.Tiles)
	// Individuals kopieren
	next.Individuals = next.Individuals[:0]
	next.Individuals = append(next.Individuals, snap.Individuals...)
	e.current.Store(next)
	e.writeIdx = 1 - e.writeIdx
}

// Load gibt den aktuellen Snapshot zurück — lock-frei, für Draw()-Goroutine.
func (e *SnapshotExporter) Load() *WorldSnapshot {
	return e.current.Load()
}
