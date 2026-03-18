package partition

import (
	"image"

	"github.com/Sternrassler/evolution/sim/entity"
)

// GhostRow enthält eine kopierte Grenzzeile einer Nachbarpartition (read-only in Phase 1).
type GhostRow struct {
	X      []int32
	Y      []int32
	Energy []float32
	Genes  [][entity.NumGenes]float32
}

// Partition hält alle SoA-Arrays für eine horizontale Welt-Partition.
// Alle Arrays sind einmalig pre-allokiert (cap = MaxPopulation).
type Partition struct {
	// SoA-Hot-Arrays — row-major, cache-freundlich
	X          []int32
	Y          []int32
	Energy     []float32
	Age        []int32
	Alive      []bool
	Genes      [][entity.NumGenes]float32
	IDs        []uint64
	EntityType []entity.EntityType

	// Management
	FreeList []int32           // Indizes toter Slots zur Wiederverwendung
	Buf      entity.EventBuffer // pre-allokiert, ein Buffer pro Partition
	Len      int               // Anzahl belegter Slots (inkl. freier)

	// Ghost-Rows (read-only für Phase 1, gefüllt vom Koordinator vor Phase 1)
	GhostTop    []GhostRow
	GhostBottom []GhostRow

	// Partition-Grenzen (Y-Zeilen in der Gesamtwelt, exklusiv EndRow)
	StartRow int
	EndRow   int
}

// NewPartition allokiert alle Arrays einmalig mit cap=maxPop.
func NewPartition(maxPop, startRow, endRow int) *Partition {
	cap := maxPop
	buf := entity.NewEventBuffer(cap * 4) // pro Individuum max ~4 Events
	return &Partition{
		X:          make([]int32, 0, cap),
		Y:          make([]int32, 0, cap),
		Energy:     make([]float32, 0, cap),
		Age:        make([]int32, 0, cap),
		Alive:      make([]bool, 0, cap),
		Genes:      make([][entity.NumGenes]float32, 0, cap),
		IDs:        make([]uint64, 0, cap),
		EntityType: make([]entity.EntityType, 0, cap),
		FreeList:   make([]int32, 0, cap/4),
		Buf:        buf,
		StartRow:   startRow,
		EndRow:     endRow,
	}
}

// AddIndividual fügt ein Individuum hinzu. Reused einen freien Slot (FreeList) wenn möglich.
// Gibt den SoA-Index zurück.
func (p *Partition) AddIndividual(ind entity.Individual) int32 {
	if len(p.FreeList) > 0 {
		idx := p.FreeList[len(p.FreeList)-1]
		p.FreeList = p.FreeList[:len(p.FreeList)-1]
		p.X[idx] = int32(ind.Pos.X)
		p.Y[idx] = int32(ind.Pos.Y)
		p.Energy[idx] = ind.Energy
		p.Age[idx] = int32(ind.Age)
		p.Alive[idx] = true
		p.Genes[idx] = ind.Genes
		p.IDs[idx] = ind.ID
		p.EntityType[idx] = ind.EntityType
		return idx
	}
	idx := int32(p.Len)
	p.X = append(p.X, int32(ind.Pos.X))
	p.Y = append(p.Y, int32(ind.Pos.Y))
	p.Energy = append(p.Energy, ind.Energy)
	p.Age = append(p.Age, int32(ind.Age))
	p.Alive = append(p.Alive, true)
	p.Genes = append(p.Genes, ind.Genes)
	p.IDs = append(p.IDs, ind.ID)
	p.EntityType = append(p.EntityType, ind.EntityType)
	p.Len++
	return idx
}

// MarkDead markiert Slot i als tot und fügt ihn in die FreeList ein.
func (p *Partition) MarkDead(i int32) {
	p.Alive[i] = false
	p.FreeList = append(p.FreeList, i)
}

// LiveCount gibt die Anzahl lebender Individuen zurück.
func (p *Partition) LiveCount() int {
	count := 0
	for i := range p.Len {
		if p.Alive[i] {
			count++
		}
	}
	return count
}

// ToIndividuals konvertiert SoA → AoS für den Snapshot-Export.
// Gibt nur lebende Individuen zurück. alive=true via entity.NewIndividual.
func (p *Partition) ToIndividuals() []entity.Individual {
	result := make([]entity.Individual, 0, p.LiveCount())
	for i := range p.Len {
		if !p.Alive[i] {
			continue
		}
		ind := entity.NewIndividual(
			p.IDs[i],
			image.Pt(int(p.X[i]), int(p.Y[i])),
			p.Genes[i],
			p.Energy[i],
		)
		ind.EntityType = p.EntityType[i]
		result = append(result, ind)
	}
	return result
}
