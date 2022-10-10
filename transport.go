package mediasoup

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
)

type ITransport interface {
	IEventEmitter
	Id() string
	Closed() bool
	AppData() interface{}
	Observer() IEventEmitter
	Close()
	Dump() (*TransportDump, error)
	GetStats() ([]*TransportStat, error)
	Connect(TransportConnectOptions) error
	SetMaxIncomingBitrate(bitrate int) error
	Produce(ProducerOptions) (*Producer, error)
	Consume(ConsumerOptions) (*Consumer, error)
	ProduceData(DataProducerOptions) (*DataProducer, error)
	ConsumeData(DataConsumerOptions) (*DataConsumer, error)
	EnableTraceEvent(types ...TransportTraceEventType) error
	OnTrace(handler func(trace *TransportTraceEventData))
	OnClose(handler func())

	// internal methods
	routerClosed()
	listenServerClosed()
	handleEvent(event string, data []byte)
}

type TransportListenIp struct {
	// Listening IPv4 or IPv6.
	Ip string `json:"ip,omitempty"`

	// Announced IPv4 or IPv6 (useful when running mediasoup behind NAT with private IP).
	AnnouncedIp string `json:"announcedIp,omitempty"`
}

// Transport protocol.
type TransportProtocol string

const (
	TransportProtocol_Udp TransportProtocol = "udp"
	TransportProtocol_Tcp TransportProtocol = "tcp"
)

type TransportTraceEventType string

const (
	TransportTraceEventType_Probation TransportTraceEventType = "probation"
	TransportTraceEventType_Bwe       TransportTraceEventType = "bwe"
)

type TransportTuple struct {
	LocalIp    string `json:"localIp,omitempty"`
	LocalPort  uint16 `json:"localPort,omitempty"`
	RemoteIp   string `json:"remoteIp,omitempty"`
	RemotePort uint16 `json:"remotePort,omitempty"`
	Protocol   string `json:"protocol,omitempty"`
}

type TransportTraceEventData struct {
	// Trace type.
	Type TransportTraceEventType `json:"type,omitempty"`

	// Event timestamp.
	Timestamp int64 `json:"timestamp,omitempty"`

	// Event direction.
	Direction string `json:"direction,omitempty"`

	// Per type information.
	Info interface{} `json:"info,omitempty"`
}

type SctpState string

const (
	SctpState_New        = "new"
	SctpState_Connecting = "connecting"
	SctpState_Connected  = "connected"
	SctpState_Failed     = "failed"
	SctpState_Closed     = "closed"
)

type TransportStat struct {
	// Common to all Transports.
	Type                     string    `json:"type,omitempty"`
	TransportId              string    `json:"transportId,omitempty"`
	Timestamp                int64     `json:"timestamp,omitempty"`
	SctpState                SctpState `json:"sctpState,omitempty"`
	BytesReceived            int64     `json:"bytesReceived,omitempty"`
	RecvBitrate              int64     `json:"recvBitrate,omitempty"`
	BytesSent                int64     `json:"bytesSent,omitempty"`
	SendBitrate              int64     `json:"sendBitrate,omitempty"`
	RtpBytesReceived         int64     `json:"rtpBytesReceived,omitempty"`
	RtpRecvBitrate           int64     `json:"rtpRecvBitrate,omitempty"`
	RtpBytesSent             int64     `json:"rtpBytesSent,omitempty"`
	RtpSendBitrate           int64     `json:"rtpSendBitrate,omitempty"`
	RtxBytesReceived         int64     `json:"rtxBytesReceived,omitempty"`
	RtxRecvBitrate           int64     `json:"rtxRecvBitrate,omitempty"`
	RtxBytesSent             int64     `json:"rtxBytesSent,omitempty"`
	RtxSendBitrate           int64     `json:"rtxSendBitrate,omitempty"`
	ProbationBytesSent       int64     `json:"probationBytesSent,omitempty"`
	ProbationSendBitrate     int64     `json:"probationSendBitrate,omitempty"`
	AvailableOutgoingBitrate int64     `json:"availableOutgoingBitrate,omitempty"`
	AvailableIncomingBitrate int64     `json:"availableIncomingBitrate,omitempty"`
	MaxIncomingBitrate       int64     `json:"maxIncomingBitrate,omitempty"`
	RtpPacketLossReceived    float64   `json:"rtpPacketLossReceived,omitempty"`
	RtpPacketLossSent        float64   `json:"rtpPacketLossSent,omitempty"`

	*WebRtcTransportSpecificStat
	*PlainTransportSpecificStat // share tuple with pipe transport stat
}

