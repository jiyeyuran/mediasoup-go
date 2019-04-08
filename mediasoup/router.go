package mediasoup

import (
	"errors"

	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
)

type Router struct {
	EventEmitter
	logger                  logrus.FieldLogger
	internal                Internal
	data                    RouterData
	channel                 *Channel
	transports              map[string]Transport
	producers               map[string]*Producer
	rtpObservers            map[string]RtpObserver
	mapRouterPipeTransports map[*Router][]Transport
	observer                EventEmitter
	closed                  bool
}

func NewRouter(internal Internal, data RouterData, channel *Channel) *Router {
	logger := TypeLogger("Router")

	logger.Debug("constructor()")

	return &Router{
		EventEmitter:            NewEventEmitter(AppLogger()),
		logger:                  logger,
		internal:                internal,
		data:                    data,
		channel:                 channel,
		transports:              make(map[string]Transport),
		producers:               make(map[string]*Producer),
		rtpObservers:            make(map[string]RtpObserver),
		mapRouterPipeTransports: make(map[*Router][]Transport),
		observer:                NewEventEmitter(AppLogger()),
	}
}

// Router id
func (router *Router) Id() string {
	return router.internal.RouterId
}

// Whether the Router is closed.
func (router *Router) Closed() bool {
	return router.closed
}

// RTC capabilities of the Router.
func (router *Router) RtpCapabilities() RtpCapabilities {
	return router.data.RtpCapabilities
}

func (router *Router) Observer() EventEmitter {
	return router.observer
}

// Close the Router.
func (router *Router) Close() (err error) {
	if router.closed {
		return
	}

	router.logger.Debug("close()")

	router.closed = true

	resp := router.channel.Request("router.close", router.internal)

	if err = resp.Err(); err != nil {
		return
	}

	// Close every Transport.
	for _, transport := range router.transports {
		transport.routerClosed()
	}
	router.transports = make(map[string]Transport)

	// Clear the Producers map.
	router.producers = make(map[string]*Producer)

	// Close every RtpObserver.
	for _, rtpObserver := range router.rtpObservers {
		rtpObserver.routerClosed()
	}
	router.rtpObservers = make(map[string]RtpObserver)

	// Clear map of Router/PipeTransports.
	router.mapRouterPipeTransports = make(map[*Router][]Transport)

	router.Emit("@close")

	// Emit observer event.
	router.observer.SafeEmit("close")

	return
}

// Worker was closed.
func (router *Router) workerClosed() {
	if router.closed {
		return
	}

	router.logger.Debug("workerClosed()")

	router.closed = true

	// Close every Transport.
	for _, transport := range router.transports {
		transport.routerClosed()
	}
	router.transports = make(map[string]Transport)

	// Clear the Producers map.
	router.producers = make(map[string]*Producer)

	// Close every RtpObserver.
	for _, rtpObserver := range router.rtpObservers {
		rtpObserver.routerClosed()
	}
	router.rtpObservers = make(map[string]RtpObserver)

	// Clear map of Router/PipeTransports.
	router.mapRouterPipeTransports = make(map[*Router][]Transport)

	router.SafeEmit("workerclose")

	// Emit observer event.
	router.observer.SafeEmit("close")

	return
}

// Dump Router.
func (router *Router) Dump() Response {
	router.logger.Debug("dump()")

	return router.channel.Request("router.dump", router.internal)
}

/**
 * Create a WebRtcTransport.
 *
 * @param {Array<String|Object>} listenIps - Listen IPs in order of preference.
 *   Each entry can be a IP string or an object with ip and optional
 *   announcedIp strings.
 * @param {Boolean} [enableUdp=true] - Enable UDP.
 * @param {Boolean} [enableTcp=false] - Enable TCP.
 * @param {Boolean} [preferUdp=false] - Prefer UDP.
 * @param {Boolean} [preferTcp=false] - Prefer TCP.
 * @param {Object} [appData={}] - Custom app data.
 */
func (router *Router) CreateWebRtcTransport(
	params CreateWebRtcTransportParams,
) (transport Transport, err error) {
	router.logger.Debug("createWebRtcTransport()")

	internal := router.internal
	internal.TransportId = uuid.NewV4().String()
	reqData := params
	reqData.AppData = nil

	resp := router.channel.Request("router.createWebRtcTransport", internal, reqData)

	var data WebRtcTransportData
	if err = resp.Result(&data); err != nil {
		return
	}

	transport = NewWebRtcTransport(data, createTransportParams{
		Internal: router.internal,
		Channel:  router.channel,
		AppData:  params.AppData,
		GetRouterRtpCapabilities: func() RtpCapabilities {
			return router.data.RtpCapabilities
		},
		GetProducerById: func(producerId string) *Producer {
			return router.producers[producerId]
		},
	})

	router.transports[transport.Id()] = transport
	transport.On("@close", func() {
		delete(router.transports, transport.Id())
	})
	transport.On("@newproducer", func(producer *Producer) {
		router.producers[producer.Id()] = producer
	})
	transport.On("@producerclose", func(producer *Producer) {
		delete(router.producers, producer.Id())
	})

	// Emit observer event.
	router.observer.SafeEmit("newtransport", transport)

	return
}

