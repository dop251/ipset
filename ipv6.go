package ipset

import (
	"encoding/binary"
	"io"
	"net/netip"
)

func ipPrefixFromIP6Addr(addr [16]byte) ipPrefix {
	return ipPrefix{
		hi: binary.BigEndian.Uint64(addr[:8]),
		lo: binary.BigEndian.Uint64(addr[8:]),
	}
}

type IPSet6 struct {
	ipsetBase
}

func (s *IPSet6) matchNode(addr ipPrefix, ptr uint32) bool {
	ip := addr.hi32()
	for {
		if ptr == ptrPresent {
			return true
		}
		if ptr == ptrAbsent {
			return false
		}
		if !isSkipNode(ptr) { // regular node
			ptr = s.nodes[ptr+(ip>>31)]
			ip = addr.shl(1)
		} else { // skip node
			idx := ptrToIdx(ptr)
			prefix, prefixLen := unpackPrefixLen(s.nodes[idx])
			mask := ^uint32(0) << (32 - prefixLen)
			if prefix == ip&mask {
				ip = addr.shl(prefixLen)
				ptr = s.nodes[idx+1]
			} else {
				return false
			}
		}
	}
}

func (s *IPSet6) Add(prefix [16]byte, length uint32) {
	s.add(ipPrefixFromIP6Addr(prefix), length)
}

func (s *IPSet6) Contains(addr [16]byte) bool {
	if len(s.nodes) < 2 {
		return false
	}

	return s.matchNode(ipPrefixFromIP6Addr(addr), s.nodes[1])
}

// WriteTextTo writes a textual representation of the IP set to the provided Writer.
// See IPSet.WriteTextTo for more details.
func (s *IPSet6) WriteTextTo(w io.Writer) (n int64, err error) {
	var buf [43]byte

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

func (s *IPSet6) iterateNode(step IterStepFunc, prefix ipPrefix, prefixLen, ptr uint32) bool {
	if ptr == ptrAbsent {
		return true
	}
	if ptr == ptrPresent {
		var a [16]byte
		binary.BigEndian.PutUint64(a[:8], prefix.hi)
		binary.BigEndian.PutUint64(a[8:], prefix.lo)
		return step(netip.PrefixFrom(netip.AddrFrom16(a), int(prefixLen)))
	}
	idx := ptrToIdx(ptr)
	if isSkipNode(ptr) {
		p, l := unpackPrefixLen(s.nodes[idx])
		p64 := uint64(p) << 32
		if prefixLen >= 64 {
			prefix.lo |= p64 >> (prefixLen - 64)
		} else {
			prefix.hi |= p64 >> (prefixLen)
			prefix.lo |= p64 << (64 - prefixLen)
		}
		return s.iterateNode(step, prefix, prefixLen+l, s.nodes[idx+1])
	} else {
		prefixLen++
		if !s.iterateNode(step, prefix, prefixLen, s.nodes[idx]) {
			return false
		}
		if prefixLen > 64 {
			prefix.lo |= 1 << (128 - prefixLen)
		} else {
			prefix.hi |= 1 << (64 - prefixLen)
		}
		return s.iterateNode(step, prefix, prefixLen, s.nodes[idx+1])
	}
}

// Iterate traverses the tree and calls the step function for each prefix within the set.
// See IPSet.Iterate for more details.
func (s *IPSet6) Iterate(step IterStepFunc) bool {
	if len(s.nodes) < 2 {
		return true
	}
	return s.iterateNode(step, ipPrefix{}, 0, s.nodes[1])
}
