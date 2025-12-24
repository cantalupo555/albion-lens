package components

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// StatsPanel displays session statistics
type StatsPanel struct {
	fame      int64
	silver    int64
	kills     int
	deaths    int
	lootCount int
	width     int
	height    int
}

// NewStatsPanel creates a new StatsPanel component
func NewStatsPanel() StatsPanel {
	return StatsPanel{}
}

// SetSize updates the dimensions of the stats panel
func (s StatsPanel) SetSize(width, height int) StatsPanel {
	s.width = width
	s.height = height
	return s
}

// AddFame adds fame to the session total
func (s StatsPanel) AddFame(amount int64) StatsPanel {
	s.fame += amount
	return s
}

// SetFame sets the session fame total
func (s StatsPanel) SetFame(amount int64) StatsPanel {
	s.fame = amount
	return s
}

// AddSilver adds silver to the session total
func (s StatsPanel) AddSilver(amount int64) StatsPanel {
	s.silver += amount
	return s
}

// SetSilver sets the session silver total
func (s StatsPanel) SetSilver(amount int64) StatsPanel {
	s.silver = amount
	return s
}

// IncrKills increments the kill counter
func (s StatsPanel) IncrKills() StatsPanel {
	s.kills++
	return s
}

// IncrDeaths increments the death counter
func (s StatsPanel) IncrDeaths() StatsPanel {
	s.deaths++
	return s
}

// IncrLoot increments the loot counter
func (s StatsPanel) IncrLoot() StatsPanel {
	s.lootCount++
	return s
}

// Reset clears all session stats
func (s StatsPanel) Reset() StatsPanel {
	s.fame = 0
	s.silver = 0
	s.kills = 0
	s.deaths = 0
	s.lootCount = 0
	return s
}

// View renders the stats panel
func (s StatsPanel) View() string {
	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")).
		Width(8)

	fameValueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("42")).
		Bold(true)

	silverValueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("214")).
		Bold(true)

	killsValueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true)

	deathsValueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Bold(true)

	lootValueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true)

	// Format numbers with + sign for positive values
	formatNum := func(n int64) string {
		if n >= 0 {
			return fmt.Sprintf("+%d", n)
		}
		return fmt.Sprintf("%d", n)
	}

	rows := []string{
		fmt.Sprintf("%s %s",
			labelStyle.Render("Fame"),
			fameValueStyle.Render(formatNum(s.fame)),
		),
		fmt.Sprintf("%s %s",
			labelStyle.Render("Silver"),
			silverValueStyle.Render(formatNum(s.silver)),
		),
		fmt.Sprintf("%s %s",
			labelStyle.Render("Kills"),
			killsValueStyle.Render(fmt.Sprintf("%d", s.kills)),
		),
		fmt.Sprintf("%s %s",
			labelStyle.Render("Deaths"),
			deathsValueStyle.Render(fmt.Sprintf("%d", s.deaths)),
		),
		fmt.Sprintf("%s %s",
			labelStyle.Render("Loot"),
			lootValueStyle.Render(fmt.Sprintf("%d items", s.lootCount)),
		),
	}

	content := lipgloss.JoinVertical(lipgloss.Left, rows...)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Width(s.width - 2).
		Height(s.height - 2).
		Padding(0, 1)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("62")).
		MarginBottom(1)

	title := titleStyle.Render("Session Stats")

	return boxStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left, title, content),
	)
}
