package mediasoup

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	uuid "github.com/satori/go.uuid"
)

type ITransport interface {
	IEventEmitter
	Id() string
	Closed() bool
	AppData() interface{}
	Observer() IEventEmitter
	Close()
	routerClosed()
	Dump() (*TransportDump, error)
	GetStats() ([]*TransportStat, error)
	Connect(TransportConnectOptions) error
	SetMaxIncomingBitrate(bitrate int) error
	Produce(ProducerOptions) (*Producer, error)
	Consume(ConsumerOptions) (*Consumer, error)
	ProduceData(DataProducerOptions) (*DataProducer, error)
	ConsumeData(DataConsumerOptions) (*DataConsumer, error)
	EnableTraceEvent(types ...TransportTraceEventType) error
}

type TransportListenIp struct {
	/**
	 * Listening IPv4 or IPv6.
	 */
	Ip string `json:"ip,omitempty"`

	/**
	 * Announced IPv4 or IPv6 (useful when running mediasoup behind NAT with
	 * private IP).
	 */
	AnnouncedIp string `json:"announcedIp,omitempty"`
}

/**
 * Transport protocol.
 */
type TransportProtocol string

const (
	TransportProtocol_Udp TransportProtocol = "udp"
	TransportProtocol_Tcp                   = "tcp"
)

type TransportTraceEventType string

const (
	TransportTraceEventType_Probation TransportTraceEventType = "probation"
	TransportTraceEventType_Bwe                               = "bwe"
)

type TransportTuple struct {
	LocalIp    string `json:"localIp,omitempty"`
	LocalPort  uint16 `json:"localPort,omitempty"`
	RemoteIp   string `json:"remoteIp,omitempty"`
	RemotePort uint16 `json:"remotePort,omitempty"`
	Protocol   string `json:"protocol,omitempty"`
}

type TransportTraceEventData struct {
	/**
	 * Trace type.
	 */
	Type TransportTraceEventType `json:"type,omitempty"`
	/**
	 * Event timestamp.
	 */
	Timestamp int64 `json:"timestamp,omitempty"`
	/**
	 * Event direction.
	 */
	Direction string `json:"direction,omitempty"`
	/**
	 * Per type information.
	 */
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
	Type                     string    `json:"type"`
	TransportId              string    `json:"transportId"`
	Timestamp                int64     `json:"timestamp"`
	SctpState                SctpState `json:"sctpState,omitempty"`
	BytesReceived            int64     `json:"bytesReceived"`
	RecvBitrate              int64     `json:"recvBitrate"`
	BytesSent                int64     `json:"bytesSent"`
	SendBitrate              int64     `json:"sendBitrate"`
	RtpBytesReceived         int64     `json:"rtpBytesReceived"`
	RtpRecvBitrate           int64     `json:"rtpRecvBitrate"`
	RtpBytesSent             int64     `json:"rtpBytesSent"`
	RtpSendBitrate           int64     `json:"rtpSendBitrate"`
	RtxBytesReceived         int64     `json:"rtxBytesReceived"`
	RtxRecvBitrate           int64     `json:"rtxRecvBitrate"`
	RtxBytesSent             int64     `json:"rtxBytesSent"`
	RtxSendBitrate           int64     `json:"rtxSendBitrate"`
	ProbationBytesSent       int64     `json:"probationBytesSent"`
	ProbationSendBitrate     int64     `json:"probationSendBitrate"`
	AvailableOutgoingBitrate int64     `json:"availableOutgoingBitrate,omitempty"`
	AvailableIncomingBitrate int64     `json:"availableIncomingBitrate,omitempty"`
	MaxIncomingBitrate       int64     `json:"maxIncomingBitrate,omitempty"`

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
	TransportType_Plain                = "PlainTransport"
	TransportType_Pipe                 = "PipeTransport"
	TransportType_Webrtc               = "WebrtcTransport"
)

type transportData struct {
	sctpParameters SctpParameters
	sctpState      SctpState
	transportType  TransportType
}

type transportParams struct {
	// {
	// 	routerId: string;
	// 	transportId: string;
	// };
	internal                 internalData
	data                     interface{}
	channel                  *Channel
	payloadChannel           *PayloadChannel
	appData                  interface{}
	getRouterRtpCapabilities func() RtpCapabilities
	getProducerById          func(string) *Producer
	getDataProducerById      func(string) *DataProducer
	logger                   Logger
}

/**
 * Transport
 * @emits routerclose
 * @emits @close
 * @emits @newproducer - (producer: Producer)
 * @emits @producerclose - (producer: Producer)
 * @emits @newdataproducer - (dataProducer: DataProducer)
 * @emits @dataproducerclose - (dataProducer: DataProducer)
 */
