package mediasoup

import (
	"errors"
	"sync"
	"sync/atomic"

	uuid "github.com/satori/go.uuid"
)

type RouterOptions struct {
	/**
	 * Router media codecs.
	 */
	MediaCodecs []*RtpCodecCapability `json:"mediaCodecs,omitempty"`

	/**
	 * Custom application data.
	 */
	AppData interface{} `json:"appData,omitempty"`
}

type PipeToRouterOptions struct {
	/**
	 * The id of the Producer to consume.
	 */
	ProducerId string `json:"producerId,omitempty"`

	/**
	 * The id of the DataProducer to consume.
	 */
	DataProducerId string `json:"dataProducerId,omitempty"`

	/**
	 * Target Router instance.
	 */
	Router *Router `json:"router,omitempty"`

	/**
	 * IP used in the PipeTransport pair. Default '127.0.0.1'.
	 */
	ListenIp TransportListenIp `json:"listenIp,omitempty"`

	/**
	 * Create a SCTP association. Default false.
	 */
	EnableSctp bool `json:"enableSctp,omitempty"`

	/**
	 * SCTP streams number.
	 */
	NumSctpStreams NumSctpStreams `json:"numSctpStreams,omitempty"`

	/**
	 * Enable RTX and NACK for RTP retransmission.
	 */
	EnableRtx bool `json:"enableRtx,omitempty"`

	/**
	 * Enable SRTP.
	 */
	EnableSrtp bool `json:"enableSrtp,omitempty"`
}

type PipeToRouterResult struct {
	/**
	 * The Consumer created in the current Router.
	 */
	PipeConsumer *Consumer

	/**
	 * The Producer created in the target Router.
	 */
	PipeProducer *Producer

	/**
	 * The DataConsumer created in the current Router.
	 */
	PipeDataConsumer *DataConsumer

	/**
	 * The DataProducer created in the target Router.
	 */
	PipeDataProducer *DataProducer
}

type routerData struct {
	RtpCapabilities RtpCapabilities `json:"rtpCapabilities,omitempty"`
}

type routerParams struct {
	// {
	// 	routerId: string;
	// };
	internal       internalData
	data           routerData
	channel        *Channel
	payloadChannel *PayloadChannel
	appData        interface{}
}

/**
 * Router
 * @emits workerclose
 * @emits @close
 */
type Router struct {
	IEventEmitter
	logger                  Logger
	internal                internalData
	data                    routerData
	channel                 *Channel
	payloadChannel          *PayloadChannel
	closed                  uint32
	appData                 interface{}
	transports              sync.Map
	producers               sync.Map
	rtpObservers            sync.Map
	dataProducers           sync.Map
	mapRouterPipeTransports sync.Map
	observer                IEventEmitter
	locker                  sync.Mutex
}

func newRouter(params routerParams) *Router {
	logger := NewLogger("Router")
	logger.Debug("constructor()")

	return &Router{
		IEventEmitter:  NewEventEmitter(),
		logger:         logger,
		internal:       params.internal,
		data:           params.data,
		channel:        params.channel,
		payloadChannel: params.payloadChannel,
		appData:        params.appData,
		observer:       NewEventEmitter(),
	}
}

// Router id
func (router *Router) Id() string {
	return router.internal.RouterId
}

// Whether the Router is closed.
func (router *Router) Closed() bool {
	return atomic.LoadUint32(&router.closed) > 0
}

// RTC capabilities of the Router.
func (router *Router) RtpCapabilities() RtpCapabilities {
	return router.data.RtpCapabilities
}

func (router *Router) Observer() IEventEmitter {
	return router.observer
}

// Close the Router.
func (router *Router) Close() {
	if atomic.CompareAndSwapUint32(&router.closed, 0, 1) {
		router.logger.Debug("close()")

		router.channel.Request("router.close", router.internal)

		// Close every Transport.
		router.transports.Range(func(key, value interface{}) bool {
			value.(ITransport).routerClosed()
			return true
		})
		router.transports = sync.Map{}

		// Clear the Producers map.
		router.producers = sync.Map{}

		// Close every RtpObserver.
		router.rtpObservers.Range(func(key, value interface{}) bool {
			value.(IRtpObserver).routerClosed()
			return true
		})
		router.rtpObservers = sync.Map{}

		// Clear map of Router/PipeTransports.
		router.mapRouterPipeTransports = sync.Map{}

		router.Emit("@close")
		router.RemoveAllListeners()

		// Emit observer event.
		router.observer.SafeEmit("close")
		router.observer.RemoveAllListeners()
	}

	return
}

