package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cantalupo555/albion-lens/internal/serverdetect"
	"github.com/cantalupo555/albion-lens/internal/storage"
	"github.com/cantalupo555/albion-lens/pkg/handlers"
)

// clusterSaveWindow is the minimum interval between two cluster-change saves.
// It coalesces rapid map transitions (e.g. hopping through several zones) into
// a single write, matching the reference's 15-minute rate limit. The periodic
// 5-minute ticker remains as a separate safety net.
const clusterSaveWindow = 15 * time.Minute

// Names of the three persisted JSON state files plus the server-hint override
// file. Kept in one place so the resolver, the legacy migration, and any
// diagnostics reference the same constants.
const (
	dungeonRunsFile  = "dungeon-runs.json"
	sessionStatsFile = "session-stats.json"
	killDeathLogFile = "kill-death-log.json"
	serverHintFile   = "server-hint.txt"
)

// persistPaths owns the on-disk locations of the persisted state files for the
// currently active server region, and serializes all save/load/switch
// operations through a single mutex.
//
// The region field is the on-disk subdirectory name (e.g. "UserData-AMERICA");
// the empty string means the legacy/shared root. When the detected region
// changes, SwitchRegion flushes the old region to disk, resets in-memory state,
// and loads the new region — observed as one atomic transition by savers.
// warnFn delivers a diagnostic message to the user-visible event log. When
// nil (tests, pre-TUI init), callers fall back to fmt.Printf so the message
// is not lost.
type warnFn func(format string, args ...any)

type persistPaths struct {
	mu      sync.Mutex
	enabled bool
	region  string
	dungeon string
	stats   string
	kdlog   string
	initErr error
	warn    warnFn
}

// warnf emits a diagnostic message via the configured callback, or via
// fmt.Printf when no callback is set (tests).
func (p *persistPaths) warnf(format string, args ...any) {
	if p.warn != nil {
		p.warn(format, args...)
		return
	}
	fmt.Printf(format, args...)
}

// newPersistPaths resolves the initial paths for the given region. If path
// resolution fails (e.g. HOME unset or data dir read-only), persistence is
// disabled: enabled=false and all operations become no-ops, matching the
// pre-region behavior of emitting a warning and continuing without persistence.
// warn routes diagnostics to the TUI event log during operation; pass nil for
// tests or pre-TUI initialization (falls back to fmt.Printf).
func newPersistPaths(initialRegion string, warn warnFn) *persistPaths {
	p := &persistPaths{enabled: true, warn: warn}
	if err := p.reassignLocked(initialRegion); err != nil {
		p.enabled = false
		p.initErr = err
	}
	return p
}

// Enabled reports whether persistence is active. When false, Load/Save/Switch
// are no-ops and the caller should skip their warnings about failed saves.
func (p *persistPaths) Enabled() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.enabled
}

// InitError returns the error that caused persistence to be disabled at
// startup, or nil if initialization succeeded. It lets the caller log the
// specific cause (e.g. HOME unset, permissions) instead of a generic message.
func (p *persistPaths) InitError() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.initErr
}

// Region returns the currently active region subdirectory name ("" = legacy).
func (p *persistPaths) Region() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.region
}

// reassignLocked re-resolves all three paths for the given region. The caller
// MUST hold p.mu. An error leaves the previous paths untouched.
func (p *persistPaths) reassignLocked(region string) error {
	dungeon, err := storage.DataFileFor(region, dungeonRunsFile)
	if err != nil {
		return err
	}
	stats, err := storage.DataFileFor(region, sessionStatsFile)
	if err != nil {
		return err
	}
	kdlog, err := storage.DataFileFor(region, killDeathLogFile)
	if err != nil {
		return err
	}
	p.region = region
	p.dungeon = dungeon
	p.stats = stats
	p.kdlog = kdlog
	return nil
}

