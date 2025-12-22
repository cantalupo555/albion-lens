package photon

import (
	"testing"
	"time"
)

// mockHandler implements PhotonHandler for testing
type mockHandler struct {
	events    int
	requests  int
	responses int
}

func (m *mockHandler) OnEvent(eventCode byte, parameters map[byte]interface{}) {
	m.events++
}

func (m *mockHandler) OnRequest(operationCode byte, parameters map[byte]interface{}) {
	m.requests++
}

func (m *mockHandler) OnResponse(operationCode byte, returnCode int16, debugMessage string, parameters map[byte]interface{}) {
	m.responses++
}

func TestNewParser(t *testing.T) {
	handler := &mockHandler{}
	parser := NewParser(handler)
	defer parser.Close()

	if parser == nil {
		t.Fatal("NewParser returned nil")
	}

	if parser.pendingFragments == nil {
		t.Error("pendingFragments map not initialized")
	}

	if parser.stopCleanup == nil {
		t.Error("stopCleanup channel not initialized")
	}
}

func TestParserClose(t *testing.T) {
	handler := &mockHandler{}
	parser := NewParser(handler)

	// Close should not panic
	parser.Close()

	// Give goroutine time to stop
	time.Sleep(50 * time.Millisecond)
}

func TestPendingFragmentsCount(t *testing.T) {
	handler := &mockHandler{}
	parser := NewParser(handler)
	defer parser.Close()

	if count := parser.PendingFragmentsCount(); count != 0 {
		t.Errorf("expected 0 pending fragments, got %d", count)
	}

	// Manually add a fragment for testing
	parser.fragmentsMu.Lock()
	parser.pendingFragments[1] = &fragmentedPacket{
		totalLength:  100,
		payload:      make([]byte, 100),
		bytesWritten: 50,
		createdAt:    time.Now(),
	}
	parser.fragmentsMu.Unlock()

	if count := parser.PendingFragmentsCount(); count != 1 {
		t.Errorf("expected 1 pending fragment, got %d", count)
	}
}

func TestCleanupExpiredFragments(t *testing.T) {
	handler := &mockHandler{}
	parser := NewParser(handler)
	defer parser.Close()

	// Add an expired fragment (created 1 minute ago)
	parser.fragmentsMu.Lock()
	parser.pendingFragments[1] = &fragmentedPacket{
		totalLength:  100,
		payload:      make([]byte, 100),
		bytesWritten: 50,
		createdAt:    time.Now().Add(-1 * time.Minute), // 1 minute ago (expired)
	}

	// Add a fresh fragment
	parser.pendingFragments[2] = &fragmentedPacket{
		totalLength:  100,
		payload:      make([]byte, 100),
		bytesWritten: 50,
		createdAt:    time.Now(), // Just created (not expired)
	}
	parser.fragmentsMu.Unlock()

	if count := parser.PendingFragmentsCount(); count != 2 {
		t.Fatalf("expected 2 pending fragments before cleanup, got %d", count)
	}

	// Run cleanup
	parser.cleanupExpiredFragments()

	// Should have removed the expired one
	if count := parser.PendingFragmentsCount(); count != 1 {
		t.Errorf("expected 1 pending fragment after cleanup, got %d", count)
	}

	// Verify the correct one was removed
	parser.fragmentsMu.RLock()
	_, exists1 := parser.pendingFragments[1]
	_, exists2 := parser.pendingFragments[2]
	parser.fragmentsMu.RUnlock()

	if exists1 {
		t.Error("expired fragment (seq 1) should have been removed")
	}

	if !exists2 {
		t.Error("fresh fragment (seq 2) should still exist")
	}
}

func TestCleanupLoopStops(t *testing.T) {
	handler := &mockHandler{}
	parser := NewParser(handler)

	// Close immediately
	parser.Close()

	// Wait a bit to ensure goroutine has time to stop
	time.Sleep(100 * time.Millisecond)

	// If we get here without hanging, the test passes
}

func TestFragmentTTLConstants(t *testing.T) {
	// Verify constants are set correctly
	if FragmentTTL != 30*time.Second {
		t.Errorf("expected FragmentTTL to be 30s, got %v", FragmentTTL)
	}

	if FragmentCleanupInterval != 10*time.Second {
		t.Errorf("expected FragmentCleanupInterval to be 10s, got %v", FragmentCleanupInterval)
	}
}
