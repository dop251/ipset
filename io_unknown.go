//go:build !(386 || amd64 || arm || arm64 || loong64 || mipsle || mips64le || ppc64le || riscv || riscv64 || wasm || arm64be || armbe || mips || mips64 || ppc || ppc64 || s390 || s390x || sparc || sparc64)

package ipset

import "encoding/binary"

var nativeByteOrder binary.ByteOrder

func (s *ipsetBase) writeBytes(w io.Writer) error {
	if nativeByteOrder == binary.LittleEndian {
		return s.writeBytesNative(w)
	}
	return s.writeBytesGeneric(w)
}

func (s *ipsetBase) loadBytes(b []byte) {
	if nativeByteOrder == binary.LittleEndian {
		s.loadBytesNative(b)
	} else {
		s.loadBytesForeign(b)
	}
}

func init() {
	buf := [2]byte{}
	*(*uint16)(unsafe.Pointer(&buf[0])) = uint16(0xCAFE)

	switch buf {
	case [2]byte{0xFE, 0xCA}:
		nativeByteOrder = binary.LittleEndian
	case [2]byte{0xCA, 0xFE}:
		nativeByteOrder = binary.BigEndian
	default:
		panic("Could not determine native endianness.")
	}
}