// Load reads all three files into the handler under the current region's paths.
// A missing file is the first-run case and loads empty state (storage.Load
// contract); a corrupt file surfaces as an error and leaves state unchanged.
// A no-op when persistence is disabled.
func (p *persistPaths) Load(h *handlers.AlbionHandler) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.enabled {
		return nil
	}
	return p.loadLocked(h)
}

func (p *persistPaths) loadLocked(h *handlers.AlbionHandler) error {
	// Best-effort: attempt all three files independently. Each failure is
	// logged so the user knows exactly which file is corrupt, not just the
	// first one. The first error is returned so callers can report a summary.
	// Because SwitchRegion has already reset the in-memory state to empty, a
	// corrupt file leaves that piece empty (not stale) — the new region
	// degrades to partial data rather than mixing regions.
	var firstErr error
	if err := h.LoadDungeonRuns(p.dungeon); err != nil {
		p.warnf("Warning: could not load %s: %v\n", dungeonRunsFile, err)
		if firstErr == nil {
			firstErr = err
		}
	}
	if err := h.LoadTotalStats(p.stats); err != nil {
		p.warnf("Warning: could not load %s: %v\n", sessionStatsFile, err)
		if firstErr == nil {
			firstErr = err
		}
	}
	if err := h.LoadKillDeathLog(p.kdlog); err != nil {
		p.warnf("Warning: could not load %s: %v\n", killDeathLogFile, err)
		if firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// Save writes all three files from the handler under the current region's
// paths. Each Save is atomic (tmp+rename); a failure mid-way leaves previously
// written files intact and aborts the remainder. A no-op when persistence is
// disabled.
func (p *persistPaths) Save(h *handlers.AlbionHandler) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.enabled {
		return nil
	}
	return p.saveLocked(h)
}

func (p *persistPaths) saveLocked(h *handlers.AlbionHandler) error {
	if err := h.SaveDungeonRuns(p.dungeon); err != nil {
		return err
	}
	if err := h.SaveTotalStats(p.stats); err != nil {
		return err
	}
	return h.SaveKillDeathLog(p.kdlog)
}

// SwitchRegion transitions persistence to a new region as a single atomic
// operation (under p.mu):
//
//  1. Flush the current state to the OLD region's paths (so no data is lost).
//  2. Reset in-memory state to empty (so a corrupt new-region file cannot leak
//     the old region's data — Load* leaves state untouched on decode error).
//  3. Run the one-time legacy→region migration, which copies any pre-region
//     data into the new region dir on first detection only.
//  4. Resolve the new region's paths.
//  5. Load the new region's state.
//
// A no-op when persistence is disabled or the region is unchanged.
//
// Concurrency note: the capture goroutine may keep updating the handler's
// in-memory state during the switch. Events arriving between the reset (step 2)
// and the load (step 5) are dropped. Region switches are rare (roughly one per
// login), so this microsecond window is an acceptable tradeoff versus the
// complexity of pausing capture.
func (p *persistPaths) SwitchRegion(newRegion string, h *handlers.AlbionHandler) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.enabled || p.region == newRegion {
		return nil
	}

	// 1. Flush the region we are leaving.
	if err := p.saveLocked(h); err != nil {
		return fmt.Errorf("flush %q before switch: %w", p.region, err)
	}

	// 2. Reset so the new region cannot inherit the old one's data on a
	//    corrupt load.
	h.ResetPersistedState()

	// 3. One-time legacy migration. Atomically claimed via an exclusive marker
	//    create, so it runs exactly once across the whole process lifetime:
	//    only the first confirmed region inherits the pre-region data; later
	//    regions start empty.
	if err := storage.MigrateLegacyData(newRegion); err != nil {
		// Non-fatal: a failed migration should not block region switching.
		// The new region simply starts empty.
		p.warnf("Warning: legacy migration to %s skipped: %v\n", newRegion, err)
	}

	// 4. Re-resolve paths for the new region.
	if err := p.reassignLocked(newRegion); err != nil {
		return fmt.Errorf("resolve %q: %w", newRegion, err)
	}

	// 5. Load the new region.
	return p.loadLocked(h)
}

