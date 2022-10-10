package mediasoup

import (
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/singleflight"
)

// pipeRouterGroup is a singleflight group to avoid pipe the same routers pair
// at the same time.
var pipeRouterGroup = &singleflight.Group{}

// RouterOptions define options to create a router.
type RouterOptions struct {
	// MediaCodecs defines Router media codecs.
	MediaCodecs []*RtpCodecCapability `json:"mediaCodecs,omitempty"`

	// AppData is custom application data.
	AppData interface{} `json:"appData,omitempty"`
}

// PipeToRouterOptions define options to pipe an another router.
type PipeToRouterOptions struct {
	// ProducerId is the id of the Producer to consume.
	ProducerId string `json:"producerId,omitempty"`

	// DataProducerId is the id of the DataProducer to consume.
	DataProducerId string `json:"dataProducerId,omitempty"`

	// Router is the target Router instance.
	Router *Router `json:"router,omitempty"`

	// ListenIp is the ip used in the PipeTransport pair. Default '127.0.0.1'.
	ListenIp TransportListenIp `json:"listenIp,omitempty"`

	// Create a SCTP association. Default false.
	EnableSctp bool `json:"enableSctp,omitempty"`

	// NumSctpStreams define SCTP streams number.
	NumSctpStreams NumSctpStreams `json:"numSctpStreams,omitempty"`

	// EnableRtx enable RTX and NACK for RTP retransmission.
	EnableRtx bool `json:"enableRtx,omitempty"`

	// EnableSrtp enable SRTP.
	EnableSrtp bool `json:"enableSrtp,omitempty"`
}

