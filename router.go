package mediasoup

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"

	FbsActiveSpeakerObserver "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/ActiveSpeakerObserver"
	FbsAudioLevelObserver "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/AudioLevelObserver"
	FbsCommon "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/Common"
	FbsDirectTransport "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/DirectTransport"
	FbsPipeTransport "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/PipeTransport"
	FbsPlainTransport "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/PlainTransport"
	FbsRequest "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/Request"
	FbsRouter "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/Router"
	FbsSctpParameters "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/SctpParameters"
	FbsSrtpParameters "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/SrtpParameters"
	FbsTransport "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/Transport"
	FbsWebRtcTransport "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/WebRtcTransport"
	FbsWorker "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/Worker"

	"github.com/jiyeyuran/mediasoup-go/v2/internal/channel"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/singleflight"
)

// pipeRouterGroup is a singleflight group to avoid pipe the same routers pair
// at the same time.
var pipeRouterGroup = &singleflight.Group{}

type routerData struct {
	RouterId        string
	RtpCapabilities *RtpCapabilities
	AppData         H
}

type Router struct {
	baseListener

	channel *channel.Channel
	data    *routerData
	closed  bool
	logger  *slog.Logger

	transports    sync.Map
	rtpObservers  sync.Map
	producers     sync.Map
	consumers     sync.Map
	dataProducers sync.Map
	dataConsumers sync.Map

	newRtpObserverListeners []func(context.Context, *RtpObserver)
	newTransportListeners   []func(context.Context, *Transport)

	routerPipeMu            sync.Mutex
	mapRouterPipeTransports map[*Router][2]*Transport
}

func newRouter(channel *channel.Channel, logger *slog.Logger, data *routerData) *Router {
	return &Router{
		channel:                 channel,
		data:                    data,
		logger:                  logger.With("routerId", data.RouterId),
		mapRouterPipeTransports: make(map[*Router][2]*Transport),
	}
}

func (r *Router) Id() string {
	return r.data.RouterId
}

func (r *Router) RtpCapabilities() *RtpCapabilities {
	return r.data.RtpCapabilities
}

func (r *Router) GetTransportById(id string) *Transport {
	transport, ok := r.transports.Load(id)
	if !ok {
		return nil
	}
	return transport.(*Transport)
}

func (r *Router) GetRtpObserverById(id string) *RtpObserver {
	observer, ok := r.rtpObservers.Load(id)
	if !ok {
		return nil
	}
	return observer.(*RtpObserver)
}

func (r *Router) GetProducerById(id string) *Producer {
	producer, ok := r.producers.Load(id)
	if !ok {
		return nil
	}
	return producer.(*Producer)
}

func (r *Router) GetConsumerById(id string) *Consumer {
	consumer, ok := r.consumers.Load(id)
	if !ok {
		return nil
	}
	return consumer.(*Consumer)
}

func (r *Router) GetDataProducerById(id string) *DataProducer {
	dataProducer, ok := r.dataProducers.Load(id)
	if !ok {
		return nil
	}
	return dataProducer.(*DataProducer)
}

func (r *Router) GetDataConsumerById(id string) *DataConsumer {
	dataConsumer, ok := r.dataConsumers.Load(id)
	if !ok {
		return nil
	}
	return dataConsumer.(*DataConsumer)
}

func (r *Router) ConsumersForProducer(producerId string) (consumers []*Consumer) {
	r.consumers.Range(func(key, value any) bool {
		if consumer := value.(*Consumer); consumer.ProducerId() == producerId {
			consumers = append(consumers, consumer)
		}
		return true
	})
	return
}

func (r *Router) DataConsumersForDataProducer(dataProducerId string) (dataConsumers []*DataConsumer) {
	r.dataConsumers.Range(func(key, value any) bool {
		if dataConsumer := value.(*DataConsumer); dataConsumer.DataProducerId() == dataProducerId {
			dataConsumers = append(dataConsumers, dataConsumer)
		}
		return true
	})
	return
}

// CanConsume check whether the given RTP capabilities can consume the given Producer.
func (r *Router) CanConsume(producerId string, rtpCapabilities *RtpCapabilities) bool {
	producer := r.GetProducerById(producerId)
	if producer == nil {
		return false
	}
	ok, _ := CanConsume(producer.ConsumableRtpParameters(), rtpCapabilities)
	return ok
}

func (r *Router) Close() error {
	return r.CloseContext(context.Background())
}

func (r *Router) CloseContext(ctx context.Context) error {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil
	}
	r.logger.DebugContext(ctx, "Close()")

	_, err := r.channel.Request(ctx, &FbsRequest.RequestT{
		Method: FbsRequest.MethodWORKER_CLOSE_ROUTER,
		Body: &FbsRequest.BodyT{
			Type: FbsRequest.BodyWorker_CloseRouterRequest,
			Value: &FbsWorker.CloseRouterRequestT{
				RouterId: r.Id(),
			},
		},
	})
	if err != nil {
		r.mu.Unlock()
		return err
	}
	r.closed = true
	r.mu.Unlock()

	r.cleanupAfterClosed(ctx)
	return nil
}

func (r *Router) Closed() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.closed
}

func (r *Router) Dump() (dump *RouterDump, err error) {
	return r.DumpContext(context.Background())
}

