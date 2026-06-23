package components

import (
	"strings"
	"testing"
	"time"

	"github.com/cantalupo555/albion-lens/pkg/handlers"
)

func TestNewZonePanelDefaults(t *testing.T) {
	z := NewZonePanel()
	if z.maxHistory != defaultMaxZoneHistory {
		t.Errorf("expected maxHistory=%d, got %d", defaultMaxZoneHistory, z.maxHistory)
	}
	if z.nowFunc == nil {
		t.Error("expected nowFunc to be set")
	}
}

func TestZonePanelSetSize(t *testing.T) {
	z := NewZonePanel()
	z = z.SetSize(80, 24)

	if z.width != 80 || z.height != 24 {
		t.Errorf("expected width=80 height=24, got width=%d height=%d", z.width, z.height)
	}
	if !z.ready {
		t.Error("expected ready=true after SetSize")
	}
}

func TestZonePanelAddTransition(t *testing.T) {
	z := NewZonePanel()
	z = z.SetSize(80, 24)

	info := handlers.ZoneInfo{MapType: handlers.MapTypeIsland, IslandName: "Farm"}
	ts := time.Now()
	z = z.AddTransition(info, ts)

	if z.current.MapType != handlers.MapTypeIsland {
		t.Errorf("expected current MapTypeIsland, got %v", z.current.MapType)
	}
	if len(z.history) != 1 {
		t.Errorf("expected 1 history entry, got %d", len(z.history))
	}
	if !z.enteredAt.Equal(ts) {
		t.Error("enteredAt should match the transition timestamp")
	}
}

func TestZonePanelAddTransitionTrims(t *testing.T) {
	z := NewZonePanel()
	z.maxHistory = 3
	z = z.SetSize(80, 24)

	for i := 0; i < 5; i++ {
		z = z.AddTransition(handlers.ZoneInfo{
			MapType:      handlers.MapTypeRandomDungeon,
			ClusterIndex: "dungeon",
		}, time.Now())
	}

	if len(z.history) != 3 {
		t.Errorf("expected history trimmed to 3, got %d", len(z.history))
	}
}

func TestZonePanelClear(t *testing.T) {
	z := NewZonePanel()
	z = z.SetSize(80, 24)

	z = z.AddTransition(handlers.ZoneInfo{MapType: handlers.MapTypeArena}, time.Now())
	if len(z.history) != 1 {
		t.Fatalf("expected 1 entry before clear, got %d", len(z.history))
	}

	z = z.Clear()

	if len(z.history) != 0 {
		t.Errorf("expected 0 entries after clear, got %d", len(z.history))
	}
}

func TestZonePanelSetZone(t *testing.T) {
	z := NewZonePanel()
	info := handlers.ZoneInfo{MapType: handlers.MapTypeMists}

	z = z.SetZone(info)

	if z.current.MapType != handlers.MapTypeMists {
		t.Errorf("expected current MapTypeMists, got %v", z.current.MapType)
	}
	// SetZone must not add to history or change enteredAt.
	if len(z.history) != 0 {
		t.Errorf("SetZone must not add to history, got %d", len(z.history))
	}
}

func TestZonePanelViewEmpty(t *testing.T) {
	z := NewZonePanel()
	z = z.SetSize(80, 24)

	view := z.View()

	if !strings.Contains(view, "Zone") {
		t.Error("expected 'Zone' title in view")
	}
	if !strings.Contains(view, "Unknown") {
		t.Error("expected 'Unknown' for unset zone")
	}
}

func TestZonePanelViewWithZone(t *testing.T) {
	z := NewZonePanel()
	z = z.SetSize(80, 24)

	z = z.AddTransition(handlers.ZoneInfo{
		MapType:    handlers.MapTypeIsland,
		IslandName: "Farm",
	}, time.Now())

	view := z.View()

	if !strings.Contains(view, "Island") {
		t.Error("expected 'Island' in view")
	}
}

func TestZonePanelViewElapsed(t *testing.T) {
	z := NewZonePanel()
	z.nowFunc = func() time.Time { return time.Unix(100, 0) }
	z = z.SetSize(80, 24)

	z = z.AddTransition(handlers.ZoneInfo{MapType: handlers.MapTypeArena}, time.Unix(90, 0))

	view := z.View()

	// 10 seconds elapsed → should show "10s" somewhere.
	if !strings.Contains(view, "10s") {
		t.Errorf("expected elapsed '10s' in view, got: %s", view)
	}
}

