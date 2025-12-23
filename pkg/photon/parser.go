// Package photon implements the Photon Engine network protocol parser.
// Albion Online uses Photon Engine for all game networking.
package photon

import (
	"fmt"
	"sync"
	"time"
)

const (
	// Header sizes
	PhotonHeaderLength        = 12
	CommandHeaderLength       = 12
	FragmentHeaderLength      = 20

	// Command types
	CommandTypeDisconnect     = 4
	CommandTypeSendReliable   = 6
	CommandTypeSendUnreliable = 7
	CommandTypeSendFragment   = 8

	// Message types
	MessageTypeOperationRequest  = 2
	MessageTypeOperationResponse = 3
	MessageTypeEventData         = 4
	MessageTypeInternalRequest   = 6
	MessageTypeInternalResponse  = 7

	// Fragment cleanup settings
	FragmentTTL             = 30 * time.Second // Fragments expire after 30s
	FragmentCleanupInterval = 10 * time.Second // Cleanup runs every 10s
)

// PhotonHandler is called when a Photon message is decoded
type PhotonHandler interface {
	OnRequest(operationCode byte, parameters map[byte]interface{})
	OnResponse(operationCode byte, returnCode int16, debugMessage string, parameters map[byte]interface{})
	OnEvent(eventCode byte, parameters map[byte]interface{})
}

// Parser parses Photon protocol packets
type Parser struct {
	handler          PhotonHandler
	pendingFragments map[int32]*fragmentedPacket
	fragmentsMu      sync.RWMutex  // Protects pendingFragments
	debug            bool
	stopCleanup      chan struct{} // Signal to stop cleanup goroutine
	Stats            *Stats        // Parser statistics
}

// fragmentedPacket holds data for reassembling fragmented packets
type fragmentedPacket struct {
	totalLength  int32
	payload      []byte
	bytesWritten int
	createdAt    time.Time // When the fragment was first received
}

// NewParser creates a new Photon parser
func NewParser(handler PhotonHandler) *Parser {
	p := &Parser{
		handler:          handler,
		pendingFragments: make(map[int32]*fragmentedPacket),
		debug:            false,
		stopCleanup:      make(chan struct{}),
		Stats:            NewStats(),
	}

	// Start background cleanup goroutine
	go p.cleanupLoop()

	return p
}

// SetDebug enables or disables debug output
func (p *Parser) SetDebug(debug bool) {
	p.debug = debug
}

// Close stops the cleanup goroutine and releases resources.
// Should be called when the parser is no longer needed.
func (p *Parser) Close() {
	close(p.stopCleanup)
}

// cleanupLoop periodically removes expired fragments
func (p *Parser) cleanupLoop() {
	ticker := time.NewTicker(FragmentCleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.cleanupExpiredFragments()
		case <-p.stopCleanup:
			return
		}
	}
}

// cleanupExpiredFragments removes fragments older than FragmentTTL
func (p *Parser) cleanupExpiredFragments() {
	p.fragmentsMu.Lock()
	defer p.fragmentsMu.Unlock()

	now := time.Now()
	expired := 0

	for seqNum, frag := range p.pendingFragments {
		if now.Sub(frag.createdAt) > FragmentTTL {
			delete(p.pendingFragments, seqNum)
			expired++
			p.Stats.IncrFragmentsExpired()
		}
	}

	if p.debug && expired > 0 {
		fmt.Printf("  [Photon] Cleaned up %d expired fragments\n", expired)
	}
}

// PendingFragmentsCount returns the number of pending fragments (for debugging/stats)
func (p *Parser) PendingFragmentsCount() int {
	p.fragmentsMu.RLock()
	defer p.fragmentsMu.RUnlock()
	return len(p.pendingFragments)
}

