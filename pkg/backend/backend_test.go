package backend

import (
	"testing"
	"time"

	"github.com/cantalupo555/albion-lens/pkg/photon"
)

// ============================================
// Tests for options.go
// ============================================

// TestNewServiceDefaults tests default service creation
func TestNewServiceDefaults(t *testing.T) {
	s := New()

	if s == nil {
		t.Fatal("New() returned nil")
	}

	// Check default values
	if s.eventBufferSize != defaultEventBufferSize {
		t.Errorf("eventBufferSize: expected %d, got %d", defaultEventBufferSize, s.eventBufferSize)
	}

	if s.statsBufferSize != defaultStatsBufferSize {
		t.Errorf("statsBufferSize: expected %d, got %d", defaultStatsBufferSize, s.statsBufferSize)
	}

	if s.device != "" {
		t.Errorf("device: expected empty, got '%s'", s.device)
	}

	if s.debug != false {
		t.Error("debug: expected false")
	}

	if s.discovery != false {
		t.Error("discovery: expected false")
	}

	// Check channels are created
	if s.Events == nil {
		t.Error("Events channel not created")
	}

	if s.Stats == nil {
		t.Error("Stats channel not created")
	}

	if s.OnlineStatus == nil {
		t.Error("OnlineStatus channel not created")
	}

	if s.stopChan == nil {
		t.Error("stopChan not created")
	}
}

// TestWithDevice tests device option
func TestWithDevice(t *testing.T) {
	s := New(WithDevice("eth0"))

	if s.device != "eth0" {
		t.Errorf("expected 'eth0', got '%s'", s.device)
	}
}

// TestWithDebug tests debug option
func TestWithDebug(t *testing.T) {
	s := New(WithDebug(true))

	if s.debug != true {
		t.Error("expected debug to be true")
	}

	s = New(WithDebug(false))
	if s.debug != false {
		t.Error("expected debug to be false")
	}
}

// TestWithDiscovery tests discovery option
func TestWithDiscovery(t *testing.T) {
	s := New(WithDiscovery(true))

	if s.discovery != true {
		t.Error("expected discovery to be true")
	}

	s = New(WithDiscovery(false))
	if s.discovery != false {
		t.Error("expected discovery to be false")
	}
}

// TestWithItemDatabasePath tests item database path option
func TestWithItemDatabasePath(t *testing.T) {
	s := New(WithItemDatabasePath("/path/to/items"))

	if s.itemDBPath != "/path/to/items" {
		t.Errorf("expected '/path/to/items', got '%s'", s.itemDBPath)
	}
}

// TestWithBPFFilter tests BPF filter option
func TestWithBPFFilter(t *testing.T) {
	s := New(WithBPFFilter("udp port 5056"))

	if s.bpfFilter != "udp port 5056" {
		t.Errorf("expected 'udp port 5056', got '%s'", s.bpfFilter)
	}
}

// TestWithBPFFilter_EmptyString documents the backward-compatibility
// contract: passing an empty filter is equivalent to omitting the option,
// because capture.NewCaptureWithFilter falls back to the default BPFFilter
// when handed an empty string. The backend must store exactly what it
// received so the fallback can happen at the capture layer.
func TestWithBPFFilter_EmptyString(t *testing.T) {
	s := New(WithBPFFilter(""))

	if s.bpfFilter != "" {
		t.Errorf("empty filter should be stored as empty string, got '%s'", s.bpfFilter)
	}

	// Omitting the option must produce the same stored state so that
	// capture.NewCaptureWithFilter behaves identically in both cases.
	without := New()
	if without.bpfFilter != s.bpfFilter {
		t.Errorf("WithBPFFilter(\"\") = %q, omitting = %q; must match", s.bpfFilter, without.bpfFilter)
	}
}

// TestWithEventBufferSize tests event buffer size option
func TestWithEventBufferSize(t *testing.T) {
	s := New(WithEventBufferSize(500))

	if s.eventBufferSize != 500 {
		t.Errorf("expected 500, got %d", s.eventBufferSize)
	}
}

