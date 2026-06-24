package components

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/viewport"
	"charm.land/lipgloss/v2"
	"github.com/cantalupo555/albion-lens/pkg/handlers"
)

const defaultMaxZoneHistory = 500

// zoneEntry records a single zone transition in the history.
type zoneEntry struct {
	Zone      handlers.ZoneInfo
	Timestamp time.Time
}

// ZonePanel displays the current zone, time-in-zone, and a scrollable
// transition history. It follows the immutable builder pattern.
type ZonePanel struct {
	viewport      viewport.Model
	current       handlers.ZoneInfo
	enteredAt     time.Time
	nowFunc       func() time.Time
	history       []zoneEntry
	renderedLines []string // cached formatted lines for incremental rendering
	maxHistory    int
	width         int
	height        int
	ready         bool
}

// NewZonePanel creates a ZonePanel with sensible defaults.
func NewZonePanel() ZonePanel {
	return ZonePanel{
		nowFunc:    time.Now,
		maxHistory: defaultMaxZoneHistory,
	}
}

// SetSize updates the dimensions of the zone panel.
func (z ZonePanel) SetSize(width, height int) ZonePanel {
	z.width = width
	z.height = height

	headerHeight := 2 // title + border
	footerHeight := 1 // border
	infoHeight := 4   // current zone (2 lines) + elapsed (1 line) + blank (1 line)

	viewportHeight := height - headerHeight - footerHeight - infoHeight
	if viewportHeight < 1 {
		viewportHeight = 1
	}

	viewportWidth := width - 4 // borders + padding
	if viewportWidth < 10 {
		viewportWidth = 10
	}

	if !z.ready {
		z.viewport = viewport.New(viewport.WithWidth(viewportWidth), viewport.WithHeight(viewportHeight))
		z.ready = true
		z = z.refreshViewport()
	} else {
		z.viewport.SetWidth(viewportWidth)
		z.viewport.SetHeight(viewportHeight)
	}

	return z
}

// SetZone updates the current zone for display (used by the tick sync path).
// This does not change enteredAt or add to the history.
func (z ZonePanel) SetZone(info handlers.ZoneInfo) ZonePanel {
	z.current = info
	return z
}

// SetNow sets the reference time used for the elapsed-time display. Called by
// the tick path to keep the elapsed counter fresh; tests can inject a fixed
// time via the nowFunc field for deterministic assertions.
func (z ZonePanel) SetNow(t time.Time) ZonePanel {
	z.nowFunc = func() time.Time { return t }
	return z
}

// AddTransition appends a zone transition to the history, updates the current
// zone and its entry time, and trims entries beyond maxHistory. Only the new
// entry is formatted and appended to the rendered cache (incremental), avoiding
// a full history rebuild on each transition.
func (z ZonePanel) AddTransition(info handlers.ZoneInfo, ts time.Time) ZonePanel {
	z.current = info
	z.enteredAt = ts

	entry := zoneEntry{Zone: info, Timestamp: ts}
	z.history = append(z.history, entry)
	z.renderedLines = append(z.renderedLines, formatZoneEntry(entry))

	// Trim oldest entries beyond maxHistory (history and cache in sync).
	if len(z.history) > z.maxHistory {
		z.history = z.history[1:]
		z.renderedLines = z.renderedLines[1:]
	}

	return z.syncViewport()
}

// ScrollUp scrolls the history viewport up.
func (z ZonePanel) ScrollUp() ZonePanel {
	z.viewport.ScrollUp(1)
	return z
}

// ScrollDown scrolls the history viewport down.
func (z ZonePanel) ScrollDown() ZonePanel {
	z.viewport.ScrollDown(1)
	return z
}

// Clear empties the transition history.
func (z ZonePanel) Clear() ZonePanel {
	z.history = z.history[:0]
	z.renderedLines = z.renderedLines[:0]
	return z.syncViewport()
}

// syncViewport pushes the cached rendered lines into the viewport without
// rebuilding them. Used after incremental additions or clears.
func (z ZonePanel) syncViewport() ZonePanel {
	if !z.ready {
		return z
	}

	if len(z.renderedLines) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Italic(true)
		z.viewport.SetContent(emptyStyle.Render("No zone transitions yet..."))
		return z
	}

	z.viewport.SetContent(strings.Join(z.renderedLines, "\n"))
	z.viewport.GotoBottom()
	return z
}

// refreshViewport rebuilds the rendered cache from scratch from the full
// history. Used only during initial setup; incremental updates use
// syncViewport instead.
func (z ZonePanel) refreshViewport() ZonePanel {
	z.renderedLines = make([]string, 0, len(z.history))
	for _, entry := range z.history {
		z.renderedLines = append(z.renderedLines, formatZoneEntry(entry))
	}
	return z.syncViewport()
}

// formatZoneEntry renders a single history entry as a timestamped line.
func formatZoneEntry(entry zoneEntry) string {
	tsStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	label := entry.Zone.DisplayString()
	if label == "" {
		label = "Unknown"
	}
	return fmt.Sprintf("%s → %s",
		tsStyle.Render(entry.Timestamp.Format("15:04:05")),
		label,
	)
}

// formatElapsed renders a duration as a compact human-readable string.
func formatElapsed(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	switch {
	case h > 0:
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	case m > 0:
		return fmt.Sprintf("%dm %ds", m, s)
	default:
		return fmt.Sprintf("%ds", s)
	}
}

// View renders the zone panel.
func (z ZonePanel) View() string {
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Width(z.width - 2).
		Height(z.height - 2)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("62")).
		Padding(0, 1)

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	valueStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39"))

	// Current zone display
	zoneLabel := z.current.DisplayString()
	if zoneLabel == "" {
		zoneLabel = "Unknown"
	}

	// Elapsed time
	var elapsed string
	if z.enteredAt.IsZero() {
		elapsed = "—"
	} else {
		elapsed = formatElapsed(z.nowFunc().Sub(z.enteredAt))
	}

	zoneLine := fmt.Sprintf("%s  %s",
		labelStyle.Render("Current:"),
		valueStyle.Render(zoneLabel),
	)

	timeLine := fmt.Sprintf("%s  %s",
		labelStyle.Render("Time:"),
		valueStyle.Render(elapsed),
	)

	historyTitle := labelStyle.Render("Transition History:")

	content := lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render("Zone"),
		zoneLine,
		timeLine,
		"",
		historyTitle,
		z.viewport.View(),
	)

	return boxStyle.Render(content)
}
