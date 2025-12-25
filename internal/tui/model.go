package tui

import (
	"fmt"
	"math"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/cantalupo555/albion-lens/internal/tui/components"
	"github.com/cantalupo555/albion-lens/pkg/backend"
	"github.com/cantalupo555/albion-lens/pkg/handlers"
	"github.com/cantalupo555/albion-lens/pkg/photon"
)

// Model is the main TUI model
type Model struct {
	statusBar  components.StatusBar
	eventLog   components.EventLog
	statsPanel components.StatsPanel

	// Backend service reference for runtime control
	svc *backend.Service

	// Channels for receiving data from parser
	eventChan chan EventMsg
	statsChan chan *photon.Stats

	// UI state
	width    int
	height   int
	debug    bool
	quitting bool
	ready    bool

	// Display settings
	fullNumbers bool // Show full numbers instead of abbreviated (e.g., 4984 vs 4.9k)
}

// New creates a new TUI Model
func New(svc *backend.Service, eventChan chan EventMsg, statsChan chan *photon.Stats) Model {
	m := Model{
		statusBar:   components.NewStatusBar(),
		eventLog:    components.NewEventLog(),
		statsPanel:  components.NewStatsPanel(),
		svc:         svc,
		eventChan:   eventChan,
		statsChan:   statsChan,
		fullNumbers: false, // Default: abbreviated numbers (e.g., 4.9k)
	}
	// Sync debug state from service
	if svc != nil {
		m.debug = svc.IsDebug()
	}
	return m
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{
		TickCmd(), // Start the tick timer
	}

	// Listen for events if channel provided
	if m.eventChan != nil {
		cmds = append(cmds, WaitForEvent(m.eventChan))
	}

	// Listen for stats if channel provided
	if m.statsChan != nil {
		cmds = append(cmds, WaitForStats(m.statsChan))
	}

	return tea.Batch(cmds...)
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	// Window resize
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m = m.updateLayout()
		m.ready = true
		return m, nil

	// Keyboard input
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "Q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "c", "C":
			m.eventLog = m.eventLog.Clear()
			return m, nil
		case "d", "D":
			m.debug = !m.debug
			// Propagate to backend service
			if m.svc != nil {
				m.svc.SetDebug(m.debug)
			}
			return m, nil
		case "f", "F":
			m.fullNumbers = !m.fullNumbers
			m.statsPanel = m.statsPanel.SetFullNumbers(m.fullNumbers)
			m.eventLog = m.eventLog.SetFullNumbers(m.fullNumbers)
			return m, nil
		case "r", "R":
			m.statsPanel = m.statsPanel.Reset()
			return m, nil
		case "up", "k":
			m.eventLog = m.eventLog.ScrollUp()
			return m, nil
		case "down", "j":
			m.eventLog = m.eventLog.ScrollDown()
			return m, nil
		}

	// Game event from parser
	case EventMsg:
		displayMsg := msg.Message

		// Update session stats based on event type and data
		switch msg.Type {
		case "fame":
			if data, ok := msg.Data.(*handlers.FameEventData); ok && data != nil {
				m.statsPanel = m.statsPanel.SetFame(data.Session)
				// Format fame message based on fullNumbers setting
				displayMsg = fmt.Sprintf("⭐ FAME: +%s | Total: %s | Session: %s",
					formatNumber(data.Gained, m.fullNumbers),
					formatNumber(data.Total, m.fullNumbers),
					formatNumber(data.Session, m.fullNumbers))
			}
		case "silver":
			if data, ok := msg.Data.(*handlers.SilverEventData); ok && data != nil {
				m.statsPanel = m.statsPanel.SetSilver(data.Session)
				// Format silver message based on fullNumbers setting
				displayMsg = fmt.Sprintf("💰 %s looted silver (%s) from %s | Session: %s",
					data.LootedBy,
					formatNumber(data.Amount, m.fullNumbers),
					data.LootedFrom,
					formatNumber(data.Session, m.fullNumbers))
			}
		case "loot":
			m.statsPanel = m.statsPanel.IncrLoot()
		case "kill":
			m.statsPanel = m.statsPanel.IncrKills()
		case "death":
			m.statsPanel = m.statsPanel.IncrDeaths()
		}

		m.eventLog = m.eventLog.AddEvent(msg.Type, displayMsg, msg.Timestamp, msg.Data)

		// Continue listening for events
		if m.eventChan != nil {
			cmds = append(cmds, WaitForEvent(m.eventChan))
		}
		return m, tea.Batch(cmds...)

	// Stats update from parser
	case StatsUpdateMsg:
		m.statusBar = m.statusBar.UpdateStats(msg.Stats)
		m.statusBar = m.statusBar.SetOnline(true)

		// Continue listening for stats
		if m.statsChan != nil {
			cmds = append(cmds, WaitForStats(m.statsChan))
		}
		return m, tea.Batch(cmds...)

	// Online status change
	case OnlineMsg:
		m.statusBar = m.statusBar.SetOnline(msg.Online)
		return m, nil

	// Periodic tick
	case TickMsg:
		// Refresh display periodically
		cmds = append(cmds, TickCmd())
		return m, tea.Batch(cmds...)

	// Session stats update (from handler)
	case SessionStatsMsg:
		if msg.Fame > 0 {
			m.statsPanel = m.statsPanel.AddFame(msg.Fame)
		}
		if msg.Silver > 0 {
			m.statsPanel = m.statsPanel.AddSilver(msg.Silver)
		}
		return m, nil
	}

	return m, tea.Batch(cmds...)
}

