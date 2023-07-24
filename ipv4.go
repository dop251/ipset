package ipset

import (
	"encoding/binary"
	"io"
	"net/netip"
)

type IPSet4 struct {
	ipsetBase
}

func (s *IPSet4) Contains(ip uint32) bool {
	if len(s.nodes) < 2 {
		return false
	}

	ptr := s.nodes[1]
	for {
		if ptr == ptrPresent {
			return true
		}
		if ptr == ptrAbsent {
			return false
		}
		if !isSkipNode(ptr) { // regular node
			//ptr = *(*uint32)(unsafe.Pointer(uintptr(unsafe.Pointer(&s.nodes[0])) + uintptr(ptr+(ip>>31))*4))
			ptr = s.nodes[ptr+(ip>>31)]
			ip <<= 1
		} else { // skip node
			idx := ptrToIdx(ptr)
			prefix, prefixLen := unpackPrefixLen(s.nodes[idx])
			mask := ^uint32(0) << (32 - prefixLen)
			if prefix == ip&mask {
				ip <<= prefixLen
				ptr = s.nodes[idx+1]
			} else {
				return false
			}
		}
	}
}

func ipPrefixFromIP4Addr(addr uint32) ipPrefix {
	return ipPrefix{
		hi: uint64(addr) << 32,
	}
}

func (s *IPSet4) Add(prefix, length uint32) {
	s.add(ipPrefixFromIP4Addr(prefix), length)
}

func (s *IPSet4) iterateNode(step IterStepFunc, prefix, prefixLen, ptr uint32) bool {
	if ptr == ptrAbsent {
		return true
	}
	if ptr == ptrPresent {
		var a [4]byte
		binary.BigEndian.PutUint32(a[:], prefix)
		return step(netip.PrefixFrom(netip.AddrFrom4(a), int(prefixLen)))
	}
	idx := ptrToIdx(ptr)
	if isSkipNode(ptr) {
		p, l := unpackPrefixLen(s.nodes[idx])
		return s.iterateNode(step, prefix|(p>>prefixLen), prefixLen+l, s.nodes[idx+1])
	} else {
		prefixLen++
		if !s.iterateNode(step, prefix, prefixLen, s.nodes[idx]) {
			return false
		}
		return s.iterateNode(step, prefix|(1<<(32-prefixLen)), prefixLen, s.nodes[idx+1])
	}
}

// WriteTextTo writes a textual representation of the IP set to the provided Writer.
// See IPSet.WriteTextTo for more details.
func (s *IPSet4) WriteTextTo(w io.Writer) (n int64, err error) {
	var buf [19]byte

	s.Iterate(func(prefix netip.Prefix) bool {
		buf1 := prefix.AppendTo(buf[:0])
		buf1 = append(buf1, '\n')
		n1, err1 := w.Write(buf1)
		n += int64(n1)
		if err1 != nil {
			err = err1
			return false
		}
		return true
	})

	return
}

// Iterate traverses the tree and calls the step function for each prefix within the set.
// See IPSet.Iterate for more details.
func (s *IPSet4) Iterate(step IterStepFunc) bool {
	if len(s.nodes) < 2 {
		return true
	}
	return s.iterateNode(step, 0, 0, s.nodes[1])
}
