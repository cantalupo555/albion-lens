// Package handlers implements event handlers for Albion Online game events
package handlers

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cantalupo555/albion-lens/internal/storage"
	"github.com/cantalupo555/albion-lens/pkg/events"
	"github.com/cantalupo555/albion-lens/pkg/items"
)

// EventCallback is called when a game event is processed
// eventType: "fame", "silver", "loot", "respec", "combat", "info", "death", "kill", "zone", "debug"
// message: formatted message to display
// data: optional structured data (FameEventData, SilverEventData, RespecEventData, ZoneEventData, etc.)
type EventCallback func(eventType, message string, data interface{})

// defaultDeathDedupWindow is how long a (victim, killer) death signature is
// remembered so that redundant copies of the same death (delivered when the local
// player is the killer) are collapsed into a single event. Short enough that two
// legitimate kills this far apart are both counted. Overridable per-handler for
// testing via AlbionHandler.deathDedupWindow.
const defaultDeathDedupWindow = 5 * time.Second

// totalCounters holds the seven cumulative session counters behind a single
// RWMutex so a snapshot reads all of them consistently (independent atomics
// would allow a snapshot to mix pre- and post-update values). The lock is
// fine-grained: event handlers take a write lock per increment, persistence
// and UI hydration take a read lock for the whole set.
type totalCounters struct {
	mu           sync.RWMutex
	fame         int64
	silver       int64
	respec       int64
	respecSilver int64
	kills        int64
	deaths       int64
	loot         int64
}

// snapshot returns a consistent point-in-time copy of all seven counters.
func (c *totalCounters) snapshot() TotalStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return TotalStats{
		Fame:         c.fame,
		Silver:       c.silver,
		Respec:       c.respec,
		RespecSilver: c.respecSilver,
		Kills:        c.kills,
		Deaths:       c.deaths,
		Loot:         c.loot,
	}
}

// apply overwrites all seven counters. Used when restoring persisted state.
func (c *totalCounters) apply(s TotalStats) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.fame = s.Fame
	c.silver = s.Silver
	c.respec = s.Respec
	c.respecSilver = s.RespecSilver
	c.kills = s.Kills
	c.deaths = s.Deaths
	c.loot = s.Loot
}

func (c *totalCounters) addFame(n int64) {
	c.mu.Lock()
	c.fame += n
	c.mu.Unlock()
}

func (c *totalCounters) addSilver(n int64) {
	c.mu.Lock()
	c.silver += n
	c.mu.Unlock()
}

func (c *totalCounters) addRespec(n int64) {
	c.mu.Lock()
	c.respec += n
	c.mu.Unlock()
}

func (c *totalCounters) addRespecSilver(n int64) {
	c.mu.Lock()
	c.respecSilver += n
	c.mu.Unlock()
}

func (c *totalCounters) addKill() {
	c.mu.Lock()
	c.kills++
	c.mu.Unlock()
}

func (c *totalCounters) addDeath() {
	c.mu.Lock()
	c.deaths++
	c.mu.Unlock()
}

func (c *totalCounters) addLoot() {
	c.mu.Lock()
	c.loot++
	c.mu.Unlock()
}

func (c *totalCounters) fameNow() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.fame
}

func (c *totalCounters) silverNow() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.silver
}

func (c *totalCounters) respecNow() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.respec
}

func (c *totalCounters) respecSilverNow() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.respecSilver
}

func (c *totalCounters) killsNow() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return int(c.kills)
}

func (c *totalCounters) deathsNow() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return int(c.deaths)
}

func (c *totalCounters) lootNow() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return int(c.loot)
}

// AlbionHandler handles Albion Online game events
type AlbionHandler struct {
	debug     bool
	discovery bool

	// Fame tracking
	totalFame atomic.Int64

	// Cumulative session counters (fame, silver, respec, respecSilver, kills,
	// deaths, loot). Grouped under one RWMutex so snapshots are consistent.
	stats totalCounters

	// Items database
	itemDB *items.ItemDatabase

	// Mob database for dungeon tier classification (loaded from mobs.json).
	mobDB *MobDatabase

	// Discovery mode tracking
	discoveredEvents map[int16]*DiscoveredEvent
	discoveryMu      sync.RWMutex

	// Local player identity, auto-detected from the OpJoin response (operation
	// code 2, parameters[2] = Username). Used to filter foreign loot and attribute
	// kills/deaths to the local player. Empty until the first OpJoin is captured.
	localPlayer   string
	localPlayerMu sync.RWMutex

	// Current zone identity, auto-detected from the ChangeCluster response
	// (operation code 41, parameters[0] = cluster string). Used to show which
	// map the player is currently in. Zero-value (MapTypeUnknown, empty
	// ClusterIndex) until the first ChangeCluster is captured.
	currentZone ZoneInfo
	zoneMu      sync.RWMutex

	// Recent death deduplication. When the local player kills someone, the server
	// delivers EventDied redundantly (multiple copies within the same second).
	// We dedup by a (victim, killer) signature within deathDedupWindow.
	recentDeaths     map[string]time.Time
	deathsMu         sync.Mutex
	deathDedupWindow time.Duration
	lastDeathsPurge  time.Time
	nowFunc          func() time.Time

	// Dungeon run tracking. Single active run detected via OpJoin and
	// ChangeCluster transitions (MapType == RandomDungeon). Classification
	// is lazy: mode/faction from shrine/chest events, tier from mob events.
	dungeonMu   sync.Mutex
	dungeonRuns []*DungeonRun
	activeRun   *DungeonRun

	// Event callback for frontend integration (TUI, Wails, etc.)
	eventCallback EventCallback
}

