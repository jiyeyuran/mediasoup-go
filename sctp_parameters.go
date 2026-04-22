package mediasoup

type SctpCapabilities struct {
	NumStreams NumSctpStreams `json:"numStreams,omitempty"`
}

// NumSctpStreams defines the SCTP streams configuration.
//
// Both OS and MIS are part of the SCTP INIT+ACK handshake. OS refers to the
// initial int of outgoing SCTP streams that the server side transport creates
// (to be used by DataConsumers), while MIS refers to the maximum int of
// incoming SCTP streams that the server side transport can receive (to be used
// by DataProducers). So, if the server side transport will just be used to
// create data producers (but no data consumers), OS can be low (~1).
// mediasoup-client provides specific per browser/version OS and MIS values via
// the device.sctpCapabilities getter. However those values must be reversed
// when provided to the mediasoup server transport.
type NumSctpStreams struct {
	// OS defines initially requested int of outgoing SCTP streams.
	OS uint16 `json:"OS,omitempty"`

	// MIS defines maximum int of incoming SCTP streams.
	MIS uint16 `json:"MIS,omitempty"`
}

// SctpParameters represents the SCTP parameters for a WebRTC data channel.
type SctpParameters struct {
	// Port represents the SCTP port number. Must always be 5000 as per WebRTC specification.
	Port uint16 `json:"port,omitempty"`

	// OS (Outgoing Streams) defines the initially requested number of outgoing SCTP streams.
	OS uint16 `json:"OS,omitempty"`

	// MIS (Maximum Incoming Streams) defines the maximum number of incoming SCTP streams.
	MIS uint16 `json:"MIS,omitempty"`

	// MaxMessageSize defines the maximum allowed size in bytes for SCTP messages.
	MaxMessageSize uint32 `json:"maxMessageSize,omitempty"`

	// Internal fields used for monitoring and debugging purposes:

	// SctpBufferedAmount indicates the number of bytes currently buffered in the SCTP stack.
	SctpBufferedAmount uint32 `json:"sctpBufferedAmount,omitempty"`

	// SendBufferSize represents the size of the SCTP send buffer in bytes.
	SendBufferSize uint32 `json:"sendBufferSize,omitempty"`

	// IsDataChannel indicates whether this SCTP association is used for WebRTC DataChannels.
	IsDataChannel bool `json:"isDataChannel,omitempty"`
}

// SctpStreamParameters describe the reliability of a certain SCTP stream.
// If ordered is true then maxPacketLifeTime and maxRetransmits must be false.
// If ordered if false, only one of maxPacketLifeTime or maxRetransmits can be true.
type SctpStreamParameters struct {
	// StreamId defines SCTP stream id.
	StreamId uint16 `json:"streamId"`

	// Ordered defines whether data messages must be received in order. If true the messages will
	// be sent reliably. Default true.
	Ordered *bool `json:"ordered,omitempty"`

	// MaxPacketLifeTime defines when ordered is false indicates the time (in milliseconds) after
	// which a SCTP packet will stop being retransmitted.
	MaxPacketLifeTime *uint16 `json:"maxPacketLifeTime,omitempty"`

	// MaxRetransmits defines when ordered is false indicates the maximum number of times a packet
	// will be retransmitted.
	MaxRetransmits *uint16 `json:"maxRetransmits,omitempty"`
}
