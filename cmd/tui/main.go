package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/cantalupo555/albion-lens/internal/serverdetect"
	"github.com/cantalupo555/albion-lens/internal/tui"
	"github.com/cantalupo555/albion-lens/pkg/backend"
	"github.com/cantalupo555/albion-lens/pkg/capture"
	"github.com/cantalupo555/albion-lens/pkg/handlers"
	"github.com/cantalupo555/albion-lens/pkg/photon"
)

const (
	bulkEventChannelSize = 5 // 5 batches × eventBatchSize = 250 events buffered
	statsChannelSize     = 10
	eventBatchSize       = 50
	eventFlushInterval   = 50 * time.Millisecond
	// saveInterval is how often the in-memory state is flushed to disk while
	// the TUI runs, bounding data loss on crash or kill to this window.
	saveInterval = 5 * time.Minute
)

func main() {
	// Parse command line flags
	listDevices := flag.Bool("list", false, "List available network devices")
	deviceName := flag.String("device", "", "Specific device to capture on (captures all if not specified)")
	debug := flag.Bool("debug", false, "Enable debug output")
	discover := flag.Bool("discover", false, "Enable discovery mode to log all events (dumps JSON to output/ on exit)")
	itemsPath := flag.String("items", "", "Path to ao-bin-dumps directory for item name resolution")
	flag.Parse()

	// List devices if requested
	if *listDevices {
		if err := capture.PrintDevices(); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Create backend service with options
	opts := []backend.Option{
		backend.WithDebug(*debug),
	}
	if *deviceName != "" {
		opts = append(opts, backend.WithDevice(*deviceName))
	}
	if *itemsPath != "" {
		opts = append(opts, backend.WithItemDatabasePath(*itemsPath))
	}
	if *discover {
		opts = append(opts, backend.WithDiscovery(true))
	}

	svc := backend.New(opts...)

	// Create channels for TUI communication
	bulkEventChan := make(chan tui.BulkEventMsg, bulkEventChannelSize)
	statsChan := make(chan *photon.Stats, statsChannelSize)

	// sendWarning delivers a diagnostic to the TUI event log so the user can
	// see it during operation (fmt.Printf is hidden by the TUI alt-screen).
	// When the channel is full the fallback writes to stdout, which is not
	// visible during TUI operation — this is an accepted trade-off: warnings
	// are rare and the channel buffer (250) is sized to absorb bursts.
	sendWarning := func(msgType, format string, args ...any) {
		select {
		case bulkEventChan <- tui.BulkEventMsg{{
			Type:      msgType,
			Message:   strings.TrimSpace(fmt.Sprintf(format, args...)),
			Timestamp: time.Now(),
		}}:
		default:
			fmt.Printf(format, args...)
		}
	}

	// Bridge backend events to TUI with batching
	go bridgeEvents(svc.Events, bulkEventChan, svc.ParserStats)

	// Bridge backend stats to TUI
	go func() {
		for stats := range svc.Stats {
			select {
			case statsChan <- stats:
			default:
				// Stats channel full - not critical
			}
		}
	}()

	// Start backend service
	if err := svc.Start(); err != nil {
		fmt.Printf("Error starting capture: %v\n", err)
		fmt.Println("Try running with sudo or as administrator.")
		os.Exit(1)
	}
	defer svc.Stop()

	// --- Persistence: restore state, keep it warm, and segregate it per
	// Albion server region. ---
	// A user-provided server-hint.txt overrides IP-based detection: when set,
	// persistence pins to that region for the whole session and detection
	// events are ignored. This is the escape hatch for when the server IP
	// prefixes drift (see internal/serverdetect).
	hintRegion := serverHintOverride()
	if hintRegion.IsKnown() {
		fmt.Printf("Server hint active: persistence pinned to %s; IP detection disabled.\n", hintRegion)
	}
	warn := warnFn(func(format string, args ...any) {
		sendWarning("warning", format, args...)
	})
	paths := newPersistPaths(hintRegion.DirName(), warn)
	if !paths.Enabled() {
		fmt.Printf("Warning: persistence disabled, could not resolve data directory: %v\n", paths.InitError())
	}

	// Load-on-startup. The handler is created by Start(), so this must run
	// after it. A missing file is the first-run case (handled as empty state
	// inside Load); a corrupt file leaves the in-memory state unchanged.
	if h := svc.Handler(); h != nil {
		if err := paths.Load(h); err != nil {
			sendWarning("warning", "Warning: could not load persisted state: %v\n", err)
		}
	}

	// React to server-region changes: flush the old region, switch paths, and
	// reload state for the new region. Skipped when a hint override is active
	// (the user explicitly pinned the region) or persistence is disabled.
	var watchWg sync.WaitGroup
	notify := func(msgType, format string, args ...any) {
		sendWarning(msgType, format, args...)
	}
	if hintRegion == serverdetect.ServerLocationUnknown && paths.Enabled() {
		if h := svc.Handler(); h != nil {
			watchWg.Add(1)
			go func() {
				defer watchWg.Done()
				watchServerChanges(svc.ServerChanged(), h, paths, notify)
			}()
		}
	}

	// Periodic flush while the TUI runs: a crash or kill loses at most
	// saveInterval of progress. Cancelled after p.Run returns.
	saveCtx, saveCancel := context.WithCancel(context.Background())
	defer saveCancel()
	go periodicSave(saveCtx, saveInterval, func() error {
		if dropped := svc.ServerChangesDropped(); dropped > 0 {
			notify("warning", "Warning: %d server-region change(s) dropped; persistence may not have switched to the correct region.\n", dropped)
		}
		h := svc.Handler()
		if h == nil || !paths.Enabled() {
			return nil
		}
		return paths.Save(h)
	}, notify)

	// Cluster-change save: each confirmed zone transition asks for a
	// persistence flush, rate-limited so rapid map hops coalesce into at most
	// one write per clusterSaveWindow. The save runs in its own goroutine so
	// the capture hot path that produced the transition is never blocked on
	// I/O. The periodic ticker and save-on-exit remain as safety nets.
	// saveWg tracks these goroutines so the final save-on-exit can drain them
	// first, avoiding an in-flight .tmp write racing with the exit save.
	// clusterMu pairs the shutdown gate with saveWg.Add(1) so the WaitGroup
	// cannot be re-entered after the exit drain (which would panic). The
	// mutex is held only for the gate check + Add, never during I/O.
	var clusterShutdown bool
	var clusterMu sync.Mutex
	clusterSaver := newRateLimiter(clusterSaveWindow)
	var saveWg sync.WaitGroup
	if h := svc.Handler(); h != nil {
		h.SetClusterChangeCallback(func() {
			clusterMu.Lock()
			if clusterShutdown || !paths.Enabled() || !clusterSaver.Allow(time.Now()) {
				clusterMu.Unlock()
				return
			}
			saveWg.Add(1)
			clusterMu.Unlock()
			go func() {
				defer saveWg.Done()
				hh := svc.Handler()
				if hh == nil {
					return
				}
				if err := paths.Save(hh); err != nil {
					notify("warning", "Warning: cluster-change save failed: %v\n", err)
				}
			}()
		})
	}

	// Send initial status event (as a batch)
	bulkEventChan <- tui.BulkEventMsg{
		{
			Type:      "info",
			Message:   "Waiting for Albion Online traffic...",
			Timestamp: time.Now(),
		},
	}

	// Create and run TUI
	model := tui.New(svc, bulkEventChan, statsChan)
	p := tea.NewProgram(model)

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running TUI: %v\n", err)
		os.Exit(1)
	}

	// --- Shutdown sequence ---
	// 1. Gate the cluster-change callback so no new saveWg.Add(1) can fire
	//    after we start draining (Add on a waited WaitGroup panics).
	// 2. Stop the periodic flush.
	// 3. Drain in-flight cluster-change saves.
	// 4. Final save-on-exit.
	// 5. Stop capture (closes serverChangedCh so watchServerChanges exits).
	// 6. Drain the region-change watcher.
	clusterMu.Lock()
	clusterShutdown = true
	clusterMu.Unlock()
	saveCancel()
	saveWg.Wait()
	if h := svc.Handler(); h != nil && paths.Enabled() {
		if err := paths.Save(h); err != nil {
			fmt.Printf("Warning: final save failed: %v\n", err)
		}
	}

	svc.Stop()
	watchWg.Wait()

	// In discovery mode, persist the captured event map so unmapped events
	// (e.g. the dungeon-closure notification) can be identified offline.
	if *discover {
		if handler := svc.Handler(); handler != nil {
			dumpPath := fmt.Sprintf("output/discovered-events-%s.json", time.Now().Format("20060102-150405"))
			if err := handler.SaveDiscoveredEvents(dumpPath); err != nil {
				fmt.Printf("Error saving discovered events: %v\n", err)
			} else {
				fmt.Printf("Discovered events saved to %s\n", dumpPath)
			}
		}
	}
}

