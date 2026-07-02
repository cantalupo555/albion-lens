// Package backend provides a unified service layer for Albion Online packet capture and event processing.
// It serves as the backend for multiple frontends (TUI, Wails, Web API).
package backend

import "time"

// EventType represents the type of game event
type EventType string

const (
	EventTypeFame    EventType = "fame"
	EventTypeSilver  EventType = "silver"
	EventTypeLoot    EventType = "loot"
	EventTypeKill    EventType = "kill"
	EventTypeDeath   EventType = "death"
	EventTypeRespec  EventType = "respec"
	EventTypeInfo    EventType = "info"
	EventTypeWarning EventType = "warning"
	EventTypeZone    EventType = "zone"
)

// GameEvent represents a game event for display in frontends
type GameEvent struct {
	Type      EventType   // Type of event (fame, silver, loot, etc.)
	Message   string      // Formatted message to display
	Timestamp time.Time   // When the event occurred
	Data      interface{} // Optional structured data for specific event types
}

// FameData contains fame-specific event data
type FameData struct {
	Gained  int64 // Fame gained in this event
	Total   int64 // Total fame pool reported by the server after this event
	AllTime int64 // Cumulative fame gained across all launches (long-term total)
}

// SilverData contains silver-specific event data
type SilverData struct {
	Amount int64 // Silver amount in this event
	Total  int64 // Cumulative silver gained across all launches (long-term total)
}

// LootData contains loot-specific event data
type LootData struct {
	ItemName string // Name of the item
	ItemID   int32  // Item ID
	Quantity int32  // Number of items
	LootedBy string // Player who looted
	From     string // Source of loot
}

// CombatData contains combat-specific event data
type CombatData struct {
	KillerName string // Name of the killer
	VictimName string // Name of the victim
	Total      int    // Cumulative kills or deaths across all launches (long-term total)
}
