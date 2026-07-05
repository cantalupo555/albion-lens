package main

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/cantalupo555/albion-lens/internal/serverdetect"
	"github.com/cantalupo555/albion-lens/pkg/events"
	"github.com/cantalupo555/albion-lens/pkg/handlers"
)

// TestPersistPathsSwitchRegionEndToEnd verifies the full region-switch flow:
//
//  1. State accumulates in the legacy root and is saved there.
//  2. First SwitchRegion migrates the legacy snapshot into the new region dir
//     and reloads it (state preserved across the switch).
//  3. A second SwitchRegion to a different region starts EMPTY — the migration
//     marker prevents the original legacy data from seeding every region.
//
// This is the core correctness guarantee of region segregation: no
// cross-region bleed.
func TestPersistPathsSwitchRegionEndToEnd(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_DATA_HOME", root)

	h := handlers.NewAlbionHandler()
	h.SetLocalPlayer("Hero")

	// Seed in-memory state with a fame event so there is something to persist.
	seedFame(t, h, 1234)

	paths := newPersistPaths("", nil) // legacy / Unknown region
	if !paths.Enabled() {
		t.Fatal("persistence disabled; expected enabled")
	}
	if paths.Region() != "" {
		t.Fatalf("initial region = %q, want empty (legacy)", paths.Region())
	}

	// Save to legacy root.
	if err := paths.Save(h); err != nil {
		t.Fatalf("initial save: %v", err)
	}
	legacyStats := filepath.Join(root, "albion-lens", "session-stats.json")
	if _, err := os.Stat(legacyStats); err != nil {
		t.Fatalf("legacy stats file not created: %v", err)
	}

	// Switch to Americas. The legacy data should migrate into UserData-AMERICA
	// and the in-memory fame should be preserved (reloaded from the new dir).
	if err := paths.SwitchRegion("UserData-AMERICA", h); err != nil {
		t.Fatalf("switch to Americas: %v", err)
	}
	if paths.Region() != "UserData-AMERICA" {
		t.Fatalf("region after switch = %q, want UserData-AMERICA", paths.Region())
	}
	if got := h.GetTotalFame(); got != 1234 {
		t.Errorf("fame after migrate-and-reload = %d, want 1234 (preserved)", got)
	}
	if _, err := os.Stat(filepath.Join(root, "albion-lens", "UserData-AMERICA", "session-stats.json")); err != nil {
		t.Errorf("Americas stats file missing after migration: %v", err)
	}

	// Switch to Europe. Europe must start EMPTY: the migration marker should
	// prevent the original legacy snapshot from being copied again.
	if err := paths.SwitchRegion("UserData-EUROPE", h); err != nil {
		t.Fatalf("switch to Europe: %v", err)
	}
	if got := h.GetTotalFame(); got != 0 {
		t.Errorf("Europe fame = %d, want 0 (region starts empty, no bleed)", got)
	}
}

// TestPersistPathsSwitchToSameRegionIsNoOp confirms switching to the current
// region does not flush/reset/load (which would drop in-flight state).
func TestPersistPathsSwitchToSameRegionIsNoOp(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	h := handlers.NewAlbionHandler()

	paths := newPersistPaths("UserData-AMERICA", nil)
	if err := paths.Load(h); err != nil {
		t.Fatalf("load: %v", err)
	}

	// Accumulate state during the session (after the initial load).
	seedFame(t, h, 500)

	// Switch to the same region: no-op, fame preserved.
	if err := paths.SwitchRegion("UserData-AMERICA", h); err != nil {
		t.Fatalf("switch same: %v", err)
	}
	if got := h.GetTotalFame(); got != 500 {
		t.Errorf("fame after same-region switch = %d, want 500 (unchanged)", got)
	}
}

