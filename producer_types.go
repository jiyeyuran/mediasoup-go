package mediasoup

// ProducerOptions define options to create a producer.
type ProducerOptions struct {
	// Id is the producer id (just for Router.pipeToRouter() method).
	Id string `json:"id,omitempty"`

	// Kind is media kind ("audio" or "video").
	Kind MediaKind `json:"kind,omitempty"`

	// RtpParameters define what the endpoint is sending.
	RtpParameters *RtpParameters `json:"rtpParameters,omitempty"`

	// Paused define whether the producer must start in paused mode. Default false.
	Paused bool `json:"paused,omitempty"`

	// KeyFrameRequestDelay is just used for video. Time (in ms) before asking
	// the sender for a new key frame after having asked a previous one. Default 0.
	KeyFrameRequestDelay uint32 `json:"keyFrameRequestDelay,omitempty"`

	// AppData is custom application data.
	AppData H `json:"appData,omitempty"`
}

// ProducerScore define "score" event data
type ProducerScore struct {
	// Index of the RTP stream in the rtpParameters.encodings array.
	EncodingIdx uint32 `json:"encodingIdx"`

	// Rid of the RTP stream.
	Rid string `json:"rid,omitempty"`

	// Ssrc of the RTP stream.
	Ssrc uint32 `json:"ssrc"`

	// Score of the RTP stream.
	Score uint8 `json:"score"`
}

// ProducerVideoOrientation define "videoorientationchange" event data
type ProducerVideoOrientation struct {
	// Camera define whether the source is a video camera.
	Camera bool `json:"Camera,omitempty"`

	// Flip define whether the video source is flipped.
	Flip bool `json:"flip,omitempty"`

	// Rotation degrees (0, 90, 180 or 270).
	Rotation uint16 `json:"rotation"`
}

// ProducerType define Producer type.
type ProducerType string

const (
	ProducerSimple    ProducerType = "simple"
	ProducerSimulcast ProducerType = "simulcast"
	ProducerSvc       ProducerType = "svc"
	ProducerPipe      ProducerType = "pipe"
)

type ProducerDump struct {
	Id              string                   `json:"id,omitempty"`
	Kind            MediaKind                `json:"kind,omitempty"`
	Type            ProducerType             `json:"type,omitempty"`
	RtpParameters   *RtpParameters           `json:"rtpParameters,omitempty"`
	RtpMapping      *RtpMapping              `json:"rtpMapping,omitempty"`
	RtpStreams      []*RtpStreamDump         `json:"rtpStreams,omitempty"`
	TraceEventTypes []ProducerTraceEventType `json:"traceEventTypes,omitempty"`
	Paused          bool                     `json:"paused,omitempty"`
}

type RtpMapping struct {
	Codecs    []*RtpMappingCodec    `json:"codecs,omitempty"`
	Encodings []*RtpMappingEncoding `json:"encodings,omitempty"`
}

type RtpMappingCodec struct {
	PayloadType       uint8 `json:"payloadType"`
	MappedPayloadType uint8 `json:"mappedPayloadType"`
}

type RtpMappingEncoding struct {
	Rid             string  `json:"rid,omitempty"`
	Ssrc            *uint32 `json:"ssrc,omitempty"`
	ScalabilityMode string  `json:"scalabilityMode,omitempty"`
	MappedSsrc      uint32  `json:"mappedSsrc"`
}

// ProducerTraceEventType define the type for "trace" event.
type ProducerTraceEventType string

const (
	ProducerTraceEventRtp      ProducerTraceEventType = "rtp"
	ProducerTraceEventKeyframe ProducerTraceEventType = "keyframe"
	ProducerTraceEventPli      ProducerTraceEventType = "pli"
	ProducerTraceEventFir      ProducerTraceEventType = "fir"
	ProducerTraceEventSr       ProducerTraceEventType = "sr"
)

// ProducerTraceEventData is "trace" event data.
type ProducerTraceEventData struct {
	// Type specifies the type of trace event.
	Type ProducerTraceEventType `json:"type,omitempty"`

	// Timestamp indicates when the event occurred.
	Timestamp uint64 `json:"timestamp,omitempty"`

	// Direction is event direction, "in" | "out".
	Direction string `json:"direction,omitempty"`

	// Info contains additional data specific to the trace event type, which can be one of:
	// - *RtpTraceInfo
	// - *KeyFrameTraceInfo
	// - *FirTraceInfo
	// - *PliTraceInfo
	// - *SrTraceInfo
	Info any `json:"info,omitempty"`
}

// ProducerStat define the statistic info of a producer
type ProducerStat = RtpStreamRecvStats
