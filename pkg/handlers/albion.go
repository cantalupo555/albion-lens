// Package handlers implements event handlers for Albion Online game events
package handlers

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/cantalupo555/albion-lens/pkg/events"
	"github.com/cantalupo555/albion-lens/pkg/items"
	"github.com/fatih/color"
)

// EventCallback is called when a game event is processed
// eventType: "fame", "silver", "loot", "combat", "info"
// message: formatted message to display
type EventCallback func(eventType, message string)

// AlbionHandler handles Albion Online game events
type AlbionHandler struct {
	debug     bool
	discovery bool

	// Fame tracking
	totalFame   int64
	sessionFame int64

	// Silver tracking
	sessionSilver int64

	// Kill/Death tracking
	sessionKills  int
	sessionDeaths int
	sessionLoot   int

	// Items database
	itemDB *items.ItemDatabase

	// Discovery mode tracking
	discoveredEvents map[int16]*DiscoveredEvent
	discoveryMu      sync.RWMutex

	// Output colors
	fameColor      *color.Color
	silverColor    *color.Color
	lootColor      *color.Color
	combatColor    *color.Color
	infoColor      *color.Color
	discoveryColor *color.Color

	// Event callback for TUI integration
	eventCallback EventCallback
}

// DiscoveredEvent tracks unknown events in discovery mode
type DiscoveredEvent struct {
	Code       int16                  `json:"code"`
	Count      int                    `json:"count"`
	FirstSeen  time.Time              `json:"first_seen"`
	LastSeen   time.Time              `json:"last_seen"`
	SampleData map[byte]interface{}   `json:"sample_data"`
	ParamTypes map[byte]string        `json:"param_types"`
}

// NewAlbionHandler creates a new Albion event handler
func NewAlbionHandler() *AlbionHandler {
	return &AlbionHandler{
		debug:            false,
		discovery:        false,
		discoveredEvents: make(map[int16]*DiscoveredEvent),
		fameColor:        color.New(color.FgGreen, color.Bold),
		silverColor:      color.New(color.FgYellow, color.Bold),
		lootColor:        color.New(color.FgMagenta, color.Bold),
		combatColor:      color.New(color.FgRed, color.Bold),
		infoColor:        color.New(color.FgCyan),
		discoveryColor:   color.New(color.FgHiBlue),
	}
}

// SetDebug enables or disables debug output
func (h *AlbionHandler) SetDebug(debug bool) {
	h.debug = debug
}

// SetDiscoveryMode enables discovery mode to log all unknown events
func (h *AlbionHandler) SetDiscoveryMode(discovery bool) {
	h.discovery = discovery
	if discovery {
		h.discoveryColor.Println("🔍 Discovery mode enabled - logging all events")
	}
}

// SetEventCallback sets a callback function for TUI integration
func (h *AlbionHandler) SetEventCallback(callback EventCallback) {
	h.eventCallback = callback
}

// notifyEvent calls the event callback if set
func (h *AlbionHandler) notifyEvent(eventType, message string) {
	if h.eventCallback != nil {
		h.eventCallback(eventType, message)
	}
}

// GetSessionKills returns the number of kills in this session
func (h *AlbionHandler) GetSessionKills() int {
	return h.sessionKills
}

// GetSessionDeaths returns the number of deaths in this session
func (h *AlbionHandler) GetSessionDeaths() int {
	return h.sessionDeaths
}

// GetSessionLoot returns the number of loot items in this session
func (h *AlbionHandler) GetSessionLoot() int {
	return h.sessionLoot
}

// LoadItemDatabase loads the item database from ao-bin-dumps
func (h *AlbionHandler) LoadItemDatabase(path string) error {
	h.itemDB = items.GetDatabase()
	if err := h.itemDB.LoadFromPath(path); err != nil {
		return err
	}
	h.infoColor.Printf("📚 Loaded %d items from database\n", h.itemDB.ItemCount())
	return nil
}

// OnRequest handles operation requests (client -> server)
func (h *AlbionHandler) OnRequest(operationCode byte, parameters map[byte]interface{}) {
	if h.debug {
		fmt.Printf("  [Request] op=%d params=%v\n", operationCode, parameters)
	}
}

