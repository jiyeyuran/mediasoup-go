package mediasoup

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	FbsCommon "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/Common"
	FbsConsumer "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/Consumer"
	FbsNotification "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/Notification"
	FbsRequest "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/Request"
	FbsRtpParameters "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/RtpParameters"
	FbsRtpStream "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/RtpStream"
	FbsTransport "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/Transport"
	"github.com/jiyeyuran/mediasoup-go/v2/internal/channel"
)

type consumerData struct {
	TransportId   string
	ConsumerId    string
	ProducerId    string
	Kind          MediaKind
	Type          ConsumerType
	RtpParameters *RtpParameters
	AppData       H

	// changable fields
	Paused          bool
	ProducerPaused  bool
	Score           ConsumerScore
	PreferredLayers *ConsumerLayers
}

type Consumer struct {
	baseListener

	channel                 *channel.Channel
	logger                  *slog.Logger
	data                    *consumerData
	priority                byte
	currentLayers           *ConsumerLayers // Current video layers (just for video with simulcast or SVC).
	closed                  bool
	pauseListeners          []func(ctx context.Context)
	resumeListeners         []func(ctx context.Context)
	producerCloseListeners  []func(ctx context.Context)
	producerPauseListeners  []func(ctx context.Context)
	producerResumeListeners []func(ctx context.Context)
	scoreListeners          []func(score ConsumerScore)
	layersChangeListeners   []func(layers *ConsumerLayers)
	traceListeners          []func(trace ConsumerTraceEventData)
	rtpListeners            []func(data []byte)
	sub                     *channel.Subscription
}

func newConsumer(channel *channel.Channel, logger *slog.Logger, data *consumerData) *Consumer {
	c := &Consumer{
		channel:  channel,
		data:     data,
		logger:   logger.With("consumerId", data.ConsumerId),
		priority: 1,
	}
	c.handleWorkerNotifications()
	return c
}

// Id returns consumer id
func (c *Consumer) Id() string {
	return c.data.ConsumerId
}

// ProducerId returns associated Producer id.
func (c *Consumer) ProducerId() string {
	return c.data.ProducerId
}

// Kind returns media kind.
func (c *Consumer) Kind() MediaKind {
	return c.data.Kind
}

// RtpParameters returns RTP parameters.
func (c *Consumer) RtpParameters() *RtpParameters {
	return c.data.RtpParameters
}

// Type returns consumer type.
func (c *Consumer) Type() ConsumerType {
	return c.data.Type
}

// Paused returns whether the Consumer is paused.
func (c *Consumer) Paused() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.data.Paused
}

// ProducerPaused returns whether the associate Producer is paused.
func (c *Consumer) ProducerPaused() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.data.ProducerPaused
}

// Closed returns whether the Consumer is closed.
func (c *Consumer) Closed() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.closed
}

// Priority returns current priority.
func (c *Consumer) Priority() byte {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.priority
}

// Score returns consumer score with consumer and consumer keys.
func (c *Consumer) Score() ConsumerScore {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.data.Score
}

// PreferredLayers returns preferred video layers.
func (c *Consumer) PreferredLayers() *ConsumerLayers {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.data.PreferredLayers
}

// CurrentLayers returns current video layers.
func (c *Consumer) CurrentLayers() *ConsumerLayers {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.currentLayers
}

// AppData returns app custom data.
func (c *Consumer) AppData() H {
	return c.data.AppData
}

// Close the consumer.
func (c *Consumer) Close() error {
	return c.CloseContext(context.Background())
}

func (c *Consumer) CloseContext(ctx context.Context) error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.logger.DebugContext(ctx, "Close()")

	_, err := c.channel.Request(ctx, &FbsRequest.RequestT{
		Method:    FbsRequest.MethodTRANSPORT_CLOSE_CONSUMER,
		HandlerId: c.data.TransportId,
		Body: &FbsRequest.BodyT{
			Type: FbsRequest.BodyTransport_CloseConsumerRequest,
			Value: &FbsTransport.CloseConsumerRequestT{
				ConsumerId: c.Id(),
			},
		},
	})
	if err != nil {
		c.mu.Unlock()
		return err
	}
	c.closed = true
	c.mu.Unlock()

	c.cleanupAfterClosed(ctx)

	return nil
}

