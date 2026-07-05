package serverdetect

import (
	"net"
	"strings"
)

// ServerInfo describes a known Albion Online game server. It pairs a region
// with the IPv4 prefix the server's Photon endpoints publish. Mirrors the
// reference's AlbionServerInfo.
type ServerInfo struct {
	Location ServerLocation
	// Name is the human-readable region label (e.g. "Americas").
	Name string
	// IpPrefix is the dotted-quad prefix the region's Photon servers share
	// (e.g. "5.188.125"). Matched as a string prefix against an IP's textual
	// form, exactly like the reference's sourceIp.StartsWith(prefix).
	IpPrefix string
}

// unknown is the zero-match result returned when an IP does not map to any
// known region (including when the input is nil or not an IPv4 address).
var unknown = ServerInfo{}

// servers is the authoritative list of known Albion Photon server regions.
// The prefixes are maintained by the AlbionDataProject and used by the
// reference StatisticsAnalysisTool. If a region's IPs change, update them here.
//
// To refresh: https://github.com/ao-data/albiondata  (server IP map).
var servers = []ServerInfo{
	{ServerLocationAmerica, "Americas", "5.188.125"},
	{ServerLocationAsia, "Asia", "5.45.187"},
	{ServerLocationEurope, "Europe", "193.169.238"},
}

// Unknown returns the sentinel "no match" ServerInfo. It is the zero value and
// carries ServerLocationUnknown.
func Unknown() ServerInfo { return unknown }

// Servers returns the known server list (a copy of the backing slice header) so
// callers can enumerate regions for migrations or diagnostics. The returned
// slice must not be mutated.
func Servers() []ServerInfo {
	out := make([]ServerInfo, len(servers))
	copy(out, servers)
	return out
}

// MatchByIP resolves a server IP to its ServerInfo. Returns Unknown if ip is
// nil, is not a 4-byte / 16-byte IPv4 representation, or does not start with
// any known prefix.
//
// The match is a string prefix over the canonical dotted-quad form (e.g.
// "5.188.125.42" starts with "5.188.125"), matching the reference's
// AlbionServerRegistry.GetBySourceIp.
func MatchByIP(ip net.IP) ServerInfo {
	if ip == nil {
		return unknown
	}
	// Normalize: To4 returns a 4-byte IPv4 if the input is one (or a v4-in-v6
	// form), nil otherwise. We deliberately ignore IPv6 endpoints since the
	// known prefixes are IPv4 only.
	v4 := ip.To4()
	if v4 == nil {
		return unknown
	}
	s := v4.String()
	for _, srv := range servers {
		if strings.HasPrefix(s, srv.IpPrefix) {
			return srv
		}
	}
	return unknown
}

// MatchByIPString is a convenience over MatchByIP that accepts a textual IP.
// Invalid addresses yield Unknown rather than an error, since detection is
// best-effort: a malformed source IP simply means "no signal this packet".
func MatchByIPString(s string) ServerInfo {
	if s == "" {
		return unknown
	}
	ip := net.ParseIP(s)
	if ip == nil {
		return unknown
	}
	return MatchByIP(ip)
}
