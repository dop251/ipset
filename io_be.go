//go:build arm64be || armbe || mips || mips64 || ppc || ppc64 || s390 || s390x || sparc || sparc64

package ipset

import "io"

func (s *ipsetBase) writeBytes(w io.Writer) error {
	return s.writeBytesGeneric(w)
}

func (s *ipsetBase) loadBytes(b []byte) {
	s.loadBytesForeign(b)
}