// PipeToRouterResult is the result to piping router.
type PipeToRouterResult struct {
	// PipeConsumer is the Consumer created in the current Router.
	PipeConsumer *Consumer

	// PipeConsumer is the Producer created in the target Router.
	PipeProducer *Producer

	// PipeDataConsumer is the DataConsumer created in the current Router.
	PipeDataConsumer *DataConsumer

	// PipeDataProducer is the DataProducer created in the target Router.
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

// Router enables injection, selection and forwarding of media streams through
// Transport instances created on it.
//
// - @emits workerclose
// - @emits @close
type Router struct {
	IEventEmitter
	logger                  logr.Logger
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
	onNewRtpObserver        func(observer IRtpObserver)
	onNewTransport          func(transport ITransport)
}

func newRouter(params routerParams) *Router {
	logger := NewLogger("Router")
	logger.V(1).Info("constructor()", "internal", params.internal)

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

// Id returns Router id
func (router *Router) Id() string {
	return router.internal.RouterId
}

// Closed returns whether the Router is closed.
func (router *Router) Closed() bool {
	return atomic.LoadUint32(&router.closed) > 0
}

// RtpCapabilities returns RTC capabilities of the Router.
func (router *Router) RtpCapabilities() RtpCapabilities {
	return router.data.RtpCapabilities
}

// AppData returns App custom data.
func (router *Router) AppData() interface{} {
	return router.appData
}

// Observer returns the observer instance.
//
// - @emits close
// - @emits newrtpobserver - (observer IRtpObserver)
// - @emits newtransport - (transport ITransport)
func (router *Router) Observer() IEventEmitter {
	return router.observer
}

// transportsForTesting returns all transports in map. Just for testing purposes.
func (router *Router) transportsForTesting() map[string]ITransport {
	transports := make(map[string]ITransport)

	router.transports.Range(func(key, value interface{}) bool {
		transports[key.(string)] = value.(ITransport)
		return true
	})

	return transports
}

// Close the Router.
func (router *Router) Close() (err error) {
	router.logger.V(1).Info("close()")

	if !atomic.CompareAndSwapUint32(&router.closed, 0, 1) {
		return
	}

	reqData := H{"routerId": router.internal.RouterId}

	resp := router.channel.Request("worker.closeRouter", router.internal, reqData)
	if err = resp.Err(); err != nil {
		return
	}
	router.close()
	router.Emit("@close")
	return
}

func (router *Router) workerClosed() {
	router.logger.V(1).Info("workerClosed()")

	if !atomic.CompareAndSwapUint32(&router.closed, 0, 1) {
		return
	}
	router.close()
	router.Emit("workerclose")
}

func (router *Router) close() {
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

	// Emit observer event.
	router.observer.SafeEmit("close")
}

// Dump Router.
func (router *Router) Dump() (data *RouterDump, err error) {
	router.logger.V(1).Info("dump()")

	resp := router.channel.Request("router.dump", router.internal)
	err = resp.Unmarshal(&data)

	return
}

// Producers returns available producers on the router.
func (router *Router) Producers() []*Producer {
	router.logger.V(1).Info("Producers()")
	producers := make([]*Producer, 0)
	router.producers.Range(func(key, value interface{}) bool {
		producer, ok := value.(*Producer)
		if ok {
			producers = append(producers, producer)
		}
		return true
	})
	return producers
}

// DataProducers returns available producers on the router.
func (router *Router) DataProducers() []*DataProducer {
	router.logger.V(1).Info("DataProducers()")
	dataProducers := make([]*DataProducer, 0)
	router.dataProducers.Range(func(key, value interface{}) bool {
		dataProducer, ok := value.(*DataProducer)
		if ok {
			dataProducers = append(dataProducers, dataProducer)
		}
		return true
	})
	return dataProducers
}

// Transports returns available transports on the router.
func (router *Router) Transports() []ITransport {
	router.logger.V(1).Info("Transports()")
	transports := make([]ITransport, 0)
	router.transports.Range(func(key, value interface{}) bool {
		transport, ok := value.(ITransport)
		if ok {
			transports = append(transports, transport)
		}
		return true
	})
	return transports
}

// CreateWebRtcTransport create a WebRtcTransport.
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

	if len(options.ListenIps) == 0 && options.WebRtcServer == nil {
		err = NewTypeError("missing webRtcServer and listenIps (one of them is mandatory)")
		return
	}

	router.logger.V(1).Info("createWebRtcTransport()")

	method := "router.createWebRtcTransport"
	internal := router.internal
	internal.TransportId = uuid.NewString()

	reqData := H{
		"transportId":                     internal.TransportId,
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

	if options.WebRtcServer != nil {
		method = "router.createWebRtcTransportWithServer"
		reqData["webRtcServerId"] = option.WebRtcServer.Id()
	}

	var data *webrtcTransportData
	if err = router.channel.Request(method, internal, reqData).Unmarshal(&data); err != nil {
		return
	}
	transport = router.createTransport(internal, data, options.AppData).(*WebRtcTransport)

	if options.WebRtcServer != nil {
		options.WebRtcServer.handleWebRtcTransport(transport)
	}

	return
}

// CreatePlainTransport create a PlainTransport.
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

	router.logger.V(1).Info("createPlainTransport()")

	internal := router.internal
	internal.TransportId = uuid.NewString()
	reqData := H{
		"transportId":        internal.TransportId,
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

	var data *plainTransportData
	if err = resp.Unmarshal(&data); err != nil {
		return
	}

	iTransport := router.createTransport(internal, data, options.AppData)

	return iTransport.(*PlainTransport), nil
}

// CreatePipeTransport create a PipeTransport.
func (router *Router) CreatePipeTransport(option PipeTransportOptions) (transport *PipeTransport, err error) {
	options := &PipeTransportOptions{
		NumSctpStreams:     NumSctpStreams{OS: 1024, MIS: 1024},
		MaxSctpMessageSize: 268435456,
		SctpSendBufferSize: 268435456,
	}
	if err = override(options, option); err != nil {
		return
	}

	router.logger.V(1).Info("createPipeTransport()")

	internal := router.internal
	internal.TransportId = uuid.NewString()
	reqData := H{
		"transportId":        internal.TransportId,
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

	var data *pipeTransortData
	if err = resp.Unmarshal(&data); err != nil {
		return
	}

	iTransport := router.createTransport(internal, data, options.AppData)

	return iTransport.(*PipeTransport), nil
}

// CreateDirectTransport create a DirectTransport.
func (router *Router) CreateDirectTransport(params ...DirectTransportOptions) (transport *DirectTransport, err error) {
	options := &DirectTransportOptions{
		MaxMessageSize: 262144,
	}
	for _, option := range params {
		if err = override(options, option); err != nil {
			return
		}
	}

	router.logger.V(1).Info("createDirectTransport()")

	internal := router.internal
	internal.TransportId = uuid.NewString()
	reqData := H{
		"transportId":    internal.TransportId,
		"direct":         true,
		"maxMessageSize": options.MaxMessageSize,
	}

	resp := router.channel.Request("router.createDirectTransport", internal, reqData)

	var data *directTransportData
	if err = resp.Unmarshal(&data); err != nil {
		return
	}

	iTransport := router.createTransport(internal, data, options.AppData)

	return iTransport.(*DirectTransport), nil
}

// PipeToRouter pipes the given Producer or DataProducer into another Router in same host.
func (router *Router) PipeToRouter(option PipeToRouterOptions) (result *PipeToRouterResult, err error) {
	router.logger.V(1).Info("pipeToRouter()")

	options := &PipeToRouterOptions{
		ListenIp: TransportListenIp{
			Ip: "127.0.0.1",
		},
		EnableSctp:     false,
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
	var localPipeTransport, remotePipeTransport *PipeTransport

	if value, ok := router.mapRouterPipeTransports.Load(options.Router); ok {
		pipeTransportPair := value.([2]*PipeTransport)
		localPipeTransport, remotePipeTransport = pipeTransportPair[0], pipeTransportPair[1]
	} else {
		// Here we may have to create a new PipeTransport pair to connect source and
		// destination Routers. We just want to keep a PipeTransport pair for each
		// pair of Routers. Since this operation is async, it may happen that two
		// simultaneous calls to router1.pipeToRouter({ producerId: xxx, router: router2 })
		// would end up generating two pairs of PipeTranports. To prevent that, let's
		// use singleflight.
		routerPairIds := []string{router.Id(), options.Router.Id()}
		sort.Strings(routerPairIds)
		key := strings.Join(routerPairIds, "_")
		v, err, _ := pipeRouterGroup.Do(key, func() (result interface{}, err error) {
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

			option := PipeTransportOptions{
				ListenIp:       options.ListenIp,
				EnableSctp:     options.EnableSctp,
				NumSctpStreams: options.NumSctpStreams,
				EnableRtx:      options.EnableRtx,
				EnableSrtp:     options.EnableSrtp,
			}
			errgroup := new(errgroup.Group)

			errgroup.Go(func() error {
				localPipeTransport, err = router.CreatePipeTransport(option)
				return err
			})
			errgroup.Go(func() error {
				remotePipeTransport, err = options.Router.CreatePipeTransport(option)
				return err
			})
			if err = errgroup.Wait(); err != nil {
				router.logger.Error(err, "pipeToRouter() | error creating PipeTransport pair")
				return
			}
			errgroup.Go(func() error {
				return localPipeTransport.Connect(TransportConnectOptions{
					Ip:             remotePipeTransport.Tuple().LocalIp,
					Port:           remotePipeTransport.Tuple().LocalPort,
					SrtpParameters: remotePipeTransport.SrtpParameters(),
				})
			})
			errgroup.Go(func() error {
				return remotePipeTransport.Connect(TransportConnectOptions{
					Ip:             localPipeTransport.Tuple().LocalIp,
					Port:           localPipeTransport.Tuple().LocalPort,
					SrtpParameters: localPipeTransport.SrtpParameters(),
				})
			})
			if err = errgroup.Wait(); err != nil {
				router.logger.Error(err, "pipeToRouter() | error connecting PipeTransport pair")
				return
			}
			return []interface{}{router, [2]*PipeTransport{localPipeTransport, remotePipeTransport}}, nil
		})
		if err != nil {
			return nil, err
		}
		result := v.([]interface{})
		actualRouter := result[0].(*Router)
		pipeTransportPair := result[1].([2]*PipeTransport)
		// swap local and remote PipeTransport if pipeTransportPair is shared from the other router
		if actualRouter != router {
			pipeTransportPair[0], pipeTransportPair[1] = pipeTransportPair[1], pipeTransportPair[0]
		}
		localPipeTransport, remotePipeTransport = router.addPipeTransportPair(options.Router, pipeTransportPair)
	}

	if producer != nil {
		var pipeConsumer *Consumer
		var pipeProducer *Producer

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

		pipeConsumer, err = localPipeTransport.Consume(ConsumerOptions{
			ProducerId: options.ProducerId,
		})
		if err != nil {
			router.logger.Error(err, "pipeToRouter() | error creating pipe Consumer")
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
			router.logger.Error(err, "pipeToRouter() | error creating pipe Producer")
			return
		}
		// Ensure that the producer has not been closed in the meanwhile.
		if producer.Closed() {
			err = NewInvalidStateError("original Producer closed")
			router.logger.Error(err, "pipeToRouter() | failed")
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
				router.logger.Error(err, "pipeToRouter() | error pause or resume producer")
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
			router.logger.Error(err, "pipeToRouter() | error creating pipe DataConsumer pair")
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
			router.logger.Error(err, "pipeToRouter() | error creating pipe DataProducer pair")
			return
		}
		// Ensure that the dataProducer has not been closed in the meanwhile.
		if dataProducer.Closed() {
			err = NewInvalidStateError("original DataProducer closed")
			router.logger.Error(err, "pipeToRouter() | failed")
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
	}
	return
}

// addPipeTransportPair add PipeTransport pair for the another router
func (router *Router) addPipeTransportPair(anotherRouter *Router, pipeTransportPair [2]*PipeTransport) (*PipeTransport, *PipeTransport) {
	if val, loaded := router.mapRouterPipeTransports.LoadOrStore(anotherRouter, pipeTransportPair); loaded {
		router.logger.Info("pipeTransport exists, use the old pair", "router", anotherRouter.Id(), "warn", true)
		oldPipeTransportPair := val.([2]*PipeTransport)

		// close useless pipeTransport pair
		for i, transport := range pipeTransportPair {
			if transport != oldPipeTransportPair[i] {
				transport.Close()
			}
		}

		return oldPipeTransportPair[0], oldPipeTransportPair[1]
	}

	localPipeTransport, remotePipeTransport := pipeTransportPair[0], pipeTransportPair[1]

	localPipeTransport.Observer().On("close", func() {
		remotePipeTransport.Close()
		router.mapRouterPipeTransports.Delete(anotherRouter)
	})

	return localPipeTransport, remotePipeTransport
}

// CreateActiveSpeakerObserver create an ActiveSpeakerObserver
func (router *Router) CreateActiveSpeakerObserver(options ...func(*ActiveSpeakerObserverOptions)) (activeSpeakerObserver *ActiveSpeakerObserver, err error) {
	router.logger.V(1).Info("createActiveSpeakerObserver()")

	o := &ActiveSpeakerObserverOptions{
		Interval: 300,
	}
	for _, option := range options {
		option(o)
	}

	internal := router.internal
	internal.RtpObserverId = uuid.NewString()
	reqData := H{
		"rtpObserverId": internal.RtpObserverId,
		"interval":      o.Interval,
	}

	resp := router.channel.Request("router.createActiveSpeakerObserver", internal, reqData)
	if err = resp.Err(); err != nil {
		return
	}
	activeSpeakerObserver = newActiveSpeakerObserver(rtpObserverParams{
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
	router.rtpObservers.Store(activeSpeakerObserver.Id(), activeSpeakerObserver)
	activeSpeakerObserver.On("@close", func() {
		router.rtpObservers.Delete(activeSpeakerObserver.Id())
	})
	// Emit observer event.
	router.observer.SafeEmit("newrtpobserver", activeSpeakerObserver)

	if handler := router.onNewRtpObserver; handler != nil {
		handler(activeSpeakerObserver)
	}

	return
}

// CreateAudioLevelObserver create an AudioLevelObserver.
func (router *Router) CreateAudioLevelObserver(options ...func(o *AudioLevelObserverOptions)) (audioLevelObserver *AudioLevelObserver, err error) {
	router.logger.V(1).Info("createAudioLevelObserver()")

	o := AudioLevelObserverOptions{
		MaxEntries: 1,
		Threshold:  -80,
		Interval:   1000,
	}

	for _, option := range options {
		option(&o)
	}

	internal := router.internal
	internal.RtpObserverId = uuid.NewString()
	reqData := H{
		"rtpObserverId": internal.RtpObserverId,
		"maxEntries":    o.MaxEntries,
		"threshold":     o.Threshold,
		"interval":      o.Interval,
	}
	resp := router.channel.Request("router.createAudioLevelObserver", internal, reqData)
	if err = resp.Err(); err != nil {
		return
	}
	audioLevelObserver = newAudioLevelObserver(rtpObserverParams{
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

	router.rtpObservers.Store(audioLevelObserver.Id(), audioLevelObserver)
	audioLevelObserver.On("@close", func() {
		router.rtpObservers.Delete(audioLevelObserver.Id())
	})

	// Emit observer event.
	router.observer.SafeEmit("newrtpobserver", audioLevelObserver)

	if handler := router.onNewRtpObserver; handler != nil {
		handler(audioLevelObserver)
	}

	return
}

// CanConsume check whether the given RTP capabilities can consume the given Producer.
func (router *Router) CanConsume(producerId string, rtpCapabilities RtpCapabilities) bool {
	router.logger.V(1).Info("CanConsume()")

	value, ok := router.producers.Load(producerId)
	if !ok {
		router.logger.Error(nil, "canConsume() | Producer is not found", "id", producerId)
		return false
	}

	producer := value.(*Producer)
	ok, err := canConsume(producer.ConsumableRtpParameters(), rtpCapabilities)

	if err != nil {
		router.logger.Error(err, "canConsume() | unexpected error")
	}

	return ok
}

// OnNewRtpObserver set handler on "newrtpobserver" event
func (router *Router) OnNewRtpObserver(handler func(transport IRtpObserver)) {
	router.onNewRtpObserver = handler
}

// OnNewTransport set handler on "newtransport" event
func (router *Router) OnNewTransport(handler func(transport ITransport)) {
	router.onNewTransport = handler
}

// createTransport create a Transport interface.
func (router *Router) createTransport(internal internalData, data, appData interface{}) (transport ITransport) {
	if appData == nil {
		appData = H{}
	}

	var newTransport func(transportParams) ITransport

	switch data.(type) {
	case *directTransportData:
		newTransport = newDirectTransport

	case *plainTransportData:
		newTransport = newPlainTransport

	case *pipeTransortData:
		newTransport = newPipeTransport

	case *webrtcTransportData:
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
	transport.On("@listenserverclose", func() {
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

	if handler := router.onNewTransport; handler != nil {
		handler(transport)
	}

	return
}
