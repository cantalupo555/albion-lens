package handlers

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cantalupo555/albion-lens/internal/storage"
)

func TestClassifyDungeonMode(t *testing.T) {
	cases := []struct {
		name string
		want DungeonMode
	}{
		{"GENERAL_SHRINE_COMBAT_BUFF", DungeonModeSolo},
		{"KEEPER_SOLO_BOOKCHEST_STANDARD", DungeonModeSolo},
		{"T4_VETERAN_LOOTCHEST_STANDARD", DungeonModeStandard},
		{"T4_HALLOWEEN_CHEST", DungeonModeStandard},
		{"AVALONIAN_CACHE", DungeonModeAvalon},
		{"CORRUPTED_SHRINE", DungeonModeCorrupted},
		{"HELLGATE_CHEST", DungeonModeHellGate},
		{"HD_SHRINE_WRATH_BUFF", DungeonModeAbyssalDepths},
		{"SOME_UNKNOWN_CHEST", DungeonModeUnknown},
		{"", DungeonModeUnknown},
	}
	for _, tc := range cases {
		got := ClassifyDungeonMode(tc.name)
		if got != tc.want {
			t.Errorf("ClassifyDungeonMode(%q) = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestClassifyDungeonFaction(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"KEEPER_SOLO_BOOKCHEST_STANDARD", "Keeper"},
		{"HERETIC_CHEST", "Heretic"},
		{"MORGANA_CHEST", "Morgana"},
		{"UNDEAD_SHRINE", "Undead"},
		{"AVALON_CACHE", "Avalon"},
		{"UNKNOWN_FACTION", ""},
		{"", ""},
	}
	for _, tc := range cases {
		got := ClassifyDungeonFaction(tc.name)
		if got != tc.want {
			t.Errorf("ClassifyDungeonFaction(%q) = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestEnterExitDungeonLifecycle(t *testing.T) {
	h := NewAlbionHandler()
	base := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
	h.nowFunc = func() time.Time { return base }

	// No active run initially
	if h.GetActiveDungeon() != nil {
		t.Fatal("expected nil active run before entry")
	}

	// Enter
	h.EnterDungeon(base)
	active := h.GetActiveDungeon()
	if active == nil {
		t.Fatal("expected active run after EnterDungeon")
	}
	if active.Status != RunStatusActive {
		t.Errorf("expected Status=Active, got %v", active.Status)
	}
	if active.Mode != DungeonModeUnknown {
		t.Errorf("expected Mode=Unknown initially, got %v", active.Mode)
	}
	if active.Tier != -1 {
		t.Errorf("expected Tier=-1 initially, got %d", active.Tier)
	}
	if !active.CloseAt.Equal(base.Add(DungeonCloseTimeout)) {
		t.Errorf("CloseAt: expected %v, got %v", base.Add(DungeonCloseTimeout), active.CloseAt)
	}

	// Exit — deltas should be zero (no stats gained)
	h.stats.addFame(500, h.nowFunc())
	h.stats.addKill(h.nowFunc())
	exitTime := base.Add(2 * time.Minute)
	h.nowFunc = func() time.Time { return exitTime }
	h.ExitDungeon(exitTime)

	active = h.GetActiveDungeon()
	if active != nil {
		t.Fatal("expected nil active run after exit")
	}

	runs := h.GetDungeonRuns()
	if len(runs) != 1 {
		t.Fatalf("expected 1 run in history, got %d", len(runs))
	}
	r := runs[0]
	if r.Status != RunStatusDone {
		t.Errorf("expected Status=Done, got %v", r.Status)
	}
	if r.Fame != 500 {
		t.Errorf("expected Fame delta 500, got %d", r.Fame)
	}
	if r.Kills != 1 {
		t.Errorf("expected Kills delta 1, got %d", r.Kills)
	}
}

func TestEnterDungeonClosesPrevious(t *testing.T) {
	h := NewAlbionHandler()
	t0 := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)

	h.nowFunc = func() time.Time { return t0 }
	h.EnterDungeon(t0)

	// Gain some fame during first run
	h.stats.addFame(1000, h.nowFunc())

	// Enter a second dungeon without exiting the first
	t1 := t0.Add(3 * time.Minute)
	h.nowFunc = func() time.Time { return t1 }
	h.EnterDungeon(t1)

	runs := h.GetDungeonRuns()
	if len(runs) != 2 {
		t.Fatalf("expected 2 runs, got %d", len(runs))
	}

	// First run should be Done with Fame delta 1000
	if runs[0].Status != RunStatusDone {
		t.Errorf("first run: expected Done, got %v", runs[0].Status)
	}
	if runs[0].Fame != 1000 {
		t.Errorf("first run: expected Fame 1000, got %d", runs[0].Fame)
	}

	// Second run should be Active
	if runs[1].Status != RunStatusActive {
		t.Errorf("second run: expected Active, got %v", runs[1].Status)
	}
	active := h.GetActiveDungeon()
	if active == nil || !active.EnteredAt.Equal(runs[1].EnteredAt) {
		t.Error("active run should be the second run")
	}
}

func TestExitDungeonNoActiveIsNoOp(t *testing.T) {
	h := NewAlbionHandler()
	h.ExitDungeon(time.Now())

	if len(h.GetDungeonRuns()) != 0 {
		t.Error("expected 0 runs when exiting with no active run")
	}
}

func TestUpdateActiveRunMode(t *testing.T) {
	h := NewAlbionHandler()
	t0 := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
	h.nowFunc = func() time.Time { return t0 }
	h.EnterDungeon(t0)

	h.UpdateActiveRunMode(DungeonModeSolo)
	if h.GetActiveDungeon().Mode != DungeonModeSolo {
		t.Errorf("expected Mode=Solo, got %v", h.GetActiveDungeon().Mode)
	}

	// Second call with a different mode should NOT overwrite (already classified)
	h.UpdateActiveRunMode(DungeonModeStandard)
	if h.GetActiveDungeon().Mode != DungeonModeSolo {
		t.Errorf("mode should not change after first classification, got %v", h.GetActiveDungeon().Mode)
	}
}

func TestUpdateActiveRunModeNoActiveRun(t *testing.T) {
	h := NewAlbionHandler()
	h.UpdateActiveRunMode(DungeonModeSolo) // should not panic
}

func TestUpdateActiveRunFaction(t *testing.T) {
	h := NewAlbionHandler()
	t0 := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
	h.nowFunc = func() time.Time { return t0 }
	h.EnterDungeon(t0)

	h.UpdateActiveRunFaction("Keeper")
	if h.GetActiveDungeon().Faction != "Keeper" {
		t.Errorf("expected Faction=Keeper, got %q", h.GetActiveDungeon().Faction)
	}

	// Second call with different faction should NOT overwrite
	h.UpdateActiveRunFaction("Heretic")
	if h.GetActiveDungeon().Faction != "Keeper" {
		t.Errorf("faction should not change, got %q", h.GetActiveDungeon().Faction)
	}
}

func TestUpdateActiveRunTierMonotonic(t *testing.T) {
	h := NewAlbionHandler()
	t0 := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
	h.nowFunc = func() time.Time { return t0 }
	h.EnterDungeon(t0)

	h.UpdateActiveRunTier(3)
	if h.GetActiveDungeon().Tier != 3 {
		t.Errorf("expected Tier=3, got %d", h.GetActiveDungeon().Tier)
	}

	// Lower tier should be ignored
	h.UpdateActiveRunTier(2)
	if h.GetActiveDungeon().Tier != 3 {
		t.Errorf("tier should not decrease, got %d", h.GetActiveDungeon().Tier)
	}

	// Higher tier should be accepted
	h.UpdateActiveRunTier(5)
	if h.GetActiveDungeon().Tier != 5 {
		t.Errorf("expected Tier=5, got %d", h.GetActiveDungeon().Tier)
	}
}

func TestUpdateActiveRunTierNegative(t *testing.T) {
	h := NewAlbionHandler()
	t0 := time.Now()
	h.nowFunc = func() time.Time { return t0 }
	h.EnterDungeon(t0)
	h.UpdateActiveRunTier(-1)
	if h.GetActiveDungeon().Tier != -1 {
		t.Errorf("negative tier should be ignored, got %d", h.GetActiveDungeon().Tier)
	}
}

func TestDungeonRunIsClosedNow(t *testing.T) {
	t0 := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
	r := &DungeonRun{
		EnteredAt: t0,
		CloseAt:   t0.Add(DungeonCloseTimeout),
		Status:    RunStatusActive,
	}

	// Before close time — not closed
	if r.IsClosedNow(t0.Add(30 * time.Second)) {
		t.Error("should not be closed before CloseAt")
	}

	// After close time — closed
	if !r.IsClosedNow(t0.Add(DungeonCloseTimeout + time.Second)) {
		t.Error("should be closed after CloseAt")
	}

	// Done runs are never "closed by timer"
	r.Status = RunStatusDone
	if r.IsClosedNow(t0.Add(DungeonCloseTimeout * 2)) {
		t.Error("Done runs should not report IsClosedNow")
	}
}

func TestDungeonRunDuration(t *testing.T) {
	t0 := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
	t1 := t0.Add(5 * time.Minute)

	// Active run: duration = now - entry
	r := &DungeonRun{EnteredAt: t0, Status: RunStatusActive}
	d := r.Duration(t0.Add(2 * time.Minute))
	if d != 2*time.Minute {
		t.Errorf("active duration: expected 2m, got %v", d)
	}

	// Done run: duration = exit - entry
	r2 := &DungeonRun{EnteredAt: t0, ExitedAt: t1, Status: RunStatusDone}
	d2 := r2.Duration(t0.Add(10 * time.Minute))
	if d2 != 5*time.Minute {
		t.Errorf("done duration: expected 5m, got %v", d2)
	}
}

func TestIsReliableRandomDungeonTierMob(t *testing.T) {
	cases := []struct {
		mob  mobEntry
		want bool
	}{
		{mobEntry{UniqueName: "T4_MOB_RD_UNDEAD_SOLDIER", Tier: 4}, true},
		{mobEntry{UniqueName: "T6_MOB_RD_KEEPER_MAGE", Tier: 6}, true},
		{mobEntry{UniqueName: "T4_MOB_RD_UNDEAD_BOSS", Tier: 4}, false},
		{mobEntry{UniqueName: "T4_MOB_RD_KEEPER_MINIBOSS", Tier: 4}, false},
		{mobEntry{UniqueName: "T4_MOB_RD_SUMMON", Tier: 4}, false},
		{mobEntry{UniqueName: "T4_MOB_RD_TRAP", Tier: 4}, false},
		{mobEntry{UniqueName: "T4_MOB_NOTRD_SOLDIER", Tier: 4}, false},
		{mobEntry{UniqueName: "", Tier: 4}, false},
		{mobEntry{UniqueName: "T0_MOB_RD_X", Tier: 0}, false},
		{mobEntry{UniqueName: "T9_MOB_RD_X", Tier: 9}, false},
	}
	for _, tc := range cases {
		got := isReliableRandomDungeonTierMob(tc.mob)
		if got != tc.want {
			t.Errorf("isReliable(%+v) = %v, want %v", tc.mob, got, tc.want)
		}
	}
}

func TestMobDatabaseGetRandomDungeonMobTier(t *testing.T) {
	// Build a minimal in-memory database
	db := &MobDatabase{
		mobs: []mobEntry{
			// index 16 (offset) → dataIndex 0
			{UniqueName: "T4_MOB_RD_UNDEAD_SOLDIER", Tier: 4},
			// index 17 → dataIndex 1
			{UniqueName: "T4_MOB_RD_UNDEAD_BOSS", Tier: 4},
			// index 18 → dataIndex 2
			{UniqueName: "T6_MOB_RD_KEEPER_MAGE", Tier: 6},
			// index 19 → dataIndex 3
			{UniqueName: "T4_MOB_NOTRD_X", Tier: 4},
		},
		loaded: true,
	}

	// Valid RD mob: tier 4 → returns 3 (tier - 1)
	if tier := db.GetRandomDungeonMobTier(inGameMobIndexOffset + 0); tier != 3 {
		t.Errorf("index 16: expected tier 3, got %d", tier)
	}

	// Boss: not reliable → -1
	if tier := db.GetRandomDungeonMobTier(inGameMobIndexOffset + 1); tier != -1 {
		t.Errorf("boss: expected -1, got %d", tier)
	}

	// Valid RD mob tier 6 → returns 5
	if tier := db.GetRandomDungeonMobTier(inGameMobIndexOffset + 2); tier != 5 {
		t.Errorf("index 18: expected tier 5, got %d", tier)
	}

	// Non-RD mob → -1
	if tier := db.GetRandomDungeonMobTier(inGameMobIndexOffset + 3); tier != -1 {
		t.Errorf("non-RD: expected -1, got %d", tier)
	}

	// Out of range → -1
	if tier := db.GetRandomDungeonMobTier(inGameMobIndexOffset + 100); tier != -1 {
		t.Errorf("out-of-range: expected -1, got %d", tier)
	}

	// Negative index → -1
	if tier := db.GetRandomDungeonMobTier(0); tier != -1 {
		t.Errorf("zero index: expected -1, got %d", tier)
	}
}

func TestMobDatabaseNilSafe(t *testing.T) {
	var db *MobDatabase
	if tier := db.GetRandomDungeonMobTier(100); tier != -1 {
		t.Errorf("nil db: expected -1, got %d", tier)
	}
}

func TestDungeonModeString(t *testing.T) {
	cases := []struct {
		mode DungeonMode
		want string
	}{
		{DungeonModeSolo, "Solo"},
		{DungeonModeStandard, "Group"},
		{DungeonModeAvalon, "Avalon"},
		{DungeonModeCorrupted, "Corrupted"},
		{DungeonModeHellGate, "HellGate"},
		{DungeonModeAbyssalDepths, "Abyssal"},
		{DungeonModeUnknown, "Unknown"},
	}
	for _, tc := range cases {
		if got := tc.mode.String(); got != tc.want {
			t.Errorf("%v.String() = %q, want %q", tc.mode, got, tc.want)
		}
	}
}

func TestRunStatusString(t *testing.T) {
	cases := []struct {
		status RunStatus
		want   string
	}{
		{RunStatusActive, "Active"},
		{RunStatusDone, "Done"},
		{RunStatusClosed, "Closed"},
	}
	for _, tc := range cases {
		if got := tc.status.String(); got != tc.want {
			t.Errorf("%v.String() = %q, want %q", tc.status, got, tc.want)
		}
	}
}

func TestGetDungeonRunsReturnsCopy(t *testing.T) {
	h := NewAlbionHandler()
	t0 := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
	h.nowFunc = func() time.Time { return t0 }
	h.EnterDungeon(t0)

	runs := h.GetDungeonRuns()
	if len(runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(runs))
	}

	// Mutate the returned slice — original should be unaffected
	runs[0] = nil
	runs2 := h.GetDungeonRuns()
	if runs2[0] == nil {
		t.Error("GetDungeonRuns should return a copy, not the original slice")
	}
}

func TestGetActiveDungeonClosesOnRead(t *testing.T) {
	h := NewAlbionHandler()
	t0 := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
	h.nowFunc = func() time.Time { return t0 }
	h.EnterDungeon(t0)

	// Before close time — active run exists
	if active := h.GetActiveDungeon(); active == nil {
		t.Fatal("expected active run before close time")
	}

	// After close time — run should auto-close on read
	h.nowFunc = func() time.Time { return t0.Add(DungeonCloseTimeout + time.Second) }

	if active := h.GetActiveDungeon(); active != nil {
		t.Error("expected nil active run after close timer elapsed")
	}

	runs := h.GetDungeonRuns()
	if len(runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(runs))
	}
	if runs[0].Status != RunStatusClosed {
		t.Errorf("expected Closed status, got %v", runs[0].Status)
	}
}

func TestGetActiveDungeonReturnsCopy(t *testing.T) {
	h := NewAlbionHandler()
	t0 := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
	h.nowFunc = func() time.Time { return t0 }
	h.EnterDungeon(t0)

	a1 := h.GetActiveDungeon()
	a2 := h.GetActiveDungeon()
	if a1 == nil || a2 == nil {
		t.Fatal("expected non-nil active runs")
	}
	// Each call must return a distinct copy (different pointers)
	if a1 == a2 {
		t.Error("GetActiveDungeon should return a new copy each call, not the same pointer")
	}
	// But same field values
	if !a1.EnteredAt.Equal(a2.EnteredAt) {
		t.Error("copies should have same EnteredAt")
	}
}

func TestSaveLoadDungeonRunsRoundTrip(t *testing.T) {
	h := NewAlbionHandler()
	t0 := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
	h.nowFunc = func() time.Time { return t0 }
	h.EnterDungeon(t0)
	h.stats.addFame(500, h.nowFunc())
	h.stats.addKill(h.nowFunc())
	h.stats.addKill(h.nowFunc())
	exit := t0.Add(2 * time.Minute)
	h.nowFunc = func() time.Time { return exit }
	h.ExitDungeon(exit)

	// Second completed run with a different stat.
	t1 := t0.Add(10 * time.Minute)
	h.nowFunc = func() time.Time { return t1 }
	h.EnterDungeon(t1)
	h.stats.addSilver(300, h.nowFunc())
	t2 := t1.Add(5 * time.Minute)
	h.nowFunc = func() time.Time { return t2 }
	h.ExitDungeon(t2)

	path := filepath.Join(t.TempDir(), "dungeon-runs.json")
	if err := h.SaveDungeonRuns(path); err != nil {
		t.Fatalf("SaveDungeonRuns: %v", err)
	}

	fresh := NewAlbionHandler()
	if err := fresh.LoadDungeonRuns(path); err != nil {
		t.Fatalf("LoadDungeonRuns: %v", err)
	}
	got := fresh.GetDungeonRuns()
	if len(got) != 2 {
		t.Fatalf("expected 2 runs after load, got %d", len(got))
	}
	if got[0].Fame != 500 || got[0].Kills != 2 || got[0].Status != RunStatusDone {
		t.Errorf("run0 = %+v, want Fame=500 Kills=2 Status=Done", got[0])
	}
	if !got[0].EnteredAt.Equal(t0) {
		t.Errorf("run0 EnteredAt = %v, want %v", got[0].EnteredAt, t0)
	}
	if got[1].Silver != 300 || got[1].Status != RunStatusDone {
		t.Errorf("run1 = %+v, want Silver=300 Status=Done", got[1])
	}
	// Unexported snapshot fields are not persisted.
	if got[0].snapFame != 0 {
		t.Errorf("run0 snapFame should be zero after load (not persisted), got %d", got[0].snapFame)
	}
}

func TestSaveDungeonRunsExcludesActiveRun(t *testing.T) {
	h := NewAlbionHandler()
	t0 := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
	h.nowFunc = func() time.Time { return t0 }
	h.EnterDungeon(t0) // active, never exited

	path := filepath.Join(t.TempDir(), "dungeon-runs.json")
	if err := h.SaveDungeonRuns(path); err != nil {
		t.Fatalf("SaveDungeonRuns: %v", err)
	}

	fresh := NewAlbionHandler()
	if err := fresh.LoadDungeonRuns(path); err != nil {
		t.Fatalf("LoadDungeonRuns: %v", err)
	}
	if got := fresh.GetDungeonRuns(); len(got) != 0 {
		t.Errorf("active run should not be persisted; got %d runs", len(got))
	}
}

func TestLoadDungeonRunsMissingFileIsEmptyNoError(t *testing.T) {
	h := NewAlbionHandler()
	path := filepath.Join(t.TempDir(), "nope.json")
	if err := h.LoadDungeonRuns(path); err != nil {
		t.Errorf("LoadDungeonRuns missing file: %v", err)
	}
	if got := h.GetDungeonRuns(); len(got) != 0 {
		t.Errorf("expected empty history, got %d runs", len(got))
	}
}

func TestLoadDungeonRunsCorruptFileReturnsError(t *testing.T) {
	h := NewAlbionHandler()
	// Seed an existing history that must survive a corrupt load.
	t0 := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
	h.nowFunc = func() time.Time { return t0 }
	h.EnterDungeon(t0)
	h.ExitDungeon(t0.Add(time.Minute))

	path := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(path, []byte("{not valid"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := h.LoadDungeonRuns(path); err == nil {
		t.Fatal("expected error loading corrupt file, got nil")
	}
	if got := h.GetDungeonRuns(); len(got) != 1 {
		t.Errorf("corrupt load should leave existing history intact; got %d runs", len(got))
	}
}

func TestLoadDungeonRunsRespectsCapAndSorts(t *testing.T) {
	// Persist more than the cap, in reverse order, directly to disk so we can
	// verify Load trims to the newest and re-sorts oldest-first.
	total := maxDungeonRuns + 50
	runs := make([]*DungeonRun, 0, total)
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < total; i++ {
		runs = append(runs, &DungeonRun{
			EnteredAt: base.Add(time.Duration(i) * time.Minute),
			Status:    RunStatusDone,
			Fame:      int64(i),
		})
	}
	for i, j := 0, len(runs)-1; i < j; i, j = i+1, j-1 {
		runs[i], runs[j] = runs[j], runs[i]
	}

	path := filepath.Join(t.TempDir(), "many.json")
	if err := storage.Save(path, runs); err != nil {
		t.Fatalf("Save: %v", err)
	}

	h := NewAlbionHandler()
	if err := h.LoadDungeonRuns(path); err != nil {
		t.Fatalf("LoadDungeonRuns: %v", err)
	}
	got := h.GetDungeonRuns()
	if len(got) != maxDungeonRuns {
		t.Fatalf("expected %d runs after cap, got %d", maxDungeonRuns, len(got))
	}
	// Oldest surviving run is index (total - cap) = 50.
	wantFirst := int64(total - maxDungeonRuns)
	if got[0].Fame != wantFirst {
		t.Errorf("first run Fame = %d, want %d (oldest after trim)", got[0].Fame, wantFirst)
	}
	if !got[0].EnteredAt.Before(got[len(got)-1].EnteredAt) {
		t.Errorf("runs should be ordered oldest-first after load")
	}
}

func TestDungeonRunsSnapshotIsDefensiveCopy(t *testing.T) {
	h := NewAlbionHandler()
	t0 := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
	h.nowFunc = func() time.Time { return t0 }
	h.EnterDungeon(t0)
	h.ExitDungeon(t0.Add(time.Minute))

	snap := h.DungeonRunsSnapshot()
	if len(snap) != 1 {
		t.Fatalf("expected 1 run, got %d", len(snap))
	}

	// Mutate the returned slice and its element; the handler's internal state
	// must be insulated.
	snap[0].Fame = 999999
	snap[0].Mode = DungeonModeCorrupted
	snap = append(snap, &DungeonRun{EnteredAt: t0.Add(time.Hour)})

	again := h.DungeonRunsSnapshot()
	if len(again) != 1 {
		t.Errorf("internal history changed after mutating snapshot: got %d runs", len(again))
	}
	if again[0].Fame == 999999 || again[0].Mode == DungeonModeCorrupted {
		t.Errorf("internal run mutated via snapshot: %+v", again[0])
	}
}
