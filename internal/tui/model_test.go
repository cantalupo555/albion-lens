package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/cantalupo555/albion-lens/internal/tui/components"
	"github.com/cantalupo555/albion-lens/pkg/backend"
	"github.com/cantalupo555/albion-lens/pkg/events"
	"github.com/cantalupo555/albion-lens/pkg/handlers"
	"github.com/cantalupo555/albion-lens/pkg/photon"
)

// updateModel is a test helper that calls Update and asserts the result is a Model.
func updateModel(m Model, msg tea.Msg) Model {
	nm, _ := m.Update(msg)
	return nm.(Model)
}

// readyModel creates a new Model with a WindowSizeMsg applied (ready=true).
func readyModel() Model {
	m := New(nil, nil, nil)
	return updateModel(m, tea.WindowSizeMsg{Width: 80, Height: 24})
}

// ============================================
// formatNumber tests
// ============================================

func TestFormatNumberFull(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{0, "0"},
		{999, "999"},
		{1000, "1000"},
		{4984, "4984"},
		{1000000, "1000000"},
	}
	for _, tt := range tests {
		got := formatNumber(tt.input, true)
		if got != tt.want {
			t.Errorf("formatNumber(%d, true) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFormatNumberAbbreviated(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{0, "0"},
		{999, "999"},
		{1000, "1.0k"},
		{4984, "4.9k"},
		{999999, "999.9k"},
		{1000000, "1.0M"},
		{1500000, "1.5M"},
	}
	for _, tt := range tests {
		got := formatNumber(tt.input, false)
		if got != tt.want {
			t.Errorf("formatNumber(%d, false) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// ============================================
// New model tests
// ============================================

func TestNewModelDefaults(t *testing.T) {
	m := New(nil, nil, nil)

	if m.debug {
		t.Error("expected debug=false by default")
	}
	if m.fullNumbers {
		t.Error("expected fullNumbers=false by default")
	}
	if m.quitting {
		t.Error("expected quitting=false by default")
	}
	if m.ready {
		t.Error("expected ready=false by default")
	}
}

func TestHydrateStatsPanel(t *testing.T) {
	h := handlers.NewAlbionHandler()

	// Seed the handler via the public disk-load API: the session counters are
	// not settable from outside the handlers package.
	path := filepath.Join(t.TempDir(), "stats.json")
	seed := []byte(`{"version":1,"fame":123456,"silver":987654,"respec":333,"respec_silver":2222,"kills":42,"deaths":17,"loot":99,"saved_at":"2026-06-29T00:00:00Z"}`)
	if err := os.WriteFile(path, seed, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := h.LoadTotalStats(path); err != nil {
		t.Fatalf("LoadTotalStats: %v", err)
	}

	panel := hydrateStatsPanel(components.NewStatsPanel(), h, time.Now())
	// NewStatsPanel defaults to fullNumbers, so fame renders as "+123456".
	view := panel.View()
	for _, want := range []string{"123456", "987654", "99 items"} {
		if !strings.Contains(view, want) {
			t.Errorf("stats panel view missing %q after hydration\n%s", want, view)
		}
	}
}

func TestHydrateStatsPanelNilHandlerIsNoOp(t *testing.T) {
	empty := components.NewStatsPanel()
	got := hydrateStatsPanel(empty, nil, time.Now())
	if got.View() != empty.View() {
		t.Error("hydrateStatsPanel with nil handler should leave the panel unchanged")
	}
}

// TestHydrateStatsPanelSeedsTodayFromDailyBucket verifies that hydration picks
// up today's daily bucket values (fame/silver/respec) from persisted state.
func TestHydrateStatsPanelSeedsTodayFromDailyBucket(t *testing.T) {
	h := handlers.NewAlbionHandler()

	// Seed with a v2 file that has a daily bucket for 2026-07-01.
	path := filepath.Join(t.TempDir(), "stats.json")
	seed := []byte(`{"version":2,"fame":1000000,"silver":500000,"respec":20000,"respec_silver":1000,"kills":42,"deaths":17,"loot":99,"daily":{"2026-07-01":{"fame":12000,"silver":3400,"respec":800,"respec_silver":0,"kills":0,"deaths":0,"loot":0}},"saved_at":"2026-07-01T12:00:00Z"}`)
	if err := os.WriteFile(path, seed, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := h.LoadTotalStats(path); err != nil {
		t.Fatalf("LoadTotalStats: %v", err)
	}

	// Hydrate using the same date the bucket is keyed on.
	now := time.Date(2026, 7, 1, 14, 0, 0, 0, time.UTC)
	panel := hydrateStatsPanel(components.NewStatsPanel(), h, now)
	view := panel.View()

	for _, want := range []string{"12000", "3400", "800", "today"} {
		if !strings.Contains(view, want) {
			t.Errorf("stats panel view missing %q after hydration with daily bucket\n%s", want, view)
		}
	}
}

// ============================================
// Update: keyboard input tests
// ============================================

func TestUpdateKeyTabNext(t *testing.T) {
	m := readyModel()

	if m.tabBar.Active() != TabDashboard {
		t.Fatalf("expected Dashboard active initially, got %d", m.tabBar.Active())
	}

	m = updateModel(m, tea.KeyPressMsg{Code: tea.KeyTab})

	if m.tabBar.Active() != TabZone {
		t.Errorf("expected Zone active after Tab, got %d", m.tabBar.Active())
	}

	m = updateModel(m, tea.KeyPressMsg{Code: tea.KeyTab})

	if m.tabBar.Active() != TabDungeons {
		t.Errorf("expected Dungeons active after second Tab, got %d", m.tabBar.Active())
	}
}

func TestUpdateKeyTabPrev(t *testing.T) {
	m := readyModel()

	m = updateModel(m, tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})

	if m.tabBar.Active() != TabDungeons {
		t.Errorf("expected Dungeons active after Shift+Tab, got %d", m.tabBar.Active())
	}
}

func TestUpdateKeyNumberSwitch(t *testing.T) {
	m := readyModel()

	// Press '2' → Zone tab.
	m = updateModel(m, tea.KeyPressMsg{Code: '2'})
	if m.tabBar.Active() != TabZone {
		t.Errorf("expected Zone active after '2', got %d", m.tabBar.Active())
	}

	// Press '3' → Dungeons tab.
	m = updateModel(m, tea.KeyPressMsg{Code: '3'})
	if m.tabBar.Active() != TabDungeons {
		t.Errorf("expected Dungeons active after '3', got %d", m.tabBar.Active())
	}

	// Press '1' → Dashboard tab.
	m = updateModel(m, tea.KeyPressMsg{Code: '1'})
	if m.tabBar.Active() != TabDashboard {
		t.Errorf("expected Dashboard active after '1', got %d", m.tabBar.Active())
	}
}

func TestUpdateKeyClearRouting(t *testing.T) {
	m := readyModel()

	// Add events to both event log and zone panel.
	m = updateModel(m, BulkEventMsg{
		{Type: "info", Message: "log event", Timestamp: time.Now()},
	})
	m = updateModel(m, BulkEventMsg{
		{
			Type:      "zone",
			Timestamp: time.Now(),
			Data: &handlers.ZoneEventData{
				MapType: handlers.MapTypeIsland,
				Display: "Island",
			},
		},
	})

	// Switch to Zone tab and clear — should clear zone panel, not event log.
	m = updateModel(m, tea.KeyPressMsg{Code: tea.KeyTab})
	m = updateModel(m, tea.KeyPressMsg{Code: 'c'})

	// Switch back to Dashboard and verify event log is intact.
	m = updateModel(m, tea.KeyPressMsg{Code: '1'})
	view := m.View().Content
	if !strings.Contains(view, "log event") {
		t.Error("expected event log to retain events after clearing zone panel")
	}
}

func TestUpdateKeyScrollRouting(t *testing.T) {
	m := readyModel()

	// These should not panic on either tab.
	m = updateModel(m, tea.KeyPressMsg{Code: tea.KeyTab})
	updateModel(m, tea.KeyPressMsg{Code: tea.KeyUp})
	updateModel(m, tea.KeyPressMsg{Code: tea.KeyDown})

	m = updateModel(m, tea.KeyPressMsg{Code: '1'})
	updateModel(m, tea.KeyPressMsg{Code: 'k'})
	updateModel(m, tea.KeyPressMsg{Code: 'j'})
}

// ============================================
// Update: BulkEventMsg zone tests
// ============================================

func TestUpdateBulkEventZone(t *testing.T) {
	m := readyModel()

	bulkMsg := BulkEventMsg{
		{
			Type:      "zone",
			Timestamp: time.Now(),
			Data: &handlers.ZoneEventData{
				MapType: handlers.MapTypeIsland,
				Display: "Island — Farm",
			},
		},
	}

	m = updateModel(m, bulkMsg)

	// Zone tab should contain the transition.
	m = updateModel(m, tea.KeyPressMsg{Code: '2'})
	view := m.View().Content
	if !strings.Contains(view, "Island") {
		t.Error("expected 'Island' in zone panel view after zone event")
	}

	// Dashboard event log should also show the zone transition.
	m = updateModel(m, tea.KeyPressMsg{Code: '1'})
	dashView := m.View().Content
	if !strings.Contains(dashView, "Island") {
		t.Error("expected zone transition in dashboard event log")
	}
}

// ============================================
// Original keyboard input tests
// ============================================

func TestUpdateKeyQuit(t *testing.T) {
	m := New(nil, nil, nil)

	newModel, cmd := m.Update(tea.KeyPressMsg{Code: 'q'})
	m2 := newModel.(Model)

	if !m2.quitting {
		t.Error("expected quitting=true after 'q'")
	}
	if cmd == nil {
		t.Error("expected non-nil command (tea.Quit)")
	}
}

func TestUpdateKeyCtrlC(t *testing.T) {
	m := New(nil, nil, nil)

	newModel, cmd := m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	m2 := newModel.(Model)

	if !m2.quitting {
		t.Error("expected quitting=true after ctrl+c")
	}
	if cmd == nil {
		t.Error("expected non-nil command (tea.Quit)")
	}
}

func TestUpdateKeyDebugToggle(t *testing.T) {
	m := New(nil, nil, nil)

	if m.debug {
		t.Error("expected debug=false initially")
	}

	m = updateModel(m, tea.KeyPressMsg{Code: 'd'})
	if !m.debug {
		t.Error("expected debug=true after pressing 'd'")
	}

	m = updateModel(m, tea.KeyPressMsg{Code: 'd'})
	if m.debug {
		t.Error("expected debug=false after pressing 'd' again")
	}
}

func TestUpdateKeyFullNumbersToggle(t *testing.T) {
	m := New(nil, nil, nil)

	if m.fullNumbers {
		t.Error("expected fullNumbers=false initially")
	}

	m = updateModel(m, tea.KeyPressMsg{Code: 'f'})
	if !m.fullNumbers {
		t.Error("expected fullNumbers=true after pressing 'f'")
	}
}

// ============================================
// Debug event filter tests
// ============================================

func TestDebugFilterHidesNoisyByDefault(t *testing.T) {
	m := readyModel()
	if !m.debugHidden[events.EventMove] {
		t.Error("expected EventMove hidden by default")
	}
	if !m.debugHidden[events.EventLeave] {
		t.Error("expected EventLeave hidden by default")
	}
}

func TestDebugFilterCountsAndSkipsHidden(t *testing.T) {
	m := readyModel()
	// Move is hidden by default; SimpleFeedback is not.
	m = updateModel(m, BulkEventMsg{
		{Type: "debug", Timestamp: time.Now(), Data: events.EventMove},
		{Type: "debug", Timestamp: time.Now(), Data: events.EventMove},
		{Type: "debug", Timestamp: time.Now(), Data: events.EventSimpleFeedback},
	})

	if m.debugCounts[events.EventMove] != 2 {
		t.Errorf("expected Move counted 2, got %d", m.debugCounts[events.EventMove])
	}
	if m.debugCounts[events.EventSimpleFeedback] != 1 {
		t.Errorf("expected SimpleFeedback counted 1, got %d", m.debugCounts[events.EventSimpleFeedback])
	}
	// The hidden Move events must not appear in the rendered log view.
	view := m.View().Content
	if strings.Contains(view, "Move") {
		t.Error("hidden Move event leaked into the view")
	}
	if !strings.Contains(view, "SimpleFeedback") {
		t.Error("expected SimpleFeedback to be visible in the view")
	}
}

func TestDebugFilterToggleRevealsHidden(t *testing.T) {
	m := readyModel()
	m = updateModel(m, BulkEventMsg{
		{Type: "debug", Timestamp: time.Now(), Data: events.EventMove},
	})
	// Open filter, move cursor onto Move (first sorted code), toggle it visible.
	m = updateModel(m, tea.KeyPressMsg{Code: '/'})
	m = updateModel(m, tea.KeyPressMsg{Code: tea.KeySpace}) // toggle cursor row

	if m.debugHidden[events.EventMove] {
		t.Error("expected Move to be revealed after toggling")
	}
}

func TestDebugFilterKeyOpensOverlay(t *testing.T) {
	m := readyModel()
	if m.filterOpen {
		t.Error("expected filter closed initially")
	}
	m = updateModel(m, tea.KeyPressMsg{Code: '/'})
	if !m.filterOpen {
		t.Error("expected filter open after pressing '/'")
	}
	// The overlay should render its title.
	if !strings.Contains(m.View().Content, "Debug Event Filter") {
		t.Error("expected filter overlay title in view")
	}
	// Closing restores the normal view.
	m = updateModel(m, tea.KeyPressMsg{Code: '/'})
	if m.filterOpen {
		t.Error("expected filter closed after pressing '/' again")
	}
}

func TestScrollWindow(t *testing.T) {
	tests := []struct {
		name     string
		cursor   int
		n        int
		maxItems int
		wantS    int
		wantE    int
	}{
		{"fits entirely", 2, 5, 10, 0, 5},
		{"cursor near top", 0, 20, 5, 0, 5},
		{"cursor centered", 10, 20, 5, 8, 13},
		{"cursor near bottom", 19, 20, 5, 15, 20},
		{"exact fit", 4, 5, 5, 0, 5},
		{"single item window", 3, 10, 1, 3, 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, e := scrollWindow(tt.cursor, tt.n, tt.maxItems)
			if s != tt.wantS || e != tt.wantE {
				t.Errorf("scrollWindow(%d,%d,%d) = [%d,%d), want [%d,%d)",
					tt.cursor, tt.n, tt.maxItems, s, e, tt.wantS, tt.wantE)
			}
			// Cursor must always be within [start, end).
			if tt.cursor < s || tt.cursor >= e {
				t.Errorf("cursor %d outside window [%d,%d)", tt.cursor, s, e)
			}
			// Window must not exceed bounds.
			if s < 0 || e > tt.n {
				t.Errorf("window [%d,%d) out of bounds [0,%d)", s, e, tt.n)
			}
		})
	}
}

// ============================================
// Overlay tests
// ============================================

func TestOverlayPreservesBackground(t *testing.T) {
	// 5-line background, 1-line modal.
	bg := "aaaa\nbbbb\ncccc\ndddd\neeee"
	modal := "MOD"
	result := overlayText(bg, modal, 10, 5)

	lines := strings.Split(result, "\n")
	// Lines outside the modal row must be intact.
	if lines[0] != "aaaa" {
		t.Errorf("line 0 changed: %q", lines[0])
	}
	if lines[4] != "eeee" {
		t.Errorf("line 4 changed: %q", lines[4])
	}
	// The modal line must contain "MOD".
	merged := strings.Join(lines, "\n")
	if !strings.Contains(merged, "MOD") {
		t.Error("expected MOD in overlay result")
	}
	// Background content around the modal must still be visible.
	if !strings.Contains(merged, "bbb") {
		t.Error("expected background 'bbb' preserved around modal")
	}
}

func TestDisplayOffsetToByte(t *testing.T) {
	// Plain ASCII: byte offset == display offset.
	if got := displayOffsetToByte("abcdef", 3); got != 3 {
		t.Errorf("plain: got %d, want 3", got)
	}
	// With ANSI prefix: escape codes are zero-width.
	s := "\x1b[31mABC\x1b[0m"
	if got := displayOffsetToByte(s, 2); got != len("\x1b[31mAB") {
		t.Errorf("ansi: got %d, want %d", got, len("\x1b[31mAB"))
	}
	// Target beyond end: returns full length.
	if got := displayOffsetToByte("ab", 10); got != 2 {
		t.Errorf("overflow: got %d, want 2", got)
	}
	// Zero target: returns 0.
	if got := displayOffsetToByte("abc", 0); got != 0 {
		t.Errorf("zero: got %d, want 0", got)
	}
}

func TestUpdateKeyClear(t *testing.T) {
	m := readyModel()

	// Add an event first
	m = updateModel(m, BulkEventMsg{
		{Type: "info", Message: "test event", Timestamp: time.Now()},
	})

	// Press 'c' to clear
	m = updateModel(m, tea.KeyPressMsg{Code: 'c'})

	// View should not contain the event text after clear
	view := m.View().Content
	if strings.Contains(view, "test event") {
		t.Error("expected event log to be cleared after 'c'")
	}
}

func TestUpdateKeyReset(t *testing.T) {
	m := readyModel()

	// Add some stats via events
	m = updateModel(m, BulkEventMsg{
		{Type: "fame", Data: &handlers.FameEventData{Gained: 100, Total: 500, AllTime: 200},
			Timestamp: time.Now()},
	})

	// Press 'r' to reset
	m = updateModel(m, tea.KeyPressMsg{Code: 'r'})

	// Stats panel view should show 0 after reset
	statsView := m.statsPanel.View()
	if strings.Contains(statsView, "+200") {
		t.Error("expected stats reset to clear fame value")
	}
}

func TestUpdateKeyScroll(t *testing.T) {
	m := readyModel()

	// These should not panic
	updateModel(m, tea.KeyPressMsg{Code: tea.KeyUp})
	updateModel(m, tea.KeyPressMsg{Code: tea.KeyDown})
	updateModel(m, tea.KeyPressMsg{Code: 'k'})
	updateModel(m, tea.KeyPressMsg{Code: 'j'})
}

// ============================================
// Update: BulkEventMsg tests
// ============================================

func TestUpdateBulkEventFame(t *testing.T) {
	m := readyModel()

	bulkMsg := BulkEventMsg{
		{
			Type:      "fame",
			Data:      &handlers.FameEventData{Gained: 100, Total: 500, AllTime: 200},
			Timestamp: time.Now(),
		},
	}

	m = updateModel(m, bulkMsg)

	view := m.View().Content
	if !strings.Contains(view, "Fame") {
		t.Error("expected 'Fame' in stats panel after fame event")
	}
}

// TestUpdateBulkEventFameSetsToday verifies the event's Daily value flows into
// the stats panel's "today" row.
func TestUpdateBulkEventFameSetsToday(t *testing.T) {
	m := readyModel()

	bulkMsg := BulkEventMsg{
		{
			Type:      "fame",
			Data:      &handlers.FameEventData{Gained: 100, Total: 500, AllTime: 200, Daily: 12000},
			Timestamp: time.Now(),
		},
	}

	m = updateModel(m, bulkMsg)

	view := m.View().Content
	if !strings.Contains(view, "today") {
		t.Error("expected 'today' section in stats panel after fame event with Daily value")
	}
	if !strings.Contains(view, "12000") {
		t.Error("expected today fame value 12000 in stats panel")
	}
}

func TestUpdateBulkEventSilver(t *testing.T) {
	m := readyModel()

	bulkMsg := BulkEventMsg{
		{
			Type: "silver",
			Data: &handlers.SilverEventData{
				Amount:     500,
				Total:      1000,
				LootedBy:   "Player1",
				LootedFrom: "Mob",
			},
			Timestamp: time.Now(),
		},
	}

	m = updateModel(m, bulkMsg)

	view := m.View().Content
	if !strings.Contains(view, "Silver") {
		t.Error("expected 'Silver' in stats panel after silver event")
	}
}

func TestUpdateBulkEventKillDeath(t *testing.T) {
	m := readyModel()

	bulkMsg := BulkEventMsg{
		{
			Type:      "kill",
			Data:      &handlers.KillEventData{TotalKills: 1},
			Timestamp: time.Now(),
		},
		{
			Type:      "death",
			Data:      &handlers.DeathEventData{Victim: "Player1", Killer: "Player2"},
			Timestamp: time.Now(),
		},
		{
			Type:      "loot",
			Data:      &handlers.LootEventData{ItemName: "Sword", Quantity: 1},
			Timestamp: time.Now(),
		},
	}

	m = updateModel(m, bulkMsg)

	statsView := m.statsPanel.View()
	if !strings.Contains(statsView, "Kills") {
		t.Error("expected 'Kills' in stats panel")
	}
	if !strings.Contains(statsView, "Deaths") {
		t.Error("expected 'Deaths' in stats panel")
	}
	if !strings.Contains(statsView, "Loot") {
		t.Error("expected 'Loot' in stats panel")
	}
}

func TestUpdateBulkEventDungeonEnter(t *testing.T) {
	m := readyModel()

	bulkMsg := BulkEventMsg{
		{
			Type: "zone",
			Data: &handlers.ZoneEventData{
				MapType:  handlers.MapTypeRandomDungeon,
				Display:  "Random Dungeon",
				Previous: handlers.MapTypeUnknown,
			},
			Timestamp: time.Now(),
		},
	}

	m = updateModel(m, bulkMsg)

	// Zone events are passed to the zone panel; the transition should appear.
	zoneView := m.zonePanel.View()
	if !strings.Contains(zoneView, "Random Dungeon") {
		t.Error("expected 'Random Dungeon' in zone panel after zone event")
	}
}

// ============================================
// Update: StatsUpdateMsg tests
// ============================================

func TestUpdateStatsUpdate(t *testing.T) {
	m := New(nil, nil, nil)

	stats := photon.NewStats()
	stats.IncrPacketsReceived()
	stats.IncrEventsDecoded()

	m = updateModel(m, StatsUpdateMsg{Stats: stats})

	// StatsUpdateMsg should NOT change online status — that is owned by the
	// tick handler (polling svc.IsOnline()) and OnlineMsg.
	if m.statusBar.Online() {
		t.Error("expected online=false after stats update (online status is set by tick/OnlineMsg)")
	}
}

func TestUpdateOnlineMsg(t *testing.T) {
	m := New(nil, nil, nil)

	m = updateModel(m, OnlineMsg{Online: true})
	if !m.statusBar.Online() {
		t.Error("expected online=true after OnlineMsg")
	}

	m = updateModel(m, OnlineMsg{Online: false})
	if m.statusBar.Online() {
		t.Error("expected online=false after OnlineMsg{false}")
	}
}

func TestUpdateTotalStatsMsg(t *testing.T) {
	m := readyModel()

	m = updateModel(m, TotalStatsMsg{Fame: 500, Silver: 1000})

	statsView := m.statsPanel.View()
	if !strings.Contains(statsView, "Fame") {
		t.Error("expected Fame in stats after TotalStatsMsg")
	}
}

func TestUpdateTickMsg(t *testing.T) {
	// TickMsg with a nil svc: must not panic, returns a TickCmd.
	m := readyModel()
	_, cmd := m.Update(TickMsg(time.Now()))
	if cmd == nil {
		t.Error("expected non-nil command from TickMsg (TickCmd)")
	}
}

func TestUpdateTickMsgWithService(t *testing.T) {
	// TickMsg with a non-nil svc exercises the zone-refresh path.
	// backend.New() has a nil handler, so CurrentZone() returns zero-value
	// ZoneInfo — but the code path must not panic.
	svc := backend.New()
	m := New(svc, nil, nil)
	m = updateModel(m, tea.WindowSizeMsg{Width: 80, Height: 24})

	_, cmd := m.Update(TickMsg(time.Now()))
	if cmd == nil {
		t.Error("expected non-nil command from TickMsg")
	}

	// Zone display should be empty (zero-value ZoneInfo.DisplayString() == ""),
	// so no zone indicator should appear in the status bar.
	view := m.View().Content
	if strings.Contains(view, "◎") {
		t.Error("expected no zone indicator with nil handler")
	}
}

func TestUpdateWindowSizeMsg(t *testing.T) {
	m := New(nil, nil, nil)

	m = updateModel(m, tea.WindowSizeMsg{Width: 100, Height: 30})

	if !m.ready {
		t.Error("expected ready=true after WindowSizeMsg")
	}
	if m.width != 100 {
		t.Errorf("expected width=100, got %d", m.width)
	}
	if m.height != 30 {
		t.Errorf("expected height=30, got %d", m.height)
	}
}

// ============================================
// View tests
// ============================================

func TestViewQuitting(t *testing.T) {
	m := New(nil, nil, nil)
	m.quitting = true

	view := m.View().Content
	if !strings.Contains(view, "Goodbye!") {
		t.Errorf("expected 'Goodbye!' in view, got %q", view)
	}
}

func TestViewNotReady(t *testing.T) {
	m := New(nil, nil, nil)

	view := m.View().Content
	if !strings.Contains(view, "Initializing") {
		t.Errorf("expected 'Initializing' in view, got %q", view)
	}
}

func TestViewReady(t *testing.T) {
	m := readyModel()

	view := m.View().Content
	if strings.Contains(view, "Initializing") {
		t.Error("expected view to be past 'Initializing' state")
	}
	if !strings.Contains(view, "Events") {
		t.Error("expected 'Events' section in ready view")
	}
	if !strings.Contains(view, "Total Stats") {
		t.Error("expected 'Total Stats' section in ready view")
	}
}

func TestViewZoneTab(t *testing.T) {
	m := readyModel()

	// Switch to Zone tab.
	m = updateModel(m, tea.KeyPressMsg{Code: '2'})

	view := m.View().Content
	if !strings.Contains(view, "Zone") {
		t.Error("expected 'Zone' panel in zone tab view")
	}
	if !strings.Contains(view, "Current:") {
		t.Error("expected 'Current:' label in zone tab view")
	}
	if !strings.Contains(view, "Time:") {
		t.Error("expected 'Time:' label in zone tab view")
	}
}

func TestViewHelpBarTabIndicator(t *testing.T) {
	m := readyModel()

	// Dashboard active.
	helpDash := m.renderHelpBar()
	if !strings.Contains(helpDash, "[DASH]") {
		t.Error("expected [DASH] indicator on dashboard tab")
	}

	// Switch to Zone.
	m = updateModel(m, tea.KeyPressMsg{Code: tea.KeyTab})
	helpZone := m.renderHelpBar()
	if !strings.Contains(helpZone, "[ZONE]") {
		t.Error("expected [ZONE] indicator on zone tab")
	}
}

// ============================================
// BulkEventMsg warning count tests
// ============================================

// TestBulkEventWarningCountIncrement verifies that warning events emitted by
// the backend (e.g. when a capture device fails to open) increment the model's
// warning counter, which is then forwarded to the status bar.
func TestBulkEventWarningCountIncrement(t *testing.T) {
	m := readyModel()

	if m.warningCount != 0 {
		t.Fatalf("expected initial warningCount=0, got %d", m.warningCount)
	}

	bulkMsg := BulkEventMsg{
		{Type: "warning", Message: "Could not capture on eth0", Timestamp: time.Now()},
		{Type: "warning", Message: "Could not capture on wlan0", Timestamp: time.Now()},
		{Type: "fame", Message: "ignored", Timestamp: time.Now()},
	}

	m = updateModel(m, bulkMsg)

	if m.warningCount != 2 {
		t.Errorf("expected warningCount=2 after batch with 2 warnings, got %d", m.warningCount)
	}

	// Forwarding to status bar is verified indirectly: setting the model
	// online and rendering the dashboard view must surface the warning badge.
	m = updateModel(m, OnlineMsg{Online: true})
	dashView := m.View().Content
	if !strings.Contains(dashView, "Warnings") {
		t.Error("expected 'Warnings' badge in dashboard view after forwarding count to status bar")
	}
}
