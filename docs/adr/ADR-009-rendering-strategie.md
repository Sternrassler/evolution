# ADR-009: Rendering-Strategie — Pixel-Buffer + Zoom-abhängige Darstellung

- **Datum:** 2026-03-17
- **Status:** Accepted

---

## Kontext

Die Simulation produziert pro Tick einen `WorldSnapshot` mit bis zu 200×200 Tiles
und bis zu `MaxPopulation` Individuen. Das Rendering muss:

1. **Tile-Schicht:** Biom-Farbe + Nahrungsstand pro Tile (40 000 Tiles)
2. **Individuen-Schicht:** Farbige Punkte oder Sprites je nach Zoom
3. **60 FPS halten** — `RenderToBuffer()` muss im Sub-Millisekunden-Bereich bleiben
4. **Zoom-abhängig:** Nah = Sprite, Mittel = farbiger Punkt, Weit = Dichte-Heatmap
5. **Kein GC-Druck** im Hot-Path (Allokations-Budget aus ARCHITECTURE.md)

Ebiten bietet mehrere Rendering-Ansätze:
- `screen.WritePixels([]byte)` — direkter RGBA-Byte-Buffer
- `screen.DrawImage(*ebiten.Image)` — GPU-beschleunigtes Blit
- Shader (`ebiten.NewShader`) — GPU-seitige Berechnung
- Kombinationen

---

## Entscheidung

**Primär: Pre-allokierter RGBA-Byte-Buffer + `screen.WritePixels()`.**
**Zoom-In (Nah): `DrawImage` mit Sub-Image-Sprites aus Atlas.**

### Tile-Schicht

```go
type Renderer struct {
    buf      []byte           // RGBA, len = WorldWidth * WorldHeight * TileSize² * 4
    tileSize int              // Pixel pro Tile (Default: 4)
}

func (r *Renderer) RenderToBuffer(snap *WorldSnapshot) {
    for _, tile := range snap.Tiles {
        color := biomeColor(tile)  // O(1) Lookup
        r.fillTile(i, color)       // Schreibt TileSize² Pixel in r.buf
    }
    // Individuen-Schicht darüber
    for _, ind := range snap.Individuals {
        r.drawIndividual(ind)
    }
}

func (r *Renderer) DrawBuffer(screen *ebiten.Image) {
    screen.WritePixels(r.buf)
}
```

`r.buf` wird einmalig mit `cap = WorldWidth * WorldHeight * TileSize² * 4` allokiert.
Kein `make()` im Hot-Path — Ziel: `RenderToBuffer()` 0 allocs.

### Zoom-abhängige Darstellung

| Zoom-Stufe | Kriterium | Individuum | Umsetzung |
|---|---|---|---|
| Nah | TileSize ≥ 16 px | Sprite aus Atlas | `DrawImage` mit Sub-Image |
| Mittel | TileSize 4–15 px | Farbiger Punkt (Genotyp-Farbe) | `r.buf`-Pixel direkt schreiben |
| Weit | TileSize < 4 px | Dichte-Heatmap | Zähler pro Tile, Farbgradient |

Zoom ändert `TileSize`. `RenderToBuffer()` wählt den Darstellungspfad anhand
von `r.tileSize`. Kein polymorphes Interface — `switch` auf `tileSize`-Range.

### Sprite-Atlas (Zoom-Nah)

Ein einzelnes `*ebiten.Image` (Atlas) enthält alle Sprites. Sub-Images über
`Atlas.SubImage(rect)` — kein Alloc bei bekannten Rects, pre-berechnet bei
`tileSize`-Änderung.

```go
type Renderer struct {
    // ...
    atlas      *ebiten.Image
    spriteRect [NumSpriteTypes]image.Rectangle  // pre-berechnet
}
```

### Kein separater Render-Goroutine

`RenderToBuffer()` und `DrawBuffer()` werden ausschließlich aus `Game.Draw()`
aufgerufen — derselben Goroutine wie `Update()` (Ebiten-Modell).
Kein Channel, kein Mutex für Renderer-State.

