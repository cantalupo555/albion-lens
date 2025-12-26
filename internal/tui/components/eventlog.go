package components

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/cantalupo555/albion-lens/pkg/events"
	"github.com/cantalupo555/albion-lens/pkg/handlers"
)

const maxEvents = 1000

// Event represents a single event in the log
type Event struct {
	Type      string
	Message   string
	Timestamp time.Time
	Data      interface{} // Raw event data for dynamic formatting
}

// EventLog displays a scrollable list of game events
type EventLog struct {
	viewport      viewport.Model
	events        []Event
	renderedLines []string // Cache of already formatted lines
	width         int
	height        int
	ready         bool
	fullNumbers   bool
}

// NewEventLog creates a new EventLog component
func NewEventLog() EventLog {
	return EventLog{
		events:        make([]Event, 0, maxEvents),
		renderedLines: make([]string, 0, maxEvents),
		fullNumbers:   true, // Default: show full numbers
	}
}

// SetFullNumbers sets whether to display full or abbreviated numbers
func (e EventLog) SetFullNumbers(full bool) EventLog {
	e.fullNumbers = full
	// Settings changed, must re-render everything
	e = e.reRenderAll()
	return e
}

// SetSize updates the dimensions of the event log
func (e EventLog) SetSize(width, height int) EventLog {
	e.width = width
	e.height = height

	headerHeight := 2 // title + border
	footerHeight := 1 // border

	viewportHeight := height - headerHeight - footerHeight
	if viewportHeight < 1 {
		viewportHeight = 1
	}

	viewportWidth := width - 4 // borders + padding
	if viewportWidth < 10 {
		viewportWidth = 10
	}

	if !e.ready {
		e.viewport = viewport.New(viewportWidth, viewportHeight)
		e.ready = true
		// Initial render
		e = e.reRenderAll()
	} else {
		e.viewport.Width = viewportWidth
		e.viewport.Height = viewportHeight
		e.viewport.SetContent(strings.Join(e.renderedLines, "\n"))
	}

	return e
}

// AddEvent adds a new event to the log
func (e EventLog) AddEvent(eventType, message string, timestamp time.Time, data interface{}) EventLog {
	event := Event{
		Type:      eventType,
		Message:   message,
		Timestamp: timestamp,
		Data:      data,
	}
	return e.AddEvents([]Event{event})
}

// AddEvents adds multiple events to the log efficiently
func (e EventLog) AddEvents(newEvents []Event) EventLog {
	if len(newEvents) == 0 {
		return e
	}

	e.events = append(e.events, newEvents...)

	// Trim old events if needed
	if len(e.events) > maxEvents {
		e.events = e.events[len(e.events)-maxEvents:] // Keep newest maxEvents
		
		// Trim the rendered cache to match
		if len(e.renderedLines) > len(e.events)-len(newEvents) {
			keepCount := maxEvents - len(newEvents)
			if keepCount > 0 && keepCount < len(e.renderedLines) {
				e.renderedLines = e.renderedLines[len(e.renderedLines)-keepCount:]
			} else if keepCount <= 0 {
				e.renderedLines = nil // Replaced all
			}
		}
	}

	// Format NEW events only
	for _, event := range newEvents {
		e.renderedLines = append(e.renderedLines, e.renderSingleEvent(event))
	}

	// Ensure cache size doesn't exceed maxEvents (sanity check)
	if len(e.renderedLines) > maxEvents {
		e.renderedLines = e.renderedLines[len(e.renderedLines)-maxEvents:]
	}

	// Update viewport content efficiently - ONCE per batch
	e.viewport.SetContent(strings.Join(e.renderedLines, "\n"))
	e.viewport.GotoBottom()

	return e
}

// Clear removes all events from the log
func (e EventLog) Clear() EventLog {
	e.events = e.events[:0]
	e.renderedLines = e.renderedLines[:0]
	e.viewport.SetContent("")
	return e
}

