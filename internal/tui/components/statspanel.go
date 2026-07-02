package components

import (
	"fmt"
	"math"

	"charm.land/lipgloss/v2"
)

// StatsPanel displays session statistics
type StatsPanel struct {
	fame         int64
	silver       int64
	respec       int64
	respecSilver int64
	kills        int
	deaths       int
	lootCount    int
	width        int
	height       int
	fullNumbers  bool
}

// NewStatsPanel creates a new StatsPanel component
func NewStatsPanel() StatsPanel {
	return StatsPanel{
		fullNumbers: true, // Default: show full numbers
	}
}

// SetFullNumbers sets whether to display full or abbreviated numbers
func (s StatsPanel) SetFullNumbers(full bool) StatsPanel {
	s.fullNumbers = full
	return s
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

// SetRespec sets the session respec credits total
func (s StatsPanel) SetRespec(amount int64) StatsPanel {
	s.respec = amount
	return s
}

// SetRespecSilver sets the session respec silver cost total
func (s StatsPanel) SetRespecSilver(amount int64) StatsPanel {
	s.respecSilver = amount
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

// SetKills sets the kill counter to an absolute value (used to hydrate the
// panel from persisted state on startup).
func (s StatsPanel) SetKills(n int) StatsPanel {
	s.kills = n
	return s
}

// SetDeaths sets the death counter to an absolute value.
func (s StatsPanel) SetDeaths(n int) StatsPanel {
	s.deaths = n
	return s
}

// SetLoot sets the loot counter to an absolute value.
func (s StatsPanel) SetLoot(n int) StatsPanel {
	s.lootCount = n
	return s
}

// Reset clears all session stats
func (s StatsPanel) Reset() StatsPanel {
	s.fame = 0
	s.silver = 0
	s.respec = 0
	s.respecSilver = 0
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

	respecValueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("220")).
		Bold(true)

	respecSilverStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("214"))

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
		if s.fullNumbers {
			if n >= 0 {
				return fmt.Sprintf("+%d", n)
			}
			return fmt.Sprintf("%d", n)
		}
		// Abbreviated format with truncation
		sign := "+"
		if n < 0 {
			sign = "-"
		}
		return sign + formatAbbreviated(n)
	}

	respecRow := fmt.Sprintf("%s %s",
		labelStyle.Render("Respec"),
		respecValueStyle.Render(formatNum(s.respec)),
	)
	if s.respecSilver > 0 {
		respecRow = fmt.Sprintf("%s %s %s",
			labelStyle.Render("Respec"),
			respecValueStyle.Render(formatNum(s.respec)),
			respecSilverStyle.Render(fmt.Sprintf("(%s silver)", formatNum(-s.respecSilver))),
		)
	}

	statRows := []string{
		fmt.Sprintf("%s %s",
			labelStyle.Render("Fame"),
			fameValueStyle.Render(formatNum(s.fame)),
		),
		respecRow,
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

	content := lipgloss.JoinVertical(lipgloss.Left, statRows...)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Width(s.width-2).
		Height(s.height-2).
		Padding(0, 1)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("62")).
		MarginBottom(1)

	title := titleStyle.Render("Total Stats")

	return boxStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left, title, content),
	)
}

// formatAbbreviated formats a number in abbreviated form (e.g., 4.9k, 1.3M)
func formatAbbreviated(amount int64) string {
	absAmount := amount
	if absAmount < 0 {
		absAmount = -absAmount
	}
	if absAmount >= 1000000 {
		val := math.Floor(float64(absAmount)/100000.0) / 10.0
		return fmt.Sprintf("%.1fM", val)
	} else if absAmount >= 1000 {
		val := math.Floor(float64(absAmount)/100.0) / 10.0
		return fmt.Sprintf("%.1fk", val)
	}
	return fmt.Sprintf("%d", absAmount)
}
