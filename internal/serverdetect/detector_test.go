package serverdetect

import (
	"net"
	"sync/atomic"
	"testing"
	"time"
)

// fakeClock is a controllable time source for deterministic detector tests.
type fakeClock struct {
	t atomic.Int64 // unix nanos
}

func newFakeClock(start time.Time) *fakeClock {
	f := &fakeClock{}
	f.t.Store(start.UnixNano())
	return f
}

func (f *fakeClock) now() time.Time {
	return time.Unix(0, f.t.Load())
}

func (f *fakeClock) advance(d time.Duration) {
	f.t.Add(d.Nanoseconds())
}

func TestDetectorInitial(t *testing.T) {
	d := NewDetector()
	if got := d.Current().Location; got != ServerLocationUnknown {
		t.Errorf("Current() before any detection = %v, want Unknown", got)
	}
	if got := d.LastPacketReceived(); !got.IsZero() {
		t.Errorf("LastPacketReceived before any detection = %v, want zero", got)
	}
}

func TestDetectorIgnoresUnknownIP(t *testing.T) {
	var called int32
	d := NewDetector(WithOnChange(func(ChangeEvent) { atomic.AddInt32(&called, 1) }))
	// Non-matching / invalid IPs must not change state or emit events.
	d.DetectFromIP(net.ParseIP("8.8.8.8"))
	d.DetectFromIP(net.ParseIP("not-an-ip")) // ParseIP returns nil → ignored
	d.DetectFromIP(nil)
	d.DetectFromIP(net.ParseIP("2001:db8::1")) // IPv6 → ignored

	if d.Current().Location != ServerLocationUnknown {
		t.Errorf("state changed after unknown IPs: %v", d.Current().Location)
	}
	if got := d.LastPacketReceived(); !got.IsZero() {
		t.Errorf("LastPacketReceived updated by unknown IPs: %v", got)
	}
	if atomic.LoadInt32(&called) != 0 {
		t.Errorf("onChange fired for unknown IPs: %d calls", called)
	}
}

func TestDetectorPromotesAfterStability(t *testing.T) {
	clk := newFakeClock(time.Unix(1_700_000_000, 0))
	var events []ChangeEvent
	d := NewDetector(
		WithNowFunc(clk.now),
		WithStability(5*time.Second),
		WithOnChange(func(e ChangeEvent) { events = append(events, e) }),
	)

	// First packet from Americas: becomes candidate, NOT current yet.
	d.DetectFromIP(net.IPv4(5, 188, 125, 10))
	if d.Current().Location != ServerLocationUnknown {
		t.Fatalf("promoted before stability window: %v", d.Current().Location)
	}
	if len(events) != 0 {
		t.Fatalf("emitted %d events before promotion, want 0", len(events))
	}

	// Within the window: still a candidate, no promotion.
	clk.advance(4 * time.Second)
	d.DetectFromIP(net.IPv4(5, 188, 125, 11))
	if d.Current().Location != ServerLocationUnknown {
		t.Fatalf("promoted before stability elapsed: %v", d.Current().Location)
	}

	// After stability window: promoted.
	clk.advance(2 * time.Second) // total 6s >= 5s
	d.DetectFromIP(net.IPv4(5, 188, 125, 12))
	if got := d.Current().Location; got != ServerLocationAmerica {
		t.Errorf("after stability, Current = %v, want Americas", got)
	}
	if len(events) != 1 || events[0].Previous.Location != ServerLocationUnknown ||
		events[0].Current.Location != ServerLocationAmerica {
		t.Errorf("promotion event = %+v, want prev=Unknown cur=Americas", events[0])
	}
}

func TestDetectorNoPromotionForUnstableCandidate(t *testing.T) {
	// A candidate seen only briefly (then signal disappears) must never promote.
	clk := newFakeClock(time.Unix(1_700_000_000, 0))
	d := NewDetector(WithNowFunc(clk.now), WithStability(5*time.Second))

	d.DetectFromIP(net.IPv4(5, 188, 125, 10)) // candidate Americas
	clk.advance(2 * time.Second)
	// No more Americas packets. Even after a long time, Current stays Unknown
	// because no packet re-confirmed the candidate past the window.
	clk.advance(1 * time.Hour)
	if d.Current().Location != ServerLocationUnknown {
		t.Errorf("promoted without re-confirmation past window: %v", d.Current().Location)
	}
}

