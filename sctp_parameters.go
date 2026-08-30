package mediasoup

// SctpParameters represents the SCTP parameters of a transport.
type SctpParameters struct {
	// Port is the SCTP source port of the transport.
	Port uint16 `json:"port,omitempty"`

	// MaxSendMessageSize is the maximum size in bytes for SCTP messages sent by DataConsumers.
	MaxSendMessageSize uint32 `json:"maxSendMessageSize,omitempty"`

	// MaxReceiveMessageSize is the maximum size in bytes for SCTP messages received by
	// DataProducers.
	MaxReceiveMessageSize uint32 `json:"maxReceiveMessageSize,omitempty"`

	// SendBufferSize is the maximum SCTP send buffer in bytes used by DataConsumers.
	SendBufferSize uint32 `json:"sendBufferSize,omitempty"`

	// PerStreamSendQueueLimit is the per stream send queue size limit. Similar to
	// SendBufferSize, but limiting the size of individual streams.
	PerStreamSendQueueLimit uint32 `json:"perStreamSendQueueLimit,omitempty"`

	// MaxReceiverWindowBufferSize is the maximum received window buffer size in bytes.
	MaxReceiverWindowBufferSize uint32 `json:"maxReceiverWindowBufferSize,omitempty"`

	// IsDataChannel indicates whether this SCTP association is used for WebRTC DataChannels.
	// Only true in WebRTC transports.
	IsDataChannel bool `json:"isDataChannel,omitempty"`
}

// SctpNegotiatedCapabilities holds the SCTP capabilities negotiated with the remote endpoint
// once the SCTP association is established.
type SctpNegotiatedCapabilities struct {
	// NegotiatedMaxOutboundStreams is the number of outgoing SCTP streams usable by DataConsumers.
	NegotiatedMaxOutboundStreams uint16 `json:"negotiatedMaxOutboundStreams"`

	// NegotiatedMaxInboundStreams is the number of incoming SCTP streams usable by DataProducers.
	NegotiatedMaxInboundStreams uint16 `json:"negotiatedMaxInboundStreams"`
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