// Dump Consumer.
func (c *Consumer) Dump() (*ConsumerDump, error) {
	return c.DumpContext(context.Background())
}

func (c *Consumer) DumpContext(ctx context.Context) (*ConsumerDump, error) {
	c.logger.DebugContext(ctx, "Dump()")

	msg, err := c.channel.Request(ctx, &FbsRequest.RequestT{
		HandlerId: c.Id(),
		Method:    FbsRequest.MethodCONSUMER_DUMP,
	})
	if err != nil {
		return nil, err
	}
	resp := msg.(*FbsConsumer.DumpResponseT)
	base := resp.Data.Base
	dump := &ConsumerDump{
		BaseConsumerDump: BaseConsumerDump{
			Id:                         base.Id,
			ProducerId:                 base.ProducerId,
			Type:                       ConsumerType(strings.ToLower(base.Type.String())),
			Kind:                       MediaKind(strings.ToLower(base.Kind.String())),
			RtpParameters:              parseRtpParameters(base.RtpParameters),
			ConsumableRtpEncodings:     collect(base.ConsumableRtpEncodings, parseRtpEncodingParameters),
			SupportedCodecPayloadTypes: collect(base.SupportedCodecPayloadTypes, func(v byte) int { return int(v) }),
			TraceEventTypes: collect(base.TraceEventTypes, func(v FbsConsumer.TraceEventType) ConsumerTraceEventType {
				return ConsumerTraceEventType(strings.ToLower(v.String()))
			}),
			Paused:         base.Paused,
			ProducerPaused: base.ProducerPaused,
			Priority:       base.Priority,
		},
	}

	switch data := resp.Data; data.Base.Type {
	case FbsRtpParameters.TypeSIMPLE:
		dump.SimpleConsumerDump = &SimpleConsumerDump{
			RtpStream: parseRtpStreamDump(data.RtpStreams[0]),
		}

	case FbsRtpParameters.TypeSIMULCAST, FbsRtpParameters.TypeSVC:
		specificDump := &SimulcastConsumerDump{
			RtpStream:              parseRtpStreamDump(data.RtpStreams[0]),
			PreferredSpatialLayer:  *data.PreferredSpatialLayer,
			TargetSpatialLayer:     *data.TargetSpatialLayer,
			CurrentSpatialLayer:    *data.CurrentSpatialLayer,
			PreferredTemporalLayer: *data.PreferredTemporalLayer,
			TargetTemporalLayer:    *data.TargetTemporalLayer,
			CurrentTemporalLayer:   *data.CurrentTemporalLayer,
		}
		if data.Base.Type == FbsRtpParameters.TypeSVC {
			dump.SvcConsumerDump = specificDump
		} else {
			dump.SimulcastConsumerDump = specificDump
		}

	case FbsRtpParameters.TypePIPE:
		dump.PipeConsumerDump = &PipeConsumerDump{
			RtpStreams: collect(data.RtpStreams, parseRtpStreamDump),
		}

	default:
		return nil, fmt.Errorf("invalid Consumer type: %s", data.Base.Type)
	}

	return dump, nil
}

// GetStats returns Consumer stats.
func (c *Consumer) GetStats() ([]*ConsumerStat, error) {
	return c.GetStatsContext(context.Background())
}

func (c *Consumer) GetStatsContext(ctx context.Context) ([]*ConsumerStat, error) {
	c.logger.DebugContext(ctx, "GetStats()")

	msg, err := c.channel.Request(ctx, &FbsRequest.RequestT{
		Method:    FbsRequest.MethodCONSUMER_GET_STATS,
		HandlerId: c.Id(),
	})
	if err != nil {
		return nil, err
	}
	resp := msg.(*FbsConsumer.GetStatsResponseT)

	return collect(resp.Stats, parseRtpStreamStats), nil
}

