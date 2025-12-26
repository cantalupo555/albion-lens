package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cantalupo555/albion-lens/pkg/photon"
)

// EventMsg represents a game event to display in the log
type EventMsg struct {
	Type      string      // "fame", "silver", "loot", "combat", "info"
	Message   string      // Formatted message to display
	Timestamp time.Time   // When the event occurred
	Data      interface{} // Optional structured data (FameEventData, SilverEventData, etc.)
}

// BulkEventMsg represents a batch of game events
type BulkEventMsg []EventMsg

// StatsUpdateMsg triggers a stats panel update
type StatsUpdateMsg struct {
	Stats *photon.Stats
}

// OnlineMsg updates the online status
type OnlineMsg struct {
	Online bool
}

// TickMsg is sent periodically to update the UI
type TickMsg time.Time

// SessionStatsMsg updates session-specific stats (fame, silver, etc.)
type SessionStatsMsg struct {
	Fame   int64
	Silver int64
	Kills  int
	Deaths int
	Loot   int
}

// Commands

// TickCmd returns a command that sends a TickMsg after 1 second
func TickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// WaitForEvent returns a command that waits for an event from the channel
func WaitForEvent(ch <-chan EventMsg) tea.Cmd {
	return func() tea.Msg {
		return <-ch
	}
}

// WaitForBulkEvent returns a command that waits for a batch of events from the channel
func WaitForBulkEvent(ch <-chan BulkEventMsg) tea.Cmd {
	return func() tea.Msg {
		return <-ch
	}
}

// WaitForStats returns a command that waits for stats update from the channel
func WaitForStats(ch <-chan *photon.Stats) tea.Cmd {
	return func() tea.Msg {
		stats := <-ch
		return StatsUpdateMsg{Stats: stats}
	}
}