// DiscoveredEvent tracks unknown events in discovery mode
type DiscoveredEvent struct {
	Code       int16                `json:"code"`
	Count      int                  `json:"count"`
	FirstSeen  time.Time            `json:"first_seen"`
	LastSeen   time.Time            `json:"last_seen"`
	SampleData map[byte]interface{} `json:"sample_data"`
	ParamTypes map[byte]string      `json:"param_types"`
}

// NewAlbionHandler creates a new Albion event handler
func NewAlbionHandler() *AlbionHandler {
	return &AlbionHandler{
		discoveredEvents: make(map[int16]*DiscoveredEvent),
		recentDeaths:     make(map[string]time.Time),
		deathDedupWindow: defaultDeathDedupWindow,
		nowFunc:          time.Now,
	}
}

// SetDebug enables or disables debug output
func (h *AlbionHandler) SetDebug(debug bool) {
	h.debug = debug
}

// SetDiscoveryMode enables discovery mode to log all unknown events
func (h *AlbionHandler) SetDiscoveryMode(discovery bool) {
	h.discovery = discovery
}

// SetEventCallback sets a callback function for TUI integration
func (h *AlbionHandler) SetEventCallback(callback EventCallback) {
	h.eventCallback = callback
}

// SetLocalPlayer records the local player's name, auto-detected from the OpJoin
// response. An empty argument is ignored.
func (h *AlbionHandler) SetLocalPlayer(name string) {
	if name == "" {
		return
	}
	h.localPlayerMu.Lock()
	h.localPlayer = name
	h.localPlayerMu.Unlock()
}

// GetLocalPlayer returns the local player's name, or "" if the OpJoin response
// has not been received yet (e.g. the tool was started after the game was
// already running and no map change / relog has occurred since).
func (h *AlbionHandler) GetLocalPlayer() string {
	h.localPlayerMu.RLock()
	defer h.localPlayerMu.RUnlock()
	return h.localPlayer
}

// GetCurrentZone returns the current zone identity, or a zero-value ZoneInfo
// (MapTypeUnknown, empty ClusterIndex) if the ChangeCluster response has not
// been received yet (same "unknown until detected" semantics as GetLocalPlayer).
func (h *AlbionHandler) GetCurrentZone() ZoneInfo {
	h.zoneMu.RLock()
	defer h.zoneMu.RUnlock()
	return h.currentZone
}

// handleChangeCluster parses the ChangeCluster response and forwards it to
// processZoneTransition. ChangeCluster fires for open-world cluster changes
// (city, overworld, island). Dungeon entries are primarily detected via OpJoin.
func (h *AlbionHandler) handleChangeCluster(params map[byte]interface{}) {
	h.processZoneTransition(parseChangeClusterResponse(params))
}

// processZoneTransition updates the current zone state, detects dungeon
// entry/exit, and emits a zone event. Called from both ChangeCluster and Join
// handlers so that dungeon detection works regardless of which operation the
// server sends. Duplicate transitions (same map type, cluster index, and
// island name) are suppressed. Empty fields on the incoming ZoneInfo are
// merged from the previous zone so that a partial update (e.g. OpJoin, which
// only carries ClusterIndex and MapType) does not clobber richer data already
// set by ChangeCluster (IslandName, MainClusterIndex, HasDungeonInfo).
func (h *AlbionHandler) processZoneTransition(newZone ZoneInfo) {
	h.zoneMu.RLock()
	prev := h.currentZone
	h.zoneMu.RUnlock()

	// Merge: preserve non-zero fields from prev when the incoming zone
	// leaves them empty (OpJoin sends a partial ZoneInfo).
	if newZone.IslandName == "" {
		newZone.IslandName = prev.IslandName
	}
	if newZone.MainClusterIndex == "" {
		newZone.MainClusterIndex = prev.MainClusterIndex
	}
	if !newZone.HasDungeonInfo {
		newZone.HasDungeonInfo = prev.HasDungeonInfo
	}

	// Dedup: the server may resend the same response or deliver the same
	// transition via both Join and ChangeCluster. Skip if nothing materially
	// changed. HasDungeonInfo is excluded — it's metadata that may differ
	// between Join (always false) and ChangeCluster (may be true) for the
	// same zone, and doesn't warrant a duplicate event.
	if newZone.MapType == prev.MapType &&
		newZone.ClusterIndex == prev.ClusterIndex &&
		newZone.IslandName == prev.IslandName &&
		newZone.MainClusterIndex == prev.MainClusterIndex {
		// Silently update HasDungeonInfo if the new zone has it but the
		// stored one doesn't (ChangeCluster arriving after OpJoin).
		if newZone.HasDungeonInfo && !prev.HasDungeonInfo {
			h.zoneMu.Lock()
			h.currentZone.HasDungeonInfo = true
			h.zoneMu.Unlock()
		}
		return
	}

	h.zoneMu.Lock()
	h.currentZone = newZone
	h.zoneMu.Unlock()

	// Detect random dungeon entry/exit for run tracking. Guard against
	// double-entry when both Join and ChangeCluster fire for the same
	// dungeon transition (prev and new both RandomDungeon → skip).
	now := h.nowFunc()
	if newZone.MapType == MapTypeRandomDungeon && prev.MapType != MapTypeRandomDungeon {
		h.EnterDungeon(now)
	} else if newZone.MapType != MapTypeRandomDungeon && prev.MapType == MapTypeRandomDungeon {
		h.ExitDungeon(now)
	}

	h.notifyEvent("zone", "", &ZoneEventData{
		MapType:      newZone.MapType,
		Display:      newZone.DisplayString(),
		IslandName:   newZone.IslandName,
		ClusterIndex: newZone.ClusterIndex,
		Previous:     prev.MapType,
	})
}

