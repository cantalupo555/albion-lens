// Package capture handles network packet capture using gopacket/pcap.
// It filters for Albion Online traffic on UDP ports 5055, 5056, and TCP port 4535.
package capture

import (
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

const (
	// Albion Online uses these ports for game traffic
	PortMaster = 5055 // Master/Login Server (UDP)
	PortGame   = 5056 // Game Server (UDP)
	PortChat   = 4535 // Chat Server (TCP)

	// BPF filter for Albion Online traffic
	BPFFilter = "udp and (port 5055 or port 5056)"

	// Capture settings
	// Large enough to capture full UDP datagrams regardless of MTU/VPN/tunneling.
	// gopacket only allocates ci.CaptureLength bytes (actual packet size), not SnapshotLen.
	SnapshotLen = 65536
	Promiscuous = false
	Timeout     = pcap.BlockForever
)

// PacketHandler is a callback function for received packets
type PacketHandler func(payload []byte, srcIP, dstIP net.IP, srcPort, dstPort uint16)

// Capture handles Albion Online network traffic capture
type Capture struct {
	handles []*pcap.Handle
	handler PacketHandler

	// Hot path: checked on every packet
	running atomic.Bool

	// Handles slice protection (append at startup, read at shutdown)
	handlesMu sync.Mutex

	// WaitGroup for all goroutines
	wg sync.WaitGroup

	// Status tracking (updated per-packet, protected by mutex)
	mu             sync.Mutex
	lastPacketTime time.Time
	isOnline       bool
	OnlineCallback func(online bool)
}

// NewCapture creates a new network capture instance
func NewCapture(handler PacketHandler) *Capture {
	return &Capture{
		handler: handler,
		handles: make([]*pcap.Handle, 0),
	}
}

// ListDevices returns all available network devices
func ListDevices() ([]pcap.Interface, error) {
	return pcap.FindAllDevs()
}

// PrintDevices prints all available network devices
func PrintDevices() error {
	devices, err := ListDevices()
	if err != nil {
		return fmt.Errorf("failed to list devices: %w", err)
	}

	fmt.Println("Available network devices:")
	for i, device := range devices {
		fmt.Printf("  %d. %s\n", i+1, device.Name)
		if device.Description != "" {
			fmt.Printf("     Description: %s\n", device.Description)
		}
		for _, addr := range device.Addresses {
			if addr.IP.To4() != nil {
				fmt.Printf("     IPv4: %s\n", addr.IP)
			}
		}
	}
	return nil
}

// Start begins capturing packets on all available interfaces
func (s *Capture) Start() error {
	devices, err := ListDevices()
	if err != nil {
		return fmt.Errorf("failed to list devices: %w", err)
	}

	s.running.Store(true)

	// Start capturing on all devices with IPv4 addresses
	for _, device := range devices {
		for _, addr := range device.Addresses {
			if addr.IP.To4() != nil {
				s.wg.Add(1)
				go s.captureOnDevice(device.Name, addr.IP.String())
			}
		}
	}

	// Start online status checker
	s.wg.Add(1)
	go s.checkOnlineStatus()

	return nil
}

// StartOnDevice begins capturing packets on a specific device
func (s *Capture) StartOnDevice(deviceName string) error {
	s.running.Store(true)

	s.wg.Add(1)
	go s.captureOnDevice(deviceName, "")

	// Start online status checker
	s.wg.Add(1)
	go s.checkOnlineStatus()

	return nil
}

// captureOnDevice captures packets on a specific network device
func (s *Capture) captureOnDevice(deviceName, ipAddr string) {
	defer s.wg.Done()

	handle, err := pcap.OpenLive(deviceName, SnapshotLen, Promiscuous, Timeout)
	if err != nil {
		return
	}

	// Set BPF filter
	if err := handle.SetBPFFilter(BPFFilter); err != nil {
		handle.Close()
		return
	}

	s.handlesMu.Lock()
	s.handles = append(s.handles, handle)
	s.handlesMu.Unlock()

	// Use NextPacket() loop instead of Packets() channel.
	// This avoids the 1000-packet channel buffer and background goroutine
	// that leaks on shutdown when the channel is full.
	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())

	for {
		if !s.running.Load() {
			break
		}

		packet, err := packetSource.NextPacket()
		if err != nil {
			continue
		}

		s.processPacket(packet)
	}
}

// processPacket extracts UDP payload and passes it to the handler
func (s *Capture) processPacket(packet gopacket.Packet) {
	// Get IP layer
	ipLayer := packet.Layer(layers.LayerTypeIPv4)
	if ipLayer == nil {
		return
	}
	ip, _ := ipLayer.(*layers.IPv4)

	// Get UDP layer
	udpLayer := packet.Layer(layers.LayerTypeUDP)
	if udpLayer == nil {
		return
	}
	udp, _ := udpLayer.(*layers.UDP)

	// Get application layer (payload)
	appLayer := packet.ApplicationLayer()
	if appLayer == nil {
		return
	}

	payload := appLayer.Payload()
	if len(payload) == 0 {
		return
	}

	// Update last packet time
	s.mu.Lock()
	s.lastPacketTime = time.Now()
	if !s.isOnline {
		s.isOnline = true
		s.mu.Unlock()
		if s.OnlineCallback != nil {
			s.OnlineCallback(true)
		}
	} else {
		s.mu.Unlock()
	}

	// Call handler
	if s.handler != nil {
		s.handler(
			payload,
			ip.SrcIP,
			ip.DstIP,
			uint16(udp.SrcPort),
			uint16(udp.DstPort),
		)
	}
}

// checkOnlineStatus periodically checks if the game is still sending packets
func (s *Capture) checkOnlineStatus() {
	defer s.wg.Done()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if !s.running.Load() {
			return
		}

		s.mu.Lock()
		if s.isOnline && time.Since(s.lastPacketTime) > 5*time.Second {
			s.isOnline = false
			s.mu.Unlock()
			if s.OnlineCallback != nil {
				s.OnlineCallback(false)
			}
		} else {
			s.mu.Unlock()
		}
	}
}

// Stop stops all packet capture
func (s *Capture) Stop() {
	s.running.Store(false)

	s.handlesMu.Lock()
	for _, handle := range s.handles {
		handle.Close()
	}
	s.handlesMu.Unlock()

	s.wg.Wait()
}

// IsOnline returns whether the game is currently sending packets
func (s *Capture) IsOnline() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.isOnline
}
