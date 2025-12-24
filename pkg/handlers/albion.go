// Package handlers implements event handlers for Albion Online game events
package handlers

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/cantalupo555/albion-lens/pkg/events"
	"github.com/cantalupo555/albion-lens/pkg/items"
)

// EventCallback is called when a game event is processed
// eventType: "fame", "silver", "loot", "combat", "info", "death", "kill"
// message: formatted message to display
// data: optional structured data (FameEventData, SilverEventData, etc.)
type EventCallback func(eventType, message string, data interface{})

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

	// Event callback for frontend integration (TUI, Wails, etc.)
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
		discoveredEvents: make(map[int16]*DiscoveredEvent),
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

// notifyEvent calls the event callback if set
func (h *AlbionHandler) notifyEvent(eventType, message string, data interface{}) {
	if h.eventCallback != nil {
		h.eventCallback(eventType, message, data)
	}
}

// FameEventData contains fame-specific event data
type FameEventData struct {
	Gained  int64 // Fame gained in this event
	Total   int64 // Total fame after this event
	Session int64 // Total fame gained this session
}

// SilverEventData contains silver-specific event data
type SilverEventData struct {
	Amount  int64 // Silver amount in this event
	Session int64 // Total silver gained this session
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
	return h.itemDB.LoadFromPath(path)
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

	// Deduplication: Server sends both Event #81 and #82 for the same fame gain
	// Skip if we already processed an event with this exact totalFame
	if totalFame == h.totalFame {
		if h.debug {
			fmt.Printf("  [Fame] Ignored: duplicate event (totalFame=%d already processed)\n", totalFame)
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

		// Only notify if fame was actually gained
		if fameGainedVal > 0 {
			h.sessionFame += int64(fameGainedVal)
			h.totalFame = totalFame // Update tracked total

			msg := fmt.Sprintf("⭐ FAME: +%.0f | Total: %.0f | Session: %d", fameGainedVal, totalFameVal, h.sessionFame)
			h.notifyEvent("fame", msg, &FameEventData{
				Gained:  int64(fameGainedVal),
				Total:   int64(totalFameVal),
				Session: h.sessionFame,
			})
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
				h.notifyEvent("fame", msg, &FameEventData{
					Gained:  int64(gainedVal),
					Total:   int64(totalFameVal),
					Session: h.sessionFame,
				})
			}
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
		h.sessionSilver += silverAmount
		msg := fmt.Sprintf("💰 %s looted silver (%s) from %s | Session: %s",
			lootedBy, formatSilver(silverAmount), lootedFrom, formatSilver(h.sessionSilver))
		h.notifyEvent("silver", msg, &SilverEventData{
			Amount:  silverAmount,
			Session: h.sessionSilver,
		})
	} else {
		// Try to get item name from database
		itemName := fmt.Sprintf("Item#%d", itemID)
		if h.itemDB != nil && h.itemDB.IsLoaded() {
			itemName = h.itemDB.GetItemName(itemID)
		}

		h.sessionLoot++
		msg := fmt.Sprintf("📦 %s looted %s (x%d) from %s", lootedBy, itemName, quantity, lootedFrom)
		h.notifyEvent("loot", msg, nil)
	}
}

// handleNewLoot handles new loot available events (debug only, no callback)
func (h *AlbionHandler) handleNewLoot(params map[byte]interface{}) {
	// New loot events are informational only
}

// handleKilledPlayer handles player kill events
func (h *AlbionHandler) handleKilledPlayer(params map[byte]interface{}) {
	h.sessionKills++
	msg := fmt.Sprintf("⚔️ Player Killed! (Session: %d kills)", h.sessionKills)
	h.notifyEvent("kill", msg, nil)
}

// handleDied handles death events
func (h *AlbionHandler) handleDied(params map[byte]interface{}) {
	h.sessionDeaths++
	msg := fmt.Sprintf("💀 You died! (Session: %d deaths)", h.sessionDeaths)
	h.notifyEvent("death", msg, nil)
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