func (r *Router) DumpContext(ctx context.Context) (dump *RouterDump, err error) {
	r.logger.DebugContext(ctx, "Dump()")

	msg, err := r.channel.Request(ctx, &FbsRequest.RequestT{
		Method:    FbsRequest.MethodROUTER_DUMP,
		HandlerId: r.Id(),
	})
	if err != nil {
		return nil, err
	}
	result := msg.(*FbsRouter.DumpResponseT)

	return &RouterDump{
		Id:             result.Id,
		TransportIds:   result.TransportIds,
		RtpObserverIds: result.RtpObserverIds,
		MapProducerIdConsumerIds: collect(result.MapProducerIdConsumerIds,
			func(item *FbsCommon.StringStringArrayT) KeyValues[string, string] {
				return KeyValues[string, string]{
					Key:    item.Key,
					Values: item.Values,
				}
			}),
		MapConsumerIdProducerId: collect(result.MapConsumerIdProducerId,
			func(item *FbsCommon.StringStringT) KeyValue[string, string] {
				return KeyValue[string, string]{
					Key:   item.Key,
					Value: item.Value,
				}
			}),
		MapProducerIdObserverIds: collect(result.MapProducerIdObserverIds,
			func(item *FbsCommon.StringStringArrayT) KeyValues[string, string] {
				return KeyValues[string, string]{
					Key:    item.Key,
					Values: item.Values,
				}
			}),
		MapDataProducerIdDataConsumerIds: collect(result.MapDataProducerIdDataConsumerIds,
			func(item *FbsCommon.StringStringArrayT) KeyValues[string, string] {
				return KeyValues[string, string]{
					Key:    item.Key,
					Values: item.Values,
				}
			}),
		MapDataConsumerIdDataProducerId: collect(result.MapDataConsumerIdDataProducerId,
			func(item *FbsCommon.StringStringT) KeyValue[string, string] {
				return KeyValue[string, string]{
					Key:   item.Key,
					Value: item.Value,
				}
			}),
	}, nil
}

func (r *Router) CreateActiveSpeakerObserver(options *ActiveSpeakerObserverOptions) (*RtpObserver, error) {
	return r.CreateActiveSpeakerObserverContext(context.Background(), options)
}

