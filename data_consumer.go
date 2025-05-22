package mediasoup

import (
	"context"
	"log/slog"

	FbsDataConsumer "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/DataConsumer"
	FbsDataProducer "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/DataProducer"
	FbsNotification "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/Notification"
	FbsRequest "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/Request"
	FbsTransport "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/Transport"
	"github.com/jiyeyuran/mediasoup-go/v2/internal/channel"
)

var (
	emptyBytes  = [1]byte{0}
	emptyString = [1]byte{' '}
)

type dataconsumerData struct {
	TransportId          string
	DataConsumerId       string
	DataProducerId       string
	Type                 DataConsumerType
	SctpStreamParameters *SctpStreamParameters
	Label                string
	Protocol             string
	AppData              H

	// changable fields
	Paused             bool
	DataProducerPaused bool
	Subchannels        []uint16
}

type DataConsumer struct {
	baseListener

	channel                     *channel.Channel
	data                        *dataconsumerData
	closed                      bool
	pauseListeners              []func(context.Context)
	resumeListeners             []func(context.Context)
	dataProducerCloseListeners  []func(context.Context)
	dataProducerPauseListeners  []func(context.Context)
	dataProducerResumeListeners []func(context.Context)
	sctpSendBufferFullListeners []func()
	bufferedAmountLowListeners  []func(bufferAmount uint32)
	messageListeners            []func(payload []byte, ppid SctpPayloadType)
	sub                         *channel.Subscription
	logger                      *slog.Logger
}

func newDataConsumer(channel *channel.Channel, logger *slog.Logger, data *dataconsumerData) *DataConsumer {
	c := &DataConsumer{
		channel: channel,
		data:    data,
		logger:  logger.With("dataConsumerId", data.DataConsumerId),
	}
	c.sub = c.handleWorkerNotifications()
	return c
}

// Id returns DataConsumer id.
func (c *DataConsumer) Id() string {
	return c.data.DataConsumerId
}

// DataProducerId returns the associated DataProducer id.
func (c *DataConsumer) DataProducerId() string {
	return c.data.DataProducerId
}

// Type returns DataConsumer type.
func (c *DataConsumer) Type() DataConsumerType {
	return c.data.Type
}

// SctpStreamParameters returns SCTP stream parameters.
func (c *DataConsumer) SctpStreamParameters() *SctpStreamParameters {
	return c.data.SctpStreamParameters
}

// Label returns DataChannel label.
func (c *DataConsumer) Label() string {
	return c.data.Label
}

// Protocol returns DataChannel protocol.
func (c *DataConsumer) Protocol() string {
	return c.data.Protocol
}

// Paused returns whether the DataConsumer is paused.
func (c *DataConsumer) Paused() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.data.Paused
}

// DataProducerPaused returns whether the associated DataProducer is paused.
func (c *DataConsumer) DataProducerPaused() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.data.DataProducerPaused
}

// AppData returns app custom data.
func (c *DataConsumer) AppData() H {
	return c.data.AppData
}

// Closed returns whether the DataConsumer is closed.
func (c *DataConsumer) Closed() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.closed
}

// Close the DataConsumer.
func (c *DataConsumer) Close() error {
	return c.CloseContext(context.Background())
}

