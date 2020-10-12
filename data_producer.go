package mediasoup

import "sync/atomic"

type DataProducerOptions struct {
	/**
	 * DataProducer id (just for Router.pipeToRouter() method).
	 */
	Id string

	/**
	 * SCTP parameters defining how the endpoint is sending the data.
	 * Just if messages are sent over SCTP.
	 */
	SctpStreamParameters SctpStreamParameters

	/**
	 * A label which can be used to distinguish this DataChannel from others.
	 */
	Label string

	/**
	 * Name of the sub-protocol used by this DataChannel.
	 */
	Protocol string

	/**
	 * Custom application data.
	 */
	AppData interface{}
}

type DataProducerStat struct {
	Type             string
	Timestamp        uint32
	Label            string
	Protocol         string
	MessagesReceived uint32
	BytesReceived    uint32
}

/**
 * DataProducer type.
 */
type DataProducerType = DataConsumerType

const (
	DataProducerType_Sctp   DataProducerType = DataConsumerType_Sctp
	DataProducerType_Direct                  = DataConsumerType_Direct
)

type newDataProducerOptions struct {
	internal       internalData
	data           dataProducerData
	channel        *Channel
	payloadChannel *PayloadChannel
	appData        interface{}
}

type dataProducerData struct {
	Type                 DataProducerType
	SctpStreamParameters SctpStreamParameters
	Label                string
	Protocol             string
}

type DataProducer struct {
	IEventEmitter
	logger Logger
	// {
	// 	routerId: string;
	// 	transportId: string;
	// 	dataProducerId: string;
	// };
	internal       internalData
	data           dataProducerData
	channel        *Channel
	payloadChannel *PayloadChannel
	appData        interface{}
	closed         uint32
	observer       IEventEmitter
}

/**
 * newDataProducer
 * @emits transportclose
 * @emits @close
 */
func newDataProducer(options newDataProducerOptions) *DataProducer {
	logger := NewLogger("DataProducer")

	logger.Debug("constructor()")

	p := &DataProducer{
		IEventEmitter:  NewEventEmitter(),
		logger:         logger,
		internal:       options.internal,
		data:           options.data,
		channel:        options.channel,
		payloadChannel: options.payloadChannel,
		appData:        options.appData,
		observer:       NewEventEmitter(),
	}

	p.handleWorkerNotifications()

	return p
}

// DataProducer id
func (p *DataProducer) Id() string {
	return p.Id()
}

// Whether the DataProducer is closed.
func (p *DataProducer) Closed() bool {
	return atomic.LoadUint32(&p.closed) > 0
}

// DataProducer type.
func (p *DataProducer) Type() DataConsumerType {
	return p.data.Type
}

/**
 * SCTP stream parameters.
 */
func (p *DataProducer) SctpStreamParameters() SctpStreamParameters {
	return p.data.SctpStreamParameters
}

/**
 * DataChannel label.
 */
func (p *DataProducer) Label() string {
	return p.data.Label
}

/**
 * DataChannel protocol.
 */
func (p *DataProducer) Protocol() string {
	return p.data.Protocol
}

/**
 * App custom data.
 */
func (p *DataProducer) AppData() interface{} {
	return p.appData
}

/**
 * Observer.
 *
 * @emits close
 */
func (p *DataProducer) Observer() IEventEmitter {
	return p.observer
}

// Close the DataProducer.
func (p *DataProducer) Close() (err error) {
	if atomic.CompareAndSwapUint32(&p.closed, 0, 1) {
		p.logger.Debug("close()")

		// Remove notification subscriptions.
		p.channel.RemoveAllListeners(p.Id())
		p.payloadChannel.RemoveAllListeners(p.Id())

		response := p.channel.Request("dataProducer.close", p.internal)

		if err = response.Err(); err != nil {
			return
		}

		p.Emit("@close")

		// Emit observer event.
		p.observer.SafeEmit("close")
	}
	return
}

// Transport was closed.
func (p *DataProducer) transportClosed() {
	if atomic.CompareAndSwapUint32(&p.closed, 0, 1) {
		p.logger.Debug("transportClosed()")

		p.SafeEmit("transportclose")

		// Emit observer event.
		p.observer.SafeEmit("close")
	}
}

// Dump DataConsumer.
func (p *DataProducer) Dump() ([]byte, error) {
	p.logger.Debug("dump()")

	resp := p.channel.Request("dataProducer.dump", p.internal)

	return resp.Data(), resp.Err()
}

// Get DataConsumer stats.
func (p *DataProducer) GetStats() (stats []DataProducerStat, err error) {
	p.logger.Debug("getStats()")

	resp := p.channel.Request("dataProducer.getStats", p.internal)
	err = resp.Unmarshal(&stats)

	return
}

/**
 * Send data.
 */
func (p *DataProducer) Send(data []byte, ppid ...int) (err error) {
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

	resp := p.payloadChannel.Request("dataProducer.send", p.internal, H{"ppid": ppid}, data)

	return resp.Err()
}

/**
 * Send string.
 */
func (p *DataProducer) SendString(message string) error {
	ppid := 51

	if len(message) == 0 {
		ppid = 56
	}

	return p.Send([]byte(message), ppid)
}

func (p *DataProducer) handleWorkerNotifications() {
	// No need to subscribe to any event.
}