// bridgeEvents reads game events from eventsCh, batches them, and forwards
// them as BulkEventMsg to bulkEventChan. When the channel is full, dropped
// events are counted via statsFn.
func bridgeEvents(
	eventsCh <-chan backend.GameEvent,
	bulkEventChan chan<- tui.BulkEventMsg,
	statsFn func() *photon.Stats,
) {
	buffer := make([]tui.EventMsg, 0, eventBatchSize)
	ticker := time.NewTicker(eventFlushInterval)
	defer ticker.Stop()

	flush := func() {
		if len(buffer) == 0 {
			return
		}
		// Create a copy of the buffer to send
		msg := make(tui.BulkEventMsg, len(buffer))
		copy(msg, buffer)

		select {
		case bulkEventChan <- msg:
			// Success
		default:
			// Channel full, drop ENTIRE batch
			if stats := statsFn(); stats != nil {
				// Increment dropped count for each event in the batch
				for i := 0; i < len(buffer); i++ {
					stats.IncrEventsDropped()
				}
			}
		}
		// Reset buffer
		buffer = buffer[:0]
	}

	for {
		select {
		case event, ok := <-eventsCh:
			if !ok {
				// Channel closed
				flush()
				return
			}

			// Add to buffer
			buffer = append(buffer, tui.EventMsg{
				Type:      string(event.Type),
				Message:   event.Message,
				Timestamp: event.Timestamp,
				Data:      event.Data,
			})

			// Flush if full
			if len(buffer) >= eventBatchSize {
				flush()
				// Reset ticker to avoid double flushing
				ticker.Reset(eventFlushInterval)
			}

		case <-ticker.C:
			flush()
		}
	}
}