// notifyEvent calls the event callback if set
func (h *AlbionHandler) notifyEvent(eventType, message string, data interface{}) {
	if h.eventCallback != nil {
		h.eventCallback(eventType, message, data)
	}
}

// debugNotify emits a debug event only when debug mode is enabled. Used for
// diagnostic signals (identity capture, kill routing, override decisions) that
// should not pollute the normal event log.
func (h *AlbionHandler) debugNotify(message string, data interface{}) {
	if h.debug {
		h.notifyEvent("debug", message, data)
	}
}

// FameEventData contains fame-specific event data
type FameEventData struct {
	Gained  int64 // Fame gained in this event
	Total   int64 // Total fame pool reported by the server after this event
	AllTime int64 // Cumulative fame gained across all launches (long-term total)
}

// SilverEventData contains silver-specific event data
type SilverEventData struct {
	Amount     int64  // Silver amount in this event
	Total      int64  // Cumulative silver gained across all launches (long-term total)
	LootedBy   string // Player who looted
	LootedFrom string // Source of the loot
	Counted    bool   // Whether this amount was added to the long-term total
}

// LootEventData contains loot-specific event data
type LootEventData struct {
	LootedBy   string // Player who looted
	ItemName   string // Name of the item
	Quantity   int32  // Quantity of the item
	LootedFrom string // Source of the loot
}

// KillEventData contains kill-specific event data
type KillEventData struct {
	TotalKills int // Cumulative kills across all launches (long-term total)
}

// DeathEventData contains death-specific event data
type DeathEventData struct {
	Victim      string // Player who died
	Killer      string // Player who killed
	TotalDeaths int    // Cumulative deaths across all launches (long-term total)
}

// RespecEventData contains respec-specific event data
type RespecEventData struct {
	Gained      int64 // Credits gained in this event
	PaidSilver  int64 // Silver paid for auto-respec in this event
	Total       int64 // Cumulative credits gained across all launches (long-term total)
	TotalSilver int64 // Cumulative silver spent on auto-respec across all launches
}

// ZoneEventData contains zone-transition event data emitted on ChangeCluster.
type ZoneEventData struct {
	MapType      MapType // The map type entered
	Display      string  // Human-friendly label (ZoneInfo.DisplayString())
	IslandName   string  // Island name (non-empty only on islands)
	ClusterIndex string  // Raw cluster string from param[0]
	Previous     MapType // The zone the player left (MapTypeUnknown on first transition)
}

// GetTotalKills returns the number of kills in this session
func (h *AlbionHandler) GetTotalKills() int {
	return h.stats.killsNow()
}

// GetTotalDeaths returns the number of deaths in this session
func (h *AlbionHandler) GetTotalDeaths() int {
	return h.stats.deathsNow()
}

// GetTotalLoot returns the number of loot items in this session
func (h *AlbionHandler) GetTotalLoot() int {
	return h.stats.lootNow()
}

// LoadItemDatabase loads the item database from ao-bin-dumps
func (h *AlbionHandler) LoadItemDatabase(path string) error {
	h.itemDB = items.GetDatabase()
	return h.itemDB.LoadFromPath(path)
}

// LoadMobDatabase loads the mob database from ao-bin-dumps for dungeon tier
// classification. Errors are non-fatal (tier classification is best-effort).
func (h *AlbionHandler) LoadMobDatabase(path string) error {
	db, err := LoadMobDatabase(path)
	if err != nil {
		return err
	}
	h.mobDB = db
	return nil
}

// resolveOperationCode extracts the actual operation code from param[253].
// Albion embeds the real code in the parameter dictionary, which can differ
// from the Photon header byte. Mirrors the reference AlbionParser which
// ignores the header byte and reads from parameters[253].
func resolveOperationCode(headerCode byte, parameters map[byte]interface{}) byte {
	if code, ok := parameters[events.ParamOperationCode]; ok {
		switch v := code.(type) {
		case byte:
			return v
		case int:
			return byte(v)
		case int16:
			return byte(v)
		case int32:
			return byte(v)
		case int64:
			return byte(v)
		}
	}
	return headerCode
}

// OnRequest handles operation requests (client -> server)
func (h *AlbionHandler) OnRequest(operationCode byte, parameters map[byte]interface{}) {
	// Requests are not logged to avoid polluting TUI output
}

