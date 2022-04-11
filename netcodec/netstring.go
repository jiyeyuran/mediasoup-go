package netcodec

import (
	"bufio"
	"errors"
	"io"
	"strconv"
	"sync"
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

type NetstringCodec struct {
	w       io.Writer
	r       *bufio.Reader
	wLocker sync.Mutex
	rLocker sync.Mutex
}

func NewNetstringCodec(w io.Writer, r io.Reader) Codec {
	return &NetstringCodec{
		w: w,
		r: bufio.NewReader(r),
	}
}

func (c *NetstringCodec) WritePayload(payload []byte) error {
	c.wLocker.Lock()
	defer c.wLocker.Unlock()

	length := strconv.Itoa(len(payload))

	buffer := make([]byte, 0, len(length)+len(payload)+2)
	buffer = append(buffer, []byte(length)...)
	buffer = append(buffer, SEPARATOR_SYMBOL)
	buffer = append(buffer, payload...)
	buffer = append(buffer, END_SYMBOL)

	_, err := c.w.Write(buffer)
	return err
}

func (c *NetstringCodec) ReadPayload() (payload []byte, err error) {
	c.rLocker.Lock()
	defer c.rLocker.Unlock()

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