// periodicSave calls save at each interval until ctx is cancelled. Used to
// bound the data-loss window on crash/kill while the TUI is running. A non-nil
// error from save is surfaced via notify (or fmt.Printf when nil) so the user
// learns mid-session that persistence has degraded.
func periodicSave(ctx context.Context, interval time.Duration, save func() error, notify func(msgType, format string, args ...any)) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := save(); err != nil {
				if notify != nil {
					notify("warning", "Warning: periodic save failed: %v\n", err)
				} else {
					fmt.Printf("Warning: periodic save failed: %v\n", err)
				}
			}
		}
	}
}

// watchServerChanges consumes region transitions from changes and re-points
// persistence at the newly active region. A transition to a known region
// triggers SwitchRegion (flush old → reset → migrate → load new); a transition
// back to Unknown (the detector lost confidence after seeing a different
// region) is ignored so paths stay on the last known region rather than
// churning to the shared root. Returns when the service stops and closes the
// changes channel.
//
// notify routes diagnostics to the TUI event log (nil falls back to fmt.Printf
// for tests). The handler is captured once (it lives for the whole capture
// session) so the function depends only on its arguments, making it
// unit-testable without a real Service.
func watchServerChanges(changes <-chan serverdetect.ChangeEvent, h *handlers.AlbionHandler, paths *persistPaths, notify func(msgType, format string, args ...any)) {
	for ev := range changes {
		region := ev.Current.Location
		if !region.IsKnown() {
			continue
		}
		dir := region.DirName()
		if h == nil {
			continue
		}
		if err := paths.SwitchRegion(dir, h); err != nil {
			if notify != nil {
				notify("warning", "Warning: region switch to %s failed: %v\n", dir, err)
			} else {
				fmt.Printf("Warning: region switch to %s failed: %v\n", dir, err)
			}
			continue
		}
		if notify != nil {
			notify("info", "Persistence switched to region %s (%s)\n", region, dir)
		}
	}
}
