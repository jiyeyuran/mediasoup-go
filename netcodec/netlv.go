package netcodec

import (
	"encoding/binary"
	"io"
	"sync"
	"unsafe"
)

func hostByteOrder() binary.ByteOrder {
	buf := [2]byte{}
	*(*uint16)(unsafe.Pointer(&buf[0])) = uint16(0xABCD)

	switch buf {
	case [2]byte{0xCD, 0xAB}:
		return binary.LittleEndian
	case [2]byte{0xAB, 0xCD}:
		return binary.BigEndian
	default:
		panic("Could not determine native endianness.")
	}
}

type NetLVCodec struct {
	w            io.WriteCloser
	r            io.ReadCloser
	nativeEndian binary.ByteOrder
	mu           sync.Mutex
}

func NewNetLVCodec(w io.WriteCloser, r io.ReadCloser) Codec {
	return &NetLVCodec{
		w:            w,
		r:            r,
		nativeEndian: hostByteOrder(),
	}
}

func (c *NetLVCodec) WritePayload(payload []byte) error {
	length := uint32(len(payload))
	if length == 0 {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := binary.Write(c.w, c.nativeEndian, length); err != nil {
		return err
	}
	_, err := c.w.Write(payload)
	return err
}

func (c *NetLVCodec) ReadPayload() (payload []byte, err error) {
	var payloadLen uint32
	if err = binary.Read(c.r, c.nativeEndian, &payloadLen); err != nil {
		return
	}
	payload = make([]byte, payloadLen)
	_, err = io.ReadFull(c.r, payload)
	return
}

func (c *NetLVCodec) Close() (err error) {
	err1 := c.w.Close()
	err2 := c.r.Close()

	if err1 != nil {
		return err1
	}
	return err2
}