Der Dirty-Flag-Mechanismus (ADR-003) stellt sicher, dass `RenderToBuffer()`
nur bei neuem `WorldSnapshot.Tick` aufgerufen wird (~20×/s statt 60×/s).

---

## Konsequenzen

**Positiv:**
- `WritePixels` ist der schnellste Weg, um viele Pixel zu schreiben — direkter
  Transfer in Ebiten-internen Textur-Buffer, keine Zwischenschritte
- Pre-allokierter Buffer: kein GC-Druck, konstanter Speicher
- Einfaches Debugging: `r.buf` ist ein normales Go-Slice, inspizierbar in Tests
- `RenderToBuffer()` ist pure (liest nur Snapshot, schreibt nur `r.buf`) — testbar
  ohne Ebiten

**Negativ:**
- `WritePixels` überträgt immer den vollständigen Buffer — kein Partial-Update.
  Bei 200×200×4×4 = 640 KB pro Frame. Bei 60 FPS: ~38 MB/s GPU-Transfer.
  Messbar, aber für dedizierte GPUs vernachlässigbar. Gegenmittel falls nötig:
  Dirty-Region-Tracking (Optimierungspfad, nicht MVP)
- Sprites (Zoom-Nah) kombinieren `WritePixels` (Hintergrund) + `DrawImage` (Sprites)
  — zwei Rendering-Pässe pro Frame. Einfacher als Pure-Pixel-Sprite-Rendering
  und ausreichend schnell
- Atlas muss bei Zoom-Änderung neu berechnet werden (Sub-Image-Rects) —
  O(NumSpriteTypes), einmaliger Aufruf, kein Hot-Path

**Invariante:**
- `r.buf` hat immer `len = r.tileSize² × WorldWidth × WorldHeight × 4`.
  Bei `tileSize`-Änderung: neuer Buffer allokiert (einmaliger Alloc) und
  Dirty-Flag gesetzt für vollständiges Neuzeichnen.
- `RenderToBuffer()` darf kein `*ebiten.Image` erstellen oder verändern —
  ausschließlich `r.buf` schreiben. Sprite-Draws (`DrawImage`) nur in `DrawBuffer()`.

---

## Verworfene Alternativen

### A: Ausschließlich `DrawImage` (kein WritePixels)

Jede Tile-Farbe als `ebiten.Image` oder eingefärbte Rechtecke via `vector`-Package.
Problem: 40 000 `DrawImage`-Calls pro Frame — Ebiten-Overhead pro Call ist messbar.
`WritePixels` mit vorab berechnetem Buffer ist für homogene Flächen (Tiles) schneller.
Abgelehnt für die Tile-Schicht.

### B: GLSL-Shader für Tile-Rendering

Tile-Daten als Textur hochladen, Shader wandelt Biom-ID → Farbe um.
Maximum GPU-Effizienz, aber: Shader-Entwicklung für MVP unverhältnismäßig.
Ebiten-Shader-API ist nicht trivial. Als Optimierungspfad dokumentiert
(Stufe 3: Umweltbedingungen mit saisonalen Farben könnten Shader rechtfertigen).

### C: Separater Render-Goroutine

Render-Goroutine wartet auf neuen Snapshot per Channel, rendert in Hintergrundpuffer,
signalisiert Fertigstellung an `Draw()`.
Problem: Ebiten's `DrawImage` und `WritePixels` sind **nicht** threadsafe — müssen
aus der Ebiten-Goroutine aufgerufen werden. Eine Render-Goroutine könnte nur
die CPU-seitige Puffer-Berechnung übernehmen (nicht den `screen`-Zugriff).
Mehrheit des Render-Aufwands liegt im `WritePixels`-Transfer — Auslagerung
des Rests bringt kaum Gewinn. Abgelehnt.

### D: `ebiten.Image.Set()` pro Pixel

Pixel einzeln via `Set(x, y, color)` schreiben. Nachweislich langsam —
jeder Call hat Overhead. `WritePixels` ist der dokumentierte schnelle Pfad
für Massen-Pixel-Updates. Abgelehnt.
