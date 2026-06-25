package components

import (
	"strings"
	"testing"
	"time"

	"github.com/cantalupo555/albion-lens/pkg/handlers"
)

func TestNewDungeonsPanelDefaults(t *testing.T) {
	d := NewDungeonsPanel()
	if d.ready {
		t.Error("expected ready=false initially")
	}
	if d.fullNumbers {
		t.Error("expected fullNumbers=false by default")
	}
	if d.nowFunc == nil {
		t.Error("expected nowFunc to be set")
	}
}

func TestDungeonsPanelViewNoActiveRun(t *testing.T) {
	d := NewDungeonsPanel().
		SetSize(80, 24).
		SetNow(time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC))

	view := d.View()

	if !strings.Contains(view, "No active dungeon run") {
		t.Error("expected 'No active dungeon run' in view when no active run")
	}
}

func TestDungeonsPanelViewWithActiveRun(t *testing.T) {
	t0 := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
	run := &handlers.DungeonRun{
		EnteredAt: t0,
		CloseAt:   t0.Add(handlers.DungeonCloseTimeout),
		Mode:      handlers.DungeonModeSolo,
		Faction:   "Keeper",
		Tier:      3,
		Status:    handlers.RunStatusActive,
	}

	d := NewDungeonsPanel().
		SetSize(80, 24).
		SetNow(t0.Add(30*time.Second)).
		SetRuns(nil, run)

	view := d.View()

	if !strings.Contains(view, "Active:") {
		t.Error("expected 'Active:' label in view")
	}
	if !strings.Contains(view, "Solo") {
		t.Error("expected 'Solo' mode in view")
	}
	if !strings.Contains(view, "Close in:") {
		t.Error("expected 'Close in:' countdown in view")
	}
}

func TestDungeonsPanelCountdownAtZero(t *testing.T) {
	t0 := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
	run := &handlers.DungeonRun{
		EnteredAt: t0,
		CloseAt:   t0.Add(handlers.DungeonCloseTimeout),
		Status:    handlers.RunStatusActive,
	}

	d := NewDungeonsPanel().
		SetSize(80, 24).
		SetNow(t0.Add(handlers.DungeonCloseTimeout+5*time.Second)).
		SetRuns(nil, run)

	view := d.View()

	if !strings.Contains(view, "Close in:") {
		t.Error("expected 'Close in:' label")
	}
	if !strings.Contains(view, "0s") {
		t.Error("expected countdown clamped to 0s")
	}
}

func TestDungeonsPanelRunTableDoneRun(t *testing.T) {
	t0 := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
	run := &handlers.DungeonRun{
		EnteredAt: t0,
		ExitedAt:  t0.Add(5 * time.Minute),
		Mode:      handlers.DungeonModeStandard,
		Faction:   "Heretic",
		Tier:      4,
		Status:    handlers.RunStatusDone,
		Fame:      15000,
		Kills:     2,
		Loot:      3,
	}

	d := NewDungeonsPanel().
		SetSize(80, 24).
		SetNow(t0.Add(10*time.Minute)).
		SetRuns([]*handlers.DungeonRun{run}, nil)

	view := d.View()

	if !strings.Contains(view, "Group") {
		t.Error("expected 'Group' mode in run table")
	}
	if !strings.Contains(view, "Heretic") {
		t.Error("expected 'Heretic' faction in run table")
	}
	if !strings.Contains(view, "T5") {
		t.Error("expected 'T5' tier (tier 4 → T5) in run table")
	}
}

func TestDungeonsPanelRunTableExcludesActiveRun(t *testing.T) {
	t0 := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
	activeRun := &handlers.DungeonRun{
		EnteredAt: t0,
		CloseAt:   t0.Add(handlers.DungeonCloseTimeout),
		Mode:      handlers.DungeonModeSolo,
		Status:    handlers.RunStatusActive,
	}
	doneRun := &handlers.DungeonRun{
		EnteredAt: t0.Add(-10 * time.Minute),
		ExitedAt:  t0.Add(-5 * time.Minute),
		Mode:      handlers.DungeonModeStandard,
		Status:    handlers.RunStatusDone,
	}

	d := NewDungeonsPanel().
		SetSize(80, 24).
		SetNow(t0).
		SetRuns([]*handlers.DungeonRun{doneRun, activeRun}, activeRun)

	view := d.View()

	if strings.Count(view, "Solo") > 1 {
		t.Error("active run (Solo) should not appear in the run table, only in the info block")
	}
}

