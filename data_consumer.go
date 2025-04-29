package mediasoup

import (
	"log/slog"

	FbsDataConsumer "github.com/jiyeyuran/mediasoup-go/internal/FBS/DataConsumer"
	FbsDataProducer "github.com/jiyeyuran/mediasoup-go/internal/FBS/DataProducer"
	FbsNotification "github.com/jiyeyuran/mediasoup-go/internal/FBS/Notification"
	FbsRequest "github.com/jiyeyuran/mediasoup-go/internal/FBS/Request"
	FbsTransport "github.com/jiyeyuran/mediasoup-go/internal/FBS/Transport"
	"github.com/jiyeyuran/mediasoup-go/internal/channel"
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
	Paused               bool
	DataProducerPaused   bool
	Subchannels          []uint16
	AppData              H
}

type DataConsumer struct {
	baseNotifier

	channel                     *channel.Channel
	data                        *dataconsumerData
	closed                      bool
	paused                      bool
	dataProducerPaused          bool
	subchannels                 []uint16
	pausedListeners             []func(bool)
	sctpSendBufferFullListeners []func()
	bufferedAmountLowListeners  []func(bufferAmount uint32)
	messageListeners            []func(payload []byte, ppid SctpPayloadType)
	sub                         *channel.Subscription
	logger                      *slog.Logger
}

