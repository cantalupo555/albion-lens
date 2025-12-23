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
	"github.com/cantalupo555/albion-lens/pkg/photon"
)

var (
	appName    = "Albion Lens"
	appVersion = "dev"
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

	// Create TUI handler that sends events to channels
	handler := NewTUIHandler(eventChan, *debug)

	// Load item database if path provided
	if *itemsPath != "" {
		// TODO: Implement item loading in handler
		_ = itemsPath
	} else {
		// Try to auto-detect ao-bin-dumps in common locations
		commonPaths := []string{
			"../ao-bin-dumps",
			"../../ao-bin-dumps",
			filepath.Join(os.Getenv("HOME"), "Documents/albion/ao-bin-dumps"),
		}
		for _, path := range commonPaths {
			if _, err := os.Stat(filepath.Join(path, "items.json")); err == nil {
				// TODO: Load item database
				_ = path
				break
			}
		}
	}

	// Create Photon parser
	parser := photon.NewParser(handler)
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

// TUIHandler implements photon.PhotonHandler and sends events to the TUI
type TUIHandler struct {
	eventChan chan<- tui.EventMsg
	debug     bool
}

// NewTUIHandler creates a new handler that sends events to the TUI
func NewTUIHandler(eventChan chan<- tui.EventMsg, debug bool) *TUIHandler {
	return &TUIHandler{
		eventChan: eventChan,
		debug:     debug,
	}
}

// OnEvent handles game events from the Photon parser
func (h *TUIHandler) OnEvent(code byte, params map[byte]interface{}) {
	// TODO: Map event codes to meaningful messages
	// For now, just log that we received an event
	if h.debug {
		h.sendEvent("info", fmt.Sprintf("Event %d received", code))
	}
}

// OnRequest handles operation requests from the Photon parser
func (h *TUIHandler) OnRequest(code byte, params map[byte]interface{}) {
	// Requests are usually from client to server, less interesting for display
	if h.debug {
		h.sendEvent("info", fmt.Sprintf("Request %d sent", code))
	}
}

// OnResponse handles operation responses from the Photon parser
func (h *TUIHandler) OnResponse(code byte, returnCode int16, debugMsg string, params map[byte]interface{}) {
	if h.debug {
		h.sendEvent("info", fmt.Sprintf("Response %d received (return: %d)", code, returnCode))
	}
}

// sendEvent sends an event to the TUI channel (non-blocking)
func (h *TUIHandler) sendEvent(eventType, message string) {
	select {
	case h.eventChan <- tui.EventMsg{
		Type:      eventType,
		Message:   message,
		Timestamp: time.Now(),
	}:
	default:
		// Channel full, drop event
	}
}
