package mediasoup

import (
	"encoding/json"
	"sync/atomic"
)

type DataConsumerOptions struct {
	/**
	 * The id of the DataProducer to consume.
	 */
	DataProducerId string `json:"dataProducerId,omitempty"`

	/**
	 * Just if consuming over SCTP.
	 * Whether data messages must be received in order. If true the messages will
	 * be sent reliably. Defaults to the value in the DataProducer if it has type
	 * 'sctp' or to true if it has type 'direct'.
	 */
	Ordered *bool `json:"ordered,omitempty"`

	/**
	 * Just if consuming over SCTP.
	 * When ordered is false indicates the time (in milliseconds) after which a
	 * SCTP packet will stop being retransmitted. Defaults to the value in the
	 * DataProducer if it has type 'sctp' or unset if it has type 'direct'.
	 */
	MaxPacketLifeTime uint16 `json:"maxPacketLifeTime,omitempty"`

	/**
	 * Just if consuming over SCTP.
	 * When ordered is false indicates the maximum number of times a packet will
	 * be retransmitted. Defaults to the value in the DataProducer if it has type
	 * 'sctp' or unset if it has type 'direct'.
	 */
	MaxRetransmits uint16 `json:"maxRetransmits,omitempty"`

	/**
	 * Custom application data.
	 */
	AppData interface{} `json:"appData,omitempty"`
}

type DataConsumerStat struct {
	Type           string `json:"type,omitempty"`
	Timestamp      int64  `json:"timestamp,omitempty"`
	Label          string `json:"label,omitempty"`
	Protocol       string `json:"protocol,omitempty"`
	MessagesSent   int64  `json:"messagesSent,omitempty"`
	BytesSent      int64  `json:"bytesSent,omitempty"`
	BufferedAmount uint32 `json:"bufferedAmount,omitempty"`
}

/**
 * DataConsumer type.
 */
type DataConsumerType string

const (
	DataConsumerType_Sctp   DataConsumerType = "sctp"
	DataConsumerType_Direct                  = "direct"
)

type dataConsumerParams struct {
	internal       internalData
	data           dataConsumerData
	channel        *Channel
	payloadChannel *PayloadChannel
	appData        interface{}
}

type dataConsumerData struct {
	Type                 DataConsumerType
	SctpStreamParameters *SctpStreamParameters
	Label                string
	Protocol             string
}

/**
 * DataConsumer
 * @emits transportclose
 * @emits dataproducerclose
 * @emits message - (message: Buffer, ppid: number)
 * @emits sctpsendbufferfull
 * @emits bufferedamountlow - (bufferedAmount: number)
 * @emits @close
 * @emits @dataproducerclose
 */
type DataConsumer struct {
	IEventEmitter
	logger Logger
	// {
	// 	routerId: string;
	// 	transportId: string;
	// 	dataProducerId: string;
	// 	dataConsumerId: string;
	// };
	internal       internalData
	data           dataConsumerData
	channel        *Channel
	payloadChannel *PayloadChannel
	appData        interface{}
	closed         uint32
	observer       IEventEmitter
}

