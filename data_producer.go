package mediasoup

import "sync/atomic"

type DataProducerOptions struct {
	/**
	 * DataProducer id (just for Router.pipeToRouter() method).
	 */
	Id string `json:"id,omitempty"`

	/**
	 * SCTP parameters defining how the endpoint is sending the data.
	 * Just if messages are sent over SCTP.
	 */
	SctpStreamParameters *SctpStreamParameters `json:"sctpStreamParameters,omitempty"`

	/**
	 * A label which can be used to distinguish this DataChannel from others.
	 */
	Label string `json:"label,omitempty"`

	/**
	 * Name of the sub-protocol used by this DataChannel.
	 */
	Protocol string `json:"protocol,omitempty"`

	/**
	 * Custom application data.
	 */
	AppData interface{} `json:"app_data,omitempty"`
}

type DataProducerStat struct {
	Type             string
	Timestamp        int64
	Label            string
	Protocol         string
	MessagesReceived int64
	BytesReceived    int64
}

/**
 * DataProducer type.
 */
type DataProducerType = DataConsumerType

const (
	DataProducerType_Sctp   DataProducerType = DataConsumerType_Sctp
	DataProducerType_Direct                  = DataConsumerType_Direct
)

type dataProducerParams struct {
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
}

type dataProducerData struct {
	Type                 DataProducerType
	SctpStreamParameters SctpStreamParameters
	Label                string
	Protocol             string
}

/**
 * DataProducer
 * @emits transportclose
 * @emits @close
 */
type DataProducer struct {
	IEventEmitter
	logger         Logger
	internal       internalData
	data           dataProducerData
	channel        *Channel
	payloadChannel *PayloadChannel
	appData        interface{}
	closed         uint32
	observer       IEventEmitter
}

func newDataProducer(params dataProducerParams) *DataProducer {
	logger := NewLogger("DataProducer")

	logger.Debug("constructor()")

	if params.appData == nil {
		params.appData = H{}
	}

	p := &DataProducer{
		IEventEmitter:  NewEventEmitter(),
		logger:         logger,
		internal:       params.internal,
		data:           params.data,
		channel:        params.channel,
		payloadChannel: params.payloadChannel,
		appData:        params.appData,
		observer:       NewEventEmitter(),
	}

	p.handleWorkerNotifications()

	return p
}

// DataProducer id
func (p *DataProducer) Id() string {
	return p.internal.DataProducerId
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
			p.logger.Error("dataProducer close error: %s", err)
		}

		p.Emit("@close")
		p.RemoveAllListeners()

		// Emit observer event.
		p.observer.SafeEmit("close")
		p.observer.RemoveAllListeners()
	}
	return
}

// Transport was closed.
func (p *DataProducer) transportClosed() {
	if atomic.CompareAndSwapUint32(&p.closed, 0, 1) {
		p.logger.Debug("transportClosed()")

		p.SafeEmit("transportclose")
		p.RemoveAllListeners()

		// Emit observer event.
		p.observer.SafeEmit("close")
		p.observer.RemoveAllListeners()
	}
}

// Dump DataConsumer.
func (p *DataProducer) Dump() (dump DataProducerDump, err error) {
	p.logger.Debug("dump()")

	resp := p.channel.Request("dataProducer.dump", p.internal)
	err = resp.Unmarshal(&dump)
	return
}

// Get DataConsumer stats.
func (p *DataProducer) GetStats() (stats []*DataProducerStat, err error) {
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
			ppidVal = PPID_WEBRTC_BINARY
		} else {
			ppidVal = 57
		}
	} else {
		ppidVal = ppid[0]
	}

	if ppidVal == 56 || ppidVal == 57 {
		data = make([]byte, 1)
	}

	notifData := H{"ppid": ppidVal}

	return p.payloadChannel.Notify("dataProducer.send", p.internal, notifData, data)
}

/**
 * Send text.
 */
func (p *DataProducer) SendText(message string) error {
	ppid := PPID_WEBRTC_STRING

	if len(message) == 0 {
		ppid = 56
	}

	return p.Send([]byte(message), ppid)
}

func (p *DataProducer) handleWorkerNotifications() {
	// No need to subscribe to any event.
}
