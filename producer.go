package mediasoup

import (
	"context"
	"log/slog"
	"strings"

	FbsCommon "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/Common"
	FbsNotification "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/Notification"
	FbsProducer "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/Producer"
	FbsRequest "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/Request"
	FbsRtpParameters "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/RtpParameters"
	FbsTransport "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/Transport"

	"github.com/jiyeyuran/mediasoup-go/v2/internal/channel"
)

type producerData struct {
	TransportId             string
	ProducerId              string
	Kind                    MediaKind
	Type                    ProducerType
	RtpParameters           *RtpParameters
	ConsumableRtpParameters *RtpParameters
	AppData                 H
	// changable fields
	Paused bool
}

type Producer struct {
	baseListener

	channel                         *channel.Channel
	logger                          *slog.Logger
	data                            *producerData
	score                           []ProducerScore
	closed                          bool
	pauseListeners                  []func(context.Context)
	resumeListeners                 []func(context.Context)
	transportCloseListeners         []func(context.Context)
	scoreListeners                  []func([]ProducerScore)
	videoOrientationChangeListeners []func(ProducerVideoOrientation)
	traceListeners                  []func(ProducerTraceEventData)
	sub                             *channel.Subscription
}

func newProducer(channel *channel.Channel, logger *slog.Logger, data *producerData) *Producer {
	p := &Producer{
		channel: channel,
		data:    data,
		logger:  logger.With("producerId", data.ProducerId),
	}
	p.handleWorkerNotifications()
	return p
}

// Id returns producer id.
func (p *Producer) Id() string {
	return p.data.ProducerId
}

// Kind returns media kind.
func (p *Producer) Kind() MediaKind {
	return p.data.Kind
}

// RtpParameters returns RTP parameters.
func (p *Producer) RtpParameters() *RtpParameters {
	return p.data.RtpParameters
}

// Type returns producer type.
func (p *Producer) Type() ProducerType {
	return p.data.Type
}

// ConsumableRtpParameters returns consumable RTP parameters.
func (p *Producer) ConsumableRtpParameters() *RtpParameters {
	return p.data.ConsumableRtpParameters
}

// Paused returns whether the Producer is paused.
func (p *Producer) Paused() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.data.Paused
}

// Score returns producer score list.
func (p *Producer) Score() []ProducerScore {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.score
}

// AppData returns app custom data.
func (p *Producer) AppData() H {
	return p.data.AppData
}

// Closed returns whether the Producer is closed.
func (p *Producer) Closed() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.closed
}

// Close the producer.
func (p *Producer) Close() error {
	return p.CloseContext(context.Background())
}

func (p *Producer) CloseContext(ctx context.Context) error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.logger.DebugContext(ctx, "Close()")

	_, err := p.channel.Request(ctx, &FbsRequest.RequestT{
		Method:    FbsRequest.MethodTRANSPORT_CLOSE_PRODUCER,
		HandlerId: p.data.TransportId,
		Body: &FbsRequest.BodyT{
			Type: FbsRequest.BodyTransport_CloseProducerRequest,
			Value: &FbsTransport.CloseProducerRequestT{
				ProducerId: p.Id(),
			},
		},
	})
	if err != nil {
		p.mu.Unlock()
		return err
	}
	p.closed = true
	p.mu.Unlock()

	p.cleanupAfterClosed(ctx)
	return nil
}

// Dump producer.
func (p *Producer) Dump() (*ProducerDump, error) {
	return p.DumpContext(context.Background())
}

func (p *Producer) DumpContext(ctx context.Context) (*ProducerDump, error) {
	p.logger.DebugContext(ctx, "Dump()")

	msg, err := p.channel.Request(ctx, &FbsRequest.RequestT{
		HandlerId: p.Id(),
		Method:    FbsRequest.MethodPRODUCER_DUMP,
	})
	if err != nil {
		return nil, err
	}
	resp := msg.(*FbsProducer.DumpResponseT)
	dump := &ProducerDump{
		Id:            resp.Id,
		Kind:          MediaKind(strings.ToLower(resp.Kind.String())),
		Type:          ProducerType(strings.ToLower(resp.Type.String())),
		RtpParameters: parseRtpParameters(resp.RtpParameters),
		RtpMapping: &RtpMapping{
			Codecs: collect(resp.RtpMapping.Codecs, func(codec *FbsRtpParameters.CodecMappingT) *RtpMappingCodec {
				return &RtpMappingCodec{
					PayloadType:       codec.PayloadType,
					MappedPayloadType: codec.MappedPayloadType,
				}
			}),
			Encodings: collect(resp.RtpMapping.Encodings, func(encoding *FbsRtpParameters.EncodingMappingT) *RtpMappingEncoding {
				return &RtpMappingEncoding{
					Rid:             encoding.Rid,
					Ssrc:            encoding.Ssrc,
					ScalabilityMode: encoding.ScalabilityMode,
					MappedSsrc:      encoding.MappedSsrc,
				}
			}),
		},
		RtpStreams: collect(resp.RtpStreams, parseRtpStreamDump),
		TraceEventTypes: collect(resp.TraceEventTypes, func(typ FbsProducer.TraceEventType) ProducerTraceEventType {
			return ProducerTraceEventType(strings.ToLower(typ.String()))
		}),
		Paused: resp.Paused,
	}
	return dump, nil
}