type TransportConnectOptions struct {
	// pipe and plain transport
	Ip             string          `json:"ip,omitempty"`
	Port           uint16          `json:"port,omitempty"`
	SrtpParameters *SrtpParameters `json:"srtpParameters,omitempty"`

	// plain transport
	RtcpPort uint16 `json:"rtcpPort,omitempty"`

	// webrtc transport
	DtlsParameters *DtlsParameters `json:"dtlsParameters,omitempty"`
}

type TransportType string

const (
	TransportType_Direct TransportType = "DirectTransport"
	TransportType_Plain  TransportType = "PlainTransport"
	TransportType_Pipe   TransportType = "PipeTransport"
	TransportType_Webrtc TransportType = "WebrtcTransport"
)

type transportData struct {
	sctpParameters SctpParameters
	sctpState      SctpState
	transportType  TransportType
}

type transportParams struct {
	// routerId, transportId
	internal                 internalData
	data                     interface{}
	channel                  *Channel
	payloadChannel           *PayloadChannel
	appData                  interface{}
	getRouterRtpCapabilities func() RtpCapabilities
	getProducerById          func(string) *Producer
	getDataProducerById      func(string) *DataProducer
	logger                   logr.Logger
}

// Transport is a base class inherited by PlainTransport, PipeTransport, DirectTransport and WebRtcTransport.
//
// - @emits routerclose
// - @emits @close
// - @emits @newproducer - (producer *Producer)
// - @emits @producerclose - (producer *Producer)
// - @emits @newdataproducer - (dataProducer *DataProducer)
// - @emits @dataproducerclose - (dataProducer *DataProducer)
type Transport struct {
	IEventEmitter
	logger logr.Logger
	// Internal data.
	internal internalData
	// Transport data. This is set by the subclass.
	data transportData
	// Channel instance.
	channel *Channel
	// PayloadChannel instance.
	payloadChannel *PayloadChannel
	// Close flag.
	closed uint32
	// Custom app data.
	appData interface{}
	// Method to retrieve Router RTP capabilities.
	getRouterRtpCapabilities func() RtpCapabilities
	// Method to retrieve a Producer.
	getProducerById func(string) *Producer
	// Method to retrieve a DataProducer.
	getDataProducerById func(string) *DataProducer
	// Producers map.
	producers sync.Map
	// Consumers map.
	consumers sync.Map
	// DataProducers map.
	dataProducers sync.Map
	// DataConsumers map.
	dataConsumers sync.Map
	// RTCP CNAME for Producers.
	cnameForProducers string
	// Next MID for Consumers. It's converted into string when used.
	nextMidForConsumers uint32
	// Buffer with available SCTP stream ids.
	sctpStreamIds []byte
	// Next SCTP stream id.
	nextSctpStreamId int
	// Deprecated
	observer IEventEmitter
	// locker instance
	locker sync.Mutex

	onTrace func(*TransportTraceEventData)
	onClose func()
}

func newTransport(params transportParams) ITransport {
	params.logger.V(1).Info("constructor()", "internal", params.internal)

	transport := &Transport{
		IEventEmitter:            NewEventEmitter(),
		logger:                   params.logger,
		internal:                 params.internal,
		data:                     params.data.(transportData),
		channel:                  params.channel,
		payloadChannel:           params.payloadChannel,
		appData:                  params.appData,
		getRouterRtpCapabilities: params.getRouterRtpCapabilities,
		getProducerById:          params.getProducerById,
		getDataProducerById:      params.getDataProducerById,
		observer:                 NewEventEmitter(),
	}

	return transport
}

// Id returns Transport id
func (transport *Transport) Id() string {
	return transport.internal.TransportId
}

