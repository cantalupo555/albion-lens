package serverdetect

import (
	"net"
	"testing"
)

func TestServerLocationIsKnown(t *testing.T) {
	known := []ServerLocation{ServerLocationAmerica, ServerLocationAsia, ServerLocationEurope}
	for _, loc := range known {
		if !loc.IsKnown() {
			t.Errorf("%v.IsKnown() = false, want true", loc)
		}
	}
	if ServerLocationUnknown.IsKnown() {
		t.Errorf("Unknown.IsKnown() = true, want false")
	}
}

func TestServerLocationString(t *testing.T) {
	cases := map[ServerLocation]string{
		ServerLocationAmerica: "Americas",
		ServerLocationAsia:    "Asia",
		ServerLocationEurope:  "Europe",
		ServerLocationUnknown: "Unknown",
		// any other value is not valid enum input but should not panic
		ServerLocation(99): "Unknown",
	}
	for loc, want := range cases {
		if got := loc.String(); got != want {
			t.Errorf("%v.String() = %q, want %q", loc, got, want)
		}
	}
}

func TestServerLocationDirName(t *testing.T) {
	cases := map[ServerLocation]string{
		ServerLocationAmerica: "UserData-AMERICA",
		ServerLocationAsia:    "UserData-ASIA",
		ServerLocationEurope:  "UserData-EUROPE",
		ServerLocationUnknown: "",
	}
	for loc, want := range cases {
		if got := loc.DirName(); got != want {
			t.Errorf("%v.DirName() = %q, want %q", loc, got, want)
		}
	}
}

func TestMatchByIP(t *testing.T) {
	cases := []struct {
		name string
		ip   string
		want ServerLocation
	}{
		{"Americas full", "5.188.125.10", ServerLocationAmerica},
		{"Americas boundary", "5.188.125.255", ServerLocationAmerica},
		{"Asia", "5.45.187.5", ServerLocationAsia},
		{"Europe", "193.169.238.42", ServerLocationEurope},
		{"unknown prefix", "8.8.8.8", ServerLocationUnknown},
		// Prefix must be a real prefix — 5.188.12 is NOT 5.188.125.
		{"partial non-match", "5.188.12.9", ServerLocationUnknown},
		// Prefix shares the leading octet but diverges immediately after.
		{"shared first octet", "5.188.200.1", ServerLocationUnknown},
		{"loopback", "127.0.0.1", ServerLocationUnknown},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := MatchByIPString(tc.ip)
			if got.Location != tc.want {
				t.Errorf("MatchByIPString(%q) = %v, want %v", tc.ip, got.Location, tc.want)
			}
		})
	}
}

func TestMatchByIPNil(t *testing.T) {
	if got := MatchByIP(nil); got.Location != ServerLocationUnknown {
		t.Errorf("MatchByIP(nil) = %v, want Unknown", got.Location)
	}
}

func TestMatchByIPV6Ignored(t *testing.T) {
	// A v4-mapped IPv6 address (::ffff:a.b.c.d) represents an IPv4 connection
	// and is normalized by To4(), so it IS matched to the underlying region.
	if got := MatchByIPString("::ffff:5.188.125.10"); got.Location != ServerLocationAmerica {
		t.Errorf("MatchByIP(v4-mapped v6) = %v, want Americas (normalized to v4)", got.Location)
	}
	// A native IPv6 address does not match any known (IPv4-only) prefix.
	if got := MatchByIPString("2001:db8::1"); got.Location != ServerLocationUnknown {
		t.Errorf("MatchByIP(native v6) = %v, want Unknown", got.Location)
	}
}

func TestMatchByIPStringInvalid(t *testing.T) {
	cases := []string{"", "not-an-ip", "999.999.999.999", "5.188"}
	for _, s := range cases {
		if got := MatchByIPString(s); got.Location != ServerLocationUnknown {
			t.Errorf("MatchByIPString(%q) = %v, want Unknown", s, got.Location)
		}
	}
}

func TestMatchByIPNetIPType(t *testing.T) {
	// Verify the net.IP-based entrypoint behaves like the string one.
	ip := net.IPv4(5, 188, 125, 10)
	if got := MatchByIP(ip); got.Location != ServerLocationAmerica {
		t.Errorf("MatchByIP(net.IPv4(...)) = %v, want Americas", got.Location)
	}
	// A 16-byte v4 representation (v4-in-v6) must still match.
	if got := MatchByIP(ip.To16()); got.Location != ServerLocationAmerica {
		t.Errorf("MatchByIP(v4-in-v6) = %v, want Americas", got.Location)
	}
}

func TestServersReturnsCopy(t *testing.T) {
	s1 := Servers()
	s1[0] = ServerInfo{Location: ServerLocationUnknown, Name: "tampered", IpPrefix: "0.0.0"}
	s2 := Servers()
	// Mutating the returned slice must not affect subsequent calls.
	if s2[0].Location == ServerLocationUnknown {
		t.Fatalf("Servers() returned a slice that shared backing storage")
	}
	if len(s2) != 3 {
		t.Errorf("len(Servers()) = %d, want 3", len(s2))
	}
}

func TestUnknownSentinel(t *testing.T) {
	if Unknown().Location != ServerLocationUnknown {
		t.Fatalf("Unknown().Location = %v, want Unknown", Unknown().Location)
	}
}
