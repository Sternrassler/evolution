package entity

import "image"

// EntityType klassifiziert ein Individuum als Herbivore oder Predator.
type EntityType uint8

const (
	Herbivore EntityType = iota // 0 — Standardwert, bestehender Code unberührt
	Predator                    // 1
)

// Individual repräsentiert ein Individuum in der Simulation (AoS-Format für öffentliche API).
type Individual struct {
	ID         uint64
	Pos        image.Point
	Energy     float32
	Age        int
	Genes      [NumGenes]float32
	EntityType EntityType
	alive      bool
}

// NewIndividual erstellt ein neues lebendiges Individuum.
func NewIndividual(id uint64, pos image.Point, genes [NumGenes]float32, energy float32) Individual {
	return Individual{ID: id, Pos: pos, Genes: genes, Energy: energy, alive: true}
}

// IsAlive gibt zurück, ob das Individuum lebt.
func (ind *Individual) IsAlive() bool { return ind.alive }

// Kill markiert das Individuum als tot.
func (ind *Individual) Kill() { ind.alive = false }
