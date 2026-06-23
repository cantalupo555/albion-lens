package handlers

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/cantalupo555/albion-lens/pkg/events"
)

// TestNewAlbionHandler tests handler creation
func TestNewAlbionHandler(t *testing.T) {
	handler := NewAlbionHandler()

	if handler == nil {
		t.Fatal("NewAlbionHandler returned nil")
	}

	if handler.discoveredEvents == nil {
		t.Error("discoveredEvents map not initialized")
	}

	if handler.debug != false {
		t.Error("debug should default to false")
	}

	if handler.discovery != false {
		t.Error("discovery should default to false")
	}
}

// TestSetDebug tests debug mode toggle
func TestSetDebug(t *testing.T) {
	handler := NewAlbionHandler()

	handler.SetDebug(true)
	if handler.debug != true {
		t.Error("SetDebug(true) failed")
	}

	handler.SetDebug(false)
	if handler.debug != false {
		t.Error("SetDebug(false) failed")
	}
}

// TestSetDiscoveryMode tests discovery mode toggle
func TestSetDiscoveryMode(t *testing.T) {
	handler := NewAlbionHandler()

	handler.SetDiscoveryMode(true)
	if handler.discovery != true {
		t.Error("SetDiscoveryMode(true) failed")
	}

	handler.SetDiscoveryMode(false)
	if handler.discovery != false {
		t.Error("SetDiscoveryMode(false) failed")
	}
}

// TestSetEventCallback tests callback registration
func TestSetEventCallback(t *testing.T) {
	handler := NewAlbionHandler()

	called := false
	callback := func(eventType, message string, data interface{}) {
		called = true
	}

	handler.SetEventCallback(callback)
	handler.notifyEvent("test", "test message", nil)

	if !called {
		t.Error("callback was not called")
	}
}

// TestNotifyEventNoCallback tests that notifyEvent doesn't panic without callback
func TestNotifyEventNoCallback(t *testing.T) {
	handler := NewAlbionHandler()

	// Should not panic
	handler.notifyEvent("test", "test message", nil)
}

// TestSessionCounters tests kill/death/loot counters
func TestSessionCounters(t *testing.T) {
	handler := NewAlbionHandler()
	handler.SetLocalPlayer("Hero")

	// Initial values should be 0
	if handler.GetSessionKills() != 0 {
		t.Error("initial kills should be 0")
	}
	if handler.GetSessionDeaths() != 0 {
		t.Error("initial deaths should be 0")
	}
	if handler.GetSessionLoot() != 0 {
		t.Error("initial loot should be 0")
	}

	// EventKilledPlayer is a no-op; kills are counted via EventDied.
	handler.OnEvent(byte(events.EventKilledPlayer), map[byte]interface{}{})
	if handler.GetSessionKills() != 0 {
		t.Errorf("EventKilledPlayer should not count kills, got %d", handler.GetSessionKills())
	}

	// Local player kills someone -> kill counted.
	handler.OnEvent(byte(events.EventDied), map[byte]interface{}{
		2:  "Enemy",
		10: "Hero",
	})
	if handler.GetSessionKills() != 1 {
		t.Errorf("expected 1 kill, got %d", handler.GetSessionKills())
	}

	// Local player dies -> death counted.
	handler.OnEvent(byte(events.EventDied), map[byte]interface{}{
		2:  "Hero",
		10: "Enemy",
	})
	if handler.GetSessionDeaths() != 1 {
		t.Errorf("expected 1 death, got %d", handler.GetSessionDeaths())
	}
}

// TestGetSessionFame tests fame getter
func TestGetSessionFame(t *testing.T) {
	handler := NewAlbionHandler()

	if handler.GetSessionFame() != 0 {
		t.Error("initial session fame should be 0")
	}
}

// TestGetSessionSilver tests silver getter
func TestGetSessionSilver(t *testing.T) {
	handler := NewAlbionHandler()

	if handler.GetSessionSilver() != 0 {
		t.Error("initial session silver should be 0")
	}
}

// TestHandleUpdateFameDetailedFormat tests fame handling with detailed format (Event #82)
func TestHandleUpdateFameDetailedFormat(t *testing.T) {
	handler := NewAlbionHandler()

	var receivedData *FameEventData
	handler.SetEventCallback(func(eventType, message string, data interface{}) {
		if eventType == "fame" {
			receivedData = data.(*FameEventData)
		}
	})

	// Simulate detailed fame event (Event #82)
	// Values are in FixPoint format (multiply by 10000)
	params := map[byte]interface{}{
		0:                     int64(123456),      // Player ID
		1:                     int64(50000000000), // Total fame (5M in FixPoint)
		2:                     int64(10000000),    // Gained fame (1000 in FixPoint)
		3:                     int64(0),           // Zone fame
		events.ParamEventCode: int16(events.EventUpdateFame),
	}

	handler.OnEvent(byte(events.EventUpdateFame), params)

	if receivedData == nil {
		t.Fatal("fame callback was not called")
	}

	if receivedData.Gained != 1000 {
		t.Errorf("expected gained 1000, got %d", receivedData.Gained)
	}

	if handler.GetSessionFame() != 1000 {
		t.Errorf("expected session fame 1000, got %d", handler.GetSessionFame())
	}
}

