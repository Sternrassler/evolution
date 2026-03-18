# Regelkreise — Evolution Simulation

Dieses Dokument beschreibt alle implementierten Regelkreise fachlich und mathematisch,
so wie sie im Code tatsächlich umgesetzt sind. Parameternamen verweisen direkt auf `config.Config`.

---

## 1. Energiekreislauf (pro Individuum, pro Tick)

### Verbrauch

Jedes lebende Individuum zahlt pro Tick einen Energiebetrag, der von seinem Speed-Gen abhängt:

```
Kosten = BaseEnergyCost + GeneSpeed × 0.1
```

Standardwerte: `BaseEnergyCost = 0.5`, `GeneSpeed ∈ [0.5, 3.0]`
→ Kosten liegen zwischen **0.55** und **0.80** pro Tick.

### Nahrungsaufnahme

Steht das Individuum auf einer Tile mit Nahrung, nimmt es bis zu 50 % des vorhandenen
Nahrungsvorrats, maximal 2.0 Einheiten:

```
gegessen  = min(Tile.Food × 0.5,  2.0)
Energiegewinn = gegessen × GeneEfficiency
```

`GeneEfficiency ∈ [0.5, 2.0]` → Gewinn zwischen **0** (kein Essen) und **4.0** pro Tick.

### Netto-Energiebilanz

```
ΔE = Energiegewinn − Kosten
```

| Szenario | ΔE (Richtwert) |
|---|---|
| Frisst voll (Eff=2.0, Spd=0.5) | +3.45 |
| Frisst voll (Eff=1.0, Spd=1.5) | +1.85 |
| Frisst nicht | −0.55 bis −0.80 |

### Tod

Fällt die Energie auf ≤ 0, stirbt das Individuum:

```
wenn E(t+1) ≤ 0  →  Tod
```

Ohne Nahrung überlebt ein Individuum (Startenergie `ReproductionThreshold × 0.5 = 50`):

```
Überlebensticks ≈ 50 / Kosten  ≈  63–91 Ticks  (≈ 3–5 Sekunden bei 20 TPS)
```

### Reproduktion

Überschreitet die Energie die Schwelle, erzeugt das Individuum ein Kind:

```
wenn E ≥ ReproductionThreshold (100):
    E_Elter  -= ReproductionReserve (50)
    E_Kind    = ReproductionReserve (50)
    Gene_Kind = mutate(Gene_Elter)
```

Zeit bis zur ersten Reproduktion ab Startenergie 50 (kontinuierliches Fressen, ΔE = +2):

```
Ticks bis Repro ≈ (ReproductionThreshold − E_start) / ΔE_netto = 50 / 2 = 25 Ticks
```

**Rückkopplung:** Hohe Effizienz → schnellere Reproduktion → mehr Individuen →
mehr Fraß → weniger Nahrung → Selektionsdruck auf Effizienz steigt.

---

## 2. Nahrungskreislauf (pro Tile, pro Tick)

### Nachwuchs (Regrowth)

Jede nicht-Wasser-Tile wächst pro Tick um einen Anteil ihres Maximums nach:

```
ΔFood = RegrowthRate × FoodMax
Food(t+1) = min(Food(t) + ΔFood,  FoodMax)
```

| Biom | RegrowthRate | FoodMax | ΔFood/Tick |
|---|---|---|---|
| Wiese | `RegrowthMeadow = 0.002` | 10 | 0.02 |
| Wüste | `RegrowthDesert = 0.0005` | 10* | 0.005 |

*FoodMax bleibt beim Biomwechsel erhalten (siehe Verwüstungskreislauf).

Vollständige Erholung einer leeren Wiese:

```
Ticks bis FoodMax = 1 / RegrowthMeadow = 500 Ticks  (25 Sekunden)
```

### Fraß (durch Population)

Pro Tick entnimmt jedes Individuum auf der Tile:

```
gegessen = min(Food × 0.5,  2.0)
Food(t+1) = Food(t) − Σ gegessen  (alle Individuen auf dieser Tile)
```

### Gleichgewichtsbedingung

Eine Tile ist im Gleichgewicht wenn Nachwuchs = Fraß:

```
RegrowthMeadow × FoodMax = n × min(Food × 0.5,  2.0)
```

Bei vollem Bestand (`Food = FoodMax = 10`) und `n` Individuen:

```
0.002 × 10 = n × 2.0
n_max = 0.1  →  weniger als 1 Individuum pro Tile für Gleichgewicht
```

Das bedeutet: **Eine Wiese kann dauerhaft genau ~0.1 Individuen ernähren.**
Bei höherer Dichte verarmt die Tile und desertifiziert (→ Regelkreis 3).

---

## 3. Verwüstungskreislauf

### Desertifizierung

Eine Wiese wird zur Wüste wenn ihr Füllstand unter die Schwelle fällt:

```
wenn Biom == Wiese  UND  Food/FoodMax < DesertifyThreshold (0.05):
    Biom → Wüste
```

### Erholung

Eine Wüste erholt sich zur Wiese wenn genug Nahrung nachgewachsen ist:

```
wenn Biom == Wüste  UND  Food/FoodMax > RecoverThreshold (0.50):
    Biom → Wiese
```

### Hysterese

Die Schwellen sind asymmetrisch — das verhindert schnelles Hin- und Herwechseln:

```
Desertifizierung bei  Food < 0.05 × FoodMax  =  0.5
Erholung         bei  Food > 0.50 × FoodMax  =  5.0
```

Eine desertifizierte Tile muss von ~0.5 auf 5.0 Nahrung anwachsen (bei Wüsten-Rate):

```
Ticks bis Erholung ≈ (5.0 − 0.5) / (RegrowthDesert × FoodMax) = 4.5 / 0.005 = 900 Ticks  (45 s)
```

**Rückkopplung (negativ — stabilisierend):**
Hohe Population → starker Fraß → Tiles verarmen → Desertifizierung →
weniger Nahrung verfügbar → Population sinkt → weniger Fraß → Erholung.

---

## 4. Räuber-Beute-Kreislauf (Lotka-Volterra)

### Populationsdynamik

Räuber jagen Herbivoren. Die daraus entstehende Schwingung folgt qualitativ den Lotka-Volterra-Gleichungen:

```
dH/dt = αH − βHP      (Herbivoren wachsen, sterben durch Räuber)
dP/dt = δHP − γP      (Räuber wachsen durch Jagd, sterben ohne Beute)
```

Im Simulations-Modell abgebildet durch:
- Herbivore verlieren Energie bei `EventAttack` → Tod wenn E ≤ 0
- Räuber gewinnen `EnergyPerKill` pro erfolgreichem Kill
- Räuber reproduzieren sich bei E ≥ `ReproThreshold`

### Startbedingung

```
InitialPredators = 10  (~2 % von InitialPop = 500)
```

**Herleitung aus der Energiepyramide:**
In realen Ökosystemen liegt das Räuber-Beute-Verhältnis bei 1:10 bis 1:100
(Energieeffizienz ~10 % pro Trophiestufe). Für Savannensysteme typisch ~1:50.
2 % entsprechen dem unteren Ende dieses Bereichs — bewusst konservativ,
damit die Räuber-Population nicht sofort kollabiert, bevor Lotka-Volterra-Schwingungen entstehen können.

### Kill-Wahrscheinlichkeit

```
P(Kill) = GeneAggression  ∈ [0, 1]
```

Ein Jagdversuch gelingt nur mit Wahrscheinlichkeit `GeneAggression`. Bei Misserfolg
(oder keiner Beute in Sichtweite) führt der Räuber einen Random Walk aus.

Effektive Kill-Rate pro Tick und Räuber:
```
β_eff = GeneAggression × (Suchfläche / Weltfläche)
      ≈ 0.5 × (400 / 40.000) = 0.005 Kills/Tick  (bei Durchschnitts-Aggression)
```

**Evolutionsdruck:** Räuber mit zu niedriger Aggression verhungern. Räuber mit zu hoher
Aggression overhunten die Beute und kollabieren anschließend selbst. Evolution pendelt
sich auf eine mittlere Aggression ein.

### Energie-Transfer

```
EnergyPerKill = 8.0
```

Geringer Energiegewinn pro Kill — Räuber brauchen viele Kills um zu überleben und
sich zu reproduzieren. Dies dämpft den Populationsboom.

### Reproduktionsschwelle

```
ReproThreshold = 360.0  (ReproReserve = 60.0 → ReproEnergy = 300)
```

Kills bis zur Reproduktion ab Startenergie (Durchschnitt, β_eff = 0.005):
```
ReproEnergy / EnergyPerKill = 300 / 8 ≈ 38 erfolgreiche Kills
Ticks bis Repro ≈ 38 / 0.5 = 76 Ticks  (bei avg. Aggression und vollem Beuteangebot)
```

### Gleichgewicht (Lotka-Volterra)

```
H* = γ / (δ × β_eff)
P* = α / β_eff

γ = Sterberate ohne Beute  = 1.06/60  ≈ 0.018
δ = Repro-Effizienz        = 8/300    ≈ 0.027   (EnergyPerKill / ReproEnergy)
β_eff = eff. Kill-Rate     = 0.5×0.01 = 0.005   (avg. Aggression × Suchfläche)
α = Herbivoren-Wachstum    ≈ 0.04/Tick

H* = 0.018 / (0.027 × 0.005) ≈ 133 Herbivoren
P* = 0.04  / 0.005           ≈ 8   Räuber
```

Die Startzustände (H=500, P=10) liegen nahe am Gleichgewicht → gedämpfte L-V-Schwingungen.

