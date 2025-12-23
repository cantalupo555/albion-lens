package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

const maxEvents = 1000

// Event represents a single event in the log
type Event struct {
	Type      string
	Message   string
	Timestamp time.Time
}

// EventLog displays a scrollable list of game events
type EventLog struct {
	viewport viewport.Model
	events   []Event
	width    int
	height   int
	ready    bool
}

// NewEventLog creates a new EventLog component
func NewEventLog() EventLog {
	return EventLog{
		events: make([]Event, 0, maxEvents),
	}
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
		e.viewport.SetContent(e.renderEvents())
		e.ready = true
	} else {
		e.viewport.Width = viewportWidth
		e.viewport.Height = viewportHeight
	}

	return e
}

// AddEvent adds a new event to the log
func (e EventLog) AddEvent(eventType, message string, timestamp time.Time) EventLog {
	event := Event{
		Type:      eventType,
		Message:   message,
		Timestamp: timestamp,
	}

	e.events = append(e.events, event)

	// Trim old events if needed
	if len(e.events) > maxEvents {
		e.events = e.events[100:] // Remove oldest 100
	}

	// Update viewport content
	e.viewport.SetContent(e.renderEvents())
	e.viewport.GotoBottom()

	return e
}

// Clear removes all events from the log
func (e EventLog) Clear() EventLog {
	e.events = e.events[:0]
	e.viewport.SetContent("")
	return e
}

// renderEvents formats all events for display
func (e EventLog) renderEvents() string {
	if len(e.events) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Italic(true)
		return emptyStyle.Render("No events yet...")
	}

	var lines []string
	timestampStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	for _, event := range e.events {
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
		default:
			msgStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
		}

		line := fmt.Sprintf("%s %s",
			timestampStyle.Render(event.Timestamp.Format("15:04:05")),
			msgStyle.Render(event.Message),
		)
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
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
