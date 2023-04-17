package ipset

import "math/bits"

const (
	ptrAbsent  = 0
	ptrPresent = 1

	skipNodeMask = 1
)

type ipPrefix struct {
	hi, lo uint64
}

func (p *ipPrefix) hi32() uint32 {
	return uint32(p.hi >> 32)
}

func (p *ipPrefix) shl(bits uint32) uint32 {
	p.hi <<= bits
	p.hi |= p.lo >> (64 - bits)
	p.lo <<= bits

	return p.hi32()
}

type ipsetBase struct {
	nodes    []uint32
	freeList []uint32
}

func ptrToIdx(ptr uint32) uint32 {
	return ptr &^ skipNodeMask
}

func idxToPtr(idx uint32) uint32 {
	return idx
}

func isSkipNode(ptr uint32) bool {
	return ptr&skipNodeMask != 0
}

const maxPackablePrefixLen = 31

// 0...1<prefix>
func packPrefixLen(prefix, length uint32) uint32 {
	return prefix>>(32-length) | (1 << length)
}

func unpackPrefixLen(prefixLen uint32) (prefix, length uint32) {
	length = uint32(31 - bits.LeadingZeros32(prefixLen))
	prefix = prefixLen << (32 - length)
	return
}

func (s *ipsetBase) allocateNode(v1, v2 uint32) (idx uint32) {
	if len(s.freeList) > 0 {
		idx = s.freeList[len(s.freeList)-1]
		s.freeList = s.freeList[:len(s.freeList)-1]
		s.nodes[idx+1], s.nodes[idx] = v2, v1
	} else {
		idx = uint32(len(s.nodes))
		s.nodes = append(s.nodes, v1, v2)
	}
	return
}

// getPtr gets a pointer to an element with the specified index and ensures that the next allocateNode does not
// copy the slice (thus invalidating pointers) so that things like *ptr = s.allocateNode(...) could work.
func (s *ipsetBase) getPtr(idx uint32) *uint32 {
	if cap(s.nodes)-len(s.nodes) < 2 && len(s.freeList) == 0 {
		s.nodes = append(s.nodes, 0, 0)[:len(s.nodes)]
	}
	return &s.nodes[idx]
}

func (s *ipsetBase) freeNode(ptr uint32) {
	if ptr <= ptrPresent {
		return
	}
	idx := ptrToIdx(ptr)
	s.freeList = append(s.freeList, idx)
	s.freeNode(s.nodes[idx+1])
	if !isSkipNode(ptr) {
		s.freeNode(s.nodes[idx])
	}
}

func (s *ipsetBase) createSplitNode(prefix uint32, prefixLen uint32, childPtr, idx uint32) (newIdx uint32) {
	var newNodePtr uint32
	switch prefixLen {
	case 1:
		if prefix&0x8000_0000 == 0 {
			s.nodes[idx] = childPtr
			newIdx = idx + 1
		} else {
			s.nodes[idx+1] = childPtr
			newIdx = idx
		}
		return
	case 2:
		if prefix&0x4000_0000 == 0 {
			newNodePtr = idxToPtr(s.allocateNode(childPtr, ptrAbsent))
		} else {
			newNodePtr = idxToPtr(s.allocateNode(ptrAbsent, childPtr))
		}
	default:
		// See if we can merge two nodes
		if childPtr > ptrPresent {
			if isSkipNode(childPtr) {
				childIdx := ptrToIdx(childPtr)
				pl := &s.nodes[childIdx]
				cp, cpl := unpackPrefixLen(*pl)
				if newPrefixLen := prefixLen + cpl - 1; newPrefixLen <= maxPackablePrefixLen {
					// Merging two split nodes
					*pl = packPrefixLen((prefix<<1)|(cp>>(prefixLen-1)), newPrefixLen)
					newNodePtr = childPtr
					break
				}
			} else {
				// Check if we can merge a regular node with the split node
				a := &s.nodes[childPtr]
				if *a == ptrAbsent {
					*a = packPrefixLen((prefix<<1)|(1<<(32-prefixLen)), prefixLen)
					newNodePtr = childPtr | skipNodeMask
					break
				}
				b := &s.nodes[childPtr+1]
				if *b == ptrAbsent {
					*b = *a
					*a = packPrefixLen(prefix<<1, prefixLen)
					newNodePtr = childPtr | skipNodeMask
					break
				}
			}
		}
		// Create a copy of the split node with prefix reduced by 1 bit
		newNodePtr = idxToPtr(s.allocateNode(packPrefixLen(prefix<<1, prefixLen-1), childPtr)) | skipNodeMask
	}

	if prefix&0x8000_0000 == 0 {
		s.nodes[idx] = newNodePtr
		newIdx = idx + 1
	} else {
		s.nodes[idx+1] = newNodePtr
		newIdx = idx
	}
	return
}

