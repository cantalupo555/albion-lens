package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cantalupo555/albion-lens/internal/tui/components"
	"github.com/cantalupo555/albion-lens/pkg/events"
	"github.com/cantalupo555/albion-lens/pkg/handlers"
)

// TestEventToBucketToPanel is an integration check that the full runtime
// path works end to end: a game event populates today's daily bucket, and the
// stats panel (hydrated as the TUI does on startup) renders the "today"
// section with the accumulated value. The per-package unit tests cover each
// layer in isolation; this guards the wiring between them.
func TestEventToBucketToPanel(t *testing.T) {
	h := handlers.NewAlbionHandler()
	h.SetLocalPlayer("Hero")

	// Feed a fame event (EventUpdateFame, FixPoint): gained 1000.
	h.OnEvent(0, map[byte]interface{}{
		1:                     int64(50000000000), // total fame (FixPoint)
		2:                     int64(10000000),    // gained 1000 (FixPoint)
		events.ParamEventCode: int16(events.EventUpdateFame),
	})

	// The fame must land in today's daily bucket.
	now := time.Now()
	snap := h.TotalSnapshot()
	today := snap.Daily[now.Format("2006-01-02")]
	if today.Fame != 1000 {
		t.Fatalf("today bucket fame = %d, want 1000", today.Fame)
	}

	// Hydrate the panel as the TUI does, then render.
	panel := components.NewStatsPanel().
		SetFullNumbers(true).
		SetFame(snap.Fame).
		SetKills(int(snap.Kills)).
		SetTodayFame(today.Fame).
		SetSize(40, 30)
	out := panel.View()
	for _, want := range []string{"Total Stats", "today", "1000"} {
		if !strings.Contains(out, want) {
			t.Errorf("panel view missing %q\n%s", want, out)
		}
	}

	// Negative control: a panel with no today values must hide the section.
	empty := components.NewStatsPanel().
		SetFullNumbers(true).
		SetFame(5000).
		SetSize(40, 30)
	if strings.Contains(empty.View(), "today") {
		t.Errorf("panel with zero today values should not render 'today' section")
	}
}

// TestTotalStatsPersistenceLifecycle exercises the on-disk lifecycle the TUI
// orchestrates (main.go): a v1 file (pre-bucketing) loads with empty buckets,
// cumulative values survive a save->reload round-trip, and a corrupt file
// errors instead of crashing. Guards the real persistence path the unit tests
// touch only in pieces.
func TestTotalStatsPersistenceLifecycle(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session-stats.json")

	// 1. A v1 file (no daily/hourly) must load with empty buckets, cumulative
	//    intact — the upgrade path for existing users.
	v1 := `{"version":1,"fame":100,"silver":200,"respec":30,"respec_silver":8,"kills":1,"deaths":0,"loot":2,"saved_at":"2026-06-29T00:00:00Z"}`
	if err := os.WriteFile(path, []byte(v1), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	h := handlers.NewAlbionHandler()
	if err := h.LoadTotalStats(path); err != nil {
		t.Fatalf("LoadTotalStats v1: %v", err)
	}
	if h.GetTotalFame() != 100 || h.GetTotalSilver() != 200 || h.GetTotalKills() != 1 {
		t.Errorf("v1 load: Fame=%d Silver=%d Kills=%d, want 100/200/1",
			h.GetTotalFame(), h.GetTotalSilver(), h.GetTotalKills())
	}
	snap := h.TotalSnapshot()
	if len(snap.Daily) != 0 || len(snap.Hourly) != 0 {
		t.Errorf("v1 load should yield empty buckets, got daily=%d hourly=%d",
			len(snap.Daily), len(snap.Hourly))
	}

	// 2. Save (now v2) and reload — cumulative values must round-trip.
	out := filepath.Join(dir, "out.json")
	if err := h.SaveTotalStats(out); err != nil {
		t.Fatalf("SaveTotalStats: %v", err)
	}
	fresh := handlers.NewAlbionHandler()
	if err := fresh.LoadTotalStats(out); err != nil {
		t.Fatalf("reload: %v", err)
	}
	if fresh.GetTotalFame() != 100 || fresh.GetTotalSilver() != 200 {
		t.Errorf("round-trip: Fame=%d Silver=%d, want 100/200",
			fresh.GetTotalFame(), fresh.GetTotalSilver())
	}

	// 3. A corrupt file must error and not crash the app.
	bad := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(bad, []byte("{broken"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := handlers.NewAlbionHandler().LoadTotalStats(bad); err == nil {
		t.Error("expected error loading corrupt file, got nil")
	}
}