func (router *Router) workerClosed() {
	if atomic.CompareAndSwapUint32(&router.closed, 0, 1) {
		router.logger.Debug("workerClosed()")

		// Close every Transport.
		router.transports.Range(func(key, value interface{}) bool {
			value.(ITransport).routerClosed()
			return true
		})
		router.transports = sync.Map{}

		// Clear the Producers map.
		router.producers = sync.Map{}

		// Close every RtpObserver.
		router.rtpObservers.Range(func(key, value interface{}) bool {
			value.(IRtpObserver).routerClosed()
			return true
		})
		router.rtpObservers = sync.Map{}

		// Clear map of Router/PipeTransports.
		router.mapRouterPipeTransports = sync.Map{}

		router.Emit("workerclose")
		router.RemoveAllListeners()

		// Emit observer event.
		router.observer.SafeEmit("close")
		router.observer.RemoveAllListeners()
	}

	return
}

// Dump Router.
func (router *Router) Dump() (data *RouterDump, err error) {
	router.logger.Debug("dump()")

	resp := router.channel.Request("router.dump", router.internal)
	err = resp.Unmarshal(&data)

	return
}

/**
 * Create a WebRtcTransport.
 */
func (router *Router) CreateWebRtcTransport(option WebRtcTransportOptions) (transport *WebRtcTransport, err error) {
	options := &WebRtcTransportOptions{
		EnableUdp:                       Bool(true),
		InitialAvailableOutgoingBitrate: 600000,
		NumSctpStreams:                  NumSctpStreams{OS: 1024, MIS: 1024},
		MaxSctpMessageSize:              262144,
		SctpSendBufferSize:              262144,
	}
	if err = override(options, option); err != nil {
		return
	}

	router.logger.Debug("createWebRtcTransport()")

	internal := router.internal
	internal.TransportId = uuid.NewV4().String()
	reqData := H{
		"listenIps":                       options.ListenIps,
		"enableUdp":                       options.EnableUdp,
		"enableTcp":                       options.EnableTcp,
		"preferUdp":                       options.PreferUdp,
		"preferTcp":                       options.PreferTcp,
		"initialAvailableOutgoingBitrate": options.InitialAvailableOutgoingBitrate,
		"enableSctp":                      options.EnableSctp,
		"numSctpStreams":                  options.NumSctpStreams,
		"maxSctpMessageSize":              options.MaxSctpMessageSize,
		"sctpSendBufferSize":              options.SctpSendBufferSize,
		"isDataChannel":                   true,
	}

	resp := router.channel.Request("router.createWebRtcTransport", internal, reqData)

	var data webrtcTransportData
	if err = resp.Unmarshal(&data); err != nil {
		return
	}

	iTransport := router.createTransport(internal, data, options.AppData)

	return iTransport.(*WebRtcTransport), nil
}

/**
 * Create a PlainTransport.
 */
func (router *Router) CreatePlainTransport(option PlainTransportOptions) (transport *PlainTransport, err error) {
	options := &PlainTransportOptions{
		RtcpMux:            Bool(true),
		NumSctpStreams:     NumSctpStreams{OS: 1024, MIS: 1024},
		MaxSctpMessageSize: 262144,
		SctpSendBufferSize: 262144,
		SrtpCryptoSuite:    AES_CM_128_HMAC_SHA1_80,
	}
	if err = override(options, option); err != nil {
		return
	}

	router.logger.Debug("createPlainTransport()")

	internal := router.internal
	internal.TransportId = uuid.NewV4().String()
	reqData := H{
		"listenIp":           options.ListenIp,
		"rtcpMux":            options.RtcpMux,
		"comedia":            options.Comedia,
		"enableSctp":         options.EnableSctp,
		"numSctpStreams":     options.NumSctpStreams,
		"maxSctpMessageSize": options.MaxSctpMessageSize,
		"sctpSendBufferSize": options.SctpSendBufferSize,
		"isDataChannel":      false,
		"enableSrtp":         options.EnableSrtp,
		"srtpCryptoSuite":    options.SrtpCryptoSuite,
	}

	resp := router.channel.Request("router.createPlainTransport", internal, reqData)

	var data plainTransportData
	if err = resp.Unmarshal(&data); err != nil {
		return
	}

	iTransport := router.createTransport(internal, data, options.AppData)

	return iTransport.(*PlainTransport), nil
}

/**
 * Create a PipeTransport.
 */
