package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/cantalupo555/albion-lens/internal/tui/components"
	"github.com/cantalupo555/albion-lens/pkg/photon"
)

// Model is the main TUI model
type Model struct {
	statusBar  components.StatusBar
	eventLog   components.EventLog
	statsPanel components.StatsPanel

	// Channels for receiving data from parser
	eventChan chan EventMsg
	statsChan chan *photon.Stats

	// UI state
	width    int
	height   int
	debug    bool
	quitting bool
	ready    bool
}

// New creates a new TUI Model
func New(eventChan chan EventMsg, statsChan chan *photon.Stats) Model {
	return Model{
		statusBar:  components.NewStatusBar(),
		eventLog:   components.NewEventLog(),
		statsPanel: components.NewStatsPanel(),
		eventChan:  eventChan,
		statsChan:  statsChan,
	}
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
		m.eventLog = m.eventLog.AddEvent(msg.Type, msg.Message, msg.Timestamp)

		// Update session stats based on event type
		switch msg.Type {
		case "fame":
			// Could extract amount from message if needed
		case "silver":
			// Could extract amount from message if needed
		case "loot":
			m.statsPanel = m.statsPanel.IncrLoot()
		case "kill":
			m.statsPanel = m.statsPanel.IncrKills()
		case "death":
			m.statsPanel = m.statsPanel.IncrDeaths()
		}

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
		keyStyle.Render("D"), textStyle.Render("ebug"),
	)

	if m.debug {
		debugStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
			Bold(true)
		help += "  " + debugStyle.Render("[DEBUG ON]")
	}

	return lipgloss.NewStyle().
		Padding(0, 1).
		Render(help)
}
