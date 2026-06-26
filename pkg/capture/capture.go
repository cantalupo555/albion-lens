// Package capture handles network packet capture using gopacket/pcap.
// It filters for Albion Online traffic on UDP ports 5055, 5056, and TCP port 4535.
package capture

import (
	"errors"
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

// deviceOpener abstracts pcap.OpenLive so capture startup logic can be tested
// without requiring raw-socket privileges or a real network device.
type deviceOpener func(device string, snapshotLen int32, promisc bool, timeout time.Duration) (*pcap.Handle, error)

// Capture handles Albion Online network traffic capture
type Capture struct {
	handles []*pcap.Handle
	handler PacketHandler

	// opener is used by openDevice. Defaults to pcap.OpenLive; injectable for tests.
	opener deviceOpener

	// Hot path: checked on every packet
	running atomic.Bool

	// Handles slice protection (append at startup, read at shutdown)
	handlesMu sync.Mutex

	// WaitGroup for all goroutines
	wg sync.WaitGroup

	// Status tracking (updated per-packet, protected by mutex)
	mu                  sync.Mutex
	lastPacketTime      time.Time
	isOnline            bool
	OnlineCallback      func(online bool)
	DeviceErrorCallback func(deviceName string, err error)
}

// NewCapture creates a new network capture instance
func NewCapture(handler PacketHandler) *Capture {
	return &Capture{
		handler: handler,
		handles: make([]*pcap.Handle, 0),
		opener:  pcap.OpenLive,
	}
}

// NewCaptureWithOpener creates a capture instance with a custom device opener.
// Intended for tests that need to simulate open failures without a real device.
func NewCaptureWithOpener(handler PacketHandler, opener deviceOpener) *Capture {
	c := NewCapture(handler)
	if opener != nil {
		c.opener = opener
	}
	return c
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

// Start begins capturing packets on all available interfaces.
//
// Devices are opened synchronously so that an honest error can be returned
// when none of them can be captured (e.g. missing privileges). Each device
// that fails to open is reported via DeviceErrorCallback (when set); only the
// "all failed" case returns an error, since partial failures still leave the
// capture functional.
//
// The returned error aggregates every per-device failure so callers can
// surface the full diagnostic context (which device failed and why) even
// when the TUI — the usual consumer of DeviceErrorCallback — has not started
// yet (e.g. when main.go exits immediately on Start error).
func (s *Capture) Start() error {
	devices, err := ListDevices()
	if err != nil {
		return fmt.Errorf("failed to list devices: %w", err)
	}

	s.running.Store(true)

	// Deduplicate by device name. The previous implementation spawned one
	// goroutine per IPv4 address, opening the same device multiple times when
	// it had several IPv4 addresses (causing duplicate packets or EBUSY).
	deviceNames := selectDevices(devices)

	// Collect per-device errors so the fatal "all failed" return value can
	// carry the full diagnostic context, not just a generic hint. The callback
	// still fires for each failure so partial-success callers stay informed.
	var openErrors []error
	for _, name := range deviceNames {
		handle, err := s.openDevice(name)
		if err != nil {
			openErrors = append(openErrors, fmt.Errorf("%s: %w", name, err))
			if s.DeviceErrorCallback != nil {
				s.DeviceErrorCallback(name, err)
			}
			continue
		}
		s.launchDevice(handle)
	}

	if len(s.handles) == 0 {
		s.running.Store(false)
		return fmt.Errorf("failed to open any capture device: tried %d device(s), all failed (errors: %w). Common cause: insufficient permissions — try running with sudo",
			len(deviceNames), errors.Join(openErrors...))
	}

	// Start online status checker only when at least one device is capturing.
	s.wg.Add(1)
	go s.checkOnlineStatus()

	return nil
}

// StartOnDevice begins capturing packets on a specific device.
//
// The device is opened synchronously so callers learn immediately whether
// capture actually started.
func (s *Capture) StartOnDevice(deviceName string) error {
	s.running.Store(true)

	handle, err := s.openDevice(deviceName)
	if err != nil {
		s.running.Store(false)
		if s.DeviceErrorCallback != nil {
			s.DeviceErrorCallback(deviceName, err)
		}
		return fmt.Errorf("failed to open device %q: %w", deviceName, err)
	}

	s.launchDevice(handle)

	s.wg.Add(1)
	go s.checkOnlineStatus()

	return nil
}

// selectDevices returns the unique device names that have at least one IPv4
// address. It is a pure function so it can be unit-tested without pcap.
func selectDevices(devices []pcap.Interface) []string {
	seen := make(map[string]bool, len(devices))
	names := make([]string, 0, len(devices))
	for _, device := range devices {
		if seen[device.Name] {
			continue
		}
		for _, addr := range device.Addresses {
			if addr.IP.To4() != nil {
				names = append(names, device.Name)
				seen[device.Name] = true
				break
			}
		}
	}
	return names
}

// openDevice opens a pcap handle on deviceName and applies the BPF filter.
// Returns an error describing which step failed so callers can report it.
func (s *Capture) openDevice(deviceName string) (*pcap.Handle, error) {
	handle, err := s.opener(deviceName, SnapshotLen, Promiscuous, Timeout)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	if err := handle.SetBPFFilter(BPFFilter); err != nil {
		handle.Close()
		return nil, fmt.Errorf("bpf filter: %w", err)
	}
	return handle, nil
}

// launchDevice registers an already-opened handle and starts its packet-read
// goroutine. Centralized so the handle/WaitGroup/goroutine sequence stays in
// sync between Start and StartOnDevice.
func (s *Capture) launchDevice(handle *pcap.Handle) {
	s.handlesMu.Lock()
	s.handles = append(s.handles, handle)
	s.handlesMu.Unlock()

	s.wg.Add(1)
	go s.readPackets(handle)
}

// readPackets runs the packet read loop on handle until s.running becomes false.
// Uses NextPacket() instead of Packets() channel to avoid the 1000-packet
// channel buffer and background goroutine that leaks on shutdown when full.
//
// The caller must balance the WaitGroup: wg.Add(1) before launching the
// goroutine, this function calls wg.Done() on return.
func (s *Capture) readPackets(handle *pcap.Handle) {
	defer s.wg.Done()

	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())

	for {
		if !s.running.Load() {
			return
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
