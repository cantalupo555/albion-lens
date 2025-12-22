package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/cantalupo555/albion-lens/pkg/capture"
	"github.com/cantalupo555/albion-lens/pkg/handlers"
	"github.com/cantalupo555/albion-lens/pkg/photon"
	"github.com/fatih/color"
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
	discovery := flag.Bool("discovery", false, "Enable discovery mode to log all unknown events")
	itemsPath := flag.String("items", "", "Path to ao-bin-dumps directory for item name resolution")
	saveDiscovery := flag.String("save-discovery", "", "Save discovered events to JSON file on exit")
	flag.Parse()

	// Print header
	printHeader()

	// List devices if requested
	if *listDevices {
		if err := capture.PrintDevices(); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Create Albion handler
	albionHandler := handlers.NewAlbionHandler()
	albionHandler.SetDebug(*debug)
	
	// Enable discovery mode if requested
	if *discovery {
		albionHandler.SetDiscoveryMode(true)
	}
	
	// Load item database if path provided
	if *itemsPath != "" {
		if err := albionHandler.LoadItemDatabase(*itemsPath); err != nil {
			color.Yellow("⚠️  Could not load item database: %v\n", err)
			color.Yellow("   Continuing without item name resolution...\n")
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
				if err := albionHandler.LoadItemDatabase(path); err == nil {
					break
				}
			}
		}
	}

	// Create Photon parser
	parser := photon.NewParser(albionHandler)
	parser.SetDebug(*debug)

	// Create network capture
	netCapture := capture.NewCapture(func(payload []byte, srcIP, dstIP net.IP, srcPort, dstPort uint16) {
		if *debug {
			fmt.Printf("\n📦 Packet: %s:%d -> %s:%d (%d bytes)\n",
				srcIP, srcPort, dstIP, dstPort, len(payload))
		}

		// Parse Photon packet
		if err := parser.ParsePacket(payload); err != nil {
			if *debug {
				fmt.Printf("  Error parsing packet: %v\n", err)
			}
		}
	})

	// Set online/offline callback
	netCapture.OnlineCallback = func(online bool) {
		if online {
			color.Green("\n✅ Albion Online detected! Capturing packets...\n")
		} else {
			color.Yellow("\n⏸️  No packets received for 5 seconds. Waiting for game...\n")
		}
	}

	// Start capture
	var err error
	if *deviceName != "" {
		fmt.Printf("Starting capture on device: %s\n", *deviceName)
		err = netCapture.StartOnDevice(*deviceName)
	} else {
		fmt.Println("Starting capture on all network interfaces...")
		err = netCapture.Start()
	}

	if err != nil {
		fmt.Printf("Error starting capture: %v\n", err)
		os.Exit(1)
	}

	fmt.Println()
	color.Cyan("🎮 Waiting for Albion Online traffic...")
	color.Cyan("   Listening on UDP ports 5055 (Master) and 5056 (Game)")
	if *discovery {
		color.HiBlue("   🔍 Discovery mode: ON - logging unknown events")
	}
	fmt.Println()
	fmt.Println("Press Ctrl+C to stop")
	fmt.Println()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\n\nStopping...")
	netCapture.Stop()

	// Print session summary
	printSessionSummary(albionHandler, *discovery, *saveDiscovery)
	
	fmt.Println("Goodbye!")
}

func printHeader() {
	color.Cyan("╔═══════════════════════════════════════════════════════════╗")
	color.Cyan("║                                                           ║")
	color.Cyan("║   ")
	color.New(color.FgYellow, color.Bold).Print("🎮 " + appName + " v" + appVersion)
	color.Cyan("                                ║")
	color.Cyan("║                                                           ║")
	color.Cyan("║   Multiplataforma - Linux / Windows / macOS               ║")
	color.Cyan("║                                                           ║")
	color.Cyan("╚═══════════════════════════════════════════════════════════╝")
	fmt.Println()
}

func printSessionSummary(handler *handlers.AlbionHandler, discoveryMode bool, saveFile string) {
	fmt.Println()
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║                  📊 SESSION SUMMARY                        ║")
	fmt.Println("╠════════════════════════════════════════════════════════════╣")
	
	sessionFame := handler.GetSessionFame()
	if sessionFame > 0 {
		fmt.Printf("║   ⭐ Total Fame Gained: %-35d ║\n", sessionFame)
	} else {
		fmt.Println("║   ⭐ Total Fame Gained: 0                                  ║")
	}
	
	sessionSilver := handler.GetSessionSilver()
	if sessionSilver > 0 {
		fmt.Printf("║   💰 Total Silver Looted: %-33d ║\n", sessionSilver)
	} else {
		fmt.Println("║   💰 Total Silver Looted: 0                                ║")
	}
	
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	
	// Discovery mode summary
	if discoveryMode {
		handler.PrintDiscoverySummary()
		
		// Save discovered events if requested
		if saveFile != "" {
			if err := handler.SaveDiscoveredEvents(saveFile); err != nil {
				color.Red("❌ Failed to save discovered events: %v\n", err)
			} else {
				color.Green("✅ Discovered events saved to: %s\n", saveFile)
			}
		} else {
			// Auto-save to output directory with timestamp
			timestamp := time.Now().Format("2006-01-02_15-04-05")
			autoSaveFile := fmt.Sprintf("output/discovered_events_%s.json", timestamp)
			if err := handler.SaveDiscoveredEvents(autoSaveFile); err == nil {
				color.Green("✅ Discovered events auto-saved to: %s\n", autoSaveFile)
			}
		}
	}
}

// formatSilver formats silver amount in a human-readable way
func formatSilver(amount int64) string {
	if amount >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(amount)/1000000.0)
	} else if amount >= 1000 {
		return fmt.Sprintf("%.1fk", float64(amount)/1000.0)
	}
	return fmt.Sprintf("%d", amount)
}
