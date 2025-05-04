package mediasoup

// RouterOptions defines the configuration parameters for creating a new Router instance.
type RouterOptions struct {
	// MediaCodecs defines Router media codecs
	MediaCodecs []*RtpCodecCapability `json:"mediaCodecs,omitempty"`

	// AppData is custom application data
	AppData H `json:"appData,omitempty"`
}

// PipeToRouterOptions defines options to pipe to another router
type PipeToRouterOptions struct {
	// ListenInfo define Listening IP address.
	ListenInfo TransportListenInfo `json:"listenInfo,omitempty"`

	// ProducerId is the id of the Producer to consume
	ProducerId string `json:"producerId,omitempty"`

	// DataProducerId is the id of the DataProducer to consume
	DataProducerId string `json:"dataProducerId,omitempty"`

	// Router is the target Router instance
	Router *Router `json:"-"`

	// EnableSctp creates a SCTP association. Default true
	EnableSctp *bool `json:"enableSctp,omitempty"`

	// NumSctpStreams configures SCTP streams
	NumSctpStreams *NumSctpStreams `json:"numSctpStreams,omitempty"`

	// EnableRtx enables RTX and NACK for RTP retransmission
	EnableRtx bool `json:"enableRtx,omitempty"`

	// EnableSrtp enables SRTP
	EnableSrtp bool `json:"enableSrtp,omitempty"`
}

// PipeToRouterResult contains the result of piping router
type PipeToRouterResult struct {
	// PipeConsumer is the Consumer created in the current Router
	PipeConsumer *Consumer

	// PipeProducer is the Producer created in the target Router
	PipeProducer *Producer

	// PipeDataConsumer is the DataConsumer created in the current Router
	PipeDataConsumer *DataConsumer

	// PipeDataProducer is the DataProducer created in the target Router
	PipeDataProducer *DataProducer
}

// RouterDump represents the dump of a Router.
type RouterDump struct {
	// The Router id.
	Id string `json:"id,omitempty"`

	// Id of Transports.
	TransportIds []string `json:"transportIds,omitempty"`

	// Id of RtpObservers.
	RtpObserverIds []string `json:"rtpObserverIds,omitempty"`

	// Array of Producer id and its respective Consumer ids.
	MapProducerIdConsumerIds []KeyValues[string, string] `json:"mapProducerIdConsumerIds,omitempty"`

	// Array of Consumer id and its Producer id.
	MapConsumerIdProducerId []KeyValue[string, string] `json:"mapConsumerIdProducerId,omitempty"`

	// Array of Producer id and its respective Observer ids.
	MapProducerIdObserverIds []KeyValues[string, string] `json:"mapProducerIdObserverIds,omitempty"`

	// Array of Producer id and its respective DataConsumer ids.
	MapDataProducerIdDataConsumerIds []KeyValues[string, string] `json:"mapDataProducerIdDataConsumerIds,omitempty"`

	// Array of DataConsumer id and its DataProducer id.
	MapDataConsumerIdDataProducerId []KeyValue[string, string] `json:"mapDataConsumerIdDataProducerId,omitempty"`
}