// Closed returns Whether the Transport is closed.
func (transport *Transport) Closed() bool {
	return atomic.LoadUint32(&transport.closed) > 0
}

// AppData returns app custom data.
func (transport *Transport) AppData() interface{} {
	return transport.appData
}

// Deprecated
//
// - @emits close
// - @emits newproducer - (producer *Producer)
// - @emits newconsumer - (producer *Producer)
// - @emits newdataproducer - (dataProducer *DataProducer)
// - @emits newdataconsumer - (dataProducer *DataProducer)
func (transport *Transport) Observer() IEventEmitter {
	return transport.observer
}

// Close the Transport.
func (transport *Transport) Close() {
	if atomic.CompareAndSwapUint32(&transport.closed, 0, 1) {
		transport.logger.V(1).Info("close()")

		// Remove notification subscriptions.
		transport.channel.Unsubscribe(transport.Id())
		transport.payloadChannel.Unsubscribe(transport.Id())

		reqData := H{"transportId": transport.internal.TransportId}

		transport.channel.Request("router.closeTransport", transport.internal, reqData)

		transport.producers.Range(func(key, value interface{}) bool {
			producer := value.(*Producer)

			producer.transportClosed()
			transport.Emit("@producerclose", producer)

			return true
		})

		transport.consumers.Range(func(key, value interface{}) bool {
			value.(*Consumer).transportClosed()

			return true
		})

		transport.dataProducers.Range(func(key, value interface{}) bool {
			producer := value.(*DataProducer)

			producer.transportClosed()
			transport.Emit("@dataproducerclose", producer)

			return true
		})

		transport.dataConsumers.Range(func(key, value interface{}) bool {
			value.(*DataConsumer).transportClosed()

			return true
		})

		transport.Emit("@close")
		transport.RemoveAllListeners()

		transport.close()
	}
}

// close send "close" event
func (transport *Transport) close() {
	// Emit observer event.
	transport.observer.SafeEmit("close")
	transport.observer.RemoveAllListeners()

	if handler := transport.onClose; handler != nil {
		handler()
	}
}

// routerClosed is called when Router was closed.
func (transport *Transport) routerClosed() {
	if atomic.CompareAndSwapUint32(&transport.closed, 0, 1) {
		transport.logger.V(1).Info("routerClosed()")

		// Remove notification subscriptions.
		transport.channel.Unsubscribe(transport.Id())
		transport.payloadChannel.Unsubscribe(transport.Id())

		transport.producers.Range(func(key, value interface{}) bool {
			producer := value.(*Producer)

			producer.transportClosed()
			transport.Emit("@producerclose", producer)

			return true
		})

		transport.consumers.Range(func(key, value interface{}) bool {
			value.(*Consumer).transportClosed()

			return true
		})

		transport.dataProducers.Range(func(key, value interface{}) bool {
			producer := value.(*DataProducer)

			producer.transportClosed()
			transport.Emit("@dataproducerclose", producer)

			return true
		})

		transport.dataConsumers.Range(func(key, value interface{}) bool {
			value.(*DataConsumer).transportClosed()

			return true
		})

		transport.SafeEmit("routerclose")
		transport.RemoveAllListeners()

		transport.close()
	}
}

