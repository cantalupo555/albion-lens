package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/cantalupo555/albion-lens/internal/storage"
	"github.com/cantalupo555/albion-lens/internal/tui"
	"github.com/cantalupo555/albion-lens/pkg/backend"
	"github.com/cantalupo555/albion-lens/pkg/capture"
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

	// --- Persistence (issues #103, #104): restore state, then keep it warm. ---
	// Resolve XDG data paths first (the directory is created if missing).
	dungeonPath, err := storage.DataFile("dungeon-runs.json")
	if err != nil {
		fmt.Printf("Warning: could not resolve dungeon data path: %v\n", err)
	}
	statsPath, err := storage.DataFile("session-stats.json")
	if err != nil {
		fmt.Printf("Warning: could not resolve stats data path: %v\n", err)
	}

	// Load-on-startup. The handler is created by Start(), so this must run
	// after it. A missing file is the first-run case (handled as empty state
	// inside Load); a corrupt file leaves the in-memory state unchanged.
	if h := svc.Handler(); h != nil {
		if err := h.LoadDungeonRuns(dungeonPath); err != nil {
			fmt.Printf("Warning: could not load dungeon history: %v\n", err)
		}
		if err := h.LoadSessionStats(statsPath); err != nil {
			fmt.Printf("Warning: could not load session stats: %v\n", err)
		}
	}

	// Periodic flush while the TUI runs: a crash or kill loses at most
	// saveInterval of progress. Cancelled after p.Run returns.
	saveCtx, saveCancel := context.WithCancel(context.Background())
	defer saveCancel()
	go periodicSave(saveCtx, saveInterval, func() error {
		h := svc.Handler()
		if h == nil {
			return nil
		}
		if err := h.SaveDungeonRuns(dungeonPath); err != nil {
			return err
		}
		return h.SaveSessionStats(statsPath)
	})

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

	// Stop the periodic flush, then do a final save-on-exit so the on-disk
	// state reflects the very last in-memory values.
	saveCancel()
	if h := svc.Handler(); h != nil {
		if err := h.SaveDungeonRuns(dungeonPath); err != nil {
			fmt.Printf("Warning: could not save dungeon history: %v\n", err)
		}
		if err := h.SaveSessionStats(statsPath); err != nil {
			fmt.Printf("Warning: could not save session stats: %v\n", err)
		}
	}

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
// error from save is surfaced as a warning so the user learns mid-session that
// persistence has degraded (the final save-on-exit remains the safety net).
func periodicSave(ctx context.Context, interval time.Duration, save func() error) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := save(); err != nil {
				fmt.Printf("Warning: periodic save failed: %v\n", err)
			}
		}
	}
}
