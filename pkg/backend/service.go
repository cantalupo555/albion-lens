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
	defaultEventBufferSize = 250
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
	s.handler.SetEventCallback(func(eventType, message string, data interface{}) {
		event := GameEvent{
			Type:      EventType(eventType),
			Message:   message,
			Timestamp: time.Now(),
			Data:      data,
		}

		// Update peak buffer usage stats before sending
		if s.parser != nil && s.parser.Stats != nil {
			s.parser.Stats.UpdateBufferPeak(len(s.eventsChan))
		}

		s.emitEvent(event)
	})

	// Load item database (errors are non-fatal)
	_ = s.loadItemDatabase()

	// Load mob database for dungeon tier classification (errors are non-fatal)
	_ = s.loadMobDatabase()

	// Create parser
	s.parser = photon.NewParser(s.handler)
	s.parser.Stats.BufferCapacity = cap(s.eventsChan) // Set once at startup
	// Note: Parser debug is not enabled because it uses fmt.Printf which interferes with TUI

	// Create capture. NewCaptureWithFilter falls back to the default Albion
	// port set when s.bpfFilter is empty, so WithBPFFilter("") and omitting
	// the option behave identically.
	packetHandler := func(payload []byte, srcIP, dstIP net.IP, srcPort, dstPort uint16) {
		_ = s.parser.ParsePacket(payload)
	}
	s.capture = capture.NewCaptureWithFilter(packetHandler, s.bpfFilter)

	// Set online/offline callback
	s.capture.OnlineCallback = func(online bool) {
		select {
		case s.onlineStatusChan <- online:
		default:
			// Status updates are idempotent, drop is safe
		}

		// Also send as info event
		msg := "Waiting for Albion Online traffic..."
		if online {
			msg = "Albion Online detected! Capturing packets..."
		}
		s.emitEvent(GameEvent{
			Type:      EventTypeInfo,
			Message:   msg,
			Timestamp: time.Now(),
		})
	}

	// Set device error callback: surfaces per-device open failures (partial
	// failures during Start) as warning events so the user knows some capture
	// interfaces were skipped. Fatal "no device opened" errors are returned
	// from Start() instead and handled by the caller.
	s.capture.DeviceErrorCallback = func(deviceName string, err error) {
		s.emitEvent(GameEvent{
			Type:      EventTypeWarning,
			Message:   fmt.Sprintf("Could not capture on %s: %v", deviceName, err),
			Timestamp: time.Now(),
		})
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

// emitEvent sends a game event to the events channel without blocking. If the
// channel is full the event is dropped and the parser's dropped-events counter
// is incremented (when available). Centralized so all event sources route
// through a single place.
func (s *Service) emitEvent(event GameEvent) {
	select {
	case s.eventsChan <- event:
	default:
		if s.parser != nil && s.parser.Stats != nil {
			s.parser.Stats.IncrEventsDropped()
		}
	}
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
				// Snapshot buffer metrics (Peak usage in last interval)
				s.parser.Stats.SnapshotBufferPeak()

				select {
				case s.statsChan <- s.parser.Stats:
				default:
					// Stats channel full - this is less critical than events
					// We don't increment EventsDropped for stats updates
				}
			}
		}
	}
}

// aoBinDumpsPaths returns the common fallback paths for the ao-bin-dumps
// directory, used when no explicit --items path is provided.
func aoBinDumpsPaths() []string {
	return []string{
		"../ao-bin-dumps",
		"../../ao-bin-dumps",
		filepath.Join(os.Getenv("HOME"), "Documents/albion/ao-bin-dumps"),
	}
}

// loadItemDatabase attempts to load the item database.
func (s *Service) loadItemDatabase() error {
	if s.itemDBPath != "" {
		return s.handler.LoadItemDatabase(s.itemDBPath)
	}

	for _, path := range aoBinDumpsPaths() {
		if _, err := os.Stat(filepath.Join(path, "items.json")); err == nil {
			return s.handler.LoadItemDatabase(path)
		}
	}

	return nil
}

// loadMobDatabase attempts to load the mob database for dungeon tier classification.
func (s *Service) loadMobDatabase() error {
	for _, path := range aoBinDumpsPaths() {
		if _, err := os.Stat(filepath.Join(path, "mobs.json")); err == nil {
			return s.handler.LoadMobDatabase(path)
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

// SessionRespec returns the total combat fame credits gained in this session.
func (s *Service) SessionRespec() int64 {
	if s.handler == nil {
		return 0
	}
	return s.handler.GetSessionRespec()
}

// SessionRespecSilver returns the total silver spent on auto-respec this session.
func (s *Service) SessionRespecSilver() int64 {
	if s.handler == nil {
		return 0
	}
	return s.handler.GetSessionRespecSilver()
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

// LocalPlayerName returns the auto-detected local player name, or "" if the
// OpJoin response has not been captured yet.
func (s *Service) LocalPlayerName() string {
	if s.handler == nil {
		return ""
	}
	return s.handler.GetLocalPlayer()
}

// CurrentZone returns the current zone identity, or a zero-value ZoneInfo if
// the ChangeCluster response has not been captured yet.
func (s *Service) CurrentZone() handlers.ZoneInfo {
	if s.handler == nil {
		return handlers.ZoneInfo{}
	}
	return s.handler.GetCurrentZone()
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

// SetDebug enables or disables debug mode at runtime.
// This propagates to the handler only (not parser, which uses fmt.Printf).
func (s *Service) SetDebug(debug bool) {
	s.mu.Lock()
	s.debug = debug
	s.mu.Unlock()

	if s.handler != nil {
		s.handler.SetDebug(debug)
	}
	// Note: We don't propagate to parser because it uses fmt.Printf
	// which interferes with the TUI. Handler sends events via callback instead.
}

// IsDebug returns whether debug mode is enabled.
func (s *Service) IsDebug() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.debug
}
