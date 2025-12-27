package photon

import (
	"sync"
	"testing"
	"time"
)

func TestNewStats(t *testing.T) {
	stats := NewStats()

	if stats == nil {
		t.Fatal("NewStats() returned nil")
	}

	if stats.StartTime.IsZero() {
		t.Error("StartTime should be set")
	}

	if stats.GetPacketsReceived() != 0 {
		t.Error("PacketsReceived should be 0")
	}
}

func TestStatsIncrementers(t *testing.T) {
	stats := NewStats()

	// Test packet counters
	stats.IncrPacketsReceived()
	stats.IncrPacketsReceived()
	if stats.GetPacketsReceived() != 2 {
		t.Errorf("Expected PacketsReceived=2, got %d", stats.GetPacketsReceived())
	}

	stats.IncrPacketsProcessed()
	if stats.GetPacketsProcessed() != 1 {
		t.Errorf("Expected PacketsProcessed=1, got %d", stats.GetPacketsProcessed())
	}

	stats.IncrPacketsEncrypted()
	if stats.GetPacketsEncrypted() != 1 {
		t.Errorf("Expected PacketsEncrypted=1, got %d", stats.GetPacketsEncrypted())
	}

	stats.IncrPacketsWithCRC()
	if stats.GetPacketsWithCRC() != 1 {
		t.Errorf("Expected PacketsWithCRC=1, got %d", stats.GetPacketsWithCRC())
	}

	stats.IncrPacketsMalformed()
	if stats.GetPacketsMalformed() != 1 {
		t.Errorf("Expected PacketsMalformed=1, got %d", stats.GetPacketsMalformed())
	}

	// Test fragment counters
	stats.IncrFragmentsReceived()
	if stats.GetFragmentsReceived() != 1 {
		t.Errorf("Expected FragmentsReceived=1, got %d", stats.GetFragmentsReceived())
	}

	stats.IncrFragmentsCompleted()
	if stats.GetFragmentsCompleted() != 1 {
		t.Errorf("Expected FragmentsCompleted=1, got %d", stats.GetFragmentsCompleted())
	}

	stats.IncrFragmentsExpired()
	if stats.GetFragmentsExpired() != 1 {
		t.Errorf("Expected FragmentsExpired=1, got %d", stats.GetFragmentsExpired())
	}

	// Test message counters
	stats.IncrEventsDecoded()
	if stats.GetEventsDecoded() != 1 {
		t.Errorf("Expected EventsDecoded=1, got %d", stats.GetEventsDecoded())
	}

	stats.IncrRequestsDecoded()
	if stats.GetRequestsDecoded() != 1 {
		t.Errorf("Expected RequestsDecoded=1, got %d", stats.GetRequestsDecoded())
	}

	stats.IncrResponsesDecoded()
	if stats.GetResponsesDecoded() != 1 {
		t.Errorf("Expected ResponsesDecoded=1, got %d", stats.GetResponsesDecoded())
	}

	stats.IncrEventsDropped()
	if stats.GetEventsDropped() != 1 {
		t.Errorf("Expected EventsDropped=1, got %d", stats.GetEventsDropped())
	}

	// Test bytes counter
	stats.AddBytesReceived(100)
	stats.AddBytesReceived(50)
	if stats.GetBytesReceived() != 150 {
		t.Errorf("Expected BytesReceived=150, got %d", stats.GetBytesReceived())
	}
}

func TestStatsUptime(t *testing.T) {
	stats := NewStats()

	// Small delay to ensure uptime > 0
	time.Sleep(10 * time.Millisecond)

	uptime := stats.Uptime()
	if uptime <= 0 {
		t.Errorf("Expected uptime > 0, got %v", uptime)
	}
}

func TestStatsPacketsPerSecond(t *testing.T) {
	stats := NewStats()

	// Initially should be 0 (no packets, minimal time)
	pps := stats.PacketsPerSecond()
	if pps != 0 {
		// It's okay if it's very small due to timing
		t.Logf("Initial PacketsPerSecond: %f", pps)
	}

	// Add packets and wait a bit
	for i := 0; i < 100; i++ {
		stats.IncrPacketsReceived()
	}

	time.Sleep(100 * time.Millisecond)

	pps = stats.PacketsPerSecond()
	if pps <= 0 {
		t.Errorf("Expected PacketsPerSecond > 0 after receiving packets, got %f", pps)
	}
}

