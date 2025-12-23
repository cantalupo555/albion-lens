// Package backend provides a unified service layer for Albion Online packet capture and event processing.
// It serves as the backend for multiple frontends (TUI, Wails, Web API).
package backend

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/cantalupo555/albion-lens/pkg/capture"
	"github.com/cantalupo555/albion-lens/pkg/handlers"
	"github.com/cantalupo555/albion-lens/pkg/photon"
)

const (
	defaultEventBufferSize = 100
	defaultStatsBufferSize = 10
)

// Service encapsulates the Albion Online packet capture and event processing backend.
// It provides channels for frontend communication and can be used by TUI, Wails, or Web API.
type Service struct {
	// Configuration
	device          string
	debug           bool
	discovery       bool
	itemDBPath      string
	bpfFilter       string
	eventBufferSize int
	statsBufferSize int

	// Internal components
	handler  *handlers.AlbionHandler
	parser   *photon.Parser
	capture  *capture.Capture
	stopChan chan struct{}

	// Public channels (read-only for frontends)
	Events       <-chan GameEvent
	Stats        <-chan *photon.Stats
	OnlineStatus <-chan bool

	// Internal writable channels
	eventsChan       chan GameEvent
	statsChan        chan *photon.Stats
	onlineStatusChan chan bool

	// State
	running bool
	mu      sync.RWMutex
}

// New creates a new Service with the given options.
func New(opts ...Option) *Service {
	s := &Service{
		eventBufferSize: defaultEventBufferSize,
		statsBufferSize: defaultStatsBufferSize,
	}

	// Apply options
	for _, opt := range opts {
		opt(s)
	}

	// Create channels
	s.eventsChan = make(chan GameEvent, s.eventBufferSize)
	s.statsChan = make(chan *photon.Stats, s.statsBufferSize)
	s.onlineStatusChan = make(chan bool, 1)
	s.stopChan = make(chan struct{})

	// Expose read-only channels
	s.Events = s.eventsChan
	s.Stats = s.statsChan
	s.OnlineStatus = s.onlineStatusChan

	return s
}

// Start initializes and starts the packet capture and event processing.
// Returns an error if capture fails to start.
func (s *Service) Start() error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("service already running")
	}
	s.running = true
	s.mu.Unlock()

	// Create handler
	s.handler = handlers.NewAlbionHandler()
	s.handler.SetDebug(s.debug)
	s.handler.SetDiscoveryMode(s.discovery)

	// Set event callback to send events to channel
	s.handler.SetEventCallback(func(eventType, message string) {
		event := GameEvent{
			Type:      EventType(eventType),
			Message:   message,
			Timestamp: time.Now(),
		}
		select {
		case s.eventsChan <- event:
		default:
			// Channel full, drop event
		}
	})

	// Load item database
	if err := s.loadItemDatabase(); err != nil && s.debug {
		fmt.Printf("Warning: Could not load item database: %v\n", err)
	}

	// Create parser
	s.parser = photon.NewParser(s.handler)
	s.parser.SetDebug(s.debug)

	// Create capture
	s.capture = capture.NewCapture(func(payload []byte, srcIP, dstIP net.IP, srcPort, dstPort uint16) {
		_ = s.parser.ParsePacket(payload)
	})

	// Set online/offline callback
	s.capture.OnlineCallback = func(online bool) {
		select {
		case s.onlineStatusChan <- online:
		default:
		}

		// Also send as info event
		msg := "Waiting for Albion Online traffic..."
		if online {
			msg = "Albion Online detected! Capturing packets..."
		}
		select {
		case s.eventsChan <- GameEvent{
			Type:      EventTypeInfo,
			Message:   msg,
			Timestamp: time.Now(),
		}:
		default:
		}
	}

	// Start stats updater
	go s.statsUpdater()

	// Start capture
	var err error
	if s.device != "" {
		err = s.capture.StartOnDevice(s.device)
	} else {
		err = s.capture.Start()
	}

	if err != nil {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
		return fmt.Errorf("failed to start capture: %w", err)
	}

	return nil
}

// Stop stops the service and cleans up resources.
func (s *Service) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	s.mu.Unlock()

	// Signal stop
	close(s.stopChan)

	// Stop capture
	if s.capture != nil {
		s.capture.Stop()
	}

	// Close parser
	if s.parser != nil {
		s.parser.Close()
	}

	// Close channels
	close(s.eventsChan)
	close(s.statsChan)
	close(s.onlineStatusChan)
}

// statsUpdater periodically sends stats to the channel.
func (s *Service) statsUpdater() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			if s.parser != nil {
				select {
				case s.statsChan <- s.parser.Stats:
				default:
				}
			}
		}
	}
}

// loadItemDatabase attempts to load the item database.
func (s *Service) loadItemDatabase() error {
	if s.itemDBPath != "" {
		return s.handler.LoadItemDatabase(s.itemDBPath)
	}

	// Try auto-detection
	commonPaths := []string{
		"../ao-bin-dumps",
		"../../ao-bin-dumps",
		filepath.Join(os.Getenv("HOME"), "Documents/albion/ao-bin-dumps"),
	}

	for _, path := range commonPaths {
		if _, err := os.Stat(filepath.Join(path, "items.json")); err == nil {
			return s.handler.LoadItemDatabase(path)
		}
	}

	return nil
}

// IsRunning returns whether the service is currently running.
func (s *Service) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// IsOnline returns whether Albion Online traffic is currently being detected.
func (s *Service) IsOnline() bool {
	if s.capture == nil {
		return false
	}
	return s.capture.IsOnline()
}

// SessionFame returns the total fame gained in this session.
func (s *Service) SessionFame() int64 {
	if s.handler == nil {
		return 0
	}
	return s.handler.GetSessionFame()
}

// SessionSilver returns the total silver gained in this session.
func (s *Service) SessionSilver() int64 {
	if s.handler == nil {
		return 0
	}
	return s.handler.GetSessionSilver()
}

// SessionKills returns the number of kills in this session.
func (s *Service) SessionKills() int {
	if s.handler == nil {
		return 0
	}
	return s.handler.GetSessionKills()
}

// SessionDeaths returns the number of deaths in this session.
func (s *Service) SessionDeaths() int {
	if s.handler == nil {
		return 0
	}
	return s.handler.GetSessionDeaths()
}

// SessionLoot returns the number of loot items in this session.
func (s *Service) SessionLoot() int {
	if s.handler == nil {
		return 0
	}
	return s.handler.GetSessionLoot()
}

// ParserStats returns the current parser statistics.
func (s *Service) ParserStats() *photon.Stats {
	if s.parser == nil {
		return nil
	}
	return s.parser.Stats
}

// Handler returns the underlying AlbionHandler for advanced usage.
// This is useful for discovery mode operations.
func (s *Service) Handler() *handlers.AlbionHandler {
	return s.handler
}
