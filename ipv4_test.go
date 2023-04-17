package ipset

import (
	"bufio"
	"encoding/binary"
	"errors"
	"math/rand"
	"net/netip"
	"os"
	"testing"
)

func TestIPSetAdd(t *testing.T) {
	var s IPSet4
	s.Add(0x7F00_0000, 8)
	if !s.Contains(0x7F00_0001) {
		t.Fatal("false")
	}
	s.Add(0x7001_0203, 32)
	s.Add(0x8000_0000, 8)
	s.Add(0x0A00_0000, 8)
	if !s.Contains(0x7F00_0001) {
		t.Fatal("false")
	}
	if !s.Contains(0x0A00_0002) {
		t.Fatal("false")
	}
	if s.Contains(0x0B00_0002) {
		t.Fatal("false")
	}
	if !s.Contains(0x7001_0203) {
		t.Fatal("false")
	}
}

func TestIPSetAdd1(t *testing.T) {
	var s IPSet4
	s.Add(0xF000_0000, 4)
	s.Add(0xE001_0000, 16)
	if !s.Contains(0xE001_0001) {
		t.Fatal()
	}
	if !s.Contains(0xF001_0001) {
		t.Fatal()
	}
}

func TestIPSetAdd2(t *testing.T) {
	var s IPSet4
	s.Add(0xFFFF_0100, 24)
	s.Add(0xFFFF_0300, 22)
	if !s.Contains(0xFFFF_0101) {
		t.Fatal()
	}
	if !s.Contains(0xFFFF_0301) {
		t.Fatal()
	}
}

func TestIPSetAdd3(t *testing.T) {
	var s IPSet4
	s.Add(0xFFFF_2800, 22)
	s.Add(0xFFFF_3000, 22)
	s.Add(0xFFFF_5000, 22)
	if !s.Contains(0xFFFF_2801) {
		t.Fatal()
	}
}

func TestIPSetAdd4(t *testing.T) {
	var s IPSet4
	s.Add(0x0102_0301, 32)
	s.Add(0x0102_0302, 32)
	if !s.Contains(0x0102_0301) {
		t.Fatal()
	}
	if !s.Contains(0x0102_0302) {
		t.Fatal()
	}
	if s.Contains(0x0102_0303) {
		t.Fatal()
	}
}

func TestIPSetFreeNode(t *testing.T) {
	var s IPSet4
	s.Add(0x0102_0301, 32)
	s.Add(0x0100_0000, 8)
	s.Add(0x0200_0000, 8)

	if !s.Contains(0x0102_0301) {
		t.Fatal()
	}
	if !s.Contains(0x0100_0301) {
		t.Fatal()
	}
	if !s.Contains(0x0200_0001) {
		t.Fatal()
	}
}

func TestMerge(t *testing.T) {
	var s IPSet4
	s.Add(0xF000_0000, 16)
	s.Add(0xFF80_0000, 9)
	s.Add(0xFF00_0000, 9)
	s.Compact()
	if len(s.nodes) > 10 {
		t.Fatal("Too many nodes")
	}
}

func TestMerge1(t *testing.T) {
	var s IPSet4
	s.Add(0xF000_0000, 32)
	s.Add(0xF000_0001, 32)
	s.Compact()
	if len(s.nodes) > 4 {
		t.Fatal("Too many nodes")
	}
}

func TestMerge2(t *testing.T) {
	var s IPSet4
	s.Add(0xF000_0000, 32)
	s.Add(0xF000_0101, 32)
	if len(s.nodes) > 10 {
		t.Fatal("Too many nodes")
	}
	if !s.Contains(0xF000_0000) {
		t.Fatal()
	}
	if !s.Contains(0xF000_0101) {
		t.Fatal()
	}

	s.Add(0xF000_0100, 24)
	s.Compact()
	if len(s.nodes) > 8 {
		t.Fatal("Too many nodes")
	}
	if !s.Contains(0xF000_0000) {
		t.Fatal()
	}
	if !s.Contains(0xF000_0101) {
		t.Fatal()
	}
	if !s.Contains(0xF000_0102) {
		t.Fatal()
	}
}

