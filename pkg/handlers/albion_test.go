package handlers

import (
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

	// Simulate kill event
	handler.OnEvent(byte(events.EventKilledPlayer), map[byte]interface{}{})
	if handler.GetSessionKills() != 1 {
		t.Errorf("expected 1 kill, got %d", handler.GetSessionKills())
	}

	// Simulate death event
	handler.OnEvent(byte(events.EventDied), map[byte]interface{}{})
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
		events.ParamEventCode: int16(events.EventUpdateFameDetails),
	}

	handler.OnEvent(byte(events.EventUpdateFameDetails), params)

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
	handler.totalFame = int64(40000000000) // 4M in FixPoint

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
	handler.OnEvent(byte(events.EventUpdateFameDetails), params)

	// Duplicate event with same total fame
	handler.OnEvent(byte(events.EventUpdateFameDetails), params)

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
	handler.OnEvent(byte(events.EventUpdateFameDetails), params)

	if callCount != 0 {
		t.Errorf("expected 0 callbacks (low fame should be ignored), got %d", callCount)
	}
}

// TestHandleOtherGrabbedLootSilver tests silver loot handling
func TestHandleOtherGrabbedLootSilver(t *testing.T) {
	handler := NewAlbionHandler()

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

	var receivedMessage string
	handler.SetEventCallback(func(eventType, message string, data interface{}) {
		if eventType == "loot" {
			receivedMessage = message
		}
	})

	// Simulate item loot event
	// Note: EventOtherGrabbedLoot (275) > 255, so we pass it via ParamEventCode
	params := map[byte]interface{}{
		1:                     "Chest",       // Looted from
		2:                     "Player1",     // Looted by
		3:                     false,         // Is silver (false = item)
		4:                     int32(12345),  // Item ID
		5:                     int32(3),      // Quantity
		events.ParamEventCode: int16(events.EventOtherGrabbedLoot),
	}

	handler.OnEvent(0, params) // Event code comes from param 252

	if receivedMessage == "" {
		t.Fatal("loot callback was not called")
	}

	if handler.GetSessionLoot() != 1 {
		t.Errorf("expected session loot 1, got %d", handler.GetSessionLoot())
	}
}

// TestHandleKilledPlayer tests kill event handling
func TestHandleKilledPlayer(t *testing.T) {
	handler := NewAlbionHandler()

	var receivedMessage string
	handler.SetEventCallback(func(eventType, message string, data interface{}) {
		if eventType == "kill" {
			receivedMessage = message
		}
	})

	handler.OnEvent(byte(events.EventKilledPlayer), map[byte]interface{}{})

	if receivedMessage == "" {
		t.Fatal("kill callback was not called")
	}

	if handler.GetSessionKills() != 1 {
		t.Errorf("expected 1 kill, got %d", handler.GetSessionKills())
	}
}

// TestHandleDied tests death event handling
func TestHandleDied(t *testing.T) {
	handler := NewAlbionHandler()

	var receivedMessage string
	handler.SetEventCallback(func(eventType, message string, data interface{}) {
		if eventType == "death" {
			receivedMessage = message
		}
	})

	handler.OnEvent(byte(events.EventDied), map[byte]interface{}{})

	if receivedMessage == "" {
		t.Fatal("death callback was not called")
	}

	if handler.GetSessionDeaths() != 1 {
		t.Errorf("expected 1 death, got %d", handler.GetSessionDeaths())
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

// TestIsKnownEventCode tests known event code detection
func TestIsKnownEventCode(t *testing.T) {
	handler := NewAlbionHandler()

	knownCodes := []int16{
		int16(events.EventUpdateFame),
		int16(events.EventKilledPlayer),
		int16(events.EventDied),
		int16(events.EventOtherGrabbedLoot),
	}

	for _, code := range knownCodes {
		if !handler.isKnownEventCode(code) {
			t.Errorf("code %d should be known", code)
		}
	}

	// Test unknown code
	if handler.isKnownEventCode(9999) {
		t.Error("code 9999 should be unknown")
	}
}

// TestOnEventWithParamEventCode tests that event code is read from param 252
func TestOnEventWithParamEventCode(t *testing.T) {
	handler := NewAlbionHandler()
	handler.SetDiscoveryMode(true)

	// Send event with event code in param 252
	params := map[byte]interface{}{
		events.ParamEventCode: int16(events.EventKilledPlayer),
	}
	handler.OnEvent(0, params) // byte code is 0, but actual code is in param 252

	if handler.GetSessionKills() != 1 {
		t.Error("event code from param 252 was not used")
	}
}

// TestOnEventParamEventCodeConversion tests different types for param event code
func TestOnEventParamEventCodeConversion(t *testing.T) {
	testCases := []struct {
		name     string
		codeVal  interface{}
		expected int
	}{
		{"int16", int16(events.EventKilledPlayer), 1},
		{"int32", int32(events.EventKilledPlayer), 1},
		{"int64", int64(events.EventKilledPlayer), 1},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			h := NewAlbionHandler()
			params := map[byte]interface{}{
				events.ParamEventCode: tc.codeVal,
			}
			h.OnEvent(0, params)
			if h.GetSessionKills() != tc.expected {
				t.Errorf("expected %d kills with %s, got %d", tc.expected, tc.name, h.GetSessionKills())
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

// TestFormatSilver tests the formatSilver helper function
func TestFormatSilver(t *testing.T) {
	testCases := []struct {
		input    int64
		expected string
	}{
		{500, "500"},
		{1000, "1.0k"},
		{1500, "1.5k"},
		{10000, "10.0k"},
		{1000000, "1.0M"},
		{2500000, "2.5M"},
	}

	for _, tc := range testCases {
		result := formatSilver(tc.input)
		if result != tc.expected {
			t.Errorf("formatSilver(%d) = '%s', expected '%s'", tc.input, result, tc.expected)
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
