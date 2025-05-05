package mediasoup

type RtpStreamStats struct {
	Type                 string    `json:"type"`
	Timestamp            uint64    `json:"timestamp"`
	Ssrc                 uint32    `json:"ssrc"`
	Kind                 MediaKind `json:"kind"`
	MimeType             string    `json:"mimeType"`
	PacketsLost          uint64    `json:"packetsLost"`
	FractionLost         uint8     `json:"fractionLost"`
	PacketsDiscarded     uint64    `json:"packetsDiscarded"`
	PacketsRetransmitted uint64    `json:"packetsRetransmitted"`
	PacketsRepaired      uint64    `json:"packetsRepaired"`
	NackCount            uint64    `json:"nackCount"`
	NackPacketCount      uint64    `json:"nackPacketCount"`
	PliCount             uint64    `json:"pliCount"`
	FirCount             uint64    `json:"firCount"`
	Score                uint8     `json:"score"`
	Rid                  string    `json:"rid,omitempty"`
	RtxSsrc              *uint32   `json:"rtxSsrc,omitempty"`
	RoundTripTime        float32   `json:"roundTripTime,omitempty"`
	RtxPacketsDiscarded  uint64    `json:"rtxPacketsDiscarded,omitempty"`
	PacketCount          uint64    `json:"packetCount"`
	ByteCount            uint64    `json:"byteCount"`
	Bitrate              uint32    `json:"bitrate"`

	// specific stats of our recv stream.
	Jitter         uint32            `json:"jitter,omitempty"`
	BitrateByLayer map[string]uint32 `json:"bitrateByLayer,omitempty"`
}
