package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/cantalupo555/albion-lens/internal/tui/components"
	"github.com/cantalupo555/albion-lens/pkg/backend"
	"github.com/cantalupo555/albion-lens/pkg/events"
	"github.com/cantalupo555/albion-lens/pkg/handlers"
	"github.com/cantalupo555/albion-lens/pkg/photon"
)

// Model is the main TUI model
type Model struct {
	statusBar     components.StatusBar
	eventLog      components.EventLog
	statsPanel    components.StatsPanel
	tabBar        components.TabBar
	zonePanel     components.ZonePanel
	dungeonsPanel components.DungeonsPanel

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

	// Debug event filter (interactive). High-frequency/low-signal events are
	// hidden by default but toggleable via the filter overlay (key "/").
	debugCounts  map[events.EventCode]int  // how many times each debug event was seen
	debugHidden  map[events.EventCode]bool // codes currently suppressed from the log
	filterOpen   bool
	filterCursor int
}

// Tab indices for navigation. Kept as constants so the help bar and key
// handlers stay in sync if new tabs are added.
const (
	TabDashboard = iota
	TabZone
	TabDungeons
)

// minFilterVisibleItems is the minimum number of rows shown in the debug event
// filter overlay (one row for a single entry, even on very short terminals).
const minFilterVisibleItems = 3

// New creates a new TUI Model
func New(svc *backend.Service, bulkEventChan chan BulkEventMsg, statsChan chan *photon.Stats) Model {
	m := Model{
		statusBar:     components.NewStatusBar(),
		eventLog:      components.NewEventLog(),
		statsPanel:    components.NewStatsPanel(),
		tabBar:        components.NewTabBar(),
		zonePanel:     components.NewZonePanel(),
		dungeonsPanel: components.NewDungeonsPanel(),
		svc:           svc,
		bulkEventChan: bulkEventChan,
		statsChan:     statsChan,
		fullNumbers:   false, // Default: abbreviated numbers (e.g., 4.9k)
		debugCounts:   make(map[events.EventCode]int),
		debugHidden:   defaultHiddenDebugEvents(),
	}
	// Sync debug state from service
	if svc != nil {
		m.debug = svc.IsDebug()
		m.onlineChan = svc.OnlineStatus
	}
	return m
}

