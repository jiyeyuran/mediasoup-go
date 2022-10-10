package mediasoup

import (
	"encoding/json"
	"sync/atomic"

	"github.com/go-logr/logr"
)

// DataConsumerOptions define options to create a DataConsumer.
type DataConsumerOptions struct {
	// DataProducerId is the id of the DataProducer to consume.
	DataProducerId string `json:"dataProducerId,omitempty"`

	// Ordered define just if consuming over SCTP.
	// Whether data messages must be received in order. If true the messages will
	// be sent reliably. Defaults to the value in the DataProducer if it has type
	// "sctp" or to true if it has type "direct".
	Ordered *bool `json:"ordered,omitempty"`

	// MaxPacketLifeTime define just if consuming over SCTP.
	// When ordered is false indicates the time (in milliseconds) after which a
	// SCTP packet will stop being retransmitted. Defaults to the value in the
	// DataProducer if it has type 'sctp' or unset if it has type 'direct'.
	MaxPacketLifeTime uint16 `json:"maxPacketLifeTime,omitempty"`

	// MaxRetransmits define just if consuming over SCTP.
	// When ordered is false indicates the maximum number of times a packet will
	// be retransmitted. Defaults to the value in the DataProducer if it has type
	// 'sctp' or unset if it has type 'direct'.
	MaxRetransmits uint16 `json:"maxRetransmits,omitempty"`

	// AppData is custom application data.
	AppData interface{} `json:"appData,omitempty"`
}

// DataConsumerStat define the statistic info for DataConsumer.
type DataConsumerStat struct {
	Type           string `json:"type,omitempty"`
	Timestamp      int64  `json:"timestamp,omitempty"`
	Label          string `json:"label,omitempty"`
	Protocol       string `json:"protocol,omitempty"`
	MessagesSent   int64  `json:"messagesSent,omitempty"`
	BytesSent      int64  `json:"bytesSent,omitempty"`
	BufferedAmount uint32 `json:"bufferedAmount,omitempty"`
}

// DataConsumerType define DataConsumer type.
type DataConsumerType string

const (
	DataConsumerType_Sctp   DataConsumerType = "sctp"
	DataConsumerType_Direct DataConsumerType = "direct"
)

type dataConsumerParams struct {
	internal       internalData
	data           dataConsumerData
	channel        *Channel
	payloadChannel *PayloadChannel
	appData        interface{}
}

type dataConsumerData struct {
	DataProducerId       string                `json:"dataProducerId,omitempty"`
	Type                 DataConsumerType      `json:"type,omitempty"`
	SctpStreamParameters *SctpStreamParameters `json:"sctpStreamParameters,omitempty"`
	Label                string                `json:"label,omitempty"`
	Protocol             string                `json:"protocol,omitempty"`
}

// DataConsumer represents an endpoint capable of receiving data messages from a mediasoup Router.
// A data consumer can use SCTP (AKA DataChannel) to receive those messages, or can directly
// receive them in the golang application if the data consumer was created on top of a
// DirectTransport.
//
// - @emits transportclose
// - @emits dataproducerclose
// - @emits message - (message []bytee, ppid int)
// - @emits sctpsendbufferfull
// - @emits bufferedamountlow - (bufferedAmount int64)
// - @emits @close
// - @emits @dataproducerclose
type DataConsumer struct {
	IEventEmitter
	logger logr.Logger
	// internal uses routerId, transportId, dataProducerId, dataConsumerId
	internal             internalData
	data                 dataConsumerData
	channel              *Channel
	payloadChannel       *PayloadChannel
	appData              interface{}
	closed               uint32
	observer             IEventEmitter
	onClose              func()
	onDataProducerClose  func()
	onTransportClose     func()
	onSctpSendBufferFull func()
	onBufferedAmountLow  func(bufferAmount uint32)
	onMessage            func(payload []byte, ppid int)
}