// TestHandleUpdateFameSimpleFormat tests fame handling with simple format (Event #81)
func TestHandleUpdateFameSimpleFormat(t *testing.T) {
	handler := NewAlbionHandler()
	// Set initial total fame
	handler.totalFame.Store(int64(40000000000)) // 4M in FixPoint

	var receivedData *FameEventData
	handler.SetEventCallback(func(eventType, message string, data interface{}) {
		if eventType == "fame" {
			receivedData = data.(*FameEventData)
		}
	})

	// Simulate simple fame event (Event #81) - gain of 500 fame
	// 500 fame in FixPoint = 500 * 10000 = 5000000
	params := map[byte]interface{}{
		0:                     int64(123456),      // Player ID
		1:                     int64(40005000000), // New total fame (4M + 500 in FixPoint)
		events.ParamEventCode: int16(events.EventUpdateFame),
	}

	handler.OnEvent(byte(events.EventUpdateFame), params)

	if receivedData == nil {
		t.Fatal("fame callback was not called")
	}

	if receivedData.Gained != 500 {
		t.Errorf("expected gained 500, got %d", receivedData.Gained)
	}
}

// TestHandleUpdateFameDuplicateIgnored tests that duplicate fame events are ignored
func TestHandleUpdateFameDuplicateIgnored(t *testing.T) {
	handler := NewAlbionHandler()

	callCount := 0
	handler.SetEventCallback(func(eventType, message string, data interface{}) {
		if eventType == "fame" {
			callCount++
		}
	})

	// First fame event
	params := map[byte]interface{}{
		1: int64(50000000000), // Total fame
		2: int64(10000000),    // Gained fame
	}
	handler.OnEvent(byte(events.EventUpdateFame), params)

	// Duplicate event with same total fame
	handler.OnEvent(byte(events.EventUpdateFame), params)

	if callCount != 1 {
		t.Errorf("expected 1 callback, got %d (duplicate should be ignored)", callCount)
	}
}

// TestHandleUpdateFameLowTotalIgnored tests that low total fame values are ignored
func TestHandleUpdateFameLowTotalIgnored(t *testing.T) {
	handler := NewAlbionHandler()

	callCount := 0
	handler.SetEventCallback(func(eventType, message string, data interface{}) {
		if eventType == "fame" {
			callCount++
		}
	})

	// Fame event with low total (below threshold)
	params := map[byte]interface{}{
		1: int64(500000), // Total fame below 1M threshold
		2: int64(10000),  // Gained fame
	}
	handler.OnEvent(byte(events.EventUpdateFame), params)

	if callCount != 0 {
		t.Errorf("expected 0 callbacks (low fame should be ignored), got %d", callCount)
	}
}

// TestHandleOtherGrabbedLootSilver tests silver loot handling
func TestHandleOtherGrabbedLootSilver(t *testing.T) {
	handler := NewAlbionHandler()
	handler.SetLocalPlayer("Player1")

	var receivedData *SilverEventData
	handler.SetEventCallback(func(eventType, message string, data interface{}) {
		if eventType == "silver" {
			receivedData = data.(*SilverEventData)
		}
	})

	// Simulate silver loot event
	// Silver is in FixPoint format (multiply by 10000)
	// Note: EventOtherGrabbedLoot (275) > 255, so we pass it via ParamEventCode
	params := map[byte]interface{}{
		1:                     "Monster",       // Looted from
		2:                     "Player1",       // Looted by
		3:                     true,            // Is silver
		4:                     int32(0),        // Item ID (0 for silver)
		5:                     int64(50000000), // Quantity (5000 silver in FixPoint)
		events.ParamEventCode: int16(events.EventOtherGrabbedLoot),
	}

	handler.OnEvent(0, params) // Event code comes from param 252

	if receivedData == nil {
		t.Fatal("silver callback was not called")
	}

	if receivedData.Amount != 5000 {
		t.Errorf("expected 5000 silver, got %d", receivedData.Amount)
	}

	if receivedData.LootedBy != "Player1" {
		t.Errorf("expected LootedBy 'Player1', got '%s'", receivedData.LootedBy)
	}

	if receivedData.LootedFrom != "Monster" {
		t.Errorf("expected LootedFrom 'Monster', got '%s'", receivedData.LootedFrom)
	}

	if handler.GetSessionSilver() != 5000 {
		t.Errorf("expected session silver 5000, got %d", handler.GetSessionSilver())
	}
}

// TestHandleOtherGrabbedLootItem tests item loot handling
func TestHandleOtherGrabbedLootItem(t *testing.T) {
	handler := NewAlbionHandler()

	var receivedData *LootEventData
	handler.SetEventCallback(func(eventType, message string, data interface{}) {
		if eventType == "loot" {
			if lootData, ok := data.(*LootEventData); ok {
				receivedData = lootData
			}
		}
	})

	// Simulate item loot event
	// Note: EventOtherGrabbedLoot (275) > 255, so we pass it via ParamEventCode
	params := map[byte]interface{}{
		1:                     "Chest",      // Looted from
		2:                     "Player1",    // Looted by
		3:                     false,        // Is silver (false = item)
		4:                     int32(12345), // Item ID
		5:                     int32(3),     // Quantity
		events.ParamEventCode: int16(events.EventOtherGrabbedLoot),
	}

	handler.OnEvent(0, params) // Event code comes from param 252

	if receivedData == nil {
		t.Fatal("loot callback was not called")
	}

	if receivedData.LootedBy != "Player1" {
		t.Errorf("expected LootedBy 'Player1', got %s", receivedData.LootedBy)
	}

	if receivedData.LootedFrom != "Chest" {
		t.Errorf("expected LootedFrom 'Chest', got %s", receivedData.LootedFrom)
	}

	if receivedData.Quantity != 3 {
		t.Errorf("expected Quantity 3, got %d", receivedData.Quantity)
	}

	if handler.GetSessionLoot() != 1 {
		t.Errorf("expected session loot 1, got %d", handler.GetSessionLoot())
	}
}

