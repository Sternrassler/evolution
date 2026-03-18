package config

import (
	"fmt"
	"runtime"

	"github.com/Sternrassler/evolution/sim/entity"
)

// RandSource ist ein Type-Alias für entity.RandSource — für gen/ zugänglich via config-Import.
type RandSource = entity.RandSource

// Config enthält alle Simulations-Parameter. Werttyp (kein Pointer) für thread-sichere Kopien.
type Config struct {
	// Welt
	WorldWidth    int `toml:"world_width"`    // Default: 200
	WorldHeight   int `toml:"world_height"`   // Default: 200
	NumPartitions int `toml:"num_partitions"` // Default: runtime.GOMAXPROCS(0)

	// Simulation
	MaxPopulation  int  `toml:"max_population"`  // Default: 10000
	InitialPop     int  `toml:"initial_pop"`     // Default: 500
	TicksPerSecond int  `toml:"ticks_per_second"` // Default: 20
	DebugIntegrity bool `toml:"debug_integrity"` // Default: false

	// Spatial Grid
	SpatialCellSize int `toml:"spatial_cell_size"` // Default: MaxSightRange

	// Gen-Grenzen (für Ghost-Row-Berechnung)
	MaxSpeedRange int `toml:"max_speed_range"` // Default: 5
	MaxSightRange int `toml:"max_sight_range"` // Default: 10

	// Energie
	BaseEnergyCost        float32 `toml:"base_energy_cost"`
	ReproductionThreshold float32 `toml:"reproduction_threshold"`
	ReproductionReserve   float32 `toml:"reproduction_reserve"`

	// Nahrungsregrowth (Anteil von FoodMax pro Tick)
	RegrowthMeadow float32 `toml:"regrowth_meadow"` // Default: 0.002
	RegrowthDesert float32 `toml:"regrowth_desert"` // Default: 0.0005

	// Verwüstung / Erholung
	DesertifyThreshold float32 `toml:"desertify_threshold"` // Food/FoodMax < dieser Wert → Wiese wird Wüste
	RecoverThreshold   float32 `toml:"recover_threshold"`   // Food/FoodMax > dieser Wert → Wüste wird Wiese

	// Räuber
	Predator PredatorConfig `toml:"predator"`

	// Gen-Definitionen
	GeneDefinitions []GeneDef `toml:"gene_definitions"`
}

// PredatorConfig enthält alle Räuber-spezifischen Simulations-Parameter.
type PredatorConfig struct {
	InitialPredators int     `toml:"initial_predators"` // Default: 10 (~2% von InitialPop; Energiepyramide: 1 Räuber pro ~50 Beute)
	EnergyPerKill    float32 `toml:"energy_per_kill"`   // Default: 8.0 (Kills nötig bis Repro ≈ ReproEnergy/EnergyPerKill = 300/8 ≈ 38)
	ReproThreshold   float32 `toml:"repro_threshold"`   // Default: 360.0 (ReproEnergy=300 → ~38 erfolgreiche Kills bis Reproduktion)
	ReproReserve     float32 `toml:"repro_reserve"`     // Default: 60.0 (Startenergie des Kindes)
}

// GeneDef beschreibt ein Gen: Wertebereich und Mutationsparameter.
type GeneDef struct {
	Key          entity.GeneKey `toml:"key"`
	Min          float32        `toml:"min"`
	Max          float32        `toml:"max"`
	MutationRate float32        `toml:"mutation_rate"`
	MutationStep float32        `toml:"mutation_step"`
}

// DefaultConfig gibt eine vollständige, valide Standardkonfiguration zurück.
func DefaultConfig() Config {
	maxSight := 10
	maxSpeed := 5
	worldHeight := 200
	k := max(maxSpeed, maxSight)
	// NumPartitions: GOMAXPROCS, aber begrenzt so dass WorldHeight/NumPartitions >= 2*K
	maxPartitions := max(1, worldHeight/(2*k))
	nproc := min(runtime.GOMAXPROCS(0), maxPartitions)
	return Config{
		WorldWidth:    200,
		WorldHeight:   worldHeight,
		NumPartitions: nproc,

		MaxPopulation:  10000,
		InitialPop:     500,
		TicksPerSecond: 20,
		DebugIntegrity: false,

		SpatialCellSize: maxSight,
		MaxSpeedRange:   5,
		MaxSightRange:   maxSight,

		BaseEnergyCost:        0.5,
		ReproductionThreshold: 100.0,
		ReproductionReserve:   50.0,

		RegrowthMeadow: 0.002,
		RegrowthDesert: 0.0005,

		DesertifyThreshold: 0.05,
		RecoverThreshold:   0.50,

		Predator: PredatorConfig{
			InitialPredators: 10,
			EnergyPerKill:    8.0,
			ReproThreshold:   360.0,
			ReproReserve:     60.0,
		},

		GeneDefinitions: []GeneDef{
			{Key: entity.GeneSpeed, Min: 0.5, Max: 3.0, MutationRate: 0.1, MutationStep: 0.1},
			{Key: entity.GeneSight, Min: 1.0, Max: 10.0, MutationRate: 0.1, MutationStep: 0.5},
			{Key: entity.GeneEfficiency, Min: 0.5, Max: 2.0, MutationRate: 0.05, MutationStep: 0.05},
			{Key: entity.GeneAggression, Min: 0.0, Max: 1.0, MutationRate: 0.05, MutationStep: 0.05},
		},
	}
}