func (router *Router) CreatePipeTransport(option PipeTransportOptions) (transport *PipeTransport, err error) {
	options := &PipeTransportOptions{
		NumSctpStreams:     NumSctpStreams{OS: 1024, MIS: 1024},
		MaxSctpMessageSize: 268435456,
		SctpSendBufferSize: 268435456,
	}
	if err = override(options, option); err != nil {
		return
	}

	router.logger.Debug("createPipeTransport()")

	internal := router.internal
	internal.TransportId = uuid.NewV4().String()
	reqData := H{
		"listenIp":           options.ListenIp,
		"enableSctp":         options.EnableSctp,
		"numSctpStreams":     options.NumSctpStreams,
		"maxSctpMessageSize": options.MaxSctpMessageSize,
		"sctpSendBufferSize": options.SctpSendBufferSize,
		"isDataChannel":      false,
		"enableRtx":          options.EnableRtx,
		"enableSrtp":         options.EnableSrtp,
	}

	resp := router.channel.Request("router.createPipeTransport", internal, reqData)

	var data pipeTransortData
	if err = resp.Unmarshal(&data); err != nil {
		return
	}

	iTransport := router.createTransport(internal, data, options.AppData)

	return iTransport.(*PipeTransport), nil
}

/**
 * Create a DirectTransport.
 */
func (router *Router) CreateDirectTransport(params ...DirectTransportOptions) (transport *DirectTransport, err error) {
	options := &DirectTransportOptions{
		MaxMessageSize: 262144,
	}
	for _, option := range params {
		if err = override(options, option); err != nil {
			return
		}
	}

	router.logger.Debug("createDirectTransport()")

	internal := router.internal
	internal.TransportId = uuid.NewV4().String()
	reqData := H{"direct": true, "maxMessageSize": options.MaxMessageSize}

	resp := router.channel.Request("router.createDirectTransport", internal, reqData)

	var data directTransportData
	if err = resp.Unmarshal(&data); err != nil {
		return
	}

	iTransport := router.createTransport(internal, data, options.AppData)

	return iTransport.(*DirectTransport), nil
}

/**
 * Pipes the given Producer or DataProducer into another Router in same host.
 */
