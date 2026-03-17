package testutil

import (
	"github.com/Sternrassler/evolution/config"
	"github.com/Sternrassler/evolution/sim"
	"github.com/Sternrassler/evolution/sim/entity"
	"github.com/Sternrassler/evolution/sim/partition"
)

// BuildPartition konvertiert []Individual (AoS) → *Partition (SoA).
// startRow/endRow: Partition-Grenzen (typisch 0 / cfg.WorldHeight für Tests).
func BuildPartition(individuals []entity.Individual, cfg config.Config, startRow, endRow int) *partition.Partition {
	p := partition.NewPartition(max(len(individuals)*2, 100), startRow, endRow)
	for _, ind := range individuals {
		p.AddIndividual(ind)
	}
	return p
}

// HashSnapshot berechnet einen deterministischen Hash über alle Snapshot-Felder.
// Delegiert an snap.Hash().
func HashSnapshot(snap *sim.WorldSnapshot) uint64 {
	if snap == nil {
		return 0
	}
	return snap.Hash()
}
