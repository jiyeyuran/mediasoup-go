package mediasoup

import "encoding/json"

type WorkerDump struct {
	Pid             int      `json:"pid,omitempty"`
	RouterIds       []string `json:"routerIds,omitempty"`
	WebRtcServerIds []string `json:"webRtcServerIds,omitempty"`
}

func (d WorkerDump) String() string {
	data, _ := json.Marshal(d)
	return string(data)
}

type WebRtcServerDump struct {
	Id                        string                     `json:"id,omitempty"`
	UdpSockets                []NetAddr                  `json:"udpSockets,omitempty"`
	TcpServers                []NetAddr                  `json:"tcpServers,omitempty"`
	WebRtcTransportIds        []string                   `json:"webRtcTransportIds,omitempty"`
	LocalIceUsernameFragments []LocalIceUsernameFragment `json:"localIceUsernameFragments,omitempty"`
	TupleHashes               []TupleHash                `json:"tupleHashes,omitempty"`
}

func (d WebRtcServerDump) String() string {
	data, _ := json.Marshal(d)
	return string(data)
}

type NetAddr struct {
	Ip   string `json:"ip,omitempty"`
	Port uint16 `json:"port,omitempty"`
}

type LocalIceUsernameFragment struct {
	LocalIceUsernameFragment string `json:"localIceUsernameFragment,omitempty"`
	WebRtcTransportId        string `json:"webRtcTransportId,omitempty"`
}

type TupleHash struct {
	TupleHash         uint64 `json:"tupleHash,omitempty"`
	WebRtcTransportId string `json:"webRtcTransportId,omitempty"`
}

type RouterDump struct {
	Id                               string              `json:"id,omitempty"`
	TransportIds                     []string            `json:"transportIds,omitempty"`
	RtpObserverIds                   []string            `json:"rtpObserverIds,omitempty"`
	MapProducerIdConsumerIds         map[string][]string `json:"mapProducerIdConsumerIds,omitempty"`
	MapConsumerIdProducerId          map[string]string   `json:"mapConsumerIdProducerId,omitempty"`
	MapProducerIdObserverIds         map[string][]string `json:"mapProducerIdObserverIds,omitempty"`
	MapDataProducerIdDataConsumerIds map[string][]string `json:"mapDataProducerIdDataConsumerIds,omitempty"`
	MapDataConsumerIdDataProducerId  map[string]string   `json:"mapDataConsumerIdDataProducerId,omitempty"`
}

func (d RouterDump) String() string {
	data, _ := json.Marshal(d)
	return string(data)
}

type TransportDump struct {
	Id                      string                   `json:"id,omitempty"`
	Direct                  bool                     `json:"direct,omitempty"`
	ProducerIds             []string                 `json:"producerIds,omitempty"`
	ConsumerIds             []string                 `json:"consumerIds,omitempty"`
	MapSsrcConsumerId       map[string]string        `json:"mapSsrcConsumerId,omitempty"`
	MapRtxSsrcConsumerId    map[string]string        `json:"mapRtxSsrcConsumerId,omitempty"`
	DataProducerIds         []string                 `json:"dataProducerIds,omitempty"`
	DataConsumerIds         []string                 `json:"dataConsumerIds,omitempty"`
	RecvRtpHeaderExtensions *RecvRtpHeaderExtensions `json:"recvRtpHeaderExtensions,omitempty"`
	RtpListener             *RtpListener             `json:"rtpListener,omitempty"`
	SctpParameters          SctpParameters           `json:"SctpParameters,omitempty"`
	SctpState               SctpState                `json:"sctpState,omitempty"`
	SctpListener            *SctpListener            `json:"sctpListener,omitempty"`
	TraceEventTypes         string                   `json:"traceEventTypes,omitempty"`

	// plain transport
	*PlainTransportDump

	// webrtc transport
	*WebRtcTransportDump
}

func (d TransportDump) String() string {
	data, _ := json.Marshal(d)
	return string(data)
}

type PlainTransportDump struct {
	RtcpMux        bool            `json:"rtcpMux,omitempty"`
	Comedia        bool            `json:"comedia,omitempty"`
	Tuple          *TransportTuple `json:"tuple,omitempty"`
	RtcpTuple      *TransportTuple `json:"rtcpTuple,omitempty"`
	SrtpParameters *SrtpParameters `json:"srtpParameters,omitempty"`
}

func (d PlainTransportDump) String() string {
	data, _ := json.Marshal(d)
	return string(data)
}

type WebRtcTransportDump struct {
	IceRole          string          `json:"iceRole,omitempty"`
	IceParameters    IceParameters   `json:"iceParameters,omitempty"`
	IceCandidates    []IceCandidate  `json:"iceCandidates,omitempty"`
	IceState         IceState        `json:"iceState,omitempty"`
	IceSelectedTuple *TransportTuple `json:"iceSelectedTuple,omitempty"`
	DtlsParameters   DtlsParameters  `json:"dtlsParameters,omitempty"`
	DtlsState        DtlsState       `json:"dtlsState,omitempty"`
	DtlsRemoteCert   string          `json:"dtlsRemoteCert,omitempty"`
}

func (d WebRtcTransportDump) String() string {
	data, _ := json.Marshal(d)
	return string(data)
}