// ParsePacket parses a raw UDP payload as a Photon packet
func (p *Parser) ParsePacket(payload []byte) error {
	p.Stats.IncrPacketsReceived()
	p.Stats.AddBytesReceived(uint64(len(payload)))
	p.Stats.LastPacketTime = time.Now()

	if len(payload) < PhotonHeaderLength {
		p.Stats.IncrPacketsMalformed()
		return fmt.Errorf("packet too short: %d bytes", len(payload))
	}

	r := NewBufferReader(payload)

	// Read Photon header
	_ = r.Skip(2) // peerId (ignored)

	flags, _ := r.ReadByte()
	commandCount, _ := r.ReadByte()

	_ = r.Skip(4) // timestamp (ignored)
	_ = r.Skip(4) // challenge (ignored)

	// Check flags
	isEncrypted := flags == 1
	isCrcEnabled := flags == 0xCC

	if isEncrypted {
		p.Stats.IncrPacketsEncrypted()
		if p.debug {
			fmt.Println("  [Photon] Skipping encrypted packet")
		}
		return nil
	}

	if isCrcEnabled {
		p.Stats.IncrPacketsWithCRC()
		// Skip CRC field and validate (for now, just skip)
		// In a full implementation, we'd validate the CRC
		_ = r.Skip(4)
		if p.debug {
			fmt.Println("  [Photon] Packet has CRC enabled (skipping validation)")
		}
	}

	// Process each command
	for i := 0; i < int(commandCount) && !r.IsEmpty(); i++ {
		if r.Remaining() < CommandHeaderLength {
			break
		}

		commandType, _ := r.ReadByte()
		_ = r.Skip(1) // channelId (ignored)
		_ = r.Skip(1) // commandFlags (ignored)
		_ = r.Skip(1) // reserved

		commandLength, _ := r.ReadUint32()
		sequenceNumber, _ := r.ReadInt32()

		dataLength := int(commandLength) - CommandHeaderLength

		if r.Remaining() < dataLength {
			if p.debug {
				fmt.Printf("  [Photon] Command length exceeds packet: remaining=%d, need=%d\n", r.Remaining(), dataLength)
			}
			break
		}

		switch commandType {
		case CommandTypeDisconnect:
			if p.debug {
				fmt.Println("  [Photon] Disconnect command")
			}
			return nil

		case CommandTypeSendUnreliable:
			// Skip 4 bytes for unreliable sequence
			_ = r.Skip(4)
			dataLength -= 4
			commandData, _ := r.ReadBytesNoCopy(dataLength)
			p.handleSendReliable(commandData)

		case CommandTypeSendReliable:
			commandData, _ := r.ReadBytesNoCopy(dataLength)
			p.handleSendReliable(commandData)

		case CommandTypeSendFragment:
			commandData, _ := r.ReadBytesNoCopy(dataLength)
			p.handleSendFragment(commandData, sequenceNumber)

		default:
			_ = r.Skip(dataLength)
		}
	}

	p.Stats.IncrPacketsProcessed()

	return nil
}

// handleSendReliable processes a reliable command payload
func (p *Parser) handleSendReliable(data []byte) {
	if len(data) < 2 {
		return
	}

	r := NewBufferReader(data)

	// Read signal byte
	signalByte, _ := r.ReadByte()

	// Check signal byte
	if signalByte != 243 && signalByte != 253 {
		return
	}

	messageType, _ := r.ReadByte()

	// Check if encrypted
	if messageType > 128 {
		if p.debug {
			fmt.Println("  [Photon] Skipping encrypted message")
		}
		return
	}

	// Get remaining data as a new BufferReader for decoding
	remaining := NewBufferReader(r.RemainingBytes())

	switch messageType {
	case MessageTypeOperationRequest, MessageTypeInternalRequest:
		p.decodeOperationRequest(remaining)

	case MessageTypeOperationResponse, MessageTypeInternalResponse:
		p.decodeOperationResponse(remaining)

	case MessageTypeEventData:
		p.decodeEventData(remaining)
	}
}

