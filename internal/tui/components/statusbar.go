package components

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/cantalupo555/albion-lens/pkg/photon"
)

// StatusBar displays connection status, packet stats, and uptime
type StatusBar struct {
	online         bool
	packetsTotal   uint64
	packetsPerSec  float64
	eventsDecoded  uint64
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

// UpdateStats updates the stats display
func (s StatusBar) UpdateStats(stats *photon.Stats) StatusBar {
	if stats != nil {
		s.packetsTotal = stats.GetPacketsReceived()
		s.packetsPerSec = stats.PacketsPerSecond()
		s.eventsDecoded = stats.GetEventsDecoded()
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

	// Stats
	statsStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
	stats := statsStyle.Render(fmt.Sprintf(
		"Packets: %d (%.1f/s)  │  Events: %d  │  %s",
		s.packetsTotal,
		s.packetsPerSec,
		s.eventsDecoded,
		s.uptime,
	))

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