// GhostK berechnet K = max(MaxSpeedRange, MaxSightRange).
// K ist die Anzahl der Ghost-Zeilen an Partitionsgrenzen.
func (c *Config) GhostK() int {
	return max(c.MaxSpeedRange, c.MaxSightRange)
}

// Validate prüft die Konsistenz der Config.
// Gibt nil zurück wenn alles in Ordnung ist, sonst einen beschreibenden Fehler.
func (c *Config) Validate() error {
	if c.WorldWidth <= 0 {
		return fmt.Errorf("WorldWidth muss > 0 sein, ist %d", c.WorldWidth)
	}
	if c.WorldHeight <= 0 {
		return fmt.Errorf("WorldHeight muss > 0 sein, ist %d", c.WorldHeight)
	}
	if c.MaxPopulation <= 0 {
		return fmt.Errorf("MaxPopulation muss > 0 sein, ist %d", c.MaxPopulation)
	}
	if c.InitialPop < 0 {
		return fmt.Errorf("InitialPop darf nicht negativ sein, ist %d", c.InitialPop)
	}
	if c.InitialPop > c.MaxPopulation {
		return fmt.Errorf("InitialPop (%d) darf MaxPopulation (%d) nicht überschreiten", c.InitialPop, c.MaxPopulation)
	}
	if c.NumPartitions <= 0 {
		return fmt.Errorf("NumPartitions muss > 0 sein, ist %d", c.NumPartitions)
	}
	k := c.GhostK()
	minHeight := c.WorldHeight / c.NumPartitions
	if minHeight < 2*k {
		return fmt.Errorf(
			"Partitions zu viele: WorldHeight/NumPartitions=%d < 2*GhostK=%d (NumPartitions=%d, WorldHeight=%d, K=%d)",
			minHeight, 2*k, c.NumPartitions, c.WorldHeight, k,
		)
	}
	if c.SpatialCellSize <= 0 {
		return fmt.Errorf("SpatialCellSize muss > 0 sein, ist %d", c.SpatialCellSize)
	}
	if c.TicksPerSecond <= 0 {
		return fmt.Errorf("TicksPerSecond muss > 0 sein, ist %d", c.TicksPerSecond)
	}
	if c.Predator.InitialPredators < 0 {
		return fmt.Errorf("Predator.InitialPredators darf nicht negativ sein, ist %d", c.Predator.InitialPredators)
	}
	if c.Predator.EnergyPerKill <= 0 {
		return fmt.Errorf("Predator.EnergyPerKill muss > 0 sein, ist %f", c.Predator.EnergyPerKill)
	}
	if c.Predator.ReproThreshold <= c.Predator.ReproReserve {
		return fmt.Errorf("Predator.ReproThreshold (%f) muss > ReproReserve (%f) sein",
			c.Predator.ReproThreshold, c.Predator.ReproReserve)
	}
	if len(c.GeneDefinitions) != entity.NumGenes {
		return fmt.Errorf("GeneDefinitions muss genau %d Einträge haben, hat %d", entity.NumGenes, len(c.GeneDefinitions))
	}
	for i, gd := range c.GeneDefinitions {
		if gd.Min >= gd.Max {
			return fmt.Errorf("GeneDefinitions[%d] (Key=%d): Min (%f) muss < Max (%f) sein", i, gd.Key, gd.Min, gd.Max)
		}
		if gd.MutationRate < 0 || gd.MutationRate > 1 {
			return fmt.Errorf("GeneDefinitions[%d] (Key=%d): MutationRate (%f) muss in [0,1] liegen", i, gd.Key, gd.MutationRate)
		}
		if gd.MutationStep <= 0 {
			return fmt.Errorf("GeneDefinitions[%d] (Key=%d): MutationStep (%f) muss > 0 sein", i, gd.Key, gd.MutationStep)
		}
	}
	return nil
}