// listenServerClosed is called when listen server was closed (this just happens
// in WebRtcTransports when their associated WebRtcServer is closed).
func (transport *Transport) listenServerClosed() {
	if !atomic.CompareAndSwapUint32(&transport.closed, 0, 1) {
		return
	}
	transport.logger.V(1).Info("listenServerClosed()")

	// Remove notification subscriptions.
	transport.channel.Unsubscribe(transport.Id())
	transport.payloadChannel.Unsubscribe(transport.Id())

	// Close every Producer.
	transport.producers.Range(func(key, value interface{}) bool {
		producer := value.(*Producer)
		producer.transportClosed()
		// NOTE: No need to tell the Router since it already knows (it has
		// been closed in fact).
		return true
	})
	transport.producers = sync.Map{}

	// Close every Consumer.
	transport.consumers.Range(func(key, value interface{}) bool {
		consumer := value.(*Consumer)
		consumer.transportClosed()
		return true
	})
	transport.consumers = sync.Map{}

	// Close every DataProducer.
	transport.dataProducers.Range(func(key, value interface{}) bool {
		producer := value.(*DataProducer)
		producer.transportClosed()
		// NOTE: No need to tell the Router since it already knows (it has
		// been closed in fact).
		return true
	})
	transport.dataProducers = sync.Map{}

	// Close every DataConsumer.
	transport.dataConsumers.Range(func(key, value interface{}) bool {
		consumer := value.(*DataConsumer)
		consumer.transportClosed()
		return true
	})
	transport.dataConsumers = sync.Map{}

	// Need to emit this event to let the parent Router know since
	// transport.listenServerClosed() is called by the listen server.
	// NOTE: Currently there is just WebRtcServer for WebRtcTransports.
	transport.Emit("@listenserverclose")

	transport.SafeEmit("listenserverclose")

	transport.close()
}

// Dump Transport.
func (transport *Transport) Dump() (data *TransportDump, err error) {
	transport.logger.V(1).Info("dump()")

	resp := transport.channel.Request("transport.dump", transport.internal)
	err = resp.Unmarshal(&data)

	return
}

// GetStats returns the Transport stats.
func (transport *Transport) GetStats() (stat []*TransportStat, err error) {
	transport.logger.V(1).Info("getStats()")

	resp := transport.channel.Request("transport.getStats", transport.internal)
	err = resp.Unmarshal(&stat)

	return
}

// Connect provide the Transport remote parameters.
func (transport *Transport) Connect(TransportConnectOptions) error {
	return errors.New("method not implemented in the subclass")
}

// SetMaxIncomingBitrate set maximum incoming bitrate for receiving media.
func (transport *Transport) SetMaxIncomingBitrate(bitrate int) error {
	transport.logger.V(1).Info("SetMaxIncomingBitrate()", "bitrate", bitrate)

	resp := transport.channel.Request(
		"transport.setMaxIncomingBitrate", transport.internal, H{"bitrate": bitrate})

	return resp.Err()
}

// Produce creates a Producer.
func (transport *Transport) Produce(options ProducerOptions) (producer *Producer, err error) {
	transport.logger.V(1).Info("produce()")

	id := options.Id
	kind := options.Kind
	rtpParameters := options.RtpParameters
	paused := options.Paused
	keyFrameRequestDelay := options.KeyFrameRequestDelay
	appData := options.AppData

	if len(id) > 0 {
		if _, ok := transport.producers.Load(id); ok {
			err = NewTypeError(`a Producer with same id "%s" already exists`, id)
			return
		}
	} else {
		id = uuid.NewString()
	}

	// This may throw.
	if err = validateRtpParameters(&rtpParameters); err != nil {
		return
	}

	// If missing or empty encodings, add one.
	if len(rtpParameters.Encodings) == 0 {
		rtpParameters.Encodings = []RtpEncodingParameters{{}}
	}

	// Don't do this in PipeTransports since there we must keep CNAME value in each Producer.
	if transport.data.transportType != TransportType_Pipe {
		// If CNAME is given and we don't have yet a CNAME for Producers in this Transport, take it.
		if len(transport.cnameForProducers) == 0 && len(rtpParameters.Rtcp.Cname) > 0 {
			transport.cnameForProducers = rtpParameters.Rtcp.Cname
		} else if len(transport.cnameForProducers) == 0 {
			// Otherwise if we don't have yet a CNAME for Producers and the RTP parameters
			// do not include CNAME, create a random one.
			transport.cnameForProducers = uuid.NewString()[:8]
		}

		// Override Producer's CNAME.
		rtpParameters.Rtcp.Cname = transport.cnameForProducers
	}

	routerRtpCapabilities := transport.getRouterRtpCapabilities()

	rtpMapping, err := getProducerRtpParametersMapping(
		rtpParameters, routerRtpCapabilities)
	if err != nil {
		return
	}

	consumableRtpParameters, err := getConsumableRtpParameters(
		kind, rtpParameters, routerRtpCapabilities, rtpMapping)
	if err != nil {
		return
	}

	internal := transport.internal
	internal.ProducerId = id

	reqData := H{
		"producerId":           id,
		"kind":                 kind,
		"rtpParameters":        rtpParameters,
		"rtpMapping":           rtpMapping,
		"keyFrameRequestDelay": keyFrameRequestDelay,
		"paused":               paused,
	}
	resp := transport.channel.Request("transport.produce", internal, reqData)

	var status struct {
		Type ProducerType
	}
	if err = resp.Unmarshal(&status); err != nil {
		return
	}

	producerData := producerData{
		Kind:                    kind,
		RtpParameters:           rtpParameters,
		Type:                    status.Type,
		ConsumableRtpParameters: consumableRtpParameters,
	}

	producer = newProducer(producerParams{
		internal:       internal,
		data:           producerData,
		channel:        transport.channel,
		payloadChannel: transport.payloadChannel,
		appData:        appData,
		paused:         paused,
	})

	transport.producers.Store(producer.Id(), producer)

	producer.On("@close", func() {
		transport.producers.Delete(producer.Id())
		transport.Emit("@producerclose", producer)
	})

	transport.Emit("@newproducer", producer)

	// Emit observer event.
	transport.observer.SafeEmit("newproducer", producer)

	return
}

