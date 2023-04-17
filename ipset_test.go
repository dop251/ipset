package ipset

import (
	"net/netip"
	"testing"
)

func TestIPSet_Add(t *testing.T) {
	var s IPSet
	s.Add(netip.MustParsePrefix("127.0.0.0/8"))
	s.Add(netip.MustParsePrefix("2603:C000::/24"))
	if !s.Contains(netip.MustParseAddr("127.0.0.1")) {
		t.Fatal()
	}
	if !s.Contains(netip.MustParseAddr("2603:C000::4")) {
		t.Fatal()
	}
	if s.Contains(netip.MustParseAddr("8.8.8.8")) {
		t.Fatal()
	}
	if s.Contains(netip.MustParseAddr("2603:D000::1")) {
		t.Fatal()
	}

	var invalid netip.Addr
	if s.Contains(invalid) {
		t.Fatal()
	}
}
