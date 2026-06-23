package tui

import (
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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
	tabBar     components.TabBar
	zonePanel  components.ZonePanel

	// Backend service reference for runtime control
	svc *backend.Service

	// Channels for receiving data from parser
	bulkEventChan chan BulkEventMsg
	statsChan     chan *photon.Stats
	onlineChan    <-chan bool

	// UI state
	width    int
	height   int
	debug    bool
	quitting bool
	ready    bool

	// Display settings
	fullNumbers bool // Show full numbers instead of abbreviated (e.g., 4984 vs 4.9k)
}

// Tab indices for navigation. Kept as constants so the help bar and key
// handlers stay in sync if new tabs are added.
const (
	TabDashboard = iota
	TabZone
)

// New creates a new TUI Model
func New(svc *backend.Service, bulkEventChan chan BulkEventMsg, statsChan chan *photon.Stats) Model {
	m := Model{
		statusBar:     components.NewStatusBar(),
		eventLog:      components.NewEventLog(),
		statsPanel:    components.NewStatsPanel(),
		tabBar:        components.NewTabBar(),
		zonePanel:     components.NewZonePanel(),
		svc:           svc,
		bulkEventChan: bulkEventChan,
		statsChan:     statsChan,
		fullNumbers:   false, // Default: abbreviated numbers (e.g., 4.9k)
	}
	// Sync debug state from service
	if svc != nil {
		m.debug = svc.IsDebug()
		m.onlineChan = svc.OnlineStatus
	}
	return m
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{
		TickCmd(), // Start the tick timer
	}

	// Listen for events if channel provided
	if m.bulkEventChan != nil {
		cmds = append(cmds, WaitForBulkEvent(m.bulkEventChan))
	}

	// Listen for stats if channel provided
	if m.statsChan != nil {
		cmds = append(cmds, WaitForStats(m.statsChan))
	}

	// Listen for online status changes if channel provided
	if m.onlineChan != nil {
		cmds = append(cmds, WaitForOnline(m.onlineChan))
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
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "Q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "tab":
			m.tabBar = m.tabBar.Next()
			return m, nil
		case "shift+tab":
			m.tabBar = m.tabBar.Prev()
			return m, nil
		case "1":
			m.tabBar = m.tabBar.SetActive(TabDashboard)
			return m, nil
		case "2":
			m.tabBar = m.tabBar.SetActive(TabZone)
			return m, nil
		case "c", "C":
			if m.tabBar.Active() == TabZone {
				m.zonePanel = m.zonePanel.Clear()
			} else {
				m.eventLog = m.eventLog.Clear()
			}
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
			if m.tabBar.Active() == TabZone {
				m.zonePanel = m.zonePanel.ScrollUp()
			} else {
				m.eventLog = m.eventLog.ScrollUp()
			}
			return m, nil
		case "down", "j":
			if m.tabBar.Active() == TabZone {
				m.zonePanel = m.zonePanel.ScrollDown()
			} else {
				m.eventLog = m.eventLog.ScrollDown()
			}
			return m, nil
		}

	// Batch of game events from parser
	case BulkEventMsg:
		var logEvents []components.Event

		for _, eventMsg := range msg {
			displayMsg := eventMsg.Message

			// Update session stats based on event type and data
			switch eventMsg.Type {
			case "fame":
				if data, ok := eventMsg.Data.(*handlers.FameEventData); ok && data != nil {
					m.statsPanel = m.statsPanel.SetFame(data.Session)
					displayMsg = fmt.Sprintf("⭐ FAME: +%s | Total: %s | Session: %s",
						formatNumber(data.Gained, m.fullNumbers),
						formatNumber(data.Total, m.fullNumbers),
						formatNumber(data.Session, m.fullNumbers))
				}
			case "silver":
				if data, ok := eventMsg.Data.(*handlers.SilverEventData); ok && data != nil {
					m.statsPanel = m.statsPanel.SetSilver(data.Session)
					displayMsg = fmt.Sprintf("💰 %s looted silver (%s) from %s | Session: %s",
						data.LootedBy,
						formatNumber(data.Amount, m.fullNumbers),
						data.LootedFrom,
						formatNumber(data.Session, m.fullNumbers))
				}
			case "respec":
				if data, ok := eventMsg.Data.(*handlers.RespecEventData); ok && data != nil {
					m.statsPanel = m.statsPanel.SetRespec(data.SessionTotal)
					m.statsPanel = m.statsPanel.SetRespecSilver(data.SessionSilverTotal)
					if data.PaidSilver > 0 {
						displayMsg = fmt.Sprintf("🏆 RESPEC: +%s credits | Silver cost: -%s | Session: %s",
							formatNumber(data.Gained, m.fullNumbers),
							formatNumber(data.PaidSilver, m.fullNumbers),
							formatNumber(data.SessionTotal, m.fullNumbers))
					} else {
						displayMsg = fmt.Sprintf("🏆 RESPEC: +%s credits | Session: %s",
							formatNumber(data.Gained, m.fullNumbers),
							formatNumber(data.SessionTotal, m.fullNumbers))
					}
				}
			case "loot":
				m.statsPanel = m.statsPanel.IncrLoot()
			case "kill":
				m.statsPanel = m.statsPanel.IncrKills()
			case "death":
				m.statsPanel = m.statsPanel.IncrDeaths()
			case "zone":
				if data, ok := eventMsg.Data.(*handlers.ZoneEventData); ok && data != nil {
					info := handlers.ZoneInfo{
						MapType:      data.MapType,
						ClusterIndex: data.ClusterIndex,
						IslandName:   data.IslandName,
					}
					m.zonePanel = m.zonePanel.AddTransition(info, eventMsg.Timestamp)
				}
			}

			logEvents = append(logEvents, components.Event{
				Type:      eventMsg.Type,
				Message:   displayMsg,
				Timestamp: eventMsg.Timestamp,
				Data:      eventMsg.Data,
			})
		}

		// Add all events to log at once (efficient batch render)
		m.eventLog = m.eventLog.AddEvents(logEvents)

		// Continue listening for events
		if m.bulkEventChan != nil {
			cmds = append(cmds, WaitForBulkEvent(m.bulkEventChan))
		}
		return m, tea.Batch(cmds...)

	// Stats update from parser
	case StatsUpdateMsg:
		m.statusBar = m.statusBar.UpdateStats(msg.Stats)

		// Continue listening for stats
		if m.statsChan != nil {
			cmds = append(cmds, WaitForStats(m.statsChan))
		}
		return m, tea.Batch(cmds...)

	// Online status change
	case OnlineMsg:
		m.statusBar = m.statusBar.SetOnline(msg.Online)
		// Continue listening for status changes
		if m.onlineChan != nil {
			cmds = append(cmds, WaitForOnline(m.onlineChan))
		}
		return m, tea.Batch(cmds...)

	// Periodic tick
	case TickMsg:
		// Refresh player identification, zone, and online status from the backend
		if m.svc != nil {
			m.statusBar = m.statusBar.SetPlayerName(m.svc.LocalPlayerName())
			m.statusBar = m.statusBar.SetOnline(m.svc.IsOnline())

			zone := m.svc.CurrentZone()
			m.statusBar = m.statusBar.SetZone(zone.DisplayString())
			m.zonePanel = m.zonePanel.SetZone(zone).SetNow(time.Now())
		}
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
	// Reserve space for status bar (4 lines), tab bar (1 line), and help bar (1 line)
	statusBarHeight := 4
	tabBarHeight := 1
	helpBarHeight := 1
	mainHeight := m.height - statusBarHeight - tabBarHeight - helpBarHeight

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
	m.tabBar = m.tabBar.SetWidth(m.width)
	m.eventLog = m.eventLog.SetSize(eventLogWidth, mainHeight)
	m.statsPanel = m.statsPanel.SetSize(statsPanelWidth, mainHeight)
	m.zonePanel = m.zonePanel.SetSize(m.width, mainHeight)

	return m
}