// TestHandleKilledPlayer tests that EventKilledPlayer is a no-op. Kills are
// counted via EventDied (where killer == localPlayer), which carries a victim
// identifier and can be deduped per kill.
func TestHandleKilledPlayer(t *testing.T) {
	handler := NewAlbionHandler()
	handler.SetLocalPlayer("Hero")

	called := false
	handler.SetEventCallback(func(eventType, message string, data interface{}) {
		if eventType == "kill" {
			called = true
		}
	})

	handler.OnEvent(byte(events.EventKilledPlayer), map[byte]interface{}{})

	if called {
		t.Error("EventKilledPlayer should not produce a kill callback")
	}

	if handler.GetSessionKills() != 0 {
		t.Errorf("expected 0 kills (no-op), got %d", handler.GetSessionKills())
	}
}

// TestHandleDied tests death event handling and the victim name default.
func TestHandleDied(t *testing.T) {
	handler := NewAlbionHandler()

	var receivedData *DeathEventData
	handler.SetEventCallback(func(eventType, message string, data interface{}) {
		if eventType == "death" {
			if deathData, ok := data.(*DeathEventData); ok {
				receivedData = deathData
			}
		}
	})

	// Empty victim name defaults to "Someone".
	handler.OnEvent(byte(events.EventDied), map[byte]interface{}{})

	if receivedData == nil {
		t.Fatal("death callback was not called")
	}

	if receivedData.Victim != "Someone" {
		t.Errorf("expected Victim 'Someone', got %s", receivedData.Victim)
	}

	// Without a known local player, deaths are not counted.
	if receivedData.SessionDeaths != 0 {
		t.Errorf("expected SessionDeaths 0 (no local player), got %d", receivedData.SessionDeaths)
	}

	if handler.GetSessionDeaths() != 0 {
		t.Errorf("expected 0 deaths (no local player), got %d", handler.GetSessionDeaths())
	}
}

// TestDiscoveryModeTracking tests event discovery tracking
func TestDiscoveryModeTracking(t *testing.T) {
	handler := NewAlbionHandler()
	handler.SetDiscoveryMode(true)

	// Trigger an event
	params := map[byte]interface{}{
		1: int32(100),
		2: "test",
	}
	handler.OnEvent(50, params)

	// Check discovered events
	discovered := handler.GetDiscoveredEvents()
	if len(discovered) != 1 {
		t.Errorf("expected 1 discovered event, got %d", len(discovered))
	}

	event, exists := discovered[50]
	if !exists {
		t.Fatal("event code 50 not found in discovered events")
	}

	if event.Count != 1 {
		t.Errorf("expected count 1, got %d", event.Count)
	}

	if event.Code != 50 {
		t.Errorf("expected code 50, got %d", event.Code)
	}

	// Trigger same event again
	handler.OnEvent(50, params)

	discovered = handler.GetDiscoveredEvents()
	event = discovered[50]
	if event.Count != 2 {
		t.Errorf("expected count 2 after second event, got %d", event.Count)
	}
}

// TestDiscoveryModeParamTypes tests that param types are recorded correctly
func TestDiscoveryModeParamTypes(t *testing.T) {
	handler := NewAlbionHandler()
	handler.SetDiscoveryMode(true)

	params := map[byte]interface{}{
		1: int32(100),
		2: "test",
		3: true,
		4: int64(999),
	}
	handler.OnEvent(60, params)

	discovered := handler.GetDiscoveredEvents()
	event := discovered[60]

	expectedTypes := map[byte]string{
		1: "int32",
		2: "string",
		3: "bool",
		4: "int64",
	}

	for key, expectedType := range expectedTypes {
		if event.ParamTypes[key] != expectedType {
			t.Errorf("param %d: expected type %s, got %s", key, expectedType, event.ParamTypes[key])
		}
	}
}

// TestSaveDiscoveredEvents tests saving discovered events to file
func TestSaveDiscoveredEvents(t *testing.T) {
	handler := NewAlbionHandler()
	handler.SetDiscoveryMode(true)

	// Trigger some events
	handler.OnEvent(100, map[byte]interface{}{1: int32(1)})
	handler.OnEvent(101, map[byte]interface{}{1: "test"})

	// Save to temp file
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "subdir", "discovered.json")

	err := handler.SaveDiscoveredEvents(filename)
	if err != nil {
		t.Fatalf("SaveDiscoveredEvents failed: %v", err)
	}

	// Check file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		t.Error("output file was not created")
	}
}

// TestGetDiscoveredEventsReturnsCopy tests that GetDiscoveredEvents returns a copy
func TestGetDiscoveredEventsReturnsCopy(t *testing.T) {
	handler := NewAlbionHandler()
	handler.SetDiscoveryMode(true)

	handler.OnEvent(50, map[byte]interface{}{})

	discovered := handler.GetDiscoveredEvents()
	// Modify the returned map
	delete(discovered, 50)

	// Original should still have the event
	original := handler.GetDiscoveredEvents()
	if _, exists := original[50]; !exists {
		t.Error("GetDiscoveredEvents should return a copy, not the original map")
	}
}

// TestOnEventWithParamEventCode tests that event code is read from param 252
func TestOnEventWithParamEventCode(t *testing.T) {
	handler := NewAlbionHandler()

	called := false
	handler.SetEventCallback(func(eventType, message string, data interface{}) {
		if eventType == "death" {
			called = true
		}
	})

	// Send event with event code in param 252
	params := map[byte]interface{}{
		events.ParamEventCode: int16(events.EventDied),
	}
	handler.OnEvent(0, params) // byte code is 0, but actual code is in param 252

	if !called {
		t.Error("event code from param 252 was not used")
	}
}

