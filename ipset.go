//go:generate sh -c "cd generate-prefixlist && go run main.go US"

package ipset

import (
	"encoding/binary"
	"io"
	"net/netip"
)

type IterStepFunc func(prefix netip.Prefix) (continueIteration bool)

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

// Iterate calls the step function for each prefix within the set.
// If the step function returns false, the iteration stops and the function returns false, otherwise
// it returns true after all nodes are traversed.
// The step function must not modify the set.
func (s *IPSet) Iterate(step IterStepFunc) bool {
	return s.s4.Iterate(step) && s.s6.Iterate(step)
}

// WriteTextTo writes a textual representation of the IP set to the provided Writer.
// The text will contain one prefix per line, separated by '\n'. The order is not guaranteed to match the order
// in which the prefixes were added. Some contiguous prefixes may be merged.
// There will be one w.Write() call per prefix, so it is advisable to provide a buffered Writer.
func (s *IPSet) WriteTextTo(w io.Writer) (n int64, err error) {
	n, err = s.s4.WriteTextTo(w)
	if err != nil {
		return
	}
	n1, err := s.s6.WriteTextTo(w)
	n += n1
	return
}