// Consume creates a Consumer.
func (transport *Transport) Consume(options ConsumerOptions) (consumer *Consumer, err error) {
	transport.logger.V(1).Info("consume()")

	producerId := options.ProducerId
	rtpCapabilities := options.RtpCapabilities
	paused := options.Paused
	preferredLayers := options.PreferredLayers
	appData := options.AppData

	producer := transport.getProducerById(producerId)

	if producer == nil {
		err = fmt.Errorf(`Producer with id "%s" not found`, producerId)
		return
	}

	rtpParameters, err := getConsumerRtpParameters(producer.ConsumableRtpParameters(), rtpCapabilities, options.Pipe)
	if err != nil {
		return
	}

	if !options.Pipe {
		if len(options.Mid) > 0 {
			rtpParameters.Mid = options.Mid
		} else {
			transport.locker.Lock()

			// Set MID.
			rtpParameters.Mid = fmt.Sprintf("%d", transport.nextMidForConsumers)
			transport.nextMidForConsumers++

			// We use up to 8 bytes for MID (string).
			if maxMid := uint32(100000000); transport.nextMidForConsumers == maxMid {
				transport.logger.Error(nil, "consume() | reaching max MID value", "mid", maxMid)
				transport.nextMidForConsumers = 0
			}

			transport.locker.Unlock()
		}
	}

	internal := transport.internal
	internal.ConsumerId = uuid.NewString()

	tp := producer.Type()

	if options.Pipe {
		tp = "pipe"
	}

	data := consumerData{
		ProducerId:    producerId,
		Kind:          producer.Kind(),
		RtpParameters: rtpParameters,
		Type:          ConsumerType(tp),
	}

	reqData := struct {
		consumerData
		ConsumerId             string                  `json:"consumerId"`
		ConsumableRtpEncodings []RtpEncodingParameters `json:"consumableRtpEncodings"`
		Paused                 bool                    `json:"paused"`
		PreferredLayers        *ConsumerLayers         `json:"preferredLayers,omitempty"`
		IgnoreDtx              bool                    `json:"ignoreDtx,omitempty"`
	}{
		consumerData:           data,
		ConsumerId:             internal.ConsumerId,
		ConsumableRtpEncodings: producer.ConsumableRtpParameters().Encodings,
		Paused:                 paused,
		PreferredLayers:        preferredLayers,
		IgnoreDtx:              options.IgnoreDtx,
	}

	resp := transport.channel.Request("transport.consume", internal, reqData)

	var status struct {
		Paused         bool
		ProducerPaused bool
		Score          *ConsumerScore
	}
	if err = resp.Unmarshal(&status); err != nil {
		return
	}

	consumer = newConsumer(consumerParams{
		internal:        internal,
		data:            data,
		channel:         transport.channel,
		payloadChannel:  transport.payloadChannel,
		appData:         appData,
		paused:          status.Paused,
		producerPaused:  status.ProducerPaused,
		score:           status.Score,
		preferredLayers: preferredLayers,
	})

	transport.consumers.Store(consumer.Id(), consumer)
	consumer.On("@close", func() {
		transport.consumers.Delete(consumer.Id())
	})
	consumer.On("@producerclose", func() {
		transport.consumers.Delete(consumer.Id())
	})

	// Emit observer event.
	transport.observer.SafeEmit("newconsumer", consumer)

	return
}