func TestDetectorDifferentCandidateReplacesAndRestartsClock(t *testing.T) {
	clk := newFakeClock(time.Unix(1_700_000_000, 0))
	d := NewDetector(WithNowFunc(clk.now), WithStability(5*time.Second))

	// Start observing Americas.
	d.DetectFromIP(net.IPv4(5, 188, 125, 10))
	clk.advance(3 * time.Second)

	// Switch candidate to Europe mid-window: candidate replaced, clock restarts.
	d.DetectFromIP(net.IPv4(193, 169, 238, 5))
	clk.advance(3 * time.Second) // 3s on Europe, still < 5s
	d.DetectFromIP(net.IPv4(193, 169, 238, 6))
	if d.Current().Location != ServerLocationUnknown {
		t.Errorf("promoted Europe before stability: %v", d.Current().Location)
	}

	// Push past the window for Europe.
	clk.advance(3 * time.Second) // total 6s on Europe
	d.DetectFromIP(net.IPv4(193, 169, 238, 7))
	if got := d.Current().Location; got != ServerLocationEurope {
		t.Errorf("after Europe stability, Current = %v, want Europe", got)
	}
}

func TestDetectorSameRegionResetsCandidate(t *testing.T) {
	// Once a region is Current, a stray packet from another region that does
	// NOT confirm should not cause the same-region current to be lost.
	// (Confirmed-different-region reset is covered separately.)
	clk := newFakeClock(time.Unix(1_700_000_000, 0))
	d := NewDetector(WithNowFunc(clk.now), WithStability(5*time.Second))

	// Promote Americas.
	stabilize(d, clk, net.IPv4(5, 188, 125, 10), 5*time.Second)
	if d.Current().Location != ServerLocationAmerica {
		t.Fatalf("setup: Americas not current, got %v", d.Current().Location)
	}

	// More Americas packets while already current: no-op, candidate cleared.
	d.DetectFromIP(net.IPv4(5, 188, 125, 99))
	if got := d.Current().Location; got != ServerLocationAmerica {
		t.Errorf("same-region re-detection lost current: %v", got)
	}
}

func TestDetectorResetToUnknownOnRegionChange(t *testing.T) {
	clk := newFakeClock(time.Unix(1_700_000_000, 0))
	var events []ChangeEvent
	d := NewDetector(
		WithNowFunc(clk.now),
		WithStability(5*time.Second),
		WithOnChange(func(e ChangeEvent) { events = append(events, e) }),
	)

	// Establish Americas.
	stabilize(d, clk, net.IPv4(5, 188, 125, 10), 5*time.Second)
	events = events[:0] // drop the initial promotion event

	// Packet from Europe arrives while Americas is current:
	// detector must immediately reset to Unknown (event 1), then promote Europe
	// only after stability (event 2).
	d.DetectFromIP(net.IPv4(193, 169, 238, 5))
	if d.Current().Location != ServerLocationUnknown {
		t.Fatalf("after first Europe packet, Current = %v, want Unknown (reset)", d.Current().Location)
	}
	if len(events) != 1 || events[0].Previous.Location != ServerLocationAmerica ||
		events[0].Current.Location != ServerLocationUnknown {
		t.Errorf("reset event = %+v, want prev=Americas cur=Unknown", events[0])
	}

	// Confirm Europe past the window.
	clk.advance(6 * time.Second)
	d.DetectFromIP(net.IPv4(193, 169, 238, 6))
	if got := d.Current().Location; got != ServerLocationEurope {
		t.Errorf("after Europe stability, Current = %v, want Europe", got)
	}
	if len(events) != 2 {
		t.Fatalf("expected promotion event after reset, got %d total events", len(events))
	}
	if events[1].Previous.Location != ServerLocationUnknown ||
		events[1].Current.Location != ServerLocationEurope {
		t.Errorf("promotion event = %+v, want prev=Unknown cur=Europe", events[1])
	}
}

func TestDetectorOnChangeInvokedOutsideLock(t *testing.T) {
	// onChange must be able to re-enter Current() without deadlocking.
	clk := newFakeClock(time.Unix(1_700_000_000, 0))
	var d *Detector
	d = NewDetector(
		WithNowFunc(clk.now),
		WithStability(5*time.Second),
		WithOnChange(func(e ChangeEvent) {
			// Re-entrant read during callback dispatch.
			_ = d.Current()
		}),
	)
	stabilize(d, clk, net.IPv4(5, 188, 125, 10), 5*time.Second)
	// If the callback held the lock, this test would hang/timeout under -race.
}

func TestDetectorLastPacketUpdatedOnKnownMatch(t *testing.T) {
	clk := newFakeClock(time.Unix(1_700_000_000, 0))
	d := NewDetector(WithNowFunc(clk.now), WithStability(5*time.Second))
	if got := d.LastPacketReceived(); !got.IsZero() {
		t.Fatalf("initial LastPacketReceived = %v, want zero", got)
	}
	d.DetectFromIP(net.IPv4(5, 188, 125, 10))
	want := clk.now()
	if got := d.LastPacketReceived(); !got.Equal(want) {
		t.Errorf("LastPacketReceived = %v, want %v", got, want)
	}
}

// stabilize feeds the same server IP repeatedly, advancing the clock past the
// stability window, to drive the detector to a confirmed Current state.
func stabilize(d *Detector, clk *fakeClock, ip net.IP, window time.Duration) {
	d.DetectFromIP(ip)
	clk.advance(window + time.Second)
	d.DetectFromIP(ip)
}
