package mediasoup

type RtpStreamStats struct {
	Type                 string    `json:"type"`
	Timestamp            uint64    `json:"timestamp"`
	Ssrc                 uint32    `json:"ssrc"`
	Kind                 MediaKind `json:"kind"`
	MimeType             string    `json:"mimeType"`
	PacketsLost          int32     `json:"packetsLost"`
	FractionLost         uint8     `json:"fractionLost"`
	Jitter               uint32    `json:"jitter,omitempty"`
	PacketsDiscarded     uint64    `json:"packetsDiscarded"`
	PacketsRetransmitted uint64    `json:"packetsRetransmitted"`
	PacketsRepaired      uint64    `json:"packetsRepaired"`
	NackCount            uint64    `json:"nackCount"`
	NackPacketCount      uint64    `json:"nackPacketCount"`
	PliCount             uint64    `json:"pliCount"`
	FirCount             uint64    `json:"firCount"`
	Rid                  string    `json:"rid,omitempty"`
	RtxSsrc              *uint32   `json:"rtxSsrc,omitempty"`
	RoundTripTime        float32   `json:"roundTripTime,omitempty"`
	RtxPacketsDiscarded  uint64    `json:"rtxPacketsDiscarded,omitempty"`
	PacketCount          uint64    `json:"packetCount"`
	ByteCount            uint64    `json:"byteCount"`
	Bitrate              uint32    `json:"bitrate"`
	Score                uint8     `json:"score"`

	// specific stats of our recv stream.
	BitrateByLayer map[string]uint32 `json:"bitrateByLayer,omitempty"`
}
