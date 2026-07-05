// Package backend provides a unified service layer for Albion Online packet capture and event processing.
// It serves as the backend for multiple frontends (TUI, Wails, Web API).
package backend

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cantalupo555/albion-lens/internal/serverdetect"
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
	detectorOpts    []serverdetect.Option

	// Internal components
	handler  *handlers.AlbionHandler
	parser   *photon.Parser
	capture  *capture.Capture
	detector *serverdetect.Detector
	stopChan chan struct{}
	wg       sync.WaitGroup // Tracks statsUpdater goroutine

	// serverChangesDropped counts region transitions that could not be forwarded
	// to serverChangedCh because its buffer was full. Exposed for diagnostics
	// (the buffer is small and region changes are rare, so this stays at zero
	// in normal use; a non-zero value signals the frontend is not draining).
	serverChangesDropped atomic.Int64

	// Public channels (read-only for frontends)
	Events       <-chan GameEvent
	Stats        <-chan *photon.Stats
	OnlineStatus <-chan bool

	// Internal writable channels
	eventsChan       chan GameEvent
	statsChan        chan *photon.Stats
	onlineStatusChan chan bool
	serverChangedCh  chan serverdetect.ChangeEvent

	// serverChanged exposes region transitions to frontends (e.g. so the TUI
	// can re-resolve persistence paths and reload per-region state). It is
	// buffered because region changes are rare and the capture goroutine that
	// produces them must not block.
	serverChanged <-chan serverdetect.ChangeEvent

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
	// serverChangedCh is intentionally small: region transitions are rare
	// (typically one per login). The buffer absorbs the burst between the
	// capture goroutine producing the event and the frontend draining it.
	s.serverChangedCh = make(chan serverdetect.ChangeEvent, 4)
	s.stopChan = make(chan struct{})

	// Expose read-only channels
	s.Events = s.eventsChan
	s.Stats = s.statsChan
	s.OnlineStatus = s.onlineStatusChan
	s.serverChanged = s.serverChangedCh

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

	// Server region detector: identifies Americas/Europe/Asia from the Photon
	// server's IP. The detector's onChange forwards transitions to
	// serverChangedCh for frontends to act on (e.g. switching persistence
	// directories). Created here so it lives for the whole capture session.
	detectorOpts := append([]serverdetect.Option{}, s.detectorOpts...)
	detectorOpts = append(detectorOpts, serverdetect.WithOnChange(func(e serverdetect.ChangeEvent) {
		select {
		case s.serverChangedCh <- e:
		default:
			// Buffer full: drop rather than block the capture hot path.
			// Region changes are rare; the periodic save + save-on-exit
			// remain as safety nets so a dropped transition is recoverable.
			s.serverChangesDropped.Add(1)
		}
	}))
	s.detector = serverdetect.NewDetector(detectorOpts...)

	// Create capture. NewCaptureWithFilter falls back to the default Albion
	// port set when s.bpfFilter is empty, so WithBPFFilter("") and omitting
	// the option behave identically.
	s.capture = capture.NewCaptureWithFilter(s.handlePacket, s.bpfFilter)

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
	s.wg.Add(1)
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

// SetDetectorForTest attaches a pre-configured detector, bypassing Start().
// Intended only for tests that need a known server region without live packet
// capture. The detector's onChange callback is NOT wired to serverChangedCh,
// so CurrentServer() works but ServerChanged() will not emit events.
func (s *Service) SetDetectorForTest(d *serverdetect.Detector) {
	s.detector = d
}

// Stop stops the service and cleans up resources. It blocks until the
// statsUpdater goroutine has fully exited, then closes all public channels.
// Safe to call multiple times (subsequent calls are no-ops).
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

	// Wait for statsUpdater to exit before closing channels,
	// preventing send-on-closed-channel panics during shutdown.
	s.wg.Wait()

	// Close channels
	close(s.eventsChan)
	close(s.statsChan)
	close(s.onlineStatusChan)
	close(s.serverChangedCh)
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
	defer s.wg.Done()

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

// TotalFame returns the cumulative fame gained across all launches.
func (s *Service) TotalFame() int64 {
	if s.handler == nil {
		return 0
	}
	return s.handler.GetTotalFame()
}

// TotalSilver returns the cumulative silver gained across all launches.
func (s *Service) TotalSilver() int64 {
	if s.handler == nil {
		return 0
	}
	return s.handler.GetTotalSilver()
}

// TotalRespec returns the cumulative combat fame credits gained across all launches.
func (s *Service) TotalRespec() int64 {
	if s.handler == nil {
		return 0
	}
	return s.handler.GetTotalRespec()
}

// TotalRespecSilver returns the cumulative silver spent on auto-respec across all launches.
func (s *Service) TotalRespecSilver() int64 {
	if s.handler == nil {
		return 0
	}
	return s.handler.GetTotalRespecSilver()
}

// TotalKills returns the cumulative number of kills across all launches.
func (s *Service) TotalKills() int {
	if s.handler == nil {
		return 0
	}
	return s.handler.GetTotalKills()
}

// TotalDeaths returns the cumulative number of deaths across all launches.
func (s *Service) TotalDeaths() int {
	if s.handler == nil {
		return 0
	}
	return s.handler.GetTotalDeaths()
}

// TotalLoot returns the cumulative number of loot items across all launches.
func (s *Service) TotalLoot() int {
	if s.handler == nil {
		return 0
	}
	return s.handler.GetTotalLoot()
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

// CurrentServer returns the currently detected Albion server region, or
// ServerLocationUnknown if no region has been confirmed yet (the detector
// requires a stable Photon server IP for 5 seconds before promoting).
func (s *Service) CurrentServer() serverdetect.ServerLocation {
	if s.detector == nil {
		return serverdetect.ServerLocationUnknown
	}
	return s.detector.Current().Location
}

// ServerChanged returns a channel that receives an event each time the
// detected Albion server region transitions (Unknown→region, region→Unknown,
// or region→region). The channel is closed when the service stops.
//
// Frontends use this to re-resolve per-region persistence directories and
// reload state for the newly active region.
func (s *Service) ServerChanged() <-chan serverdetect.ChangeEvent {
	return s.serverChanged
}

// ServerChangesDropped returns the count of region transitions that were
// discarded because the ServerChanged buffer was full. Non-zero in normal use
// means the frontend is not draining the channel fast enough.
func (s *Service) ServerChangesDropped() int64 {
	return s.serverChangesDropped.Load()
}

// handlePacket is the per-packet callback wired into the capture layer. It
// feeds both packet endpoints to the region detector (for incoming packets the
// server is the source IP, for outgoing it is the destination; the detector
// ignores non-matching local IPs) and then forwards the payload to the Photon
// parser.
//
// Extracted as a method so the capture→detector→parser wiring is unit-testable
// without a real capture device.
func (s *Service) handlePacket(payload []byte, srcIP, dstIP net.IP, srcPort, dstPort uint16) {
	if s.detector != nil {
		s.detector.DetectFromIP(srcIP)
		s.detector.DetectFromIP(dstIP)
	}
	if s.parser != nil {
		_ = s.parser.ParsePacket(payload)
	}
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
