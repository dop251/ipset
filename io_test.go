package ipset

import (
	"bytes"
	"math/rand"
	"net/netip"
	"os"
	"testing"
	"unsafe"
)

func TestReadWrite(t *testing.T) {
	var s IPSet4
	s.Add(0x7001_0203, 32)
	s.Add(0x8000_0000, 8)
	s.Add(0x0A00_0000, 8)
	var b bytes.Buffer
	err := s.Serialize(&b)
	if err != nil {
		t.Fatal(err)
	}

	b1 := bytes.NewBuffer(b.Bytes())
	var s1 IPSet4
	err = s1.Deserialize(b1)
	if err != nil {
		t.Fatal(err)
	}

	if !s1.Contains(0x0A00_0002) {
		t.Fatal()
	}
	if s1.Contains(0x0B00_0002) {
		t.Fatal()
	}
	if !s1.Contains(0x7001_0203) {
		t.Fatal()
	}
}

func Test_writeBytesGeneric(t *testing.T) {
	var s IPSet4
	s.Add(0x7001_0203, 32)
	s.Add(0x8000_0000, 8)
	s.Add(0x0A00_0000, 8)
	var b bytes.Buffer
	s.nodes[0] = uint32(len(s.nodes)) * 4
	err := s.writeBytesGeneric(&b)
	if err != nil {
		t.Fatal(err)
	}

	b1 := bytes.NewBuffer(b.Bytes())
	var s1 IPSet4
	err = s1.Deserialize(b1)
	if err != nil {
		t.Fatal(err)
	}

	if !s1.Contains(0x0A00_0002) {
		t.Fatal()
	}
	if s1.Contains(0x0B00_0002) {
		t.Fatal()
	}
	if !s1.Contains(0x7001_0203) {
		t.Fatal()
	}
}

func TestIpsetBase_writeBytesGenericLarge(t *testing.T) {
	var s IPSet4
	rs := rand.New(rand.NewSource(12345678901234567))

	for i := uint32(0); i < 400; i++ {
		s.Add(rs.Uint32(), 32)
	}
	var b bytes.Buffer
	s.nodes[0] = uint32(len(s.nodes)) * 4
	err := s.writeBytesGeneric(&b)
	if err != nil {
		t.Fatal(err)
	}
	if b.Len() <= 4096 {
		t.Fatal("This test needs a set more than 4096 bytes in length")
	}
	b1 := bytes.NewBuffer(b.Bytes())
	var s1 IPSet4
	err = s1.Deserialize(b1)
	if err != nil {
		t.Fatal(err)
	}

	rs = rand.New(rand.NewSource(12345678901234567))

	for i := uint32(0); i < 400; i++ {
		ip := rs.Uint32()
		if !s.Contains(ip) {
			t.Fatal(ip)
		}
	}
}

func TestIpsetBase_loadBytesForeign(t *testing.T) {
	var s ipsetBase
	s.loadBytesForeign([]byte{1, 2, 3, 4})

	// Make sure byte order is reversed
	if *(*byte)(unsafe.Pointer(&s.nodes[0])) != 0x4 {
		t.Fatal(s.nodes[0])
	}
}

func TestIpsetBase_loadBytesNative(t *testing.T) {
	var s ipsetBase
	s.loadBytesNative([]byte{1, 2, 3, 4})

	// Make sure byte order is preserved
	if *(*byte)(unsafe.Pointer(&s.nodes[0])) != 0x1 {
		t.Fatal(s.nodes[0])
	}
}

func TestIpsetBase_bytesGeneric(t *testing.T) {
	var s ipsetBase
	s.nodes = []uint32{0x0102_0304}

	b := s.bytesGeneric()

	if b[0] != 4 {
		t.Fatal(b)
	}
}

func TestIpsetBase_writeBytesNative(t *testing.T) {
	var s ipsetBase
	s.nodes = make([]uint32, 1)
	*(*byte)(unsafe.Pointer(&s.nodes[0])) = 1

	var b bytes.Buffer
	err := s.writeBytesNative(&b)
	if err != nil {
		t.Fatal(err)
	}

	if bb := b.Bytes(); bb[0] != 1 {
		t.Fatal(bb)
	}
}

func TestEndiannessPortability(t *testing.T) {
	prefixes := []netip.Prefix{
		netip.MustParsePrefix("1.2.3.0/24"),
		netip.MustParsePrefix("1.2.4.0/24"),
		netip.MustParsePrefix("1.2.6.0/24"),
	}

	/*
		var s IPSet
		for _, p := range prefixes {
			s.Add(p)
		}
		f, err := os.OpenFile("testdata/ipset.bin", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o666)
		if err != nil {
			t.Fatal(err)
		}
		err = s.Serialize(f)
		if err != nil {
			t.Fatal(err)
		}
		f.Close()
	*/

	d, err := os.ReadFile("testdata/ipset.bin")
	if err != nil {
		t.Fatal(err)
	}

	b := bytes.NewBuffer(d)

	var s1 IPSet
	err = s1.Deserialize(b)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range prefixes {
		if !s1.Contains(p.Addr()) {
			t.Fatal(p)
		}
	}

	var b1 bytes.Buffer
	err = s1.Serialize(&b1)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(d, b1.Bytes()) {
		t.Fatal("Buffers are not equal")
	}
}
