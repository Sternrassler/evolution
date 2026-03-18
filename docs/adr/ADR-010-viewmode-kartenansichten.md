# ADR-010: ViewMode — Umschaltbare Kartenansichten statt Zoom-Dispatch

- **Datum:** 2026-03-18
- **Status:** Accepted

---

## Kontext

Nach dem MVP (M10) zeigte sich ein praktisches Problem: Die Karte stellt
Biom-Farben und Gen-Farben der Individuen gleichzeitig dar. Bei der Standard-Tile-Größe
von 4 px sind einzelne Individuen auf der Gesamtansicht kaum erkennbar —
Gen-Farbpunkte liefern keinen nutzbaren Informationsgehalt.

Gleichzeitig gibt es mehrere orthogonale Informationen, die sich auf demselben
Pixel-Buffer nicht gleichzeitig sinnvoll darstellen lassen:

- **Biom + Nahrungsfüllstand** — Grundzustand der Welt
- **Populationsdichte** — wo halten sich wie viele Individuen auf?
- **Genotyp-Verteilung** — wie verteilen sich Genwerte über die Karte?
- **Nahrungsfüllstand** — biomunabhängig, um Verwüstung erkennbar zu machen

ADR-009 hatte zoom-basierten Dispatch vorgesehen (Nah/Mittel/Weit per `tileSize`),
der nie umgesetzt wurde, weil er die grundlegende Mehrdeutigkeit nicht löst:
auch bei großem Zoom bleibt unklar, was dargestellt werden soll.

---

## Entscheidung

**Vier umschaltbare `ViewMode`-Ansichten, gewählt per Tastendruck 1–4.**

```go
type ViewMode int

const (
    ViewBiom    ViewMode = iota + 1 // Standard
    ViewDichte
    ViewGenotyp
    ViewNahrung
)
```

`RenderToBuffer(snap *WorldSnapshot, mode ViewMode)` dispatcht per `switch` auf den Modus.
`ViewMode` liegt in `render/viewmode.go` **ohne** `//go:build`-Tag — kein Ebiten-Import,
headless-kompatibel, importierbar von `ui/` ohne Build-Tag-Probleme.

### Ansichten

| Modus | Taste | Was wird gezeigt |
|---|---|---|
| `ViewBiom` | 1 | Biom-Farbe + Nahrungsfüllstand je Tile; Individuen als Farbpunkte |
| `ViewDichte` | 2 | Populationsdichte pro Tile als Heatmap (schwarz → rot → orange → gelb) |
| `ViewGenotyp` | 3 | Ø Gene aller Individuen pro Tile als RGB (R=Speed, G=Sight, B=Effizienz); leere Tiles dunkelgrau |
| `ViewNahrung` | 4 | `Food/FoodMax` als Grauwert→Grün; Wasser bleibt blau |

### Pre-allokierte Hilfs-Buffer

Für `ViewDichte` und `ViewGenotyp` werden Zwischen-Buffer im `Renderer`-Struct
pre-allokiert, um den zero-alloc-Anspruch im Hot-Path zu erfüllen:

```go
type Renderer struct {
    // ...
    densityBuf   []int               // width*height
    geneSumBuf   []float32           // width*height*NumGenes
    geneCountBuf []int               // width*height
}
```

Alle Buffer werden in `NewRenderer()` einmalig allokiert. `RenderToBuffer()` bleibt 0 allocs.

### HUD-Integration

Die Seitenleiste zeigt den aktiven Modus hervorgehoben (grüner Hintergrund) und
passt die Legende kontextsensitiv an den Modus an.

---

## Konsequenzen

**Positiv:**
- Jede Ansicht ist für ihren Informationsgehalt optimiert — kein Kompromiss durch Überblendung
- Null-Allokation bleibt gewahrt durch pre-allokierte Hilfs-Buffer
- `ViewMode` ist headless-testbar (kein Ebiten-Dependency)
- Erweiterbar: neue Ansicht = neuer `case` + ggf. neuer Hilfs-Buffer

**Negativ:**
- Nutzer sieht immer nur eine Informationsebene gleichzeitig
- Pre-allokierte Hilfs-Buffer erhöhen Speicherbedarf des Renderers um
  `width × height × (sizeof(int) + NumGenes×sizeof(float32) + sizeof(int))`
  — bei 200×200 ca. 1 MB, vertretbar

**Kein Zoom-Dispatch (ADR-009):**
Der zoom-basierte Ansatz ist für die Tile-Größen 1–8 px weiterhin als
Optimierungspfad denkbar (z.B. Sprite-Darstellung bei tileSize ≥ 16),
schließt sich aber mit ViewMode nicht aus — beide Konzepte sind orthogonal.

---

## Verworfene Alternativen

### A: Overlay-Modus (halbtransparentes Überblenden)

Zwei Ebenen gleichzeitig rendern (z.B. Biom + Dichte als Transparenz-Overlay).
Erfordert Alpha-Blending im Pixel-Buffer oder zweiten Render-Pass.
Komplexer, kaum lesbarer als getrennte Ansichten. Abgelehnt.

### B: Zoom-basierter Auto-Switch

Bei kleinem Zoom automatisch Dichte-Heatmap, bei großem Zoom Biom. Intuitiv,
aber nimmt dem Nutzer die Kontrolle und deckt nicht den Genotyp-Fall ab. Abgelehnt.

### C: Mehrere Fenster / Split-Screen

Zu aufwändig für den aktuellen Entwicklungsstand. Als Stufe-5-Feature denkbar. Abgelehnt.