func TestStatsFormatUptime(t *testing.T) {
	stats := NewStats()

	formatted := stats.FormatUptime()
	if formatted == "" {
		t.Error("FormatUptime() returned empty string")
	}

	// Should be in HH:MM:SS format
	if len(formatted) != 8 {
		t.Errorf("Expected format HH:MM:SS (8 chars), got %s (%d chars)", formatted, len(formatted))
	}
}

func TestStatsSummary(t *testing.T) {
	stats := NewStats()
	stats.IncrPacketsReceived()
	stats.IncrEventsDecoded()
	stats.IncrPacketsEncrypted()
	stats.IncrPacketsWithCRC()

	summary := stats.Summary()
	if summary == "" {
		t.Error("Summary() returned empty string")
	}

	// Should contain key metrics
	if !containsSubstring(summary, "Packets:") {
		t.Error("Summary should contain 'Packets:'")
	}
	if !containsSubstring(summary, "Events:") {
		t.Error("Summary should contain 'Events:'")
	}
}

func TestStatsReset(t *testing.T) {
	stats := NewStats()

	// Set some values
	stats.IncrPacketsReceived()
	stats.IncrPacketsReceived()
	stats.IncrEventsDecoded()
	stats.AddBytesReceived(1000)

	// Set buffer metrics
	stats.UpdateBufferPeak(200)
	stats.SnapshotBufferPeak() // This sets BufferPeakDisplay=200, bufferPeakInternal=0
	stats.UpdateBufferPeak(50) // Set internal to 50
	stats.BufferCapacity = 250

	// Verify they're set
	if stats.GetPacketsReceived() != 2 {
		t.Errorf("Expected PacketsReceived=2, got %d", stats.GetPacketsReceived())
	}
	if stats.BufferPeakDisplay != 200 {
		t.Errorf("Expected BufferPeakDisplay=200, got %d", stats.BufferPeakDisplay)
	}
	if stats.bufferPeakInternal != 50 {
		t.Errorf("Expected bufferPeakInternal=50, got %d", stats.bufferPeakInternal)
	}

	// Reset
	stats.Reset()

	// Verify they're zeroed
	if stats.GetPacketsReceived() != 0 {
		t.Errorf("After reset, expected PacketsReceived=0, got %d", stats.GetPacketsReceived())
	}
	if stats.GetEventsDecoded() != 0 {
		t.Errorf("After reset, expected EventsDecoded=0, got %d", stats.GetEventsDecoded())
	}
	if stats.GetBytesReceived() != 0 {
		t.Errorf("After reset, expected BytesReceived=0, got %d", stats.GetBytesReceived())
	}

	// Verify buffer metrics are reset
	if stats.BufferPeakDisplay != 0 {
		t.Errorf("After reset, expected BufferPeakDisplay=0, got %d", stats.BufferPeakDisplay)
	}
	if stats.bufferPeakInternal != 0 {
		t.Errorf("After reset, expected bufferPeakInternal=0, got %d", stats.bufferPeakInternal)
	}
	// Note: BufferCapacity is an invariant and intentionally not reset
}

func TestEventsDropped(t *testing.T) {
	stats := NewStats()

	// Initial value should be 0
	if stats.GetEventsDropped() != 0 {
		t.Error("Initial events dropped should be 0")
	}

	// Increment and verify
	stats.IncrEventsDropped()
	if stats.GetEventsDropped() != 1 {
		t.Error("Events dropped should be 1 after increment")
	}

	// Multiple increments
	stats.IncrEventsDropped()
	stats.IncrEventsDropped()
	if stats.GetEventsDropped() != 3 {
		t.Errorf("Events dropped should be 3, got %d", stats.GetEventsDropped())
	}

	// Reset should zero it
	stats.Reset()
	if stats.GetEventsDropped() != 0 {
		t.Error("Events dropped should be 0 after reset")
	}
}

func TestEventsDroppedConcurrent(t *testing.T) {
	stats := NewStats()
	var wg sync.WaitGroup

	// Simulate 1000 concurrent drops from multiple goroutines
	// This tests thread-safety of atomic operations
	numGoroutines := 10
	dropsPerGoroutine := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < dropsPerGoroutine; j++ {
				stats.IncrEventsDropped()
			}
		}()
	}

	wg.Wait()

	expectedDrops := uint64(numGoroutines * dropsPerGoroutine)
	actualDrops := stats.GetEventsDropped()
	if actualDrops != expectedDrops {
		t.Errorf("Expected %d drops, got %d", expectedDrops, actualDrops)
	}
}

