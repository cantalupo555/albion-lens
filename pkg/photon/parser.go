// Package photon implements the Photon Engine network protocol parser.
// Albion Online uses Photon Engine for all game networking.
package photon

import (
	"encoding/binary"
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
	if len(payload) < PhotonHeaderLength {
		return fmt.Errorf("packet too short: %d bytes", len(payload))
	}

	offset := 0

	// Read Photon header
	// peerId := binary.BigEndian.Uint16(payload[offset:])
	offset += 2

	flags := payload[offset]
	offset++

	commandCount := payload[offset]
	offset++

	// timestamp := binary.BigEndian.Uint32(payload[offset:])
	offset += 4

	// challenge := binary.BigEndian.Uint32(payload[offset:])
	offset += 4

	// Check flags
	isEncrypted := flags == 1
	isCrcEnabled := flags == 0xCC

	if isEncrypted {
		if p.debug {
			fmt.Println("  [Photon] Skipping encrypted packet")
		}
		return nil
	}

	if isCrcEnabled {
		// Skip CRC field and validate (for now, just skip)
		// In a full implementation, we'd validate the CRC
		offset += 4
		if p.debug {
			fmt.Println("  [Photon] Packet has CRC enabled (skipping validation)")
		}
	}

	// Process each command
	for i := 0; i < int(commandCount) && offset < len(payload); i++ {
		if offset+CommandHeaderLength > len(payload) {
			break
		}

		commandType := payload[offset]
		offset++

		// channelId := payload[offset]
		offset++

		// commandFlags := payload[offset]
		offset++

		// reserved
		offset++

		commandLength := int(binary.BigEndian.Uint32(payload[offset:]))
		offset += 4

		sequenceNumber := int32(binary.BigEndian.Uint32(payload[offset:]))
		offset += 4

		commandLength -= CommandHeaderLength

		if offset+commandLength > len(payload) {
			if p.debug {
				fmt.Printf("  [Photon] Command length exceeds packet: %d + %d > %d\n", offset, commandLength, len(payload))
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
			offset += 4
			commandLength -= 4
			p.handleSendReliable(payload[offset:offset+commandLength], commandLength)
			offset += commandLength

		case CommandTypeSendReliable:
			p.handleSendReliable(payload[offset:offset+commandLength], commandLength)
			offset += commandLength

		case CommandTypeSendFragment:
			p.handleSendFragment(payload[offset:offset+commandLength], commandLength, sequenceNumber)
			offset += commandLength

		default:
			offset += commandLength
		}
	}

	return nil
}

// handleSendReliable processes a reliable command payload
func (p *Parser) handleSendReliable(data []byte, length int) {
	if length < 2 {
		return
	}

	offset := 0

	// Skip signal byte
	signalByte := data[offset]
	offset++

	// Check signal byte
	if signalByte != 243 && signalByte != 253 {
		return
	}

	messageType := data[offset]
	offset++

	// Check if encrypted
	if messageType > 128 {
		if p.debug {
			fmt.Println("  [Photon] Skipping encrypted message")
		}
		return
	}

	remaining := data[offset:]

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
func (p *Parser) handleSendFragment(data []byte, length int, sequenceNumber int32) {
	if length < FragmentHeaderLength {
		return
	}

	offset := 0

	startSequenceNumber := int32(binary.BigEndian.Uint32(data[offset:]))
	offset += 4

	// fragmentCount := binary.BigEndian.Uint32(data[offset:])
	offset += 4

	// fragmentNumber := binary.BigEndian.Uint32(data[offset:])
	offset += 4

	totalLength := int32(binary.BigEndian.Uint32(data[offset:]))
	offset += 4

	fragmentOffset := int(binary.BigEndian.Uint32(data[offset:]))
	offset += 4

	fragmentLength := length - FragmentHeaderLength

	// Validate we have enough data
	if offset+fragmentLength > len(data) {
		if p.debug {
			fmt.Printf("  [Photon] Fragment data exceeds buffer: offset=%d, fragLen=%d, dataLen=%d\n", offset, fragmentLength, len(data))
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
	if fragmentOffset >= 0 && fragmentOffset+fragmentLength <= int(totalLength) {
		copy(frag.payload[fragmentOffset:], data[offset:offset+fragmentLength])
		frag.bytesWritten += fragmentLength
	}

	// Check if complete
	if frag.bytesWritten >= int(frag.totalLength) {
		delete(p.pendingFragments, startSequenceNumber)
		p.fragmentsMu.Unlock()

		if p.debug {
			fmt.Printf("  [Photon] Reassembled fragmented packet: %d bytes\n", frag.totalLength)
		}

		p.handleSendReliable(frag.payload, int(frag.totalLength))
	} else {
		p.fragmentsMu.Unlock()
	}
}

// decodeOperationRequest decodes an operation request
func (p *Parser) decodeOperationRequest(data []byte) {
	if len(data) < 1 {
		return
	}

	operationCode := data[0]
	parameters := decodeParameterTable(data[1:])

	if p.debug {
		fmt.Printf("  [Photon] Request: code=%d, params=%d\n", operationCode, len(parameters))
	}

	if p.handler != nil {
		p.handler.OnRequest(operationCode, parameters)
	}
}

// decodeOperationResponse decodes an operation response
func (p *Parser) decodeOperationResponse(data []byte) {
	if len(data) < 4 {
		return
	}

	offset := 0

	operationCode := data[offset]
	offset++

	returnCode := int16(binary.BigEndian.Uint16(data[offset:]))
	offset += 2

	// Read debug message (optional)
	debugMessage := ""
	if offset < len(data) {
		paramType := data[offset]
		offset++
		if paramType != 0 && paramType != 42 { // Not null
			// Read string
			msg, newOffset := readString(data, offset-1)
			debugMessage = msg
			offset = newOffset
		}
	}

	parameters := decodeParameterTable(data[offset:])

	if p.debug {
		fmt.Printf("  [Photon] Response: code=%d, return=%d, params=%d\n", operationCode, returnCode, len(parameters))
	}

	if p.handler != nil {
		p.handler.OnResponse(operationCode, returnCode, debugMessage, parameters)
	}
}

// decodeEventData decodes an event
func (p *Parser) decodeEventData(data []byte) {
	if len(data) < 1 {
		return
	}

	eventCode := data[0]
	parameters := decodeParameterTable(data[1:])

	if p.debug {
		fmt.Printf("  [Photon] Event: code=%d, params=%d\n", eventCode, len(parameters))
	}

	if p.handler != nil {
		p.handler.OnEvent(eventCode, parameters)
	}
}

// readString reads a Protocol16 string from the buffer
func readString(data []byte, offset int) (string, int) {
	if offset+3 > len(data) {
		return "", offset
	}

	// Skip type byte
	offset++

	length := int(binary.BigEndian.Uint16(data[offset:]))
	offset += 2

	if offset+length > len(data) {
		return "", offset
	}

	str := string(data[offset : offset+length])
	return str, offset + length
}
