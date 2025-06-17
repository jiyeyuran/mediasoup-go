package mediasoup

// DataProducerOptions define options to create a DataProducer.
type DataProducerOptions struct {
	// Id is DataProducer id (just for Router.PipeToRouter() method).
	Id string `json:"id,omitempty"`

	// SctpStreamParameters define how the endpoint is sending the data.
	// Just if messages are sent over SCTP.
	SctpStreamParameters *SctpStreamParameters `json:"sctpStreamParameters,omitempty"`

	// Label can be used to distinguish this DataChannel from others.
	Label string `json:"label,omitempty"`

	// Protocol is the name of the sub-protocol used by this DataChannel.
	Protocol string `json:"protocol,omitempty"`

	// Paused indicate whether the data producer must start in paused mode. Default false.
	Paused bool `json:"paused,omitempty"`

	// AppData is custom application data.
	AppData H `json:"appData,omitempty"`
}

// DataProducerType define DataProducer type.
type DataProducerType string

const (
	DataProducerSctp   DataProducerType = "sctp"
	DataProducerDirect DataProducerType = "direct"
)

// DataProducerDump define the dump info for DataProducer.
type DataProducerDump struct {
	Id                   string                `json:"id,omitempty"`
	Paused               bool                  `json:"paused,omitempty"`
	Type                 DataProducerType      `json:"type,omitempty"`
	SctpStreamParameters *SctpStreamParameters `json:"sctpStreamParameters,omitempty"`
	Label                string                `json:"label,omitempty"`
	Protocol             string                `json:"protocol,omitempty"`
}

// DataProducerStat define the statistic info for DataProducer.
type DataProducerStat struct {
	Type             string `json:"type,omitempty"`
	Timestamp        uint64 `json:"timestamp,omitempty"`
	Label            string `json:"label,omitempty"`
	Protocol         string `json:"protocol,omitempty"`
	MessagesReceived uint64 `json:"messagesReceived,omitempty"`
	BytesReceived    uint64 `json:"bytesReceived,omitempty"`
}

type DataProducerSendOptions struct {
	// Subchannels specifies that only data consumers subscribed to any of these
	// subchannels will receive the message.
	Subchannels []uint16 `json:"subchannels,omitempty"`

	// RequiredSubchannel specifies that only data consumers subscribed to this specific
	// subchannel will receive the message.
	RequiredSubchannel *uint16 `json:"requiredSubchannel,omitempty"`

	// PPID specifies the SCTP Payload Protocol Identifier to be used when sending
	PPID SctpPayloadType
}

type DataProducerSendOption func(*DataProducerSendOptions)

func DataProducerSendWithSubchannels(subchannels []uint16) DataProducerSendOption {
	return func(o *DataProducerSendOptions) {
		o.Subchannels = subchannels
	}
}

func DataProducerSendWithRequiredSubchannel(subchannel uint16) DataProducerSendOption {
	return func(o *DataProducerSendOptions) {
		o.RequiredSubchannel = &subchannel
	}
}

func DataProducerSendWithPayloadType(ppid SctpPayloadType) DataProducerSendOption {
	return func(o *DataProducerSendOptions) {
		if ppid != SctpPayloadUnknown {
			o.PPID = ppid
		}
	}
}
