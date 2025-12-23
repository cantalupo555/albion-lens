package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cantalupo555/albion-lens/internal/tui"
	"github.com/cantalupo555/albion-lens/pkg/capture"
	"github.com/cantalupo555/albion-lens/pkg/handlers"
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

	// Create channels for TUI communication
	eventChan := make(chan tui.EventMsg, 100)
	statsChan := make(chan *photon.Stats, 10)

	// Create Albion handler (uses existing event parsing logic)
	albionHandler := handlers.NewAlbionHandler()
	albionHandler.SetDebug(*debug)

	// Set event callback to send events to TUI
	albionHandler.SetEventCallback(func(eventType, message string) {
		select {
		case eventChan <- tui.EventMsg{
			Type:      eventType,
			Message:   message,
			Timestamp: time.Now(),
		}:
		default:
			// Channel full, drop event
		}
	})

	// Load item database if path provided
	if *itemsPath != "" {
		if err := albionHandler.LoadItemDatabase(*itemsPath); err != nil {
			// Silently continue without item names
		}
	} else {
		// Try to auto-detect ao-bin-dumps in common locations
		commonPaths := []string{
			"../ao-bin-dumps",
			"../../ao-bin-dumps",
			filepath.Join(os.Getenv("HOME"), "Documents/albion/ao-bin-dumps"),
		}
		for _, path := range commonPaths {
			if _, err := os.Stat(filepath.Join(path, "items.json")); err == nil {
				_ = albionHandler.LoadItemDatabase(path)
				break
			}
		}
	}

	// Create Photon parser with Albion handler
	parser := photon.NewParser(albionHandler)
	parser.SetDebug(*debug)
	defer parser.Close()

	// Create network capture
	netCapture := capture.NewCapture(func(payload []byte, srcIP, dstIP net.IP, srcPort, dstPort uint16) {
		// Parse Photon packet
		_ = parser.ParsePacket(payload)
	})

	// Set online/offline callback
	netCapture.OnlineCallback = func(online bool) {
		select {
		case eventChan <- tui.EventMsg{
			Type:      "info",
			Message:   statusMessage(online),
			Timestamp: time.Now(),
		}:
		default:
		}
	}

	// Start stats update goroutine
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for range ticker.C {
			select {
			case statsChan <- parser.Stats:
			default:
			}
		}
	}()

	// Start capture
	var err error
	if *deviceName != "" {
		err = netCapture.StartOnDevice(*deviceName)
	} else {
		err = netCapture.Start()
	}

	if err != nil {
		fmt.Printf("Error starting capture: %v\n", err)
		fmt.Println("Try running with sudo or as administrator.")
		os.Exit(1)
	}
	defer netCapture.Stop()

	// Create and run TUI
	model := tui.New(eventChan, statsChan)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running TUI: %v\n", err)
		os.Exit(1)
	}
}

func statusMessage(online bool) string {
	if online {
		return "Albion Online detected! Capturing packets..."
	}
	return "Waiting for Albion Online traffic..."
}