// OnResponse handles operation responses (server -> client)
func (h *AlbionHandler) OnResponse(operationCode byte, returnCode int16, debugMessage string, parameters map[byte]interface{}) {
	// Albion embeds the real operation code in param[253], not the Photon header byte.
	operationCode = resolveOperationCode(operationCode, parameters)

	switch operationCode {
	case events.OperationJoin:
		// The Join response carries the local player's identity.
		// parameters[2] is the Username (character name).
		if name := getString(parameters, 2); name != "" {
			h.debugNotify("local player detected from OpJoin", name)
			h.SetLocalPlayer(name)
		} else {
			h.debugNotify("OpJoin response had no player name", nil)
		}

		// The Join response also carries the map identity in param[8]
		// (MapIndex). This is the primary signal for dungeon entry/exit —
		// the reference project (JoinResponseHandler) calls AddDungeon from
		// here, not from ChangeCluster. param[65]=SourceClusterIndex,
		// param[64]=exit position.
		if mapIndex := getString(parameters, 8); mapIndex != "" {
			h.processZoneTransition(ZoneInfo{
				ClusterIndex: mapIndex,
				MapType:      ParseMapType(mapIndex),
			})
		}

		// Debug: surface the full OpJoin map identity for live confirmation.
		if h.debug {
			sourceCluster := getString(parameters, 65)
			posX, posY, hasPos := getFloatPair(parameters, 64)
			if sourceCluster != "" || hasPos {
				pos := "?"
				if hasPos {
					pos = fmt.Sprintf("(%.0f,%.0f)", posX, posY)
				}
				h.debugNotify(fmt.Sprintf("OpJoin source=%q pos=%s", sourceCluster, pos), nil)
			}
		}

	case events.OperationChangeCluster:
		h.handleChangeCluster(parameters)
	}
}

// OnEvent handles incoming game events
func (h *AlbionHandler) OnEvent(eventCode byte, parameters map[byte]interface{}) {
	// Get actual event code from parameter 252 if available
	actualEventCode := events.EventCode(eventCode)
	if code, ok := parameters[events.ParamEventCode]; ok {
		switch v := code.(type) {
		case int16:
			actualEventCode = events.EventCode(v)
		case int32:
			actualEventCode = events.EventCode(v)
		case int64:
			actualEventCode = events.EventCode(v)
		}
	}

	handled := false

	switch actualEventCode {
	case events.EventUpdateFame:
		h.handleUpdateFame(parameters)
		handled = true

	case events.EventUpdateMoney:
		h.handleUpdateMoney(parameters)
		handled = true

	case events.EventHealthUpdate:
		h.handleHealthUpdate(parameters)
		handled = true

	case events.EventNewCharacter:
		h.handleNewCharacter(parameters)
		handled = true

	case events.EventOtherGrabbedLoot:
		h.handleOtherGrabbedLoot(parameters)
		handled = true

	case events.EventNewLoot:
		h.handleNewLoot(parameters)
		handled = true

	case events.EventKilledPlayer:
		h.handleKilledPlayer(parameters)
		handled = true

	case events.EventDied:
		h.handleDied(parameters)
		handled = true

	case events.EventUpdateReSpecPoints:
		h.handleUpdateReSpecPoints(parameters)
		handled = true

	case events.EventNewShrine:
		h.handleNewShrine(parameters)
		handled = true

	case events.EventNewLootChest:
		h.handleNewLootChest(parameters)
		handled = true

	case events.EventNewMob:
		h.handleNewMob(parameters)
		handled = true

	case events.EventSimpleFeedback:
		// Debug-only: capture params to confirm whether this is the dungeon
		// "closed" notification. The reference defines it but does not handle it.
		if h.debug {
			h.debugNotify(fmt.Sprintf("SimpleFeedback %s", dumpParams(parameters)), nil)
		}
		handled = true

	default:
		if h.debug {
			// Pass "debug" type and the raw event code as data.
			// The TUI applies interactive filtering (high-frequency events are
			// hidden by default but toggleable); see internal/tui debug filter.
			h.notifyEvent("debug", "", actualEventCode)
		}
	}

	// Discovery mode: track all events (including handled ones for completeness)
	if h.discovery {
		h.trackDiscoveredEvent(int16(actualEventCode), parameters, handled)
	}
}

// trackDiscoveredEvent records event details in discovery mode
func (h *AlbionHandler) trackDiscoveredEvent(code int16, params map[byte]interface{}, handled bool) {
	h.discoveryMu.Lock()
	defer h.discoveryMu.Unlock()

	event, exists := h.discoveredEvents[code]
	if !exists {
		event = &DiscoveredEvent{
			Code:       code,
			Count:      0,
			FirstSeen:  time.Now(),
			SampleData: make(map[byte]interface{}),
			ParamTypes: make(map[byte]string),
		}
		h.discoveredEvents[code] = event
	}

	event.Count++
	event.LastSeen = time.Now()

	// Store sample data and types (only first occurrence or if new params appear)
	for key, val := range params {
		if _, exists := event.ParamTypes[key]; !exists {
			event.ParamTypes[key] = fmt.Sprintf("%T", val)
			event.SampleData[key] = val
		}
	}
}

// GetDiscoveredEvents returns all discovered events
func (h *AlbionHandler) GetDiscoveredEvents() map[int16]*DiscoveredEvent {
	h.discoveryMu.RLock()
	defer h.discoveryMu.RUnlock()

	// Return a copy
	result := make(map[int16]*DiscoveredEvent)
	for k, v := range h.discoveredEvents {
		result[k] = v
	}
	return result
}