// TestOnEventParamEventCodeConversion tests different types for param event code
func TestOnEventParamEventCodeConversion(t *testing.T) {
	testCases := []struct {
		name    string
		codeVal interface{}
	}{
		{"int16", int16(events.EventDied)},
		{"int32", int32(events.EventDied)},
		{"int64", int64(events.EventDied)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			h := NewAlbionHandler()
			called := false
			h.SetEventCallback(func(eventType, message string, data interface{}) {
				if eventType == "death" {
					called = true
				}
			})
			params := map[byte]interface{}{
				events.ParamEventCode: tc.codeVal,
			}
			h.OnEvent(0, params)
			if !called {
				t.Errorf("expected death callback with %s, got none", tc.name)
			}
		})
	}
}

// TestConcurrentDiscoveryAccess tests thread safety of discovery mode
func TestConcurrentDiscoveryAccess(t *testing.T) {
	handler := NewAlbionHandler()
	handler.SetDiscoveryMode(true)

	var wg sync.WaitGroup

	// Spawn multiple goroutines to trigger events
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(code int) {
			defer wg.Done()
			handler.OnEvent(byte(code%10), map[byte]interface{}{1: int32(code)})
		}(i)
	}

	// Also spawn goroutines reading discovered events
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = handler.GetDiscoveredEvents()
		}()
	}

	wg.Wait()

	// If we get here without race conditions, the test passes
}

// TestHelperGetInt64 tests the getInt64 helper function
func TestHelperGetInt64(t *testing.T) {
	params := map[byte]interface{}{
		1: int64(100),
		2: int32(200),
		3: int16(300),
		4: int(400),
		5: "not a number",
	}

	if v := getInt64(params, 1); v != 100 {
		t.Errorf("expected 100, got %d", v)
	}
	if v := getInt64(params, 2); v != 200 {
		t.Errorf("expected 200, got %d", v)
	}
	if v := getInt64(params, 3); v != 300 {
		t.Errorf("expected 300, got %d", v)
	}
	if v := getInt64(params, 4); v != 400 {
		t.Errorf("expected 400, got %d", v)
	}
	if v := getInt64(params, 5); v != 0 {
		t.Errorf("expected 0 for string, got %d", v)
	}
	if v := getInt64(params, 99); v != 0 {
		t.Errorf("expected 0 for missing key, got %d", v)
	}
}

// TestHelperGetInt32 tests the getInt32 helper function
func TestHelperGetInt32(t *testing.T) {
	params := map[byte]interface{}{
		1: int32(100),
		2: int64(200),
		3: int16(300),
		4: int(400),
	}

	if v := getInt32(params, 1); v != 100 {
		t.Errorf("expected 100, got %d", v)
	}
	if v := getInt32(params, 2); v != 200 {
		t.Errorf("expected 200, got %d", v)
	}
	if v := getInt32(params, 3); v != 300 {
		t.Errorf("expected 300, got %d", v)
	}
	if v := getInt32(params, 4); v != 400 {
		t.Errorf("expected 400, got %d", v)
	}
}

// TestHelperGetString tests the getString helper function
func TestHelperGetString(t *testing.T) {
	params := map[byte]interface{}{
		1: "hello",
		2: int32(100),
	}

	if v := getString(params, 1); v != "hello" {
		t.Errorf("expected 'hello', got '%s'", v)
	}
	if v := getString(params, 2); v != "" {
		t.Errorf("expected empty string for int, got '%s'", v)
	}
	if v := getString(params, 99); v != "" {
		t.Errorf("expected empty string for missing key, got '%s'", v)
	}
}

// TestHelperGetBool tests the getBool helper function
func TestHelperGetBool(t *testing.T) {
	params := map[byte]interface{}{
		1: true,
		2: false,
		3: "true", // string, not bool
	}

	if v := getBool(params, 1); v != true {
		t.Error("expected true, got false")
	}
	if v := getBool(params, 2); v != false {
		t.Error("expected false, got true")
	}
	if v := getBool(params, 3); v != false {
		t.Error("expected false for string 'true', got true")
	}
	if v := getBool(params, 99); v != false {
		t.Error("expected false for missing key, got true")
	}
}

// TestHelperToInt64 tests the toInt64 helper function
func TestHelperToInt64(t *testing.T) {
	testCases := []struct {
		input    interface{}
		expected int64
	}{
		{int64(100), 100},
		{int32(200), 200},
		{int16(300), 300},
		{int(400), 400},
		{uint8(50), 50},
		{float32(1.5), 1},
		{float64(2.9), 2},
		{"not a number", 0},
		{nil, 0},
	}

	for _, tc := range testCases {
		result := toInt64(tc.input)
		if result != tc.expected {
			t.Errorf("toInt64(%v) = %d, expected %d", tc.input, result, tc.expected)
		}
	}
}

// TestOnRequestDebugMode tests request handling in debug mode
func TestOnRequestDebugMode(t *testing.T) {
	handler := NewAlbionHandler()
	handler.SetDebug(true)

	// Should not panic
	handler.OnRequest(1, map[byte]interface{}{1: "test"})
}

// TestOnResponseDebugMode tests response handling in debug mode
func TestOnResponseDebugMode(t *testing.T) {
	handler := NewAlbionHandler()
	handler.SetDebug(true)

	// Should not panic
	handler.OnResponse(1, 0, "debug message", map[byte]interface{}{1: "test"})
}

// TestFameEventDataStructure tests the FameEventData struct fields
func TestFameEventDataStructure(t *testing.T) {
	data := &FameEventData{
		Gained:  100,
		Total:   5000,
		Session: 500,
	}

	if data.Gained != 100 {
		t.Errorf("Gained field incorrect")
	}
	if data.Total != 5000 {
		t.Errorf("Total field incorrect")
	}
	if data.Session != 500 {
		t.Errorf("Session field incorrect")
	}
}