func (s *ipsetBase) add(p ipPrefix, prefixLen uint32) {
	if len(s.nodes) == 0 {
		s.nodes = make([]uint32, 2, 8)
	}
	prefix := p.hi32()
	origPrefix := p
	var lastRegIdx uint32 = 0
	var nextIdx uint32 = 1
	for prefixLen > 0 {
		ptr := s.getPtr(nextIdx)
		if *ptr == ptrAbsent {
			if prefixLen > 1 {
				// Adding a skip node
				var l uint32
				if prefixLen < maxPackablePrefixLen {
					l = prefixLen
				} else {
					l = maxPackablePrefixLen
				}
				newIdx := s.allocateNode(packPrefixLen(prefix, l), 0)
				*ptr = idxToPtr(newIdx) | skipNodeMask
				if l < prefixLen {
					nextIdx = newIdx + 1
					prefix = p.shl(l)
					prefixLen -= l
				} else {
					nextIdx = newIdx + 1
					break
				}
			} else {
				var v1, v2 uint32
				if prefix&0x8000_0000 == 0 {
					v1, v2 = ptrPresent, ptrAbsent
				} else {
					v1, v2 = ptrAbsent, ptrPresent
				}
				*ptr = idxToPtr(s.allocateNode(v1, v2))
				return
			}
		} else if *ptr == ptrPresent {
			// Already contains a larger or matching prefix
			return
		} else {
			if *ptr&skipNodeMask == 0 {
				// Traverse a regular node
				lastRegIdx = *ptr
				nextIdx = *ptr + (prefix >> 31)
				prefix = p.shl(1)
				prefixLen--
			} else {
				// Need to split a skip node
				idx := ptrToIdx(*ptr)
				lastRegIdx = 0
				curPrefix, curPrefixLen := unpackPrefixLen(s.nodes[idx])
				// Determining the split point
				commonLen := uint32(bits.LeadingZeros32(prefix ^ curPrefix))
				if commonLen > curPrefixLen {
					commonLen = curPrefixLen
				}
				if commonLen > prefixLen {
					commonLen = prefixLen
				}
				if commonLen == curPrefixLen {
					prefix = p.shl(curPrefixLen)
					prefixLen -= curPrefixLen
					nextIdx = idx + 1
					continue
				}
				if commonLen == prefixLen {
					// Already contains a more specific prefix
					s.freeNode(s.nodes[idx+1])
					// Shorten the prefix to commonLen. If commonLen is 1, convert the skip node into a regular one
					if commonLen > 1 {
						s.nodes[idx] = packPrefixLen(curPrefix, commonLen)
						s.nodes[idx+1] = ptrPresent
					} else {
						*ptr &^= skipNodeMask
						if curPrefix&0x8000_0000 == 0 {
							s.nodes[idx], s.nodes[idx+1] = ptrPresent, ptrAbsent
						} else {
							s.nodes[idx], s.nodes[idx+1] = ptrAbsent, ptrPresent
						}
					}
					return
				}
				switch commonLen {
				case 0:
					// Convert the skip node into a regular node
					*ptr &^= skipNodeMask
					lastRegIdx = idx
					nextIdx = s.createSplitNode(curPrefix, curPrefixLen, s.nodes[idx+1], idx)
					s.nodes[nextIdx] = 0
				case 1:
					// Convert the skip node into a regular node
					*ptr &^= skipNodeMask
					newNodeIdx := s.allocateNode(0, 0)
					oldPtr := s.nodes[idx+1]
					if oldPtr == ptrPresent {
						lastRegIdx = newNodeIdx
					}
					if prefix&0x8000_0000 == 0 {
						s.nodes[idx], s.nodes[idx+1] = idxToPtr(newNodeIdx), ptrAbsent
					} else {
						s.nodes[idx], s.nodes[idx+1] = ptrAbsent, idxToPtr(newNodeIdx)
					}
					nextIdx = s.createSplitNode(curPrefix<<1, curPrefixLen-1, oldPtr, newNodeIdx)
				default:
					newNodeIdx := s.allocateNode(0, 0)
					s.nodes[idx] = packPrefixLen(curPrefix, commonLen)
					oldPtr := s.nodes[idx+1]
					if oldPtr == ptrPresent {
						lastRegIdx = newNodeIdx
					}
					s.nodes[idx+1] = idxToPtr(newNodeIdx)
					nextIdx = s.createSplitNode(curPrefix<<commonLen, curPrefixLen-commonLen, oldPtr, newNodeIdx)
				}
				commonLen++
				prefix = p.shl(commonLen)
				prefixLen -= commonLen
			}
		}
	}
	ptr := &s.nodes[nextIdx]
	if *ptr != ptrAbsent {
		s.freeNode(*ptr)
	}
	*ptr = ptrPresent
	if lastRegIdx > 0 && s.nodes[lastRegIdx] == ptrPresent && s.nodes[lastRegIdx+1] == ptrPresent {
		s.mergeNodes(origPrefix)
	}
}

