package entity

// GeneKey identifiziert ein Gen in [NumGenes]float32.
type GeneKey int

const (
	GeneSpeed      GeneKey = 0
	GeneSight      GeneKey = 1
	GeneEfficiency GeneKey = 2
	GeneAggression GeneKey = 3
	NumGenes               = 4 // für Stufe 3+ erhöhen → neuen case-Branch hinzufügen
)
