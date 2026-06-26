package capture

import (
	"errors"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/gopacket/pcap"
)

// failingOpener always returns an error, simulating a device that cannot be
// opened (e.g. insufficient permissions).
func failingOpener(_ string, _ int32, _ bool, _ time.Duration) (*pcap.Handle, error) {
	return nil, errors.New("permission denied")
}

// noopHandler is a PacketHandler that discards every packet.
func noopHandler(_ []byte, _, _ net.IP, _, _ uint16) {}

// ============================================
// selectDevices tests (pure function, no pcap required)
// ============================================

func TestSelectDevices_EmptyInput(t *testing.T) {
	got := selectDevices(nil)
	if len(got) != 0 {
		t.Errorf("expected empty slice for nil input, got %v", got)
	}
}

func TestSelectDevices_FiltersNonIPv4(t *testing.T) {
	devices := []pcap.Interface{
		{
			Name: "ipv6only",
			Addresses: []pcap.InterfaceAddress{
				{IP: net.ParseIP("::1")},
			},
		},
	}
	got := selectDevices(devices)
	if len(got) != 0 {
		t.Errorf("expected no device for IPv6-only interface, got %v", got)
	}
}

// TestSelectDevices_DeduplicatesMultipleIPv4 verifies the bug fix: previously
// Start opened one handle per IPv4 address, so a device with several IPv4
// addresses was opened multiple times. selectDevices must return the device
// name only once.
func TestSelectDevices_DeduplicatesMultipleIPv4(t *testing.T) {
	devices := []pcap.Interface{
		{
			Name: "eth0",
			Addresses: []pcap.InterfaceAddress{
				{IP: net.ParseIP("192.168.1.10")},
				{IP: net.ParseIP("10.0.0.5")},
			},
		},
	}
	got := selectDevices(devices)
	if len(got) != 1 {
		t.Fatalf("expected 1 device (deduped), got %d: %v", len(got), got)
	}
	if got[0] != "eth0" {
		t.Errorf("expected 'eth0', got %q", got[0])
	}
}

func TestSelectDevices_MultipleDevicesEachOnce(t *testing.T) {
	devices := []pcap.Interface{
		{Name: "eth0", Addresses: []pcap.InterfaceAddress{{IP: net.ParseIP("10.0.0.1")}}},
		{Name: "wlan0", Addresses: []pcap.InterfaceAddress{{IP: net.ParseIP("192.168.0.2")}}},
		// duplicate eth0 entry from a second pcap.Interface (shouldn't normally
		// happen but selectDevices must be defensive)
		{Name: "eth0", Addresses: []pcap.InterfaceAddress{{IP: net.ParseIP("10.0.0.9")}}},
	}
	got := selectDevices(devices)
	if len(got) != 2 {
		t.Fatalf("expected 2 unique devices, got %d: %v", len(got), got)
	}
}

// ============================================
// StartOnDevice tests with injected failing opener
// ============================================

func TestStartOnDevice_OpenerFails_ReturnsError(t *testing.T) {
	c := NewCaptureWithOpener(noopHandler, failingOpener)
	err := c.StartOnDevice("fake0")
	if err == nil {
		t.Fatal("expected error when opener fails, got nil")
	}
	if !strings.Contains(err.Error(), "fake0") {
		t.Errorf("expected error to mention device name 'fake0', got: %v", err)
	}
	if c.running.Load() {
		t.Error("expected running=false after StartOnDevice failure")
	}
}

func TestStartOnDevice_OpenerFails_InvokesCallback(t *testing.T) {
	var (
		mu        sync.Mutex
		calls     []struct{ name string }
		callbacks int
	)

	c := NewCaptureWithOpener(noopHandler, failingOpener)
	c.DeviceErrorCallback = func(deviceName string, _ error) {
		mu.Lock()
		defer mu.Unlock()
		calls = append(calls, struct{ name string }{deviceName})
		callbacks++
	}

	_ = c.StartOnDevice("fake0")

	mu.Lock()
	defer mu.Unlock()
	if callbacks != 1 {
		t.Errorf("expected DeviceErrorCallback called once, got %d", callbacks)
	}
	if len(calls) != 1 || calls[0].name != "fake0" {
		t.Errorf("expected callback called with 'fake0', got %+v", calls)
	}
}

func TestStartOnDevice_NoCallback_DoesNotPanic(t *testing.T) {
	c := NewCaptureWithOpener(noopHandler, failingOpener)
	// DeviceErrorCallback left nil; StartOnDevice must not panic.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("StartOnDevice panicked with nil callback: %v", r)
		}
	}()
	_ = c.StartOnDevice("fake0")
}

// ============================================
// Start tests with injected failing opener
// ============================================

