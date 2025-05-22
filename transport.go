package mediasoup

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	FbsCommon "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/Common"
	FbsConsumer "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/Consumer"
	FbsDataProducer "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/DataProducer"
	FbsDirectTransport "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/DirectTransport"
	FbsNotification "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/Notification"
	FbsPipeTransport "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/PipeTransport"
	FbsPlainTransport "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/PlainTransport"
	FbsRequest "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/Request"
	FbsRouter "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/Router"
	FbsRtpParameters "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/RtpParameters"
	FbsSctpParameters "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/SctpParameters"
	FbsTransport "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/Transport"
	FbsWebRtcTransport "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/WebRtcTransport"

	"github.com/jiyeyuran/mediasoup-go/v2/internal/channel"
)

type internalTransportData struct {
	*TransportData

	RouterId      string
	TransportId   string
	TransportType TransportType
	AppData       H

	// plain transport specific data
	RtcpMux bool
	Comedia bool

	// pipe transport specific data
	Rtx bool

	// additional functions
	GetProducerId            func(id string) *Producer
	GetDataProducerId        func(id string) *DataProducer
	GetRouterRtpCapabilities func() *RtpCapabilities
	OnAddProducer            func(producer *Producer)
	OnAddDataProducer        func(dataProducer *DataProducer)
	OnAddConsumer            func(consumer *Consumer)
	OnAddDataConsumer        func(dataConsumer *DataConsumer)
	OnRemoveProducer         func(producer *Producer)
	OnRemoveDataProducer     func(dataProducer *DataProducer)
	OnRemoveConsumer         func(consumer *Consumer)
	OnRemoveDataConsumer     func(dataConsumer *DataConsumer)
}

type Transport struct {
	baseListener

	channel *channel.Channel
	sub     *channel.Subscription
	data    *internalTransportData
	logger  *slog.Logger
	closed  bool

	// cname for producers
	cname     string
	cnameOnce sync.Once

	producers     sync.Map
	consumers     sync.Map
	dataProducers sync.Map
	dataConsumers sync.Map

	streamIds sync.Map
	nextMid   uint32

	// event handlers
	newConsumerListeners            []func(context.Context, *Consumer)
	newProducerListeners            []func(context.Context, *Producer)
	newDataConsumerListeners        []func(context.Context, *DataConsumer)
	newDataProducerListeners        []func(context.Context, *DataProducer)
	tupleListeners                  []func(TransportTuple)
	rtcpTupleListeners              []func(TransportTuple)
	sctpStateChangeListeners        []func(SctpState)
	iceStateChangeListeners         []func(IceState)
	iceSelectedTupleChangeListeners []func(TransportTuple)
	dtlsStateChangeListeners        []func(DtlsState)
	rtcpListeners                   []func([]byte)
	traceListeners                  []func(*TransportTraceEventData)
}

func newTransport(channel *channel.Channel, logger *slog.Logger, data *internalTransportData) *Transport {
	t := &Transport{
		channel: channel,
		data:    data,
		logger:  logger.With("transportId", data.TransportId, "transportType", data.TransportType),
	}
	t.handleWorkerNotifications()
	return t
}

func (t *Transport) Id() string {
	return t.data.TransportId
}

func (t *Transport) Type() TransportType {
	return t.data.TransportType
}

func (t *Transport) Data() TransportData {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return *t.data.clone()
}

func (t *Transport) AppData() H {
	return t.data.AppData
}

func (t *Transport) Closed() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.closed
}

func (t *Transport) Close() error {
	return t.CloseContext(context.Background())
}

func (t *Transport) CloseContext(ctx context.Context) error {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return nil
	}
	t.logger.DebugContext(ctx, "Close()")

	_, err := t.channel.Request(ctx, &FbsRequest.RequestT{
		Method:    FbsRequest.MethodROUTER_CLOSE_TRANSPORT,
		HandlerId: t.data.RouterId,
		Body: &FbsRequest.BodyT{
			Type: FbsRequest.BodyRouter_CloseTransportRequest,
			Value: &FbsRouter.CloseTransportRequestT{
				TransportId: t.Id(),
			},
		},
	})
	if err != nil {
		t.mu.Unlock()
		return err
	}
	t.closed = true
	t.mu.Unlock()

	t.cleanupAfterClosed(ctx)
	return nil
}

// Dump transport.
func (t *Transport) Dump() (*TransportDump, error) {
	return t.DumpContext(context.Background())
}

