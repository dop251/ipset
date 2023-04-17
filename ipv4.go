package ipset

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
