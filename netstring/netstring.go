package netstring

import (
	"bytes"
	"strconv"
)

type State int

const (
	SEPARATOR_SYMBOL byte = ':'
	END_SYMBOL       byte = ','

	BUFFER_SIZE int = 10

	PARSE_LENGTH State = iota
	PARSE_SEPARATOR
	PARSE_DATA
	PARSE_END
)

func Encode(payload []byte) (raw []byte) {
	length := strconv.Itoa(len(payload))

	var buffer bytes.Buffer

	buffer.WriteString(length)
	buffer.WriteByte(':')
	buffer.Write(payload)
	buffer.WriteByte(',')

	return buffer.Bytes()
}

type Decoder struct {
	parsedData []byte
	length     int
	state      State
	outputCh   chan []byte
}

func NewDecoder() *Decoder {
	return &Decoder{
		state:    PARSE_LENGTH,
		outputCh: make(chan []byte, BUFFER_SIZE),
	}
}

func (decoder *Decoder) Reset() {
	decoder.length = 0
	decoder.parsedData = []byte{}
	decoder.state = PARSE_LENGTH
}

func (decoder *Decoder) Length() int {
	return decoder.length
}

func (decoder *Decoder) Result() chan []byte {
	return decoder.outputCh
}

// Feed New incoming parsedData packets are feeded into the decoder using this method.
// Call this method every time we have a new set of parsedData.
func (decoder *Decoder) Feed(data []byte) {
	for i := 0; i < len(data); {
		i = decoder.parse(i, data)
	}
}

func (decoder *Decoder) parse(i int, data []byte) int {
	switch decoder.state {
	case PARSE_LENGTH:
		return decoder.parseLength(i, data)
	case PARSE_SEPARATOR:
		return decoder.parseSeparator(i, data)
	case PARSE_DATA:
		return decoder.parseData(i, data)
	case PARSE_END:
		return decoder.parseEnd(i, data)
	}
	return i
}

func (decoder *Decoder) parseLength(i int, data []byte) int {
	symbol := data[i]
	if symbol >= '0' && symbol <= '9' {
		decoder.length = (decoder.length * 10) + (int(symbol) - 48)
		i++
	} else {
		decoder.state = PARSE_SEPARATOR
	}

	return i
}

func (decoder *Decoder) parseSeparator(i int, data []byte) int {
	if data[i] != SEPARATOR_SYMBOL {
		// Something is wrong with the parsedData.
		// let's reset everything to start looking for next valid parsedData
		decoder.Reset()
	} else {
		decoder.state = PARSE_DATA
	}
	i++
	return i
}

func (decoder *Decoder) parseData(i int, data []byte) int {
	dataSize := len(data) - i
	dataLength := min(decoder.length, dataSize)
	decoder.parsedData = append(decoder.parsedData, data[i:i+dataLength]...)
	decoder.length = decoder.length - dataLength
	if decoder.length == 0 {
		decoder.state = PARSE_END
	}
	// We already parsed till i + dataLength
	return i + dataLength
}

func (decoder *Decoder) parseEnd(i int, data []byte) int {
	symbol := data[i]
	if symbol == END_SYMBOL {
		// Symbol matches, that means this is valid data
		decoder.outputCh <- decoder.parsedData
	}
	// Irrespective of what symbol we got we have to reset.
	// Since we are looking for new data from now onwards.
	decoder.Reset()
	return i
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