// TestWithStatsBufferSize tests stats buffer size option
func TestWithStatsBufferSize(t *testing.T) {
	s := New(WithStatsBufferSize(50))

	if s.statsBufferSize != 50 {
		t.Errorf("expected 50, got %d", s.statsBufferSize)
	}
}

// TestMultipleOptions tests applying multiple options
func TestMultipleOptions(t *testing.T) {
	s := New(
		WithDevice("wlan0"),
		WithDebug(true),
		WithDiscovery(true),
		WithItemDatabasePath("/db/path"),
		WithBPFFilter("udp"),
		WithEventBufferSize(200),
		WithStatsBufferSize(20),
	)

	if s.device != "wlan0" {
		t.Errorf("device: expected 'wlan0', got '%s'", s.device)
	}
	if s.debug != true {
		t.Error("debug: expected true")
	}
	if s.discovery != true {
		t.Error("discovery: expected true")
	}
	if s.itemDBPath != "/db/path" {
		t.Errorf("itemDBPath: expected '/db/path', got '%s'", s.itemDBPath)
	}
	if s.bpfFilter != "udp" {
		t.Errorf("bpfFilter: expected 'udp', got '%s'", s.bpfFilter)
	}
	if s.eventBufferSize != 200 {
		t.Errorf("eventBufferSize: expected 200, got %d", s.eventBufferSize)
	}
	if s.statsBufferSize != 20 {
		t.Errorf("statsBufferSize: expected 20, got %d", s.statsBufferSize)
	}
}

// TestOptionOrder tests that later options override earlier ones
func TestOptionOrder(t *testing.T) {
	s := New(
		WithDevice("eth0"),
		WithDevice("eth1"),
		WithDevice("eth2"),
	)

	if s.device != "eth2" {
		t.Errorf("expected last option to win, got '%s'", s.device)
	}
}

// ============================================
// Tests for events.go
// ============================================

// TestEventTypeConstants tests event type constants
func TestEventTypeConstants(t *testing.T) {
	testCases := []struct {
		eventType EventType
		expected  string
	}{
		{EventTypeFame, "fame"},
		{EventTypeSilver, "silver"},
		{EventTypeLoot, "loot"},
		{EventTypeKill, "kill"},
		{EventTypeDeath, "death"},
		{EventTypeRespec, "respec"},
		{EventTypeInfo, "info"},
		{EventTypeWarning, "warning"},
		{EventTypeZone, "zone"},
	}

	for _, tc := range testCases {
		if string(tc.eventType) != tc.expected {
			t.Errorf("expected '%s', got '%s'", tc.expected, string(tc.eventType))
		}
	}
}

// TestGameEventStructure tests GameEvent struct
func TestGameEventStructure(t *testing.T) {
	now := time.Now()
	data := &FameData{Gained: 100}

	event := GameEvent{
		Type:      EventTypeFame,
		Message:   "Test message",
		Timestamp: now,
		Data:      data,
	}

	if event.Type != EventTypeFame {
		t.Errorf("Type: expected %s, got %s", EventTypeFame, event.Type)
	}

	if event.Message != "Test message" {
		t.Errorf("Message: expected 'Test message', got '%s'", event.Message)
	}

	if event.Timestamp != now {
		t.Error("Timestamp mismatch")
	}

	if fameData, ok := event.Data.(*FameData); !ok {
		t.Error("Data should be *FameData")
	} else if fameData.Gained != 100 {
		t.Errorf("Data.Gained: expected 100, got %d", fameData.Gained)
	}
}

// TestFameDataStructure tests FameData struct
func TestFameDataStructure(t *testing.T) {
	data := FameData{
		Gained:  1000,
		Total:   50000,
		Session: 5000,
	}

	if data.Gained != 1000 {
		t.Errorf("Gained: expected 1000, got %d", data.Gained)
	}

	if data.Total != 50000 {
		t.Errorf("Total: expected 50000, got %d", data.Total)
	}

	if data.Session != 5000 {
		t.Errorf("Session: expected 5000, got %d", data.Session)
	}
}

// TestSilverDataStructure tests SilverData struct
func TestSilverDataStructure(t *testing.T) {
	data := SilverData{
		Amount:  5000,
		Session: 25000,
	}

	if data.Amount != 5000 {
		t.Errorf("Amount: expected 5000, got %d", data.Amount)
	}

	if data.Session != 25000 {
		t.Errorf("Session: expected 25000, got %d", data.Session)
	}
}