func (t *Transport) DumpContext(ctx context.Context) (*TransportDump, error) {
	t.logger.DebugContext(ctx, "Dump()")

	msg, err := t.channel.Request(ctx, &FbsRequest.RequestT{
		Method:    FbsRequest.MethodTRANSPORT_DUMP,
		HandlerId: t.Id(),
	})
	if err != nil {
		return nil, err
	}

	parseBaseTransportDump := func(dump *FbsTransport.DumpT) BaseTransportDump {
		return BaseTransportDump{
			Id:          dump.Id,
			Type:        t.Type(),
			Direct:      dump.Direct,
			ProducerIds: dump.ProducerIds,
			ConsumerIds: dump.ConsumerIds,
			MapSsrcConsumerId: collect(dump.MapSsrcConsumerId,
				func(item *FbsCommon.Uint32StringT) KeyValue[uint32, string] {
					return KeyValue[uint32, string]{
						Key:   item.Key,
						Value: item.Value,
					}
				}),
			MapRtxSsrcConsumerId: collect(dump.MapRtxSsrcConsumerId,
				func(item *FbsCommon.Uint32StringT) KeyValue[uint32, string] {
					return KeyValue[uint32, string]{
						Key:   item.Key,
						Value: item.Value,
					}
				}),
			DataProducerIds: dump.DataProducerIds,
			DataConsumerIds: dump.DataConsumerIds,
			RecvRtpHeaderExtensions: ifElse(dump.RecvRtpHeaderExtensions != nil, func() *RecvRtpHeaderExtensions {
				return &RecvRtpHeaderExtensions{
					Mid:               dump.RecvRtpHeaderExtensions.Mid,
					Rid:               dump.RecvRtpHeaderExtensions.Rid,
					Rrid:              dump.RecvRtpHeaderExtensions.Rrid,
					AbsSendTime:       dump.RecvRtpHeaderExtensions.AbsSendTime,
					TransportWideCC01: dump.RecvRtpHeaderExtensions.TransportWideCc01,
				}
			}),
			RtpListener: &RtpListenerDump{
				SsrcTable: collect(dump.RtpListener.SsrcTable,
					func(item *FbsCommon.Uint32StringT) KeyValue[uint32, string] {
						return KeyValue[uint32, string]{
							Key:   item.Key,
							Value: item.Value,
						}
					}),
				MidTable: collect(dump.RtpListener.MidTable,
					func(item *FbsCommon.StringStringT) KeyValue[string, string] {
						return KeyValue[string, string]{
							Key:   item.Key,
							Value: item.Value,
						}
					}),
				RidTable: collect(dump.RtpListener.RidTable,
					func(item *FbsCommon.StringStringT) KeyValue[string, string] {
						return KeyValue[string, string]{
							Key:   item.Key,
							Value: item.Value,
						}
					}),
			},
			MaxMessageSize: dump.MaxMessageSize,
			SctpParameters: parseSctpParameters(dump.SctpParameters),
			SctpState: ifElse(dump.SctpState != nil, func() SctpState {
				return SctpState(strings.ToLower(dump.SctpState.String()))
			}),
			SctpListener: ifElse(dump.SctpListener != nil, func() *SctpListener {
				return &SctpListener{
					StreamIdTable: collect(dump.SctpListener.StreamIdTable,
						func(item *FbsCommon.Uint16StringT) KeyValue[uint16, string] {
							return KeyValue[uint16, string]{
								Key:   item.Key,
								Value: item.Value,
							}
						}),
				}
			}),
			TraceEventTypes: collect(dump.TraceEventTypes,
				func(item FbsTransport.TraceEventType) TransportTraceEventType {
					return TransportTraceEventType(strings.ToLower(item.String()))
				}),
		}
	}

	switch t.Type() {
	case TransportWebRTC:
		resp := msg.(*FbsWebRtcTransport.DumpResponseT)
		dump := &TransportDump{
			BaseTransportDump: parseBaseTransportDump(resp.Base),
			WebRtcTransportDump: &WebRtcTransportDump{
				IceRole: strings.ToLower(resp.IceRole.String()),
				IceParameters: IceParameters{
					UsernameFragment: resp.IceParameters.UsernameFragment,
					Password:         resp.IceParameters.Password,
				},
				IceCandidates: collect(resp.IceCandidates,
					func(item *FbsWebRtcTransport.IceCandidateT) IceCandidate {
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
				IceState:         IceState(strings.ToLower(resp.IceState.String())),
				IceSelectedTuple: parseTransportTuple(resp.IceSelectedTuple),
				DtlsParameters: DtlsParameters{
					Role: DtlsRole(strings.ToLower(resp.DtlsParameters.Role.String())),
					Fingerprints: collect(resp.DtlsParameters.Fingerprints,
						func(item *FbsWebRtcTransport.FingerprintT) DtlsFingerprint {
							return DtlsFingerprint{
								Algorithm: strings.ToLower(item.Algorithm.String()),
								Value:     item.Value,
							}
						},
					),
				},
				DtlsState: DtlsState(strings.ToLower(resp.DtlsState.String())),
			},
		}

		return dump, nil

	case TransportPlain:
		resp := msg.(*FbsPlainTransport.DumpResponseT)
		dump := &TransportDump{
			BaseTransportDump: parseBaseTransportDump(resp.Base),
			PlainTransportDump: &PlainTransportDump{
				RtcpMux:        resp.RtcpMux,
				Comedia:        resp.Comedia,
				Tuple:          *parseTransportTuple(resp.Tuple),
				RtcpTuple:      parseTransportTuple(resp.Tuple),
				SrtpParameters: parseSrtpParameters(resp.SrtpParameters),
			},
		}

		return dump, nil

	case TransportPipe:
		resp := msg.(*FbsPipeTransport.DumpResponseT)
		dump := &TransportDump{
			BaseTransportDump: parseBaseTransportDump(resp.Base),
			PipeTransportDump: &PipeTransportDump{
				Tuple:          *parseTransportTuple(resp.Tuple),
				Rtx:            resp.Rtx,
				SrtpParameters: parseSrtpParameters(resp.SrtpParameters),
			},
		}

		return dump, nil

	case TransportDirect:
		resp := msg.(*FbsDirectTransport.DumpResponseT)
		dump := &TransportDump{
			BaseTransportDump: parseBaseTransportDump(resp.Base),
		}

		return dump, nil

	default:
		return nil, fmt.Errorf("unknown transport type: %s", t.Type())
	}
}

// GetStats returns the Transport stats.
func (t *Transport) GetStats() (*TransportStat, error) {
	return t.GetStatsContext(context.Background())
}

func (t *Transport) GetStatsContext(ctx context.Context) (*TransportStat, error) {
	t.logger.DebugContext(ctx, "GetStats()")

	msg, err := t.channel.Request(ctx, &FbsRequest.RequestT{
		Method:    FbsRequest.MethodTRANSPORT_GET_STATS,
		HandlerId: t.Id(),
	})
	if err != nil {
		return nil, err
	}

	parseBaseTransportStat := func(stats *FbsTransport.StatsT) BaseTransportStat {
		return BaseTransportStat{
			Type:        string(t.Type()) + "-transport",
			TransportId: stats.TransportId,
			Timestamp:   stats.Timestamp,
			SctpState: ifElse(stats.SctpState != nil, func() SctpState {
				return SctpState(strings.ToLower(stats.SctpState.String()))
			}),
			BytesReceived:            stats.BytesReceived,
			RecvBitrate:              stats.RecvBitrate,
			BytesSent:                stats.BytesSent,
			SendBitrate:              stats.SendBitrate,
			RtpBytesReceived:         stats.RtpBytesReceived,
			RtpRecvBitrate:           stats.RtpRecvBitrate,
			RtpBytesSent:             stats.RtpBytesSent,
			RtpSendBitrate:           stats.RtpSendBitrate,
			RtxBytesReceived:         stats.RtxBytesReceived,
			RtxRecvBitrate:           stats.RtxRecvBitrate,
			RtxBytesSent:             stats.RtxBytesSent,
			RtxSendBitrate:           stats.RtxSendBitrate,
			ProbationBytesSent:       stats.ProbationBytesSent,
			ProbationSendBitrate:     stats.ProbationSendBitrate,
			AvailableOutgoingBitrate: stats.AvailableOutgoingBitrate,
			AvailableIncomingBitrate: stats.AvailableIncomingBitrate,
			MaxIncomingBitrate:       stats.MaxIncomingBitrate,
			RtpPacketLossReceived:    stats.RtpPacketLossReceived,
			RtpPacketLossSent:        stats.RtpPacketLossSent,
		}
	}

	switch t.Type() {
	case TransportWebRTC:
		resp := msg.(*FbsWebRtcTransport.GetStatsResponseT)

		return &TransportStat{
			BaseTransportStat: parseBaseTransportStat(resp.Base),
			WebRtcTransportStat: &WebRtcTransportStat{
				IceRole:          strings.ToLower(resp.IceRole.String()),
				IceState:         IceState(strings.ToLower(resp.IceState.String())),
				DtlsState:        DtlsState(strings.ToLower(resp.DtlsState.String())),
				IceSelectedTuple: parseTransportTuple(resp.IceSelectedTuple),
			},
		}, nil

	case TransportPlain:
		resp := msg.(*FbsPlainTransport.GetStatsResponseT)
		return &TransportStat{
			BaseTransportStat: parseBaseTransportStat(resp.Base),
			PlainTransportStat: &PlainTransportStat{
				RtcpMux:   resp.RtcpMux,
				Comedia:   resp.Comedia,
				Tuple:     *parseTransportTuple(resp.Tuple),
				RtcpTuple: parseTransportTuple(resp.RtcpTuple),
			},
		}, nil

	case TransportPipe:
		resp := msg.(*FbsPipeTransport.GetStatsResponseT)
		return &TransportStat{
			BaseTransportStat: parseBaseTransportStat(resp.Base),
			PipeTransportStat: &PipeTransportStat{
				Tuple: *parseTransportTuple(resp.Tuple),
			},
		}, nil

	case TransportDirect:
		resp := msg.(*FbsDirectTransport.GetStatsResponseT)
		return &TransportStat{
			BaseTransportStat: parseBaseTransportStat(resp.Base),
		}, nil

	default:
		return nil, fmt.Errorf("unknown transport type: %s", t.Type())
	}
}

// Connect provide the Transport remote parameters.
func (t *Transport) Connect(connectOpts *TransportConnectOptions) error {
	return t.ConnectContext(context.Background(), connectOpts)
}

func (t *Transport) ConnectContext(ctx context.Context, connectOpts *TransportConnectOptions) error {
	t.logger.DebugContext(ctx, "Connect()")

	srtpParameters, err := convertSrtpParameters(connectOpts.SrtpParameters)
	if err != nil {
		return err
	}
	switch t.Type() {
	case TransportWebRTC:
		role := FbsWebRtcTransport.EnumValuesDtlsRole[strings.ToUpper(string(connectOpts.DtlsParameters.Role))]
		fingerprints := collect(connectOpts.DtlsParameters.Fingerprints, convertDtlsFingerprint)

		resp, err := t.channel.Request(ctx, &FbsRequest.RequestT{
			Method:    FbsRequest.MethodWEBRTCTRANSPORT_CONNECT,
			HandlerId: t.Id(),
			Body: &FbsRequest.BodyT{
				Type: FbsRequest.BodyWebRtcTransport_ConnectRequest,
				Value: &FbsWebRtcTransport.ConnectRequestT{
					DtlsParameters: &FbsWebRtcTransport.DtlsParametersT{
						Role:         role,
						Fingerprints: fingerprints,
					},
				},
			},
		})
		if err != nil {
			return err
		}
		result := resp.(*FbsWebRtcTransport.ConnectResponseT)
		// Update data.
		t.data.DtlsParameters.Role = DtlsRole(strings.ToLower(result.DtlsLocalRole.String()))

	case TransportPlain:
		resp, err := t.channel.Request(ctx, &FbsRequest.RequestT{
			Method:    FbsRequest.MethodPLAINTRANSPORT_CONNECT,
			HandlerId: t.Id(),
			Body: &FbsRequest.BodyT{
				Type: FbsRequest.BodyPlainTransport_ConnectRequest,
				Value: &FbsPlainTransport.ConnectRequestT{
					Ip:             connectOpts.Ip,
					Port:           connectOpts.Port,
					RtcpPort:       connectOpts.RtcpPort,
					SrtpParameters: srtpParameters,
				},
			},
		})
		if err != nil {
			return err
		}
		result := resp.(*FbsPlainTransport.ConnectResponseT)
		// Update data.
		data := t.data.PlainTransportData
		data.Tuple = *parseTransportTuple(result.Tuple)
		data.RtcpTuple = parseTransportTuple(result.RtcpTuple)
		data.SrtpParameters = parseSrtpParameters(result.SrtpParameters)

	case TransportPipe:
		resp, err := t.channel.Request(ctx, &FbsRequest.RequestT{
			Method:    FbsRequest.MethodPIPETRANSPORT_CONNECT,
			HandlerId: t.Id(),
			Body: &FbsRequest.BodyT{
				Type: FbsRequest.BodyPipeTransport_ConnectRequest,
				Value: &FbsPipeTransport.ConnectRequestT{
					Ip:             connectOpts.Ip,
					Port:           connectOpts.Port,
					SrtpParameters: srtpParameters,
				},
			},
		})
		if err != nil {
			return err
		}
		result := resp.(*FbsPipeTransport.ConnectResponseT)
		// Update data.
		t.data.PipeTransportData.Tuple = *parseTransportTuple(result.Tuple)
	}

	return nil
}

// RestartIce restarts webrtc transport ICE.
func (t *Transport) RestartIce() (*IceParameters, error) {
	return t.RestartIceContext(context.Background())
}

func (t *Transport) RestartIceContext(ctx context.Context) (*IceParameters, error) {
	if t.Type() != TransportWebRTC {
		return nil, ErrNotImplemented
	}
	t.logger.DebugContext(ctx, "RestartIce()")

	msg, err := t.channel.Request(ctx, &FbsRequest.RequestT{
		Method:    FbsRequest.MethodTRANSPORT_RESTART_ICE,
		HandlerId: t.Id(),
	})
	if err != nil {
		return nil, err
	}
	result := msg.(*FbsTransport.RestartIceResponseT)
	iceParameters := IceParameters{
		UsernameFragment: result.UsernameFragment,
		Password:         result.Password,
		IceLite:          result.IceLite,
	}

	// Update data.
	t.data.IceParameters = iceParameters

	return &iceParameters, nil
}

// SetMaxIncomingBitrate set maximum incoming bitrate for receiving media.
func (t *Transport) SetMaxIncomingBitrate(bitrate uint32) error {
	return t.SetMaxIncomingBitrateContext(context.Background(), bitrate)
}

func (t *Transport) SetMaxIncomingBitrateContext(ctx context.Context, bitrate uint32) error {
	if t.Type() == TransportDirect {
		return ErrNotImplemented
	}
	t.logger.DebugContext(ctx, "SetMaxIncomingBitrate()")

	_, err := t.channel.Request(ctx, &FbsRequest.RequestT{
		Method:    FbsRequest.MethodTRANSPORT_SET_MAX_INCOMING_BITRATE,
		HandlerId: t.Id(),
		Body: &FbsRequest.BodyT{
			Type: FbsRequest.BodyTransport_SetMaxIncomingBitrateRequest,
			Value: &FbsTransport.SetMaxIncomingBitrateRequestT{
				MaxIncomingBitrate: bitrate,
			},
		},
	})
	return err
}

// SendRtcp send RTCP packet.
func (t *Transport) SendRtcp(data []byte) error {
	return t.SendRtcpContext(context.Background(), data)
}

func (t *Transport) SendRtcpContext(ctx context.Context, data []byte) error {
	if t.Type() != TransportDirect {
		return ErrNotImplemented
	}
	t.logger.DebugContext(ctx, "SendRtcp()")

	return t.channel.Notify(ctx, &FbsNotification.NotificationT{
		Event:     FbsNotification.EventTRANSPORT_SEND_RTCP,
		HandlerId: t.Id(),
		Body: &FbsNotification.BodyT{
			Type: FbsNotification.BodyTransport_SendRtcpNotification,
			Value: &FbsTransport.SendRtcpNotificationT{
				Data: data,
			},
		},
	})
}

// EnableTraceEvent enables 'trace' events: "probation", "bwe".
func (t *Transport) EnableTraceEvent(events []TransportTraceEventType) error {
	return t.EnableTraceEventContext(context.Background(), events)
}

func (t *Transport) EnableTraceEventContext(ctx context.Context, events []TransportTraceEventType) error {
	t.logger.DebugContext(ctx, "EnableTraceEvent()")

	events = filter(events, func(typ TransportTraceEventType) bool {
		_, ok := FbsTransport.EnumValuesTraceEventType[strings.ToUpper(string(typ))]
		return ok
	})

	_, err := t.channel.Request(ctx, &FbsRequest.RequestT{
		Method:    FbsRequest.MethodTRANSPORT_ENABLE_TRACE_EVENT,
		HandlerId: t.Id(),
		Body: &FbsRequest.BodyT{
			Type: FbsRequest.BodyTransport_EnableTraceEventRequest,
			Value: &FbsTransport.EnableTraceEventRequestT{
				Events: collect(events, func(item TransportTraceEventType) FbsTransport.TraceEventType {
					return FbsTransport.EnumValuesTraceEventType[strings.ToUpper(string(item))]
				}),
			},
		},
	})
	return err
}

// Produce creates a Producer.
func (t *Transport) Produce(options *ProducerOptions) (*Producer, error) {
	return t.ProduceContext(context.Background(), options)
}

func (t *Transport) ProduceContext(ctx context.Context, options *ProducerOptions) (*Producer, error) {
	t.logger.DebugContext(ctx, "Produce()")

	id := options.Id
	kind := options.Kind
	rtpParameters := clone(options.RtpParameters)

	if len(id) > 0 {
		if t.data.GetProducerId(id) != nil {
			return nil, fmt.Errorf(`a Producer with same id "%s" already exists`, id)
		}
	} else {
		id = UUID(transportPrefix)
	}
	if rtpParameters == nil {
		rtpParameters = &RtpParameters{}
	}
	if rtpParameters.Rtcp == nil {
		rtpParameters.Rtcp = &RtcpParameters{}
	}

	err := validateRtpParameters(rtpParameters)
	if err != nil {
		return nil, fmt.Errorf("validate rtp parameters failed: %w", err)
	}

	// If missing or empty encodings, add one.
	if len(rtpParameters.Encodings) == 0 {
		rtpParameters.Encodings = []*RtpEncodingParameters{{}}
	}

	// Don't do this in PipeTransports since there we must keep CNAME value in each Producer.
	if t.Type() != TransportPipe {
		t.cnameOnce.Do(func() {
			// If CNAME is given and we don't have yet a CNAME for Producers in this Transport, take it.
			if cname := rtpParameters.Rtcp.Cname; len(cname) > 0 {
				t.cname = cname
			} else {
				// Otherwise if we don't have yet a CNAME for Producers and the RTP parameters
				// do not include CNAME, create a random one.
				t.cname = randString(8)
			}
		})
		// Override Producer's CNAME.
		rtpParameters.Rtcp.Cname = t.cname
	}

	routerRtpCapabilities := t.data.GetRouterRtpCapabilities()
	rtpMapping, err := getProducerRtpParametersMapping(rtpParameters, routerRtpCapabilities)
	if err != nil {
		return nil, err
	}

	consumableRtpParameters := getConsumableRtpParameters(
		kind,
		rtpParameters,
		routerRtpCapabilities,
		rtpMapping,
	)

	msg, err := t.channel.Request(ctx, &FbsRequest.RequestT{
		Method:    FbsRequest.MethodTRANSPORT_PRODUCE,
		HandlerId: t.Id(),
		Body: &FbsRequest.BodyT{
			Type: FbsRequest.BodyTransport_ProduceRequest,
			Value: &FbsTransport.ProduceRequestT{
				ProducerId: id,
				Kind:       FbsRtpParameters.EnumValuesMediaKind[strings.ToUpper(string(kind))],
				RtpParameters: &FbsRtpParameters.RtpParametersT{
					Mid: rtpParameters.Mid,
					Codecs: collect(rtpParameters.Codecs,
						func(item *RtpCodecParameters) *FbsRtpParameters.RtpCodecParametersT {
							return &FbsRtpParameters.RtpCodecParametersT{
								MimeType:    item.MimeType,
								PayloadType: item.PayloadType,
								ClockRate:   item.ClockRate,
								Channels:    orElse(item.Channels > 0, ref(item.Channels), nil),
								Parameters:  convertRtpCodecSpecificParameters(&item.Parameters),
								RtcpFeedback: collect(item.RtcpFeedback, func(item *RtcpFeedback) *FbsRtpParameters.RtcpFeedbackT {
									return &FbsRtpParameters.RtcpFeedbackT{
										Type:      item.Type,
										Parameter: item.Parameter,
									}
								}),
							}
						}),
					HeaderExtensions: collect(rtpParameters.HeaderExtensions,
						func(item *RtpHeaderExtensionParameters) *FbsRtpParameters.RtpHeaderExtensionParametersT {
							return &FbsRtpParameters.RtpHeaderExtensionParametersT{
								Uri:        convertHeaderExtensionUri(item.Uri),
								Id:         item.Id,
								Encrypt:    item.Encrypt,
								Parameters: convertRtpCodecSpecificParameters(&item.Parameters),
							}
						}),
					Encodings: collect(rtpParameters.Encodings,
						func(item *RtpEncodingParameters) *FbsRtpParameters.RtpEncodingParametersT {
							return &FbsRtpParameters.RtpEncodingParametersT{
								Ssrc:             orElse(item.Ssrc > 0, ref(item.Ssrc), nil),
								Rid:              item.Rid,
								CodecPayloadType: item.CodecPayloadType,
								Rtx: ifElse(item.Rtx != nil, func() *FbsRtpParameters.RtxT {
									return &FbsRtpParameters.RtxT{
										Ssrc: item.Rtx.Ssrc,
									}
								}),
								Dtx:             item.Dtx,
								ScalabilityMode: item.ScalabilityMode,
								MaxBitrate:      orElse(item.MaxBitrate > 0, ref(item.MaxBitrate), nil),
							}
						}),
					Rtcp: ifElse(rtpParameters.Rtcp != nil, func() *FbsRtpParameters.RtcpParametersT {
						return &FbsRtpParameters.RtcpParametersT{
							Cname:       rtpParameters.Rtcp.Cname,
							ReducedSize: unref(rtpParameters.Rtcp.ReducedSize, true),
						}
					}, func() *FbsRtpParameters.RtcpParametersT {
						return &FbsRtpParameters.RtcpParametersT{
							ReducedSize: true,
						}
					}),
				},
				RtpMapping: &FbsRtpParameters.RtpMappingT{
					Codecs: collect(rtpMapping.Codecs,
						func(item *RtpMappingCodec) *FbsRtpParameters.CodecMappingT {
							return &FbsRtpParameters.CodecMappingT{
								PayloadType:       item.PayloadType,
								MappedPayloadType: item.MappedPayloadType,
							}
						}),
					Encodings: collect(rtpMapping.Encodings,
						func(item *RtpMappingEncoding) *FbsRtpParameters.EncodingMappingT {
							return &FbsRtpParameters.EncodingMappingT{
								Rid:             item.Rid,
								Ssrc:            item.Ssrc,
								ScalabilityMode: item.ScalabilityMode,
								MappedSsrc:      item.MappedSsrc,
							}
						}),
				},
				KeyFrameRequestDelay: options.KeyFrameRequestDelay,
				Paused:               options.Paused,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return nil, ErrTransportClosed
	}
	result := msg.(*FbsTransport.ProduceResponseT)

	producer := newProducer(t.channel, t.logger, &producerData{
		TransportId:             t.Id(),
		ProducerId:              id,
		Kind:                    kind,
		RtpParameters:           rtpParameters,
		Type:                    ProducerType(strings.ToLower(result.Type.String())),
		ConsumableRtpParameters: consumableRtpParameters,
		Paused:                  options.Paused,
		AppData:                 orElse(options.AppData != nil, options.AppData, H{}),
	})

	t.producers.Store(id, producer)
	safeCall(t.data.OnAddProducer, producer)

	producer.OnClose(func(ctx context.Context) {
		t.producers.Delete(id)
		safeCall(t.data.OnRemoveProducer, producer)
	})

	listeners := t.newProducerListeners

	t.mu.Unlock()

	for _, listener := range listeners {
		listener(ctx, producer)
	}

	return producer, nil
}

// Consume creates a Consumer.
func (t *Transport) Consume(options *ConsumerOptions) (*Consumer, error) {
	return t.ConsumeContext(context.Background(), options)
}

func (t *Transport) ConsumeContext(ctx context.Context, options *ConsumerOptions) (*Consumer, error) {
	t.logger.DebugContext(ctx, "Consume()")

	producer := t.data.GetProducerId(options.ProducerId)
	if producer == nil {
		return nil, fmt.Errorf(`Producer with id "%s" not found`, options.ProducerId)
	}
	var rtpParameters *RtpParameters

	if t.Type() == TransportPipe {
		rtpParameters = getPipeConsumerRtpParameters(producer.ConsumableRtpParameters(), t.data.Rtx)
	} else {
		var (
			enableRtx = unref(options.EnableRtx, true)
			err       error
		)
		rtpParameters, err = getConsumerRtpParameters(
			producer.ConsumableRtpParameters(), options.RtpCapabilities, options.Pipe, enableRtx)
		if err != nil {
			return nil, err
		}
		if !options.Pipe {
			if len(options.Mid) > 0 {
				rtpParameters.Mid = options.Mid
			} else {
				rtpParameters.Mid = t.nextMidString()
			}
		}
	}

	consumerId := UUID(consumerPrefix)
	typ := orElse(options.Pipe || t.Type() == TransportPipe, ConsumerPipe, ConsumerType(producer.Type()))

	msg, err := t.channel.Request(ctx, &FbsRequest.RequestT{
		Method:    FbsRequest.MethodTRANSPORT_CONSUME,
		HandlerId: t.Id(),
		Body: &FbsRequest.BodyT{
			Type: FbsRequest.BodyTransport_ConsumeRequest,
			Value: &FbsTransport.ConsumeRequestT{
				ConsumerId: consumerId,
				ProducerId: producer.Id(),
				Kind:       FbsRtpParameters.EnumValuesMediaKind[strings.ToUpper(string(producer.Kind()))],
				Type:       FbsRtpParameters.EnumValuesType[strings.ToUpper(string(typ))],
				Paused:     options.Paused,
				RtpParameters: &FbsRtpParameters.RtpParametersT{
					Mid: rtpParameters.Mid,
					Codecs: collect(rtpParameters.Codecs,
						func(item *RtpCodecParameters) *FbsRtpParameters.RtpCodecParametersT {
							return &FbsRtpParameters.RtpCodecParametersT{
								MimeType:    item.MimeType,
								PayloadType: item.PayloadType,
								ClockRate:   item.ClockRate,
								Channels:    orElse(item.Channels > 0, ref(item.Channels), nil),
								Parameters:  convertRtpCodecSpecificParameters(&item.Parameters),
								RtcpFeedback: collect(item.RtcpFeedback,
									func(item *RtcpFeedback) *FbsRtpParameters.RtcpFeedbackT {
										return &FbsRtpParameters.RtcpFeedbackT{
											Type:      item.Type,
											Parameter: item.Parameter,
										}
									}),
							}
						}),
					HeaderExtensions: collect(rtpParameters.HeaderExtensions,
						func(item *RtpHeaderExtensionParameters) *FbsRtpParameters.RtpHeaderExtensionParametersT {
							return &FbsRtpParameters.RtpHeaderExtensionParametersT{
								Uri:        convertHeaderExtensionUri(item.Uri),
								Id:         item.Id,
								Encrypt:    item.Encrypt,
								Parameters: convertRtpCodecSpecificParameters(&item.Parameters),
							}
						}),
					Encodings: collect(rtpParameters.Encodings,
						func(item *RtpEncodingParameters) *FbsRtpParameters.RtpEncodingParametersT {
							return &FbsRtpParameters.RtpEncodingParametersT{
								Ssrc:             orElse(item.Ssrc > 0, ref(item.Ssrc), nil),
								Rid:              item.Rid,
								CodecPayloadType: item.CodecPayloadType,
								Rtx: ifElse(item.Rtx != nil, func() *FbsRtpParameters.RtxT {
									return &FbsRtpParameters.RtxT{
										Ssrc: item.Rtx.Ssrc,
									}
								}),
								Dtx:             item.Dtx,
								ScalabilityMode: item.ScalabilityMode,
							}
						},
					),
					Rtcp: ifElse(rtpParameters.Rtcp != nil, func() *FbsRtpParameters.RtcpParametersT {
						return &FbsRtpParameters.RtcpParametersT{
							Cname:       rtpParameters.Rtcp.Cname,
							ReducedSize: unref(rtpParameters.Rtcp.ReducedSize, true),
						}
					}, func() *FbsRtpParameters.RtcpParametersT {
						return &FbsRtpParameters.RtcpParametersT{
							ReducedSize: true,
						}
					}),
				},
				ConsumableRtpEncodings: collect(producer.ConsumableRtpParameters().Encodings, convertRtpEncodingParameters),
				PreferredLayers: ifElse(options.PreferredLayers != nil, func() *FbsConsumer.ConsumerLayersT {
					return &FbsConsumer.ConsumerLayersT{
						SpatialLayer:  options.PreferredLayers.SpatialLayer,
						TemporalLayer: options.PreferredLayers.TemporalLayer,
					}
				}),
				IgnoreDtx: options.IgnoreDtx,
			},
		},
	})
	if err != nil {
		return nil, err
	}
	result := msg.(*FbsTransport.ConsumeResponseT)
	score := ConsumerScore{}
	if t.Type() == TransportPipe {
		score = ConsumerScore{
			Score:          10,
			ProducerScore:  10,
			ProducerScores: []int{},
		}
	} else {
		score = ConsumerScore{
			Score:         int(result.Score.Score),
			ProducerScore: int(result.Score.ProducerScore),
			ProducerScores: collect(result.Score.ProducerScores, func(v byte) int {
				return int(v)
			}),
		}
	}
	data := &consumerData{
		TransportId:    t.Id(),
		ConsumerId:     consumerId,
		ProducerId:     producer.Id(),
		Kind:           producer.Kind(),
		Type:           typ,
		RtpParameters:  rtpParameters,
		Paused:         result.Paused,
		ProducerPaused: result.ProducerPaused,
		Score:          score,
		PreferredLayers: ifElse(result.PreferredLayers != nil, func() *ConsumerLayers {
			return &ConsumerLayers{
				SpatialLayer:  result.PreferredLayers.SpatialLayer,
				TemporalLayer: result.PreferredLayers.TemporalLayer,
			}
		}),
		AppData: orElse(options.AppData != nil, options.AppData, H{}),
	}

	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return nil, ErrTransportClosed
	}

	consumer := newConsumer(t.channel, t.logger, data)

	t.consumers.Store(consumer.Id(), consumer)
	safeCall(t.data.OnAddConsumer, consumer)

	consumer.OnClose(func(ctx context.Context) {
		t.consumers.Delete(consumer.Id())
		safeCall(t.data.OnRemoveConsumer, consumer)
	})

	// producer state may change before consumer created.
	if producer.Closed() {
		consumer.CloseContext(ctx)
		return nil, errors.New("original Producer closed")
	}
	// sync producer state.
	consumer.syncProducer(producer)

	listeners := t.newConsumerListeners

	t.mu.Unlock()

	for _, listener := range listeners {
		listener(ctx, consumer)
	}

	return consumer, nil
}

// ProduceData creates a DataProducer.
func (t *Transport) ProduceData(options *DataProducerOptions) (*DataProducer, error) {
	return t.ProduceDataContext(context.Background(), options)
}

func (t *Transport) ProduceDataContext(ctx context.Context, options *DataProducerOptions) (*DataProducer, error) {
	t.logger.DebugContext(ctx, "ProduceData()")

	id := options.Id

	if len(id) > 0 {
		if t.data.GetDataProducerId(id) != nil {
			return nil, fmt.Errorf(`a DataProducer with same id "%s" already exists`, id)
		}
	} else {
		id = UUID(dataProducerPrefix)
	}

	sctpStreamParameters := clone(options.SctpStreamParameters)
	typ := DataProducerDirect

	if t.Type() != TransportDirect {
		typ = DataProducerSctp
		if err := validateSctpStreamParameters(sctpStreamParameters); err != nil {
			return nil, err
		}
	}
	data := &dataProducerData{
		TransportId:          t.Id(),
		DataProducerId:       id,
		Type:                 typ,
		SctpStreamParameters: sctpStreamParameters,
		Label:                options.Label,
		Protocol:             options.Protocol,
		Paused:               options.Paused,
		AppData:              orElse(options.AppData != nil, options.AppData, H{}),
	}
	_, err := t.channel.Request(ctx, &FbsRequest.RequestT{
		Method:    FbsRequest.MethodTRANSPORT_PRODUCE_DATA,
		HandlerId: t.Id(),
		Body: &FbsRequest.BodyT{
			Type: FbsRequest.BodyTransport_ProduceDataRequest,
			Value: &FbsTransport.ProduceDataRequestT{
				DataProducerId: data.DataProducerId,
				Type:           FbsDataProducer.EnumValuesType[strings.ToUpper(string(data.Type))],
				SctpStreamParameters: ifElse(sctpStreamParameters != nil, func() *FbsSctpParameters.SctpStreamParametersT {
					return &FbsSctpParameters.SctpStreamParametersT{
						StreamId:          sctpStreamParameters.StreamId,
						Ordered:           sctpStreamParameters.Ordered,
						MaxPacketLifeTime: sctpStreamParameters.MaxPacketLifeTime,
						MaxRetransmits:    sctpStreamParameters.MaxRetransmits,
					}
				}),
				Label:    data.Label,
				Protocol: data.Protocol,
				Paused:   options.Paused,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return nil, ErrTransportClosed
	}

	dataProducer := newDataProducer(t.channel, t.logger, data)

	t.dataProducers.Store(dataProducer.Id(), dataProducer)
	safeCall(t.data.OnAddDataProducer, dataProducer)

	dataProducer.OnClose(func(ctx context.Context) {
		t.dataProducers.Delete(dataProducer.Id())
		safeCall(t.data.OnRemoveDataProducer, dataProducer)
	})

	listeners := t.newDataProducerListeners

	t.mu.Unlock()

	for _, listener := range listeners {
		listener(ctx, dataProducer)
	}

	return dataProducer, nil
}

// ConsumeData creates a DataConsumer.
func (t *Transport) ConsumeData(options *DataConsumerOptions) (*DataConsumer, error) {
	return t.ConsumeDataContext(context.Background(), options)
}

func (t *Transport) ConsumeDataContext(ctx context.Context, options *DataConsumerOptions) (*DataConsumer, error) {
	t.logger.DebugContext(ctx, "ConsumeData()")

	dataProducer := t.data.GetDataProducerId(options.DataProducerId)
	if dataProducer == nil {
		return nil, fmt.Errorf(`DataProducer with id "%s" not found`, options.DataProducerId)
	}
	var (
		typ                  = DataConsumerDirect
		sctpStreamParameters *SctpStreamParameters
		err                  error
	)

	if t.Type() != TransportDirect {
		typ = DataConsumerSctp
		sctpStreamId, err := t.getNextSctpStreamId()
		if err != nil {
			return nil, err
		}
		if sctpStreamParameters = dataProducer.SctpStreamParameters(); sctpStreamParameters == nil {
			sctpStreamParameters = &SctpStreamParameters{
				StreamId: sctpStreamId,
				Ordered:  ref(true),
			}
		} else {
			sctpStreamParameters.StreamId = sctpStreamId
		}
		// Override if given.
		if ordered := options.Ordered; ordered != nil {
			sctpStreamParameters.Ordered = ordered
		}
		if maxPacketLifeTime := options.MaxPacketLifeTime; maxPacketLifeTime > 0 {
			sctpStreamParameters.MaxPacketLifeTime = &maxPacketLifeTime
		}
		if maxRetransmits := options.MaxRetransmits; maxRetransmits > 0 {
			sctpStreamParameters.MaxRetransmits = &maxRetransmits
		}
	}

	dataConsumerId := UUID(dataConsumerPrefix)

	_, err = t.channel.Request(ctx, &FbsRequest.RequestT{
		Method:    FbsRequest.MethodTRANSPORT_CONSUME_DATA,
		HandlerId: t.Id(),
		Body: &FbsRequest.BodyT{
			Type: FbsRequest.BodyTransport_ConsumeDataRequest,
			Value: &FbsTransport.ConsumeDataRequestT{
				DataConsumerId: dataConsumerId,
				DataProducerId: dataProducer.Id(),
				Type:           FbsDataProducer.EnumValuesType[strings.ToUpper(string(typ))],
				SctpStreamParameters: ifElse(sctpStreamParameters != nil, func() *FbsSctpParameters.SctpStreamParametersT {
					return &FbsSctpParameters.SctpStreamParametersT{
						StreamId:          sctpStreamParameters.StreamId,
						Ordered:           sctpStreamParameters.Ordered,
						MaxPacketLifeTime: sctpStreamParameters.MaxPacketLifeTime,
						MaxRetransmits:    sctpStreamParameters.MaxRetransmits,
					}
				}),
				Label:       dataProducer.Label(),
				Protocol:    dataProducer.Protocol(),
				Paused:      options.Paused,
				Subchannels: options.Subchannels,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return nil, ErrTransportClosed
	}

	data := &dataconsumerData{
		TransportId:          t.Id(),
		DataConsumerId:       dataConsumerId,
		DataProducerId:       options.DataProducerId,
		Type:                 typ,
		SctpStreamParameters: sctpStreamParameters,
		Label:                dataProducer.Label(),
		Protocol:             dataProducer.Protocol(),
		Paused:               options.Paused,
		DataProducerPaused:   dataProducer.Paused(),
		Subchannels:          options.Subchannels,
		AppData:              orElse(options.AppData != nil, options.AppData, H{}),
	}
	dataConsumer := newDataConsumer(t.channel, t.logger, data)

	t.dataConsumers.Store(dataConsumer.Id(), dataConsumer)
	safeCall(t.data.OnAddDataConsumer, dataConsumer)

	dataConsumer.OnClose(func(ctx context.Context) {
		if sctpStreamParameters != nil {
			t.streamIds.Delete(sctpStreamParameters.StreamId)
		}
		t.dataConsumers.Delete(dataConsumer.Id())
		safeCall(t.data.OnRemoveDataConsumer, dataConsumer)
	})

	// dataProducer state may change before dataConsumer created.
	if dataProducer.Closed() {
		dataConsumer.CloseContext(ctx)
		return nil, errors.New("original DataProducer closed")
	}
	// sync dataProducer state.
	dataConsumer.syncDataProducer(dataProducer)

	listeners := t.newDataConsumerListeners

	t.mu.Unlock()

	for _, listener := range listeners {
		listener(ctx, dataConsumer)
	}

	return dataConsumer, nil
}

func (t *Transport) OnNewConsumer(listener func(context.Context, *Consumer)) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	t.newConsumerListeners = append(t.newConsumerListeners, listener)
}

func (t *Transport) OnNewProducer(listener func(context.Context, *Producer)) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	t.newProducerListeners = append(t.newProducerListeners, listener)
}

func (t *Transport) OnNewDataProducer(listener func(context.Context, *DataProducer)) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	t.newDataProducerListeners = append(t.newDataProducerListeners, listener)
}

func (t *Transport) OnNewDataConsumer(listener func(context.Context, *DataConsumer)) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	t.newDataConsumerListeners = append(t.newDataConsumerListeners, listener)
}

// OnTuple add listener on "tuple" event
func (t *Transport) OnTuple(listener func(TransportTuple)) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.tupleListeners = append(t.tupleListeners, listener)
}

// OnRtcpTuple add listener on "rtcptuple" event
func (t *Transport) OnRtcpTuple(listener func(TransportTuple)) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.rtcpTupleListeners = append(t.rtcpTupleListeners, listener)
}

// OnSctpStateChange add listener on "sctpstatechange" event
func (t *Transport) OnSctpStateChange(listener func(SctpState)) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.sctpStateChangeListeners = append(t.sctpStateChangeListeners, listener)
}

// OnIceStateChange add listener on "icestatechange" event
func (t *Transport) OnIceStateChange(listener func(IceState)) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.iceStateChangeListeners = append(t.iceStateChangeListeners, listener)
}

// OnIceSelectedTupleChange add listener on "iceselectedtuplechange" event
func (t *Transport) OnIceSelectedTupleChange(listener func(TransportTuple)) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.iceSelectedTupleChangeListeners = append(t.iceSelectedTupleChangeListeners, listener)
}

// OnDtlsStateChange add listener on "dtlsstatechange" event
func (t *Transport) OnDtlsStateChange(listener func(DtlsState)) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.dtlsStateChangeListeners = append(t.dtlsStateChangeListeners, listener)
}

