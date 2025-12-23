package tui

import "github.com/charmbracelet/lipgloss"

// Color palette
var (
	ColorPrimary   = lipgloss.Color("62")  // Purple/blue
	ColorSecondary = lipgloss.Color("241") // Gray
	ColorSuccess   = lipgloss.Color("42")  // Green
	ColorWarning   = lipgloss.Color("214") // Yellow/Orange
	ColorDanger    = lipgloss.Color("196") // Red
	ColorInfo      = lipgloss.Color("39")  // Cyan
	ColorMagenta   = lipgloss.Color("205") // Magenta/Pink

	ColorFame   = ColorSuccess
	ColorSilver = ColorWarning
	ColorLoot   = ColorMagenta
	ColorCombat = ColorDanger
)

// Base styles
var (
	// Container with border
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary)

	// Title style for boxes
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			Padding(0, 1)

	// Help bar style
	HelpStyle = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Padding(0, 1)

	// Key style for help
	KeyStyle = lipgloss.NewStyle().
			Foreground(ColorWarning).
			Bold(true)

	// Timestamp style
	TimestampStyle = lipgloss.NewStyle().
			Foreground(ColorSecondary)

	// Status indicators
	OnlineStyle = lipgloss.NewStyle().
			Foreground(ColorSuccess).
			Bold(true)

	OfflineStyle = lipgloss.NewStyle().
			Foreground(ColorDanger).
			Bold(true)

	// Event type styles
	FameStyle = lipgloss.NewStyle().
			Foreground(ColorFame)

	SilverStyle = lipgloss.NewStyle().
			Foreground(ColorSilver)

	LootStyle = lipgloss.NewStyle().
			Foreground(ColorLoot)

	CombatStyle = lipgloss.NewStyle().
			Foreground(ColorCombat)

	// Stats label style
	LabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("255"))

	// Stats value style
	ValueStyle = lipgloss.NewStyle().
			Bold(true)
)

// GetEventStyle returns the appropriate style for an event type
func GetEventStyle(eventType string) lipgloss.Style {
	switch eventType {
	case "fame":
		return FameStyle
	case "silver":
		return SilverStyle
	case "loot":
		return LootStyle
	case "combat", "kill", "death":
		return CombatStyle
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
	}
}