func newDataConsumer(params dataConsumerParams) *DataConsumer {
	logger := NewLogger("DataConsumer")

	logger.Debug("constructor()")

	if params.appData == nil {
		params.appData = H{}
	}

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

// DataConsumer id
func (c *DataConsumer) Id() string {
	return c.internal.DataConsumerId
}

// Associated DataProducer id.
func (c *DataConsumer) DataProducerId() string {
	return c.internal.DataProducerId
}

// Whether the DataConsumer is closed.
func (c *DataConsumer) Closed() bool {
	return atomic.LoadUint32(&c.closed) > 0
}

// DataConsumer type.
func (c *DataConsumer) Type() DataConsumerType {
	return c.data.Type
}

/**
 * SCTP stream parameters.
 */
func (c *DataConsumer) SctpStreamParameters() *SctpStreamParameters {
	return c.data.SctpStreamParameters
}

/**
 * DataChannel label.
 */
func (c *DataConsumer) Label() string {
	return c.data.Label
}

/**
 * DataChannel protocol.
 */
func (c *DataConsumer) Protocol() string {
	return c.data.Protocol
}

/**
 * App custom data.
 */
func (c *DataConsumer) AppData() interface{} {
	return c.appData
}

/**
 * Observer.
 *
 * @emits close
 */
func (c *DataConsumer) Observer() IEventEmitter {
	return c.observer
}

// Close the DataConsumer.
func (c *DataConsumer) Close() (err error) {
	if atomic.CompareAndSwapUint32(&c.closed, 0, 1) {
		c.logger.Debug("close()")

		// Remove notification subscriptions.
		c.channel.RemoveAllListeners(c.Id())
		c.payloadChannel.RemoveAllListeners(c.Id())

		response := c.channel.Request("dataConsumer.close", c.internal)

		if err = response.Err(); err != nil {
			c.logger.Error("dataConsumer close error: %s", err)
		}

		c.Emit("@close")
		c.RemoveAllListeners()

		// Emit observer event.
		c.observer.SafeEmit("close")
		c.observer.RemoveAllListeners()
	}
	return
}

// Transport was closed.
func (c *DataConsumer) transportClosed() {
	if atomic.CompareAndSwapUint32(&c.closed, 0, 1) {
		c.logger.Debug("transportClosed()")

		// Remove notification subscriptions.
		c.channel.RemoveAllListeners(c.Id())
		c.payloadChannel.RemoveAllListeners(c.Id())

		c.SafeEmit("transportclose")
		c.RemoveAllListeners()

		// Emit observer event.
		c.observer.SafeEmit("close")
		c.observer.RemoveAllListeners()
	}
}

// Dump DataConsumer.
func (c *DataConsumer) Dump() (data DataConsumerDump, err error) {
	c.logger.Debug("dump()")

	resp := c.channel.Request("dataConsumer.dump", c.internal)
	err = resp.Unmarshal(&data)

	return
}

// Get DataConsumer stats.
func (c *DataConsumer) GetStats() (stats []*DataConsumerStat, err error) {
	c.logger.Debug("getStats()")

	resp := c.channel.Request("dataConsumer.getStats", c.internal)
	err = resp.Unmarshal(&stats)

	return
}

/**
 * Set buffered amount low threshold.
 */
func (c *DataConsumer) SetBufferedAmountLowThreshold(threshold int) error {
	c.logger.Debug("setBufferedAmountLowThreshold() [threshold:%s]", threshold)

	resp := c.channel.Request("dataConsumer.setBufferedAmountLowThreshold", c.internal, H{
		"threshold": threshold,
	})

	return resp.Err()
}

/**
 * Send data.
 */
func (c *DataConsumer) Send(data []byte, ppid ...int) (err error) {
	/*
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
	ppidVal := 0

	if len(ppid) == 0 {
		if len(data) > 0 {
			ppidVal = 53
		} else {
			ppidVal = 57
		}
	} else {
		ppidVal = ppid[0]
	}

	if ppidVal == 56 || ppidVal == 57 {
		data = make([]byte, 1)
	}

	resp := c.payloadChannel.Request("dataConsumer.send", c.internal, H{"ppid": ppid}, data)

	return resp.Err()
}

/**
 * Send text.
 */
func (c *DataConsumer) SendText(message string) error {
	ppid := 51

	if len(message) == 0 {
		ppid = 56
	}

	return c.Send([]byte(message), ppid)
}

/**
 * Get buffered amount size.
 */
func (c *DataConsumer) GetBufferedAmount() (bufferedAmount int64, err error) {
	c.logger.Debug("getBufferedAmount()")

	resp := c.channel.Request("dataConsumer.getBufferedAmount", c.internal)

	var result struct {
		BufferAmount int64
	}
	err = resp.Unmarshal(&result)

	return result.BufferAmount, err
}

func (c *DataConsumer) handleWorkerNotifications() {
	c.channel.On(c.Id(), func(event string, data []byte) {
		switch event {
		case "dataproducerclose":
			if atomic.CompareAndSwapUint32(&c.closed, 0, 1) {
				c.channel.RemoveAllListeners(c.internal.DataConsumerId)
				c.payloadChannel.RemoveAllListeners(c.internal.DataConsumerId)

				c.Emit("@dataproducerclose")
				c.SafeEmit("dataproducerclose")
				c.RemoveAllListeners()

				// Emit observer event.
				c.observer.SafeEmit("close")
				c.observer.RemoveAllListeners()
			}
		case "sctpsendbufferfull":
			c.SafeEmit("sctpsendbufferfull")

		case "bufferedamountlow":
			var result struct {
				BufferAmount int64
			}
			json.Unmarshal(data, &result)

			c.SafeEmit("bufferedamountlow", result.BufferAmount)

		default:
			c.logger.Error(`ignoring unknown event "%s" in channel listener`, event)
		}
	})

	c.payloadChannel.On(c.Id(), func(event string, data, payload []byte) {
		switch event {
		case "message":
			if c.Closed() {
				return
			}
			var result struct {
				Ppid int
			}
			json.Unmarshal(data, &result)

			c.SafeEmit("message", payload, result.Ppid)

		default:
			c.logger.Error(`ignoring unknown event "%s" in payload channel listener`, event)
		}
	})
}