type ConsumerDump struct {
	Id                         string               `json:"id,omitempty"`
	ProducerId                 string               `json:"producerId,omitempty"`
	Kind                       string               `json:"kind,omitempty"`
	Type                       string               `json:"type,omitempty"`
	RtpParameters              RtpParameters        `json:"rtpParameters,omitempty"`
	ConsumableRtpEncodings     []RtpMappingEncoding `json:"consumableRtpEncodings,omitempty"`
	SupportedCodecPayloadTypes []uint32             `json:"supportedCodecPayloadTypes,omitempty"`
	Paused                     bool                 `json:"paused,omitempty"`
	ProducerPaused             bool                 `json:"producerPaused,omitempty"`
	Priority                   uint8                `json:"priority,omitempty"`
	TraceEventTypes            string               `json:"traceEventTypes,omitempty"`
	RtpStreams                 []RtpStream          `json:"rtpStreams,omitempty"`
	RtpStream                  *RtpStream           `json:"rtpStream,omitempty"` // dump by SvcConsumer
	*SimulcastConsumerDump
}

func (d ConsumerDump) String() string {
	data, _ := json.Marshal(d)
	return string(data)
}

type RtpStream struct {
	Params    RtpStreamParams `json:"params,omitempty"`
	Score     uint8           `json:"score,omitempty"`
	RtxStream *RtpStream      `json:"rtxStream,omitempty"`
}

type RtpStreamParams struct {
	EncodingIdx    int    `json:"encodingIdx,omitempty"`
	Ssrc           uint32 `json:"ssrc,omitempty"`
	PayloadType    uint8  `json:"payloadType,omitempty"`
	MimeType       string `json:"mimeType,omitempty"`
	ClockRate      uint32 `json:"clockRate,omitempty"`
	Rid            string `json:"rid,omitempty"`
	RRid           string `json:"rrid,omitempty"`
	Cname          string `json:"cname,omitempty"`
	RtxSsrc        uint32 `json:"rtxSsrc,omitempty"`
	RtxPayloadType uint8  `json:"rtxPayloadType,omitempty"`
	UseNack        bool   `json:"useNack,omitempty"`
	UsePli         bool   `json:"usePli,omitempty"`
	UseFir         bool   `json:"useFir,omitempty"`
	UseInBandFec   bool   `json:"useInBandFec,omitempty"`
	UseDtx         bool   `json:"useDtx,omitempty"`
	SpatialLayers  uint8  `json:"spatialLayers,omitempty"`
	TemporalLayers uint8  `json:"temporalLayers,omitempty"`
}

type SimulcastConsumerDump struct {
	PreferredSpatialLayer  int16 `json:"preferredSpatialLayer,omitempty"`
	TargetSpatialLayer     int16 `json:"targetSpatialLayer,omitempty"`
	CurrentSpatialLayer    int16 `json:"currentSpatialLayer,omitempty"`
	PreferredTemporalLayer int16 `json:"preferredTemporalLayer,omitempty"`
	TargetTemporalLayer    int16 `json:"targetTemporalLayer,omitempty"`
	CurrentTemporalLayer   int16 `json:"currentTemporalLayer,omitempty"`
}

func (d SimulcastConsumerDump) String() string {
	data, _ := json.Marshal(d)
	return string(data)
}

type ProducerDump struct {
	Id              string             `json:"id,omitempty"`
	Kind            string             `json:"kind,omitempty"`
	Type            string             `json:"type,omitempty"`
	RtpParameters   RtpParameters      `json:"rtpParameters,omitempty"`
	RtpMapping      RtpMapping         `json:"rtpMapping,omitempty"`
	Encodings       RtpMappingEncoding `json:"encodings,omitempty"`
	RtpStreams      []RtpStream        `json:"rtpStreams,omitempty"`
	Paused          bool               `json:"paused,omitempty"`
	TraceEventTypes string             `json:"traceEventTypes,omitempty"`
}

func (d ProducerDump) String() string {
	data, _ := json.Marshal(d)
	return string(data)
}

type DataConsumerDump struct {
	Id                         string                `json:"id,omitempty"`
	DataProducerId             string                `json:"dataProducerId,omitempty"`
	Type                       string                `json:"type,omitempty"`
	SctpStreamParameters       *SctpStreamParameters `json:"sctpStreamParameters,omitempty"`
	Label                      string                `json:"label,omitempty"`
	Protocol                   string                `json:"protocol,omitempty"`
	BufferedAmount             uint32                `json:"bufferedAmount,omitempty"`
	BufferedAmountLowThreshold uint32                `json:"bufferedAmountLowThreshold,omitempty"`
}

func (d DataConsumerDump) String() string {
	data, _ := json.Marshal(d)
	return string(data)
}

type DataProducerDump struct {
	Id                   string                `json:"id,omitempty"`
	Type                 string                `json:"type,omitempty"`
	SctpStreamParameters *SctpStreamParameters `json:"sctpStreamParameters,omitempty"`
	Label                string                `json:"label,omitempty"`
	Protocol             string                `json:"protocol,omitempty"`
}

func (d DataProducerDump) String() string {
	data, _ := json.Marshal(d)
	return string(data)
}

type RecvRtpHeaderExtensions struct {
	Mid               uint8 `json:"mid,omitempty"`
	Rid               uint8 `json:"rid,omitempty"`
	Rrid              uint8 `json:"rrid,omitempty"`
	AbsSendTime       uint8 `json:"absSendTime,omitempty"`
	TransportWideCc01 uint8 `json:"transportWideCc01,omitempty"`
}

type RtpListener struct {
	SsrcTable map[string]string `json:"ssrcTable,omitempty"`
	MidTable  map[string]string `json:"midTable,omitempty"`
	RidTable  map[string]string `json:"ridTable,omitempty"`
}

type SctpListener struct {
	StreamIdTable map[string]string `json:"streamIdTable,omitempty"`
}