func TestParserHasStats(t *testing.T) {
	parser := NewParser(nil)
	defer parser.Close()

	if parser.Stats == nil {
		t.Fatal("Parser.Stats should not be nil")
	}

	// Stats should be initialized
	if parser.Stats.StartTime.IsZero() {
		t.Error("Parser.Stats.StartTime should be set")
	}
}

func TestUpdateBufferPeak(t *testing.T) {
	stats := NewStats()

	// Initial value should be 0
	if stats.bufferPeakInternal != 0 {
		t.Errorf("Initial bufferPeakInternal should be 0, got %d", stats.bufferPeakInternal)
	}

	// Update with increasing values
	stats.UpdateBufferPeak(10)
	if stats.bufferPeakInternal != 10 {
		t.Errorf("Expected bufferPeakInternal=10, got %d", stats.bufferPeakInternal)
	}

	stats.UpdateBufferPeak(50)
	if stats.bufferPeakInternal != 50 {
		t.Errorf("Expected bufferPeakInternal=50, got %d", stats.bufferPeakInternal)
	}

	// Update with smaller value should not change peak
	stats.UpdateBufferPeak(30)
	if stats.bufferPeakInternal != 50 {
		t.Errorf("Expected bufferPeakInternal=50 (unchanged), got %d", stats.bufferPeakInternal)
	}

	// Update with larger value should update
	stats.UpdateBufferPeak(100)
	if stats.bufferPeakInternal != 100 {
		t.Errorf("Expected bufferPeakInternal=100, got %d", stats.bufferPeakInternal)
	}

	// Update with equal value should not change
	stats.UpdateBufferPeak(100)
	if stats.bufferPeakInternal != 100 {
		t.Errorf("Expected bufferPeakInternal=100 (unchanged), got %d", stats.bufferPeakInternal)
	}
}

func TestSnapshotBufferPeak(t *testing.T) {
	stats := NewStats()

	// Set peak to 150
	stats.UpdateBufferPeak(150)
	if stats.bufferPeakInternal != 150 {
		t.Errorf("Expected bufferPeakInternal=150, got %d", stats.bufferPeakInternal)
	}

	// Take snapshot
	stats.SnapshotBufferPeak()

	// BufferPeakDisplay should have the peak
	if stats.BufferPeakDisplay != 150 {
		t.Errorf("Expected BufferPeakDisplay=150, got %d", stats.BufferPeakDisplay)
	}

	// bufferPeakInternal should be reset to 0
	if stats.bufferPeakInternal != 0 {
		t.Errorf("Expected bufferPeakInternal=0 after snapshot, got %d", stats.bufferPeakInternal)
	}

	// Update to new value (80) in new interval
	stats.UpdateBufferPeak(80)
	if stats.bufferPeakInternal != 80 {
		t.Errorf("Expected bufferPeakInternal=80, got %d", stats.bufferPeakInternal)
	}

	// Take another snapshot
	stats.SnapshotBufferPeak()

	// BufferPeakDisplay should now show 80 (new interval peak)
	if stats.BufferPeakDisplay != 80 {
		t.Errorf("Expected BufferPeakDisplay=80 (new interval), got %d", stats.BufferPeakDisplay)
	}

	// bufferPeakInternal should be reset again
	if stats.bufferPeakInternal != 0 {
		t.Errorf("Expected bufferPeakInternal=0 after second snapshot, got %d", stats.bufferPeakInternal)
	}
}

func TestBufferPeakConcurrent(t *testing.T) {
	stats := NewStats()
	var wg sync.WaitGroup

	// Simulate 10 goroutines concurrently updating buffer peak
	// Each goroutine sends values from 0 to 99
	numGoroutines := 10
	valuesPerGoroutine := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < valuesPerGoroutine; j++ {
				stats.UpdateBufferPeak(j)
			}
		}()
	}

	wg.Wait()

	// The peak should be 99 (maximum value sent by any goroutine)
	expectedPeak := int64(valuesPerGoroutine - 1)
	actualPeak := stats.bufferPeakInternal
	if actualPeak != expectedPeak {
		t.Errorf("Expected bufferPeakInternal=%d, got %d", expectedPeak, actualPeak)
	}
}

// Helper function
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstringHelper(s, substr))
}

func containsSubstringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