// OnRtcp add listener on "directtransport.rtcp" event
func (t *Transport) OnRtcp(listener func(data []byte)) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.rtcpListeners = append(t.rtcpListeners, listener)
}

// OnTrace add listener on "trace" event
func (t *Transport) OnTrace(listener func(trace *TransportTraceEventData)) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.traceListeners = append(t.traceListeners, listener)
}

func (t *Transport) handleWorkerNotifications() {
	t.sub = t.channel.Subscribe(t.Id(), func(ctx context.Context, notification *FbsNotification.NotificationT) {
		switch event, body := notification.Event, notification.Body; event {
		case FbsNotification.EventPLAINTRANSPORT_TUPLE:
			notification := body.Value.(*FbsPlainTransport.TupleNotificationT)
			tuple := *parseTransportTuple(notification.Tuple)

			t.mu.Lock()
			t.data.PlainTransportData.Tuple = tuple
			listeners := t.tupleListeners
			t.mu.Unlock()

			for _, listener := range listeners {
				listener(tuple)
			}

		case FbsNotification.EventPLAINTRANSPORT_RTCP_TUPLE:
			notification := body.Value.(*FbsPlainTransport.RtcpTupleNotificationT)
			rtcpTuple := *parseTransportTuple(notification.Tuple)

			t.mu.Lock()
			t.data.PlainTransportData.RtcpTuple = &rtcpTuple
			listeners := t.tupleListeners
			t.mu.Unlock()

			for _, listener := range listeners {
				listener(rtcpTuple)
			}

		case FbsNotification.EventTRANSPORT_SCTP_STATE_CHANGE:
			notification := body.Value.(*FbsTransport.SctpStateChangeNotificationT)
			state := SctpState(strings.ToLower(notification.SctpState.String()))

			t.mu.Lock()
			switch t.Type() {
			case TransportPlain:
				t.data.PlainTransportData.SctpState = state
			case TransportPipe:
				t.data.PipeTransportData.SctpState = state
			case TransportWebRTC:
				t.data.WebRtcTransportData.SctpState = state
			}
			listeners := t.sctpStateChangeListeners
			t.mu.Unlock()

			for _, listener := range listeners {
				listener(state)
			}

		case FbsNotification.EventWEBRTCTRANSPORT_ICE_STATE_CHANGE:
			notification := body.Value.(*FbsWebRtcTransport.IceStateChangeNotificationT)
			state := IceState(strings.ToLower(notification.IceState.String()))

			t.mu.Lock()
			t.data.WebRtcTransportData.IceState = state
			listeners := t.iceStateChangeListeners
			t.mu.Unlock()

			for _, listener := range listeners {
				listener(state)
			}

		case FbsNotification.EventWEBRTCTRANSPORT_ICE_SELECTED_TUPLE_CHANGE:
			notification := body.Value.(*FbsWebRtcTransport.IceSelectedTupleChangeNotificationT)
			tuple := *parseTransportTuple(notification.Tuple)

			t.mu.Lock()
			t.data.WebRtcTransportData.IceSelectedTuple = &tuple
			listeners := t.iceSelectedTupleChangeListeners
			t.mu.Unlock()

			for _, listener := range listeners {
				listener(tuple)
			}

		case FbsNotification.EventWEBRTCTRANSPORT_DTLS_STATE_CHANGE:
			notification := body.Value.(*FbsWebRtcTransport.DtlsStateChangeNotificationT)
			state := DtlsState(strings.ToLower(notification.DtlsState.String()))

			t.mu.Lock()
			t.data.WebRtcTransportData.DtlsState = state
			listeners := t.dtlsStateChangeListeners
			t.mu.Unlock()

			for _, listener := range listeners {
				listener(state)
			}

		case FbsNotification.EventDIRECTTRANSPORT_RTCP:
			notification := body.Value.(*FbsDirectTransport.RtcpNotificationT)
			rtcpPacket := notification.Data

			t.mu.RLock()
			listeners := t.rtcpListeners
			t.mu.RUnlock()

			for _, listener := range listeners {
				listener(rtcpPacket)
			}

		case FbsNotification.EventTRANSPORT_TRACE:
			notification := body.Value.(*FbsTransport.TraceNotificationT)
			trace := &TransportTraceEventData{
				Type:      TransportTraceEventType(strings.ToLower(notification.Type.String())),
				Timestamp: notification.Timestamp,
				Direction: orElse(notification.Direction == FbsCommon.TraceDirectionDIRECTION_IN, "in", "out"),
			}
			if notification.Info.Type == FbsTransport.TraceInfoBweTraceInfo {
				info := notification.Info.Value.(*FbsTransport.BweTraceInfoT)
				trace.Info = H{
					"bweType":                 strings.ToLower(info.BweType.String()),
					"desiredBitrate":          info.DesiredBitrate,
					"effectiveDesiredBitrate": info.EffectiveDesiredBitrate,
					"minBitrate":              info.MinBitrate,
					"maxBitrate":              info.MaxBitrate,
					"startBitrate":            info.StartBitrate,
					"maxPaddingBitrate":       info.MaxPaddingBitrate,
					"availableBitrate":        info.AvailableBitrate,
				}
			}

			t.mu.RLock()
			listeners := t.traceListeners
			t.mu.RUnlock()

			for _, listener := range listeners {
				listener(trace)
			}

		default:
			t.logger.Warn("ignoring unknown event in channel listener", "event", event)
		}
	})
}

