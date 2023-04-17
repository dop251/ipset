package ipset

import (
	"bufio"
	"errors"
	"net/netip"
	"os"
	"testing"
)

func TestIPv6Add(t *testing.T) {
	var s IPSet6
	s.Add(netip.MustParseAddr("2603:C000::").As16(), 24)
	s.Add(netip.MustParseAddr("2a01:4f8::").As16(), 29)

	if !s.Contains(netip.MustParseAddr("2603:C000::4").As16()) {
		t.Fatal()
	}

	if !s.Contains(netip.MustParseAddr("2a01:4f8::2").As16()) {
		t.Fatal()
	}
}

func TestIPv6Add1(t *testing.T) {
	var s IPSet6
	s.Add(netip.MustParseAddr("FFFF:FFFF:FFFF:FFF1::").As16(), 64)
	s.Add(netip.MustParseAddr("FFFF:FFFF:FFEF:FFF2::").As16(), 64)

	if !s.Contains(netip.MustParseAddr("FFFF:FFFF:FFFF:FFF1::1").As16()) {
		t.Fatal()
	}

	if !s.Contains(netip.MustParseAddr("FFFF:FFFF:FFEF:FFF2::1").As16()) {
		t.Fatal()
	}

	if s.Contains(netip.MustParseAddr("FFFF:FFFF:FFDF:FFF1::1").As16()) {
		t.Fatal()
	}
}

func TestIPv6Add2(t *testing.T) {
	var s IPSet6
	s.Add(netip.MustParseAddr("2001:668:0:2::1:5111").As16(), 128)
	s.Add(netip.MustParseAddr("2001:668:0:2:ffff:0:5995:800d").As16(), 128)
	s.Add(netip.MustParseAddr("2001:668:0:2:ffff:0:5995:8016").As16(), 128)

	if !s.Contains(netip.MustParseAddr("2001:668:0:2:ffff:0:5995:800d").As16()) {
		t.Fatal()
	}

	if !s.Contains(netip.MustParseAddr("2001:668:0:2:ffff:0:5995:8016").As16()) {
		t.Fatal()
	}

	if s.Contains(netip.MustParseAddr("FFFF:FFFF:FFDF:FFF1::1").As16()) {
		t.Fatal()
	}
}

func TestIPv6Large(t *testing.T) {
	f, err := os.Open("testdata/US_ipv6.txt")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			t.Skip("testdata/US_ipv6.txt is missing. Run 'go generate' to create it")
		}
		t.Fatal(err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Split(bufio.ScanLines)

	var s IPSet6

	var list []netip.Prefix

	for scanner.Scan() {
		p := netip.MustParsePrefix(scanner.Text())
		list = append(list, p)
		s.Add(p.Addr().As16(), uint32(p.Bits()))
	}
	t.Log(len(s.nodes))

	for n, p := range list {
		_ = n
		if !s.Contains(p.Addr().As16()) {
			t.Fatal(p)
		}
	}

	var st stats
	s.gatherStats(s.nodes[1], &st)
	t.Logf("%#v", st)
}
