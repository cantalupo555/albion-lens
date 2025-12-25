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
	eventChan := make(chan tui.EventMsg, 100)
	statsChan := make(chan *photon.Stats, 10)

	// Bridge backend events to TUI
	go func() {
		for event := range svc.Events {
			select {
			case eventChan <- tui.EventMsg{
				Type:      string(event.Type),
				Message:   event.Message,
				Timestamp: event.Timestamp,
				Data:      event.Data,
			}:
			default:
			}
		}
	}()

	// Bridge backend stats to TUI
	go func() {
		for stats := range svc.Stats {
			select {
			case statsChan <- stats:
			default:
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

	// Send initial status event
	eventChan <- tui.EventMsg{
		Type:      "info",
		Message:   "Waiting for Albion Online traffic...",
		Timestamp: time.Now(),
	}

	// Create and run TUI
	model := tui.New(svc, eventChan, statsChan)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running TUI: %v\n", err)
		os.Exit(1)
	}
}