/**
 * Create a PlainRtpTransport.
 *
 * @param {String|Object} listenIp - Listen IP string or an object with ip and
 *   optional announcedIp string.
 * @param {Boolean} [rtcpMux=true] - Use RTCP-mux.
 * @param {Boolean} [comedia=false] - Whether remote IP:port should be
 *   auto-detected based on first RTP/RTCP packet received. If enabled, connect()
 *   method must not be called. This option is ignored if multiSource is set.
 * @param {Boolean} [multiSource=false] - Whether RTP/RTCP from different remote
 *   IPs:ports is allowed. If set, the transport will just be valid for receiving
 *   media (consume() cannot be called on it) and connect() must not be called.
 * @param {Object} [appData={}] - Custom app data.
 */
func (router *Router) CreatePlainRtpTransport(
	params CreatePlainRtpTransportParams,
) (transport Transport, err error) {
	router.logger.Debug("createPlainRtpTransport()")

	internal := router.internal
	internal.TransportId = uuid.NewV4().String()
	reqData := params
	reqData.AppData = nil

	resp := router.channel.Request("router.createPlainRtpTransport", internal, reqData)

	var data PlainTransportData
	if err = resp.Result(&data); err != nil {
		return
	}

	transport = NewPlainRtpTransport(data, createTransportParams{
		Internal: router.internal,
		Channel:  router.channel,
		AppData:  params.AppData,
		GetRouterRtpCapabilities: func() RtpCapabilities {
			return router.data.RtpCapabilities
		},
		GetProducerById: func(producerId string) *Producer {
			return router.producers[producerId]
		},
	})

	router.transports[transport.Id()] = transport
	transport.On("@close", func() {
		delete(router.transports, transport.Id())
	})
	transport.On("@newproducer", func(producer *Producer) {
		router.producers[producer.Id()] = producer
	})
	transport.On("@producerclose", func(producer *Producer) {
		delete(router.producers, producer.Id())
	})

	// Emit observer event.
	router.observer.SafeEmit("newtransport", transport)

	return
}

/**
 * Create a PipeTransport.
 *
 * @param {String|Object} listenIp - Listen IP string or an object with ip and optional
 *   announcedIp string.
 * @param {Object} [appData={}] - Custom app data.
 */
func (router *Router) CreatePipeTransport(
	params CreatePipeTransportParams,
) (transport Transport, err error) {
	router.logger.Debug("createPipeTransport()")

	internal := router.internal
	internal.TransportId = uuid.NewV4().String()
	reqData := params
	reqData.AppData = nil

	resp := router.channel.Request("router.createPipeTransport", internal, reqData)

	var data PipeTransportData
	if err = resp.Result(&data); err != nil {
		return
	}

	transport = NewPipeTransport(data, createTransportParams{
		Internal: router.internal,
		Channel:  router.channel,
		AppData:  params.AppData,
		GetRouterRtpCapabilities: func() RtpCapabilities {
			return router.data.RtpCapabilities
		},
		GetProducerById: func(producerId string) *Producer {
			return router.producers[producerId]
		},
	})

	router.transports[transport.Id()] = transport
	transport.On("@close", func() {
		delete(router.transports, transport.Id())
	})
	transport.On("@newproducer", func(producer *Producer) {
		router.producers[producer.Id()] = producer
	})
	transport.On("@producerclose", func(producer *Producer) {
		delete(router.producers, producer.Id())
	})

	// Emit observer event.
	router.observer.SafeEmit("newtransport", transport)

	return
}

/**
 * Pipes the given Producer into another Router in same host.
 *
 * @param {String} producerId
 * @param {Router} router
 * @param {String|Object} [listenIp="127.0.0.1"] - Listen IP string or an
 *   object with ip and optional announcedIp string.
 *
 * @returns {Object} - Contains `pipeConsumer` {Consumer} created in the current
 *   Router and `pipeProducer` {Producer} created in the destination Router.
 */