func (t *Transport) nextMidString() string {
	for {
		val := atomic.AddUint32(&t.nextMid, 1) - 1
		// We use up to 8 bytes for MID (string).
		if val < 100_000_000 {
			return strconv.Itoa(int(val))
		}
		// Reset to 0 only if nextMid == val+1 (CAS ensures we don't race)
		if atomic.CompareAndSwapUint32(&t.nextMid, val+1, 0) {
			return "0"
		}
		// CAS failed — try again
	}
}

func (t *Transport) getNextSctpStreamId() (uint16, error) {
	sctpParameters := t.getSctpParameters()
	if sctpParameters == nil || sctpParameters.MIS == 0 {
		return 0, errors.New("missing sctpParameters.MIS")
	}
	for i := uint16(0); i < sctpParameters.MIS; i++ {
		if _, ok := t.streamIds.LoadOrStore(i, struct{}{}); !ok {
			return i, nil
		}
	}
	return 0, errors.New("no sctpStreamId available")
}

func (t *Transport) getSctpParameters() *SctpParameters {
	if t.data.WebRtcTransportData != nil {
		return t.data.WebRtcTransportData.SctpParameters
	}
	if t.data.PlainTransportData != nil {
		return t.data.PlainTransportData.SctpParameters
	}
	if t.data.PipeTransportData != nil {
		return t.data.PipeTransportData.SctpParameters
	}
	return nil
}

