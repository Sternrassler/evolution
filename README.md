# Evolution Simulation

Eine biologisch inspirierte Echtzeit-Simulation, die grundlegende Prinzipien von **Leben** und **Evolution** sichtbar macht. Aus zufälligen Anfangsbedingungen entstehen durch natürliche Selektion Anpassungen — beobachtbar in Echtzeit.

## Status

**Spezifikationsphase — 0% implementiert.**
Alle Architekturentscheidungen sind getroffen; die Implementierung beginnt mit M0.

## Konzept

Individuen leben auf einem 2D-Grid, verbrauchen Energie, suchen Nahrung und reproduzieren sich. Gene steuern Verhalten (Geschwindigkeit, Sichtradius, Stoffwechsel). Wer zu wenig Energie hat, stirbt. Wer sich reproduziert, gibt seine Gene weiter — mit zufälliger Mutation. Natürliche Selektion entsteht ohne explizite Regel.

## Technologie

| Komponente | Technologie |
|---|---|
| Sprache | Go |
| Rendering | [Ebiten](https://ebitengine.org/) |
| Parallelisierung | Partition-basierte Phase-1/Phase-2-Architektur |
| Tests | `pgregory.net/rapid` (Property-Tests), Race-Detector |

## Architektur

```
cmd/evolution
  └── ui ──────────────── render
        └── sim ──────── sim/partition ── sim/entity
              └── sim/world ──────────── sim/entity
gen ──── sim/world
config ─ (keine Projekt-Imports)
```

Details: [`ARCHITECTURE.md`](ARCHITECTURE.md) · [`CONCEPT.md`](CONCEPT.md) · [`ROADMAP.md`](ROADMAP.md) · [`docs/adr/`](docs/adr/)

## Meilensteine

| Meilenstein | Inhalt |
|---|---|
| M0 | CI-Gates, Repo-Struktur |
| M1–M4 | entity, config, world, gen |
| M5–M7 | testutil, partition, sim |
| M8–M10 | render, ui, cmd — **MVP** |
| M11–M14 | Räuber, Umwelt, Editor, Details |

## Lizenz

[MIT](LICENSE)