type Transport struct {
	IEventEmitter
	logger Logger
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
	// Observer instance.
	observer IEventEmitter
	// locker instance
	locker sync.Mutex
}

func newTransport(params transportParams) ITransport {
	params.logger.Debug("constructor()")

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

// Transport id
func (transport *Transport) Id() string {
	return transport.internal.TransportId
}

// Whether the Transport is closed.
func (transport *Transport) Closed() bool {
	return atomic.LoadUint32(&transport.closed) > 0
}

//App custom data.
func (transport *Transport) AppData() interface{} {
	return transport.appData
}

/**
 * Observer.
 *
 * @emits close
 * @emits newproducer - (producer: Producer)
 * @emits newconsumer - (producer: Producer)
 * @emits newdataproducer - (dataProducer: DataProducer)
 * @emits newdataconsumer - (dataProducer: DataProducer)
 */
func (transport *Transport) Observer() IEventEmitter {
	return transport.observer
}

// Close the Transport.
func (transport *Transport) Close() {
	if atomic.CompareAndSwapUint32(&transport.closed, 0, 1) {
		transport.logger.Debug("close()")

		// Remove notification subscriptions.
		transport.channel.RemoveAllListeners(transport.Id())
		transport.payloadChannel.RemoveAllListeners(transport.Id())

		transport.channel.Request("transport.close", transport.internal)

		transport.producers.Range(func(key, value interface{}) bool {
			producer := value.(*Producer)

			producer.transportClosed()
			transport.Emit("@producerclose", producer)

			return true
		})
		transport.producers = sync.Map{}

		transport.consumers.Range(func(key, value interface{}) bool {
			value.(*Consumer).transportClosed()

			return true
		})
		transport.consumers = sync.Map{}

		transport.dataProducers.Range(func(key, value interface{}) bool {
			producer := value.(*DataProducer)

			producer.transportClosed()
			transport.Emit("@dataproducerclose", producer)

			return true
		})
		transport.dataProducers = sync.Map{}

		transport.dataConsumers.Range(func(key, value interface{}) bool {
			value.(*DataConsumer).transportClosed()

			return true
		})
		transport.dataConsumers = sync.Map{}

		transport.Emit("@close")
		transport.RemoveAllListeners()

		// Emit observer event.
		transport.observer.SafeEmit("close")
		transport.observer.RemoveAllListeners()
	}
	return
}

/**
 * Router was closed.
 *
 * @virtual
 */
func (transport *Transport) routerClosed() {
	if atomic.CompareAndSwapUint32(&transport.closed, 0, 1) {
		transport.logger.Debug("routerClosed()")

		// Remove notification subscriptions.
		transport.channel.RemoveAllListeners(transport.Id())
		transport.payloadChannel.RemoveAllListeners(transport.Id())

		transport.producers.Range(func(key, value interface{}) bool {
			producer := value.(*Producer)

			producer.transportClosed()
			transport.Emit("@producerclose", producer)

			return true
		})
		transport.producers = sync.Map{}

		transport.consumers.Range(func(key, value interface{}) bool {
			value.(*Consumer).transportClosed()

			return true
		})
		transport.consumers = sync.Map{}

		transport.dataProducers.Range(func(key, value interface{}) bool {
			producer := value.(*DataProducer)

			producer.transportClosed()
			transport.Emit("@dataproducerclose", producer)

			return true
		})
		transport.dataProducers = sync.Map{}

		transport.dataConsumers.Range(func(key, value interface{}) bool {
			value.(*DataConsumer).transportClosed()

			return true
		})
		transport.dataConsumers = sync.Map{}

		transport.SafeEmit("routerclose")
		transport.RemoveAllListeners()

		// Emit observer event.
		transport.observer.SafeEmit("close")
		transport.observer.RemoveAllListeners()
	}
}

// Dump Transport.
func (transport *Transport) Dump() (data *TransportDump, err error) {
	transport.logger.Debug("dump()")

	resp := transport.channel.Request("transport.dump", transport.internal)
	err = resp.Unmarshal(&data)

	return
}

// Get Transport stats.
func (transport *Transport) GetStats() (stat []*TransportStat, err error) {
	transport.logger.Debug("getStats()")

	resp := transport.channel.Request("transport.getStats", transport.internal)
	err = resp.Unmarshal(&stat)

	return
}

/**
 * Provide the Transport remote parameters.
 */
func (transport *Transport) Connect(TransportConnectOptions) error {
	return errors.New("method not implemented in the subclass")
}

/**
 * Set maximum incoming bitrate for receiving media.
 */
func (transport *Transport) SetMaxIncomingBitrate(bitrate int) error {
	transport.logger.Debug("SetMaxIncomingBitrate() [bitrate:%d]", bitrate)

	resp := transport.channel.Request(
		"transport.setMaxIncomingBitrate", transport.internal, H{"bitrate": bitrate})

	return resp.Err()
}

/**
 * Create a Producer.
 */
func (transport *Transport) Produce(options ProducerOptions) (producer *Producer, err error) {
	transport.logger.Debug("produce()")

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
		id = uuid.NewV4().String()
	}

	// This may throw.
	if err = validateRtpParameters(&rtpParameters); err != nil {
		return
	}

	// If missing or empty encodings, add one.
	if len(rtpParameters.Encodings) == 0 {
		rtpParameters.Encodings = []RtpEncodingParameters{{}}
	}

	// Don"t do this in PipeTransports since there we must keep CNAME value in each Producer.
	if transport.data.transportType != TransportType_Pipe {
		// If CNAME is given and we don"t have yet a CNAME for Producers in this
		// Transport, take it.
		if len(transport.cnameForProducers) == 0 && len(rtpParameters.Rtcp.Cname) > 0 {
			transport.cnameForProducers = rtpParameters.Rtcp.Cname
		} else if len(transport.cnameForProducers) == 0 {
			// Otherwise if we don"t have yet a CNAME for Producers and the RTP parameters
			// do not include CNAME, create a random one.
			transport.cnameForProducers = uuid.NewV4().String()[:8]
		}

		// Override Producer"s CNAME.
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

/**
 * Create a Consumer.
 */
func (transport *Transport) Consume(options ConsumerOptions) (consumer *Consumer, err error) {
	transport.logger.Debug("consume()")

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
		transport.locker.Lock()

		// Set MID.
		rtpParameters.Mid = fmt.Sprintf("%d", transport.nextMidForConsumers)

		transport.nextMidForConsumers++

		// We use up to 8 bytes for MID (string).
		if maxMid := uint32(100000000); transport.nextMidForConsumers == maxMid {
			transport.logger.Error(`consume() | reaching max MID value "%d"`, maxMid)

			transport.nextMidForConsumers = 0
		}

		transport.locker.Unlock()
	}

	internal := transport.internal
	internal.ConsumerId = uuid.NewV4().String()
	internal.ProducerId = producerId

	typ := producer.Type()

	if options.Pipe {
		typ = "pipe"
	}

	reqData := H{
		"kind":                   producer.Kind(),
		"rtpParameters":          rtpParameters,
		"type":                   typ,
		"consumableRtpEncodings": producer.ConsumableRtpParameters().Encodings,
		"paused":                 paused,
		"preferredLayers":        preferredLayers,
	}
	resp := transport.channel.Request("transport.consume", internal, reqData)

	var status struct {
		Paused         bool
		ProducerPaused bool
		Score          ConsumerScore
	}
	if err = resp.Unmarshal(&status); err != nil {
		return
	}

	consumerData := consumerData{
		Kind:          producer.Kind(),
		RtpParameters: rtpParameters,
		Type:          typ,
	}
	consumer = newConsumer(consumerParams{
		internal:        internal,
		data:            consumerData,
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

/**
 * Create a DataProducer.
 */
func (transport *Transport) ProduceData(options DataProducerOptions) (dataProducer *DataProducer, err error) {
	transport.logger.Debug("produceData()")

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
		id = uuid.NewV4().String()
	}

	var typ DataProducerType

	if transport.data.transportType == TransportType_Direct {
		typ = DataProducerType_Direct

		if sctpStreamParameters != nil {
			transport.logger.Warn(
				"produceData() | sctpStreamParameters are ignored when producing data on a DirectTransport")
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
		"type":     typ,
		"label":    label,
		"protocol": protocol,
	}
	if sctpStreamParameters != nil {
		reqData["sctpStreamParameters"] = sctpStreamParameters
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

/**
 * Create a DataConsumer.
 */
func (transport *Transport) ConsumeData(options DataConsumerOptions) (dataConsumer *DataConsumer, err error) {
	transport.logger.Debug("consumeData()")

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
			transport.logger.Warn(
				"consumeData() | ordered, maxPacketLifeTime and maxRetransmits are ignored when consuming data on a DirectTransport")
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
	internal.DataConsumerId = uuid.NewV4().String()
	internal.DataProducerId = dataProducerId

	reqData := H{
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

/**
 * Enable 'trace' event.
 */
func (transport *Transport) EnableTraceEvent(types ...TransportTraceEventType) error {
	transport.logger.Debug("pause()")

	if types == nil {
		types = []TransportTraceEventType{}
	}

	resp := transport.channel.Request("transport.enableTraceEvent", transport.internal, H{"types": types})

	return resp.Err()
}

func (transport *Transport) getNextSctpStreamId() (sctpStreamId int, err error) {
	if transport.data.sctpParameters.MIS == 0 {
		err = NewTypeError("missing data.sctpParameters.MIS")
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
