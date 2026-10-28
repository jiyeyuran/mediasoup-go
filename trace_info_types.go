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
	Size               uint32  `json:"size"`
	PayloadSize        uint32  `json:"payloadSize"`
	SpatialLayer       byte    `json:"spatialLayer"`
	TemporalLayer      byte    `json:"temporalLayer"`
	Mid                string  `json:"mid"`
	Rid                string  `json:"rid"`
	Rrid               string  `json:"rrid"`
	WideSequenceNumber *uint16 `json:"wideSequenceNumber,omitempty"`
}

// rtpPacketDumpInfo is the "trace" event info of a "probation" type: the
// fields of the probation RTP packet.
func rtpPacketDumpInfo(dump RtpPacketDump) H {
	info := H{
		"payloadType":    dump.PayloadType,
		"sequenceNumber": dump.SequenceNumber,
		"timestamp":      dump.Timestamp,
		"marker":         dump.Marker,
		"ssrc":           dump.Ssrc,
		"isKeyFrame":     dump.IsKeyFrame,
		"size":           dump.Size,
		"payloadSize":    dump.PayloadSize,
		"spatialLayer":   dump.SpatialLayer,
		"temporalLayer":  dump.TemporalLayer,
		"mid":            dump.Mid,
		"rid":            dump.Rid,
		"rrid":           dump.Rrid,
	}
	if dump.WideSequenceNumber != nil {
		info["wideSequenceNumber"] = *dump.WideSequenceNumber
	}

	return info
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