// OnResponse handles operation responses (server -> client)
func (h *AlbionHandler) OnResponse(operationCode byte, returnCode int16, debugMessage string, parameters map[byte]interface{}) {
	if h.debug {
		fmt.Printf("  [Response] op=%d return=%d params=%v\n", operationCode, returnCode, parameters)
	}
}

// OnEvent handles game events (server -> client)
func (h *AlbionHandler) OnEvent(eventCode byte, parameters map[byte]interface{}) {
	// Get actual event code from parameter 252 if available
	actualEventCode := int16(eventCode)
	if code, ok := parameters[events.ParamEventCode]; ok {
		switch v := code.(type) {
		case int16:
			actualEventCode = v
		case int32:
			actualEventCode = int16(v)
		case int64:
			actualEventCode = int16(v)
		}
	}

	handled := false

	switch int(actualEventCode) {
	case events.EventUpdateFame, events.EventUpdateFameDetails:
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

	default:
		if h.debug {
			fmt.Printf("  [Event] code=%d params=%v\n", actualEventCode, parameters)
		}
	}

	// Discovery mode: track all events (including handled ones for completeness)
	if h.discovery {
		h.trackDiscoveredEvent(actualEventCode, parameters, handled)
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

		// Log new unknown event
		if !handled {
			h.logNewUnknownEvent(code, params)
		}
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

// logNewUnknownEvent logs a newly discovered unknown event
func (h *AlbionHandler) logNewUnknownEvent(code int16, params map[byte]interface{}) {
	timestamp := time.Now().Format("15:04:05")
	h.discoveryColor.Printf("[%s] 🆕 NEW EVENT #%d discovered!\n", timestamp, code)
	
	// Sort parameter keys for consistent output
	keys := make([]int, 0, len(params))
	for k := range params {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)

	for _, k := range keys {
		key := byte(k)
		val := params[key]
		h.discoveryColor.Printf("    [%d] (%T) = %v\n", key, val, truncateValue(val))
	}
}

// truncateValue truncates long values for display
func truncateValue(val interface{}) interface{} {
	switch v := val.(type) {
	case string:
		if len(v) > 100 {
			return v[:100] + "..."
		}
	case []byte:
		if len(v) > 50 {
			return fmt.Sprintf("[%d bytes]", len(v))
		}
	case []interface{}:
		if len(v) > 10 {
			return fmt.Sprintf("[array of %d items]", len(v))
		}
	}
	return val
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

// PrintDiscoverySummary prints a summary of discovered events
func (h *AlbionHandler) PrintDiscoverySummary() {
	h.discoveryMu.RLock()
	defer h.discoveryMu.RUnlock()

	if len(h.discoveredEvents) == 0 {
		fmt.Println("\nNo events discovered during this session.")
		return
	}

	fmt.Println("\n╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║             📊 DISCOVERY MODE SUMMARY                      ║")
	fmt.Println("╠════════════════════════════════════════════════════════════╣")

	// Sort by event code
	codes := make([]int, 0, len(h.discoveredEvents))
	for code := range h.discoveredEvents {
		codes = append(codes, int(code))
	}
	sort.Ints(codes)

	knownCount := 0
	unknownCount := 0

	for _, code := range codes {
		event := h.discoveredEvents[int16(code)]
		isKnown := h.isKnownEventCode(int16(code))
		
		status := "❓"
		if isKnown {
			status = "✅"
			knownCount++
		} else {
			unknownCount++
		}

		fmt.Printf("║ %s Event #%-4d | Count: %-6d | Params: %-2d            ║\n",
			status, code, event.Count, len(event.ParamTypes))
	}

	fmt.Println("╠════════════════════════════════════════════════════════════╣")
	fmt.Printf("║ Total: %d events (%d known, %d unknown)                   ║\n",
		len(h.discoveredEvents), knownCount, unknownCount)
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
}

// isKnownEventCode checks if an event code is in our known list
func (h *AlbionHandler) isKnownEventCode(code int16) bool {
	knownCodes := []int{
		events.EventUnused, events.EventLeave, events.EventJoinFinished,
		events.EventMove, events.EventTeleport, events.EventHealthUpdate,
		events.EventHealthUpdates, events.EventEnergyUpdate, events.EventAttack,
		events.EventCastStart, events.EventCastHit, events.EventKilledPlayer,
		events.EventDied, events.EventKnockedDown, events.EventInventoryPutItem,
		events.EventInventoryDeleteItem, events.EventNewCharacter,
		events.EventNewEquipmentItem, events.EventNewSiegeBannerItem,
		events.EventNewSimpleItem, events.EventNewFurnitureItem,
		events.EventHarvestStart, events.EventHarvestCancel,
		events.EventHarvestFinished, events.EventTakeSilver,
		events.EventUpdateMoney, events.EventUpdateFame, events.EventUpdateFameDetails,
		events.EventNewLoot, events.EventAttachItemContainer,
		events.EventDetachItemContainer, events.EventCharacterStats,
		events.EventPartyInvitation, events.EventPartyJoinRequest,
		events.EventPartyJoined, events.EventPartyDisbanded,
		events.EventPartyPlayerJoined, events.EventPartyPlayerLeft,
		events.EventOtherGrabbedLoot, events.EventInCombatState,
	}

	for _, known := range knownCodes {
		if int(code) == known {
			return true
		}
	}
	return false
}

// GetSessionFame returns the total fame gained in this session
func (h *AlbionHandler) GetSessionFame() int64 {
	return h.sessionFame
}

// GetSessionSilver returns the total silver looted in this session
func (h *AlbionHandler) GetSessionSilver() int64 {
	return h.sessionSilver
}

// handleUpdateFame handles fame/XP gain events
// Supports multiple event formats as they vary between game versions
func (h *AlbionHandler) handleUpdateFame(params map[byte]interface{}) {
	timestamp := time.Now().Format("15:04:05")

	// Debug: show raw params
	if h.debug {
		fmt.Printf("  [Fame Debug] params=%v\n", params)
	}

	// Detect event format based on available parameters
	// Format 1 (Event #81 simple): [0]=playerID, [1]=totalFame
	// Format 2 (Event #82 detailed): [0]=playerID, [1]=totalFame, [2]=gained, [3]=zone
	
	// Get total fame from parameter 1
	totalFame := getInt64(params, 1)
	
	// Validation: Total fame should be a large number (> 1 million in FixPoint = 100 fame)
	// This helps filter out events with similar structure but different purpose
	if totalFame < 1000000 {
		if h.debug {
			fmt.Printf("  [Fame] Ignored: total fame too low (%d), likely not a fame event\n", totalFame)
		}
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
	if h.totalFame > 0 && totalFame < h.totalFame {
		if h.debug {
			fmt.Printf("  [Fame] Ignored: total decreased from %d to %d\n", h.totalFame, totalFame)
		}
		return
	}
	
	// Calculate values (divide by 10000 for FixPoint format)
	// Use Floor (truncate) to match game's display behavior
	totalFameVal := math.Floor(float64(totalFame) / 10000.0)
	
	if hasDetailedFormat {
		// Detailed format: we have the actual gained fame
		fameGainedVal := math.Floor(float64(fameGained) / 10000.0)
		_ = zoneFame // Zone fame available but not displayed in simplified view

		// Only show if fame was actually gained
		if fameGainedVal > 0 {
			h.sessionFame += int64(fameGainedVal)
			h.totalFame = totalFame // Update tracked total

			msg := fmt.Sprintf("⭐ FAME: +%.0f | Total: %.0f | Session: %d", fameGainedVal, totalFameVal, h.sessionFame)
			h.fameColor.Printf("[%s] %s\n", timestamp, msg)
			h.notifyEvent("fame", msg)
		} else if h.debug {
			fmt.Printf("  [Fame] Total fame update: %.0f (no gain)\n", totalFameVal)
		}
	} else {
		// Simple format: we only have total fame
		// Calculate gained by comparing with previous total
		if h.totalFame > 0 {
			gained := totalFame - h.totalFame
			if gained > 0 {
				gainedVal := math.Floor(float64(gained) / 10000.0)
				h.sessionFame += int64(gainedVal)
				msg := fmt.Sprintf("⭐ FAME: +%.0f | Total: %.0f | Session: %d", gainedVal, totalFameVal, h.sessionFame)
				h.fameColor.Printf("[%s] %s\n", timestamp, msg)
				h.notifyEvent("fame", msg)
			}
		} else if h.debug {
			// First fame event, just record the total
			fmt.Printf("  [Fame] Initial total: %.0f\n", totalFameVal)
		}
		h.totalFame = totalFame
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

// handleUpdateMoney handles silver gain events
func (h *AlbionHandler) handleUpdateMoney(params map[byte]interface{}) {
	timestamp := time.Now().Format("15:04:05")

	// Parameter 1: Current silver
	currentSilver := getInt64(params, 1)

	msg := fmt.Sprintf("💰 SILVER: %s", formatSilver(currentSilver))
	h.silverColor.Printf("[%s] %s\n", timestamp, msg)
	h.notifyEvent("silver", msg)
}

// handleHealthUpdate handles health update events
func (h *AlbionHandler) handleHealthUpdate(params map[byte]interface{}) {
	if !h.debug {
		return
	}

	timestamp := time.Now().Format("15:04:05")
	h.combatColor.Printf("[%s] ❤️ Health Update: %v\n", timestamp, params)
}

// handleNewCharacter handles new character events
func (h *AlbionHandler) handleNewCharacter(params map[byte]interface{}) {
	if !h.debug {
		return
	}

	// Parameters vary, but usually include player name and guild info
	h.infoColor.Printf("👤 New Character: %v\n", params)
}

// handleOtherGrabbedLoot handles when another player loots something
func (h *AlbionHandler) handleOtherGrabbedLoot(params map[byte]interface{}) {
	timestamp := time.Now().Format("15:04:05")

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
		h.sessionSilver += silverAmount
		msg := fmt.Sprintf("💰 %s looted silver (%s) from %s | Session: %s",
			lootedBy, formatSilver(silverAmount), lootedFrom, formatSilver(h.sessionSilver))
		h.silverColor.Printf("[%s] %s\n", timestamp, msg)
		h.notifyEvent("silver", msg)
	} else {
		// Try to get item name from database
		itemName := fmt.Sprintf("Item#%d", itemID)
		if h.itemDB != nil && h.itemDB.IsLoaded() {
			itemName = h.itemDB.GetItemName(itemID)
		}

		h.sessionLoot++
		msg := fmt.Sprintf("📦 %s looted %s (x%d) from %s", lootedBy, itemName, quantity, lootedFrom)
		h.lootColor.Printf("[%s] %s\n", timestamp, msg)
		h.notifyEvent("loot", msg)
	}
}

// handleNewLoot handles new loot available events
func (h *AlbionHandler) handleNewLoot(params map[byte]interface{}) {
	if !h.debug {
		return
	}

	timestamp := time.Now().Format("15:04:05")
	h.lootColor.Printf("[%s] 📦 New Loot: %v\n", timestamp, params)
}

// handleKilledPlayer handles player kill events
func (h *AlbionHandler) handleKilledPlayer(params map[byte]interface{}) {
	timestamp := time.Now().Format("15:04:05")
	h.sessionKills++
	msg := fmt.Sprintf("⚔️ Player Killed! (Session: %d kills)", h.sessionKills)
	h.combatColor.Printf("[%s] %s\n", timestamp, msg)
	h.notifyEvent("kill", msg)
}

// handleDied handles death events
func (h *AlbionHandler) handleDied(params map[byte]interface{}) {
	timestamp := time.Now().Format("15:04:05")
	h.sessionDeaths++
	msg := fmt.Sprintf("💀 You died! (Session: %d deaths)", h.sessionDeaths)
	h.combatColor.Printf("[%s] %s\n", timestamp, msg)
	h.notifyEvent("death", msg)
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

// formatSilver formats silver amount in a human-readable way
func formatSilver(amount int64) string {
	if amount >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(amount)/1000000.0)
	} else if amount >= 1000 {
		return fmt.Sprintf("%.1fk", float64(amount)/1000.0)
	}
	return fmt.Sprintf("%d", amount)
}
