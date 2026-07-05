package handlers

import (
	"sync"
	"testing"

	"github.com/cantalupo555/albion-lens/pkg/events"
)

func TestParseMapType(t *testing.T) {
	tests := []struct {
		cluster string
		want    MapType
	}{
		// Exact map-type keywords (case-insensitive).
		{"@RANDOMDUNGEON@abcdef", MapTypeRandomDungeon},
		{"@RANDOMDUNGEON@", MapTypeRandomDungeon},
		{"@HELLCLUSTER@xyz", MapTypeHellGate},
		{"@CORRUPTEDDUNGEON@1234", MapTypeCorruptedDungeon},
		{"@ISLAND@my-island-guid", MapTypeIsland},
		{"@HIDEOUT@4001@hideout-guid", MapTypeHideout},
		{"@EXPEDITION@exp1", MapTypeExpedition},
		{"@ARENA@arena1", MapTypeArena},
		// Order-dependent: MISTSDUNGEON must match before MISTS.
		{"@MISTSDUNGEON@mists-d-guid", MapTypeMistsDungeon},
		{"@MISTS@mists-guid", MapTypeMists},
		{"@HELLDUNGEON@abyss-guid", MapTypeAbyssalDepths},
		// Lowercase input must also work (reference uses ToUpper before Contains).
		{"@island@lc-guid", MapTypeIsland},
		{"@mistsdungeon@lc-guid", MapTypeMistsDungeon},
		// Open-world / city: no '@', plain index → Unknown.
		{"4000", MapTypeUnknown},
		{"3301", MapTypeUnknown},
		// Empty / no match.
		{"", MapTypeUnknown},
		{"unknown-cluster", MapTypeUnknown},
	}

	for _, tt := range tests {
		got := ParseMapType(tt.cluster)
		if got != tt.want {
			t.Errorf("ParseMapType(%q) = %v, want %v", tt.cluster, got, tt.want)
		}
	}
}

func TestMapTypeString(t *testing.T) {
	tests := []struct {
		mt   MapType
		want string
	}{
		{MapTypeUnknown, "Unknown"},
		{MapTypeRandomDungeon, "Random Dungeon"},
		{MapTypeHellGate, "Hell Gate"},
		{MapTypeCorruptedDungeon, "Corrupted Dungeon"},
		{MapTypeIsland, "Island"},
		{MapTypeHideout, "Hideout"},
		{MapTypeExpedition, "Expedition"},
		{MapTypeArena, "Arena"},
		{MapTypeMists, "Mists"},
		{MapTypeMistsDungeon, "Mists Dungeon"},
		{MapTypeAbyssalDepths, "Abyssal Depths"},
	}
	for _, tt := range tests {
		if got := tt.mt.String(); got != tt.want {
			t.Errorf("MapType(%d).String() = %q, want %q", tt.mt, got, tt.want)
		}
	}
}