// reRenderAll clears cache and re-renders all events (used when settings change)
func (e EventLog) reRenderAll() EventLog {
	e.renderedLines = make([]string, 0, len(e.events))
	for _, event := range e.events {
		e.renderedLines = append(e.renderedLines, e.renderSingleEvent(event))
	}
	
	if e.ready {
		if len(e.renderedLines) == 0 {
			emptyStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("241")).
				Italic(true)
			e.viewport.SetContent(emptyStyle.Render("No events yet..."))
		} else {
			e.viewport.SetContent(strings.Join(e.renderedLines, "\n"))
		}
	}
	return e
}

// renderSingleEvent formats a single event struct into a colored string
func (e EventLog) renderSingleEvent(event Event) string {
	timestampStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	// Get color based on event type
	var msgStyle lipgloss.Style
	switch event.Type {
	case "fame":
		msgStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	case "silver":
		msgStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	case "loot":
		msgStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	case "combat", "kill", "death":
		msgStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	case "debug":
		msgStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	default:
		msgStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
	}

	// Format message dynamically based on event data and fullNumbers setting
	message := e.formatEventMessage(event)

	return fmt.Sprintf("%s %s",
		timestampStyle.Render(event.Timestamp.Format("15:04:05")),
		msgStyle.Render(message),
	)
}

// formatEventMessage formats event message based on data and fullNumbers setting
func (e EventLog) formatEventMessage(event Event) string {
	switch event.Type {
	case "fame":
		if data, ok := event.Data.(*handlers.FameEventData); ok && data != nil {
			return fmt.Sprintf("⭐ FAME: +%s | Total: %s | Session: %s",
				formatNumber(data.Gained, e.fullNumbers),
				formatNumber(data.Total, e.fullNumbers),
				formatNumber(data.Session, e.fullNumbers))
		}
	case "silver":
		if data, ok := event.Data.(*handlers.SilverEventData); ok && data != nil {
			return fmt.Sprintf("💰 %s looted silver (%s) from %s | Session: %s",
				data.LootedBy,
				formatNumber(data.Amount, e.fullNumbers),
				data.LootedFrom,
				formatNumber(data.Session, e.fullNumbers))
		}
	case "loot":
		if data, ok := event.Data.(*handlers.LootEventData); ok && data != nil {
			return fmt.Sprintf("📦 %s looted %s (x%d) from %s",
				data.LootedBy,
				data.ItemName,
				data.Quantity,
				data.LootedFrom)
		}
	case "kill":
		if data, ok := event.Data.(*handlers.KillEventData); ok && data != nil {
			return fmt.Sprintf("⚔️ Player Killed! (Session: %d kills)", data.SessionKills)
		}
	case "death":
		if data, ok := event.Data.(*handlers.DeathEventData); ok && data != nil {
			if data.Killer != "" {
				return fmt.Sprintf("💀 %s died! (Killed by %s)", data.Victim, data.Killer)
			}
			return fmt.Sprintf("💀 %s died!", data.Victim)
		}
	case "debug":
		if code, ok := event.Data.(events.EventCode); ok {
			return fmt.Sprintf("🔍 %v (%d)", code, code)
		}
	}
	// Fallback to original message
	return event.Message
}

// formatNumber formats a number based on fullNumbers setting
func formatNumber(amount int64, full bool) string {
	if full {
		return fmt.Sprintf("%d", amount)
	}
	// Abbreviated format with truncation (floor) instead of rounding
	if amount >= 1000000 {
		val := math.Floor(float64(amount)/100000.0) / 10.0
		return fmt.Sprintf("%.1fM", val)
	} else if amount >= 1000 {
		val := math.Floor(float64(amount)/100.0) / 10.0
		return fmt.Sprintf("%.1fk", val)
	}
	return fmt.Sprintf("%d", amount)
}

// ScrollUp scrolls the viewport up
func (e EventLog) ScrollUp() EventLog {
	e.viewport.LineUp(1)
	return e
}

// ScrollDown scrolls the viewport down
func (e EventLog) ScrollDown() EventLog {
	e.viewport.LineDown(1)
	return e
}

// View renders the event log
func (e EventLog) View() string {
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Width(e.width - 2).
		Height(e.height - 2)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("62")).
		Padding(0, 1)

	title := titleStyle.Render("Events")

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		e.viewport.View(),
	)

	return boxStyle.Render(content)
}
