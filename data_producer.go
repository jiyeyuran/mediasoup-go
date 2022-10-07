package mediasoup

import (
	"sync"
	"sync/atomic"

	"github.com/go-logr/logr"
)

// DataProducerOptions define options to create a DataProducer.
type DataProducerOptions struct {
	// Id is DataProducer id (just for Router.pipeToRouter() method).
	Id string `json:"id,omitempty"`

	// SctpStreamParameters define how the endpoint is sending the data.
	// Just if messages are sent over SCTP.
	SctpStreamParameters *SctpStreamParameters `json:"sctpStreamParameters,omitempty"`

	// Label can be used to distinguish this DataChannel from others.
	Label string `json:"label,omitempty"`

	// Protocol is the name of the sub-protocol used by this DataChannel.
	Protocol string `json:"protocol,omitempty"`

	// AppData is custom application data.
	AppData interface{} `json:"app_data,omitempty"`
}

// DataProducerStat define the statistic info for DataProducer.
type DataProducerStat struct {
	Type             string
	Timestamp        int64
	Label            string
	Protocol         string
	MessagesReceived int64
	BytesReceived    int64
}

// DataProducerType define DataProducer type.
type DataProducerType string

const (
	DataProducerType_Sctp   DataProducerType = "sctp"
	DataProducerType_Direct DataProducerType = "direct"
)

type dataProducerParams struct {
	// internal uses routerId, transportId, dataProducerId
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

// DataProducer represents an endpoint capable of injecting data messages into a mediasoup Router.
// A data producer can use SCTP (AKA DataChannel) to deliver those messages, or can directly send
// them from the golang application if the data producer was created on top of a DirectTransport.
//
// - @emits transportclose
// - @emits @close
type DataProducer struct {
	IEventEmitter
	mu             sync.Mutex
	logger         logr.Logger
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

	logger.V(1).Info("constructor()", "internal", params.internal)

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

// Id returns DataProducer id
func (p *DataProducer) Id() string {
	return p.internal.DataProducerId
}

// Closed returns whether the DataProducer is closed.
func (p *DataProducer) Closed() bool {
	return atomic.LoadUint32(&p.closed) > 0
}

// Type returns DataProducer type.
func (p *DataProducer) Type() DataProducerType {
	return p.data.Type
}

// SctpStreamParameters returns SCTP stream parameters.
func (p *DataProducer) SctpStreamParameters() SctpStreamParameters {
	return p.data.SctpStreamParameters
}

// Label returns DataChannel label.
func (p *DataProducer) Label() string {
	return p.data.Label
}

// Protocol returns DataChannel protocol.
func (p *DataProducer) Protocol() string {
	return p.data.Protocol
}

// AppData returns app custom data.
func (p *DataProducer) AppData() interface{} {
	return p.appData
}

// Observer.
//
// - @emits close
func (p *DataProducer) Observer() IEventEmitter {
	return p.observer
}

// Close the DataProducer.
func (p *DataProducer) Close() (err error) {
	if atomic.CompareAndSwapUint32(&p.closed, 0, 1) {
		p.logger.V(1).Info("close()")

		// Remove notification subscriptions.
		p.channel.Unsubscribe(p.Id())
		p.payloadChannel.Unsubscribe(p.Id())

		reqData := H{"dataProducerId": p.internal.DataProducerId}

		response := p.channel.Request("transport.closeDataProducer", p.internal, reqData)

		if err = response.Err(); err != nil {
			p.logger.Error(err, "dataProducer close failed")
		}

		p.Emit("@close")
		p.RemoveAllListeners()

		// Emit observer event.
		p.observer.SafeEmit("close")
		p.observer.RemoveAllListeners()
	}
	return
}

// transportClosed is called when transport was closed.
func (p *DataProducer) transportClosed() {
	if atomic.CompareAndSwapUint32(&p.closed, 0, 1) {
		p.logger.V(1).Info("transportClosed()")

		p.SafeEmit("transportclose")
		p.RemoveAllListeners()

		// Emit observer event.
		p.observer.SafeEmit("close")
		p.observer.RemoveAllListeners()
	}
}

// Dump DataConsumer.
func (p *DataProducer) Dump() (dump DataProducerDump, err error) {
	p.logger.V(1).Info("dump()")

	resp := p.channel.Request("dataProducer.dump", p.internal)
	err = resp.Unmarshal(&dump)
	return
}

// GetStats returns DataConsumer stats.
func (p *DataProducer) GetStats() (stats []*DataProducerStat, err error) {
	p.logger.V(1).Info("getStats()")

	resp := p.channel.Request("dataProducer.getStats", p.internal)
	err = resp.Unmarshal(&stats)

	return
}

// Send data.
func (p *DataProducer) Send(data []byte) (err error) {
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

	return p.payloadChannel.Notify("dataProducer.send", p.internal, ppid, data)
}

// SendText send text.
func (p *DataProducer) SendText(message string) error {
	ppid, payload := "51", []byte(message)

	if len(payload) == 0 {
		ppid, payload = "56", []byte{' '}
	}

	return p.payloadChannel.Notify("dataProducer.send", p.internal, ppid, payload)
}

func (p *DataProducer) handleWorkerNotifications() {
	// No need to subscribe to any event.
}