// serverHintOverride reads a user-provided region override from server-hint.txt
// in the storage root. The file is a manual escape hatch for when the server IP
// prefixes change (see serverdetect.Servers): the user writes a region name and
// persistence pins to it, bypassing IP detection entirely.
//
// Accepted (case-insensitive, whitespace-trimmed) values:
//
//	"Americas", "America", "West"      → UserData-AMERICA
//	"Europe", "EU"                     → UserData-EUROPE
//	"Asia", "East"                     → UserData-ASIA
//	"UserData-AMERICA" etc. (DirName)  → matched directly
//
// Returns ServerLocationUnknown (no override) if the file is absent, empty, or
// holds an unrecognized value. A read error is treated as "no override" with a
// warning, since the hint is best-effort.
func serverHintOverride() serverdetect.ServerLocation {
	root, err := storage.DataDir()
	if err != nil {
		return serverdetect.ServerLocationUnknown
	}
	data, err := os.ReadFile(filepath.Join(root, serverHintFile))
	if err != nil {
		if !os.IsNotExist(err) {
			fmt.Printf("Warning: could not read %s: %v\n", serverHintFile, err)
		}
		return serverdetect.ServerLocationUnknown
	}
	loc := parseServerHint(string(data))
	if loc == serverdetect.ServerLocationUnknown && strings.TrimSpace(string(data)) != "" {
		fmt.Printf("Warning: unrecognized value %q in %s; expected Americas, Europe, or Asia\n",
			strings.TrimSpace(string(data)), serverHintFile)
	}
	return loc
}

// parseServerHint maps a free-form hint string to a ServerLocation. Returns
// ServerLocationUnknown for empty or unrecognized input.
func parseServerHint(s string) serverdetect.ServerLocation {
	s = strings.TrimSpace(s)
	if s == "" {
		return serverdetect.ServerLocationUnknown
	}
	lower := strings.ToLower(s)

	// Allow the exact DirName forms ("UserData-AMERICA").
	for _, loc := range []serverdetect.ServerLocation{
		serverdetect.ServerLocationAmerica,
		serverdetect.ServerLocationAsia,
		serverdetect.ServerLocationEurope,
	} {
		if lower == strings.ToLower(loc.DirName()) {
			return loc
		}
	}

	switch lower {
	case "americas", "america", "west", "us":
		return serverdetect.ServerLocationAmerica
	case "europe", "eu":
		return serverdetect.ServerLocationEurope
	case "asia", "east":
		return serverdetect.ServerLocationAsia
	}
	return serverdetect.ServerLocationUnknown
}

// rateLimiter decides whether an action should fire now, given that it last
// fired at some recorded time and must be spaced at least Window apart. It uses
// a compare-and-swap on the last-fired timestamp so concurrent triggers cannot
// stack: only the first one within a window wins.
//
// Used by the cluster-change save path: many zone transitions in quick
// succession yield at most one save per Window.
type rateLimiter struct {
	last   atomic.Int64 // unix nanos of the last approved trigger
	window time.Duration
}

// newRateLimiter returns a rateLimiter with the given minimum spacing. A
// brand-new limiter (last == 0) approves the first trigger immediately.
func newRateLimiter(window time.Duration) *rateLimiter {
	return &rateLimiter{window: window}
}

// Allow reports whether the action may fire at now. When it returns true, the
// limiter has also recorded now as the last fire time atomically, so concurrent
// callers racing on the same instant see at most one winner.
func (r *rateLimiter) Allow(now time.Time) bool {
	nowN := now.UnixNano()
	last := r.last.Load()
	if nowN-last < int64(r.window) {
		return false
	}
	return r.last.CompareAndSwap(last, nowN)
}