func newDataConsumer(channel *channel.Channel, logger *slog.Logger, data *dataconsumerData) *DataConsumer {
	c := &DataConsumer{
		channel:            channel,
		data:               data,
		paused:             data.Paused,
		dataProducerPaused: data.DataProducerPaused,
		subchannels:        data.Subchannels,
		logger:             logger.With("dataConsumerId", data.DataConsumerId),
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

	return c.paused
}

// DataProducerPaused returns whether the associated DataProducer is paused.
func (c *DataConsumer) DataProducerPaused() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.dataProducerPaused
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
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.logger.Debug("Close()")

	_, err := c.channel.Request(&FbsRequest.RequestT{
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

	c.cleanupAfterClosed()
	return nil
}

// Dump DataConsumer.
func (c *DataConsumer) Dump() (*DataConsumerDump, error) {
	c.logger.Debug("Dump()")

	msg, err := c.channel.Request(&FbsRequest.RequestT{
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
	c.logger.Debug("GetStats()")

	msg, err := c.channel.Request(&FbsRequest.RequestT{
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
	c.logger.Debug("Pause()")

	c.mu.Lock()

	_, err := c.channel.Request(&FbsRequest.RequestT{
		Method:    FbsRequest.MethodDATACONSUMER_PAUSE,
		HandlerId: c.Id(),
	})
	if err != nil {
		c.mu.Unlock()
		return err
	}
	wasPaused := c.paused
	listeners := c.pausedListeners
	c.paused = true
	c.mu.Unlock()

	if !wasPaused && !c.dataProducerPaused {
		for _, listener := range listeners {
			listener(true)
		}
	}

	return err
}

// Resume the DataConsumer.
func (c *DataConsumer) Resume() error {
	c.logger.Debug("Resume()")

	c.mu.Lock()

	_, err := c.channel.Request(&FbsRequest.RequestT{
		Method:    FbsRequest.MethodDATACONSUMER_RESUME,
		HandlerId: c.Id(),
	})
	if err != nil {
		c.mu.Unlock()
		return err
	}
	wasPaused := c.paused
	listeners := c.pausedListeners
	c.paused = false

	c.mu.Unlock()

	if wasPaused && !c.dataProducerPaused {
		for _, listener := range listeners {
			listener(false)
		}
	}

	return err
}

// SetBufferedAmountLowThreshold set buffered amount low threshold.
func (c *DataConsumer) SetBufferedAmountLowThreshold(threshold uint32) error {
	c.logger.Debug("SetBufferedAmountLowThreshold()", "threshold", threshold)

	_, err := c.channel.Request(&FbsRequest.RequestT{
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
	c.logger.Debug("GetBufferedAmount()")

	msg, err := c.channel.Request(&FbsRequest.RequestT{
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
	c.logger.Debug("Send()")

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

	return c.send(data, ppid)
}

// SendText send text.
func (c *DataConsumer) SendText(message string) error {
	c.logger.Debug("SendText()")

	ppid, data := SctpPayloadWebRTCString, []byte(message)

	if len(data) == 0 {
		data, ppid = emptyString[:], SctpPayloadWebRTCBinaryEmpty
	}

	return c.send(data, ppid)
}

func (c *DataConsumer) send(data []byte, ppid SctpPayloadType) error {
	_, err := c.channel.Request(&FbsRequest.RequestT{
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
	c.logger.Debug("SetSubchannels()", "subchannels", subchannels)

	c.mu.Lock()
	defer c.mu.Unlock()

	msg, err := c.channel.Request(&FbsRequest.RequestT{
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
	c.subchannels = resp.Subchannels
	return nil
}

func (c *DataConsumer) AddSubChannel(subchannel uint16) error {
	c.logger.Debug("AddSubChannel()", "subchannel", subchannel)

	c.mu.Lock()
	defer c.mu.Unlock()

	msg, err := c.channel.Request(&FbsRequest.RequestT{
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
	c.subchannels = resp.Subchannels
	return nil
}

func (c *DataConsumer) RemoveSubChannel(subchannel uint16) error {
	c.logger.Debug("RemoveSubChannel()", "subchannel", subchannel)

	c.mu.Lock()
	defer c.mu.Unlock()

	msg, err := c.channel.Request(&FbsRequest.RequestT{
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
	c.subchannels = resp.Subchannels
	return nil
}

// OnSctpSendBufferFull add handler on "sctpsendbufferfull" event
func (c *DataConsumer) OnSctpSendBufferFull(handler func()) {
	c.mu.Lock()
	c.sctpSendBufferFullListeners = append(c.sctpSendBufferFullListeners, handler)
	c.mu.Unlock()
}

// OnBufferedAmountLow add handler on "bufferedamountlow" event
func (c *DataConsumer) OnBufferedAmountLow(handler func(bufferAmount uint32)) {
	c.mu.Lock()
	c.bufferedAmountLowListeners = append(c.bufferedAmountLowListeners, handler)
	c.mu.Unlock()
}

// OnMessage add handler on "message" event
func (c *DataConsumer) OnMessage(handler func(payload []byte, ppid SctpPayloadType)) {
	c.mu.Lock()
	c.messageListeners = append(c.messageListeners, handler)
	c.mu.Unlock()
}

func (c *DataConsumer) handleWorkerNotifications() *channel.Subscription {
	return c.channel.Subscribe(c.Id(), func(event FbsNotification.Event, body *FbsNotification.BodyT) {
		switch event {
		case FbsNotification.EventDATACONSUMER_DATAPRODUCER_CLOSE:
			c.dataProducerClosed()

		case FbsNotification.EventDATACONSUMER_DATAPRODUCER_PAUSE:
			c.mu.Lock()
			if c.dataProducerPaused {
				c.mu.Unlock()
				return
			}
			c.dataProducerPaused = true
			paused := c.paused
			listeners := c.pausedListeners
			c.mu.Unlock()

			if !paused {
				for _, handler := range listeners {
					handler(true)
				}
			}

		case FbsNotification.EventDATACONSUMER_DATAPRODUCER_RESUME:
			c.mu.Lock()
			if !c.dataProducerPaused {
				c.mu.Unlock()
				return
			}
			c.dataProducerPaused = false
			paused := c.paused
			listeners := c.pausedListeners
			c.mu.Unlock()

			if paused {
				for _, handler := range listeners {
					handler(false)
				}
			}

		case FbsNotification.EventDATACONSUMER_SCTP_SENDBUFFER_FULL:
			c.mu.Lock()
			listeners := c.sctpSendBufferFullListeners
			c.mu.Unlock()
			for _, handler := range listeners {
				handler()
			}

		case FbsNotification.EventDATACONSUMER_BUFFERED_AMOUNT_LOW:
			Notification := body.Value.(*FbsDataConsumer.BufferedAmountLowNotificationT)
			c.mu.Lock()
			listeners := c.bufferedAmountLowListeners
			c.mu.Unlock()
			for _, handler := range listeners {
				handler(Notification.BufferedAmount)
			}

		case FbsNotification.EventDATACONSUMER_MESSAGE:
			notification := body.Value.(*FbsDataConsumer.MessageNotificationT)
			c.mu.Lock()
			listeners := c.messageListeners
			c.mu.Unlock()

			for _, handler := range listeners {
				handler(notification.Data, SctpPayloadType(notification.Ppid))
			}

		default:
			c.logger.Warn("ignoring unknown event", "event", event.String())
		}
	})
}

func (c *DataConsumer) dataProducerClosed() {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	c.closed = true
	c.mu.Unlock()

	c.cleanupAfterClosed()
}

func (c *DataConsumer) transportClosed() {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	c.closed = true
	c.mu.Unlock()

	c.cleanupAfterClosed()
}

func (c *DataConsumer) cleanupAfterClosed() {
	c.sub.Unsubscribe()
	c.notifyClosed()
}