// Pause the Consumer.
func (c *Consumer) Pause() error {
	return c.PauseContext(context.Background())
}

func (c *Consumer) PauseContext(ctx context.Context) error {
	c.logger.DebugContext(ctx, "Pause()")

	c.mu.Lock()

	_, err := c.channel.Request(ctx, &FbsRequest.RequestT{
		Method:    FbsRequest.MethodCONSUMER_PAUSE,
		HandlerId: c.Id(),
	})
	if err != nil {
		c.mu.Unlock()
		return err
	}
	wasPaused := c.data.Paused
	listeners := c.pauseListeners
	c.data.Paused = true
	c.mu.Unlock()

	if !wasPaused {
		for _, listener := range listeners {
			listener(ctx)
		}
	}

	return err
}

// Resume the Consumer.
func (c *Consumer) Resume() error {
	return c.ResumeContext(context.Background())
}

func (c *Consumer) ResumeContext(ctx context.Context) error {
	c.logger.DebugContext(ctx, "Resume()")

	c.mu.Lock()

	_, err := c.channel.Request(ctx, &FbsRequest.RequestT{
		Method:    FbsRequest.MethodCONSUMER_RESUME,
		HandlerId: c.Id(),
	})
	if err != nil {
		c.mu.Unlock()
		return err
	}

	wasPaused := c.data.Paused
	listeners := c.resumeListeners
	c.data.Paused = false

	c.mu.Unlock()

	if wasPaused {
		for _, listener := range listeners {
			listener(ctx)
		}
	}

	return err
}

// SetPreferredLayers set preferred video layers.
func (c *Consumer) SetPreferredLayers(layers ConsumerLayers) error {
	return c.SetPreferredLayersContext(context.Background(), layers)
}

func (c *Consumer) SetPreferredLayersContext(ctx context.Context, layers ConsumerLayers) error {
	c.logger.DebugContext(ctx, "SetPreferredLayers()")

	c.mu.Lock()
	defer c.mu.Unlock()

	msg, err := c.channel.Request(ctx, &FbsRequest.RequestT{
		Method:    FbsRequest.MethodCONSUMER_SET_PREFERRED_LAYERS,
		HandlerId: c.Id(),
		Body: &FbsRequest.BodyT{
			Type: FbsRequest.BodyConsumer_SetPreferredLayersRequest,
			Value: &FbsConsumer.SetPreferredLayersRequestT{
				PreferredLayers: &FbsConsumer.ConsumerLayersT{
					SpatialLayer:  layers.SpatialLayer,
					TemporalLayer: layers.TemporalLayer,
				},
			},
		},
	})
	if err != nil {
		return err
	}
	result := msg.(*FbsConsumer.SetPreferredLayersResponseT)
	if result.PreferredLayers != nil {
		c.data.PreferredLayers = &ConsumerLayers{
			SpatialLayer:  result.PreferredLayers.SpatialLayer,
			TemporalLayer: result.PreferredLayers.TemporalLayer,
		}
	}
	return nil
}

// SetPriority set priority.
func (c *Consumer) SetPriority(priority byte) error {
	return c.SetPriorityContext(context.Background(), priority)
}

func (c *Consumer) SetPriorityContext(ctx context.Context, priority byte) error {
	c.logger.DebugContext(ctx, "SetPriority()")

	c.mu.Lock()
	defer c.mu.Unlock()

	_, err := c.channel.Request(ctx, &FbsRequest.RequestT{
		Method:    FbsRequest.MethodCONSUMER_SET_PRIORITY,
		HandlerId: c.Id(),
		Body: &FbsRequest.BodyT{
			Type: FbsRequest.BodyConsumer_SetPriorityRequest,
			Value: &FbsConsumer.SetPriorityRequestT{
				Priority: priority,
			},
		},
	})
	if err != nil {
		return err
	}
	c.priority = priority
	return nil
}

