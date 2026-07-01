package handlers

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DungeonCloseTimeout matches the reference DungeonCloseTimer (90 seconds from
// entry). The close countdown is computed on read, not via a goroutine.
const DungeonCloseTimeout = 90 * time.Second

// maxDungeonRuns caps the in-memory run history. Older runs are trimmed when
// the cap is exceeded.
const maxDungeonRuns = 500

// inGameMobIndexOffset matches the reference MobsData.InGameMobIndexOffset (16).
// The mob index sent by the server is offset by this amount relative to the
// mobs.json array position.
const inGameMobIndexOffset = 16

// DungeonMode classifies the type of random dungeon. Matches the reference
// DungeonMode enum (DungeonData.GetDungeonMode).
type DungeonMode int

const (
	DungeonModeUnknown DungeonMode = iota
	DungeonModeSolo
	DungeonModeStandard // group / veteran
	DungeonModeAvalon
	DungeonModeCorrupted
	DungeonModeHellGate
	DungeonModeAbyssalDepths
)

// String returns a short label for the dungeon mode.
func (m DungeonMode) String() string {
	switch m {
	case DungeonModeSolo:
		return "Solo"
	case DungeonModeStandard:
		return "Group"
	case DungeonModeAvalon:
		return "Avalon"
	case DungeonModeCorrupted:
		return "Corrupted"
	case DungeonModeHellGate:
		return "HellGate"
	case DungeonModeAbyssalDepths:
		return "Abyssal"
	default:
		return "Unknown"
	}
}

// RunStatus tracks the lifecycle of a dungeon run.
type RunStatus int

const (
	RunStatusActive RunStatus = iota
	RunStatusDone             // player exited normally
	RunStatusClosed           // 90s timer elapsed without exit (disconnect, etc.)
)

// String returns a short label for the run status.
func (s RunStatus) String() string {
	switch s {
	case RunStatusActive:
		return "Active"
	case RunStatusDone:
		return "Done"
	case RunStatusClosed:
		return "Closed"
	default:
		return "Unknown"
	}
}

// DungeonRun represents a single random dungeon run from entry to exit.
//
// JSON tags pin the on-disk field names so the persisted format is stable and
// additive-only (new fields default to zero when reading old files). The
// unexported snap* fields below are runtime-only (delta computation) and are
// automatically excluded from serialization by encoding/json.
type DungeonRun struct {
	EnteredAt time.Time   `json:"entered_at"`
	ExitedAt  time.Time   `json:"exited_at"` // zero while active
	Mode      DungeonMode `json:"mode"`
	Faction   string      `json:"faction"` // "", Keeper, Heretic, Morgana, Undead, Avalon
	Tier      int         `json:"tier"`    // -1 unknown; 0-7 (mob tier - 1 per reference)
	Level     int         `json:"level"`   // -1 unknown (deferred — needs exit-position catalog)
	Status    RunStatus   `json:"status"`
	CloseAt   time.Time   `json:"close_at"` // EnteredAt + 90s

	// Per-run stat deltas (computed on exit).
	Fame         int64 `json:"fame"`
	Silver       int64 `json:"silver"`
	Respec       int64 `json:"respec"`
	RespecSilver int64 `json:"respec_silver"`
	Kills        int   `json:"kills"`
	Deaths       int   `json:"deaths"`
	Loot         int   `json:"loot"`

	// Snapshot of session counters at entry (internal, for delta computation).
	// These unexported fields are NOT serialized; restored runs always have
	// Status != Active, so no delta computation is needed after load.
	snapFame         int64
	snapSilver       int64
	snapRespec       int64
	snapRespecSilver int64
	snapKills        int
	snapDeaths       int
	snapLoot         int
}

// IsClosedNow reports whether the 90s close timer has elapsed. Computed on
// read — no background goroutine needed.
func (r *DungeonRun) IsClosedNow(now time.Time) bool {
	return r.Status == RunStatusActive && !now.Before(r.CloseAt)
}

// Duration returns the effective run duration. For active runs this is
// time-since-entry; for completed runs it's exit-minus-entry.
func (r *DungeonRun) Duration(now time.Time) time.Duration {
	end := r.ExitedAt
	if end.IsZero() {
		end = now
	}
	return end.Sub(r.EnteredAt)
}

// --- Classification helpers (Phase 2a/2b) ---

// ClassifyDungeonMode determines the dungeon mode from a shrine or loot chest
// uniqueName. Mirrors DungeonData.GetDungeonMode from the reference project.
func ClassifyDungeonMode(uniqueName string) DungeonMode {
	upper := strings.ToUpper(uniqueName)
	switch {
	case strings.Contains(upper, "GENERAL_SHRINE_COMBAT_BUFF"):
		return DungeonModeSolo
	case strings.Contains(upper, "_SOLO_"):
		return DungeonModeSolo
	case strings.Contains(upper, "_VETERAN_"), strings.Contains(upper, "_HALLOWEEN"):
		return DungeonModeStandard
	case strings.Contains(upper, "AVALON"):
		return DungeonModeAvalon
	case strings.Contains(upper, "CORRUPTED"):
		return DungeonModeCorrupted
	case strings.Contains(upper, "HELL_"), strings.Contains(upper, "HELLGATE"):
		return DungeonModeHellGate
	case strings.Contains(upper, "HD_SHRINE_WRATH_BUFF"):
		return DungeonModeAbyssalDepths
	}
	return DungeonModeUnknown
}

// ClassifyDungeonFaction determines the dungeon faction from a shrine, loot
// chest, or mob uniqueName. Mirrors DungeonData.GetFaction from the reference.
func ClassifyDungeonFaction(uniqueName string) string {
	upper := strings.ToUpper(uniqueName)
	switch {
	case strings.Contains(upper, "KEEPER"):
		return "Keeper"
	case strings.Contains(upper, "HERETIC"):
		return "Heretic"
	case strings.Contains(upper, "MORGANA"):
		return "Morgana"
	case strings.Contains(upper, "UNDEAD"):
		return "Undead"
	case strings.Contains(upper, "AVALON"):
		return "Avalon"
	}
	return ""
}

// --- Mob database (Phase 2b) ---

// mobEntry holds the fields needed for tier classification.
type mobEntry struct {
	UniqueName string
	Tier       int
}

// MobDatabase maps server mob indices to mob data for tier classification.
// The index is the position in mobs.json + InGameMobIndexOffset.
type MobDatabase struct {
	mobs   []mobEntry
	loaded bool
}

// LoadMobDatabase loads mobs.json from the ao-bin-dumps directory.
func LoadMobDatabase(path string) (*MobDatabase, error) {
	fullPath := filepath.Join(path, "mobs.json")
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, err
	}

	var raw struct {
		Mobs struct {
			Mob []map[string]interface{} `json:"Mob"`
		} `json:"Mobs"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	db := &MobDatabase{
		mobs:   make([]mobEntry, len(raw.Mobs.Mob)),
		loaded: true,
	}
	for i, m := range raw.Mobs.Mob {
		entry := mobEntry{}
		if v, ok := m["@uniquename"].(string); ok {
			entry.UniqueName = v
		}
		if v, ok := m["@tier"].(string); ok {
			entry.Tier = atoiSafe(v)
		} else if v, ok := m["@tier"].(float64); ok {
			entry.Tier = int(v)
		}
		db.mobs[i] = entry
	}
	return db, nil
}

// GetRandomDungeonMobTier looks up a mob by server index and returns its
// dungeon tier (mob.Tier - 1) if the mob is a reliable RD tier indicator.
// Returns -1 when the mob is not a reliable indicator or the index is out of
// range. Mirrors MobsData.GetRandomDungeonMobTierByIndex.
func (db *MobDatabase) GetRandomDungeonMobTier(mobIndex int) int {
	if db == nil || !db.loaded {
		return -1
	}
	dataIndex := mobIndex - inGameMobIndexOffset
	if dataIndex < 0 || dataIndex >= len(db.mobs) {
		return -1
	}
	mob := db.mobs[dataIndex]
	if !isReliableRandomDungeonTierMob(mob) {
		return -1
	}
	return mob.Tier - 1
}

// isReliableRandomDungeonTierMob mirrors MobsData.IsReliableRandomDungeonTierMob.
func isReliableRandomDungeonTierMob(mob mobEntry) bool {
	if mob.Tier < 1 || mob.Tier > 8 || mob.UniqueName == "" {
		return false
	}
	upper := strings.ToUpper(mob.UniqueName)
	if !strings.Contains(upper, "_MOB_RD_") {
		return false
	}
	return !strings.Contains(upper, "_BOSS") &&
		!strings.Contains(upper, "_MINIBOSS") &&
		!strings.Contains(upper, "_SUMMON") &&
		!strings.Contains(upper, "_UNATTACKABLE") &&
		!strings.Contains(upper, "_TRAP")
}

// atoiSafe parses a decimal string to int, returning 0 on failure.
func atoiSafe(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
	}
	return n
}

// --- Dungeon run lifecycle (Phase 1) ---

// EnterDungeon starts a new dungeon run. If a run is already active, it is
// closed (deltas computed, Status=Done) before the new run is created.
// Session counters are snapshotted for per-run delta attribution.
func (h *AlbionHandler) EnterDungeon(now time.Time) {
	h.dungeonMu.Lock()
	defer h.dungeonMu.Unlock()

	if h.activeRun != nil {
		h.closeActiveRunLocked(now, RunStatusDone)
	}

	// Snapshot the session counters once, consistently, so the per-run delta
	// computed on exit is coherent (no mix of pre/post-update values).
	snap := h.stats.snapshot()
	run := &DungeonRun{
		EnteredAt:        now,
		Mode:             DungeonModeUnknown,
		Tier:             -1,
		Level:            -1,
		Status:           RunStatusActive,
		CloseAt:          now.Add(DungeonCloseTimeout),
		snapFame:         snap.Fame,
		snapSilver:       snap.Silver,
		snapRespec:       snap.Respec,
		snapRespecSilver: snap.RespecSilver,
		snapKills:        int(snap.Kills),
		snapDeaths:       int(snap.Deaths),
		snapLoot:         int(snap.Loot),
	}
	h.activeRun = run
	h.dungeonRuns = append(h.dungeonRuns, run)

	// Trim oldest runs when the cap is exceeded.
	if len(h.dungeonRuns) > maxDungeonRuns {
		h.dungeonRuns = h.dungeonRuns[len(h.dungeonRuns)-maxDungeonRuns:]
	}
}

// ExitDungeon closes the active dungeon run and computes stat deltas.
// No-op when there is no active run.
func (h *AlbionHandler) ExitDungeon(now time.Time) {
	h.dungeonMu.Lock()
	defer h.dungeonMu.Unlock()

	if h.activeRun == nil {
		return
	}
	h.closeActiveRunLocked(now, RunStatusDone)
}

// closeActiveRunLocked finalizes the active run. Caller must hold dungeonMu.
func (h *AlbionHandler) closeActiveRunLocked(now time.Time, status RunStatus) {
	r := h.activeRun
	if r == nil {
		return
	}
	r.ExitedAt = now
	r.Status = status
	// Read the current counters as one consistent snapshot so every delta is
	// computed against the same point in time.
	cur := h.stats.snapshot()
	r.Fame = cur.Fame - r.snapFame
	r.Silver = cur.Silver - r.snapSilver
	r.Respec = cur.Respec - r.snapRespec
	r.RespecSilver = cur.RespecSilver - r.snapRespecSilver
	r.Kills = int(cur.Kills) - r.snapKills
	r.Deaths = int(cur.Deaths) - r.snapDeaths
	r.Loot = int(cur.Loot) - r.snapLoot
	h.activeRun = nil
}

// UpdateActiveRunMode sets the mode on the active run if it is currently
// Unknown. Thread-safe.
func (h *AlbionHandler) UpdateActiveRunMode(mode DungeonMode) {
	h.dungeonMu.Lock()
	defer h.dungeonMu.Unlock()
	if h.activeRun != nil && h.activeRun.Mode == DungeonModeUnknown && mode != DungeonModeUnknown {
		h.activeRun.Mode = mode
	}
}

// UpdateActiveRunFaction sets the faction on the active run if it is currently
// empty. Thread-safe.
func (h *AlbionHandler) UpdateActiveRunFaction(faction string) {
	if faction == "" {
		return
	}
	h.dungeonMu.Lock()
	defer h.dungeonMu.Unlock()
	if h.activeRun != nil && h.activeRun.Faction == "" {
		h.activeRun.Faction = faction
	}
}

// UpdateActiveRunTier sets the tier on the active run. The tier is
// monotonically increasing: a lower or equal value is ignored. Thread-safe.
func (h *AlbionHandler) UpdateActiveRunTier(tier int) {
	if tier < 0 {
		return
	}
	h.dungeonMu.Lock()
	defer h.dungeonMu.Unlock()
	if h.activeRun != nil && tier > h.activeRun.Tier {
		h.activeRun.Tier = tier
	}
}

// closeExpiredLocked transitions the active run to RunStatusClosed if the 90s
// timer has elapsed. Deltas are computed once (frozen on close). Caller must
// hold dungeonMu.
func (h *AlbionHandler) closeExpiredLocked(now time.Time) {
	if h.activeRun != nil && h.activeRun.IsClosedNow(now) {
		h.closeActiveRunLocked(now, RunStatusClosed)
	}
}

// GetActiveDungeon returns a snapshot of the active dungeon run, or nil if
// the player is not in a dungeon (or the 90s close timer just elapsed). The
// returned pointer is to a copy — callers may freely read all fields without
// holding the handler lock.
func (h *AlbionHandler) GetActiveDungeon() *DungeonRun {
	h.dungeonMu.Lock()
	defer h.dungeonMu.Unlock()
	h.closeExpiredLocked(h.nowFunc())
	if h.activeRun == nil {
		return nil
	}
	run := *h.activeRun
	return &run
}

// copyDungeonRunsLocked returns defensive copies of all recorded runs. Caller
// must hold dungeonMu.
func (h *AlbionHandler) copyDungeonRunsLocked() []*DungeonRun {
	result := make([]*DungeonRun, len(h.dungeonRuns))
	for i, r := range h.dungeonRuns {
		cp := *r
		result[i] = &cp
	}
	return result
}

// GetDungeonRuns returns snapshots of all recorded dungeon runs (oldest first).
// Each pointer is to a copy — callers may freely read, reorder, or retain the
// slice without holding the handler lock.
func (h *AlbionHandler) GetDungeonRuns() []*DungeonRun {
	h.dungeonMu.Lock()
	defer h.dungeonMu.Unlock()
	h.closeExpiredLocked(h.nowFunc())
	return h.copyDungeonRunsLocked()
}

// DungeonRunsSnapshot returns a defensive copy of the recorded run history
// (oldest first) for inspection by tests or the UI without holding the handler
// lock. Like GetDungeonRuns it closes any run whose 90s timer has elapsed, so
// the two APIs are consistent.
func (h *AlbionHandler) DungeonRunsSnapshot() []*DungeonRun {
	h.dungeonMu.Lock()
	defer h.dungeonMu.Unlock()
	h.closeExpiredLocked(h.nowFunc())
	return h.copyDungeonRunsLocked()
}