func TestZonePanelViewElapsedNotEntered(t *testing.T) {
	z := NewZonePanel()
	z = z.SetSize(80, 24)

	view := z.View()

	if !strings.Contains(view, "—") {
		t.Error("expected '—' for elapsed when no zone entered yet")
	}
}

func TestFormatZoneEntryEmptyDisplay(t *testing.T) {
	// A zero-value ZoneInfo has DisplayString()==""; formatZoneEntry must
	// substitute "Unknown" so the history line is never blank.
	entry := zoneEntry{Zone: handlers.ZoneInfo{}, Timestamp: time.Unix(100, 0)}
	got := formatZoneEntry(entry)
	if !strings.Contains(got, "Unknown") {
		t.Errorf("expected 'Unknown' fallback for empty DisplayString, got %q", got)
	}
}

func TestZonePanelResizeAfterReady(t *testing.T) {
	z := NewZonePanel()
	z = z.SetSize(80, 24)
	z = z.AddTransition(handlers.ZoneInfo{MapType: handlers.MapTypeArena}, time.Now())

	// Resize to a different size — must not panic and must stay ready.
	z = z.SetSize(100, 30)
	if !z.ready {
		t.Error("expected ready=true after resize")
	}
	if z.width != 100 || z.height != 30 {
		t.Errorf("expected 100x30, got %dx%d", z.width, z.height)
	}
	// History must survive resize.
	if len(z.history) != 1 {
		t.Errorf("expected 1 history entry after resize, got %d", len(z.history))
	}

	view := z.View()
	if !strings.Contains(view, "Arena") {
		t.Error("expected 'Arena' in view after resize")
	}
}

func TestZonePanelViewMultiEntry(t *testing.T) {
	z := NewZonePanel()
	z = z.SetSize(80, 24)

	z = z.AddTransition(handlers.ZoneInfo{MapType: handlers.MapTypeArena}, time.Unix(100, 0))
	z = z.AddTransition(handlers.ZoneInfo{MapType: handlers.MapTypeIsland, IslandName: "Farm"}, time.Unix(200, 0))
	z = z.AddTransition(handlers.ZoneInfo{MapType: handlers.MapTypeMists}, time.Unix(300, 0))

	view := z.View()

	// All three zone labels should appear in the history viewport.
	for _, want := range []string{"Arena", "Island", "Mists"} {
		if !strings.Contains(view, want) {
			t.Errorf("expected %q in multi-entry view", want)
		}
	}
	// The latest zone should be the current zone.
	if z.current.MapType != handlers.MapTypeMists {
		t.Errorf("expected current=Mists, got %v", z.current.MapType)
	}
}

func TestZonePanelSetNow(t *testing.T) {
	z := NewZonePanel()
	z = z.SetSize(80, 24)
	z = z.AddTransition(handlers.ZoneInfo{MapType: handlers.MapTypeArena}, time.Unix(100, 0))

	// SetNow overrides the elapsed-time reference.
	z = z.SetNow(time.Unix(160, 0))

	view := z.View()
	if !strings.Contains(view, "1m") {
		t.Errorf("expected '1m' in view with SetNow(160) - enteredAt(100), got: %s", view)
	}
}

func TestZonePanelIncrementalRenderCache(t *testing.T) {
	// After incremental AddTransition calls, renderedLines must be in sync
	// with history — not rebuilt from scratch.
	z := NewZonePanel()
	z.maxHistory = 3
	z = z.SetSize(80, 24)

	for i := 0; i < 5; i++ {
		z = z.AddTransition(handlers.ZoneInfo{
			MapType:      handlers.MapTypeRandomDungeon,
			ClusterIndex: "d",
		}, time.Unix(int64(100+i), 0))
	}

	if len(z.history) != 3 {
		t.Fatalf("expected 3 history entries after trim, got %d", len(z.history))
	}
	if len(z.renderedLines) != 3 {
		t.Errorf("expected 3 rendered lines after trim, got %d", len(z.renderedLines))
	}
}

func TestFormatElapsed(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{0, "0s"},
		{5 * time.Second, "5s"},
		{90 * time.Second, "1m 30s"},
		{3700 * time.Second, "1h 1m 40s"},
		{-1 * time.Second, "0s"},
	}
	for _, tt := range tests {
		if got := formatElapsed(tt.d); got != tt.want {
			t.Errorf("formatElapsed(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}
