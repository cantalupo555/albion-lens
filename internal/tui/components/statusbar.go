package components

import (
	"fmt"

	"github.com/cantalupo555/albion-lens/pkg/photon"
	"github.com/charmbracelet/lipgloss"
)

// StatusBar displays connection status, packet stats, and uptime
type StatusBar struct {
	online         bool
	playerName     string
	packetsTotal   uint64
	packetsPerSec  float64
	eventsDecoded  uint64
	eventsDropped  uint64
	bufferUsage    int
	bufferCapacity int
	uptime         string
	width          int
}

// NewStatusBar creates a new StatusBar component
func NewStatusBar() StatusBar {
	return StatusBar{
		uptime: "00:00:00",
	}
}

// SetWidth sets the width of the status bar
func (s StatusBar) SetWidth(width int) StatusBar {
	s.width = width
	return s
}

// SetOnline updates the online status
func (s StatusBar) SetOnline(online bool) StatusBar {
	s.online = online
	return s
}

// SetPlayerName updates the local player name. When empty and online, the
// status bar shows a warning prompting the user to change maps or relog.
func (s StatusBar) SetPlayerName(name string) StatusBar {
	s.playerName = name
	return s
}

// Online returns whether the status bar shows online status
func (s StatusBar) Online() bool {
	return s.online
}

// UpdateStats updates the stats display
func (s StatusBar) UpdateStats(stats *photon.Stats) StatusBar {
	if stats != nil {
		s.packetsTotal = stats.GetPacketsReceived()
		s.packetsPerSec = stats.PacketsPerSecond()
		s.eventsDecoded = stats.GetEventsDecoded()
		s.eventsDropped = stats.GetEventsDropped()
		s.bufferUsage = int(stats.BufferPeakDisplay)
		s.bufferCapacity = stats.BufferCapacity
		s.uptime = stats.FormatUptime()
	}
	return s
}

// View renders the status bar
func (s StatusBar) View() string {
	// Status indicator
	var status string
	if s.online {
		status = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")).
			Bold(true).
			Render("● Online")
	} else {
		status = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true).
			Render("● Offline")
	}

	// Player identification indicator
	if s.online {
		if s.playerName != "" {
			status += "  " + lipgloss.NewStyle().
				Foreground(lipgloss.Color("42")).
				Render("👤 "+s.playerName)
		} else {
			status += "  " + lipgloss.NewStyle().
				Foreground(lipgloss.Color("214")).
				Bold(true).
				Render("⚠ Player not identified — change map or relog")
		}
	}

	// Buffer Stats Logic
	var bufStatus string
	if s.bufferCapacity > 0 {
		pct := float64(s.bufferUsage) / float64(s.bufferCapacity) * 100
		bufColor := "42" // Green
		if pct >= 75 {
			bufColor = "196" // Red
		} else if pct >= 50 {
			bufColor = "214" // Yellow
		}

		bufStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(bufColor))
		bufStatus = fmt.Sprintf("│  Queue: %s", bufStyle.Render(fmt.Sprintf("%d/%d (%.0f%%)", s.bufferUsage, s.bufferCapacity, pct)))
	}

	// Stats
	statsStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("255"))

	var stats string
	if !s.online {
		// When offline, show a simple waiting message instead of zeroes.
		stats = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Render("Waiting for Albion Online traffic...")
	} else {
		// Format events with drop warning if needed
		eventsDisplay := fmt.Sprintf("Events: %d", s.eventsDecoded)
		if s.eventsDropped > 0 {
			dropStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("196")). // Red
				Bold(true)
			eventsDisplay = fmt.Sprintf("Events: %d  %s",
				s.eventsDecoded,
				dropStyle.Render(fmt.Sprintf("⚠ Dropped: %d", s.eventsDropped)))
		}

		stats = statsStyle.Render(fmt.Sprintf(
			"Packets: %d (%.1f/s)  │  %s  │  %s  %s",
			s.packetsTotal,
			s.packetsPerSec,
			eventsDisplay,
			s.uptime,
			bufStatus,
		))
	}

	// Combine
	content := fmt.Sprintf("%s  │  %s", status, stats)

	// Box style
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(0, 1).
		Width(s.width - 2).
		Align(lipgloss.Center)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("62"))

	title := titleStyle.Render(" Albion Lens ")

	return boxStyle.BorderTop(true).
		BorderBottom(true).
		BorderLeft(true).
		BorderRight(true).
		Render(title + "\n" + content)
}