### Rückkopplung (negativ — stabilisierend)

```
Viele Räuber → viele Kills → wenige Herbivoren →
Räuber verhungern → wenige Räuber → Herbivoren erholen sich → …
```

**GeneAggression-Evolution:**
Räuber mit zu niedriger Aggression finden keine Beute und verhungern.
Räuber mit zu hoher Aggression overhunten die Beute und sterben nachfolgend.
Selektion pendelt GeneAggression auf einen stabilen Mittelwert ein.

---

## 5. Wechselwirkungen und Gesamtsystem

```
  ┌──────────────────────────────────────┐     ┌──────────────────────────────┐
  │        HERBIVOREN-POPULATION         │     │      RÄUBER-POPULATION       │
  │  (+) Reproduktion wenn E ≥ 100       │     │  (+) Reproduktion (3 Kills)  │
  │  (−) Tod: Hunger (E ≤ 0)            │◀────│  (−) Tod: kein Fraß          │
  │  (−) Tod: Angriff (EventAttack)      │────▶│  (+) Energie pro Kill (+40)  │
  └──────────┬───────────────▲───────────┘     └──────────────────────────────┘
             │ Fraß          │ Energie
             ▼               │
  ┌──────────────────┐  ┌──────────────┐
  │    NAHRUNG       │  │   ENERGIE    │
  │ (+) Nachwuchs    │─▶│ (+) Fraß     │
  │ (−) Fraß         │  │ (−) Kosten   │
  └──────────┬───────┘  └──────────────┘
             │ Verarmung
             ▼
  ┌──────────────────┐
  │   VERWÜSTUNG     │
  │ (+) bei Food<5%  │
  │ (−) bei Food>50% │
  └──────────┬───────┘
             │ senkt Nachwuchsrate (×0.25)
             └───────────────────────────▶ NAHRUNG (−)
```

### Stabilitätsbedingung (Herbivoren)

Das Herbivoren-System ist stabil wenn:

```
Fraß_gesamt ≤ Nachwuchs_gesamt
N × 2.0 ≤ LandTiles × RegrowthMeadow × FoodMax
N ≤ LandTiles × 0.002 × 10 / 2.0  =  LandTiles × 0.01
```

Bei ~32.000 Land-Tiles: **N_stabil ≤ 320 Herbivoren** (rein auf Wiesennachwuchs).
Räuber-Druck senkt das effektive Gleichgewicht zusätzlich.

### Evolutionsdruck

| Gen | Selektionsvorteil | Selektionsnachteil | Verstärkt durch |
|---|---|---|---|
| GeneSpeed (hoch) | Erreicht Nahrung schneller; flieht schneller | Höhere Energiekosten | Räuber-Druck |
| GeneSight (hoch) | Findet Nahrung/Räuber in größerem Radius | Kein direkter Nachteil | Räuber-Druck |
| GeneEfficiency (hoch) | Mehr Energie pro Bissen | Kein direkter Nachteil | Nahrungsknappheit |
| GeneAggression (hoch) | Herbivore fliehen effektiver (EventFlee) | Kein direkter Nachteil | Räuber-Druck |

Bei Nahrungsknappheit dominiert **GeneEfficiency**, unter Räuber-Druck steigt **GeneAggression**.

---

## 6. Parametertabelle

| Parameter | Wert | Regelkreis | Wirkung |
|---|---|---|---|
| `BaseEnergyCost` | 0.5 | Energie | Basisverbrauch pro Tick |
| `ReproductionThreshold` | 100 | Energie | Energie für Reproduktion |
| `ReproductionReserve` | 50 | Energie | Startenergie des Kindes |
| `RegrowthMeadow` | 0.002 | Nahrung | Nachwuchsrate Wiese |
| `RegrowthDesert` | 0.0005 | Nahrung | Nachwuchsrate Wüste |
| `DesertifyThreshold` | 0.05 | Verwüstung | Untergrenze Füllstand → Wüste |
| `RecoverThreshold` | 0.50 | Verwüstung | Untergrenze Füllstand → Wiese |
| `MaxPopulation` | 10.000 | Alle | Hartes Populationslimit |
| `Predator.InitialPredators` | 10 | Räuber-Beute | Startzahl Räuber (~2% von InitialPop; Energiepyramide) |
| `Predator.EnergyPerKill` | 8.0 | Räuber-Beute | Energiegewinn pro erfolgreichem Kill |
| `Predator.ReproThreshold` | 360.0 | Räuber-Beute | Reproduktionsschwelle Räuber (~38 Kills bis Repro) |
| `Predator.ReproReserve` | 60.0 | Räuber-Beute | Startenergie Räuber-Kind |
| `GeneAggression` | [0, 1] | Räuber-Beute | Kill-Wahrscheinlichkeit pro Jagdversuch (avg 0.5) |
