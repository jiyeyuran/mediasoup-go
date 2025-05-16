package mediasoup

import (
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

	routerPipeMu            sync.Mutex
	mapRouterPipeTransports map[*Router][2]*Transport
}

func newRouter(channel *channel.Channel, logger *slog.Logger, data *routerData) *Router {
	return &Router{
		channel:                 channel,
		data:                    data,
		logger:                  logger,
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
	r.logger.Debug("Close()")

	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil
	}
	r.logger.Debug("Close()")

	_, err := r.channel.Request(&FbsRequest.RequestT{
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

	r.cleanupAfterClosed()
	return nil
}

func (r *Router) Closed() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.closed
}

func (r *Router) Dump() (dump *RouterDump, err error) {
	r.logger.Debug("Dump()")

	msg, err := r.channel.Request(&FbsRequest.RequestT{
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

func (r *Router) CreateActiveSpeakerObserver(options ...ActiveSpeakerObserverOption) (*RtpObserver, error) {
	r.logger.Debug("CreateActiveSpeakerObserver()")

	o := &ActiveSpeakerObserverOptions{
		Interval: 300,
	}
	for _, option := range options {
		option(o)
	}

	rtpObserverId := uuid()

	_, err := r.channel.Request(&FbsRequest.RequestT{
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

	return r.newRtpObserver(&rtpObserverData{
		Id:              rtpObserverId,
		RouterId:        r.Id(),
		AppData:         o.AppData,
		GetProducerById: r.GetProducerById,
	})
}

func (r *Router) CreateAudioLevelObserver(options ...AudioLevelObserverOption) (*RtpObserver, error) {
	r.logger.Debug("CreateAudioLevelObserver()")

	o := &AudioLevelObserverOptions{
		MaxEntries: 1,
		Threshold:  -80,
		Interval:   1000,
	}
	for _, option := range options {
		option(o)
	}

	rtpObserverId := uuid()

	_, err := r.channel.Request(&FbsRequest.RequestT{
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

	return r.newRtpObserver(&rtpObserverData{
		Id:              rtpObserverId,
		RouterId:        r.Id(),
		AppData:         o.AppData,
		GetProducerById: r.GetProducerById,
	})
}

func (r *Router) CreateWebRtcTransport(options *WebRtcTransportOptions) (*Transport, error) {
	r.logger.Debug("CreateWebRtcTransport()")

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
		AppData:                         options.AppData,
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

	transportId := uuid()
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

	msg, err := r.channel.Request(&FbsRequest.RequestT{
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
	transport, err := r.newTransport(data)
	if err != nil {
		return nil, err
	}
	if o.WebRtcServer != nil {
		o.WebRtcServer.handleWebRtcTransport(transport)
	}
	return transport, nil
}

func (r *Router) CreatePlainTransport(options *PlainTransportOptions) (*Transport, error) {
	r.logger.Debug("CreatePlainTransport()")

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
		AppData:            options.AppData,
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

	transportId := uuid()
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

	msg, err := r.channel.Request(&FbsRequest.RequestT{
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
	return r.newTransport(data)
}

func (r *Router) CreatePipeTransport(options *PipeTransportOptions) (*Transport, error) {
	r.logger.Debug("CreatePipeTransport()")

	o := &PipeTransportOptions{
		ListenInfo:         options.ListenInfo,
		EnableSctp:         options.EnableSctp,
		NumSctpStreams:     &NumSctpStreams{OS: 1024, MIS: 1024},
		MaxSctpMessageSize: 268435456,
		SctpSendBufferSize: 268435456,
		EnableSrtp:         options.EnableSrtp,
		EnableRtx:          options.EnableRtx,
		AppData:            options.AppData,
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

	transportId := uuid()
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

	msg, err := r.channel.Request(&FbsRequest.RequestT{
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
	return r.newTransport(data)
}

func (r *Router) CreateDirectTransport(options *DirectTransportOptions) (*Transport, error) {
	r.logger.Debug("CreateDirectTransport()")

	o := &DirectTransportOptions{
		MaxMessageSize: 262144,
	}
	if options != nil {
		if options.MaxMessageSize > 0 {
			o.MaxMessageSize = options.MaxMessageSize
		}
		o.AppData = options.AppData
	}

	transportId := uuid()
	baseTranportOptions := &FbsTransport.OptionsT{
		Direct:         true,
		MaxMessageSize: &o.MaxMessageSize,
	}

	_, err := r.channel.Request(&FbsRequest.RequestT{
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
	return r.newTransport(data)
}

// PipeToRouter pipes the given Producer or DataProducer into another Router in same host.
func (r *Router) PipeToRouter(options *PipeToRouterOptions) (result *PipeToRouterResult, err error) {
	r.logger.Debug("PipeToRouter()")

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
						localPipeTransport.Close()
					}
					if remotePipeTransport != nil {
						remotePipeTransport.Close()
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
				localPipeTransport, err = r.CreatePipeTransport(options)
				return err
			})
			errgroup.Go(func() error {
				remotePipeTransport, err = o.Router.CreatePipeTransport(options)
				return err
			})
			if err = errgroup.Wait(); err != nil {
				return
			}
			errgroup.Go(func() error {
				data := remotePipeTransport.Data().PipeTransportData
				return localPipeTransport.Connect(&TransportConnectOptions{
					Ip:             data.Tuple.LocalAddress,
					Port:           ref(data.Tuple.LocalPort),
					SrtpParameters: data.SrtpParameters,
				})
			})
			errgroup.Go(func() error {
				data := localPipeTransport.Data().PipeTransportData
				return remotePipeTransport.Connect(&TransportConnectOptions{
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

		pipeTransportPair[0].OnClose(func() {
			pipeTransportPair[1].Close()
			r.routerPipeMu.Lock()
			delete(r.mapRouterPipeTransports, o.Router)
			r.routerPipeMu.Unlock()
		})
	}
	r.routerPipeMu.Unlock()

	localPipeTransport, remotePipeTransport := pipeTransportPair[0], pipeTransportPair[1]

	if producer != nil {
		pipeConsumer, err := localPipeTransport.Consume(&ConsumerOptions{
			ProducerId: o.ProducerId,
		})
		if err != nil {
			return nil, err
		}
		pipeProducer, err := remotePipeTransport.Produce(&ProducerOptions{
			Id:            producer.Id(),
			Kind:          pipeConsumer.Kind(),
			RtpParameters: pipeConsumer.RtpParameters(),
			Paused:        pipeConsumer.ProducerPaused(),
			AppData:       producer.AppData(),
		})
		if err != nil {
			pipeConsumer.Close()
			return nil, err
		}
		// Ensure that the producer has not been closed in the meanwhile.
		if producer.Closed() {
			pipeConsumer.Close()
			pipeProducer.Close()
			return nil, errors.New("original Producer closed")
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
				pipeConsumer.Close()
				pipeProducer.Close()
				return nil, err
			}
		}

		// Pipe events from the pipe Consumer to the pipe Producer.
		pipeConsumer.OnClose(func() {
			pipeProducer.Close()
		})
		pipeConsumer.OnPause(func() {
			pipeProducer.Pause()
		})
		pipeConsumer.OnResume(func() {
			pipeProducer.Resume()
		})

		// Pipe events from the pipe Producer to the pipe Consumer.
		pipeProducer.OnClose(func() {
			pipeConsumer.Close()
		})

		return &PipeToRouterResult{
			PipeConsumer: pipeConsumer,
			PipeProducer: pipeProducer,
		}, nil
	}

	pipeDataConsumer, err := localPipeTransport.ConsumeData(&DataConsumerOptions{
		DataProducerId: o.DataProducerId,
	})
	if err != nil {
		return nil, err
	}
	pipeDataProducer, err := remotePipeTransport.ProduceData(&DataProducerOptions{
		Id:                   dataProducer.Id(),
		SctpStreamParameters: pipeDataConsumer.SctpStreamParameters(),
		Label:                pipeDataConsumer.Label(),
		Protocol:             pipeDataConsumer.Protocol(),
		AppData:              dataProducer.AppData(),
	})
	if err != nil {
		pipeDataConsumer.Close()
		return nil, err
	}
	// Ensure that the dataProducer has not been closed in the meanwhile.
	if dataProducer.Closed() {
		pipeDataConsumer.Close()
		pipeDataProducer.Close()
		return nil, errors.New("original DataProducer closed")
	}

	// Pipe events from the pipe DataConsumer to the pipe DataProducer.
	pipeDataConsumer.OnClose(func() {
		pipeDataProducer.Close()
	})
	// Pipe events from the pipe DataProducer to the pipe DataConsumer.
	pipeDataProducer.OnClose(func() {
		pipeDataConsumer.Close()
	})

	return &PipeToRouterResult{
		PipeDataConsumer: pipeDataConsumer,
		PipeDataProducer: pipeDataProducer,
	}, nil
}

func (r *Router) workerClosed() {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return
	}
	r.closed = true
	r.mu.Unlock()

	r.cleanupAfterClosed()
}

func (r *Router) cleanupAfterClosed() {
	var children []interface{ routerClosed() }

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
		child.routerClosed()
	}
	r.notifyClosed()
}

func (r *Router) newRtpObserver(data *rtpObserverData) (*RtpObserver, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return nil, ErrRouterClosed
	}

	rtpObserver := newRtpObserver(r.channel, r.logger, data)
	r.rtpObservers.Store(rtpObserver.Id(), rtpObserver)
	rtpObserver.OnClose(func() {
		r.rtpObservers.Delete(rtpObserver.Id())
	})

	return rtpObserver, nil
}

func (r *Router) newTransport(data *internalTransportData) (*Transport, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
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
	transport.OnClose(func() {
		r.transports.Delete(transport.Id())
	})

	return transport, nil
}
