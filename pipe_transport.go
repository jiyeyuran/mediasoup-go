package mediasoup

import (
	"encoding/json"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
)

// PipeTransportOptions define options to create a PipeTransport
type PipeTransportOptions struct {
	// ListenIp define Listening IP address.
	ListenIp TransportListenIp `json:"listenIp,omitempty"`

	// EnableSctp define whether create a SCTP association. Default false.
	EnableSctp bool `json:"enableSctp,omitempty"`

	// NumSctpStreams define SCTP streams number.
	NumSctpStreams NumSctpStreams `json:"numSctpStreams,omitempty"`

	// MaxSctpMessageSize define maximum allowed size for SCTP messages sent by DataProducers.
	// Default 268435456.
	MaxSctpMessageSize int `json:"maxSctpMessageSize,omitempty"`

	// SctpSendBufferSize define maximum SCTP send buffer used by DataConsumers.
	// Default 268435456.
	SctpSendBufferSize int `json:"sctpSendBufferSize,omitempty"`

	// EnableSrtp enable SRTP. For this to work, connect() must be called
	// with remote SRTP parameters. Default false.
	EnableSrtp bool `json:"enableSrtp,omitempty"`

	// EnableRtx enable RTX and NACK for RTP retransmission. Useful if both Routers are
	// located in different hosts and there is packet lost in the link. For this
	// to work, both PipeTransports must enable this setting. Default false.
	EnableRtx bool `json:"enableRtx,omitempty"`

	// AppData is custom application data.
	AppData interface{} `json:"appData,omitempty"`
}

type pipeTransortData struct {
	Tuple          TransportTuple  `json:"tuple,omitempty"`
	SctpParameters SctpParameters  `json:"sctpParameters,omitempty"`
	SctpState      SctpState       `json:"sctpState,omitempty"`
	Rtx            bool            `json:"rtx,omitempty"`
	SrtpParameters *SrtpParameters `json:"srtpParameters,omitempty"`
}

// PipeTransport represents a network path through which RTP, RTCP (optionally secured with SRTP)
// and SCTP (DataChannel) is transmitted. Pipe transports are intented to intercommunicate two
// Router instances collocated on the same host or on separate hosts.
//
// - @emits sctpstatechange - (sctpState SctpState)
// - @emits trace - (trace *TransportTraceEventData)
type PipeTransport struct {
	ITransport
	logger            logr.Logger
	internal          internalData
	data              *pipeTransortData
	channel           *Channel
	payloadChannel    *PayloadChannel
	getProducerById   func(string) *Producer
	onSctpStateChange func(sctpState SctpState)
}

func newPipeTransport(params transportParams) ITransport {
	data := params.data.(*pipeTransortData)
	params.data = transportData{
		sctpParameters: data.SctpParameters,
		sctpState:      data.SctpState,
		transportType:  TransportType_Pipe,
	}
	params.logger = NewLogger("PipeTransport")

	transport := &PipeTransport{
		ITransport:      newTransport(params),
		logger:          params.logger,
		internal:        params.internal,
		data:            data,
		channel:         params.channel,
		payloadChannel:  params.payloadChannel,
		getProducerById: params.getProducerById,
	}

	transport.handleWorkerNotifications()

	return transport
}

// Tuple returns transport tuple.
func (t PipeTransport) Tuple() TransportTuple {
	return t.data.Tuple
}

// SctpParameters returns SCTP parameters.
func (t PipeTransport) SctpParameters() SctpParameters {
	return t.data.SctpParameters
}

// SctpState returns SCTP state.
func (t PipeTransport) SctpState() SctpState {
	return t.data.SctpState
}

// SrtpParameters returns SRTP parameters.
func (t PipeTransport) SrtpParameters() *SrtpParameters {
	return t.data.SrtpParameters
}

// Observer.
//
// - @emits close
// - @emits newproducer - (producer *Producer)
// - @emits newconsumer - (consumer *Consumer)
// - @emits newdataproducer - (dataProducer *DataProducer)
// - @emits newdataconsumer - (dataConsumer *DataConsumer)
// - @emits sctpstatechange - (sctpState SctpState)
// - @emits trace - (trace: TransportTraceEventData)
func (transport *PipeTransport) Observer() IEventEmitter {
	return transport.ITransport.Observer()
}