// UnsetPriority unset priority.
func (c *Consumer) UnsetPriority() error {
	return c.UnsetPriorityContext(context.Background())
}

func (c *Consumer) UnsetPriorityContext(ctx context.Context) error {
	return c.SetPriorityContext(ctx, 1)
}

// RequestKeyFrame request a key frame to the Producer.
func (c *Consumer) RequestKeyFrame() error {
	return c.RequestKeyFrameContext(context.Background())
}

func (c *Consumer) RequestKeyFrameContext(ctx context.Context) error {
	c.logger.DebugContext(ctx, "RequestKeyFrame()")

	_, err := c.channel.Request(ctx, &FbsRequest.RequestT{
		Method:    FbsRequest.MethodCONSUMER_REQUEST_KEY_FRAME,
		HandlerId: c.Id(),
	})
	return err
}

// EnableTraceEvent enable "trace" event.
func (c *Consumer) EnableTraceEvent(events []ConsumerTraceEventType) error {
	return c.EnableTraceEventContext(context.Background(), events)
}

func (c *Consumer) EnableTraceEventContext(ctx context.Context, events []ConsumerTraceEventType) error {
	c.logger.DebugContext(ctx, "EnableTraceEvent()")

	events = filter(events, func(typ ConsumerTraceEventType) bool {
		_, ok := FbsConsumer.EnumValuesTraceEventType[strings.ToUpper(string(typ))]
		return ok
	})

	_, err := c.channel.Request(ctx, &FbsRequest.RequestT{
		Method:    FbsRequest.MethodCONSUMER_ENABLE_TRACE_EVENT,
		HandlerId: c.Id(),
		Body: &FbsRequest.BodyT{
			Type: FbsRequest.BodyConsumer_EnableTraceEventRequest,
			Value: &FbsConsumer.EnableTraceEventRequestT{
				Events: collect(events, func(event ConsumerTraceEventType) FbsConsumer.TraceEventType {
					return FbsConsumer.EnumValuesTraceEventType[strings.ToUpper(string(event))]
				}),
			},
		},
	})

	return err
}

// OnPause add listener on "pause" event.
func (p *Consumer) OnPause(listener func(ctx context.Context)) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.pauseListeners = append(p.pauseListeners, listener)
}

// OnResume add listener on "resume" event.
func (p *Consumer) OnResume(listener func(ctx context.Context)) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.resumeListeners = append(p.resumeListeners, listener)
}

// OnProducerClose add listener on "producerclose" event.
func (p *Consumer) OnProducerClose(listener func(ctx context.Context)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.producerCloseListeners = append(p.producerCloseListeners, listener)
}

// OnProducerPause add listener on "producerpause" event.
func (p *Consumer) OnProducerPause(listener func(ctx context.Context)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.producerPauseListeners = append(p.producerPauseListeners, listener)
}

// OnProducerResume add listener on "producerresume" event.
func (p *Consumer) OnProducerResume(listener func(ctx context.Context)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.producerResumeListeners = append(p.producerResumeListeners, listener)
}

// OnScore add listener on "score" event.
func (p *Consumer) OnScore(listener func(score ConsumerScore)) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.scoreListeners = append(p.scoreListeners, listener)
}

// OnLayersChange add listener on "layerschange" event.
func (p *Consumer) OnLayersChange(listener func(layers *ConsumerLayers)) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.layersChangeListeners = append(p.layersChangeListeners, listener)
}

// OnTrace add listener on "trace" event.
func (p *Consumer) OnTrace(listener func(trace ConsumerTraceEventData)) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.traceListeners = append(p.traceListeners, listener)
}

