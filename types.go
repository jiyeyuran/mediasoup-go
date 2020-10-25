package mediasoup

import "encoding/json"

const (
	PPID_WEBRTC_STRING int = 51
	PPID_WEBRTC_BINARY     = 53
)

type H map[string]interface{}

func Bool(b bool) *bool {
	return &b
}

type DumpResult struct {
	Data []byte
	Err  error
}

func NewDumpResult(data []byte, err error) DumpResult {
	return DumpResult{
		Data: data,
		Err:  err,
	}
}

func (r DumpResult) Unmarshal(v interface{}) error {
	if r.Err != nil {
		return r.Err
	}
	return json.Unmarshal(r.Data, v)
}

type WorkerDump struct {
	Pid       string   `json:"pid,omitempty"`
	RouterIds []string `json:"routerIds,omitempty"`
}

type RouterDump struct {
	Id                               string              `json:"id,omitempty"`
	MapProducerIdConsumerIds         map[string][]string `json:"mapProducerIdConsumerIds,omitempty"`
	MapConsumerIdProducerId          map[string]string   `json:"mapConsumerIdProducerId,omitempty"`
	MapDataProducerIdDataConsumerIds map[string][]string `json:"mapDataProducerIdDataConsumerIds,omitempty"`
	MapDataConsumerIdDataProducerId  map[string]string   `json:"mapDataConsumerIdDataProducerId,omitempty"`
	MapProducerIdObserverIds         map[string][]string `json:"mapProducerIdObserverIds,omitempty"`
	RtpObserverIds                   []string            `json:"rtpObserverIds,omitempty"`
	TransportIds                     []string            `json:"transportIds,omitempty"`
}

type TransportDump struct {
	Id                      string                  `json:"id,omitempty"`
	Direct                  bool                    `json:"direct,omitempty"`
	ProducerIds             []string                `json:"producerIds,omitempty"`
	ConsumerIds             []string                `json:"consumerIds,omitempty"`
	MapSsrcConsumerId       map[uint32]uint32       `json:"mapSsrcConsumerId,omitempty"`
	MapRtxSsrcConsumerId    map[uint32]uint32       `json:"mapRtxSsrcConsumerId,omitempty"`
	DataProducerIds         []string                `json:"dataProducerIds,omitempty"`
	DataConsumerIds         []string                `json:"dataConsumerIds,omitempty"`
	RecvRtpHeaderExtensions RecvRtpHeaderExtensions `json:"recvRtpHeaderExtensions,omitempty"`
	RtpListener             RtpListener             `json:"rtpListener,omitempty"`
	SctpParameters          *SctpParametersDump     `json:"SctpParameters,omitempty"`
	SctpState               SctpState               `json:"sctpState,omitempty"`
	SctpListener            SctpListener            `json:"sctpListener,omitempty"`
	TraceEventTypes         string                  `json:"traceEventTypes,omitempty"`
}

type ConsumerDump struct {
	Id                         string               `json:"id,omitempty"`
	Kind                       string               `json:"kind,omitempty"`
	Type                       string               `json:"type,omitempty"`
	RtpParameters              RtpParameters        `json:"rtpParameters,omitempty"`
	ConsumableRtpEncodings     []RtpMappingEncoding `json:"consumableRtpEncodings,omitempty"`
	SupportedCodecPayloadTypes []uint32             `json:"supportedCodecPayloadTypes,omitempty"`
	Paused                     bool                 `json:"paused,omitempty"`
	ProducerPaused             bool                 `json:"producerPaused,omitempty"`
	TraceEventTypes            string               `json:"traceEventTypes,omitempty"`
}

type DataConsumerDump struct {
	Id                   string                `json:"id,omitempty"`
	DataProducerId       string                `json:"dataProducerId,omitempty"`
	Type                 string                `json:"type,omitempty"`
	SctpStreamParameters *SctpStreamParameters `json:"sctpStreamParameters,omitempty"`
	Label                string                `json:"label,omitempty"`
	Protocol             string                `json:"protocol,omitempty"`
}

type RecvRtpHeaderExtensions struct {
	Mid               uint8 `json:"mid,omitempty"`
	Rid               uint8 `json:"rid,omitempty"`
	Rrid              uint8 `json:"rrid,omitempty"`
	AbsSendTime       uint8 `json:"absSendTime,omitempty"`
	TransportWideCc01 uint8 `json:"transportWideCc01,omitempty"`
}

type RtpListener struct {
	SsrcTable map[uint32]string `json:"ssrcTable,omitempty"`
	MidTable  map[string]string `json:"midTable,omitempty"`
	RidTable  map[string]string `json:"ridTable,omitempty"`
}

type SctpParametersDump struct {
	SctpParameters
	SctpBufferedAmount uint32 `json:"sctpBufferedAmount,omitempty"`
	IsDataChannel      bool   `json:"isDataChannel,omitempty"`
}

type SctpListener struct {
	StreamIdTable map[uint16]string `json:"streamIdTable,omitempty"`
}
