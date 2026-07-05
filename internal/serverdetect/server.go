// Package serverdetect identifies the active Albion Online server region
// (Americas/Europe/Asia) from the Photon server's IP address, mirroring the
// reference repository's AlbionServerRegistry + AlbionServerDetectionService.
//
// The region drives per-server data segregation on disk (UserData-{SERVER}
// subdirectories) so a player who plays on multiple regions keeps their stats
// isolated per region.
package serverdetect

// ServerLocation identifies an Albion Online game-server region. Values mirror
// StatisticsAnalysisTool's ServerLocation enum.
type ServerLocation int

const (
	ServerLocationUnknown ServerLocation = 0
	ServerLocationAmerica ServerLocation = 1
	ServerLocationAsia    ServerLocation = 2
	ServerLocationEurope  ServerLocation = 3
)

// IsKnown reports whether the location is one of the concrete game regions
// (not Unknown). Used by callers to decide whether per-region paths apply.
func (s ServerLocation) IsKnown() bool {
	switch s {
	case ServerLocationAmerica, ServerLocationAsia, ServerLocationEurope:
		return true
	}
	return false
}

// String returns the human-readable region name, or "Unknown".
func (s ServerLocation) String() string {
	switch s {
	case ServerLocationAmerica:
		return "Americas"
	case ServerLocationAsia:
		return "Asia"
	case ServerLocationEurope:
		return "Europe"
	default:
		return "Unknown"
	}
}

// DirName returns the on-disk subdirectory name for a region, matching the
// reference's UserData-{REGION} convention. Returns "" for Unknown so callers
// can treat it as "no per-region subdir" (the legacy/shared directory).
func (s ServerLocation) DirName() string {
	switch s {
	case ServerLocationAmerica:
		return "UserData-AMERICA"
	case ServerLocationAsia:
		return "UserData-ASIA"
	case ServerLocationEurope:
		return "UserData-EUROPE"
	default:
		return ""
	}
}
