# Evolution Simulation — Konzept & Anforderungen

## Ziel

Eine biologisch inspirierte Echtzeit-Simulation, die grundlegende Prinzipien von **Leben** und **Evolution** sichtbar macht. Der Nutzer beobachtet, wie aus zufälligen Anfangsbedingungen durch natürliche Selektion Anpassungen entstehen.

---

## Kernprinzipien

### Leben — was ein Individuum ausmacht

| Kriterium | Umsetzung |
|---|---|
| Stoffwechsel | Energie wird pro Tick verbraucht; Nahrung wird aufgenommen |
| Reproduktion | Bei ausreichend Energie: Nachkomme mit vererbten Genen |
| Reizreaktion | Nahrungssuche im Sichtradius, Flucht vor Räubern |
| Homöostase | Energielevel als innerer Zustand; Hunger treibt Verhalten |
| Tod | Energieverlust → Tod; maximales Alter |

### Evolution — was Populationen verändert

| Prinzip | Umsetzung |
|---|---|
| Variation | Gene unterscheiden sich zwischen Individuen |
| Vererbung | Kinder erben Gene der Eltern |
| Mutation | Gene ändern sich zufällig leicht bei Reproduktion |
| Selektion | Wer zu wenig Energie hat, stirbt; wer sich nicht fortpflanzt, gibt nichts weiter |

---

## Gene (MVP)

Jedes Individuum trägt einen **Genotyp** mit zunächst 3 Genen:

| Gen | Wertebereich | Effekt | Trade-off |
|---|---|---|---|
| `speed` | 0.5 – 5.0 | Mehr Tiles pro Tick | Höherer Energieverbrauch |
| `sight` | 1 – 10 | Sichtradius für Nahrung | Kostet nichts direkt, aber rechenintensiv |
| `efficiency` | 0.3 – 2.0 | Energieausbeute aus Nahrung | — |

**Visualisierung:** Farbe des Individuums kodiert den Genotyp (R=Speed, G=Sight, B=Efficiency). Populationsverschiebungen werden so als Farbveränderung sichtbar.

---

## Umgebung

### Karte

- **Tile-basiertes 2D-Gitter** (konfigurierbare Größe, Standard: 200×200)
- Prozedural generiert (zufällig + Glättung per Cellular Automaton)
- Jedes Tile hat: Biom-Typ, aktuellen Nahrungsvorrat, maximalen Nahrungsvorrat

### Biome (MVP: 3)

| Biom | Farbe | Nahrung | Begehbar |
|---|---|---|---|
| Wasser | Blau | keine | nein |
| Wiese | Grün | viel | ja |
| Wüste | Sandgelb | wenig | ja |

Nahrung wächst pro Tick nach (Rate abhängig vom Biom). Knappheit erzeugt Selektionsdruck.

---

## Simulation

### Tick-Ablauf (pro Individuum)

```
1. Alter +1
2. Energie abziehen (Basis + Speed-Kosten)
3. Energie ≤ 0 oder Alter > Max → Tod
4. Bewegen (in Richtung bester Nahrung im Sichtradius, sonst zufällig)
5. Nahrung fressen (aktuelles Tile)
6. Energie > Schwelle + Reserve → Fortpflanzung (Mutation)
```

### Parallelisierung

- Welt wird in Partitionen aufgeteilt → je eine Goroutine
- Individuen, die Partitionsgrenzen überschreiten, werden nach dem Tick synchronisiert

---

## Steuerung & UI

### Simulationssteuerung

| Element | Funktion |
|---|---|
| Geschwindigkeitsregler | 0–60 FPS einstellbar |
| "Next Step"-Button | Sichtbar bei Geschwindigkeit = 0, einzelne Ticks |
| Pause / Weiter | Simulation anhalten |

### Darstellung (zoomabhängig)

| Zoom | Darstellung Individuum | Darstellung Population |
|---|---|---|
| Nah | Symbol/Sprite mit Details | — |
| Mittel | Farbiger Punkt (Genotyp-Farbe) | — |
| Weit | — | Farbige Häufungspunkte |

### Statistik-Panel

- Aktuelle Population
- Tick-Zähler
- Ø Genwerte der Population (Trend sichtbar machen)
- Später: Graphen über Zeit

---

## Ausbaustufen (Roadmap)

### Stufe 1 — MVP
Einzelne Tierart, Nahrung als einziger Selektionsdruck, 3 Biome, Grunddarstellung

### Stufe 2 — Räuber & Beute
Zweite Tierart (Räuber), Flucht-/Jagdverhalten, neue Gene (z.B. `aggression`)

### Stufe 3 — Umweltbedingungen
Temperatur, Jahreszeiten, Katastrophen als zusätzlicher Druck

### Stufe 4 — Karten-Editor
Biome manuell platzieren, Startbedingungen konfigurieren

### Stufe 5 — Detailansicht
Zoom auf Individuum: Stammbaum, Genverlauf, Lebensgeschichte

---

## Technologie

| Komponente | Technologie | Begründung |
|---|---|---|
| Sprache | Go | Goroutinen, native Performance, einfaches Deployment |
| Rendering | Ebiten v2 | Einfache 2D-Engine, kein Browser, läuft nativ |
| Parallelismus | Goroutines + WaitGroup | Welt-Partitionen parallel berechnen |
| Darstellung | Pixel-Buffer (WritePixels) | Effizient für tile-basierte Welt |