func (r *Router) CreateActiveSpeakerObserverContext(ctx context.Context, options *ActiveSpeakerObserverOptions) (*RtpObserver, error) {
	r.logger.DebugContext(ctx, "CreateActiveSpeakerObserver()")

	o := &ActiveSpeakerObserverOptions{
		Interval: 300,
		AppData:  H{},
	}
	if options != nil {
		if options.Interval > 0 {
			o.Interval = options.Interval
		}
		if options.AppData != nil {
			o.AppData = options.AppData
		}
	}

	rtpObserverId := UUID(rtpObserverPrefix)

	_, err := r.channel.Request(ctx, &FbsRequest.RequestT{
		Method:    FbsRequest.MethodROUTER_CREATE_ACTIVESPEAKEROBSERVER,
		HandlerId: r.Id(),
		Body: &FbsRequest.BodyT{
			Type: FbsRequest.BodyRouter_CreateActiveSpeakerObserverRequest,
			Value: &FbsRouter.CreateActiveSpeakerObserverRequestT{
				RtpObserverId: rtpObserverId,
				Options: &FbsActiveSpeakerObserver.ActiveSpeakerObserverOptionsT{
					Interval: o.Interval,
				},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	return r.newRtpObserver(ctx, &rtpObserverData{
		Id:              rtpObserverId,
		Type:            RtpObserverTypeActiveSpeaker,
		RouterId:        r.Id(),
		AppData:         o.AppData,
		GetProducerById: r.GetProducerById,
	})
}

func (r *Router) CreateAudioLevelObserver(options *AudioLevelObserverOptions) (*RtpObserver, error) {
	return r.CreateAudioLevelObserverContext(context.Background(), options)
}

func (r *Router) CreateAudioLevelObserverContext(ctx context.Context, options *AudioLevelObserverOptions) (*RtpObserver, error) {
	r.logger.DebugContext(ctx, "CreateAudioLevelObserver()")

	o := &AudioLevelObserverOptions{
		MaxEntries: 1,
		Threshold:  -80,
		Interval:   1000,
		AppData:    H{},
	}
	if options != nil {
		if options.MaxEntries > 0 {
			o.MaxEntries = options.MaxEntries
		}
		if options.Threshold > 0 || options.Threshold < -127 {
			return nil, errors.New("if given, threshold must be a negative number greater than -127")
		}
		if options.Threshold != 0 {
			o.Threshold = options.Threshold
		}
		if options.Interval > 0 {
			o.Interval = options.Interval
		}
		if options.AppData != nil {
			o.AppData = options.AppData
		}
	}

	rtpObserverId := UUID(rtpObserverPrefix)

	_, err := r.channel.Request(ctx, &FbsRequest.RequestT{
		Method:    FbsRequest.MethodROUTER_CREATE_AUDIOLEVELOBSERVER,
		HandlerId: r.Id(),
		Body: &FbsRequest.BodyT{
			Type: FbsRequest.BodyRouter_CreateAudioLevelObserverRequest,
			Value: &FbsRouter.CreateAudioLevelObserverRequestT{
				RtpObserverId: rtpObserverId,
				Options: &FbsAudioLevelObserver.AudioLevelObserverOptionsT{
					MaxEntries: o.MaxEntries,
					Interval:   o.Interval,
					Threshold:  o.Threshold,
				},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	return r.newRtpObserver(ctx, &rtpObserverData{
		Id:              rtpObserverId,
		Type:            RtpObserverTypeAudioLevel,
		RouterId:        r.Id(),
		AppData:         o.AppData,
		GetProducerById: r.GetProducerById,
	})
}

func (r *Router) CreateWebRtcTransport(options *WebRtcTransportOptions) (*Transport, error) {
	return r.CreateWebRtcTransportContext(context.Background(), options)
}

func (r *Router) CreateWebRtcTransportContext(ctx context.Context, options *WebRtcTransportOptions) (*Transport, error) {
	r.logger.DebugContext(ctx, "CreateWebRtcTransport()")

	o := &WebRtcTransportOptions{
		WebRtcServer:                    options.WebRtcServer,
		ListenInfos:                     options.ListenInfos,
		EnableUdp:                       ref(true),
		EnableTcp:                       options.EnableTcp,
		PreferUdp:                       options.PreferUdp,
		PreferTcp:                       options.PreferTcp,
		IceConsentTimeout:               ref[uint8](30),
		InitialAvailableOutgoingBitrate: 600000,
		EnableSctp:                      options.EnableSctp,
		NumSctpStreams:                  &NumSctpStreams{OS: 1024, MIS: 1024},
		MaxSctpMessageSize:              262144,
		SctpSendBufferSize:              262144,
		AppData:                         orElse(options.AppData != nil, options.AppData, H{}),
	}
	if len(o.ListenInfos) == 0 && o.WebRtcServer == nil {
		return nil, errors.New("missing webRtcServerId and listenIps (one of them is mandatory)")
	}
	if options.EnableUdp != nil {
		o.EnableUdp = options.EnableUdp
	}
	if options.InitialAvailableOutgoingBitrate > 0 {
		o.InitialAvailableOutgoingBitrate = options.InitialAvailableOutgoingBitrate
	}
	if options.NumSctpStreams != nil && options.NumSctpStreams.OS > 0 && options.NumSctpStreams.MIS > 0 {
		o.NumSctpStreams = options.NumSctpStreams
	}
	if options.MaxSctpMessageSize > 0 {
		o.MaxSctpMessageSize = options.MaxSctpMessageSize
	}
	if options.SctpSendBufferSize > 0 {
		o.SctpSendBufferSize = options.SctpSendBufferSize
	}

	transportId := UUID(transportPrefix)
	baseTranportOptions := &FbsTransport.OptionsT{
		InitialAvailableOutgoingBitrate: ref(o.InitialAvailableOutgoingBitrate),
		EnableSctp:                      o.EnableSctp,
		NumSctpStreams: &FbsSctpParameters.NumSctpStreamsT{
			Os:  o.NumSctpStreams.OS,
			Mis: o.NumSctpStreams.MIS,
		},
		MaxSctpMessageSize: o.MaxSctpMessageSize,
		SctpSendBufferSize: o.SctpSendBufferSize,
		IsDataChannel:      true,
	}

	var (
		method FbsRequest.Method
		listen *FbsWebRtcTransport.ListenT
	)

	if o.WebRtcServer != nil {
		method = FbsRequest.MethodROUTER_CREATE_WEBRTCTRANSPORT_WITH_SERVER
		listen = &FbsWebRtcTransport.ListenT{
			Type: FbsWebRtcTransport.ListenListenServer,
			Value: &FbsWebRtcTransport.ListenServerT{
				WebRtcServerId: o.WebRtcServer.Id(),
			},
		}
	} else {
		method = FbsRequest.MethodROUTER_CREATE_WEBRTCTRANSPORT
		listen = &FbsWebRtcTransport.ListenT{
			Type: FbsWebRtcTransport.ListenListenIndividual,
			Value: &FbsWebRtcTransport.ListenIndividualT{
				ListenInfos: collect(o.ListenInfos, convertTransportListenInfo),
			},
		}
	}

	msg, err := r.channel.Request(ctx, &FbsRequest.RequestT{
		Method:    method,
		HandlerId: r.Id(),
		Body: &FbsRequest.BodyT{
			Type: FbsRequest.BodyRouter_CreateWebRtcTransportRequest,
			Value: &FbsRouter.CreateWebRtcTransportRequestT{
				TransportId: transportId,
				Options: &FbsWebRtcTransport.WebRtcTransportOptionsT{
					Base:              baseTranportOptions,
					Listen:            listen,
					EnableUdp:         *o.EnableUdp,
					EnableTcp:         o.EnableTcp,
					PreferUdp:         o.PreferUdp,
					PreferTcp:         o.PreferTcp,
					IceConsentTimeout: *o.IceConsentTimeout,
				},
			},
		},
	})
	if err != nil {
		return nil, err
	}
	result := msg.(*FbsWebRtcTransport.DumpResponseT)

	data := &internalTransportData{
		TransportId:   transportId,
		TransportType: TransportWebRTC,
		AppData:       o.AppData,
		TransportData: &TransportData{
			WebRtcTransportData: &WebRtcTransportData{
				IceRole: strings.ToLower(result.IceRole.String()),
				IceParameters: IceParameters{
					UsernameFragment: result.IceParameters.UsernameFragment,
					Password:         result.IceParameters.Password,
					IceLite:          result.IceParameters.IceLite,
				},
				IceCandidates: collect(result.IceCandidates, func(item *FbsWebRtcTransport.IceCandidateT) IceCandidate {
					return IceCandidate{
						Foundation: item.Foundation,
						Priority:   item.Priority,
						Address:    item.Address,
						Protocol:   TransportProtocol(strings.ToLower(item.Protocol.String())),
						Port:       item.Port,
						Type:       strings.ToLower(item.Type.String()),
						TcpType: ifElse(item.TcpType != nil, func() string {
							return strings.ToLower(item.TcpType.String())
						}),
					}
				}),
				IceState: IceState(strings.ToLower(result.IceState.String())),
				IceSelectedTuple: ifElse(result.IceSelectedTuple != nil, func() *TransportTuple {
					return &TransportTuple{
						Protocol:     TransportProtocol(strings.ToLower(result.IceSelectedTuple.Protocol.String())),
						LocalAddress: result.IceSelectedTuple.LocalAddress,
						LocalPort:    result.IceSelectedTuple.LocalPort,
						RemoteIp:     result.IceSelectedTuple.RemoteIp,
						RemotePort:   result.IceSelectedTuple.RemotePort,
					}
				}),
				DtlsParameters: DtlsParameters{
					Role:         DtlsRole(strings.ToLower(result.DtlsParameters.Role.String())),
					Fingerprints: collect(result.DtlsParameters.Fingerprints, parseDtlsFingerprint),
				},
				DtlsState: DtlsState(strings.ToLower(result.DtlsState.String())),
				SctpParameters: ifElse(result.Base.SctpParameters != nil, func() *SctpParameters {
					return &SctpParameters{
						Port:               result.Base.SctpParameters.Port,
						OS:                 result.Base.SctpParameters.Os,
						MIS:                result.Base.SctpParameters.Mis,
						MaxMessageSize:     result.Base.SctpParameters.MaxMessageSize,
						SctpBufferedAmount: result.Base.SctpParameters.SctpBufferedAmount,
						IsDataChannel:      result.Base.SctpParameters.IsDataChannel,
					}
				}),
				SctpState: ifElse(result.Base.SctpState != nil, func() SctpState {
					return SctpState(strings.ToLower(result.Base.SctpState.String()))
				}),
			},
		},
	}
	transport, err := r.newTransport(ctx, data)
	if err != nil {
		return nil, err
	}
	if o.WebRtcServer != nil {
		o.WebRtcServer.handleWebRtcTransport(transport)
	}
	return transport, nil
}

func (r *Router) CreatePlainTransport(options *PlainTransportOptions) (*Transport, error) {
	return r.CreatePlainTransportContext(context.Background(), options)
}

func (r *Router) CreatePlainTransportContext(ctx context.Context, options *PlainTransportOptions) (*Transport, error) {
	r.logger.DebugContext(ctx, "CreatePlainTransport()")

	o := &PlainTransportOptions{
		ListenInfo:         options.ListenInfo,
		RtcpListenInfo:     options.RtcpListenInfo,
		RtcpMux:            ref(true),
		Comedia:            options.Comedia,
		EnableSctp:         options.EnableSctp,
		NumSctpStreams:     &NumSctpStreams{OS: 1024, MIS: 1024},
		MaxSctpMessageSize: 262144,
		SctpSendBufferSize: 262144,
		EnableSrtp:         options.EnableSrtp,
		SrtpCryptoSuite:    AES_CM_128_HMAC_SHA1_80,
		AppData:            orElse(options.AppData != nil, options.AppData, H{}),
	}
	if options.RtcpMux != nil {
		o.RtcpMux = options.RtcpMux
	}
	if options.NumSctpStreams != nil {
		o.NumSctpStreams = options.NumSctpStreams
	}
	if options.MaxSctpMessageSize > 0 {
		o.MaxSctpMessageSize = options.MaxSctpMessageSize
	}
	if options.SctpSendBufferSize > 0 {
		o.SctpSendBufferSize = options.SctpSendBufferSize
	}

	transportId := UUID(transportPrefix)
	baseTranportOptions := &FbsTransport.OptionsT{
		EnableSctp: o.EnableSctp,
		NumSctpStreams: &FbsSctpParameters.NumSctpStreamsT{
			Os:  o.NumSctpStreams.OS,
			Mis: o.NumSctpStreams.MIS,
		},
		MaxSctpMessageSize: o.MaxSctpMessageSize,
		SctpSendBufferSize: o.SctpSendBufferSize,
		IsDataChannel:      false,
	}

	msg, err := r.channel.Request(ctx, &FbsRequest.RequestT{
		Method:    FbsRequest.MethodROUTER_CREATE_PLAINTRANSPORT,
		HandlerId: r.Id(),
		Body: &FbsRequest.BodyT{
			Type: FbsRequest.BodyRouter_CreatePlainTransportRequest,
			Value: &FbsRouter.CreatePlainTransportRequestT{
				TransportId: transportId,
				Options: &FbsPlainTransport.PlainTransportOptionsT{
					Base:       baseTranportOptions,
					ListenInfo: convertTransportListenInfo(o.ListenInfo),
					RtcpListenInfo: ifElse(o.RtcpListenInfo != nil, func() *FbsTransport.ListenInfoT {
						return convertTransportListenInfo(*o.RtcpListenInfo)
					}),
					RtcpMux:    *o.RtcpMux,
					Comedia:    o.Comedia,
					EnableSrtp: o.EnableSrtp,
					SrtpCryptoSuite: ifElse(len(o.SrtpCryptoSuite) > 0, func() *FbsSrtpParameters.SrtpCryptoSuite {
						return ref(FbsSrtpParameters.EnumValuesSrtpCryptoSuite[strings.ToLower(string(o.SrtpCryptoSuite))])
					}),
				},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	result := msg.(*FbsPlainTransport.DumpResponseT)
	data := &internalTransportData{
		TransportId:   transportId,
		TransportType: TransportPlain,
		TransportData: &TransportData{
			PlainTransportData: &PlainTransportData{
				Tuple:     *parseTransportTuple(result.Tuple),
				RtcpTuple: parseTransportTuple(result.Tuple),
				SctpParameters: ifElse(result.Base.SctpParameters != nil, func() *SctpParameters {
					return &SctpParameters{
						Port:               result.Base.SctpParameters.Port,
						OS:                 result.Base.SctpParameters.Os,
						MIS:                result.Base.SctpParameters.Mis,
						MaxMessageSize:     result.Base.SctpParameters.MaxMessageSize,
						SctpBufferedAmount: result.Base.SctpParameters.SctpBufferedAmount,
						IsDataChannel:      result.Base.SctpParameters.IsDataChannel,
					}
				}),
				SctpState: ifElse(result.Base.SctpState != nil, func() SctpState {
					return SctpState(strings.ToLower(result.Base.SctpState.String()))
				}),
				SrtpParameters: parseSrtpParameters(result.SrtpParameters),
			},
		},
		RtcpMux: result.RtcpMux,
		Comedia: result.Comedia,
		AppData: o.AppData,
	}
	return r.newTransport(ctx, data)
}

func (r *Router) CreatePipeTransport(options *PipeTransportOptions) (*Transport, error) {
	return r.CreatePipeTransportContext(context.Background(), options)
}

func (r *Router) CreatePipeTransportContext(ctx context.Context, options *PipeTransportOptions) (*Transport, error) {
	r.logger.DebugContext(ctx, "CreatePipeTransport()")

	o := &PipeTransportOptions{
		ListenInfo:         options.ListenInfo,
		EnableSctp:         options.EnableSctp,
		NumSctpStreams:     &NumSctpStreams{OS: 1024, MIS: 1024},
		MaxSctpMessageSize: 268435456,
		SctpSendBufferSize: 268435456,
		EnableSrtp:         options.EnableSrtp,
		EnableRtx:          options.EnableRtx,
		AppData:            orElse(options.AppData != nil, options.AppData, H{}),
	}
	if options.NumSctpStreams != nil {
		o.NumSctpStreams = options.NumSctpStreams
	}
	if options.MaxSctpMessageSize > 0 {
		o.MaxSctpMessageSize = options.MaxSctpMessageSize
	}
	if options.SctpSendBufferSize > 0 {
		o.SctpSendBufferSize = options.SctpSendBufferSize
	}

	transportId := UUID(transportPrefix)
	baseTranportOptions := &FbsTransport.OptionsT{
		EnableSctp: o.EnableSctp,
		NumSctpStreams: &FbsSctpParameters.NumSctpStreamsT{
			Os:  o.NumSctpStreams.OS,
			Mis: o.NumSctpStreams.MIS,
		},
		MaxSctpMessageSize: o.MaxSctpMessageSize,
		SctpSendBufferSize: o.SctpSendBufferSize,
		IsDataChannel:      false,
	}

	msg, err := r.channel.Request(ctx, &FbsRequest.RequestT{
		Method:    FbsRequest.MethodROUTER_CREATE_PIPETRANSPORT,
		HandlerId: r.Id(),
		Body: &FbsRequest.BodyT{
			Type: FbsRequest.BodyRouter_CreatePipeTransportRequest,
			Value: &FbsRouter.CreatePipeTransportRequestT{
				TransportId: transportId,
				Options: &FbsPipeTransport.PipeTransportOptionsT{
					Base:       baseTranportOptions,
					ListenInfo: convertTransportListenInfo(o.ListenInfo),
					EnableRtx:  o.EnableRtx,
					EnableSrtp: o.EnableSrtp,
				},
			},
		},
	})
	if err != nil {
		return nil, err
	}
	result := msg.(*FbsPipeTransport.DumpResponseT)
	data := &internalTransportData{
		TransportId:   transportId,
		TransportType: TransportPipe,
		TransportData: &TransportData{
			PipeTransportData: &PipeTransportData{
				Tuple: *parseTransportTuple(result.Tuple),
				SctpParameters: ifElse(result.Base.SctpParameters != nil, func() *SctpParameters {
					return &SctpParameters{
						Port:               result.Base.SctpParameters.Port,
						OS:                 result.Base.SctpParameters.Os,
						MIS:                result.Base.SctpParameters.Mis,
						MaxMessageSize:     result.Base.SctpParameters.MaxMessageSize,
						SctpBufferedAmount: result.Base.SctpParameters.SctpBufferedAmount,
						IsDataChannel:      result.Base.SctpParameters.IsDataChannel,
					}
				}),
				SctpState: ifElse(result.Base.SctpState != nil, func() SctpState {
					return SctpState(strings.ToLower(result.Base.SctpState.String()))
				}),
				SrtpParameters: parseSrtpParameters(result.SrtpParameters),
			},
		},
		Rtx:     result.Rtx,
		AppData: o.AppData,
	}
	return r.newTransport(ctx, data)
}

func (r *Router) CreateDirectTransport(options *DirectTransportOptions) (*Transport, error) {
	return r.CreateDirectTransportContext(context.Background(), options)
}

func (r *Router) CreateDirectTransportContext(ctx context.Context, options *DirectTransportOptions) (*Transport, error) {
	r.logger.DebugContext(ctx, "CreateDirectTransport()")

	o := &DirectTransportOptions{
		MaxMessageSize: 262144,
		AppData:        H{},
	}
	if options != nil {
		if options.MaxMessageSize > 0 {
			o.MaxMessageSize = options.MaxMessageSize
		}
		if options.AppData != nil {
			o.AppData = options.AppData
		}
	}

	transportId := UUID(transportPrefix)
	baseTranportOptions := &FbsTransport.OptionsT{
		Direct:         true,
		MaxMessageSize: &o.MaxMessageSize,
	}

	_, err := r.channel.Request(ctx, &FbsRequest.RequestT{
		Method:    FbsRequest.MethodROUTER_CREATE_DIRECTTRANSPORT,
		HandlerId: r.Id(),
		Body: &FbsRequest.BodyT{
			Type: FbsRequest.BodyRouter_CreateDirectTransportRequest,
			Value: &FbsRouter.CreateDirectTransportRequestT{
				TransportId: transportId,
				Options: &FbsDirectTransport.DirectTransportOptionsT{
					Base: baseTranportOptions,
				},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	data := &internalTransportData{
		TransportId:   transportId,
		TransportType: TransportDirect,
		TransportData: &TransportData{},
		AppData:       o.AppData,
	}
	return r.newTransport(ctx, data)
}

// PipeToRouter pipes the given Producer or DataProducer into another Router in same host.
func (r *Router) PipeToRouter(options *PipeToRouterOptions) (result *PipeToRouterResult, err error) {
	return r.PipeToRouterContext(context.Background(), options)
}

func (r *Router) PipeToRouterContext(ctx context.Context, options *PipeToRouterOptions) (result *PipeToRouterResult, err error) {
	r.logger.DebugContext(ctx, "PipeToRouter()")

	o := &PipeToRouterOptions{
		ListenInfo: TransportListenInfo{
			Protocol: TransportProtocolUDP,
			Ip:       "127.0.0.1",
		},
		ProducerId:     options.ProducerId,
		DataProducerId: options.DataProducerId,
		Router:         options.Router,
		EnableSctp:     ref(true),
		NumSctpStreams: &NumSctpStreams{OS: 1024, MIS: 1024},
		EnableRtx:      options.EnableRtx,
		EnableSrtp:     options.EnableSrtp,
	}
	if options.ListenInfo != (TransportListenInfo{}) {
		o.ListenInfo = options.ListenInfo
	}
	if options.EnableSctp != nil {
		o.EnableSctp = options.EnableSctp
	}
	if options.NumSctpStreams != nil {
		o.NumSctpStreams = options.NumSctpStreams
	}

	if len(o.ProducerId) == 0 && len(o.DataProducerId) == 0 {
		return nil, errors.New("missing producerId")
	}
	if len(o.ProducerId) > 0 && len(o.DataProducerId) > 0 {
		return nil, errors.New("just producerId or dataProducerId can be given")
	}
	if o.Router == nil {
		return nil, errors.New("Router not found")
	}
	if o.Router == r {
		return nil, errors.New("cannot use this Router as destination'")
	}

	var producer *Producer
	var dataProducer *DataProducer

	if len(o.ProducerId) > 0 {
		if producer = r.GetProducerById(o.ProducerId); producer == nil {
			return nil, errors.New("Producer not found")
		}
	}
	if len(o.DataProducerId) > 0 {
		if dataProducer = r.GetDataProducerById(o.DataProducerId); dataProducer == nil {
			return nil, errors.New("DataProducer not found")
		}
	}

	r.routerPipeMu.Lock()
	pipeTransportPair, ok := r.mapRouterPipeTransports[o.Router]
	if !ok {
		// Here we may have to create a new PipeTransport pair to connect source and
		// destination Routers. We just want to keep a PipeTransport pair for each
		// pair of Routers. Since this operation is async, it may happen that two
		// simultaneous calls to router1.pipeToRouter({ producerId: xxx, router: router2 })
		// would end up generating two pairs of PipeTranports. To prevent that, let's
		// use singleflight.
		keyParts := []string{r.Id(), o.Router.Id()}
		if keyParts[0] > keyParts[1] {
			keyParts[0], keyParts[1] = keyParts[1], keyParts[0]
		}
		key := strings.Join(keyParts, "-")
		v, err, _ := pipeRouterGroup.Do(key, func() (result any, err error) {
			var localPipeTransport, remotePipeTransport *Transport
			defer func() {
				if err != nil {
					if localPipeTransport != nil {
						localPipeTransport.CloseContext(ctx)
					}
					if remotePipeTransport != nil {
						remotePipeTransport.CloseContext(ctx)
					}
				}
			}()

			options := &PipeTransportOptions{
				ListenInfo:     o.ListenInfo,
				EnableSctp:     *o.EnableSctp,
				NumSctpStreams: o.NumSctpStreams,
				EnableRtx:      o.EnableRtx,
				EnableSrtp:     o.EnableSrtp,
			}
			errgroup := new(errgroup.Group)

			errgroup.Go(func() error {
				localPipeTransport, err = r.CreatePipeTransportContext(ctx, options)
				return err
			})
			errgroup.Go(func() error {
				remotePipeTransport, err = o.Router.CreatePipeTransportContext(ctx, options)
				return err
			})
			if err = errgroup.Wait(); err != nil {
				return
			}
			errgroup.Go(func() error {
				data := remotePipeTransport.Data().PipeTransportData
				return localPipeTransport.ConnectContext(ctx, &TransportConnectOptions{
					Ip:             data.Tuple.LocalAddress,
					Port:           ref(data.Tuple.LocalPort),
					SrtpParameters: data.SrtpParameters,
				})
			})
			errgroup.Go(func() error {
				data := localPipeTransport.Data().PipeTransportData
				return remotePipeTransport.ConnectContext(ctx, &TransportConnectOptions{
					Ip:             data.Tuple.LocalAddress,
					Port:           ref(data.Tuple.LocalPort),
					SrtpParameters: data.SrtpParameters,
				})
			})
			if err = errgroup.Wait(); err != nil {
				return
			}
			return [2]any{r, [2]*Transport{localPipeTransport, remotePipeTransport}}, nil
		})
		if err != nil {
			r.routerPipeMu.Unlock()
			return nil, err
		}
		result := v.([2]any)
		pipeTransportPair = result[1].([2]*Transport)
		// swap local and remote PipeTransport if pipeTransportPair is shared from the other router
		if result[0].(*Router) != r {
			pipeTransportPair[0], pipeTransportPair[1] = pipeTransportPair[1], pipeTransportPair[0]
		}
		r.mapRouterPipeTransports[o.Router] = pipeTransportPair

		pipeTransportPair[0].OnClose(func(ctx context.Context) {
			pipeTransportPair[1].CloseContext(ctx)
			r.routerPipeMu.Lock()
			delete(r.mapRouterPipeTransports, o.Router)
			r.routerPipeMu.Unlock()
		})
	}
	r.routerPipeMu.Unlock()

	localPipeTransport, remotePipeTransport := pipeTransportPair[0], pipeTransportPair[1]

	if producer != nil {
		pipeConsumer, err := localPipeTransport.ConsumeContext(ctx, &ConsumerOptions{
			ProducerId: producer.Id(),
		})
		if err != nil {
			return nil, err
		}
		pipeProducer, err := remotePipeTransport.ProduceContext(ctx, &ProducerOptions{
			Id:            producer.Id(),
			Kind:          pipeConsumer.Kind(),
			RtpParameters: pipeConsumer.RtpParameters(),
			Paused:        pipeConsumer.ProducerPaused(),
			AppData:       producer.AppData(),
		})
		if err != nil {
			pipeConsumer.CloseContext(ctx)
			return nil, err
		}

		// Pipe events from the pipe Consumer to the pipe Producer.
		pipeConsumer.OnClose(func(ctx context.Context) {
			pipeProducer.CloseContext(ctx)
		})
		pipeConsumer.OnPause(func(ctx context.Context) {
			pipeProducer.PauseContext(ctx)
		})
		pipeConsumer.OnResume(func(ctx context.Context) {
			pipeProducer.ResumeContext(ctx)
		})

		// Pipe events from the pipe Producer to the pipe Consumer.
		pipeProducer.OnClose(func(ctx context.Context) {
			pipeConsumer.CloseContext(ctx)
		})

		// Ensure that the producer has not been closed in the meanwhile.
		if producer.Closed() {
			pipeConsumer.CloseContext(ctx)
			pipeProducer.CloseContext(ctx)
			return nil, errors.New("original Producer closed")
		}

		// Ensure that producer.paused has not changed in the meanwhile and, if
		// so, sych the pipeProducer.
		if pipeProducer.Paused() != producer.Paused() {
			if producer.Paused() {
				err = pipeProducer.PauseContext(ctx)
			} else {
				err = pipeProducer.ResumeContext(ctx)
			}
			if err != nil {
				pipeConsumer.CloseContext(ctx)
				pipeProducer.CloseContext(ctx)
				return nil, err
			}
		}

		return &PipeToRouterResult{
			PipeConsumer: pipeConsumer,
			PipeProducer: pipeProducer,
		}, nil
	}

	pipeDataConsumer, err := localPipeTransport.ConsumeDataContext(ctx, &DataConsumerOptions{
		DataProducerId: dataProducer.Id(),
	})
	if err != nil {
		return nil, err
	}
	pipeDataProducer, err := remotePipeTransport.ProduceDataContext(ctx, &DataProducerOptions{
		Id:                   dataProducer.Id(),
		SctpStreamParameters: pipeDataConsumer.SctpStreamParameters(),
		Label:                pipeDataConsumer.Label(),
		Protocol:             pipeDataConsumer.Protocol(),
		AppData:              dataProducer.AppData(),
	})
	if err != nil {
		pipeDataConsumer.CloseContext(ctx)
		return nil, err
	}

	// Pipe events from the pipe DataConsumer to the pipe DataProducer.
	pipeDataConsumer.OnClose(func(ctx context.Context) {
		pipeDataProducer.CloseContext(ctx)
	})
	pipeDataConsumer.OnPause(func(ctx context.Context) {
		pipeDataProducer.PauseContext(ctx)
	})
	pipeDataConsumer.OnResume(func(ctx context.Context) {
		pipeDataProducer.ResumeContext(ctx)
	})

	// Pipe events from the pipe DataProducer to the pipe DataConsumer.
	pipeDataProducer.OnClose(func(ctx context.Context) {
		pipeDataConsumer.CloseContext(ctx)
	})

	// Ensure that the dataProducer has not been closed in the meanwhile.
	if dataProducer.Closed() {
		pipeDataConsumer.CloseContext(ctx)
		pipeDataProducer.CloseContext(ctx)
		return nil, errors.New("original DataProducer closed")
	}

	// Ensure that dataProducer.paused has not changed in the meanwhile and, if
	// so, sych the pipeDataProducer.
	if pipeDataProducer.Paused() != dataProducer.Paused() {
		if dataProducer.Paused() {
			err = pipeDataProducer.PauseContext(ctx)
		} else {
			err = pipeDataProducer.ResumeContext(ctx)
		}
		if err != nil {
			pipeDataConsumer.CloseContext(ctx)
			pipeDataProducer.CloseContext(ctx)
			return nil, err
		}
	}

	return &PipeToRouterResult{
		PipeDataConsumer: pipeDataConsumer,
		PipeDataProducer: pipeDataProducer,
	}, nil
}

func (r *Router) OnNewRtpObserver(listener func(context.Context, *RtpObserver)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.newRtpObserverListeners = append(r.newRtpObserverListeners, listener)
}

func (r *Router) OnNewTransport(listener func(context.Context, *Transport)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.newTransportListeners = append(r.newTransportListeners, listener)
}

func (r *Router) workerClosed(ctx context.Context) {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return
	}
	r.closed = true
	r.mu.Unlock()
	r.logger.DebugContext(ctx, "workerClosed()")

	r.cleanupAfterClosed(ctx)
}

func (r *Router) cleanupAfterClosed(ctx context.Context) {
	var children []interface{ routerClosed(ctx context.Context) }

	r.transports.Range(func(key, value any) bool {
		children = append(children, value.(*Transport))
		r.transports.Delete(key)
		return true
	})
	r.rtpObservers.Range(func(key, value any) bool {
		children = append(children, value.(*RtpObserver))
		r.transports.Delete(key)
		return true
	})

	clearSyncMap(&r.producers)
	clearSyncMap(&r.consumers)
	clearSyncMap(&r.dataProducers)
	clearSyncMap(&r.dataConsumers)

	for _, child := range children {
		child.routerClosed(ctx)
	}
	r.notifyClosed(ctx)
}

func (r *Router) newRtpObserver(ctx context.Context, data *rtpObserverData) (*RtpObserver, error) {
	r.mu.Lock()

	if r.closed {
		r.mu.Unlock()
		return nil, ErrRouterClosed
	}

	rtpObserver := newRtpObserver(r.channel, r.logger, data)
	r.rtpObservers.Store(rtpObserver.Id(), rtpObserver)
	rtpObserver.OnClose(func(ctx context.Context) {
		r.rtpObservers.Delete(rtpObserver.Id())
	})

	listeners := r.newRtpObserverListeners

	r.mu.Unlock()

	for _, listener := range listeners {
		listener(ctx, rtpObserver)
	}

	return rtpObserver, nil
}

func (r *Router) newTransport(ctx context.Context, data *internalTransportData) (*Transport, error) {
	r.mu.Lock()

	if r.closed {
		r.mu.Unlock()
		return nil, ErrRouterClosed
	}

	data.RouterId = r.Id()
	data.GetProducerId = r.GetProducerById
	data.GetDataProducerId = r.GetDataProducerById
	data.GetRouterRtpCapabilities = r.RtpCapabilities
	data.OnAddProducer = func(p *Producer) {
		r.producers.Store(p.Id(), p)
	}
	data.OnAddConsumer = func(c *Consumer) {
		r.consumers.Store(c.Id(), c)
	}
	data.OnAddDataProducer = func(p *DataProducer) {
		r.dataProducers.Store(p.Id(), p)
	}
	data.OnAddDataConsumer = func(c *DataConsumer) {
		r.dataConsumers.Store(c.Id(), c)
	}
	data.OnRemoveProducer = func(p *Producer) {
		r.producers.Delete(p.Id())
	}
	data.OnRemoveConsumer = func(c *Consumer) {
		r.consumers.Delete(c.Id())
	}
	data.OnRemoveDataProducer = func(p *DataProducer) {
		r.dataProducers.Delete(p.Id())
	}
	data.OnRemoveDataConsumer = func(c *DataConsumer) {
		r.dataConsumers.Delete(c.Id())
	}

	transport := newTransport(r.channel, r.logger, data)
	r.transports.Store(transport.Id(), transport)
	transport.OnClose(func(ctx context.Context) {
		r.transports.Delete(transport.Id())
	})

	listeners := r.newTransportListeners

	r.mu.Unlock()

	for _, listener := range listeners {
		listener(ctx, transport)
	}

	return transport, nil
}