// TestSilverEventDataStructure tests the SilverEventData struct fields
func TestSilverEventDataStructure(t *testing.T) {
	data := &SilverEventData{
		Amount:     1000,
		Session:    5000,
		LootedBy:   "Player1",
		LootedFrom: "Monster",
	}

	if data.Amount != 1000 {
		t.Errorf("Amount field incorrect")
	}
	if data.Session != 5000 {
		t.Errorf("Session field incorrect")
	}
	if data.LootedBy != "Player1" {
		t.Errorf("LootedBy field incorrect")
	}
	if data.LootedFrom != "Monster" {
		t.Errorf("LootedFrom field incorrect")
	}
}

// TestHandleUpdateReSpecPoints tests respec credit handling (Event #84)
func TestHandleUpdateReSpecPoints(t *testing.T) {
	handler := NewAlbionHandler()

	var receivedData *RespecEventData
	handler.SetEventCallback(func(eventType, message string, data interface{}) {
		if eventType == "respec" {
			receivedData = data.(*RespecEventData)
		}
	})

	// Values are in FixPoint format (multiply by 10000)
	params := map[byte]interface{}{
		2:                     int64(10000000), // Gained respec (1000 in FixPoint)
		3:                     int64(5000000),  // Paid silver (500 in FixPoint)
		events.ParamEventCode: int16(events.EventUpdateReSpecPoints),
	}

	handler.OnEvent(byte(events.EventUpdateReSpecPoints), params)

	if receivedData == nil {
		t.Fatal("respec callback was not called")
	}

	if receivedData.Gained != 1000 {
		t.Errorf("expected gained 1000, got %d", receivedData.Gained)
	}

	if receivedData.PaidSilver != 500 {
		t.Errorf("expected paid silver 500, got %d", receivedData.PaidSilver)
	}

	if receivedData.SessionTotal != 1000 {
		t.Errorf("expected session total 1000, got %d", receivedData.SessionTotal)
	}

	if receivedData.SessionSilverTotal != 500 {
		t.Errorf("expected session silver total 500, got %d", receivedData.SessionSilverTotal)
	}

	if handler.GetSessionRespec() != 1000 {
		t.Errorf("expected session respec 1000, got %d", handler.GetSessionRespec())
	}

	if handler.GetSessionRespecSilver() != 500 {
		t.Errorf("expected session respec silver 500, got %d", handler.GetSessionRespecSilver())
	}
}

// TestHandleUpdateReSpecPointsZeroGained tests that events with zero gained are ignored
func TestHandleUpdateReSpecPointsZeroGained(t *testing.T) {
	handler := NewAlbionHandler()

	callCount := 0
	handler.SetEventCallback(func(eventType, message string, data interface{}) {
		if eventType == "respec" {
			callCount++
		}
	})

	params := map[byte]interface{}{
		2:                     int64(0), // Zero gained
		3:                     int64(0),
		events.ParamEventCode: int16(events.EventUpdateReSpecPoints),
	}

	handler.OnEvent(byte(events.EventUpdateReSpecPoints), params)

	if callCount != 0 {
		t.Errorf("expected 0 callbacks for zero gained, got %d", callCount)
	}

	if handler.GetSessionRespec() != 0 {
		t.Errorf("expected session respec 0, got %d", handler.GetSessionRespec())
	}
}

// TestHandleUpdateReSpecPointsAccumulation tests multiple events accumulate correctly
func TestHandleUpdateReSpecPointsAccumulation(t *testing.T) {
	handler := NewAlbionHandler()

	var lastSessionTotal int64
	var lastSessionSilverTotal int64
	handler.SetEventCallback(func(eventType, message string, data interface{}) {
		if eventType == "respec" {
			if respecData, ok := data.(*RespecEventData); ok {
				lastSessionTotal = respecData.SessionTotal
				lastSessionSilverTotal = respecData.SessionSilverTotal
			}
		}
	})

	// First event: 500 credits, 100 silver
	handler.OnEvent(byte(events.EventUpdateReSpecPoints), map[byte]interface{}{
		2:                     int64(5000000), // 500 in FixPoint
		3:                     int64(1000000), // 100 in FixPoint
		events.ParamEventCode: int16(events.EventUpdateReSpecPoints),
	})

	// Second event: 300 credits, 50 silver
	handler.OnEvent(byte(events.EventUpdateReSpecPoints), map[byte]interface{}{
		2:                     int64(3000000), // 300 in FixPoint
		3:                     int64(500000),  // 50 in FixPoint
		events.ParamEventCode: int16(events.EventUpdateReSpecPoints),
	})

	if lastSessionTotal != 800 {
		t.Errorf("expected session total 800, got %d", lastSessionTotal)
	}

	if lastSessionSilverTotal != 150 {
		t.Errorf("expected session silver total 150, got %d", lastSessionSilverTotal)
	}

	if handler.GetSessionRespec() != 800 {
		t.Errorf("expected session respec 800, got %d", handler.GetSessionRespec())
	}

	if handler.GetSessionRespecSilver() != 150 {
		t.Errorf("expected session respec silver 150, got %d", handler.GetSessionRespecSilver())
	}
}

// TestDiscoveredEventStructure tests the DiscoveredEvent struct
func TestDiscoveredEventStructure(t *testing.T) {
	now := time.Now()
	event := &DiscoveredEvent{
		Code:       100,
		Count:      5,
		FirstSeen:  now,
		LastSeen:   now,
		SampleData: map[byte]interface{}{1: "test"},
		ParamTypes: map[byte]string{1: "string"},
	}

	if event.Code != 100 {
		t.Errorf("Code field incorrect")
	}
	if event.Count != 5 {
		t.Errorf("Count field incorrect")
	}
	if len(event.SampleData) != 1 {
		t.Errorf("SampleData field incorrect")
	}
	if len(event.ParamTypes) != 1 {
		t.Errorf("ParamTypes field incorrect")
	}
}

