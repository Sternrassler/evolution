package entity

// GeneKey identifiziert ein Gen in [NumGenes]float32.
type GeneKey int

const (
	GeneSpeed      GeneKey = 0
	GeneSight      GeneKey = 1
	GeneEfficiency GeneKey = 2
	NumGenes               = 3 // für Stufe 2 erhöhen → neuen case-Branch hinzufügen
)
