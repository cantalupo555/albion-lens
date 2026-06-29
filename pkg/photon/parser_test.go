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

	// Close blocks until cleanupLoop has exited; returning here means the
	// WaitGroup was satisfied — the goroutine called Done().
	parser.Close()
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

	// Close blocks until cleanupLoop exits — no sleep needed.
	parser.Close()

	// Verify stopCleanup was actually closed (proves Close ran).
	select {
	case <-parser.stopCleanup:
		// expected — channel is closed
	default:
		t.Error("stopCleanup not closed after Close()")
	}
}

func TestParserDoubleCloseNoPanic(t *testing.T) {
	handler := &mockHandler{}
	parser := NewParser(handler)

	parser.Close()
	// Second close must not panic (sync.Once guards it).
	parser.Close()
}

func TestParserCloseBlocksUntilGoroutineExits(t *testing.T) {
	handler := &mockHandler{}
	parser := NewParser(handler)

	// Before Close, stopCleanup must be open.
	select {
	case <-parser.stopCleanup:
		t.Fatal("stopCleanup closed before Close()")
	default:
	}

	// Close blocks until cleanupLoop has called wg.Done().
	parser.Close()

	// After Close, stopCleanup is closed.
	select {
	case <-parser.stopCleanup:
		// expected
	default:
		t.Error("stopCleanup not closed after Close()")
	}

	// wg.Wait() should return immediately — goroutine already exited.
	parser.wg.Wait()
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

// photonHeader builds a 12-byte Photon header for tests.
// Layout: peerId[0:2] | flags | commandCount | timestamp[4] | challenge[4]
func photonHeader(flags byte, commandCount byte) []byte {
	h := make([]byte, PhotonHeaderLength)
	h[2] = flags
	h[3] = commandCount
	return h
}

// commandHeader builds a 12-byte Photon command header for tests.
// commandLength is the total length INCLUDING the 12-byte header itself,
// matching the parser's interpretation (dataLength = commandLength - CommandHeaderLength).
func commandHeader(commandType byte, commandLength uint32) []byte {
	h := make([]byte, CommandHeaderLength)
	h[0] = commandType
	h[4] = byte(commandLength >> 24)
	h[5] = byte(commandLength >> 16)
	h[6] = byte(commandLength >> 8)
	h[7] = byte(commandLength)
	return h
}

// TestParsePacket_CommandHeaderTruncated verifies that a packet with a valid
// Photon header but no command bytes (commandCount > 0 but buffer empty) is
// counted as malformed, not processed.
func TestParsePacket_CommandHeaderTruncated(t *testing.T) {
	handler := &mockHandler{}
	parser := NewParser(handler)
	defer parser.Close()

	// 12-byte header claims 1 command, but no command bytes follow.
	payload := photonHeader(0, 1)

	if err := parser.ParsePacket(payload); err != nil {
		t.Fatalf("ParsePacket returned error: %v", err)
	}

	if got := parser.Stats.GetPacketsMalformed(); got != 1 {
		t.Errorf("PacketsMalformed = %d, want 1", got)
	}
	if got := parser.Stats.GetPacketsProcessed(); got != 0 {
		t.Errorf("PacketsProcessed = %d, want 0", got)
	}
}

// TestParsePacket_CommandDataTruncated verifies that a packet whose command
// header advertises more data bytes than actually present is counted as
// malformed.
func TestParsePacket_CommandDataTruncated(t *testing.T) {
	handler := &mockHandler{}
	parser := NewParser(handler)
	defer parser.Close()

	// Header (commandCount=1) + command header claiming commandLength=100
	// (dataLength=88) but no data bytes follow.
	payload := append(photonHeader(0, 1), commandHeader(0, 100)...)

	if err := parser.ParsePacket(payload); err != nil {
		t.Fatalf("ParsePacket returned error: %v", err)
	}

	if got := parser.Stats.GetPacketsMalformed(); got != 1 {
		t.Errorf("PacketsMalformed = %d, want 1", got)
	}
	if got := parser.Stats.GetPacketsProcessed(); got != 0 {
		t.Errorf("PacketsProcessed = %d, want 0", got)
	}
}

// TestParsePacket_NoCommands_Processed verifies that a packet with commandCount=0
// (ack-only) is counted as processed, not malformed.
func TestParsePacket_NoCommands_Processed(t *testing.T) {
	handler := &mockHandler{}
	parser := NewParser(handler)
	defer parser.Close()

	payload := photonHeader(0, 0)

	if err := parser.ParsePacket(payload); err != nil {
		t.Fatalf("ParsePacket returned error: %v", err)
	}

	if got := parser.Stats.GetPacketsProcessed(); got != 1 {
		t.Errorf("PacketsProcessed = %d, want 1", got)
	}
	if got := parser.Stats.GetPacketsMalformed(); got != 0 {
		t.Errorf("PacketsMalformed = %d, want 0", got)
	}
}

// TestParsePacket_ValidUnknownCommand_Processed verifies that a well-formed
// packet with an unknown command type (hits the default branch) is counted
// as processed. commandLength=12 means dataLength=0, so no payload bytes
// are expected.
func TestParsePacket_ValidUnknownCommand_Processed(t *testing.T) {
	handler := &mockHandler{}
	parser := NewParser(handler)
	defer parser.Close()

	// Header (commandCount=1) + command header with commandType=99 (unknown,
	// hits default branch) and commandLength=12 (dataLength=0).
	payload := append(photonHeader(0, 1), commandHeader(99, uint32(CommandHeaderLength))...)

	if err := parser.ParsePacket(payload); err != nil {
		t.Fatalf("ParsePacket returned error: %v", err)
	}

	if got := parser.Stats.GetPacketsProcessed(); got != 1 {
		t.Errorf("PacketsProcessed = %d, want 1", got)
	}
	if got := parser.Stats.GetPacketsMalformed(); got != 0 {
		t.Errorf("PacketsMalformed = %d, want 0", got)
	}
}

// TestParsePacket_Disconnect_Processed verifies that a well-formed packet
// carrying a Disconnect command is counted as processed. This guards the
// IncrPacketsProcessed() call before the early return in the Disconnect
// case (parser.go:220): if that line is accidentally removed, the processed
// counter would silently under-count disconnect packets.
func TestParsePacket_Disconnect_Processed(t *testing.T) {
	handler := &mockHandler{}
	parser := NewParser(handler)
	defer parser.Close()

	// Header (commandCount=1) + command header with commandType=Disconnect
	// and commandLength=12 (dataLength=0).
	payload := append(photonHeader(0, 1), commandHeader(CommandTypeDisconnect, uint32(CommandHeaderLength))...)

	if err := parser.ParsePacket(payload); err != nil {
		t.Fatalf("ParsePacket returned error: %v", err)
	}

	if got := parser.Stats.GetPacketsProcessed(); got != 1 {
		t.Errorf("PacketsProcessed = %d, want 1 (Disconnect must be counted as processed)", got)
	}
	if got := parser.Stats.GetPacketsMalformed(); got != 0 {
		t.Errorf("PacketsMalformed = %d, want 0", got)
	}
}
