package components

import (
	"strings"
	"testing"
	"time"

	"github.com/cantalupo555/albion-lens/pkg/photon"
)

// ============================================
// NewStatusBar tests
// ============================================

func TestNewStatusBarDefaults(t *testing.T) {
	s := NewStatusBar()

	if s.online {
		t.Error("expected online=false by default")
	}
	if s.uptime != "00:00:00" {
		t.Errorf("expected uptime='00:00:00', got %q", s.uptime)
	}
	if s.packetsTotal != 0 {
		t.Errorf("expected packetsTotal=0, got %d", s.packetsTotal)
	}
}

// ============================================
// SetOnline tests
// ============================================

func TestStatusBarSetOnline(t *testing.T) {
	s := NewStatusBar()

	s = s.SetOnline(true)
	if !s.Online() {
		t.Error("expected online=true after SetOnline(true)")
	}

	s = s.SetOnline(false)
	if s.Online() {
		t.Error("expected online=false after SetOnline(false)")
	}
}

// ============================================
// SetWidth tests
// ============================================

func TestStatusBarSetWidth(t *testing.T) {
	s := NewStatusBar()
	s = s.SetWidth(120)

	if s.width != 120 {
		t.Errorf("expected width=120, got %d", s.width)
	}
}

// ============================================
// UpdateStats tests
// ============================================

func TestStatusBarUpdateStatsNil(t *testing.T) {
	s := NewStatusBar()
	s = s.SetOnline(true)
	original := s

	s = s.UpdateStats(nil)

	if s.packetsTotal != original.packetsTotal {
		t.Error("expected no change with nil stats")
	}
}

func TestStatusBarUpdateStats(t *testing.T) {
	stats := &photon.Stats{
		PacketsReceived:   uint64(100),
		EventsDecoded:     uint64(50),
		EventsDropped:     uint64(5),
		BufferPeakDisplay: 30,
		BufferCapacity:    200,
		StartTime:         time.Now(),
	}

	s := NewStatusBar()
	s = s.UpdateStats(stats)

	if s.packetsTotal != 100 {
		t.Errorf("expected packetsTotal=100, got %d", s.packetsTotal)
	}
	if s.eventsDecoded != 50 {
		t.Errorf("expected eventsDecoded=50, got %d", s.eventsDecoded)
	}
	if s.eventsDropped != 5 {
		t.Errorf("expected eventsDropped=5, got %d", s.eventsDropped)
	}
	if s.bufferUsage != 30 {
		t.Errorf("expected bufferUsage=30, got %d", s.bufferUsage)
	}
	if s.bufferCapacity != 200 {
		t.Errorf("expected bufferCapacity=200, got %d", s.bufferCapacity)
	}
}

// ============================================
// View tests
// ============================================

func TestStatusBarViewOffline(t *testing.T) {
	s := NewStatusBar()
	s = s.SetWidth(80)

	output := s.View()

	if !strings.Contains(output, "Offline") {
		t.Error("expected 'Offline' in view")
	}
}

func TestStatusBarViewOnline(t *testing.T) {
	s := NewStatusBar()
	s = s.SetWidth(80)
	s = s.SetOnline(true)

	output := s.View()

	if !strings.Contains(output, "Online") {
		t.Error("expected 'Online' in view")
	}
}

func TestStatusBarViewDroppedEvents(t *testing.T) {
	stats := &photon.Stats{
		EventsDecoded: uint64(100),
		EventsDropped: uint64(5),
		StartTime:     time.Now(),
	}

	s := NewStatusBar()
	s = s.SetWidth(100)
	s = s.SetOnline(true)
	s = s.UpdateStats(stats)

	output := s.View()

	if !strings.Contains(output, "Dropped") {
		t.Error("expected 'Dropped' warning in view when events dropped > 0")
	}
}

func TestStatusBarViewNoDroppedEvents(t *testing.T) {
	stats := &photon.Stats{
		EventsDecoded: uint64(100),
		EventsDropped: uint64(0),
		StartTime:     time.Now(),
	}

	s := NewStatusBar()
	s = s.SetWidth(100)
	s = s.SetOnline(true)
	s = s.UpdateStats(stats)

	output := s.View()

	if strings.Contains(output, "Dropped") {
		t.Error("expected no 'Dropped' warning when events dropped = 0")
	}
}