// ProduceData creates a DataProducer.
func (transport *Transport) ProduceData(options DataProducerOptions) (dataProducer *DataProducer, err error) {
	transport.logger.V(1).Info("produceData()")

	id := options.Id
	sctpStreamParameters := options.SctpStreamParameters
	label := options.Label
	protocol := options.Protocol
	appData := options.AppData

	if len(id) > 0 {
		if _, ok := transport.dataProducers.Load(id); ok {
			err = NewTypeError(`a DataProducer with same id "%s" already exists`, id)
			return
		}
	} else {
		id = uuid.NewString()
	}

	var typ DataProducerType

	if transport.data.transportType == TransportType_Direct {
		typ = DataProducerType_Direct

		if sctpStreamParameters != nil {
			transport.logger.Info(
				"produceData() | sctpStreamParameters are ignored when producing data on a DirectTransport", "warn", true)
		}
	} else {
		typ = DataProducerType_Sctp

		if err = validateSctpStreamParameters(sctpStreamParameters); err != nil {
			return
		}
	}

	internal := transport.internal
	internal.DataProducerId = id

	reqData := H{
		"dataProducerId":       id,
		"type":                 typ,
		"sctpStreamParameters": sctpStreamParameters,
		"label":                label,
		"protocol":             protocol,
	}
	resp := transport.channel.Request("transport.produceData", internal, reqData)

	var data dataProducerData
	if err = resp.Unmarshal(&data); err != nil {
		return
	}

	dataProducer = newDataProducer(dataProducerParams{
		internal:       internal,
		data:           data,
		channel:        transport.channel,
		payloadChannel: transport.payloadChannel,
		appData:        appData,
	})

	transport.dataProducers.Store(dataProducer.Id(), dataProducer)
	dataProducer.On("@close", func() {
		transport.dataProducers.Delete(dataProducer.Id())
		transport.Emit("@dataproducerclose", dataProducer)
	})

	transport.Emit("@newdataproducer", dataProducer)

	// Emit observer event.
	transport.observer.SafeEmit("newdataproducer", dataProducer)

	return
}