// TestSetLocalPlayer tests that SetLocalPlayer ignores empty values and
// overwrites the previous name (used internally by OpJoin auto-detection).
func TestSetLocalPlayer(t *testing.T) {
	handler := NewAlbionHandler()

	if got := handler.GetLocalPlayer(); got != "" {
		t.Errorf("expected empty local player initially, got %q", got)
	}

	// Empty value is ignored.
	handler.SetLocalPlayer("")
	if got := handler.GetLocalPlayer(); got != "" {
		t.Errorf("empty SetLocalPlayer should be ignored, got %q", got)
	}

	handler.SetLocalPlayer("Hero")

	if got := handler.GetLocalPlayer(); got != "Hero" {
		t.Errorf("expected 'Hero', got %q", got)
	}

	// A subsequent call overwrites (e.g. relog with a different character).
	handler.SetLocalPlayer("Hero2")
	if got := handler.GetLocalPlayer(); got != "Hero2" {
		t.Errorf("expected 'Hero2' after overwrite, got %q", got)
	}
}

// TestOpJoinSetsLocalPlayer tests that the OpJoin response captures the local
// player name from parameters[2].
func TestOpJoinSetsLocalPlayer(t *testing.T) {
	handler := NewAlbionHandler()

	handler.OnResponse(events.OperationJoin, 0, "", map[byte]interface{}{
		2: "cantalupo",
	})

	if got := handler.GetLocalPlayer(); got != "cantalupo" {
		t.Errorf("expected local player 'cantalupo', got %q", got)
	}
}

// TestOpJoinIgnoredForUnknownOpCode tests that other operation codes do not set
// the local player.
func TestOpJoinIgnoredForUnknownOpCode(t *testing.T) {
	handler := NewAlbionHandler()

	handler.OnResponse(99, 0, "", map[byte]interface{}{
		2: "Someone",
	})

	if got := handler.GetLocalPlayer(); got != "" {
		t.Errorf("non-Join response must not set local player, got %q", got)
	}
}

// TestOpJoinIgnoresEmptyName tests that an OpJoin response with an empty or
// missing player name does not set the local player.
func TestOpJoinIgnoresEmptyName(t *testing.T) {
	handler := NewAlbionHandler()

	// Missing parameter 2 entirely.
	handler.OnResponse(events.OperationJoin, 0, "", map[byte]interface{}{})
	if got := handler.GetLocalPlayer(); got != "" {
		t.Errorf("missing OpJoin name must not set local player, got %q", got)
	}

	// Empty string parameter.
	handler.OnResponse(events.OperationJoin, 0, "", map[byte]interface{}{
		2: "",
	})
	if got := handler.GetLocalPlayer(); got != "" {
		t.Errorf("empty OpJoin name must not set local player, got %q", got)
	}
}

// TestDiedNoKillCountedWhenLocalUnknown tests that, before the local player is
// known, kills are not counted even when EventDied arrives. This guards against
// reintroducing unconditional kill counting.
func TestDiedNoKillCountedWhenLocalUnknown(t *testing.T) {
	handler := NewAlbionHandler()

	killCount := 0
	handler.SetEventCallback(func(eventType, message string, data interface{}) {
		if eventType == "kill" {
			killCount++
		}
	})

	// No local player set: a death where someone happens to be the killer must
	// not be attributed as the local player's kill.
	handler.OnEvent(byte(events.EventDied), map[byte]interface{}{
		2:  "Victim",
		10: "Someone",
	})

	if handler.GetSessionKills() != 0 {
		t.Errorf("expected 0 kills when local player unknown, got %d", handler.GetSessionKills())
	}
	if killCount != 0 {
		t.Errorf("expected 0 kill callbacks when local player unknown, got %d", killCount)
	}
}

// TestDiedCountsKillWhenLocalPlayerIsKiller tests that a kill is counted once when
// the local player is the killer, and that redundant copies are collapsed.
func TestDiedCountsKillWhenLocalPlayerIsKiller(t *testing.T) {
	handler := NewAlbionHandler()
	handler.SetLocalPlayer("cantalupo")

	killCount := 0
	deathCount := 0
	handler.SetEventCallback(func(eventType, message string, data interface{}) {
		switch eventType {
		case "kill":
			killCount++
		case "death":
			deathCount++
		}
	})

	// The server delivers EventDied redundantly when the local player is the killer.
	deathParams := map[byte]interface{}{
		2:  "TNTAbyss",
		10: "cantalupo",
	}
	for i := 0; i < 3; i++ {
		handler.OnEvent(byte(events.EventDied), deathParams)
	}

	if handler.GetSessionKills() != 1 {
		t.Errorf("expected 1 kill (deduped), got %d", handler.GetSessionKills())
	}
	if killCount != 1 {
		t.Errorf("expected 1 kill callback, got %d", killCount)
	}
	if deathCount != 0 {
		t.Errorf("expected 0 death callbacks for a local kill, got %d", deathCount)
	}

	// Local player was the killer, not the victim: deaths must not be counted.
	if handler.GetSessionDeaths() != 0 {
		t.Errorf("expected 0 deaths, got %d", handler.GetSessionDeaths())
	}
}

// TestDiedDedupByVictimKiller tests that distinct victims within the dedup window
// are counted separately (rapid multi-kills are preserved), while the same
// victim+killer is collapsed.
func TestDiedDedupByVictimKiller(t *testing.T) {
	handler := NewAlbionHandler()
	handler.SetLocalPlayer("Hero")

	// Two different victims back-to-back: both counted.
	handler.OnEvent(byte(events.EventDied), map[byte]interface{}{
		2:  "VictimA",
		10: "Hero",
	})
	handler.OnEvent(byte(events.EventDied), map[byte]interface{}{
		2:  "VictimB",
		10: "Hero",
	})

	if handler.GetSessionKills() != 2 {
		t.Errorf("expected 2 kills for distinct victims, got %d", handler.GetSessionKills())
	}

	// A duplicate of the first kill is collapsed.
	handler.OnEvent(byte(events.EventDied), map[byte]interface{}{
		2:  "VictimA",
		10: "Hero",
	})

	if handler.GetSessionKills() != 2 {
		t.Errorf("expected deduped 2 kills, got %d", handler.GetSessionKills())
	}
}

