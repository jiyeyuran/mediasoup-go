package netcodec

type Codec interface {
	WritePayload(payload []byte) error
	ReadPayload() ([]byte, error)
}