func (router *Router) PipeToRouter(
	params PipeToRouterParams,
) (pipeConsumer *Consumer, pipeProducer *Producer, err error) {
	if len(params.ListenIp.Ip) == 0 {
		params.ListenIp.Ip = "127.0.0.1"
	}

	producer := router.producers[params.ProducerId]

	if producer == nil {
		err = errors.New("Producer not found")
		return
	}

	pipeTransportPair := router.mapRouterPipeTransports[router]
	var localPipeTransport, remotePipeTransport Transport

	defer func() {
		if err != nil {
			if localPipeTransport != nil {
				localPipeTransport.Close()
			}
			if remotePipeTransport != nil {
				remotePipeTransport.Close()
			}
		}
	}()

	if pipeTransportPair != nil {
		localPipeTransport = pipeTransportPair[0]
		remotePipeTransport = pipeTransportPair[1]
	} else {
		createPipeTransportParams := CreatePipeTransportParams{
			ListenIp: params.ListenIp,
		}

		localPipeTransport, err = router.CreatePipeTransport(createPipeTransportParams)
		if err != nil {
			return
		}
		remotePipeTransport, err = params.Router.CreatePipeTransport(createPipeTransportParams)
		if err != nil {
			return
		}

		localTuple := localPipeTransport.(*PipeTransport).Tuple()
		remoteTuple := remotePipeTransport.(*PipeTransport).Tuple()

		err = localPipeTransport.Connect(transportConnectParams{
			Ip:   remoteTuple.LocalIp,
			Port: remoteTuple.LocalPort,
		})
		if err != nil {
			return
		}
		err = remotePipeTransport.Connect(transportConnectParams{
			Ip:   localTuple.LocalIp,
			Port: localTuple.LocalPort,
		})
		if err != nil {
			return
		}

		localPipeTransport.Observer().On("close", func() {
			remotePipeTransport.Close()
			delete(router.mapRouterPipeTransports, router)
		})
		remotePipeTransport.Observer().On("close", func() {
			localPipeTransport.Close()
			delete(router.mapRouterPipeTransports, router)
		})

		router.mapRouterPipeTransports[router] =
			[]Transport{localPipeTransport, remotePipeTransport}
	}

	defer func() {
		if err != nil {
			if pipeConsumer != nil {
				pipeConsumer.Close()
			}
			if pipeProducer != nil {
				pipeProducer.Close()
			}
		}
	}()

	pipeConsumer, err = localPipeTransport.Consume(transportConsumeParams{
		ProducerId: params.ProducerId,
		Paused:     producer.Paused(),
	})
	if err != nil {
		return
	}
	pipeProducer, err = localPipeTransport.Produce(transportProduceParams{
		Id:            producer.Id(),
		Kind:          pipeConsumer.Kind(),
		RtpParameters: pipeConsumer.RtpParameters(),
		AppData:       producer.AppData(),
		Paused:        pipeConsumer.ProducerPaused(),
	})
	if err != nil {
		return
	}

	// Pipe events from the pipe Consumer to the pipe Producer.
	pipeConsumer.Observer().On("close", func() { pipeProducer.Close() })
	pipeConsumer.Observer().On("pause", func() { pipeProducer.Pause() })
	pipeConsumer.Observer().On("resume", func() { pipeProducer.Resume() })

	// Pipe events from the pipe Producer to the pipe Consumer.
	pipeProducer.Observer().On("close", func() { pipeConsumer.Close() })

	return
}

/**
 * Create an AudioLevelObserver.
 *
 * @param {Number} [maxEntries=1] - Maximum number of entries in the "volumes"
 *                                  event.
 * @param {Number} [threshold=-80] - Minimum average volume (in dBvo from -127 to 0)
 *                                   for entries in the "volumes" event.
 * @param {Number} [interval=1000] - Interval in ms for checking audio volumes.
 *
 */
func (router *Router) CreateAudioLevelObserver(
	params *CreateAudioLevelObserverParams,
) (rtpObserver RtpObserver, err error) {
	router.logger.Debug("createAudioLevelObserver()")

	if params == nil {
		params = &CreateAudioLevelObserverParams{
			MaxEntries: 1,
			Threshold:  -80,
			Interval:   1000,
		}
	}

	internal := router.internal
	internal.RtpObserverId = uuid.NewV4().String()

	resp := router.channel.Request("router.createAudioLevelObserver", internal, params)

	if err = resp.Err(); err != nil {
		return
	}

	rtpObserver = NewAudioLevelObserver(
		router.internal,
		router.channel,
		func(producerId string) *Producer {
			return router.producers[producerId]
		},
	)

	router.rtpObservers[rtpObserver.Id()] = rtpObserver
	rtpObserver.On("@close", func() {
		delete(router.rtpObservers, rtpObserver.Id())
	})

	return
}

/**
 * Check whether the given RTP capabilities can consume the given Producer.
 *
 * @param {String} producerId
 * @param {RTCRtpCapabilities} rtpCapabilities - Remote RTP capabilities.
 *
 */
func (router *Router) CanConsume(producerId string, rtpCapabilities RtpRemoteCapabilities) bool {
	producer := router.producers[producerId]

	if producer == nil {
		router.logger.Errorf(`canConsume() | Producer with id "%s" not found`, producerId)

		return false
	}

	return CanConsume(producer.ConsumableRtpParameters(), rtpCapabilities)
}
