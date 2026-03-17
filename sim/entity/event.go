package entity

import "image"

// EventType klassifiziert Aktionen eines Agenten in Phase 1.
type EventType uint8

const (
	EventMove      EventType = iota // Bewegung zu TargetPos
	EventEat                        // Nahrungsaufnahme an TargetPos
	EventReproduce                  // Reproduktion an TargetPos
	EventDie                        // Individuum stirbt (Energie ≤ 0)
)

// Event beschreibt eine Aktion, die in Phase 2 angewendet wird.
type Event struct {
	Type      EventType
	AgentIdx  int32       // SoA-Index im Partition-Array
	TargetPos image.Point
	Value     float32     // Energie-Delta, Gen-Wert etc.
}

// EventBuffer sammelt Events eines Agenten pro Tick (pre-allokiert, zero-alloc im Hot-Path).
type EventBuffer struct {
	events []Event
}

// NewEventBuffer erstellt einen EventBuffer mit der angegebenen Kapazität.
func NewEventBuffer(capacity int) EventBuffer {
	return EventBuffer{events: make([]Event, 0, capacity)}
}

// Append fügt ein Event hinzu. Zero-alloc solange cap nicht überschritten.
func (b *EventBuffer) Append(e Event) { b.events = append(b.events, e) }

// Reset setzt den Buffer zurück ohne Allokation.
func (b *EventBuffer) Reset() { b.events = b.events[:0] }

// Len gibt die Anzahl der Events zurück.
func (b *EventBuffer) Len() int { return len(b.events) }

// Events gibt den Slice der Events zurück (read-only verwenden).
func (b *EventBuffer) Events() []Event { return b.events }