// SaveDiscoveredEvents saves discovered events to a JSON file
func (h *AlbionHandler) SaveDiscoveredEvents(filename string) error {
	h.discoveryMu.RLock()
	defer h.discoveryMu.RUnlock()

	// Create output directory if it doesn't exist
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Convert to a serializable format
	output := make(map[string]*DiscoveredEvent)
	for code, event := range h.discoveredEvents {
		output[fmt.Sprintf("%d", code)] = event
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}

// SaveDungeonRuns persists the completed dungeon run history to path as JSON.
// The active run (if any) is excluded because it is still incomplete. The
// snapshot is copied under the dungeon lock and the (potentially slow) atomic
// write happens without holding it, so dungeon tracking is not blocked on I/O.
func (h *AlbionHandler) SaveDungeonRuns(path string) error {
	h.dungeonMu.Lock()
	toSave := make([]*DungeonRun, 0, len(h.dungeonRuns))
	for _, r := range h.dungeonRuns {
		if r.Status == RunStatusActive {
			continue
		}
		cp := *r
		toSave = append(toSave, &cp)
	}
	h.dungeonMu.Unlock()

	return storage.Save(path, toSave)
}

// LoadDungeonRuns replaces the in-memory run history with the runs decoded from
// path. Runs are sorted oldest-first by EnteredAt and trimmed to
// maxDungeonRuns. A missing file is treated as an empty history (first run); a
// corrupt file is returned as err and leaves the current history unchanged. The
// active run is always reset to nil after a load.
func (h *AlbionHandler) LoadDungeonRuns(path string) error {
	runs, err := storage.Load[[]*DungeonRun](path)
	if err != nil {
		return err
	}
	sort.Slice(runs, func(i, j int) bool {
		return runs[i].EnteredAt.Before(runs[j].EnteredAt)
	})
	if len(runs) > maxDungeonRuns {
		runs = runs[len(runs)-maxDungeonRuns:]
	}

	h.dungeonMu.Lock()
	h.dungeonRuns = runs
	h.activeRun = nil
	h.dungeonMu.Unlock()
	return nil
}

// GetTotalFame returns the total fame gained in this session
func (h *AlbionHandler) GetTotalFame() int64 {
	return h.stats.fameNow()
}

// GetTotalSilver returns the total silver looted in this session
func (h *AlbionHandler) GetTotalSilver() int64 {
	return h.stats.silverNow()
}

// GetTotalRespec returns the total combat fame credits gained in this session
func (h *AlbionHandler) GetTotalRespec() int64 {
	return h.stats.respecNow()
}

// GetTotalRespecSilver returns the total silver spent on auto-respec this session
func (h *AlbionHandler) GetTotalRespecSilver() int64 {
	return h.stats.respecSilverNow()
}

// SessionStats is a point-in-time snapshot of the cumulative session counters,
// used to hydrate the UI on startup. All fields are absolute totals read
// atomically.
type TotalStats struct {
	Fame         int64
	Silver       int64
	Respec       int64
	RespecSilver int64
	Kills        int64
	Deaths       int64
	Loot         int64
}

// TotalSnapshot returns the current cumulative session counters. Safe to
// call concurrently with event processing.
func (h *AlbionHandler) TotalSnapshot() TotalStats {
	return h.readTotalStats()
}

// readTotalStats is the single source of truth for reading the seven
// cumulative counters into a consistent SessionStats snapshot.
func (h *AlbionHandler) readTotalStats() TotalStats {
	return h.stats.snapshot()
}

// applyTotalStats is the single source of truth for writing the seven
// cumulative counters from a SessionStats (the inverse of readTotalStats).
func (h *AlbionHandler) applyTotalStats(s TotalStats) {
	h.stats.apply(s)
}

// totalStatsFile is the on-disk JSON representation of SessionStats. The
// Version field is reserved for future additive schema migrations; missing
// fields in an old file default to zero via encoding/json.
//
// Once persisted and restored, the session* counters represent cumulative
// totals across all launches of the application (long-term) rather than a
// single session — this matches issue #104's intent. The "session" naming is
// retained for now and will be revisited when daily/hourly bucketing lands
// (follow-up #105).
type totalStatsFile struct {
	Version      int       `json:"version"`
	Fame         int64     `json:"fame"`
	Silver       int64     `json:"silver"`
	Respec       int64     `json:"respec"`
	RespecSilver int64     `json:"respec_silver"`
	Kills        int64     `json:"kills"`
	Deaths       int64     `json:"deaths"`
	Loot         int64     `json:"loot"`
	SavedAt      time.Time `json:"saved_at"`
}

// totalStatsVersion is the current on-disk schema version for session stats.
const totalStatsVersion = 1

// SaveTotalStats persists the cumulative session counters to path as JSON
// using an atomic write (tmp + rename).
func (h *AlbionHandler) SaveTotalStats(path string) error {
	s := h.readTotalStats()
	f := totalStatsFile{
		Version:      totalStatsVersion,
		Fame:         s.Fame,
		Silver:       s.Silver,
		Respec:       s.Respec,
		RespecSilver: s.RespecSilver,
		Kills:        s.Kills,
		Deaths:       s.Deaths,
		Loot:         s.Loot,
		SavedAt:      h.nowFunc(),
	}
	return storage.Save(path, f)
}

// LoadTotalStats restores the cumulative session counters from path. A
// missing file leaves counters at zero (first run). A corrupt file is returned
// as err and leaves the counters unchanged. The schema is additive: a version
// mismatch or missing fields apply best-effort (unknown/absent fields default
// to zero).
func (h *AlbionHandler) LoadTotalStats(path string) error {
	f, err := storage.Load[totalStatsFile](path)
	if err != nil {
		return err
	}
	h.applyTotalStats(TotalStats{
		Fame:         f.Fame,
		Silver:       f.Silver,
		Respec:       f.Respec,
		RespecSilver: f.RespecSilver,
		Kills:        f.Kills,
		Deaths:       f.Deaths,
		Loot:         f.Loot,
	})
	return nil
}

// handleUpdateFame handles fame/XP gain events
// Supports multiple event formats as they vary between game versions
func (h *AlbionHandler) handleUpdateFame(params map[byte]interface{}) {
	// Detect event format based on available parameters
	// Format 1 (Event #81 simple): [0]=playerID, [1]=totalFame
	// Format 2 (Event #82 detailed): [0]=playerID, [1]=totalFame, [2]=gained, [3]=zone

	// Get total fame from parameter 1
	totalFame := getInt64(params, 1)

	// Validation: Total fame should be a large number (> 1 million in FixPoint = 100 fame)
	// This helps filter out events with similar structure but different purpose
	if totalFame < 1000000 {
		return
	}

	// Deduplication: Server sends both Event #81 and #82 for the same fame gain
	// Skip if we already processed an event with this exact totalFame
	if totalFame == h.totalFame.Load() {
		return
	}

	// Check if we have additional parameters (Format 2)
	hasDetailedFormat := false
	var fameGained int64
	var zoneFame int64

	if val, ok := params[2]; ok {
		hasDetailedFormat = true
		fameGained = toInt64(val)
	}
	if val, ok := params[3]; ok {
		zoneFame = toInt64(val)
	}

	// Validation: Total fame should not decrease significantly
	// This helps filter out events with similar structure but different purpose
	if prev := h.totalFame.Load(); prev > 0 && totalFame < prev {
		return
	}

	// Calculate values (divide by 10000 for FixPoint format)
	// Use Floor (truncate) to match game's display behavior
	totalFameVal := math.Floor(float64(totalFame) / 10000.0)

	if hasDetailedFormat {
		// Detailed format: we have the actual gained fame
		fameGainedVal := math.Floor(float64(fameGained) / 10000.0)
		_ = zoneFame // Zone fame available but not displayed in simplified view

		// Only notify if fame was actually gained
		if fameGainedVal > 0 {
			h.stats.addFame(int64(fameGainedVal))
			h.totalFame.Store(totalFame) // Update tracked total

			// Message formatting is now handled by the frontend (TUI)
			h.notifyEvent("fame", "", &FameEventData{
				Gained:  int64(fameGainedVal),
				Total:   int64(totalFameVal),
				AllTime: h.stats.fameNow(),
			})
		}
	} else {
		// Simple format: we only have total fame
		// Calculate gained by comparing with previous total
		if h.totalFame.Load() > 0 {
			gained := totalFame - h.totalFame.Load()
			if gained > 0 {
				gainedVal := math.Floor(float64(gained) / 10000.0)
				h.stats.addFame(int64(gainedVal))
				// Message formatting is now handled by the frontend (TUI)
				h.notifyEvent("fame", "", &FameEventData{
					Gained:  int64(gainedVal),
					Total:   int64(totalFameVal),
					AllTime: h.stats.fameNow(),
				})
			}
		}
		h.totalFame.Store(totalFame)
	}
}

// toInt64 converts an interface{} to int64
func toInt64(val interface{}) int64 {
	switch v := val.(type) {
	case int64:
		return v
	case int32:
		return int64(v)
	case int16:
		return int64(v)
	case int:
		return int64(v)
	case uint8:
		return int64(v)
	case float32:
		return int64(v)
	case float64:
		return int64(v)
	}
	return 0
}

// handleUpdateMoney handles silver balance update events
// Note: We don't notify here because silver gains are already captured by
// handleOtherGrabbedLoot. This event only shows total balance, which would
// cause duplicate entries in the event log.
func (h *AlbionHandler) handleUpdateMoney(params map[byte]interface{}) {
	// Silver balance updates are tracked but not notified to avoid duplication
	// The actual silver gains are captured via EventOtherGrabbedLoot
}

// handleHealthUpdate handles health update events (debug only, no callback)
func (h *AlbionHandler) handleHealthUpdate(params map[byte]interface{}) {
	// Health updates are too frequent to notify, used only for debug
}

// handleNewCharacter handles new character events (debug only, no callback)
func (h *AlbionHandler) handleNewCharacter(params map[byte]interface{}) {
	// New character events are informational only
}

// handleOtherGrabbedLoot handles when another player loots something
func (h *AlbionHandler) handleOtherGrabbedLoot(params map[byte]interface{}) {
	// Parameter 1: Looted from
	lootedFrom := getString(params, 1)

	// Parameter 2: Looted by
	lootedBy := getString(params, 2)

	// Parameter 3: Is silver
	isSilver := getBool(params, 3)

	// Parameter 4: Item ID
	itemID := getInt32(params, 4)

	// Parameter 5: Quantity
	quantity := getInt32(params, 5)

	if isSilver {
		silverAmountRaw := getInt64(params, 5)
		// Silver also uses FixPoint format (divide by 10000)
		silverAmount := int64(math.Floor(float64(silverAmountRaw) / 10000.0))

		// Only silver looted by the local player counts toward the session total.
		// The event is still notified (so other players' loot stays visible in the
		// log), but it is excluded from the running total when looted by someone
		// else. Before the local player is known (local == ""), nothing is
		// counted to avoid inflating the session with other players' loot.
		local := h.GetLocalPlayer()
		counted := local != "" && lootedBy == local
		if counted {
			h.stats.addSilver(silverAmount)
		}
		// Message formatting is now handled by the frontend (TUI)
		// We just pass the raw data
		h.notifyEvent("silver", "", &SilverEventData{
			Amount:     silverAmount,
			Total:      h.stats.silverNow(),
			LootedBy:   lootedBy,
			LootedFrom: lootedFrom,
			Counted:    counted,
		})
	} else {
		// Try to get item name from database
		itemName := fmt.Sprintf("Item#%d", itemID)
		if h.itemDB != nil && h.itemDB.IsLoaded() {
			itemName = h.itemDB.GetItemName(itemID)
		}

		// Item loot counts every event the client sees (not just the local
		// player's loot), because the total volume of nearby loot is useful
		// context. Silver is different — only the local player's silver
		// contributes to the session total (see the silver branch above).
		h.stats.addLoot()

		// Message formatting is now handled by the frontend (TUI)
		h.notifyEvent("loot", "", &LootEventData{
			LootedBy:   lootedBy,
			ItemName:   itemName,
			Quantity:   quantity,
			LootedFrom: lootedFrom,
		})
	}
}

// handleNewLoot handles new loot available events (debug only, no callback)
func (h *AlbionHandler) handleNewLoot(params map[byte]interface{}) {
	// New loot events are informational only
}

// handleKilledPlayer handles player kill events.
//
// EventKilledPlayer (164) is only ever delivered to the local player when they are
// the killer, and it carries no victim identifier. When the local player kills
// someone, the server delivers it redundantly alongside EventDied. Because it has
// no distinguishable payload, it cannot be deduped per-kill.
//
// Kill counting is therefore owned by handleDied (EventDied carries both victim and
// killer names, so distinct kills are distinguishable and can be deduped correctly).
// This handler is intentionally a no-op.
func (h *AlbionHandler) handleKilledPlayer(params map[byte]interface{}) {
	// No-op: see method comment. Kills are counted via EventDied. Emit a debug
	// signal so the receipt of this event stays traceable in debug mode.
	h.debugNotify("EventKilledPlayer received (counted via EventDied)", nil)
}

// isDuplicateDeath reports whether the given death signature was already seen
// within deathDedupWindow. Otherwise it records the signature and returns false.
// Expired entries are purged at most once per dedup window (lazy cleanup) to
// avoid O(n) iteration on every call during high-frequency combat (e.g., ZvZ).
func (h *AlbionHandler) isDuplicateDeath(key string) bool {
	now := h.nowFunc()

	h.deathsMu.Lock()
	defer h.deathsMu.Unlock()

	if now.Sub(h.lastDeathsPurge) >= h.deathDedupWindow {
		cutoff := now.Add(-h.deathDedupWindow)
		for k, t := range h.recentDeaths {
			if t.Before(cutoff) {
				delete(h.recentDeaths, k)
			}
		}
		h.lastDeathsPurge = now
	}

	if _, ok := h.recentDeaths[key]; ok {
		return true
	}
	h.recentDeaths[key] = now
	return false
}

// handleDied handles death events.
//
// EventDied (165) is a world broadcast delivered once when other players die, but
// delivered redundantly (multiple copies within the same second) when the local
// player is the killer. We dedup by a (victim, killer) signature so each death is
// processed exactly once.
//
// This handler is the single source of truth for both kills and deaths:
//   - a kill is counted when the local player is the killer;
//   - a death is counted when the local player is the victim.
func (h *AlbionHandler) handleDied(params map[byte]interface{}) {
	victim := getString(params, 2)
	killer := getString(params, 10)

	local := h.GetLocalPlayer()

	// Dedup redundant copies of the same death. The server delivers EventDied
	// multiple times when the local player is the killer. Only apply dedup in
	// that case — for other deaths each EventDied is a distinct event, and
	// applying dedup before the local player is known would consume the dedup
	// window and permanently lose a legitimate kill.
	if local != "" && killer == local {
		key := victim + "\x00" + killer // use raw victim name (before "Someone" default)
		if h.isDuplicateDeath(key) {
			return
		}
	}

	// Default display name for anonymous victims
	if victim == "" {
		victim = "Someone"
	}

	isLocalKill := local != "" && killer == local

	// Count a kill when the local player is the killer.
	if isLocalKill {
		h.stats.addKill()
		h.notifyEvent("kill", "", &KillEventData{
			TotalKills: h.stats.killsNow(),
		})
	}

	// Count a death only when the local player is the victim.
	if local != "" && victim == local {
		h.stats.addDeath()
	}

	// Emit a death event for the local player's death or for nearby deaths
	// (neither party is the local player). Skip when the local player is the
	// killer — the kill event already covers it and avoids a duplicate entry.
	if !isLocalKill {
		h.notifyEvent("death", "", &DeathEventData{
			Victim:      victim,
			Killer:      killer,
			TotalDeaths: h.stats.deathsNow(),
		})
	}
}

// handleUpdateReSpecPoints handles combat fame credit (respec) gain events
// Event #84: server sends credits gained and silver paid for auto-respec
func (h *AlbionHandler) handleUpdateReSpecPoints(params map[byte]interface{}) {
	gainedReSpec := getInt64(params, 2)
	if gainedReSpec <= 0 {
		return
	}

	paidSilver := getInt64(params, 3)

	gainedVal := int64(math.Floor(float64(gainedReSpec) / 10000.0))
	silverVal := int64(math.Floor(float64(paidSilver) / 10000.0))

	h.stats.addRespec(gainedVal)
	h.stats.addRespecSilver(silverVal)

	h.notifyEvent("respec", "", &RespecEventData{
		Gained:      gainedVal,
		PaidSilver:  silverVal,
		Total:       h.stats.respecNow(),
		TotalSilver: h.stats.respecSilverNow(),
	})
}

// classifyRunFromUniqueName extracts mode and faction from a shrine/chest
// uniqueName and updates the active run lazily. Shared by shrine and chest
// handlers since both follow the same classification path.
func (h *AlbionHandler) classifyRunFromUniqueName(uniqueName string) {
	if uniqueName == "" {
		return
	}
	mode := ClassifyDungeonMode(uniqueName)
	faction := ClassifyDungeonFaction(uniqueName)
	if mode != DungeonModeUnknown {
		h.UpdateActiveRunMode(mode)
	}
	if faction != "" {
		h.UpdateActiveRunFaction(faction)
	}
}

// handleNewShrine handles EventNewShrine (code 395). The shrine uniqueName
// (param[3]) is used for lazy dungeon mode classification.
func (h *AlbionHandler) handleNewShrine(params map[byte]interface{}) {
	uniqueName := getString(params, 3)
	if h.debug {
		h.debugNotify(fmt.Sprintf("NewShrine uniqueName=%q", uniqueName), nil)
	}
	h.classifyRunFromUniqueName(uniqueName)
}

// handleNewLootChest handles EventNewLootChest (code 391). The chest uniqueName
// (param[3]) is used for lazy dungeon mode + faction classification.
func (h *AlbionHandler) handleNewLootChest(params map[byte]interface{}) {
	uniqueName := getString(params, 3)
	if h.debug {
		h.debugNotify(fmt.Sprintf("NewLootChest uniqueName=%q", uniqueName), nil)
	}
	h.classifyRunFromUniqueName(uniqueName)
}

// handleNewMob handles EventNewMob (code 123). The mob index (param[1]) is used
// for lazy dungeon tier classification via the mob database.
func (h *AlbionHandler) handleNewMob(params map[byte]interface{}) {
	mobIndex := int(getInt32(params, 1))
	if mobIndex == 0 {
		return
	}
	tier := h.mobDB.GetRandomDungeonMobTier(mobIndex)
	if h.debug {
		h.debugNotify(fmt.Sprintf("NewMob index=%d tier=%d", mobIndex, tier), nil)
	}
	if tier >= 0 {
		h.UpdateActiveRunTier(tier)
	}
}

// Helper functions to extract typed values from parameters
func getInt64(params map[byte]interface{}, key byte) int64 {
	if val, ok := params[key]; ok {
		switch v := val.(type) {
		case int64:
			return v
		case int32:
			return int64(v)
		case int16:
			return int64(v)
		case int:
			return int64(v)
		}
	}
	return 0
}

func getInt32(params map[byte]interface{}, key byte) int32 {
	if val, ok := params[key]; ok {
		switch v := val.(type) {
		case int32:
			return v
		case int64:
			return int32(v)
		case int16:
			return int32(v)
		case int:
			return int32(v)
		}
	}
	return 0
}

func getString(params map[byte]interface{}, key byte) string {
	if val, ok := params[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getBool(params map[byte]interface{}, key byte) bool {
	if val, ok := params[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return false
}

// getFloatPair extracts a 2-element numeric array (X, Y) from params[key].
// Handles the array encodings the photon decoder may produce for a world
// position: []interface{}, []float32, []float64, []int, []int32, []int64.
// Returns ok=false when the key is missing, not an array, or has fewer than 2
// elements. Mirrors the reference ParseWorldPosition acceptance policy.
func getFloatPair(params map[byte]interface{}, key byte) (x, y float64, ok bool) {
	val, found := params[key]
	if !found {
		return 0, 0, false
	}

	switch arr := val.(type) {
	case []interface{}:
		if len(arr) < 2 {
			return 0, 0, false
		}
		x, okX := toFloat(arr[0])
		y, okY := toFloat(arr[1])
		return x, y, okX && okY
	case []float32:
		if len(arr) < 2 {
			return 0, 0, false
		}
		return float64(arr[0]), float64(arr[1]), true
	case []float64:
		if len(arr) < 2 {
			return 0, 0, false
		}
		return arr[0], arr[1], true
	case []int:
		if len(arr) < 2 {
			return 0, 0, false
		}
		return float64(arr[0]), float64(arr[1]), true
	case []int32:
		if len(arr) < 2 {
			return 0, 0, false
		}
		return float64(arr[0]), float64(arr[1]), true
	case []int64:
		if len(arr) < 2 {
			return 0, 0, false
		}
		return float64(arr[0]), float64(arr[1]), true
	}
	return 0, 0, false
}

// toFloat coerces a numeric interface{} value to float64.
func toFloat(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float32:
		return float64(n), true
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	}
	return 0, false
}

// dumpParams renders a parameter map as a compact, key-sorted string for debug
// logging. Byte slices and large arrays are truncated to keep output readable.
func dumpParams(params map[byte]interface{}) string {
	keys := make([]int, 0, len(params))
	for k := range params {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		v := params[byte(k)]
		s := fmt.Sprintf("%v", v)
		if len(s) > 40 {
			s = s[:37] + "..."
		}
		parts = append(parts, fmt.Sprintf("[%d]=%s", k, s))
	}
	return "{" + strings.Join(parts, " ") + "}"
}