// TestPersistPathsLoadRoundTrip verifies the three files survive save+load and
// that a fresh handler hydrates from them.
func TestPersistPathsLoadRoundTrip(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	h := handlers.NewAlbionHandler()
	h.SetLocalPlayer("Hero")
	seedFame(t, h, 9000)

	paths := newPersistPaths("UserData-ASIA", nil)
	if err := paths.Save(h); err != nil {
		t.Fatalf("save: %v", err)
	}

	// Fresh handler + paths: load should restore the persisted fame.
	fresh := handlers.NewAlbionHandler()
	paths2 := newPersistPaths("UserData-ASIA", nil)
	if err := paths2.Load(fresh); err != nil {
		t.Fatalf("load: %v", err)
	}
	if got := fresh.GetTotalFame(); got != 9000 {
		t.Errorf("restored fame = %d, want 9000", got)
	}
}

// TestPersistPathsDisabledOnBadRoot confirms a bad data dir disables
// persistence gracefully (operations become no-ops rather than panicking).
func TestPersistPathsDisabledOnBadRoot(t *testing.T) {
	// An unset HOME and XDG_DATA_HOME makes path resolution fail.
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("HOME", "")

	paths := newPersistPaths("", nil)
	if paths.Enabled() {
		t.Fatal("expected persistence disabled when HOME and XDG_DATA_HOME unset")
	}
	// All operations must be safe no-ops.
	h := handlers.NewAlbionHandler()
	if err := paths.Load(h); err != nil {
		t.Errorf("Load on disabled paths returned err: %v", err)
	}
	if err := paths.Save(h); err != nil {
		t.Errorf("Save on disabled paths returned err: %v", err)
	}
	if err := paths.SwitchRegion("UserData-AMERICA", h); err != nil {
		t.Errorf("SwitchRegion on disabled paths returned err: %v", err)
	}
}

