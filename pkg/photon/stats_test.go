package photon

import (
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

	// Verify they're set
	if stats.GetPacketsReceived() != 2 {
		t.Errorf("Expected PacketsReceived=2, got %d", stats.GetPacketsReceived())
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