func TestDungeonsPanelEmptyRunTable(t *testing.T) {
	d := NewDungeonsPanel().
		SetSize(80, 24).
		SetNow(time.Now()).
		SetRuns(nil, nil)

	view := d.View()

	if !strings.Contains(view, "No dungeon runs yet") {
		t.Error("expected empty-state message in run table")
	}
}

func TestDungeonsPanelClear(t *testing.T) {
	t0 := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
	run := &handlers.DungeonRun{
		EnteredAt: t0,
		Status:    handlers.RunStatusDone,
	}

	d := NewDungeonsPanel().
		SetSize(80, 24).
		SetNow(t0).
		SetRuns([]*handlers.DungeonRun{run}, nil)

	d = d.Clear()
	view := d.View()

	if !strings.Contains(view, "No dungeon runs yet") {
		t.Error("expected empty run table after Clear")
	}
	if !strings.Contains(view, "No active dungeon run") {
		t.Error("expected no active run after Clear")
	}
}

func TestDungeonsPanelFormatRunFields(t *testing.T) {
	t0 := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name         string
		run          *handlers.DungeonRun
		wantMode     string
		wantFaction  string
		wantTier     string
		wantDuration string
	}{
		{
			name: "solo keeper T4 active",
			run: &handlers.DungeonRun{
				EnteredAt: t0,
				CloseAt:   t0.Add(handlers.DungeonCloseTimeout),
				Mode:      handlers.DungeonModeSolo,
				Faction:   "Keeper",
				Tier:      3,
				Status:    handlers.RunStatusActive,
			},
			wantMode:     "Solo",
			wantFaction:  "Keeper",
			wantTier:     "T4",
			wantDuration: "5m 0s",
		},
		{
			name: "unknown mode no faction",
			run: &handlers.DungeonRun{
				EnteredAt: t0,
				Mode:      handlers.DungeonModeUnknown,
				Tier:      -1,
				Status:    handlers.RunStatusActive,
			},
			wantMode:     "Unknown",
			wantFaction:  "—",
			wantTier:     "—",
			wantDuration: "5m 0s",
		},
		{
			name: "done run with exit time",
			run: &handlers.DungeonRun{
				EnteredAt: t0,
				ExitedAt:  t0.Add(3 * time.Minute),
				Mode:      handlers.DungeonModeAvalon,
				Faction:   "Avalon",
				Tier:      7,
				Status:    handlers.RunStatusDone,
			},
			wantMode:     "Avalon",
			wantFaction:  "Avalon",
			wantTier:     "T8",
			wantDuration: "3m 0s",
		},
	}

	d := NewDungeonsPanel().SetNow(t0.Add(5 * time.Minute))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mode, faction, tier, dur := d.formatRunFields(tt.run, t0.Add(5*time.Minute))
			if mode != tt.wantMode {
				t.Errorf("mode = %q, want %q", mode, tt.wantMode)
			}
			if faction != tt.wantFaction {
				t.Errorf("faction = %q, want %q", faction, tt.wantFaction)
			}
			if tier != tt.wantTier {
				t.Errorf("tier = %q, want %q", tier, tt.wantTier)
			}
			if dur != tt.wantDuration {
				t.Errorf("duration = %q, want %q", dur, tt.wantDuration)
			}
		})
	}
}

func TestDungeonsPanelSetSizeSmallDimensions(t *testing.T) {
	d := NewDungeonsPanel().SetSize(10, 5)
	if !d.ready {
		t.Error("expected ready=true after SetSize")
	}

	view := d.View()
	if view == "" {
		t.Error("expected non-empty view even with small dimensions")
	}
}