// TestDiedOnlyCountsLocalDeaths tests that a death is counted only when the local
// player is the victim; deaths of other players do not increment sessionDeaths.
func TestDiedOnlyCountsLocalDeaths(t *testing.T) {
	handler := NewAlbionHandler()
	handler.SetLocalPlayer("Hero")

	// Another player dies: not counted.
	handler.OnEvent(byte(events.EventDied), map[byte]interface{}{
		2:  "Stranger",
		10: "SomeoneElse",
	})
	if handler.GetSessionDeaths() != 0 {
		t.Errorf("foreign death must not count, got %d", handler.GetSessionDeaths())
	}

	// Local player dies: counted.
	handler.OnEvent(byte(events.EventDied), map[byte]interface{}{
		2:  "Hero",
		10: "Killer",
	})
	if handler.GetSessionDeaths() != 1 {
		t.Errorf("expected 1 local death, got %d", handler.GetSessionDeaths())
	}
}

// silverEventParams builds a silver loot event parameter map for tests.
// silverAmountRaw is in FixPoint format (value * 10000).
func silverEventParams(lootedBy, lootedFrom string, silverAmountRaw int64) map[byte]interface{} {
	return map[byte]interface{}{
		1:                     lootedFrom,
		2:                     lootedBy,
		3:                     true, // is silver
		4:                     int32(0),
		5:                     silverAmountRaw,
		events.ParamEventCode: int16(events.EventOtherGrabbedLoot),
	}
}

// TestSilverExcludesForeignLoot tests that silver looted by another player is
// notified (still visible) but not added to the session total, while silver looted
// by the local player is both notified and counted.
func TestSilverExcludesForeignLoot(t *testing.T) {
	handler := NewAlbionHandler()
	handler.SetLocalPlayer("Hero")

	var events []SilverEventData
	handler.SetEventCallback(func(eventType, message string, data interface{}) {
		if eventType == "silver" {
			if sd, ok := data.(*SilverEventData); ok {
				events = append(events, *sd)
			}
		}
	})

	// 5000 silver looted by the local player.
	handler.OnEvent(0, silverEventParams("Hero", "Monster", 5000*10000))
	// 420000 silver looted by someone else (foreign).
	handler.OnEvent(0, silverEventParams("Stranger", "Monster", 420000*10000))

	if len(events) != 2 {
		t.Fatalf("expected 2 silver callbacks, got %d", len(events))
	}

	if events[0].Amount != 5000 || events[0].LootedBy != "Hero" || !events[0].Counted {
		t.Errorf("local silver event mismatch: %+v", events[0])
	}
	if events[1].Amount != 420000 || events[1].LootedBy != "Stranger" || events[1].Counted {
		t.Errorf("foreign silver should not be counted: %+v", events[1])
	}

	// Only the local player's 5000 silver counts toward the session total.
	if handler.GetSessionSilver() != 5000 {
		t.Errorf("expected session silver 5000, got %d", handler.GetSessionSilver())
	}
}

// TestSilverCountsAllWhenLocalUnknown tests that, before the local player is known,
// all silver is counted (backward compatible).
func TestSilverNotCountedWhenLocalUnknown(t *testing.T) {
	handler := NewAlbionHandler()

	countedCount := 0
	handler.SetEventCallback(func(eventType, message string, data interface{}) {
		if eventType == "silver" {
			if sd, ok := data.(*SilverEventData); ok && sd.Counted {
				countedCount++
			}
		}
	})

	handler.OnEvent(0, silverEventParams("Stranger", "Monster", 5000*10000))
	handler.OnEvent(0, silverEventParams("OtherGuy", "Monster", 3000*10000))

	if countedCount != 0 {
		t.Errorf("expected 0 counted events when local unknown, got %d", countedCount)
	}
	if handler.GetSessionSilver() != 0 {
		t.Errorf("expected session silver 0 (nothing counted), got %d", handler.GetSessionSilver())
	}
}

// TestSilverEmptyLootedWithKnownLocal tests the malformed-packet edge case where
// lootedBy is empty while a local player is known. Such silver must NOT be counted
// toward the session total (it does not match the local player).
func TestSilverEmptyLootedWithKnownLocal(t *testing.T) {
	handler := NewAlbionHandler()
	handler.SetLocalPlayer("Hero")

	var received *SilverEventData
	handler.SetEventCallback(func(eventType, message string, data interface{}) {
		if eventType == "silver" {
			if sd, ok := data.(*SilverEventData); ok {
				received = sd
			}
		}
	})

	handler.OnEvent(0, silverEventParams("", "Monster", 5000*10000))

	if received == nil {
		t.Fatal("silver callback was not called")
	}
	if received.Counted {
		t.Error("empty LootedBy must not be counted toward the session total")
	}
	if handler.GetSessionSilver() != 0 {
		t.Errorf("expected session silver 0, got %d", handler.GetSessionSilver())
	}
}

