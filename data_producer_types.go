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