func TestStatusBarViewBufferUsage(t *testing.T) {
	stats := &photon.Stats{
		BufferPeakDisplay: 50,
		BufferCapacity:    100,
		StartTime:         time.Now(),
	}

	s := NewStatusBar()
	s = s.SetWidth(100)
	s = s.SetOnline(true)
	s = s.UpdateStats(stats)

	output := s.View()

	if !strings.Contains(output, "Queue") {
		t.Error("expected 'Queue' in view when bufferCapacity > 0")
	}
	if !strings.Contains(output, "50/100") {
		t.Error("expected '50/100' buffer usage in view")
	}
}

func TestStatusBarViewNoBufferUsage(t *testing.T) {
	s := NewStatusBar()
	s = s.SetWidth(100)

	output := s.View()

	if strings.Contains(output, "Queue") {
		t.Error("expected no 'Queue' section when bufferCapacity = 0")
	}
}

// ============================================
// SetZone tests
// ============================================

func TestStatusBarSetZone(t *testing.T) {
	s := NewStatusBar()
	s = s.SetZone("Random Dungeon")

	if s.currentZone != "Random Dungeon" {
		t.Errorf("expected currentZone='Random Dungeon', got %q", s.currentZone)
	}
}

func TestStatusBarViewZoneIndicator(t *testing.T) {
	s := NewStatusBar()
	s = s.SetWidth(100)
	s = s.SetOnline(true)
	s = s.SetZone("Random Dungeon")

	output := s.View()

	if !strings.Contains(output, "Random Dungeon") {
		t.Error("expected zone label in view")
	}
}

func TestStatusBarViewNoZoneWhenEmpty(t *testing.T) {
	s := NewStatusBar()
	s = s.SetWidth(100)
	s = s.SetOnline(true)

	output := s.View()

	if strings.Contains(output, "◎") {
		t.Error("expected no zone indicator when zone is empty")
	}
}

func TestStatusBarViewNoZoneWhenOffline(t *testing.T) {
	s := NewStatusBar()
	s = s.SetWidth(100)
	s = s.SetZone("Random Dungeon")

	output := s.View()

	if strings.Contains(output, "Random Dungeon") {
		t.Error("expected no zone indicator when offline")
	}
}

// ============================================
// SetWarningCount tests
// ============================================

func TestStatusBarSetWarningCount(t *testing.T) {
	s := NewStatusBar()
	s = s.SetWarningCount(3)

	if s.warningCount != 3 {
		t.Errorf("expected warningCount=3, got %d", s.warningCount)
	}
}

func TestStatusBarViewWarningShownWhenCount(t *testing.T) {
	s := NewStatusBar()
	s = s.SetWidth(120)
	s = s.SetOnline(true)
	s = s.SetWarningCount(2)

	output := s.View()

	if !strings.Contains(output, "Warnings") {
		t.Error("expected 'Warnings' badge in view when warningCount > 0")
	}
	if !strings.Contains(output, "2") {
		t.Error("expected warning count '2' in view")
	}
}

func TestStatusBarViewNoWarningWhenZero(t *testing.T) {
	s := NewStatusBar()
	s = s.SetWidth(120)
	s = s.SetOnline(true)
	// warningCount defaults to 0

	output := s.View()

	if strings.Contains(output, "Warnings") {
		t.Error("expected no 'Warnings' badge when warningCount = 0")
	}
}

// TestStatusBarViewHidesWarningWhenOffline pins the intended boundary: the
// warning badge lives in the online-only stats line. When offline, the status
// bar shows the waiting message instead, so the warning counter must not leak
// through. (Warnings are still visible in the event log regardless of status.)
func TestStatusBarViewHidesWarningWhenOffline(t *testing.T) {
	s := NewStatusBar()
	s = s.SetWidth(120)
	s = s.SetOnline(false)
	s = s.SetWarningCount(5)

	output := s.View()

	if strings.Contains(output, "Warnings") {
		t.Error("expected warning badge to be hidden when offline")
	}
	if !strings.Contains(output, "Waiting for Albion Online traffic...") {
		t.Error("expected offline waiting message in view")
	}
}