// OnRtp add listener on "rtp" event.
func (p *Consumer) OnRtp(listener func(data []byte)) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.rtpListeners = append(p.rtpListeners, listener)
}

func (c *Consumer) handleWorkerNotifications() {
	c.sub = c.channel.Subscribe(c.Id(), func(ctx context.Context, notification *FbsNotification.NotificationT) {
		switch event, body := notification.Event, notification.Body; event {
		case FbsNotification.EventCONSUMER_PRODUCER_CLOSE:
			c.mu.Lock()
			if c.closed {
				c.mu.Unlock()
				return
			}
			c.closed = true
			listeners := c.producerCloseListeners
			c.mu.Unlock()

			ctx = channel.UnwrapContext(ctx, c.ProducerId())

			for _, listener := range listeners {
				listener(ctx)
			}
			c.cleanupAfterClosed(ctx)

		case FbsNotification.EventCONSUMER_PRODUCER_PAUSE:
			c.mu.Lock()
			if c.data.ProducerPaused {
				c.mu.Unlock()
				break
			}
			c.data.ProducerPaused = true
			paused := c.data.Paused
			pauseListeners := c.pauseListeners
			producerPauseListeners := c.producerPauseListeners
			c.mu.Unlock()

			ctx = channel.UnwrapContext(ctx, c.ProducerId())

			for _, listener := range producerPauseListeners {
				listener(ctx)
			}
			if !paused {
				for _, listener := range pauseListeners {
					listener(ctx)
				}
			}

		case FbsNotification.EventCONSUMER_PRODUCER_RESUME:
			c.mu.Lock()
			if !c.data.ProducerPaused {
				c.mu.Unlock()
				break
			}
			c.data.ProducerPaused = false
			paused := c.data.Paused
			resumeListeners := c.resumeListeners
			producerResumeListeners := c.producerResumeListeners
			c.mu.Unlock()

			ctx = channel.UnwrapContext(ctx, c.ProducerId())

			for _, listener := range producerResumeListeners {
				listener(ctx)
			}
			if !paused {
				for _, listener := range resumeListeners {
					listener(ctx)
				}
			}

		case FbsNotification.EventCONSUMER_SCORE:
			notification := body.Value.(*FbsConsumer.ScoreNotificationT)
			score := ConsumerScore{
				Score:         int(notification.Score.Score),
				ProducerScore: int(notification.Score.ProducerScore),
				ProducerScores: ifElse(notification.Score.ProducerScores != nil, func() []int {
					return collect(notification.Score.ProducerScores, func(v byte) int {
						return int(v)
					})
				}),
			}

			c.mu.Lock()
			c.data.Score = score
			listeners := c.scoreListeners
			c.mu.Unlock()

			for _, listener := range listeners {
				listener(score)
			}

		case FbsNotification.EventCONSUMER_LAYERS_CHANGE:
			notification := body.Value.(*FbsConsumer.LayersChangeNotificationT)
			var layers *ConsumerLayers
			if notification.Layers != nil {
				layers = &ConsumerLayers{
					SpatialLayer:  notification.Layers.SpatialLayer,
					TemporalLayer: notification.Layers.TemporalLayer,
				}
			}
			c.mu.Lock()
			c.currentLayers = layers
			listeners := c.layersChangeListeners
			c.mu.Unlock()

			for _, listener := range listeners {
				listener(layers)
			}

		case FbsNotification.EventCONSUMER_TRACE:
			notification := body.Value.(*FbsConsumer.TraceNotificationT)
			trace := ConsumerTraceEventData{
				Type:      ConsumerTraceEventType(strings.ToLower(notification.Type.String())),
				Timestamp: notification.Timestamp,
				Direction: orElse(notification.Direction == FbsCommon.TraceDirectionDIRECTION_IN, "in", "out"),
				Info:      parseConsumerTraceInfo(notification.Info),
			}
			c.mu.RLock()
			listeners := c.traceListeners
			c.mu.RUnlock()

			for _, listener := range listeners {
				listener(trace)
			}

		case FbsNotification.EventCONSUMER_RTP:
			notification := body.Value.(*FbsConsumer.RtpNotificationT)
			data := notification.Data
			c.mu.RLock()
			listeners := c.rtpListeners
			c.mu.RUnlock()

			for _, listener := range listeners {
				listener(data)
			}

		default:
			c.logger.Warn("ignoring unknown event in channel listener", "event", event)
		}
	})
}

