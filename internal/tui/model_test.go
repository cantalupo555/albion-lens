package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/cantalupo555/albion-lens/pkg/handlers"
	"github.com/cantalupo555/albion-lens/pkg/photon"
	tea "github.com/charmbracelet/bubbletea"
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

// ============================================
// Update: keyboard input tests
// ============================================

func TestUpdateKeyQuit(t *testing.T) {
	m := New(nil, nil, nil)

	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
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

	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
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

	m = updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if !m.debug {
		t.Error("expected debug=true after pressing 'd'")
	}

	m = updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if m.debug {
		t.Error("expected debug=false after pressing 'd' again")
	}
}

func TestUpdateKeyFullNumbersToggle(t *testing.T) {
	m := New(nil, nil, nil)

	if m.fullNumbers {
		t.Error("expected fullNumbers=false initially")
	}

	m = updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	if !m.fullNumbers {
		t.Error("expected fullNumbers=true after pressing 'f'")
	}
}

func TestUpdateKeyClear(t *testing.T) {
	m := readyModel()

	// Add an event first
	m = updateModel(m, BulkEventMsg{
		{Type: "info", Message: "test event", Timestamp: time.Now()},
	})

	// Press 'c' to clear
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})

	// View should not contain the event text after clear
	view := m.View()
	if strings.Contains(view, "test event") {
		t.Error("expected event log to be cleared after 'c'")
	}
}

func TestUpdateKeyReset(t *testing.T) {
	m := readyModel()

	// Add some stats via events
	m = updateModel(m, BulkEventMsg{
		{Type: "fame", Data: &handlers.FameEventData{Gained: 100, Total: 500, Session: 200},
			Timestamp: time.Now()},
	})

	// Press 'r' to reset
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})

	// Stats panel view should show 0 after reset
	statsView := m.statsPanel.View()
	if strings.Contains(statsView, "+200") {
		t.Error("expected stats reset to clear fame value")
	}
}

func TestUpdateKeyScroll(t *testing.T) {
	m := readyModel()

	// These should not panic
	updateModel(m, tea.KeyMsg{Type: tea.KeyUp})
	updateModel(m, tea.KeyMsg{Type: tea.KeyDown})
	updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
}

// ============================================
// Update: BulkEventMsg tests
// ============================================

func TestUpdateBulkEventFame(t *testing.T) {
	m := readyModel()

	bulkMsg := BulkEventMsg{
		{
			Type:      "fame",
			Data:      &handlers.FameEventData{Gained: 100, Total: 500, Session: 200},
			Timestamp: time.Now(),
		},
	}

	m = updateModel(m, bulkMsg)

	view := m.View()
	if !strings.Contains(view, "Fame") {
		t.Error("expected 'Fame' in stats panel after fame event")
	}
}

func TestUpdateBulkEventSilver(t *testing.T) {
	m := readyModel()

	bulkMsg := BulkEventMsg{
		{
			Type: "silver",
			Data: &handlers.SilverEventData{
				Amount:     500,
				Session:    1000,
				LootedBy:   "Player1",
				LootedFrom: "Mob",
			},
			Timestamp: time.Now(),
		},
	}

	m = updateModel(m, bulkMsg)

	view := m.View()
	if !strings.Contains(view, "Silver") {
		t.Error("expected 'Silver' in stats panel after silver event")
	}
}

func TestUpdateBulkEventKillDeath(t *testing.T) {
	m := readyModel()

	bulkMsg := BulkEventMsg{
		{
			Type:      "kill",
			Data:      &handlers.KillEventData{SessionKills: 1},
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

func TestUpdateSessionStatsMsg(t *testing.T) {
	m := readyModel()

	m = updateModel(m, SessionStatsMsg{Fame: 500, Silver: 1000})

	statsView := m.statsPanel.View()
	if !strings.Contains(statsView, "Fame") {
		t.Error("expected Fame in stats after SessionStatsMsg")
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

	view := m.View()
	if !strings.Contains(view, "Goodbye!") {
		t.Errorf("expected 'Goodbye!' in view, got %q", view)
	}
}

func TestViewNotReady(t *testing.T) {
	m := New(nil, nil, nil)

	view := m.View()
	if !strings.Contains(view, "Initializing") {
		t.Errorf("expected 'Initializing' in view, got %q", view)
	}
}

func TestViewReady(t *testing.T) {
	m := readyModel()

	view := m.View()
	if strings.Contains(view, "Initializing") {
		t.Error("expected view to be past 'Initializing' state")
	}
	if !strings.Contains(view, "Events") {
		t.Error("expected 'Events' section in ready view")
	}
	if !strings.Contains(view, "Session Stats") {
		t.Error("expected 'Session Stats' section in ready view")
	}
}