// TestStart_AllOpenersFail_ReturnsError exercises the synchronous open path
// when every selected device fails. It depends on pcap.FindAllDevs returning
// at least one IPv4 device (the loopback interface qualifies on most CI
// runners); if none is present, the test is skipped.
//
// (Originally planned as TestStart_NoRealDevices_ReturnsError — same intent,
// renamed to reflect the actual mechanism: failing opens rather than missing
// devices, since the latter is rare on dev/CI machines.)
func TestStart_AllOpenersFail_ReturnsError(t *testing.T) {
	devices, err := pcap.FindAllDevs()
	if err != nil {
		t.Skipf("pcap.FindAllDevs unavailable in this environment: %v", err)
	}
	if len(selectDevices(devices)) == 0 {
		t.Skip("no IPv4 devices available; cannot exercise Start() failure path")
	}

	c := NewCaptureWithOpener(noopHandler, failingOpener)
	err = c.Start()
	if err == nil {
		// Cleanup: stop any goroutines that may have started if a device
		// unexpectedly succeeded.
		c.Stop()
		t.Fatal("expected error when all device opens fail, got nil")
	}
	if !strings.Contains(err.Error(), "sudo") {
		t.Errorf("expected error to mention 'sudo' hint, got: %v", err)
	}
	// The aggregated error must carry the per-device diagnostic context so
	// that callers (main.go) can surface which interface failed even though
	// the TUI never starts to consume DeviceErrorCallback events.
	if !strings.Contains(err.Error(), "permission denied") {
		t.Errorf("expected aggregated error to include per-device failure reason, got: %v", err)
	}
	if c.running.Load() {
		t.Error("expected running=false after Start() total failure")
	}
}

// TestStart_NoDevicesAvailable exercises the corner case where pcap reports no
// IPv4 device at all. In that scenario Start() must still fail honestly (with
// a descriptive error) rather than silently starting zero goroutines. The test
// is skipped on hosts that do have IPv4 devices, since that is the realistic
// production environment.
func TestStart_NoDevicesAvailable(t *testing.T) {
	devices, err := pcap.FindAllDevs()
	if err != nil {
		t.Skipf("pcap.FindAllDevs unavailable: %v", err)
	}
	if len(selectDevices(devices)) > 0 {
		t.Skip("host has IPv4 devices; cannot exercise the no-devices path")
	}

	c := NewCapture(noopHandler)
	err = c.Start()
	if err == nil {
		c.Stop()
		t.Fatal("expected error when no IPv4 devices are available, got nil")
	}
}

// TestStart_AllOpenersFail_InvokesCallbackPerDevice verifies that partial
// failure reporting fires for each device that fails to open.
func TestStart_AllOpenersFail_InvokesCallbackPerDevice(t *testing.T) {
	devices, err := pcap.FindAllDevs()
	if err != nil {
		t.Skipf("pcap.FindAllDevs unavailable: %v", err)
	}
	names := selectDevices(devices)
	if len(names) == 0 {
		t.Skip("no IPv4 devices available")
	}

	var (
		mu    sync.Mutex
		count int
	)
	c := NewCaptureWithOpener(noopHandler, failingOpener)
	c.DeviceErrorCallback = func(_ string, _ error) {
		mu.Lock()
		count++
		mu.Unlock()
	}

	_ = c.Start()

	mu.Lock()
	defer mu.Unlock()
	if count != len(names) {
		t.Errorf("expected callback called %d times (one per device), got %d", len(names), count)
	}
}

// ============================================
// Constructor tests
// ============================================

func TestNewCapture_DefaultOpenerIsPcapOpenLive(t *testing.T) {
	c := NewCapture(noopHandler)
	if c.opener == nil {
		t.Fatal("expected default opener to be non-nil")
	}
	// We cannot compare function pointers portably, but invoking it with a
	// bogus device must behave like pcap.OpenLive (i.e. return an error).
	_, err := c.opener("definitely-not-a-device", SnapshotLen, Promiscuous, Timeout)
	if err == nil {
		t.Error("expected default opener to fail for a non-existent device")
	}
}

func TestNewCaptureWithOpener_NilOpenerFallsBackToDefault(t *testing.T) {
	c := NewCaptureWithOpener(noopHandler, nil)
	if c.opener == nil {
		t.Fatal("expected fallback to default opener when nil is injected")
	}
}

func TestNewCaptureWithOpener_UsesInjectedOpener(t *testing.T) {
	called := false
	injected := func(_ string, _ int32, _ bool, _ time.Duration) (*pcap.Handle, error) {
		called = true
		return nil, errors.New("injected failure")
	}
	c := NewCaptureWithOpener(noopHandler, injected)
	_ = c.StartOnDevice("any")

	if !called {
		t.Error("expected injected opener to be called")
	}
}

// TestStartOnDevice_OpenerSucceeds_ReturnsNil exercises the happy path with a
// real device: StartOnDevice must return nil and leave the Capture in the
// running state. Uses the loopback interface ("lo" on Linux) which is usually
// openable without elevated privileges; skipped when not available.
func TestStartOnDevice_OpenerSucceeds_ReturnsNil(t *testing.T) {
	const loopback = "lo"
	// Probe whether lo can actually be opened in this environment.
	probe, err := pcap.OpenLive(loopback, SnapshotLen, Promiscuous, Timeout)
	if err != nil {
		t.Skipf("cannot open loopback device %q without privileges: %v", loopback, err)
	}
	probe.Close()

	c := NewCapture(noopHandler)
	if err := c.StartOnDevice(loopback); err != nil {
		t.Fatalf("expected nil error opening %s, got: %v", loopback, err)
	}
	if !c.running.Load() {
		t.Error("expected running=true after successful StartOnDevice")
	}
	if len(c.handles) != 1 {
		t.Errorf("expected 1 registered handle, got %d", len(c.handles))
	}

	// Cleanup must drain the readPackets goroutine and close the handle.
	c.Stop()
	if c.running.Load() {
		t.Error("expected running=false after Stop()")
	}
}
