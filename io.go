package ipset

import (
	"encoding/binary"
	"errors"
	"io"
	"unsafe"
)

var ErrInvalidFormat = errors.New("invalid format")

func (s *ipsetBase) Serialize(w io.Writer) error {
	if len(s.nodes) == 0 {
		_, err := w.Write([]byte{0, 0, 0, 0})
		return err
	}
	s.Compact()
	s.nodes[0] = uint32(len(s.nodes)) * 4
	return s.writeBytes(w)
}

func (s *ipsetBase) Deserialize(r io.Reader) error {
	var buf [4]byte

	_, err := io.ReadFull(r, buf[:])
	if err != nil {
		return err
	}

	size := binary.LittleEndian.Uint32(buf[:])
	if size&3 != 0 {
		return ErrInvalidFormat
	}

	var b []byte
	if size != 0 {
		b = make([]byte, size)
		_, err = io.ReadFull(r, b[4:])
		if err != nil {
			return err
		}
	}

	s.loadBytes(b)
	return nil
}

func (s *ipsetBase) bytesNative() []byte {
	return unsafe.Slice((*byte)(unsafe.Pointer(&s.nodes[0])), len(s.nodes)*4)
}

func (s *ipsetBase) bytesGeneric() []byte {
	b := make([]byte, len(s.nodes)*4)
	for i, v := range s.nodes {
		binary.LittleEndian.PutUint32(b[i*4:], v)
	}
	return b
}

func (s *ipsetBase) writeBytesGeneric(w io.Writer) error {
	var buf []byte
	if l := len(s.nodes) * 4; l < 4096 {
		buf = make([]byte, l)
	} else {
		buf = make([]byte, 4096)
	}

	p := uint(0)
	for p < uint(len(s.nodes)) {
		for i := 0; i < len(buf)-3; i += 4 {
			if p >= uint(len(s.nodes)) {
				buf = buf[:i]
				break
			}
			binary.LittleEndian.PutUint32(buf[i:], s.nodes[p])
			p++
		}
		_, err := w.Write(buf)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *ipsetBase) loadBytesForeign(b []byte) {
	for i := 0; i < len(b); i += 4 {
		bb := b[i : i+4]
		binary.BigEndian.PutUint32(bb, binary.LittleEndian.Uint32(bb))
	}
	s.loadBytesNative(b)
}

func (s *ipsetBase) writeBytesNative(w io.Writer) error {
	_, err := w.Write(s.bytesNative())
	return err
}

func (s *ipsetBase) loadBytesNative(b []byte) {
	var nodes []uint32
	if len(b) != 0 {
		nodes = (*(*[]uint32)(unsafe.Pointer(&b)))[: len(b)/4 : len(b)/4]
	}
	s.nodes = nodes
	s.freeList = nil
}