func (s *ipsetBase) mergeNodes(p ipPrefix) {
	var trace [128]uint32
	traceLen := 0

	ip := p.hi32()
	idx := uint32(1)
	for {
		ptr := s.nodes[idx]
		if ptr == ptrPresent {
			break
		}
		if ptr == ptrAbsent {
			return
		}
		if !isSkipNode(ptr) { // regular node
			trace[traceLen] = idx
			traceLen++
			idx = ptr + (ip >> 31)
			ip = p.shl(1)
		} else { // skip node
			traceLen = 0
			idx = ptrToIdx(ptr)
			prefix, prefixLen := unpackPrefixLen(s.nodes[idx])
			mask := ^uint32(0) << (32 - prefixLen)
			if prefix == ip&mask {
				ip = p.shl(prefixLen)
				idx++
			} else {
				return
			}
		}
	}

	for i := traceLen - 1; i >= 0; i-- {
		idx := s.nodes[trace[i]]
		if s.nodes[idx] == ptrPresent && s.nodes[idx+1] == ptrPresent {
			s.freeList = append(s.freeList, idx)
			s.nodes[trace[i]] = ptrPresent
		}
	}
}

func (s *ipsetBase) Compact() {
	if len(s.freeList) == 0 {
		if cap(s.nodes) > len(s.nodes) {
			n := make([]uint32, len(s.nodes))
			copy(n, s.nodes)
			s.nodes = n
		}
	} else {
		n := make([]uint32, 2, len(s.nodes)-len(s.freeList)*2)
		n[1] = s.compactNode(&n, s.nodes[1])
		s.nodes = n
	}
	s.freeList = nil
}

func (s *ipsetBase) compactNode(n *[]uint32, ptr uint32) (newPtr uint32) {
	if ptr <= ptrPresent {
		return ptr
	}
	newIdx := uint32(len(*n))
	*n = append(*n, 0, 0)
	if !isSkipNode(ptr) {
		(*n)[newIdx] = s.compactNode(n, s.nodes[ptr])
		(*n)[newIdx+1] = s.compactNode(n, s.nodes[ptr+1])
		newPtr = newIdx
	} else {
		idx := ptrToIdx(ptr)
		(*n)[newIdx] = s.nodes[idx]
		(*n)[newIdx+1] = s.compactNode(n, s.nodes[idx+1])
		newPtr = newIdx | skipNodeMask
	}
	return
}
