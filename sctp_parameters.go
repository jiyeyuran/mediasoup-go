package mediasoup

type SctpCapabilities struct {
	NumStreams NumSctpStreams `json:"numStreams"`
}

/**
 * Both OS and MIS are part of the SCTP INIT+ACK handshake. OS refers to the
 * initial int of outgoing SCTP streams that the server side transport creates
 * (to be used by DataConsumers), while MIS refers to the maximum int of
 * incoming SCTP streams that the server side transport can receive (to be used
 * by DataProducers). So, if the server side transport will just be used to
 * create data producers (but no data consumers), OS can be low (~1). However,
 * if data consumers are desired on the server side transport, OS must have a
 * proper value and such a proper value depends on whether the remote endpoint
 * supports  SCTP_ADD_STREAMS extension or not.
 *
 * libwebrtc (Chrome, Safari, etc) does not enable SCTP_ADD_STREAMS so, if data
 * consumers are required,  OS should be 1024 (the maximum int of DataChannels
 * that libwebrtc enables).
 *
 * Firefox does enable SCTP_ADD_STREAMS so, if data consumers are required, OS
 * can be lower (16 for instance). The mediasoup transport will allocate and
 * announce more outgoing SCTM streams when needed.
 *
 * mediasoup-client provides specific per browser/version OS and MIS values via
 * the device.sctpCapabilities getter.
 */
type NumSctpStreams struct {
	/**
	 * Initially requested int of outgoing SCTP streams.
	 */
	OS uint16 `json:"OS"`

	/**
	 * Maximum int of incoming SCTP streams.
	 */
	MIS uint16 `json:"MIS"`
}

type SctpParameters struct {
	/**
	 * Must always equal 5000.
	 */
	Port uint16 `json:"port"`

	/**
	 * Initially requested int of outgoing SCTP streams.
	 */
	OS uint16 `json:"os"`

	/**
	 * Maximum int of incoming SCTP streams.
	 */
	MIS uint16 `json:"mis"`

	/**
	 * Maximum allowed size for SCTP messages.
	 */
	MaxMessageSize uint32 `json:"maxMessageSize"`

	/**
	 * Set by worker.
	 */
	IsDataChannel      bool `json:"isDataChannel,omitempty"`
	SctpBufferedAmount int  `json:"sctpBufferedAmount,omitempty"`
	SendBufferSize     int  `json:"sendBufferSize,omitempty"`
}

/**
 * SCTP stream parameters describe the reliability of a certain SCTP stream.
 * If ordered is true then maxPacketLifeTime and maxRetransmits must be
 * false.
 * If ordered if false, only one of maxPacketLifeTime or maxRetransmits
 * can be true.
 */
type SctpStreamParameters struct {
	/**
	 * SCTP stream id.
	 */
	StreamId uint16 `json:"streamId"`

	/**
	 * Whether data messages must be received in order. If true the messages will
	 * be sent reliably. Default true.
	 */
	Ordered *bool `json:"ordered,omitempty"`

	/**
	 * When ordered is false indicates the time (in milliseconds) after which a
	 * SCTP packet will stop being retransmitted.
	 */
	MaxPacketLifeTime uint16 `json:"maxPacketLifeTime,omitempty"`

	/**
	 * When ordered is false indicates the maximum number of times a packet will
	 * be retransmitted.
	 */
	MaxRetransmits uint16 `json:"maxRetransmits,omitempty"`
}
