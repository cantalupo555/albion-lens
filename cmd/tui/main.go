package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cantalupo555/albion-lens/internal/tui"
	"github.com/cantalupo555/albion-lens/pkg/backend"
	"github.com/cantalupo555/albion-lens/pkg/capture"
	"github.com/cantalupo555/albion-lens/pkg/photon"
)

func main() {
	// Parse command line flags
	listDevices := flag.Bool("list", false, "List available network devices")
	deviceName := flag.String("device", "", "Specific device to capture on (captures all if not specified)")
	debug := flag.Bool("debug", false, "Enable debug output")
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

	svc := backend.New(opts...)

	// Create channels for TUI communication
	// Use a buffered channel for bulk messages
	bulkEventChan := make(chan tui.BulkEventMsg, 5) // 5 batches of 50 = 250 events
	statsChan := make(chan *photon.Stats, 10)

	// Bridge backend events to TUI with batching
	go func() {
		const batchSize = 50
		const flushInterval = 50 * time.Millisecond
		
		buffer := make([]tui.EventMsg, 0, batchSize)
		ticker := time.NewTicker(flushInterval)
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
				if stats := svc.ParserStats(); stats != nil {
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
			case event, ok := <-svc.Events:
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
				if len(buffer) >= batchSize {
					flush()
					// Reset ticker to avoid double flushing
					ticker.Reset(flushInterval)
				}

			case <-ticker.C:
				flush()
			}
		}
	}()

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
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running TUI: %v\n", err)
		os.Exit(1)
	}
}