func (c *DataConsumer) CloseContext(ctx context.Context) error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.logger.DebugContext(ctx, "Close()")

	_, err := c.channel.Request(ctx, &FbsRequest.RequestT{
		Method:    FbsRequest.MethodTRANSPORT_CLOSE_DATACONSUMER,
		HandlerId: c.data.TransportId,
		Body: &FbsRequest.BodyT{
			Type: FbsRequest.BodyTransport_CloseDataConsumerRequest,
			Value: &FbsTransport.CloseDataConsumerRequestT{
				DataConsumerId: c.Id(),
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

// Dump DataConsumer.
func (c *DataConsumer) Dump() (*DataConsumerDump, error) {
	return c.DumpContext(context.Background())
}

func (c *DataConsumer) DumpContext(ctx context.Context) (*DataConsumerDump, error) {
	c.logger.DebugContext(ctx, "Dump()")

	msg, err := c.channel.Request(ctx, &FbsRequest.RequestT{
		HandlerId: c.Id(),
		Method:    FbsRequest.MethodDATACONSUMER_DUMP,
	})
	if err != nil {
		return nil, err
	}
	resp := msg.(*FbsDataConsumer.DumpResponseT)
	dump := &DataConsumerDump{
		Id:                         resp.Id,
		Subchannels:                resp.Subchannels,
		Paused:                     resp.Paused,
		DataProducerId:             resp.DataProducerId,
		Type:                       orElse(resp.Type == FbsDataProducer.TypeDIRECT, DataConsumerDirect, DataConsumerSctp),
		Label:                      resp.Label,
		Protocol:                   resp.Protocol,
		BufferedAmountLowThreshold: resp.BufferedAmountLowThreshold,
	}

	if resp.SctpStreamParameters != nil {
		dump.SctpStreamParameters = &SctpStreamParameters{
			StreamId:          resp.SctpStreamParameters.StreamId,
			Ordered:           resp.SctpStreamParameters.Ordered,
			MaxPacketLifeTime: resp.SctpStreamParameters.MaxPacketLifeTime,
			MaxRetransmits:    resp.SctpStreamParameters.MaxRetransmits,
		}
	}

	return dump, nil
}

// GetStats returns DataConsumer stats.
func (c *DataConsumer) GetStats() ([]*DataConsumerStat, error) {
	return c.GetStatsContext(context.Background())
}

func (c *DataConsumer) GetStatsContext(ctx context.Context) ([]*DataConsumerStat, error) {
	c.logger.DebugContext(ctx, "GetStats()")

	msg, err := c.channel.Request(ctx, &FbsRequest.RequestT{
		Method:    FbsRequest.MethodDATACONSUMER_GET_STATS,
		HandlerId: c.Id(),
	})
	if err != nil {
		return nil, err
	}
	resp := msg.(*FbsDataConsumer.GetStatsResponseT)

	return []*DataConsumerStat{
		{
			Type:           "data-consumer",
			Timestamp:      resp.Timestamp,
			Label:          resp.Label,
			Protocol:       resp.Protocol,
			MessagesSent:   resp.MessagesSent,
			BytesSent:      resp.BytesSent,
			BufferedAmount: resp.BufferedAmount,
		},
	}, nil
}

// Pause the DataConsumer.
func (c *DataConsumer) Pause() error {
	return c.PauseContext(context.Background())
}

func (c *DataConsumer) PauseContext(ctx context.Context) error {
	c.logger.DebugContext(ctx, "Pause()")

	c.mu.Lock()

	_, err := c.channel.Request(ctx, &FbsRequest.RequestT{
		Method:    FbsRequest.MethodDATACONSUMER_PAUSE,
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

	if !wasPaused && !c.data.DataProducerPaused {
		for _, listener := range listeners {
			listener(ctx)
		}
	}

	return err
}

// Resume the DataConsumer.
func (c *DataConsumer) Resume() error {
	return c.ResumeContext(context.Background())
}

func (c *DataConsumer) ResumeContext(ctx context.Context) error {
	c.logger.DebugContext(ctx, "Resume()")

	c.mu.Lock()

	_, err := c.channel.Request(ctx, &FbsRequest.RequestT{
		Method:    FbsRequest.MethodDATACONSUMER_RESUME,
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

	if wasPaused && !c.data.DataProducerPaused {
		for _, listener := range listeners {
			listener(ctx)
		}
	}

	return err
}

// SetBufferedAmountLowThreshold set buffered amount low threshold.
func (c *DataConsumer) SetBufferedAmountLowThreshold(threshold uint32) error {
	return c.SetBufferedAmountLowThresholdContext(context.Background(), threshold)
}

func (c *DataConsumer) SetBufferedAmountLowThresholdContext(ctx context.Context, threshold uint32) error {
	c.logger.DebugContext(ctx, "SetBufferedAmountLowThreshold()")

	_, err := c.channel.Request(ctx, &FbsRequest.RequestT{
		Method:    FbsRequest.MethodDATACONSUMER_SET_BUFFERED_AMOUNT_LOW_THRESHOLD,
		HandlerId: c.Id(),
		Body: &FbsRequest.BodyT{
			Type: FbsRequest.BodyDataConsumer_SetBufferedAmountLowThresholdRequest,
			Value: &FbsDataConsumer.SetBufferedAmountLowThresholdRequestT{
				Threshold: threshold,
			},
		},
	})
	return err
}

// GetBufferedAmount returns buffered amount size.
func (c *DataConsumer) GetBufferedAmount() (uint32, error) {
	return c.GetBufferedAmountContext(context.Background())
}

func (c *DataConsumer) GetBufferedAmountContext(ctx context.Context) (uint32, error) {
	c.logger.DebugContext(ctx, "GetBufferedAmount()")

	msg, err := c.channel.Request(ctx, &FbsRequest.RequestT{
		Method:    FbsRequest.MethodDATACONSUMER_GET_BUFFERED_AMOUNT,
		HandlerId: c.Id(),
	})
	if err != nil {
		return 0, err
	}
	resp := msg.(*FbsDataConsumer.GetBufferedAmountResponseT)
	return resp.BufferedAmount, nil
}

// Send data.
func (c *DataConsumer) Send(data []byte) (err error) {
	return c.SendContext(context.Background(), data)
}

func (c *DataConsumer) SendContext(ctx context.Context, data []byte) (err error) {
	c.logger.DebugContext(ctx, "Send()")

	/**
	 * +-------------------------------+----------+
	 * | Value                         | SCTP     |
	 * |                               | PPID     |
	 * +-------------------------------+----------+
	 * | WebRTC String                 | 51       |
	 * | WebRTC Binary Partial         | 52       |
	 * | (Deprecated)                  |          |
	 * | WebRTC Binary                 | 53       |
	 * | WebRTC String Partial         | 54       |
	 * | (Deprecated)                  |          |
	 * | WebRTC String Empty           | 56       |
	 * | WebRTC Binary Empty           | 57       |
	 * +-------------------------------+----------+
	 */
	ppid := SctpPayloadWebRTCBinary

	if len(data) == 0 {
		data, ppid = emptyBytes[:], SctpPayloadWebRTCBinaryEmpty
	}

	return c.send(ctx, data, ppid)
}

// SendText send text.
func (c *DataConsumer) SendText(message string) error {
	return c.SendTextContext(context.Background(), message)
}

func (c *DataConsumer) SendTextContext(ctx context.Context, message string) error {
	c.logger.DebugContext(ctx, "SendText()")

	ppid, data := SctpPayloadWebRTCString, []byte(message)

	if len(data) == 0 {
		data, ppid = emptyString[:], SctpPayloadWebRTCBinaryEmpty
	}

	return c.send(ctx, data, ppid)
}

func (c *DataConsumer) send(ctx context.Context, data []byte, ppid SctpPayloadType) error {
	_, err := c.channel.Request(ctx, &FbsRequest.RequestT{
		Method:    FbsRequest.MethodDATACONSUMER_SEND,
		HandlerId: c.Id(),
		Body: &FbsRequest.BodyT{
			Type: FbsRequest.BodyDataConsumer_SendRequest,
			Value: &FbsDataConsumer.SendRequestT{
				Data: data,
				Ppid: uint32(ppid),
			},
		},
	})
	return err
}

func (c *DataConsumer) SetSubchannels(subchannels []uint16) error {
	return c.SetSubchannelsContext(context.Background(), subchannels)
}

func (c *DataConsumer) SetSubchannelsContext(ctx context.Context, subchannels []uint16) error {
	c.logger.DebugContext(ctx, "SetSubchannels()")

	c.mu.Lock()
	defer c.mu.Unlock()

	msg, err := c.channel.Request(ctx, &FbsRequest.RequestT{
		Method:    FbsRequest.MethodDATACONSUMER_SET_SUBCHANNELS,
		HandlerId: c.Id(),
		Body: &FbsRequest.BodyT{
			Type: FbsRequest.BodyDataConsumer_SetSubchannelsRequest,
			Value: &FbsDataConsumer.SetSubchannelsRequestT{
				Subchannels: subchannels,
			},
		},
	})
	if err != nil {
		return err
	}
	resp := msg.(*FbsDataConsumer.SetSubchannelsResponseT)
	c.data.Subchannels = resp.Subchannels
	return nil
}

func (c *DataConsumer) AddSubChannel(subchannel uint16) error {
	return c.AddSubChannelContext(context.Background(), subchannel)
}

func (c *DataConsumer) AddSubChannelContext(ctx context.Context, subchannel uint16) error {
	c.logger.DebugContext(ctx, "AddSubChannel()")

	c.mu.Lock()
	defer c.mu.Unlock()

	msg, err := c.channel.Request(ctx, &FbsRequest.RequestT{
		Method:    FbsRequest.MethodDATACONSUMER_ADD_SUBCHANNEL,
		HandlerId: c.Id(),
		Body: &FbsRequest.BodyT{
			Type: FbsRequest.BodyDataConsumer_AddSubchannelRequest,
			Value: &FbsDataConsumer.AddSubchannelRequestT{
				Subchannel: subchannel,
			},
		},
	})
	if err != nil {
		return err
	}
	resp := msg.(*FbsDataConsumer.AddSubchannelResponseT)
	c.data.Subchannels = resp.Subchannels
	return nil
}

func (c *DataConsumer) RemoveSubChannel(subchannel uint16) error {
	return c.RemoveSubChannelContext(context.Background(), subchannel)
}

func (c *DataConsumer) RemoveSubChannelContext(ctx context.Context, subchannel uint16) error {
	c.logger.DebugContext(ctx, "RemoveSubChannel()")

	c.mu.Lock()
	defer c.mu.Unlock()

	msg, err := c.channel.Request(ctx, &FbsRequest.RequestT{
		Method:    FbsRequest.MethodDATACONSUMER_REMOVE_SUBCHANNEL,
		HandlerId: c.Id(),
		Body: &FbsRequest.BodyT{
			Type: FbsRequest.BodyDataConsumer_RemoveSubchannelRequest,
			Value: &FbsDataConsumer.RemoveSubchannelRequestT{
				Subchannel: subchannel,
			},
		},
	})
	if err != nil {
		return err
	}
	resp := msg.(*FbsDataConsumer.RemoveSubchannelResponseT)
	c.data.Subchannels = resp.Subchannels
	return nil
}

// OnPause add listener on "pause" event.
func (c *DataConsumer) OnPause(listener func(ctx context.Context)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pauseListeners = append(c.pauseListeners, listener)
}

// OnResume add listener on "resume" event.
func (c *DataConsumer) OnResume(listener func(ctx context.Context)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.resumeListeners = append(c.resumeListeners, listener)
}

// OnProducerClose add listener on "dataproducerclose" event.
func (c *DataConsumer) OnDataProducerClose(listener func(ctx context.Context)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.dataProducerCloseListeners = append(c.dataProducerCloseListeners, listener)
}

// OnProducerPause add listener on "dataproducerpause" event.
func (c *DataConsumer) OnDataProducerPause(listener func(ctx context.Context)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.dataProducerPauseListeners = append(c.dataProducerPauseListeners, listener)
}

// OnProducerResume add listener on "dataproducerresume" event.
func (c *DataConsumer) OnDataProducerResume(listener func(ctx context.Context)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.dataProducerResumeListeners = append(c.dataProducerResumeListeners, listener)
}

// OnSctpSendBufferFull add listener on "sctpsendbufferfull" event
func (c *DataConsumer) OnSctpSendBufferFull(listener func()) {
	c.mu.Lock()
	c.sctpSendBufferFullListeners = append(c.sctpSendBufferFullListeners, listener)
	c.mu.Unlock()
}

// OnBufferedAmountLow add listener on "bufferedamountlow" event
func (c *DataConsumer) OnBufferedAmountLow(listener func(bufferAmount uint32)) {
	c.mu.Lock()
	c.bufferedAmountLowListeners = append(c.bufferedAmountLowListeners, listener)
	c.mu.Unlock()
}

// OnMessage add listener on "message" event
func (c *DataConsumer) OnMessage(listener func(payload []byte, ppid SctpPayloadType)) {
	c.mu.Lock()
	c.messageListeners = append(c.messageListeners, listener)
	c.mu.Unlock()
}

func (c *DataConsumer) handleWorkerNotifications() *channel.Subscription {
	return c.channel.Subscribe(c.Id(), func(ctx context.Context, notification *FbsNotification.NotificationT) {
		switch event, body := notification.Event, notification.Body; event {
		case FbsNotification.EventDATACONSUMER_DATAPRODUCER_CLOSE:
			c.mu.RLock()
			dataProducerCloseListeners := c.dataProducerCloseListeners
			c.mu.RUnlock()

			ctx = channel.UnwrapContext(ctx, c.DataProducerId())
			for _, listener := range dataProducerCloseListeners {
				listener(ctx)
			}
			c.dataProducerClosed(ctx)

		case FbsNotification.EventDATACONSUMER_DATAPRODUCER_PAUSE:
			c.mu.Lock()
			if c.data.DataProducerPaused {
				c.mu.Unlock()
				return
			}
			c.data.DataProducerPaused = true
			paused := c.data.Paused
			listeners := c.pauseListeners
			dataProducerPauseListeners := c.dataProducerPauseListeners
			c.mu.Unlock()

			ctx = channel.UnwrapContext(ctx, c.DataProducerId())

			for _, listener := range dataProducerPauseListeners {
				listener(ctx)
			}
			if !paused {
				for _, listener := range listeners {
					listener(ctx)
				}
			}

		case FbsNotification.EventDATACONSUMER_DATAPRODUCER_RESUME:
			c.mu.Lock()
			if !c.data.DataProducerPaused {
				c.mu.Unlock()
				return
			}
			c.data.DataProducerPaused = false
			paused := c.data.Paused
			listeners := c.resumeListeners
			dataProducerResumeListeners := c.dataProducerResumeListeners
			c.mu.Unlock()

			ctx = channel.UnwrapContext(ctx, c.DataProducerId())

			for _, listener := range dataProducerResumeListeners {
				listener(ctx)
			}
			if paused {
				for _, listener := range listeners {
					listener(ctx)
				}
			}

		case FbsNotification.EventDATACONSUMER_SCTP_SENDBUFFER_FULL:
			c.mu.RLock()
			listeners := c.sctpSendBufferFullListeners
			c.mu.RUnlock()

			for _, listener := range listeners {
				listener()
			}

		case FbsNotification.EventDATACONSUMER_BUFFERED_AMOUNT_LOW:
			Notification := body.Value.(*FbsDataConsumer.BufferedAmountLowNotificationT)
			c.mu.RLock()
			listeners := c.bufferedAmountLowListeners
			c.mu.RUnlock()

			for _, listener := range listeners {
				listener(Notification.BufferedAmount)
			}

		case FbsNotification.EventDATACONSUMER_MESSAGE:
			notification := body.Value.(*FbsDataConsumer.MessageNotificationT)
			c.mu.RLock()
			listeners := c.messageListeners
			c.mu.RUnlock()

			for _, listener := range listeners {
				listener(notification.Data, SctpPayloadType(notification.Ppid))
			}

		default:
			c.logger.Warn("ignoring unknown event", "event", event.String())
		}
	})
}

func (c *DataConsumer) dataProducerClosed(ctx context.Context) {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	c.closed = true
	c.mu.Unlock()

	c.cleanupAfterClosed(ctx)
}

func (c *DataConsumer) transportClosed(ctx context.Context) {
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

func (c *DataConsumer) cleanupAfterClosed(ctx context.Context) {
	c.sub.Unsubscribe()
	c.notifyClosed(ctx)
}

func (c *DataConsumer) syncDataProducer(dataProducer *DataProducer) {
	c.mu.Lock()
	c.data.DataProducerPaused = dataProducer.Paused()
	c.mu.Unlock()
}