// updateLayout recalculates component sizes based on window dimensions
func (m Model) updateLayout() Model {
	// Reserve space for status bar (4 lines) and help bar (1 line)
	statusBarHeight := 4
	helpBarHeight := 1
	mainHeight := m.height - statusBarHeight - helpBarHeight

	if mainHeight < 5 {
		mainHeight = 5
	}

	// Event log takes 75% width, stats panel takes 25%
	eventLogWidth := m.width * 3 / 4
	statsPanelWidth := m.width - eventLogWidth

	if eventLogWidth < 20 {
		eventLogWidth = 20
	}
	if statsPanelWidth < 15 {
		statsPanelWidth = 15
	}

	m.statusBar = m.statusBar.SetWidth(m.width)
	m.eventLog = m.eventLog.SetSize(eventLogWidth, mainHeight)
	m.statsPanel = m.statsPanel.SetSize(statsPanelWidth, mainHeight)

	return m
}

// View renders the TUI
func (m Model) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}

	if !m.ready {
		return "Initializing..."
	}

	// Status bar (top)
	statusBar := m.statusBar.View()

	// Main panel (event log + stats side by side)
	mainPanel := lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.eventLog.View(),
		m.statsPanel.View(),
	)

	// Help bar (bottom)
	helpBar := m.renderHelpBar()

	// Combine all sections
	return lipgloss.JoinVertical(
		lipgloss.Left,
		statusBar,
		mainPanel,
		helpBar,
	)
}

// renderHelpBar renders the help bar at the bottom
func (m Model) renderHelpBar() string {
	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("214")).
		Bold(true)

	textStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	help := lipgloss.JoinHorizontal(lipgloss.Left,
		keyStyle.Render("Q"), textStyle.Render("uit  "),
		keyStyle.Render("C"), textStyle.Render("lear  "),
		keyStyle.Render("R"), textStyle.Render("eset stats  "),
		keyStyle.Render("F"), textStyle.Render("ull numbers  "),
		keyStyle.Render("D"), textStyle.Render("ebug"),
	)

	// Show active toggles
	toggleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("214")).
		Bold(true)

	if m.fullNumbers {
		help += "  " + toggleStyle.Render("[FULL]")
	}
	if m.debug {
		help += "  " + toggleStyle.Render("[DEBUG]")
	}

	return lipgloss.NewStyle().
		Padding(0, 1).
		Render(help)
}

// formatNumber formats a number based on fullNumbers setting
// If fullNumbers is true, returns the full number (e.g., 4984)
// If fullNumbers is false, returns abbreviated form (e.g., 4.9k)
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