func (router *Router) PipeToRouter(option PipeToRouterOptions) (result *PipeToRouterResult, err error) {
	options := &PipeToRouterOptions{
		ListenIp: TransportListenIp{
			Ip: "127.0.0.1",
		},
		EnableSctp:     true,
		NumSctpStreams: NumSctpStreams{OS: 1024, MIS: 1024},
	}
	if err = override(options, option); err != nil {
		return
	}

	if len(options.ProducerId) == 0 && len(options.DataProducerId) == 0 {
		err = NewTypeError("missing producerId")
		return
	}
	if len(options.ProducerId) > 0 && len(options.DataProducerId) > 0 {
		err = NewTypeError("just producerId or dataProducerId can be given")
		return
	}
	if options.Router == nil {
		err = NewTypeError("Router not found")
		return
	}
	if options.Router == router {
		err = NewTypeError("cannot use this Router as destination'")
		return
	}

	router.logger.Debug("pipeToRouter()")

	var producer *Producer
	var dataProducer *DataProducer

	if len(options.ProducerId) > 0 {
		if value, ok := router.producers.Load(options.ProducerId); ok {
			producer = value.(*Producer)
		} else {
			err = NewTypeError("Producer not found")
			return
		}
	}
	if len(options.DataProducerId) > 0 {
		if value, ok := router.dataProducers.Load(options.DataProducerId); ok {
			dataProducer = value.(*DataProducer)
		} else {
			err = NewTypeError("DataProducer not found")
			return
		}
	}

	// Here we may have to create a new PipeTransport pair to connect source and
	// destination Routers. We just want to keep a PipeTransport pair for each
	// pair of Routers. Since this operation is async, it may happen that two
	// simultaneous calls to router1.pipeToRouter({ producerId: xxx, router: router2 })
	// would end up generating two pairs of PipeTranports. To prevent that, let's
	// use a locker.
	router.locker.Lock()
	defer router.locker.Unlock()

	var localPipeTransport, remotePipeTransport *PipeTransport

	value, ok := router.mapRouterPipeTransports.Load(options.Router)

	if ok {
		pipeTransportPair := value.([]*PipeTransport)
		localPipeTransport = pipeTransportPair[0]
		remotePipeTransport = pipeTransportPair[1]
	} else {
		defer func() {
			if err != nil {
				router.logger.Error("pipeToRouter() | error creating PipeTransport pair:%s", err)

				if localPipeTransport != nil {
					localPipeTransport.Close()
				}
				if remotePipeTransport != nil {
					remotePipeTransport.Close()
				}
			}
		}()

		option := PipeTransportOptions{
			ListenIp:       options.ListenIp,
			EnableSctp:     options.EnableSctp,
			NumSctpStreams: options.NumSctpStreams,
			EnableRtx:      options.EnableRtx,
			EnableSrtp:     options.EnableSrtp,
		}
		localPipeTransport, err = router.CreatePipeTransport(option)
		if err != nil {
			return
		}
		remotePipeTransport, err = options.Router.CreatePipeTransport(option)
		if err != nil {
			return
		}

		err = localPipeTransport.Connect(TransportConnectOptions{
			Ip:             remotePipeTransport.Tuple().LocalIp,
			Port:           remotePipeTransport.Tuple().LocalPort,
			SrtpParameters: remotePipeTransport.SrtpParameters(),
		})
		if err != nil {
			return
		}
		err = remotePipeTransport.Connect(TransportConnectOptions{
			Ip:             localPipeTransport.Tuple().LocalIp,
			Port:           localPipeTransport.Tuple().LocalPort,
			SrtpParameters: localPipeTransport.SrtpParameters(),
		})
		if err != nil {
			return
		}

		localPipeTransport.Observer().On("close", func() {
			remotePipeTransport.Close()
			router.mapRouterPipeTransports.Delete(options.Router)
		})

		remotePipeTransport.Observer().On("close", func() {
			localPipeTransport.Close()
			router.mapRouterPipeTransports.Delete(options.Router)
		})

		router.mapRouterPipeTransports.Store(options.Router, []*PipeTransport{localPipeTransport, remotePipeTransport})
	}

	if producer != nil {
		var pipeConsumer *Consumer
		var pipeProducer *Producer

		defer func() {
			if err != nil {
				router.logger.Error("pipeToRouter() | error creating pipe Consumer/Producer pair:%s", err)

				if pipeConsumer != nil {
					pipeConsumer.Close()
				}
				if pipeProducer != nil {
					pipeProducer.Close()
				}
			}
		}()

		pipeConsumer, err = localPipeTransport.Consume(ConsumerOptions{
			ProducerId: options.ProducerId,
		})
		if err != nil {
			return
		}

		pipeProducer, err = remotePipeTransport.Produce(ProducerOptions{
			Id:            producer.Id(),
			Kind:          pipeConsumer.Kind(),
			RtpParameters: pipeConsumer.RtpParameters(),
			Paused:        pipeConsumer.ProducerPaused(),
			AppData:       producer.AppData(),
		})
		if err != nil {
			return
		}

		// Ensure that the producer has not been closed in the meanwhile.
		if producer.Closed() {
			err = NewInvalidStateError("original Producer closed")
			return
		}

		// Ensure that producer.paused has not changed in the meanwhile and, if
		// so, sych the pipeProducer.
		if pipeProducer.Paused() != producer.Paused() {
			if producer.Paused() {
				err = pipeProducer.Pause()
			} else {
				err = pipeProducer.Resume()
			}
			if err != nil {
				return
			}
		}

		// Pipe events from the pipe Consumer to the pipe Producer.
		pipeConsumer.Observer().On("close", func() { pipeProducer.Close() })
		pipeConsumer.Observer().On("pause", func() { pipeProducer.Pause() })
		pipeConsumer.Observer().On("resume", func() { pipeProducer.Resume() })

		// Pipe events from the pipe Producer to the pipe Consumer.
		pipeProducer.Observer().On("close", func() { pipeConsumer.Close() })

		result = &PipeToRouterResult{
			PipeConsumer: pipeConsumer,
			PipeProducer: pipeProducer,
		}

		return
	}

	if dataProducer != nil {
		var pipeDataConsumer *DataConsumer
		var pipeDataProducer *DataProducer

		defer func() {
			if err != nil {
				router.logger.Error("pipeToRouter() | error creating pipe DataConsumer/DataProducer pair:%s", err)

				if pipeDataConsumer != nil {
					pipeDataConsumer.Close()
				}
				if pipeDataProducer != nil {
					pipeDataProducer.Close()
				}
			}
		}()

		pipeDataConsumer, err = localPipeTransport.ConsumeData(DataConsumerOptions{
			DataProducerId: options.DataProducerId,
		})
		if err != nil {
			return
		}

		pipeDataProducer, err = remotePipeTransport.ProduceData(DataProducerOptions{
			Id:                   dataProducer.Id(),
			SctpStreamParameters: pipeDataConsumer.SctpStreamParameters(),
			Label:                pipeDataConsumer.Label(),
			Protocol:             pipeDataConsumer.Protocol(),
			AppData:              dataProducer.AppData(),
		})
		if err != nil {
			return
		}

		// Ensure that the dataProducer has not been closed in the meanwhile.
		if dataProducer.Closed() {
			err = NewInvalidStateError("original DataProducer closed")
			return
		}

		// Pipe events from the pipe DataConsumer to the pipe DataProducer.
		pipeDataConsumer.Observer().On("close", func() { pipeDataProducer.Close() })

		// Pipe events from the pipe DataProducer to the pipe DataConsumer.
		pipeDataProducer.Observer().On("close", func() { pipeDataConsumer.Close() })

		result = &PipeToRouterResult{
			PipeDataConsumer: pipeDataConsumer,
			PipeDataProducer: pipeDataProducer,
		}

		return
	}

	err = errors.New("internal error")
	return
}