// View renders the TUI
func (m Model) View() tea.View {
	if m.quitting {
		return tea.NewView("Goodbye!\n")
	}

	if !m.ready {
		return tea.NewView("Initializing...")
	}

	// Status bar (top)
	statusBar := m.statusBar.View()

	// Tab bar
	tabBar := m.tabBar.View()

	// Main content depends on the active tab
	var mainPanel string
	if m.tabBar.Active() == TabZone {
		mainPanel = m.zonePanel.View()
	} else {
		mainPanel = lipgloss.JoinHorizontal(
			lipgloss.Top,
			m.eventLog.View(),
			m.statsPanel.View(),
		)
	}

	// Help bar (bottom)
	helpBar := m.renderHelpBar()

	// Combine all sections
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		statusBar,
		tabBar,
		mainPanel,
		helpBar,
	)

	v := tea.NewView(content)
	v.AltScreen = true
	return v
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
		keyStyle.Render("Tab"), textStyle.Render("Switch  "),
		keyStyle.Render("C"), textStyle.Render("lear  "),
		keyStyle.Render("R"), textStyle.Render("eset stats  "),
		keyStyle.Render("F"), textStyle.Render("ull numbers  "),
		keyStyle.Render("D"), textStyle.Render("ebug"),
	)

	// Show active toggles
	toggleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("214")).
		Bold(true)

	// Show active tab indicator
	if m.tabBar.Active() == TabZone {
		help += "  " + toggleStyle.Render("[ZONE]")
	} else {
		help += "  " + toggleStyle.Render("[DASH]")
	}
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
	return components.FormatNumber(amount, full)
}