// Close the PipeTransport.
func (transport *PipeTransport) Close() {
	if transport.Closed() {
		return
	}

	if len(transport.data.SctpState) > 0 {
		transport.data.SctpState = SctpState_Closed
	}

	transport.ITransport.Close()
}

// routerClosed is called when router was closed.
func (transport *PipeTransport) routerClosed() {
	if transport.Closed() {
		return
	}

	if len(transport.data.SctpState) > 0 {
		transport.data.SctpState = SctpState_Closed
	}

	transport.ITransport.routerClosed()
}

// Connect provide the PlainTransport remote parameters.
func (transport *PipeTransport) Connect(options TransportConnectOptions) (err error) {
	transport.logger.V(1).Info("connect()")

	reqData := TransportConnectOptions{
		Ip:             options.Ip,
		Port:           options.Port,
		SrtpParameters: options.SrtpParameters,
	}
	resp := transport.channel.Request("transport.connect", transport.internal, reqData)

	var data struct {
		Tuple TransportTuple
	}
	if err = resp.Unmarshal(&data); err != nil {
		return
	}

	// Update data.
	transport.data.Tuple = data.Tuple

	return nil
}

// Consume create a pipe Consumer.
func (transport *PipeTransport) Consume(options ConsumerOptions) (consumer *Consumer, err error) {
	transport.logger.V(1).Info("consume()")

	producerId := options.ProducerId
	appData := options.AppData

	producer := transport.getProducerById(producerId)

	if producer == nil {
		err = fmt.Errorf(`Producer with id "%s" not found`, producerId)
		return
	}

	rtpParameters := getPipeConsumerRtpParameters(producer.ConsumableRtpParameters(), transport.data.Rtx)
	internal := transport.internal
	internal.ConsumerId = uuid.NewString()

	data := consumerData{
		ProducerId:    producerId,
		Kind:          producer.Kind(),
		RtpParameters: rtpParameters,
		Type:          "pipe",
	}

	reqData := struct {
		consumerData
		ConsumerId             string                  `json:"consumerId"`
		ConsumableRtpEncodings []RtpEncodingParameters `json:"consumableRtpEncodings"`
	}{
		consumerData:           data,
		ConsumerId:             internal.ConsumerId,
		ConsumableRtpEncodings: producer.ConsumableRtpParameters().Encodings,
	}

	resp := transport.channel.Request("transport.consume", internal, reqData)

	var status struct {
		Paused         bool
		ProducerPaused bool
	}
	if err = resp.Unmarshal(&status); err != nil {
		return
	}

	consumer = newConsumer(consumerParams{
		internal:       internal,
		data:           data,
		channel:        transport.channel,
		payloadChannel: transport.payloadChannel,
		appData:        appData,
		paused:         status.Paused,
		producerPaused: status.ProducerPaused,
	})

	baseTransport := transport.ITransport.(*Transport)

	baseTransport.consumers.Store(consumer.Id(), consumer)
	consumer.On("@close", func() {
		baseTransport.consumers.Delete(consumer.Id())
	})
	consumer.On("@producerclose", func() {
		baseTransport.consumers.Delete(consumer.Id())
	})

	// Emit observer event.
	transport.Observer().SafeEmit("newconsumer", consumer)

	return
}

// OnSctpStateChange set handler on "sctpstatechange" event
func (transport *PipeTransport) OnSctpStateChange(handler func(sctpState SctpState)) {
	transport.onSctpStateChange = handler
}

func (transport *PipeTransport) handleWorkerNotifications() {
	logger := transport.logger

	transport.channel.Subscribe(transport.Id(), func(event string, data []byte) {
		switch event {
		case "sctpstatechange":
			var result struct {
				SctpState SctpState
			}

			if err := json.Unmarshal([]byte(data), &result); err != nil {
				logger.Error(err, "failed to unmarshal sctpstatechange", "data", json.RawMessage(data))
				return
			}

			transport.data.SctpState = result.SctpState

			transport.SafeEmit("sctpstatechange", result.SctpState)

			// Emit observer event.
			transport.Observer().SafeEmit("sctpstatechange", result.SctpState)

			if handler := transport.onSctpStateChange; handler != nil {
				handler(result.SctpState)
			}

		default:
			transport.ITransport.handleEvent(event, data)
		}
	})
}
