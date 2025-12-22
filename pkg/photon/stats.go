// Package photon implements the Photon Engine network protocol parser.
package photon

import (
	"fmt"
	"sync/atomic"
	"time"
)

// Stats maintains statistics for the Photon parser.
// All counters are thread-safe and can be accessed concurrently.
type Stats struct {
	// Packet counters
	PacketsReceived  uint64 // Total UDP packets received
	PacketsProcessed uint64 // Packets successfully processed
	PacketsEncrypted uint64 // Encrypted packets (skipped)
	PacketsWithCRC   uint64 // Packets with CRC enabled
	PacketsMalformed uint64 // Malformed/corrupted packets

	// Fragment counters
	FragmentsReceived  uint64 // Individual fragments received
	FragmentsCompleted uint64 // Fragmented packets successfully reassembled
	FragmentsExpired   uint64 // Fragments expired by TTL cleanup

	// Message counters
	EventsDecoded    uint64 // Game events decoded
	RequestsDecoded  uint64 // Operation requests decoded
	ResponsesDecoded uint64 // Operation responses decoded

	// Byte counters
	BytesReceived uint64 // Total bytes received

	// Timing
	StartTime      time.Time // When the parser started
	LastPacketTime time.Time // Timestamp of last packet received
}

// NewStats creates a new Stats instance with StartTime set to now.
func NewStats() *Stats {
	return &Stats{
		StartTime: time.Now(),
	}
}

// ============================================
// Thread-safe incrementers (using atomic)
// ============================================

// IncrPacketsReceived increments the packets received counter.
func (s *Stats) IncrPacketsReceived() {
	atomic.AddUint64(&s.PacketsReceived, 1)
}

// IncrPacketsProcessed increments the packets processed counter.
func (s *Stats) IncrPacketsProcessed() {
	atomic.AddUint64(&s.PacketsProcessed, 1)
}

// IncrPacketsEncrypted increments the encrypted packets counter.
func (s *Stats) IncrPacketsEncrypted() {
	atomic.AddUint64(&s.PacketsEncrypted, 1)
}

// IncrPacketsWithCRC increments the CRC packets counter.
func (s *Stats) IncrPacketsWithCRC() {
	atomic.AddUint64(&s.PacketsWithCRC, 1)
}

// IncrPacketsMalformed increments the malformed packets counter.
func (s *Stats) IncrPacketsMalformed() {
	atomic.AddUint64(&s.PacketsMalformed, 1)
}

// IncrFragmentsReceived increments the fragments received counter.
func (s *Stats) IncrFragmentsReceived() {
	atomic.AddUint64(&s.FragmentsReceived, 1)
}

// IncrFragmentsCompleted increments the fragments completed counter.
func (s *Stats) IncrFragmentsCompleted() {
	atomic.AddUint64(&s.FragmentsCompleted, 1)
}

// IncrFragmentsExpired increments the fragments expired counter.
func (s *Stats) IncrFragmentsExpired() {
	atomic.AddUint64(&s.FragmentsExpired, 1)
}

// IncrEventsDecoded increments the events decoded counter.
func (s *Stats) IncrEventsDecoded() {
	atomic.AddUint64(&s.EventsDecoded, 1)
}

// IncrRequestsDecoded increments the requests decoded counter.
func (s *Stats) IncrRequestsDecoded() {
	atomic.AddUint64(&s.RequestsDecoded, 1)
}

// IncrResponsesDecoded increments the responses decoded counter.
func (s *Stats) IncrResponsesDecoded() {
	atomic.AddUint64(&s.ResponsesDecoded, 1)
}

// AddBytesReceived adds n bytes to the bytes received counter.
func (s *Stats) AddBytesReceived(n uint64) {
	atomic.AddUint64(&s.BytesReceived, n)
}

// ============================================
// Thread-safe getters
// ============================================

// GetPacketsReceived returns the packets received count.
func (s *Stats) GetPacketsReceived() uint64 {
	return atomic.LoadUint64(&s.PacketsReceived)
}

// GetPacketsProcessed returns the packets processed count.
func (s *Stats) GetPacketsProcessed() uint64 {
	return atomic.LoadUint64(&s.PacketsProcessed)
}