func (t *Transport) routerClosed(ctx context.Context) {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return
	}
	t.closed = true
	t.mu.Unlock()
	t.logger.DebugContext(ctx, "routerClosed()")

	t.cleanupAfterClosed(ctx)
}

func (t *Transport) listenServerClosed(ctx context.Context) {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return
	}
	t.closed = true
	t.mu.Unlock()

	t.cleanupAfterClosed(ctx)
}

func (t *Transport) cleanupAfterClosed(ctx context.Context) {
	var children []interface{ transportClosed(ctx context.Context) }

	t.producers.Range(func(key, value any) bool {
		children = append(children, value.(*Producer))
		t.producers.Delete(key)
		return true
	})
	t.consumers.Range(func(key, value any) bool {
		children = append(children, value.(*Consumer))
		t.consumers.Delete(key)
		return true
	})
	t.dataProducers.Range(func(key, value any) bool {
		children = append(children, value.(*DataProducer))
		t.dataProducers.Delete(key)
		return true
	})
	t.dataConsumers.Range(func(key, value any) bool {
		children = append(children, value.(*DataConsumer))
		t.dataConsumers.Delete(key)
		return true
	})

	for _, child := range children {
		child.transportClosed(ctx)
	}

	t.sub.Unsubscribe()
	t.notifyClosed(ctx)
}