// GetStats returns producer stats.
func (p *Producer) GetStats() ([]*ProducerStat, error) {
	return p.GetStatsContext(context.Background())
}

func (p *Producer) GetStatsContext(ctx context.Context) ([]*ProducerStat, error) {
	p.logger.DebugContext(ctx, "GetStats()")

	msg, err := p.channel.Request(ctx, &FbsRequest.RequestT{
		Method:    FbsRequest.MethodPRODUCER_GET_STATS,
		HandlerId: p.Id(),
	})
	if err != nil {
		return nil, err
	}
	resp := msg.(*FbsProducer.GetStatsResponseT)

	return collect(resp.Stats, parseRtpStreamStats), nil
}

// Pause the producer.
func (p *Producer) Pause() error {
	return p.PauseContext(context.Background())
}

func (p *Producer) PauseContext(ctx context.Context) error {
	p.logger.DebugContext(ctx, "Pause()")

	p.mu.Lock()

	_, err := p.channel.Request(ctx, &FbsRequest.RequestT{
		Method:    FbsRequest.MethodPRODUCER_PAUSE,
		HandlerId: p.Id(),
	})
	if err != nil {
		p.mu.Unlock()
		return err
	}
	wasPaused := p.data.Paused
	listeners := p.pauseListeners
	p.data.Paused = true
	p.mu.Unlock()

	if !wasPaused {
		for _, listener := range listeners {
			listener(ctx)
		}
	}

	return err
}

// Resume the producer.
func (p *Producer) Resume() error {
	return p.ResumeContext(context.Background())
}

func (p *Producer) ResumeContext(ctx context.Context) error {
	p.logger.DebugContext(ctx, "Resume()")

	p.mu.Lock()

	_, err := p.channel.Request(ctx, &FbsRequest.RequestT{
		Method:    FbsRequest.MethodPRODUCER_RESUME,
		HandlerId: p.Id(),
	})
	if err != nil {
		p.mu.Unlock()
		return err
	}
	wasPaused := p.data.Paused
	listeners := p.resumeListeners
	p.data.Paused = false

	p.mu.Unlock()

	if wasPaused {
		for _, listener := range listeners {
			listener(ctx)
		}
	}

	return err
}

// EnableTraceEvent enable "trace" event.
func (p *Producer) EnableTraceEvent(events []ProducerTraceEventType) error {
	return p.EnableTraceEventContext(context.Background(), events)
}

func (p *Producer) EnableTraceEventContext(ctx context.Context, events []ProducerTraceEventType) error {
	p.logger.DebugContext(ctx, "EnableTraceEvent()")

	events = filter(events, func(typ ProducerTraceEventType) bool {
		_, ok := FbsProducer.EnumValuesTraceEventType[strings.ToUpper(string(typ))]
		return ok
	})

	_, err := p.channel.Request(ctx, &FbsRequest.RequestT{
		Method:    FbsRequest.MethodPRODUCER_ENABLE_TRACE_EVENT,
		HandlerId: p.Id(),
		Body: &FbsRequest.BodyT{
			Type: FbsRequest.BodyProducer_EnableTraceEventRequest,
			Value: &FbsProducer.EnableTraceEventRequestT{
				Events: collect(events, func(typ ProducerTraceEventType) FbsProducer.TraceEventType {
					return FbsProducer.EnumValuesTraceEventType[strings.ToUpper(string(typ))]
				}),
			},
		},
	})
	return err
}

// Send RTP packet (just valid for Producers created on a DirectTransport).
func (p *Producer) Send(rtpPacket []byte) error {
	return p.SendContext(context.Background(), rtpPacket)
}

func (p *Producer) SendContext(ctx context.Context, rtpPacket []byte) error {
	p.logger.DebugContext(ctx, "Send()")

	return p.channel.Notify(ctx, &FbsNotification.NotificationT{
		Event:     FbsNotification.EventPRODUCER_SEND,
		HandlerId: p.Id(),
		Body: &FbsNotification.BodyT{
			Type: FbsNotification.BodyProducer_SendNotification,
			Value: &FbsProducer.SendNotificationT{
				Data: rtpPacket,
			},
		},
	})
}

// OnPause add listener on "pause" event.
func (p *Producer) OnPause(listener func(ctx context.Context)) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.pauseListeners = append(p.pauseListeners, listener)
}

// OnResume add listener on "resume" event.
func (p *Producer) OnResume(listener func(ctx context.Context)) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.resumeListeners = append(p.resumeListeners, listener)
}

