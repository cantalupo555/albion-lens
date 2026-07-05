// Package backend provides a unified service layer for Albion Online packet capture and event processing.
package backend

import "github.com/cantalupo555/albion-lens/internal/serverdetect"

// Option configures the Service using functional options pattern
type Option func(*Service)

// WithDevice sets the network device to capture from
func WithDevice(device string) Option {
	return func(s *Service) {
		s.device = device
	}
}

// WithDebug enables debug output in the handler
func WithDebug(debug bool) Option {
	return func(s *Service) {
		s.debug = debug
	}
}

// WithDiscovery enables discovery mode in the handler
func WithDiscovery(discovery bool) Option {
	return func(s *Service) {
		s.discovery = discovery
	}
}

// WithItemDatabasePath sets the path to the ao-bin-dumps item database
func WithItemDatabasePath(path string) Option {
	return func(s *Service) {
		s.itemDBPath = path
	}
}

// WithBPFFilter sets a custom BPF filter for packet capture
func WithBPFFilter(filter string) Option {
	return func(s *Service) {
		s.bpfFilter = filter
	}
}

// WithEventBufferSize sets the buffer size for the events channel
func WithEventBufferSize(size int) Option {
	return func(s *Service) {
		s.eventBufferSize = size
	}
}

// WithStatsBufferSize sets the buffer size for the stats channel
func WithStatsBufferSize(size int) Option {
	return func(s *Service) {
		s.statsBufferSize = size
	}
}

// WithDetectorOptions configures the server-region detector. The service still
// appends its own OnChange callback (which forwards to ServerChanged), so
// callers should not set their own OnChange here. Useful for tests that need a
// shorter stability window or a custom nowFunc.
func WithDetectorOptions(opts ...serverdetect.Option) Option {
	return func(s *Service) {
		s.detectorOpts = append(s.detectorOpts, opts...)
	}
}
