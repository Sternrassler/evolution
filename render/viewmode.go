package render

// ViewMode bestimmt welche Information auf der Karte dargestellt wird.
type ViewMode int

const (
	ViewBiom    ViewMode = iota + 1 // Geländetyp + Nahrungsfüllstand (Standard)
	ViewDichte                       // Populationsdichte pro Tile (Heatmap)
	ViewGenotyp                      // Durchschnittsgene aller Individuen pro Tile als RGB
	ViewNahrung                      // Nahrungsfüllstand biomunabhängig
)

// ViewName gibt den deutschen Anzeigenamen einer Sicht zurück.
func (v ViewMode) ViewName() string {
	switch v {
	case ViewBiom:
		return "Biom"
	case ViewDichte:
		return "Dichte"
	case ViewGenotyp:
		return "Genotyp"
	case ViewNahrung:
		return "Nahrung"
	default:
		return "?"
	}
}