// ConsumeData creates a DataConsumer.
func (transport *Transport) ConsumeData(options DataConsumerOptions) (dataConsumer *DataConsumer, err error) {
	transport.logger.V(1).Info("consumeData()")

	dataProducerId := options.DataProducerId
	ordered := options.Ordered
	maxPacketLifeTime := options.MaxPacketLifeTime
	maxRetransmits := options.MaxRetransmits
	appData := options.AppData

	dataProducer := transport.getDataProducerById(dataProducerId)

	if dataProducer == nil {
		err = fmt.Errorf(`DataProducer with id "%s" not found`, dataProducerId)
		return
	}

	var typ DataProducerType
	var sctpStreamParameters SctpStreamParameters
	var sctpStreamId int = -1

	if transport.data.transportType == TransportType_Direct {
		typ = DataProducerType_Direct

		if ordered != nil || maxPacketLifeTime > 0 || maxRetransmits > 0 {
			transport.logger.Info(
				"consumeData() | ordered, maxPacketLifeTime and maxRetransmits are ignored when consuming data on a DirectTransport",
				"warn", true)
		}
	} else {
		typ = DataProducerType_Sctp

		sctpStreamParameters = dataProducer.SctpStreamParameters()
		// Override if given.
		if ordered != nil {
			sctpStreamParameters.Ordered = ordered
		}
		if maxPacketLifeTime > 0 {
			sctpStreamParameters.MaxPacketLifeTime = maxPacketLifeTime
		}
		if maxRetransmits > 0 {
			sctpStreamParameters.MaxRetransmits = maxRetransmits
		}

		transport.locker.Lock()

		if sctpStreamId, err = transport.getNextSctpStreamId(); err != nil {
			return
		}
		transport.sctpStreamIds[sctpStreamId] = 1
		sctpStreamParameters.StreamId = uint16(sctpStreamId)

		transport.locker.Unlock()
	}

	internal := transport.internal
	internal.DataConsumerId = uuid.NewString()
	internal.DataProducerId = dataProducerId

	reqData := H{
		"dataConsumerId":       internal.DataConsumerId,
		"dataProducerId":       dataProducerId,
		"type":                 typ,
		"sctpStreamParameters": sctpStreamParameters,
		"label":                dataProducer.Label(),
		"protocol":             dataProducer.Protocol(),
	}
	resp := transport.channel.Request("transport.consumeData", internal, reqData)

	var data dataConsumerData
	if err = resp.Unmarshal(&data); err != nil {
		return
	}

	dataConsumer = newDataConsumer(dataConsumerParams{
		internal:       internal,
		data:           data,
		channel:        transport.channel,
		payloadChannel: transport.payloadChannel,
		appData:        appData,
	})

	transport.dataConsumers.Store(dataConsumer.Id(), dataConsumer)
	dataConsumer.On("@close", func() {
		transport.dataConsumers.Delete(dataConsumer.Id())

		transport.locker.Lock()
		if sctpStreamId >= 0 {
			transport.sctpStreamIds[sctpStreamId] = 0
		}
		transport.locker.Unlock()
	})
	dataConsumer.On("@dataproducerclose", func() {
		transport.dataConsumers.Delete(dataConsumer.Id())

		transport.locker.Lock()
		if sctpStreamId >= 0 {
			transport.sctpStreamIds[sctpStreamId] = 0
		}
		transport.locker.Unlock()
	})

	// Emit observer event.
	transport.observer.SafeEmit("newdataconsumer", dataConsumer)

	return
}

// EnableTraceEvent enables 'trace' events.
func (transport *Transport) EnableTraceEvent(types ...TransportTraceEventType) error {
	transport.logger.V(1).Info("pause()")

	if types == nil {
		types = []TransportTraceEventType{}
	}

	resp := transport.channel.Request("transport.enableTraceEvent", transport.internal, H{"types": types})

	return resp.Err()
}

func (transport *Transport) getNextSctpStreamId() (sctpStreamId int, err error) {
	if transport.data.sctpParameters.MIS == 0 {
		err = NewTypeError("missing sctpParameters.MIS")
		return
	}

	numStreams := transport.data.sctpParameters.MIS

	if len(transport.sctpStreamIds) == 0 {
		transport.sctpStreamIds = make([]byte, numStreams)
	}

	for idx := 0; idx < len(transport.sctpStreamIds); idx++ {
		sctpStreamId = (transport.nextSctpStreamId + idx) % len(transport.sctpStreamIds)

		if transport.sctpStreamIds[sctpStreamId] == 0 {
			transport.nextSctpStreamId = sctpStreamId + 1
			return
		}
	}

	err = errors.New("no sctpStreamId available")

	return
}

// OnTrace set handler on "trace" event
func (transport *Transport) OnTrace(handler func(trace *TransportTraceEventData)) {
	transport.onTrace = handler
}

// OnClose set handler on "close" event
func (transport *Transport) OnClose(handler func()) {
	transport.onClose = handler
}

func (transport *Transport) handleEvent(event string, data []byte) {
	logger := transport.logger

	switch event {
	case "trace":
		var result *TransportTraceEventData

		if err := json.Unmarshal([]byte(data), &result); err != nil {
			logger.Error(err, "failed to unmarshal trace", "data", json.RawMessage(data))
			return
		}

		transport.SafeEmit("trace", result)

		// Emit observer event.
		transport.Observer().SafeEmit("trace", result)

		if handler := transport.onTrace; handler != nil {
			handler(result)
		}

	default:
		logger.Error(nil, "ignoring unknown event in channel listener", "event", event)
	}
}