func ipToUint(b [4]byte) uint32 {
	return binary.BigEndian.Uint32(b[:])
}

type stats struct {
	skipNodes         int
	skipPrefixLengths [maxPackablePrefixLen + 1]int
}

func (s *ipsetBase) gatherStats(ptr uint32, st *stats) {
	if ptr == ptrAbsent || ptr == ptrPresent {
		return
	}
	idx := ptrToIdx(ptr)
	if ptr&skipNodeMask != 0 {
		st.skipNodes++
		_, l := unpackPrefixLen(s.nodes[idx])
		st.skipPrefixLengths[l]++
		s.gatherStats(s.nodes[idx+1], st)
	} else {
		s.gatherStats(s.nodes[idx], st)
		s.gatherStats(s.nodes[idx+1], st)
	}
}

func TestIPv4Large(t *testing.T) {
	f, err := os.Open("testdata/US_ipv4.txt")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			t.Skip("testdata/US_ipv4.txt is missing. Run 'go generate' to create it")
		}
		t.Fatal(err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Split(bufio.ScanLines)

	var s IPSet4

	var list []netip.Prefix

	for scanner.Scan() {
		p := netip.MustParsePrefix(scanner.Text())
		list = append(list, p)
		s.Add(ipToUint(p.Addr().As4()), uint32(p.Bits()))
	}
	t.Log(len(s.nodes))

	for _, p := range list {
		if !s.Contains(ipToUint(p.Addr().As4())) {
			t.Fatal(p)
		}
	}

	rs := rand.New(rand.NewSource(12345678901234567))
	var ip [4]byte
	hits := 0
	for i := 0; i < 100_000; i++ {
		ipNum := rs.Uint32()
		binary.BigEndian.PutUint32(ip[:], ipNum)
		addr := netip.AddrFrom4(ip)
		contains := false
		if addr.IsGlobalUnicast() && !addr.IsPrivate() {
			for _, p := range list {
				if p.Contains(addr) {
					contains = true
					hits++
					break
				}
			}
		}

		if s.Contains(ipNum) != contains {
			t.Fatal(addr)
		}
	}

	var st stats
	s.gatherStats(s.nodes[1], &st)
	t.Log("Hits: ", hits)
	t.Logf("%#v", st)
}

/*
func TestUnalignedAccess(t *testing.T) {
	a := []byte{1, 2, 3, 4, 5}
	aa := a[1:]
	b := (*(*[]uint32)(unsafe.Pointer(&aa)))[:1:1]
	t.Logf("%x", uintptr(unsafe.Pointer(&a[0]))&3)
	t.Logf("%x", uintptr(unsafe.Pointer(&aa[0]))&3)
	t.Logf("%x", uintptr(unsafe.Pointer(&b[0]))&3)
	t.Logf("%x", b[0])
	b[0] = 123
	t.Log(a)
}
*/

func BenchmarkContainsLarge(b *testing.B) {
	f, err := os.Open("testdata/US_ipv4.txt")
	if err != nil {
		b.Fatal(err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Split(bufio.ScanLines)

	var s IPSet4

	for scanner.Scan() {
		p := netip.MustParsePrefix(scanner.Text())
		s.Add(ipToUint(p.Addr().As4()), uint32(p.Bits()))
	}

	a := ipToUint([4]byte{170, 247, 92, 1})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Contains(a)
	}
}

func BenchmarkAddLarge(b *testing.B) {
	f, err := os.Open("testdata/US_ipv4.txt")
	if err != nil {
		b.Fatal(err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Split(bufio.ScanLines)
	var prefixes []uint32
	for scanner.Scan() {
		p := netip.MustParsePrefix(scanner.Text())
		prefixes = append(prefixes, ipToUint(p.Addr().As4()), uint32(p.Bits()))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var s IPSet4
		for j := 0; j < len(prefixes); j += 2 {
			s.Add(prefixes[j], prefixes[j+1])
		}
	}
}
