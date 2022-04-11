package netcodec

import (
	"encoding/binary"
	"io"
	"sync"
)

type NetLVCodec struct {
	w            io.Writer
	r            io.Reader
	wLocker      sync.Mutex
	rLocker      sync.Mutex
	nativeEndian binary.ByteOrder
}

func NewNetLVCodec(w io.Writer, r io.Reader, nativeEndian binary.ByteOrder) Codec {
	return &NetLVCodec{
		w:            w,
		r:            r,
		nativeEndian: nativeEndian,
	}
}

func (c *NetLVCodec) WritePayload(payload []byte) error {
	c.wLocker.Lock()
	defer c.wLocker.Unlock()

	length := uint32(len(payload))
	if length == 0 {
		return nil
	}
	if err := binary.Write(c.w, c.nativeEndian, length); err != nil {
		return err
	}
	_, err := c.w.Write(payload)
	return err
}

func (c *NetLVCodec) ReadPayload() (payload []byte, err error) {
	c.rLocker.Lock()
	defer c.rLocker.Unlock()

	var payloadLen uint32
	if err = binary.Read(c.r, c.nativeEndian, &payloadLen); err != nil {
		return
	}
	payload = make([]byte, payloadLen)
	_, err = io.ReadFull(c.r, payload)
	return
}
