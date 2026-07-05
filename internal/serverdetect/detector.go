package serverdetect

import (
	"net"
	"sync"
	"time"
)

// DefaultStabilityDuration is how long a candidate region must be observed
// continuously before it is promoted to the current server. Mirrors the
// reference's 5-second ServerDetectionStabilityDuration. It prevents a single
// stray packet (e.g. a brief reroute) from flipping the active region and
// thrashing the on-disk state.
const DefaultStabilityDuration = 5 * time.Second

// ChangeEvent describes a transition of the detected server. Previous may be
// Unknown (first detection or after a reset); Current may be Unknown (the
// detector reverted to "no confident region" after seeing a different one).
type ChangeEvent struct {
	Previous ServerInfo
	Current  ServerInfo
}

// Detector tracks the active Albion server region from observed Photon server
// IPs. It is safe for concurrent use: DetectFromIP is called from the capture
// hot path (one goroutine), while Current/LastPacketReceived may be polled from
// others.
//
// State machine (mirrors AlbionServerDetectionService):
//
//   - A new region becomes Current only after being observed as a stable
//     candidate for StabilityDuration (default 5s).
//   - When a packet from a *different* known region arrives while a region is
//     already current, the detector first resets Current to Unknown (emitting a
//     change event) and then begins observing the new candidate. This avoids
//     attributing data to the old region while the new one is unconfirmed.
type Detector struct {
	// nowFunc supplies "current" time. Defaults to time.Now; overridable for
	// deterministic tests.
	nowFunc func() time.Time

	stability time.Duration

	mu                 sync.Mutex
	current            ServerInfo
	candidate          ServerInfo
	candidateFirstSeen time.Time
	lastPacketRecv     time.Time

	// onChange is invoked outside the detector's mutex (after releasing it) so
	// handlers may safely re-enter Current() without deadlocking. Nil = no-op.
	onChange func(ChangeEvent)
}

// Option configures a Detector at construction.
type Option func(*Detector)

// WithNowFunc overrides the time source (for tests).
func WithNowFunc(fn func() time.Time) Option {
	return func(d *Detector) {
		if fn != nil {
			d.nowFunc = fn
		}
	}
}

// WithStability overrides the candidate stability window. A shorter duration
// flips regions faster (useful in tests); a longer one is more conservative.
func WithStability(dur time.Duration) Option {
	return func(d *Detector) { d.stability = dur }
}

// WithOnChange registers a callback invoked after every confirmed transition.
// The callback runs on the same goroutine as DetectFromIP (the capture path),
// so it must be non-blocking; persist/IO work should be dispatched elsewhere.
func WithOnChange(fn func(ChangeEvent)) Option {
	return func(d *Detector) { d.onChange = fn }
}

// NewDetector constructs a Detector with the given options applied.
func NewDetector(opts ...Option) *Detector {
	d := &Detector{
		nowFunc:   time.Now,
		stability: DefaultStabilityDuration,
		current:   unknown,
		candidate: unknown,
	}
	for _, opt := range opts {
		opt(d)
	}
	return d
}

// Current returns the currently active server (Unknown if none is confirmed).
func (d *Detector) Current() ServerInfo {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.current
}

// LastPacketReceived returns the time of the last packet that matched a known
// region. Zero if no such packet has ever been observed.
func (d *Detector) LastPacketReceived() time.Time {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.lastPacketRecv
}

// DetectFromIP feeds one observed server IP to the detector. The IP is the
// Photon server's address — for incoming packets (server→client) this is the
// source IP, for outgoing packets (client→server) it is the destination IP.
// Callers should pass whichever endpoint belongs to the server.
//
// Non-matching IPs (Unknown region, nil, IPv6, or unrecognized prefix) are
// ignored entirely: they carry no region signal and must not perturb the
// current state.
func (d *Detector) DetectFromIP(ip net.IP) {
	detected := MatchByIP(ip)
	if detected.Location == ServerLocationUnknown {
		return
	}

	var event ChangeEvent
	hasEvent := false

	now := d.nowFunc()

	func() {
		d.mu.Lock()
		defer d.mu.Unlock()

		d.lastPacketRecv = now

		// Already confident about this region: reset any in-flight candidate
		// (a previous, unconfirmed different region) and stay put.
		if d.current.Location == detected.Location {
			d.candidate = unknown
			d.candidateFirstSeen = time.Time{}
			return
		}

		if d.current.Location != ServerLocationUnknown {
			// We were confident about a *different* region. Drop confidence
			// immediately and begin observing the new candidate.
			event = ChangeEvent{Previous: d.current, Current: unknown}
			hasEvent = true
			d.current = unknown
			d.candidate = detected
			d.candidateFirstSeen = now
			return
		}

		// Current is Unknown: try to promote a stable candidate.
		if promoted, ok := d.tryPromoteStableCandidate(detected, now); ok {
			event = promoted
			hasEvent = true
		}
	}()

	if hasEvent && d.onChange != nil {
		d.onChange(event)
	}
}

// tryPromoteStableCandidate promotes the candidate to current once it has been
// observed for at least StabilityDuration. The caller MUST hold d.mu.
//
// If the incoming region differs from the pending candidate, the candidate is
// replaced and the stability clock restarts (returns ok=false). If the same
// candidate has been observed long enough, it is promoted and a change event
// is returned (ok=true).
func (d *Detector) tryPromoteStableCandidate(detected ServerInfo, now time.Time) (ChangeEvent, bool) {
	if d.candidate.Location != detected.Location {
		d.candidate = detected
		d.candidateFirstSeen = now
		return ChangeEvent{}, false
	}

	if now.Sub(d.candidateFirstSeen) < d.stability {
		return ChangeEvent{}, false
	}

	previous := d.current
	d.current = detected
	d.candidate = unknown
	d.candidateFirstSeen = time.Time{}
	return ChangeEvent{Previous: previous, Current: detected}, true
}