// handleSendFragment processes a fragmented packet
func (p *Parser) handleSendFragment(data []byte, sequenceNumber int32) {
	if len(data) < FragmentHeaderLength {
		return
	}

	p.Stats.IncrFragmentsReceived()

	r := NewBufferReader(data)

	startSequenceNumber, _ := r.ReadInt32()
	_ = r.Skip(4) // fragmentCount (ignored)
	_ = r.Skip(4) // fragmentNumber (ignored)
	totalLength, _ := r.ReadInt32()
	fragmentOffset, _ := r.ReadUint32()

	fragmentLength := len(data) - FragmentHeaderLength

	// Validate we have enough data
	if r.Remaining() < fragmentLength {
		if p.debug {
			fmt.Printf("  [Photon] Fragment data exceeds buffer: remaining=%d, fragLen=%d\n", r.Remaining(), fragmentLength)
		}
		return
	}

	// Lock for concurrent access to pendingFragments
	p.fragmentsMu.Lock()

	// Get or create pending fragment
	frag, exists := p.pendingFragments[startSequenceNumber]
	if !exists {
		frag = &fragmentedPacket{
			totalLength: totalLength,
			payload:     make([]byte, totalLength),
			createdAt:   time.Now(),
		}
		p.pendingFragments[startSequenceNumber] = frag
	}

	// Copy fragment data (with bounds check for destination)
	fragOff := int(fragmentOffset)
	if fragOff >= 0 && fragOff+fragmentLength <= int(totalLength) {
		fragmentData, _ := r.ReadBytesNoCopy(fragmentLength)
		copy(frag.payload[fragOff:], fragmentData)
		frag.bytesWritten += fragmentLength
	}

	// Check if complete
	if frag.bytesWritten >= int(frag.totalLength) {
		delete(p.pendingFragments, startSequenceNumber)
		p.fragmentsMu.Unlock()

		p.Stats.IncrFragmentsCompleted()

		if p.debug {
			fmt.Printf("  [Photon] Reassembled fragmented packet: %d bytes\n", frag.totalLength)
		}

		p.handleSendReliable(frag.payload)
	} else {
		p.fragmentsMu.Unlock()
	}
}

// decodeOperationRequest decodes an operation request
func (p *Parser) decodeOperationRequest(r *BufferReader) {
	if r.Remaining() < 1 {
		return
	}

	operationCode, _ := r.ReadByte()
	parameters := decodeParameterTable(r)

	p.Stats.IncrRequestsDecoded()

	if p.debug {
		fmt.Printf("  [Photon] Request: code=%d, params=%d\n", operationCode, len(parameters))
	}

	if p.handler != nil {
		p.handler.OnRequest(operationCode, parameters)
	}
}

// decodeOperationResponse decodes an operation response
func (p *Parser) decodeOperationResponse(r *BufferReader) {
	if r.Remaining() < 4 {
		return
	}

	operationCode, _ := r.ReadByte()
	returnCode, _ := r.ReadInt16()

	// Read debug message (optional)
	debugMessage := ""
	if !r.IsEmpty() {
		paramType, _ := r.PeekByte()
		if paramType != 0 && paramType != TypeNull {
			// Read type byte
			_, _ = r.ReadByte()
			// Read string value
			if msg, err := r.ReadString(); err == nil {
				debugMessage = msg
			}
		} else {
			// Skip null type byte
			_, _ = r.ReadByte()
		}
	}

	parameters := decodeParameterTable(r)

	p.Stats.IncrResponsesDecoded()

	if p.debug {
		fmt.Printf("  [Photon] Response: code=%d, return=%d, params=%d\n", operationCode, returnCode, len(parameters))
	}

	if p.handler != nil {
		p.handler.OnResponse(operationCode, returnCode, debugMessage, parameters)
	}
}

// decodeEventData decodes an event
func (p *Parser) decodeEventData(r *BufferReader) {
	if r.Remaining() < 1 {
		return
	}

	eventCode, _ := r.ReadByte()
	parameters := decodeParameterTable(r)

	p.Stats.IncrEventsDecoded()

	if p.debug {
		fmt.Printf("  [Photon] Event: code=%d, params=%d\n", eventCode, len(parameters))
	}

	if p.handler != nil {
		p.handler.OnEvent(eventCode, parameters)
	}
}