// TestLootDataStructure tests LootData struct
func TestLootDataStructure(t *testing.T) {
	data := LootData{
		ItemName: "T4 Bag",
		ItemID:   12345,
		Quantity: 3,
		LootedBy: "Player1",
		From:     "Chest",
	}

	if data.ItemName != "T4 Bag" {
		t.Errorf("ItemName: expected 'T4 Bag', got '%s'", data.ItemName)
	}

	if data.ItemID != 12345 {
		t.Errorf("ItemID: expected 12345, got %d", data.ItemID)
	}

	if data.Quantity != 3 {
		t.Errorf("Quantity: expected 3, got %d", data.Quantity)
	}

	if data.LootedBy != "Player1" {
		t.Errorf("LootedBy: expected 'Player1', got '%s'", data.LootedBy)
	}

	if data.From != "Chest" {
		t.Errorf("From: expected 'Chest', got '%s'", data.From)
	}
}

// TestCombatDataStructure tests CombatData struct
func TestCombatDataStructure(t *testing.T) {
	data := CombatData{
		KillerName: "Attacker",
		VictimName: "Victim",
		Session:    5,
	}

	if data.KillerName != "Attacker" {
		t.Errorf("KillerName: expected 'Attacker', got '%s'", data.KillerName)
	}

	if data.VictimName != "Victim" {
		t.Errorf("VictimName: expected 'Victim', got '%s'", data.VictimName)
	}

	if data.Session != 5 {
		t.Errorf("Session: expected 5, got %d", data.Session)
	}
}

// TestGameEventWithNilData tests GameEvent with nil data
func TestGameEventWithNilData(t *testing.T) {
	event := GameEvent{
		Type:      EventTypeInfo,
		Message:   "Info message",
		Timestamp: time.Now(),
		Data:      nil,
	}

	if event.Data != nil {
		t.Error("Data should be nil")
	}
}

// TestEventTypeComparison tests EventType comparison
func TestEventTypeComparison(t *testing.T) {
	eventType := EventTypeFame

	if eventType != EventTypeFame {
		t.Error("EventType comparison failed")
	}

	if eventType == EventTypeSilver {
		t.Error("Different EventTypes should not be equal")
	}
}

// ============================================
// Tests for service.go (non-network parts)
// ============================================

// TestServiceIsRunningInitial tests initial running state
func TestServiceIsRunningInitial(t *testing.T) {
	s := New()

	if s.IsRunning() {
		t.Error("service should not be running initially")
	}
}

// TestServiceIsOnlineWithoutCapture tests IsOnline without capture
func TestServiceIsOnlineWithoutCapture(t *testing.T) {
	s := New()

	if s.IsOnline() {
		t.Error("service should not be online without capture")
	}
}

// TestServiceSessionMetricsWithoutHandler tests session metrics without handler
func TestServiceSessionMetricsWithoutHandler(t *testing.T) {
	s := New()

	if s.SessionFame() != 0 {
		t.Errorf("SessionFame: expected 0, got %d", s.SessionFame())
	}

	if s.SessionSilver() != 0 {
		t.Errorf("SessionSilver: expected 0, got %d", s.SessionSilver())
	}

	if s.SessionKills() != 0 {
		t.Errorf("SessionKills: expected 0, got %d", s.SessionKills())
	}

	if s.SessionDeaths() != 0 {
		t.Errorf("SessionDeaths: expected 0, got %d", s.SessionDeaths())
	}

	if s.SessionLoot() != 0 {
		t.Errorf("SessionLoot: expected 0, got %d", s.SessionLoot())
	}

	if s.SessionRespec() != 0 {
		t.Errorf("SessionRespec: expected 0, got %d", s.SessionRespec())
	}

	if s.SessionRespecSilver() != 0 {
		t.Errorf("SessionRespecSilver: expected 0, got %d", s.SessionRespecSilver())
	}
}

// TestServiceParserStatsWithoutParser tests parser stats without parser
func TestServiceParserStatsWithoutParser(t *testing.T) {
	s := New()

	if s.ParserStats() != nil {
		t.Error("ParserStats should be nil without parser")
	}
}

