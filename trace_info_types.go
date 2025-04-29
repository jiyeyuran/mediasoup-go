package mediasoup

// RtpTraceInfo is "trace" event data for "rtp" type.
type RtpTraceInfo struct {
	RtpPacket *RtpPacketDump `json:"rtpPacket"`
	IsRtx     bool           `json:"isRtx"`
}

// KeyFrameTraceInfo is "trace" event data for "keyframe" type.
type KeyFrameTraceInfo = RtpTraceInfo

type RtpPacketDump struct {
	PayloadType        byte    `json:"payloadType"`
	SequenceNumber     uint16  `json:"sequenceNumber"`
	Timestamp          uint32  `json:"timestamp"`
	Marker             bool    `json:"marker"`
	Ssrc               uint32  `json:"ssrc"`
	IsKeyFrame         bool    `json:"isKeyFrame"`
	Size               uint64  `json:"size"`
	PayloadSize        uint64  `json:"payloadSize"`
	SpatialLayer       byte    `json:"spatialLayer"`
	TemporalLayer      byte    `json:"temporalLayer"`
	Mid                string  `json:"mid"`
	Rid                string  `json:"rid"`
	Rrid               string  `json:"rrid"`
	WideSequenceNumber *uint16 `json:"wideSequenceNumber,omitempty"`
}

// FirTraceInfo is "trace" event data for "fir" type.
type FirTraceInfo struct {
	Ssrc uint32 `json:"ssrc"`
}

// PliTraceInfo is "trace" event data for "pli" type.
type PliTraceInfo = FirTraceInfo

type SrTraceInfo struct {
	Ssrc        uint32 `json:"ssrc"`
	NtpSec      uint32 `json:"ntpSec"`
	NtpFrac     uint32 `json:"ntpFrac"`
	RtpTs       uint32 `json:"rtpTs"`
	PacketCount uint32 `json:"packetCount"`
	OctetCount  uint32 `json:"octetCount"`
}
