package netcodec

import (
	"bufio"
	"errors"
	"io"
	"strconv"
)

type State int

const (
	SEPARATOR_SYMBOL byte = ':'
	END_SYMBOL       byte = ','

	PARSE_LENGTH State = iota
	PARSE_SEPARATOR
	PARSE_DATA
	PARSE_END
)

type BufferReader struct {
	*bufio.Reader
	io.Closer
}

type NetStringCodec struct {
	w io.WriteCloser
	r *BufferReader
}

func NewNetStringCodec(w io.WriteCloser, r io.ReadCloser) Codec {
	return &NetStringCodec{
		w: w,
		r: &BufferReader{Reader: bufio.NewReader(r), Closer: r},
	}
}

func (c NetStringCodec) WritePayload(payload []byte) error {
	length := strconv.Itoa(len(payload))

	buffer := make([]byte, 0, len(length)+len(payload)+2)
	buffer = append(buffer, []byte(length)...)
	buffer = append(buffer, SEPARATOR_SYMBOL)
	buffer = append(buffer, payload...)
	buffer = append(buffer, END_SYMBOL)

	_, err := c.w.Write(buffer)
	return err
}

func (c NetStringCodec) ReadPayload() (payload []byte, err error) {
	begin, err := c.r.ReadString(SEPARATOR_SYMBOL)
	if err != nil {
		return
	}
	if len(begin) < 1 {
		err = errors.New("invalid payload start")
		return
	}
	if separator := begin[len(begin)-1]; separator != SEPARATOR_SYMBOL {
		err = errors.New("invalid payload separator")
		return
	}
	length, err := strconv.Atoi(begin[:len(begin)-1])
	if err != nil {
		return
	}
	payload = make([]byte, length)
	if _, err = io.ReadFull(c.r, payload); err != nil {
		return
	}
	end, err := c.r.ReadByte()
	if err != nil {
		return
	}
	if end != END_SYMBOL {
		err = errors.New("invalid payload end")
		return
	}
	return
}

func (c *NetStringCodec) Close() (err error) {
	err1 := c.w.Close()
	err2 := c.r.Close()

	if err1 != nil {
		return err1
	}
	return err2
}