func TestZoneInfoDisplayString(t *testing.T) {
	tests := []struct {
		name string
		info ZoneInfo
		want string
	}{
		{
			name: "zero value",
			info: ZoneInfo{},
			want: "",
		},
		{
			name: "open world city",
			info: ZoneInfo{MapType: MapTypeUnknown, ClusterIndex: "4000"},
			want: "Fort Sterling",
		},
		{
			name: "open world zone with tier",
			info: ZoneInfo{MapType: MapTypeUnknown, ClusterIndex: "3348"},
			want: "Battlebrae Grassland · T7 · Black",
		},
		{
			name: "unknown open world zone",
			info: ZoneInfo{MapType: MapTypeUnknown, ClusterIndex: "9999"},
			want: "9999",
		},
		{
			name: "random dungeon",
			info: ZoneInfo{MapType: MapTypeRandomDungeon, ClusterIndex: "@RANDOMDUNGEON@guid"},
			want: "Random Dungeon",
		},
		{
			name: "island with name",
			info: ZoneInfo{MapType: MapTypeIsland, IslandName: "My Island"},
			want: "Island — My Island",
		},
		{
			name: "island without name",
			info: ZoneInfo{MapType: MapTypeIsland},
			want: "Island",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.info.DisplayString(); got != tt.want {
				t.Errorf("DisplayString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseChangeClusterResponse(t *testing.T) {
	t.Run("island with name", func(t *testing.T) {
		info := parseChangeClusterResponse(map[byte]interface{}{
			0: "@ISLAND@my-island-guid",
			2: "My Island",
		})
		if info.MapType != MapTypeIsland {
			t.Errorf("expected MapTypeIsland, got %v", info.MapType)
		}
		if info.IslandName != "My Island" {
			t.Errorf("expected IslandName 'My Island', got %q", info.IslandName)
		}
		if info.ClusterIndex != "@ISLAND@my-island-guid" {
			t.Errorf("expected raw cluster index, got %q", info.ClusterIndex)
		}
	})

	t.Run("random dungeon with dungeon info", func(t *testing.T) {
		info := parseChangeClusterResponse(map[byte]interface{}{
			0: "@RANDOMDUNGEON@dungeon-guid",
			3: []byte{0x01, 0x02, 0x03},
		})
		if info.MapType != MapTypeRandomDungeon {
			t.Errorf("expected MapTypeRandomDungeon, got %v", info.MapType)
		}
		if !info.HasDungeonInfo {
			t.Error("expected HasDungeonInfo=true")
		}
	})

	t.Run("hideout with main cluster index", func(t *testing.T) {
		info := parseChangeClusterResponse(map[byte]interface{}{
			0: "@HIDEOUT@4001@hideout-guid",
		})
		if info.MapType != MapTypeHideout {
			t.Errorf("expected MapTypeHideout, got %v", info.MapType)
		}
		if info.MainClusterIndex != "4001" {
			t.Errorf("expected MainClusterIndex '4001', got %q", info.MainClusterIndex)
		}
	})

	t.Run("open world city", func(t *testing.T) {
		info := parseChangeClusterResponse(map[byte]interface{}{
			0: "4000",
		})
		if info.MapType != MapTypeUnknown {
			t.Errorf("expected MapTypeUnknown for open world, got %v", info.MapType)
		}
		if info.ClusterIndex != "4000" {
			t.Errorf("expected ClusterIndex '4000', got %q", info.ClusterIndex)
		}
	})

	t.Run("empty params", func(t *testing.T) {
		info := parseChangeClusterResponse(map[byte]interface{}{})
		if info.MapType != MapTypeUnknown {
			t.Errorf("expected MapTypeUnknown for empty params, got %v", info.MapType)
		}
	})

	t.Run("nil params", func(t *testing.T) {
		info := parseChangeClusterResponse(nil)
		if info.MapType != MapTypeUnknown {
			t.Errorf("expected MapTypeUnknown for nil params, got %v", info.MapType)
		}
	})

	t.Run("single part at-string fallback", func(t *testing.T) {
		// Cluster strings with '@' but only one non-empty part (e.g. "@ISLAND@")
		// must fall back to MapTypeUnknown, not call ParseMapType.
		info := parseChangeClusterResponse(map[byte]interface{}{
			0: "@ISLAND@",
		})
		if info.MapType != MapTypeUnknown {
			t.Errorf("expected MapTypeUnknown for single-part @-string, got %v", info.MapType)
		}
		info = parseChangeClusterResponse(map[byte]interface{}{
			0: "@@",
		})
		if info.MapType != MapTypeUnknown {
			t.Errorf("expected MapTypeUnknown for '@@', got %v", info.MapType)
		}
	})
}

// --- Handler dispatch tests ---

func TestChangeClusterUpdatesZone(t *testing.T) {
	handler := NewAlbionHandler()

	var received *ZoneEventData
	handler.SetEventCallback(func(eventType, message string, data interface{}) {
		if eventType == "zone" {
			received = data.(*ZoneEventData)
		}
	})

	handler.OnResponse(events.OperationChangeCluster, 0, "", map[byte]interface{}{
		0: "@ISLAND@my-island",
		2: "My Island",
	})
	zone := handler.GetCurrentZone()
	if zone.MapType != MapTypeIsland {
		t.Errorf("expected GetCurrentZone MapTypeIsland, got %v", zone.MapType)
	}
	if zone.IslandName != "My Island" {
		t.Errorf("expected IslandName 'My Island', got %q", zone.IslandName)
	}
	if received == nil {
		t.Fatal("zone callback was not called")
	}
	if received.MapType != MapTypeIsland {
		t.Errorf("expected event MapTypeIsland, got %v", received.MapType)
	}
	if received.Previous != MapTypeUnknown {
		t.Errorf("expected Previous MapTypeUnknown on first transition, got %v", received.Previous)
	}
	if received.Display != "Island — My Island" {
		t.Errorf("expected Display 'Island — My Island', got %q", received.Display)
	}
}

func TestChangeClusterPreviousZone(t *testing.T) {
	handler := NewAlbionHandler()

	var zoneEvents []ZoneEventData
	handler.SetEventCallback(func(eventType, message string, data interface{}) {
		if eventType == "zone" {
			zoneEvents = append(zoneEvents, *data.(*ZoneEventData))
		}
	})

	// First transition: to a random dungeon.
	handler.OnResponse(events.OperationChangeCluster, 0, "", map[byte]interface{}{
		0: "@RANDOMDUNGEON@d1",
	})

	// Second transition: to an island.
	handler.OnResponse(events.OperationChangeCluster, 0, "", map[byte]interface{}{
		0: "@ISLAND@i1",
		2: "Farm",
	})

	if len(zoneEvents) != 2 {
		t.Fatalf("expected 2 zone events, got %d", len(zoneEvents))
	}
	if zoneEvents[0].Previous != MapTypeUnknown {
		t.Errorf("first transition Previous should be Unknown, got %v", zoneEvents[0].Previous)
	}
	if zoneEvents[1].Previous != MapTypeRandomDungeon {
		t.Errorf("second transition Previous should be RandomDungeon, got %v", zoneEvents[1].Previous)
	}
}

func TestChangeClusterDedup(t *testing.T) {
	handler := NewAlbionHandler()

	callCount := 0
	handler.SetEventCallback(func(eventType, message string, data interface{}) {
		if eventType == "zone" {
			callCount++
		}
	})

	params := map[byte]interface{}{
		0: "@RANDOMDUNGEON@dungeon-guid",
	}

	// First response: should fire.
	handler.OnResponse(events.OperationChangeCluster, 0, "", params)
	if callCount != 1 {
		t.Fatalf("expected 1 callback after first response, got %d", callCount)
	}

	// Duplicate response: should be suppressed.
	handler.OnResponse(events.OperationChangeCluster, 0, "", params)
	if callCount != 1 {
		t.Errorf("expected 1 callback after duplicate (dedup), got %d", callCount)
	}

	// Different zone: should fire again.
	handler.OnResponse(events.OperationChangeCluster, 0, "", map[byte]interface{}{
		0: "@ISLAND@different-island",
	})
	if callCount != 2 {
		t.Errorf("expected 2 callbacks after different zone, got %d", callCount)
	}
}

func TestChangeClusterOpenWorld(t *testing.T) {
	handler := NewAlbionHandler()

	var received *ZoneEventData
	handler.SetEventCallback(func(eventType, message string, data interface{}) {
		if eventType == "zone" {
			received = data.(*ZoneEventData)
		}
	})

	handler.OnResponse(events.OperationChangeCluster, 0, "", map[byte]interface{}{
		0: "4000",
	})

	zone := handler.GetCurrentZone()
	if zone.MapType != MapTypeUnknown {
		t.Errorf("expected MapTypeUnknown for open world, got %v", zone.MapType)
	}
	if zone.ClusterIndex != "4000" {
		t.Errorf("expected ClusterIndex '4000', got %q", zone.ClusterIndex)
	}
	if received == nil {
		t.Fatal("zone callback was not called for open world")
	}
	if received.Display != "Fort Sterling" {
		t.Errorf("expected Display 'Fort Sterling', got %q", received.Display)
	}
}

func TestNonChangeClusterDoesNotTouchZone(t *testing.T) {
	handler := NewAlbionHandler()

	callCount := 0
	handler.SetEventCallback(func(eventType, message string, data interface{}) {
		if eventType == "zone" {
			callCount++
		}
	})

	// OpJoin should not produce a zone event.
	handler.OnResponse(events.OperationJoin, 0, "", map[byte]interface{}{
		2: "Hero",
	})

	if callCount != 0 {
		t.Errorf("expected 0 zone callbacks from OpJoin, got %d", callCount)
	}
	zone := handler.GetCurrentZone()
	if zone.MapType != MapTypeUnknown {
		t.Errorf("expected Unknown zone before first ChangeCluster, got %v", zone.MapType)
	}
	if zone.ClusterIndex != "" {
		t.Errorf("expected empty ClusterIndex before first ChangeCluster, got %q", zone.ClusterIndex)
	}
}

func TestGetCurrentZoneInitial(t *testing.T) {
	handler := NewAlbionHandler()
	zone := handler.GetCurrentZone()
	if zone.MapType != MapTypeUnknown {
		t.Errorf("expected Unknown before first transition, got %v", zone.MapType)
	}
	if zone.ClusterIndex != "" {
		t.Errorf("expected empty ClusterIndex before first transition, got %q", zone.ClusterIndex)
	}
}

// TestGetCurrentZoneConcurrent verifies that zone state is safe for concurrent
// access: one goroutine writes via OnResponse (ChangeCluster) while another
// reads via GetCurrentZone. Under `go test -race` this flags a data race if the
// mutex is missing or misused.
func TestGetCurrentZoneConcurrent(t *testing.T) {
	handler := NewAlbionHandler()

	var wg sync.WaitGroup
	stop := make(chan struct{})

	// Reader goroutine.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				_ = handler.GetCurrentZone()
			}
		}
	}()

	// Writer goroutine.
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(stop)
		clusters := []string{
			"@RANDOMDUNGEON@d1",
			"@ISLAND@i1",
			"4000",
			"@MISTS@m1",
			"@HIDEOUT@4001@h1",
		}
		for i := 0; i < 200; i++ {
			c := clusters[i%len(clusters)]
			handler.OnResponse(events.OperationChangeCluster, 0, "", map[byte]interface{}{
				0: c,
			})
		}
	}()

	wg.Wait()
}