// TestDiedDedupWindowExpiry tests that a death signature stops being considered a
// duplicate once deathDedupWindow has elapsed, so two legitimately separate kills of
// the same victim far enough apart are both counted.
func TestDiedDedupWindowExpiry(t *testing.T) {
	handler := NewAlbionHandler()
	handler.SetLocalPlayer("Hero")

	// Use a virtual clock so the test does not have to sleep.
	base := time.Now()
	handler.nowFunc = func() time.Time { return base }
	handler.deathDedupWindow = 100 * time.Millisecond

	deathParams := map[byte]interface{}{
		2:  "Victim",
		10: "Hero",
	}

	// First kill: counted.
	handler.OnEvent(byte(events.EventDied), deathParams)
	if handler.GetSessionKills() != 1 {
		t.Fatalf("expected 1 kill after first death, got %d", handler.GetSessionKills())
	}

	// Immediate duplicate: suppressed.
	handler.OnEvent(byte(events.EventDied), deathParams)
	if handler.GetSessionKills() != 1 {
		t.Fatalf("expected duplicate to be suppressed, got %d", handler.GetSessionKills())
	}

	// Advance time past the dedup window: the same victim+killer is counted again.
	base = base.Add(handler.deathDedupWindow + time.Second)
	handler.OnEvent(byte(events.EventDied), deathParams)
	if handler.GetSessionKills() != 2 {
		t.Errorf("expected 2 kills after window expiry, got %d", handler.GetSessionKills())
	}
}

// TestDiedDedupNotConsumedBeforeIdentity tests that a death arriving before the
// local player is identified does NOT consume the dedup window. Without this
// guard, the first real kill after identity capture would be permanently lost
// because the dedup key was already recorded during the unknown-player phase.
func TestDiedDedupNotConsumedBeforeIdentity(t *testing.T) {
	handler := NewAlbionHandler()
	// local player NOT set yet (simulating tool started mid-session)

	deathParams := map[byte]interface{}{
		2:  "Victim",
		10: "Hero",
	}

	// Death arrives while identity is unknown: not counted, not deduped.
	handler.OnEvent(byte(events.EventDied), deathParams)
	if handler.GetSessionKills() != 0 {
		t.Fatalf("expected 0 kills before identity, got %d", handler.GetSessionKills())
	}

	// Identity captured (e.g. from OpJoin). The same victim+killer death arrives
	// again — it must NOT be treated as a duplicate because the dedup key was
	// never recorded during the unknown phase.
	handler.SetLocalPlayer("Hero")
	handler.OnEvent(byte(events.EventDied), deathParams)
	if handler.GetSessionKills() != 1 {
		t.Errorf("expected 1 kill after identity (dedup not pre-consumed), got %d", handler.GetSessionKills())
	}
}

// TestDebugNotifyOnLocalPlayerCapture tests that debug traces are emitted for local
// player identity capture (only in debug mode).
func TestDebugNotifyOnLocalPlayerCapture(t *testing.T) {
	handler := NewAlbionHandler()
	handler.SetDebug(true)

	debugMsgs := []string{}
	handler.SetEventCallback(func(eventType, message string, data interface{}) {
		if eventType == "debug" {
			debugMsgs = append(debugMsgs, message)
		}
	})

	// OpJoin auto-detection emits a debug trace.
	handler.OnResponse(events.OperationJoin, 0, "", map[byte]interface{}{
		2: "Hero",
	})

	found := false
	for _, m := range debugMsgs {
		if m == "local player detected from OpJoin" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected OpJoin debug trace, got %v", debugMsgs)
	}
}

// TestDebugNotifySuppressedWhenDebugOff tests that debug traces are NOT emitted when
// debug mode is disabled.
func TestDebugNotifySuppressedWhenDebugOff(t *testing.T) {
	handler := NewAlbionHandler()
	// debug is off by default

	debugCount := 0
	handler.SetEventCallback(func(eventType, message string, data interface{}) {
		if eventType == "debug" {
			debugCount++
		}
	})

	handler.OnResponse(events.OperationJoin, 0, "", map[byte]interface{}{
		2: "Hero",
	})
	handler.OnEvent(byte(events.EventKilledPlayer), map[byte]interface{}{})

	if debugCount != 0 {
		t.Errorf("expected no debug events when debug off, got %d", debugCount)
	}
}

// TestSessionCountersConcurrent verifies that session counters are safe for
// concurrent access: one goroutine writes via OnEvent while another reads via the
// GetSession* getters. Under `go test -race` this would flag a data race if the
// counters were still plain int/int64 fields.
func TestSessionCountersConcurrent(t *testing.T) {
	handler := NewAlbionHandler()
	handler.SetLocalPlayer("Hero")
	handler.deathDedupWindow = 0 // collapse nothing — each victim is unique anyway

	var wg sync.WaitGroup
	stop := make(chan struct{})

	// Reader goroutine: polls all getters continuously until the writer is done.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				_ = handler.GetSessionKills()
				_ = handler.GetSessionDeaths()
				_ = handler.GetSessionLoot()
				_ = handler.GetSessionFame()
				_ = handler.GetSessionSilver()
				_ = handler.GetSessionRespec()
				_ = handler.GetSessionRespecSilver()
			}
		}
	}()

	// Writer goroutine: fires events that mutate every counter, then signals stop.
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(stop)
		for i := 0; i < 200; i++ {
			handler.OnEvent(0, map[byte]interface{}{
				0: int32(1),
				1: int64(40_000_010_000 + i), // increasing total fame
			})
			handler.OnEvent(0, map[byte]interface{}{
				1:                     "Hero",      // looted by
				2:                     "Monster",   // looted from
				3:                     true,        // is silver
				4:                     int32(9999), // item id (silver)
				5:                     int32(1000), // quantity
				events.ParamEventCode: int16(events.EventOtherGrabbedLoot),
			})
			handler.OnEvent(byte(events.EventDied), map[byte]interface{}{
				2:  fmt.Sprintf("Victim%d", i),
				10: "Hero",
			})
		}
	}()

	wg.Wait()

	// Final sanity: getters return sensible non-negative values.
	if k := handler.GetSessionKills(); k < 0 {
		t.Errorf("sessionKills should be >= 0, got %d", k)
	}
}