func TestParseServerHint(t *testing.T) {
	cases := []struct {
		in   string
		want serverdetect.ServerLocation
	}{
		{"", serverdetect.ServerLocationUnknown},
		{"   ", serverdetect.ServerLocationUnknown},
		{"nonsense", serverdetect.ServerLocationUnknown},
		{"Americas", serverdetect.ServerLocationAmerica},
		{"AMERICA", serverdetect.ServerLocationAmerica},
		{" america ", serverdetect.ServerLocationAmerica},
		{"west", serverdetect.ServerLocationAmerica},
		{"US", serverdetect.ServerLocationAmerica},
		{"Europe", serverdetect.ServerLocationEurope},
		{"eu", serverdetect.ServerLocationEurope},
		{"Asia", serverdetect.ServerLocationAsia},
		{"east", serverdetect.ServerLocationAsia},
		{"UserData-AMERICA", serverdetect.ServerLocationAmerica},
		{"userdata-europe", serverdetect.ServerLocationEurope},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			if got := parseServerHint(tc.in); got != tc.want {
				t.Errorf("parseServerHint(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

// TestPersistPathsSwitchRegionCorruptFileNoBleed is the core correctness
// guarantee of region segregation: when the new region's stats file is corrupt,
// the switch must NOT leak the old region's data into the new one. Reset must
// run before the load, so a corrupt load (which leaves state untouched) finds
// an already-empty handler rather than the previous region's counters.
//
// It also covers the totalFame dedup baseline reset: a fresh fame event in the
// new region must be accepted even though the old region saw a higher total.
func TestPersistPathsSwitchRegionCorruptFileNoBleed(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_DATA_HOME", root)

	h := handlers.NewAlbionHandler()
	h.SetLocalPlayer("Hero")
	seedFame(t, h, 1234)

	paths := newPersistPaths("", nil)
	if err := paths.Save(h); err != nil {
		t.Fatalf("save legacy: %v", err)
	}

	// Pre-seed the Americas region dir with a corrupt session-stats.json so
	// the switch's load fails on that file. The dungeon-runs and kd-log files
	// are absent, so they load as empty.
	americasDir := filepath.Join(root, "albion-lens", "UserData-AMERICA")
	if err := os.MkdirAll(americasDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(americasDir, "session-stats.json"),
		[]byte("{not valid json"), 0o644); err != nil {
		t.Fatal(err)
	}

	// The switch loads the corrupt file, which fails — but the reset already
	// cleared the old region's fame, so the new region starts empty (no bleed).
	if err := paths.SwitchRegion("UserData-AMERICA", h); err == nil {
		t.Error("SwitchRegion with corrupt stats file: expected error, got nil")
	}
	if got := h.GetTotalFame(); got != 0 {
		t.Errorf("after switch into corrupt region, fame = %d, want 0 (no bleed from old region)", got)
	}

	// The totalFame dedup baseline must be reset, so a fresh fame event in the
	// new region is accepted (not rejected by the "total must not decrease"
	// guard). Seed 500 fame: it must land.
	seedFame(t, h, 500)
	if got := h.GetTotalFame(); got != 500 {
		t.Errorf("post-switch fame event rejected: got %d, want 500 (totalFame not reset)", got)
	}
}

// TestResetPersistedStateClearsAll directly verifies the handler zeros the
// persisted surfaces, since the SwitchRegion integration tests only check fame.
func TestResetPersistedStateClearsAll(t *testing.T) {
	h := handlers.NewAlbionHandler()
	h.SetLocalPlayer("Hero")
	seedFame(t, h, 1000)

	// Pre-reset: fame and the daily bucket are non-zero.
	if h.GetTotalFame() != 1000 {
		t.Fatalf("seed: fame = %d, want 1000", h.GetTotalFame())
	}

	h.ResetPersistedState()

	// Post-reset: every cumulative counter is zero.
	checks := []struct {
		name string
		got  int64
	}{
		{"fame", h.GetTotalFame()},
		{"silver", h.GetTotalSilver()},
		{"respec", h.GetTotalRespec()},
	}
	for _, tc := range checks {
		if tc.got != 0 {
			t.Errorf("after reset, %s = %d, want 0", tc.name, tc.got)
		}
	}
	for _, tc := range []struct {
		name string
		got  int
	}{
		{"kills", h.GetTotalKills()},
		{"deaths", h.GetTotalDeaths()},
		{"loot", h.GetTotalLoot()},
	} {
		if tc.got != 0 {
			t.Errorf("after reset, %s = %d, want 0", tc.name, tc.got)
		}
	}

	// The daily/hourly buckets must be empty too.
	snap := h.TotalSnapshot()
	if len(snap.Daily) != 0 || len(snap.Hourly) != 0 {
		t.Errorf("after reset, daily/hourly buckets not empty: daily=%d hourly=%d",
			len(snap.Daily), len(snap.Hourly))
	}
}

// TestServerHintOverrideFromFile confirms the hint is read from server-hint.txt
// in the storage root, and absent returns Unknown (no override).
func TestServerHintOverrideFromFile(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_DATA_HOME", root)

	if got := serverHintOverride(); got != serverdetect.ServerLocationUnknown {
		t.Errorf("hint with no file = %v, want Unknown", got)
	}

	hintPath := filepath.Join(root, "albion-lens", serverHintFile)
	if err := os.MkdirAll(filepath.Dir(hintPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(hintPath, []byte("Europe"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := serverHintOverride(); got != serverdetect.ServerLocationEurope {
		t.Errorf("hint from file = %v, want Europe", got)
	}

	// Unrecognized value must return Unknown (not match a wrong region).
	if err := os.WriteFile(hintPath, []byte("Europ"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := serverHintOverride(); got != serverdetect.ServerLocationUnknown {
		t.Errorf("unrecognized hint = %v, want Unknown", got)
	}
}

// TestRateLimiterFirstCallAllowed confirms a fresh limiter approves the first
// trigger immediately (no artificial startup delay). The timestamp must be far
// past epoch since last=0 means "never"; in production time.Now() always is.
func TestRateLimiterFirstCallAllowed(t *testing.T) {
	r := newRateLimiter(time.Hour)
	if !r.Allow(time.Unix(1_000_000_000, 0)) {
		t.Error("first Allow() = false, want true")
	}
}

// TestRateLimiterCoalescesWithinWindow verifies rapid triggers are coalesced:
// only the first within a window wins; the next is allowed only after window.
func TestRateLimiterCoalescesWithinWindow(t *testing.T) {
	r := newRateLimiter(15 * time.Minute)
	start := time.Unix(1_000_000, 0)

	if !r.Allow(start) {
		t.Fatal("first trigger denied")
	}
	// Within the window: denied.
	if r.Allow(start.Add(1 * time.Minute)) {
		t.Error("trigger 1min after first allowed; want denied (within window)")
	}
	if r.Allow(start.Add(14 * time.Minute)) {
		t.Error("trigger 14min after first allowed; want denied (within window)")
	}
	// Exactly at window boundary: allowed.
	if !r.Allow(start.Add(15 * time.Minute)) {
		t.Error("trigger at 15min boundary denied; want allowed")
	}
	// Immediately after: denied again (new window started).
	if r.Allow(start.Add(15*time.Minute + 1*time.Second)) {
		t.Error("trigger 1s after re-fire allowed; want denied")
	}
}

// TestRateLimiterConcurrentOnlyOneWins confirms that many concurrent triggers
// at the same instant produce exactly one winner (CAS prevents stacking).
func TestRateLimiterConcurrentOnlyOneWins(t *testing.T) {
	r := newRateLimiter(time.Hour)
	now := time.Unix(5_000_000, 0)

	const n = 50
	wins := make(chan bool, n)
	for i := 0; i < n; i++ {
		go func() { wins <- r.Allow(now) }()
	}
	count := 0
	for i := 0; i < n; i++ {
		if <-wins {
			count++
		}
	}
	if count != 1 {
		t.Errorf("concurrent Allow at one instant: %d winners, want 1", count)
	}
}

// TestWatchServerChangesSwitchesOnKnownRegion verifies watchServerChanges
// re-points persistence when a known region arrives, ignores Unknown
// transitions, and exits cleanly when the channel closes.
func TestWatchServerChangesSwitchesOnKnownRegion(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	h := handlers.NewAlbionHandler()
	paths := newPersistPaths("", nil)

	changes := make(chan serverdetect.ChangeEvent, 4)
	done := make(chan struct{})
	go func() {
		watchServerChanges(changes, h, paths, nil)
		close(done)
	}()

	// Unknown transition must be ignored (paths stay on legacy).
	changes <- serverdetect.ChangeEvent{Current: serverdetect.Unknown()}
	// Known region: paths must switch.
	changes <- serverdetect.ChangeEvent{Current: serverdetect.MatchByIPString("5.188.125.1")}
	close(changes)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("watchServerChanges did not exit after channel close")
	}

	if got := paths.Region(); got != "UserData-AMERICA" {
		t.Errorf("region after switch = %q, want UserData-AMERICA", got)
	}
}

// TestWatchServerChangesNilHandlerNoPanic confirms a nil handler does not
// crash the watcher (the guard skips the transition).
func TestWatchServerChangesNilHandlerNoPanic(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	paths := newPersistPaths("", nil)

	changes := make(chan serverdetect.ChangeEvent, 1)
	done := make(chan struct{})
	go func() {
		watchServerChanges(changes, nil, paths, nil) // nil handler
		close(done)
	}()

	changes <- serverdetect.ChangeEvent{Current: serverdetect.MatchByIPString("5.188.125.1")}
	close(changes)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("watcher did not exit; nil-handler guard may have panicked")
	}
	// Region unchanged: nil handler means SwitchRegion was never called.
	if got := paths.Region(); got != "" {
		t.Errorf("region changed despite nil handler: %q", got)
	}
}

// TestPersistPathsSwitchRegionMigrationFailureIsNonFatal verifies that a
// migration error during SwitchRegion does not block the region switch: paths
// are reassigned, the handler state is reset, and the new region loads (even
// if empty). Migration failure is triggered by placing an unreadable .json
// file in the legacy root that copyFile cannot open.
func TestPersistPathsSwitchRegionMigrationFailureIsNonFatal(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_DATA_HOME", root)

	h := handlers.NewAlbionHandler()
	h.SetLocalPlayer("Hero")
	seedFame(t, h, 5000)

	paths := newPersistPaths("", nil)
	if err := paths.Save(h); err != nil {
		t.Fatalf("initial save: %v", err)
	}

	// Add an unreadable .json file in the legacy root so migration's copyFile
	// fails on it. Using a separate file (not one of the three persisted files)
	// ensures saveLocked's atomic rename doesn't reset its permissions.
	badFile := filepath.Join(root, "albion-lens", "zzz-corrupt.json")
	if err := os.WriteFile(badFile, []byte("{}"), 0o644); err != nil {
		t.Fatalf("write bad file: %v", err)
	}
	if err := os.Chmod(badFile, 0o000); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(badFile, 0o644) })

	// SwitchRegion must not return an error: migration failure is non-fatal.
	// The unreadable file sorts after the persisted files, so migration copies
	// them first, then fails on zzz-corrupt.json and returns an error — but
	// SwitchRegion logs the warning and continues.
	err := paths.SwitchRegion("UserData-EUROPE", h)
	if err != nil {
		t.Fatalf("SwitchRegion returned error despite non-fatal migration: %v", err)
	}

	// Region must have switched.
	if got := paths.Region(); got != "UserData-EUROPE" {
		t.Fatalf("region = %q, want UserData-EUROPE", got)
	}
}

// TestPersistPathsLoadBestEffortMultipleCorrupt verifies that loadLocked
// attempts all three files independently: when multiple are corrupt, the load
// still returns an error but does not panic or abort before trying each file.
func TestPersistPathsLoadBestEffortMultipleCorrupt(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_DATA_HOME", root)

	h := handlers.NewAlbionHandler()
	seedFame(t, h, 999)

	paths := newPersistPaths("", nil)
	if err := paths.Save(h); err != nil {
		t.Fatalf("save: %v", err)
	}

	// Corrupt two of the three files.
	dataDir := filepath.Join(root, "albion-lens")
	if err := os.WriteFile(filepath.Join(dataDir, sessionStatsFile), []byte("{bad"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, killDeathLogFile), []byte("{bad"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Load must return an error (at least one file corrupt) but must not panic.
	err := paths.Load(h)
	if err == nil {
		t.Error("expected error from load with corrupt files, got nil")
	}
}

// TestPersistPathsConcurrentSaveAndSwitch verifies the persistPaths mutex
// serializes Save and SwitchRegion so they cannot interleave and corrupt
// state. Runs under -race; any data race or panic fails the test.
func TestPersistPathsConcurrentSaveAndSwitch(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	h := handlers.NewAlbionHandler()
	paths := newPersistPaths("", nil)

	regions := []string{"UserData-AMERICA", "UserData-EUROPE", "UserData-ASIA"}
	stop := make(chan struct{})
	var saverWg, switcherWg sync.WaitGroup

	// Saver: hammer Save until stopped.
	saverWg.Add(1)
	go func() {
		defer saverWg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				_ = paths.Save(h)
			}
		}
	}()

	// Switcher: cycle regions a bounded number of times.
	switcherWg.Add(1)
	go func() {
		defer switcherWg.Done()
		for i := 0; i < 200; i++ {
			_ = paths.SwitchRegion(regions[i%len(regions)], h)
		}
	}()

	switcherWg.Wait() // bounded work finishes
	close(stop)       // release the saver
	saverWg.Wait()
	// One final serialized save must succeed (the mutex is healthy).
	_ = paths.Save(h)
}

// seedFame feeds a fame event that increments the cumulative total by n, used
// to give the handler non-zero persisted state for round-trip testing.
func seedFame(t *testing.T, h *handlers.AlbionHandler, n int64) {
	t.Helper()
	h.OnEvent(0, map[byte]interface{}{
		1:                     int64(50_000_000_000), // total fame (FixPoint)
		2:                     n * 10_000,            // gained n (FixPoint = /10000)
		events.ParamEventCode: int16(events.EventUpdateFame),
	})
	if got := h.GetTotalFame(); got != n {
		t.Fatalf("seedFame: total fame = %d, want %d (seed failed)", got, n)
	}
}