/**
 * Create an AudioLevelObserver.
 */
func (router *Router) CreateAudioLevelObserver(options ...func(o *AudioLevelObserverOptions)) (rtpObserver IRtpObserver, err error) {
	router.logger.Debug("createAudioLevelObserver()")

	defaultOptions := NewAudioLevelObserverOptions()

	for _, option := range options {
		option(&defaultOptions)
	}

	internal := router.internal
	internal.RtpObserverId = uuid.NewV4().String()

	resp := router.channel.Request("router.createAudioLevelObserver", internal, defaultOptions)

	if err = resp.Err(); err != nil {
		return
	}

	rtpObserver = newAudioLevelObserver(rtpObserverParams{
		internal:       internal,
		channel:        router.channel,
		payloadChannel: router.payloadChannel,
		appData:        router.appData,
		getProducerById: func(producerId string) *Producer {
			if value, ok := router.producers.Load(producerId); ok {
				return value.(*Producer)
			}
			return nil
		},
	})

	router.rtpObservers.Store(rtpObserver.Id(), rtpObserver)
	rtpObserver.On("@close", func() {
		router.rtpObservers.Delete(rtpObserver.Id())
	})

	return
}

/**
 * Check whether the given RTP capabilities can consume the given Producer.
 */
func (router *Router) CanConsume(producerId string, rtpCapabilities RtpCapabilities) bool {
	router.logger.Debug("CanConsume()")

	value, ok := router.producers.Load(producerId)
	if !ok {
		router.logger.Error(`canConsume() | Producer with id "%s" not found`, producerId)

		return false
	}

	producer := value.(*Producer)
	ok, err := canConsume(producer.ConsumableRtpParameters(), rtpCapabilities)

	if err != nil {
		router.logger.Error("canConsume() | unexpected error: %s", err)
	}

	return ok
}

/**
 * Create a Transport interface.
 */
func (router *Router) createTransport(internal internalData, data, appData interface{}) (transport ITransport) {
	if appData == nil {
		appData = H{}
	}

	var newTransport func(transportParams) ITransport

	switch data.(type) {
	case directTransportData:
		newTransport = newDirectTransport

	case plainTransportData:
		newTransport = newPlainTransport

	case pipeTransortData:
		newTransport = newPipeTransport

	case webrtcTransportData:
		newTransport = newWebRtcTransport
	}

	transport = newTransport(transportParams{
		internal:       internal,
		channel:        router.channel,
		payloadChannel: router.payloadChannel,
		data:           data,
		appData:        appData,
		getRouterRtpCapabilities: func() RtpCapabilities {
			return router.data.RtpCapabilities
		},
		getProducerById: func(producerId string) *Producer {
			if producer, ok := router.producers.Load(producerId); ok {
				return producer.(*Producer)
			}
			return nil
		},
		getDataProducerById: func(dataProducerId string) *DataProducer {
			if dataProducer, ok := router.dataProducers.Load(dataProducerId); ok {
				return dataProducer.(*DataProducer)
			}
			return nil
		},
	})

	router.transports.Store(transport.Id(), transport)
	transport.On("@close", func() {
		router.transports.Delete(transport.Id())
	})
	transport.On("@newproducer", func(producer *Producer) {
		router.producers.Store(producer.Id(), producer)
	})
	transport.On("@producerclose", func(producer *Producer) {
		router.producers.Delete(producer.Id())
	})
	transport.On("@newdataproducer", func(dataProducer *DataProducer) {
		router.dataProducers.Store(dataProducer.Id(), dataProducer)
	})
	transport.On("@dataproducerclose", func(dataProducer *DataProducer) {
		router.dataProducers.Delete(dataProducer.Id())
	})

	// Emit observer event.
	router.observer.SafeEmit("newtransport", transport)

	return
}