func (c *Consumer) transportClosed(ctx context.Context) {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	c.closed = true
	c.mu.Unlock()
	c.logger.DebugContext(ctx, "transportClosed()")

	c.cleanupAfterClosed(ctx)
}

func (c *Consumer) cleanupAfterClosed(ctx context.Context) {
	c.sub.Unsubscribe()
	c.notifyClosed(ctx)
}

func (c *Consumer) syncProducer(producer *Producer) {
	c.mu.Lock()
	c.data.ProducerPaused = producer.Paused()
	c.mu.Unlock()
}

func parseRtpStreamDump(stream *FbsRtpStream.DumpT) *RtpStreamDump {
	return &RtpStreamDump{
		Params: RtpStreamParametersDump{
			EncodingIdx:    stream.Params.EncodingIdx,
			Ssrc:           stream.Params.Ssrc,
			PayloadType:    stream.Params.PayloadType,
			MimeType:       stream.Params.MimeType,
			ClockRate:      stream.Params.ClockRate,
			Rid:            stream.Params.Rid,
			Cname:          stream.Params.Cname,
			RtxSsrc:        stream.Params.RtxSsrc,
			RtxPayloadType: stream.Params.RtxPayloadType,
			UseNack:        stream.Params.UseNack,
			UsePli:         stream.Params.UsePli,
			UseFir:         stream.Params.UseFir,
			UseInBandFec:   stream.Params.UseInBandFec,
			UseDtx:         stream.Params.UseDtx,
			SpatialLayers:  stream.Params.SpatialLayers,
			TemporalLayers: stream.Params.TemporalLayers,
		},
		Score: stream.Score,
		RtxStream: &RtxStreamDump{
			Params: ifElse(stream.RtxStream != nil, func() RtxStreamParameters {
				return RtxStreamParameters{
					Ssrc:        stream.RtxStream.Params.Ssrc,
					PayloadType: stream.RtxStream.Params.PayloadType,
					MimeType:    stream.RtxStream.Params.MimeType,
					ClockRate:   stream.RtxStream.Params.ClockRate,
					Rrid:        stream.RtxStream.Params.Rrid,
					Cname:       stream.RtxStream.Params.Cname,
				}
			}),
		},
	}
}

func parseConsumerTraceInfo(info *FbsConsumer.TraceInfoT) any {
	switch info.Type {
	case FbsConsumer.TraceInfoRtpTraceInfo:
		value := info.Value.(*FbsConsumer.RtpTraceInfoT)
		return &RtpTraceInfo{
			RtpPacket: ref(RtpPacketDump(*value.RtpPacket)),
			IsRtx:     value.IsRtx,
		}

	case FbsConsumer.TraceInfoKeyFrameTraceInfo:
		value := info.Value.(*FbsConsumer.KeyFrameTraceInfoT)
		return &KeyFrameTraceInfo{
			RtpPacket: ref(RtpPacketDump(*value.RtpPacket)),
			IsRtx:     value.IsRtx,
		}

	case FbsConsumer.TraceInfoFirTraceInfo:
		value := info.Value.(*FbsConsumer.FirTraceInfoT)
		return &FirTraceInfo{
			Ssrc: value.Ssrc,
		}

	case FbsConsumer.TraceInfoPliTraceInfo:
		value := info.Value.(*FbsConsumer.PliTraceInfoT)
		return &PliTraceInfo{
			Ssrc: value.Ssrc,
		}

	default:
		return nil
	}

}