// GetPacketsEncrypted returns the encrypted packets count.
func (s *Stats) GetPacketsEncrypted() uint64 {
	return atomic.LoadUint64(&s.PacketsEncrypted)
}

// GetPacketsWithCRC returns the CRC packets count.
func (s *Stats) GetPacketsWithCRC() uint64 {
	return atomic.LoadUint64(&s.PacketsWithCRC)
}

// GetPacketsMalformed returns the malformed packets count.
func (s *Stats) GetPacketsMalformed() uint64 {
	return atomic.LoadUint64(&s.PacketsMalformed)
}

// GetFragmentsReceived returns the fragments received count.
func (s *Stats) GetFragmentsReceived() uint64 {
	return atomic.LoadUint64(&s.FragmentsReceived)
}

// GetFragmentsCompleted returns the fragments completed count.
func (s *Stats) GetFragmentsCompleted() uint64 {
	return atomic.LoadUint64(&s.FragmentsCompleted)
}

// GetFragmentsExpired returns the fragments expired count.
func (s *Stats) GetFragmentsExpired() uint64 {
	return atomic.LoadUint64(&s.FragmentsExpired)
}

// GetEventsDecoded returns the events decoded count.
func (s *Stats) GetEventsDecoded() uint64 {
	return atomic.LoadUint64(&s.EventsDecoded)
}

// GetRequestsDecoded returns the requests decoded count.
func (s *Stats) GetRequestsDecoded() uint64 {
	return atomic.LoadUint64(&s.RequestsDecoded)
}

// GetResponsesDecoded returns the responses decoded count.
func (s *Stats) GetResponsesDecoded() uint64 {
	return atomic.LoadUint64(&s.ResponsesDecoded)
}

// GetBytesReceived returns the bytes received count.
func (s *Stats) GetBytesReceived() uint64 {
	return atomic.LoadUint64(&s.BytesReceived)
}

// ============================================
// Calculation methods
// ============================================

// Uptime returns how long the parser has been running.
func (s *Stats) Uptime() time.Duration {
	return time.Since(s.StartTime)
}

// PacketsPerSecond calculates the packet rate per second.
func (s *Stats) PacketsPerSecond() float64 {
	uptime := s.Uptime().Seconds()
	if uptime == 0 {
		return 0
	}
	return float64(s.GetPacketsReceived()) / uptime
}

// EventsPerSecond calculates the event rate per second.
func (s *Stats) EventsPerSecond() float64 {
	uptime := s.Uptime().Seconds()
	if uptime == 0 {
		return 0
	}
	return float64(s.GetEventsDecoded()) / uptime
}

// ============================================
// Formatting methods
// ============================================

// Summary returns a one-line formatted summary.
func (s *Stats) Summary() string {
	return fmt.Sprintf(
		"Uptime: %s | Packets: %d (%.1f/s) | Events: %d | Encrypted: %d | CRC: %d",
		s.FormatUptime(),
		s.GetPacketsReceived(),
		s.PacketsPerSecond(),
		s.GetEventsDecoded(),
		s.GetPacketsEncrypted(),
		s.GetPacketsWithCRC(),
	)
}

// FormatUptime returns the uptime formatted as HH:MM:SS.
func (s *Stats) FormatUptime() string {
	d := s.Uptime()
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}

// Reset zeroes all counters and resets StartTime to now.
func (s *Stats) Reset() {
	atomic.StoreUint64(&s.PacketsReceived, 0)
	atomic.StoreUint64(&s.PacketsProcessed, 0)
	atomic.StoreUint64(&s.PacketsEncrypted, 0)
	atomic.StoreUint64(&s.PacketsWithCRC, 0)
	atomic.StoreUint64(&s.PacketsMalformed, 0)
	atomic.StoreUint64(&s.FragmentsReceived, 0)
	atomic.StoreUint64(&s.FragmentsCompleted, 0)
	atomic.StoreUint64(&s.FragmentsExpired, 0)
	atomic.StoreUint64(&s.EventsDecoded, 0)
	atomic.StoreUint64(&s.RequestsDecoded, 0)
	atomic.StoreUint64(&s.ResponsesDecoded, 0)
	atomic.StoreUint64(&s.BytesReceived, 0)
	s.StartTime = time.Now()
}