// TestResolveOperationCode verifies that the actual operation code is read
// from param[253], overriding the Photon header byte. This mirrors the
// reference AlbionParser behavior.
func TestResolveOperationCode(t *testing.T) {
	tests := []struct {
		name       string
		headerCode byte
		params     map[byte]interface{}
		want       byte
	}{
		{
			name:       "param 253 overrides header (byte)",
			headerCode: 99,
			params:     map[byte]interface{}{253: byte(41)},
			want:       41,
		},
		{
			name:       "param 253 overrides header (int16)",
			headerCode: 99,
			params:     map[byte]interface{}{253: int16(41)},
			want:       41,
		},
		{
			name:       "fallback to header when param 253 absent",
			headerCode: 2,
			params:     map[byte]interface{}{0: "data"},
			want:       2,
		},
		{
			name:       "fallback to header when params empty",
			headerCode: 46,
			params:     nil,
			want:       46,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveOperationCode(tt.headerCode, tt.params)
			if got != tt.want {
				t.Errorf("resolveOperationCode(%d, %v) = %d, want %d", tt.headerCode, tt.params, got, tt.want)
			}
		})
	}
}

// TestOnResponseUsesParam253 verifies that OnResponse dispatches based on
// param[253], not the Photon header byte.
func TestOnResponseUsesParam253(t *testing.T) {
	handler := NewAlbionHandler()

	var received *ZoneEventData
	handler.SetEventCallback(func(eventType, message string, data interface{}) {
		if eventType == "zone" {
			received = data.(*ZoneEventData)
		}
	})

	// Simulate a real packet: Photon header byte is garbage (0xFF),
	// but param[253] has the real operation code (41 = ChangeCluster).
	handler.OnResponse(0xFF, 0, "", map[byte]interface{}{
		253: int16(events.OperationChangeCluster),
		0:   "@ISLAND@test-island",
		2:   "Test Island",
	})

	if received == nil {
		t.Fatal("expected zone event when param[253] = OperationChangeCluster, got nil")
	}
	if received.MapType != MapTypeIsland {
		t.Errorf("expected MapTypeIsland, got %v", received.MapType)
	}
}

