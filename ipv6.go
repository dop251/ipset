package ipset

import "encoding/binary"

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