// OnTransportClosed add listener on "transportclosed" event.
func (p *Producer) OnTransportClosed(listener func(ctx context.Context)) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.transportCloseListeners = append(p.transportCloseListeners, listener)
}

// OnScore add listener on "score" event
func (p *Producer) OnScore(listener func(score []ProducerScore)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.scoreListeners = append(p.scoreListeners, listener)
}

// OnVideoOrientationChange add listener on "videoorientationchange" event
func (p *Producer) OnVideoOrientationChange(listener func(videoOrientation ProducerVideoOrientation)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.videoOrientationChangeListeners = append(p.videoOrientationChangeListeners, listener)
}

// OnTrace add listener on "trace" event
func (p *Producer) OnTrace(listener func(trace ProducerTraceEventData)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.traceListeners = append(p.traceListeners, listener)
}

func (p *Producer) handleWorkerNotifications() {
	p.sub = p.channel.Subscribe(p.Id(), func(ctx context.Context, notification *FbsNotification.NotificationT) {
		switch event, body := notification.Event, notification.Body; event {
		case FbsNotification.EventPRODUCER_SCORE:
			Notification := body.Value.(*FbsProducer.ScoreNotificationT)
			scores := collect(Notification.Scores, func(score *FbsProducer.ScoreT) ProducerScore {
				return ProducerScore{
					EncodingIdx: score.EncodingIdx,
					Rid:         score.Rid,
					Ssrc:        score.Ssrc,
					Score:       score.Score,
				}
			})

			p.mu.Lock()
			p.score = scores
			listeners := p.scoreListeners
			p.mu.Unlock()

			for _, listener := range listeners {
				listener(scores)
			}

		case FbsNotification.EventPRODUCER_VIDEO_ORIENTATION_CHANGE:
			notification := body.Value.(*FbsProducer.VideoOrientationChangeNotificationT)
			videoOrientation := ProducerVideoOrientation{
				Camera:   notification.Camera,
				Flip:     notification.Flip,
				Rotation: notification.Rotation,
			}

			p.mu.RLock()
			listeners := p.videoOrientationChangeListeners
			p.mu.RUnlock()

			for _, listener := range listeners {
				listener(videoOrientation)
			}

		case FbsNotification.EventPRODUCER_TRACE:
			notification := body.Value.(*FbsProducer.TraceNotificationT)
			trace := ProducerTraceEventData{
				Type:      ProducerTraceEventType(strings.ToLower(notification.Type.String())),
				Timestamp: notification.Timestamp,
				Direction: orElse(notification.Direction == FbsCommon.TraceDirectionDIRECTION_IN, "in", "out"),
				Info:      parseProducerTraceInfo(notification.Info),
			}

			p.mu.RLock()
			listeners := p.traceListeners
			p.mu.RUnlock()

			for _, listener := range listeners {
				listener(trace)
			}

		default:
			p.logger.Warn("ignoring unknown event", "event", event.String())
		}
	})
}

// transportClosed is called when transport closed.
func (p *Producer) transportClosed(ctx context.Context) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.closed = true
	listeners := p.transportCloseListeners
	p.mu.Unlock()
	p.logger.DebugContext(ctx, "transportClosed()")

	for _, listener := range listeners {
		listener(ctx)
	}

	p.cleanupAfterClosed(ctx)
}

func (p *Producer) cleanupAfterClosed(ctx context.Context) {
	p.sub.Unsubscribe()
	p.notifyClosed(ctx)
}

func (p *Producer) syncState(ctx context.Context, other *Producer) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.data.Paused != other.Paused() {
		if p.data.Paused {
			return other.PauseContext(ctx)
		} else {
			return other.ResumeContext(ctx)
		}
	}

	return nil
}

func parseProducerTraceInfo(info *FbsProducer.TraceInfoT) any {
	switch info.Type {
	case FbsProducer.TraceInfoRtpTraceInfo:
		value := info.Value.(*FbsProducer.RtpTraceInfoT)
		return &RtpTraceInfo{
			RtpPacket: ref(RtpPacketDump(*value.RtpPacket)),
			IsRtx:     value.IsRtx,
		}

	case FbsProducer.TraceInfoKeyFrameTraceInfo:
		value := info.Value.(*FbsProducer.KeyFrameTraceInfoT)
		return &KeyFrameTraceInfo{
			RtpPacket: ref(RtpPacketDump(*value.RtpPacket)),
			IsRtx:     value.IsRtx,
		}

	case FbsProducer.TraceInfoFirTraceInfo:
		value := info.Value.(*FbsProducer.FirTraceInfoT)
		return &FirTraceInfo{
			Ssrc: value.Ssrc,
		}

	case FbsProducer.TraceInfoPliTraceInfo:
		value := info.Value.(*FbsProducer.PliTraceInfoT)
		return &PliTraceInfo{
			Ssrc: value.Ssrc,
		}

	case FbsProducer.TraceInfoSrTraceInfo:
		value := info.Value.(*FbsProducer.SrTraceInfoT)
		return ref(*value)

	default:
		return nil
	}
}