func newDataConsumer(params dataConsumerParams) *DataConsumer {
	logger := NewLogger("DataConsumer")

	logger.V(1).Info("constructor()", "internal", params.internal)

	consumer := &DataConsumer{
		IEventEmitter:  NewEventEmitter(),
		logger:         logger,
		internal:       params.internal,
		data:           params.data,
		channel:        params.channel,
		payloadChannel: params.payloadChannel,
		appData:        params.appData,
		observer:       NewEventEmitter(),
	}

	consumer.handleWorkerNotifications()

	return consumer
}

// Id returns DataConsumer id
func (c *DataConsumer) Id() string {
	return c.internal.DataConsumerId
}

// DataProducerId returns the associated DataProducer id.
func (c *DataConsumer) DataProducerId() string {
	return c.data.DataProducerId
}

// Closed returns whether the DataConsumer is closed.
func (c *DataConsumer) Closed() bool {
	return atomic.LoadUint32(&c.closed) > 0
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

// AppData returns app custom data.
func (c *DataConsumer) AppData() interface{} {
	return c.appData
}

// Observer.
//
// - @emits close
// - @emits dataproducerclose
// - @emits sctpsendbufferfull
// - @emits message - (message []bytee, ppid int)
// - @emits bufferedamountlow - (bufferAmount int64)
func (c *DataConsumer) Observer() IEventEmitter {
	return c.observer
}

// Close the DataConsumer.
func (c *DataConsumer) Close() (err error) {
	if atomic.CompareAndSwapUint32(&c.closed, 0, 1) {
		c.logger.V(1).Info("close()")

		// Remove notification subscriptions.
		c.channel.Unsubscribe(c.Id())
		c.payloadChannel.Unsubscribe(c.Id())

		reqData := H{"dataConsumerId": c.internal.DataConsumerId}

		response := c.channel.Request("transport.closeDataConsumer", c.internal, reqData)

		if err = response.Err(); err != nil {
			c.logger.Error(err, "dataConsumer close failed")
		}

		c.Emit("@close")
		c.RemoveAllListeners()

		c.close()
	}
	return
}

// close send "close" event.
func (c *DataConsumer) close() {
	// Emit observer event.
	c.observer.SafeEmit("close")
	c.observer.RemoveAllListeners()

	if handler := c.onClose; handler != nil {
		handler()
	}
}

// transportClosed is called when transport was closed.
func (c *DataConsumer) transportClosed() {
	if atomic.CompareAndSwapUint32(&c.closed, 0, 1) {
		c.logger.V(1).Info("transportClosed()")

		// Remove notification subscriptions.
		c.channel.Unsubscribe(c.Id())
		c.payloadChannel.Unsubscribe(c.Id())

		c.SafeEmit("transportclose")
		c.RemoveAllListeners()

		if handler := c.onTransportClose; handler != nil {
			handler()
		}

		c.close()
	}
}

// Dump DataConsumer.
func (c *DataConsumer) Dump() (data DataConsumerDump, err error) {
	c.logger.V(1).Info("dump()")

	resp := c.channel.Request("dataConsumer.dump", c.internal)
	err = resp.Unmarshal(&data)

	return
}

// GetStats returns DataConsumer stats.
func (c *DataConsumer) GetStats() (stats []*DataConsumerStat, err error) {
	c.logger.V(1).Info("getStats()")

	resp := c.channel.Request("dataConsumer.getStats", c.internal)
	err = resp.Unmarshal(&stats)

	return
}

// SetBufferedAmountLowThreshold set buffered amount low threshold.
func (c *DataConsumer) SetBufferedAmountLowThreshold(threshold int) error {
	c.logger.V(1).Info("setBufferedAmountLowThreshold() [threshold:%s]", threshold)

	resp := c.channel.Request("dataConsumer.setBufferedAmountLowThreshold", c.internal, H{
		"threshold": threshold,
	})

	return resp.Err()
}

// Send data.
func (c *DataConsumer) Send(data []byte) (err error) {
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
	ppid := "53"

	if len(data) == 0 {
		ppid, data = "57", make([]byte, 1)
	}

	resp := c.payloadChannel.Request("dataConsumer.send", c.internal, ppid, data)

	return resp.Err()
}

// SendText send text.
func (c *DataConsumer) SendText(message string) error {
	ppid, payload := "51", []byte(message)

	if len(payload) == 0 {
		ppid, payload = "56", []byte{' '}
	}

	resp := c.payloadChannel.Request("dataConsumer.send", c.internal, ppid, payload)
	return resp.Err()
}

// GetBufferedAmount returns buffered amount size.
func (c *DataConsumer) GetBufferedAmount() (bufferedAmount int64, err error) {
	c.logger.V(1).Info("getBufferedAmount()")

	resp := c.channel.Request("dataConsumer.getBufferedAmount", c.internal)

	var result struct {
		BufferAmount int64
	}
	err = resp.Unmarshal(&result)

	return result.BufferAmount, err
}

// OnClose set handler on "close" event
func (c *DataConsumer) OnClose(handler func()) {
	c.onClose = handler
}

// OnDataProducerClose set handler on "dataproducerclose" event
func (consumer *DataConsumer) OnDataProducerClose(handler func()) {
	consumer.onDataProducerClose = handler
}

// OnTransportClose set handler on "transportclose" event
func (consumer *DataConsumer) OnTransportClose(handler func()) {
	consumer.onTransportClose = handler
}

// OnSctpSendBufferFull set handler on "sctpsendbufferfull" event
func (c *DataConsumer) OnSctpSendBufferFull(handler func()) {
	c.onSctpSendBufferFull = handler
}

// OnBufferedAmountLow set handler on "bufferedamountlow" event
func (c *DataConsumer) OnBufferedAmountLow(handler func(bufferAmount uint32)) {
	c.onBufferedAmountLow = handler
}

// OnMessage set handler on "message" event
func (c *DataConsumer) OnMessage(handler func(payload []byte, ppid int)) {
	c.onMessage = handler
}

func (c *DataConsumer) handleWorkerNotifications() {
	c.channel.Subscribe(c.Id(), func(event string, data []byte) {
		switch event {
		case "dataproducerclose":
			if atomic.CompareAndSwapUint32(&c.closed, 0, 1) {
				c.channel.Unsubscribe(c.internal.DataConsumerId)
				c.payloadChannel.Unsubscribe(c.internal.DataConsumerId)

				c.Emit("@dataproducerclose")
				c.SafeEmit("dataproducerclose")
				c.RemoveAllListeners()

				if handler := c.onDataProducerClose; handler != nil {
					handler()
				}

				c.close()
			}
		case "sctpsendbufferfull":
			c.SafeEmit("sctpsendbufferfull")

			if handler := c.onSctpSendBufferFull; handler != nil {
				handler()
			}

		case "bufferedamountlow":
			var result struct {
				BufferAmount uint32
			}
			if err := json.Unmarshal([]byte(data), &result); err != nil {
				c.logger.Error(err, "failed to unmarshal bufferedamountlow", "data", json.RawMessage(data))
				return
			}

			c.SafeEmit("bufferedamountlow", result.BufferAmount)

			if handler := c.onBufferedAmountLow; handler != nil {
				handler(result.BufferAmount)
			}

		default:
			c.logger.Error(nil, "ignoring unknown event in channel listener", "event", event)
		}
	})

	c.payloadChannel.Subscribe(c.Id(), func(event string, data, payload []byte) {
		switch event {
		case "message":
			if c.Closed() {
				return
			}
			var result struct {
				Ppid int
			}
			if err := json.Unmarshal([]byte(data), &result); err != nil {
				c.logger.Error(err, "failed to unmarshal message", "data", json.RawMessage(data))
				return
			}

			c.SafeEmit("message", payload, result.Ppid)

			if handler := c.onMessage; handler != nil {
				handler(payload, result.Ppid)
			}

		default:
			c.logger.Error(nil, "ignoring unknown event in payload channel listener", "event", event)
		}
	})
}
