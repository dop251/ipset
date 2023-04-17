//go:generate sh -c "cd generate-prefixlist && go run main.go US"

package ipset

import (
	"encoding/binary"
	"io"
	"net/netip"
)

type IPSet struct {
	s4 IPSet4
	s6 IPSet6
}

func (s *IPSet) Add(prefix netip.Prefix) {
	addr, bits := prefix.Addr(), uint32(prefix.Bits())
	if addr.Is4() || addr.Is4In6() {
		if bits > 32 {
			bits = 32
		}
		a := addr.As4()
		s.s4.Add(binary.BigEndian.Uint32(a[:]), bits)
	} else if addr.Is6() {
		if bits > 128 {
			bits = 128
		}
		s.s6.Add(addr.As16(), bits)
	}
}

func (s *IPSet) Contains(addr netip.Addr) bool {
	if addr.Is4() || addr.Is4In6() {
		a := addr.As4()
		return s.s4.Contains(binary.BigEndian.Uint32(a[:]))
	} else if addr.Is6() {
		return s.s6.Contains(addr.As16())
	}
	return false
}

func (s *IPSet) Deserialize(r io.Reader) error {
	if err := s.s4.Deserialize(r); err != nil {
		return err
	}
	if err := s.s6.Deserialize(r); err != nil {
		return err
	}
	return nil
}

func (s *IPSet) Serialize(w io.Writer) error {
	if err := s.s4.Serialize(w); err != nil {
		return err
	}
	if err := s.s6.Serialize(w); err != nil {
		return err
	}
	return nil
}

func (s *IPSet) Compact() {
	s.s4.Compact()
	s.s6.Compact()
}