// TestServiceHandlerWithoutStart tests handler access without start
func TestServiceHandlerWithoutStart(t *testing.T) {
	s := New()

	if s.Handler() != nil {
		t.Error("Handler should be nil before Start()")
	}
}

// TestDefaultBufferSizeConstants tests default buffer size constants
func TestDefaultBufferSizeConstants(t *testing.T) {
	if defaultEventBufferSize != 250 {
		t.Errorf("defaultEventBufferSize: expected 250, got %d", defaultEventBufferSize)
	}

	if defaultStatsBufferSize != 10 {
		t.Errorf("defaultStatsBufferSize: expected 10, got %d", defaultStatsBufferSize)
	}
}

// ============================================
// emitEvent helper tests
// ============================================

// newServiceWithParser builds a minimal Service suitable for exercising the
// emitEvent helper without invoking the full Start() path. The events channel
// is set to the requested buffer size and a real parser/stats instance is
// attached so the dropped-events counter is exercised.
func newServiceWithParser(t *testing.T, eventBuffer int) *Service {
	t.Helper()
	s := New(WithEventBufferSize(eventBuffer))
	// Attach a parser so emitEvent can increment EventsDropped on the drop
	// branch. We use the real photon.NewParser with a nil handler because we
	// never feed it packets — only Stats is accessed.
	s.parser = photon.NewParser(nil)
	if s.parser == nil || s.parser.Stats == nil {
		t.Fatalf("failed to construct parser with stats")
	}
	return s
}

// TestEmitEvent_SendsWhenSpaceAvailable verifies the happy path: with an empty
// channel the event lands in eventsChan and nothing is dropped.
func TestEmitEvent_SendsWhenSpaceAvailable(t *testing.T) {
	s := newServiceWithParser(t, 2)

	s.emitEvent(GameEvent{Type: EventTypeInfo, Message: "hello", Timestamp: time.Now()})

	select {
	case got := <-s.eventsChan:
		if got.Message != "hello" {
			t.Errorf("expected message 'hello', got %q", got.Message)
		}
	default:
		t.Error("expected event to be delivered, but channel is empty")
	}
	if dropped := s.parser.Stats.GetEventsDropped(); dropped != 0 {
		t.Errorf("expected 0 dropped events, got %d", dropped)
	}
}

// TestEmitEvent_DropsWhenChannelFull exercises the default branch: when the
// channel buffer is full the event is dropped and EventsDropped is incremented.
func TestEmitEvent_DropsWhenChannelFull(t *testing.T) {
	s := newServiceWithParser(t, 1)

	// Fill the single-slot buffer.
	s.emitEvent(GameEvent{Type: EventTypeInfo, Message: "first", Timestamp: time.Now()})
	// Second emit must hit the default branch (channel full).
	s.emitEvent(GameEvent{Type: EventTypeInfo, Message: "second", Timestamp: time.Now()})

	if dropped := s.parser.Stats.GetEventsDropped(); dropped != 1 {
		t.Errorf("expected 1 dropped event, got %d", dropped)
	}

	// Only the first event should be in the channel.
	got := <-s.eventsChan
	if got.Message != "first" {
		t.Errorf("expected only 'first' to be buffered, got %q", got.Message)
	}
	select {
	case extra := <-s.eventsChan:
		t.Errorf("expected channel to be empty after one receive, got %q", extra.Message)
	default:
		// expected
	}
}

// TestEmitEvent_NilParserDoesNotPanic verifies the helper tolerates a missing
// parser/stats (e.g. when called during teardown races).
func TestEmitEvent_NilParserDoesNotPanic(t *testing.T) {
	s := New(WithEventBufferSize(1))
	// s.parser is nil by default on a freshly constructed Service.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("emitEvent panicked with nil parser: %v", r)
		}
	}()
	// Fill buffer then overflow to exercise the nil-stats branch of the default.
	s.emitEvent(GameEvent{Type: EventTypeInfo, Message: "first", Timestamp: time.Now()})
	s.emitEvent(GameEvent{Type: EventTypeInfo, Message: "dropped", Timestamp: time.Now()})
}