// defaultHiddenDebugEvents returns the set of high-frequency / low-signal debug
// events that are suppressed from the event log by default. They remain
// counted and can be re-enabled interactively via the debug filter overlay.
func defaultHiddenDebugEvents() map[events.EventCode]bool {
	return map[events.EventCode]bool{
		events.EventLeave:                    true, // 1 — entity leaves render range (constant spam)
		events.EventMove:                     true, // 3 — movement updates
		events.EventActiveSpellEffectsUpdate: true, // 11 — buff tick refresh
		events.EventTimeSync:                 true, // 160 — periodic clock sync
		events.EventPlayerCounts:             true, // 275 — periodic population poll
		events.EventBattlEyeServerMessage:    true, // 406 — anti-cheat heartbeat
	}
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
		// Filter overlay captures its own keys when open.
		if m.filterOpen {
			switch msg.String() {
			case "/", "esc":
				m.filterOpen = false
				return m, nil
			case "up", "k":
				if m.filterCursor > 0 {
					m.filterCursor--
				}
				return m, nil
			case "down", "j":
				if m.filterCursor < len(m.sortedDebugCodes())-1 {
					m.filterCursor++
				}
				return m, nil
			case "space", "enter":
				codes := m.sortedDebugCodes()
				if m.filterCursor >= 0 && m.filterCursor < len(codes) {
					code := codes[m.filterCursor]
					m.debugHidden[code] = !m.debugHidden[code]
				}
				return m, nil
			}
			// Other keys are swallowed while the filter is open.
			return m, nil
		}

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
		case "3":
			m.tabBar = m.tabBar.SetActive(TabDungeons)
			return m, nil
		case "c", "C":
			m = m.clearActivePanel()
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
			m.dungeonsPanel = m.dungeonsPanel.SetFullNumbers(m.fullNumbers)
			return m, nil
		case "/":
			// Toggle the debug-event filter overlay.
			m.filterOpen = !m.filterOpen
			m.filterCursor = 0
			return m, nil
		case "r", "R":
			m.statsPanel = m.statsPanel.Reset()
			return m, nil
		case "up", "k":
			m = m.scrollActive(-1)
			return m, nil
		case "down", "j":
			m = m.scrollActive(1)
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

			// Debug event filtering: count every debug event by code and skip
			// hidden ones so they never reach the log. Diagnostic debug lines
			// (Data is not an EventCode) are always shown.
			if eventMsg.Type == "debug" {
				if code, ok := eventMsg.Data.(events.EventCode); ok {
					m.debugCounts[code]++
					if m.debugHidden[code] {
						continue
					}
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

			// Refresh dungeon runs panel
			if h := m.svc.Handler(); h != nil {
				m.dungeonsPanel = m.dungeonsPanel.SetRuns(h.GetDungeonRuns(), h.GetActiveDungeon()).SetNow(time.Now())
			}
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
	m.eventLog = m.eventLog.SetSize(eventLogWidth, mainHeight)
	m.statsPanel = m.statsPanel.SetSize(statsPanelWidth, mainHeight)
	m.zonePanel = m.zonePanel.SetSize(m.width, mainHeight)
	m.dungeonsPanel = m.dungeonsPanel.SetSize(m.width, mainHeight)

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
	switch m.tabBar.Active() {
	case TabZone:
		mainPanel = m.zonePanel.View()
	case TabDungeons:
		mainPanel = m.dungeonsPanel.View()
	default:
		mainPanel = lipgloss.JoinHorizontal(
			lipgloss.Top,
			m.eventLog.View(),
			m.statsPanel.View(),
		)
	}

	// Debug filter: overlay a centered modal ON TOP of the normal content so
	// the dashboard stays visible behind/around the pop-up.
	if m.filterOpen {
		mainH := m.height - 4 /*status*/ - 1 /*tab*/ - 1 /*help*/
		if mainH < 5 {
			mainH = 5
		}
		modal := m.renderFilterPanel(m.width, mainH)
		mainPanel = overlayText(mainPanel, modal, m.width, mainH)
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

// clearActivePanel dispatches a clear action to the panel matching the active
// tab. Adding a tab requires only adding a case here.
func (m Model) clearActivePanel() Model {
	switch m.tabBar.Active() {
	case TabZone:
		m.zonePanel = m.zonePanel.Clear()
	case TabDungeons:
		m.dungeonsPanel = m.dungeonsPanel.Clear()
	default:
		m.eventLog = m.eventLog.Clear()
	}
	return m
}

// scrollActive dispatches a scroll action to the panel matching the active
// tab. dir < 0 scrolls up; dir > 0 scrolls down.
func (m Model) scrollActive(dir int) Model {
	switch m.tabBar.Active() {
	case TabZone:
		if dir > 0 {
			m.zonePanel = m.zonePanel.ScrollDown()
		} else {
			m.zonePanel = m.zonePanel.ScrollUp()
		}
	case TabDungeons:
		if dir > 0 {
			m.dungeonsPanel = m.dungeonsPanel.ScrollDown()
		} else {
			m.dungeonsPanel = m.dungeonsPanel.ScrollUp()
		}
	default:
		if dir > 0 {
			m.eventLog = m.eventLog.ScrollDown()
		} else {
			m.eventLog = m.eventLog.ScrollUp()
		}
	}
	return m
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
		keyStyle.Render("D"), textStyle.Render("ebug  "),
		keyStyle.Render("/"), textStyle.Render("Filter"),
	)

	// Show active toggles
	toggleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("214")).
		Bold(true)

	// Show active tab indicator
	switch m.tabBar.Active() {
	case TabZone:
		help += "  " + toggleStyle.Render("[ZONE]")
	case TabDungeons:
		help += "  " + toggleStyle.Render("[DG]")
	default:
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

// sortedDebugCodes returns the debug event codes seen so far, sorted ascending
// by numeric code. Used by the filter overlay for both navigation and rendering.
func (m Model) sortedDebugCodes() []events.EventCode {
	codes := make([]events.EventCode, 0, len(m.debugCounts))
	for c := range m.debugCounts {
		codes = append(codes, c)
	}
	sort.Slice(codes, func(i, j int) bool { return int(codes[i]) < int(codes[j]) })
	return codes
}

// renderFilterPanel returns just the bordered modal box (the caller overlays
// it onto the background content via overlayText).
func (m Model) renderFilterPanel(areaW, areaH int) string {
	// Reserve: 2 border + 1 title + 1 hint + 1 blank + 2 scroll indicators + 2 margin
	maxItems := areaH - 9
	if maxItems < minFilterVisibleItems {
		maxItems = minFilterVisibleItems
	}
	inner := m.filterListContent(maxItems)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("213")).
		Background(lipgloss.Color("235")).
		Padding(0, 1).
		Width(clampInt(areaW-4, 40, 80))

	return boxStyle.Render(inner)
}

// overlayText pastes the modal string centered over the background string,
// preserving the background content visible around the modal. Both strings may
// contain ANSI escape sequences; the overlay is performed at the display-width
// level so styling is not corrupted.
func overlayText(background, modal string, areaW, areaH int) string {
	bgLines := strings.Split(background, "\n")
	modalLines := strings.Split(modal, "\n")

	modalH := len(modalLines)
	modalW := 0
	for _, l := range modalLines {
		if w := lipgloss.Width(l); w > modalW {
			modalW = w
		}
	}

	x := (areaW - modalW) / 2
	if x < 0 {
		x = 0
	}
	y := (areaH - modalH) / 2
	if y < 0 {
		y = 0
	}

	result := make([]string, len(bgLines))
	for i, bgLine := range bgLines {
		if i >= y && i < y+modalH {
			modalLine := modalLines[i-y]
			leftCut := displayOffsetToByte(bgLine, x)
			rightCut := displayOffsetToByte(bgLine, x+modalW)
			left := bgLine[:leftCut]
			if w := lipgloss.Width(left); w < x {
				left += strings.Repeat(" ", x-w)
			}
			result[i] = left + modalLine + bgLine[rightCut:]
		} else {
			result[i] = bgLine
		}
	}
	return strings.Join(result, "\n")
}

// displayOffsetToByte returns the byte index in s corresponding to the given
// display-width position, skipping ANSI escape sequences (which have zero
// display width). Each rune's actual display width is measured via
// lipgloss.Width to correctly handle CJK (width 2), emoji (width 2), and
// combining characters (width 0).
func displayOffsetToByte(s string, targetWidth int) int {
	if targetWidth <= 0 {
		return 0
	}
	width := 0
	i := 0
	for i < len(s) {
		// ANSI CSI escape: ESC [ ... <final byte 0x40-0x7e>
		if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '[' {
			j := i + 2
			for j < len(s) {
				c := s[j]
				j++
				if c >= 0x40 && c <= 0x7e {
					break
				}
			}
			i = j
			continue
		}
		if width >= targetWidth {
			return i
		}
		// Advance one UTF-8 rune, using its actual display width.
		r, size := utf8.DecodeRuneInString(s[i:])
		width += lipgloss.Width(string(r))
		i += size
	}
	return i
}

// filterListContent builds the inner text of the debug-event filter: a title
// with a close hint, and a scrollable window of rows. Only maxItems rows are
// shown at once; the window follows the cursor. When rows are hidden above or
// below, a count indicator is displayed.
func (m Model) filterListContent(maxItems int) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("213"))
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	curStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))
	onStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("119"))  // green = visible
	offStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241")) // dim = hidden

	var b strings.Builder
	b.WriteString(titleStyle.Render("Debug Event Filter") + "\n")
	b.WriteString(hintStyle.Render("  up/down move  -  space toggle  -  / close") + "\n\n")

	codes := m.sortedDebugCodes()
	if len(codes) == 0 {
		b.WriteString(hintStyle.Render("No debug events captured yet.\nEnable debug with D."))
		return b.String()
	}

	// Compute the visible scroll window centered on the cursor.
	start, end := scrollWindow(m.filterCursor, len(codes), maxItems)

	if start > 0 {
		b.WriteString(hintStyle.Render(fmt.Sprintf("  (%d more above)", start)) + "\n")
	}

	for i := start; i < end; i++ {
		code := codes[i]
		name := events.EventCodeNames[code]
		if name == "" {
			name = "?"
		}
		count := m.debugCounts[code]

		marker := "v" // visible
		rowStyle := onStyle
		if m.debugHidden[code] {
			marker = "x" // hidden
			rowStyle = offStyle
		}

		row := fmt.Sprintf("[%s] %3d  %-28s %d", marker, int(code), name, count)
		if i == m.filterCursor {
			row = curStyle.Render("> ") + rowStyle.Render(row)
		} else {
			row = "  " + rowStyle.Render(row)
		}
		b.WriteString(row + "\n")
	}

	if end < len(codes) {
		b.WriteString(hintStyle.Render(fmt.Sprintf("  (%d more below)", len(codes)-end)) + "\n")
	}

	return b.String()
}

// scrollWindow returns the [start, end) indices of a scrollable list of length
// n, such that the window of size maxItems contains the cursor and is clamped
// to valid bounds.
func scrollWindow(cursor, n, maxItems int) (int, int) {
	if n <= maxItems {
		return 0, n
	}
	start := cursor - maxItems/2
	if start < 0 {
		start = 0
	}
	if start+maxItems > n {
		start = n - maxItems
		if start < 0 {
			start = 0
		}
	}
	return start, start + maxItems
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// formatNumber formats a number based on fullNumbers setting
// If fullNumbers is true, returns the full number (e.g., 4984)
// If fullNumbers is false, returns abbreviated form (e.g., 4.9k)
func formatNumber(amount int64, full bool) string {
	return components.FormatNumber(amount, full)
}
