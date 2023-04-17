//go:build 386 || amd64 || arm || arm64 || loong64 || mipsle || mips64le || ppc64le || riscv || riscv64 || wasm

package ipset

import (
	"io"
)

func (s *ipsetBase) writeBytes(w io.Writer) error {
	return s.writeBytesNative(w)
}

func (s *ipsetBase) loadBytes(b []byte) {
	s.loadBytesNative(b)
}