// TestOnResponseJoinUsesParam253 verifies that the Join response also works
// through param[253] resolution.
func TestOnResponseJoinUsesParam253(t *testing.T) {
	handler := NewAlbionHandler()

	// Photon header byte is garbage, but param[253] = 2 (Join).
	handler.OnResponse(0xFF, 0, "", map[byte]interface{}{
		253: int16(events.OperationJoin),
		2:   "TestPlayer",
	})

	if name := handler.GetLocalPlayer(); name != "TestPlayer" {
		t.Errorf("expected local player 'TestPlayer', got %q", name)
	}
}

func TestClusterName(t *testing.T) {
	tests := []struct {
		index string
		want  string
	}{
		{"4000", "Fort Sterling"},
		{"0000", "Thetford"},
		{"3004", "Martlock"},
		{"3005", "Caerleon"},
		{"5000", "Brecilien"},
		{"9999", "9999"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := ClusterName(tt.index); got != tt.want {
			t.Errorf("ClusterName(%q) = %q, want %q", tt.index, got, tt.want)
		}
	}
}

func TestClusterDisplay(t *testing.T) {
	tests := []struct {
		index string
		want  string
	}{
		{"4000", "Fort Sterling"},                     // City T1 → just name
		{"3005", "Caerleon"},                          // City T7 → just name (cities omit tier)
		{"3348", "Battlebrae Grassland · T7 · Black"}, // Outlands T7
		{"0201", "Sleetwater Basin · T5 · Red"},       // Royal T5 → Red
		{"0202", "Chillhag · T3 · Yellow"},            // Royal T3 → Yellow
		{"1008", "The Lighthouse · Blue"},             // T1 no area → Blue
		{"9999", "9999"},                              // Unknown → raw
	}
	for _, tt := range tests {
		if got := ClusterDisplay(tt.index); got != tt.want {
			t.Errorf("ClusterDisplay(%q) = %q, want %q", tt.index, got, tt.want)
		}
	}
}

// TestClusterChangeCallbackFiresOnTransition verifies the cluster-change
// callback fires exactly once per confirmed material zone transition.
func TestClusterChangeCallbackFiresOnTransition(t *testing.T) {
	handler := NewAlbionHandler()

	var calls int
	handler.SetClusterChangeCallback(func() { calls++ })

	// First transition (Unknown → Island): fires.
	handler.OnResponse(events.OperationChangeCluster, 0, "", map[byte]interface{}{
		0: "@ISLAND@island-1",
		2: "Island One",
	})
	if calls != 1 {
		t.Errorf("after first transition, callback calls = %d, want 1", calls)
	}

	// Second transition (Island → open-world city): fires.
	handler.OnResponse(events.OperationChangeCluster, 0, "", map[byte]interface{}{
		0: "4000",
	})
	if calls != 2 {
		t.Errorf("after second transition, callback calls = %d, want 2", calls)
	}
}

// TestClusterChangeCallbackSkippedOnDedup verifies the callback does NOT fire
// for a suppressed duplicate transition (same map type, cluster, island).
func TestClusterChangeCallbackSkippedOnDedup(t *testing.T) {
	handler := NewAlbionHandler()

	var calls int
	handler.SetClusterChangeCallback(func() { calls++ })

	params := map[byte]interface{}{
		0: "@ISLAND@island-1",
		2: "Same Island",
	}
	// First transition: fires.
	handler.OnResponse(events.OperationChangeCluster, 0, "", params)
	// Identical repeat: dedup-suppressed, callback must not fire.
	handler.OnResponse(events.OperationChangeCluster, 0, "", params)

	if calls != 1 {
		t.Errorf("callback calls after duplicate transition = %d, want 1", calls)
	}
}

// TestClusterChangeCallbackClears confirms passing nil disables the callback.
func TestClusterChangeCallbackClears(t *testing.T) {
	handler := NewAlbionHandler()
	var calls int
	handler.SetClusterChangeCallback(func() { calls++ })
	handler.SetClusterChangeCallback(nil)

	handler.OnResponse(events.OperationChangeCluster, 0, "", map[byte]interface{}{
		0: "4000",
	})
	if calls != 0 {
		t.Errorf("callback fired after being cleared: %d calls", calls)
	}
}
