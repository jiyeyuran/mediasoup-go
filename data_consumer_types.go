package mediasoup

// DataConsumerOptions define options to create a DataConsumer.
type DataConsumerOptions struct {
	// DataProducerId is the id of the DataProducer to consume.
	DataProducerId string `json:"dataProducerId,omitempty"`

	// Ordered define just if consuming over SCTP.
	// Whether data messages must be received in order. If true the messages will
	// be sent reliably. Defaults to the value in the DataProducer if it has type
	// "sctp" or to true if it has type "direct".
	Ordered *bool `json:"ordered,omitempty"`

	// MaxPacketLifeTime define just if consuming over SCTP.
	// When ordered is false indicates the time (in milliseconds) after which a
	// SCTP packet will stop being retransmitted. Defaults to the value in the
	// DataProducer if it has type 'sctp' or unset if it has type 'direct'.
	MaxPacketLifeTime uint16 `json:"maxPacketLifeTime,omitempty"`

	// MaxRetransmits define just if consuming over SCTP.
	// When ordered is false indicates the maximum number of times a packet will
	// be retransmitted. Defaults to the value in the DataProducer if it has type
	// 'sctp' or unset if it has type 'direct'.
	MaxRetransmits uint16 `json:"maxRetransmits,omitempty"`

	// Paused indicates whether the data consumer must start in paused mode. Default false.
	Paused bool `json:"paused,omitempty"`

	// Subchannels this data consumer initially subscribes to.
	// Only used in case this data consumer receives messages from a local data
	// producer that specifies subchannel(s) when calling send().
	Subchannels []uint16 `json:"subchannels,omitempty"`

	// AppData is custom application data.
	AppData H `json:"appData,omitempty"`
}

// SctpPayloadType is an enum for DataChannel payload types
type SctpPayloadType uint32

const (
	// https://www.iana.org/assignments/sctp-parameters/sctp-parameters.xhtml#sctp-parameters-25
	SctpPayloadUnknown           SctpPayloadType = 0
	SctpPayloadWebRTCDCEP        SctpPayloadType = 50
	SctpPayloadWebRTCString      SctpPayloadType = 51
	SctpPayloadWebRTCBinary      SctpPayloadType = 53
	SctpPayloadWebRTCStringEmpty SctpPayloadType = 56
	SctpPayloadWebRTCBinaryEmpty SctpPayloadType = 57
)

// DataConsumerType define DataConsumer type.
type DataConsumerType string

const (
	DataConsumerSctp   DataConsumerType = "sctp"
	DataConsumerDirect DataConsumerType = "direct"
)

// DataConsumerDump define the dump info for DataConsumer.
type DataConsumerDump struct {
	Id                         string                `json:"id,omitempty"`
	Paused                     bool                  `json:"paused,omitempty"`
	Subchannels                []uint16              `json:"subchannels,omitempty"`
	DataProducerId             string                `json:"dataProducerId,omitempty"`
	Type                       DataConsumerType      `json:"type,omitempty"`
	SctpStreamParameters       *SctpStreamParameters `json:"sctpStreamParameters,omitempty"`
	Label                      string                `json:"label,omitempty"`
	Protocol                   string                `json:"protocol,omitempty"`
	BufferedAmountLowThreshold uint32                `json:"bufferedAmountLowThreshold,omitempty"`
}

// DataConsumerStat define the statistic info for DataConsumer.
type DataConsumerStat struct {
	Type           string `json:"type,omitempty"`
	Timestamp      uint64 `json:"timestamp,omitempty"`
	Label          string `json:"label,omitempty"`
	Protocol       string `json:"protocol,omitempty"`
	MessagesSent   uint64 `json:"messagesSent,omitempty"`
	BytesSent      uint64 `json:"bytesSent,omitempty"`
	BufferedAmount uint32 `json:"bufferedAmount,omitempty"`
}

type DataConsumerSendOptions struct {
	// PPID specifies the SCTP Payload Protocol Identifier to be used when sending
	PPID SctpPayloadType
}

type DataConsumerSendOption func(*DataConsumerSendOptions)

func DataConsumerSendWithPayloadType(ppid SctpPayloadType) DataConsumerSendOption {
	return func(o *DataConsumerSendOptions) {
		if ppid != SctpPayloadUnknown {
			o.PPID = ppid
		}
	}
}
