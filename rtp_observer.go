package mediasoup

import (
	"sync"
	"sync/atomic"
)

type IRtpObserver interface {
	IEventEmitter

	Id() string
	Closed() bool
	Paused() bool
	Observer() IEventEmitter
	Close()
	routerClosed()
	Pause()
	Resume()
	AddProducer(producerId string)
	RemoveProducer(producerId string)
}

/**
 * RtpObserver
 * @interface
 * @emits routerclose
 * @emits @close
 */
type RtpObserver struct {
	IEventEmitter
	logger          Logger
	internal        internalData
	channel         *Channel
	payloadChannel  *PayloadChannel
	closed          uint32
	paused          bool
	appData         interface{}
	getProducerById func(string) *Producer
	observer        IEventEmitter
	locker          sync.Mutex
}

type rtpObserverParams struct {
	internal        internalData
	channel         *Channel
	payloadChannel  *PayloadChannel
	appData         interface{}
	getProducerById func(string) *Producer
}

func newRtpObserver(params rtpObserverParams) IRtpObserver {
	logger := NewLogger("RtpObserver")

	logger.Debug("constructor()")

	return &RtpObserver{
		IEventEmitter: NewEventEmitter(),
		logger:        logger,
		// - .RouterId
		// - .RtpObserverId
		internal:        params.internal,
		channel:         params.channel,
		payloadChannel:  params.payloadChannel,
		appData:         params.appData,
		getProducerById: params.getProducerById,
		observer:        NewEventEmitter(),
	}
}

/**
 * RtpObserver id.
 */
func (o *RtpObserver) Id() string {
	return o.internal.RtpObserverId
}

/**
 * Whether the RtpObserver is closed.
 */
func (o *RtpObserver) Closed() bool {
	return atomic.LoadUint32(&o.closed) > 0
}

/**
 * Whether the RtpObserver is paused.
 */
func (o *RtpObserver) Paused() bool {
	o.locker.Lock()
	defer o.locker.Unlock()

	return o.paused
}

/**
 * App custom data.
 */
func (o *RtpObserver) AppData() interface{} {
	return o.appData
}

/**
 * Observer.
 *
 * @emits close
 * @emits pause
 * @emits resume
 * @emits addproducer - (producer: Producer)
 * @emits removeproducer - (producer: Producer)
 */
func (o *RtpObserver) Observer() IEventEmitter {
	return o.observer
}

/**
 * Close the RtpObserver.
 */
func (o *RtpObserver) Close() {
	if atomic.CompareAndSwapUint32(&o.closed, 0, 1) {
		o.logger.Debug("close()")

		// Remove notification subscriptions.
		o.channel.RemoveAllListeners(o.internal.RtpObserverId)
		o.payloadChannel.RemoveAllListeners(o.internal.RtpObserverId)

		o.channel.Request("rtpObserver.close", o.internal)

		o.Emit("@close")
		o.RemoveAllListeners()

		// Emit observer event.
		o.observer.SafeEmit("close")
		o.observer.RemoveAllListeners()
	}
}

/**
 * Router was closed.
 */
func (o *RtpObserver) routerClosed() {
	if atomic.CompareAndSwapUint32(&o.closed, 0, 1) {
		o.logger.Debug("routerClosed()")

		// Remove notification subscriptions.
		o.channel.RemoveAllListeners(o.internal.RtpObserverId)
		o.payloadChannel.RemoveAllListeners(o.internal.RtpObserverId)

		o.Emit("routerclose")
		o.RemoveAllListeners()

		// Emit observer event.
		o.observer.SafeEmit("close")
		o.observer.RemoveAllListeners()
	}
}

/**
 * Pause the RtpObserver.
 */
func (o *RtpObserver) Pause() {
	o.locker.Lock()
	defer o.locker.Unlock()

	o.logger.Debug("pause()")

	wasPaused := o.paused

	o.channel.Request("rtpObserver.pause", o.internal)

	o.paused = true

	// Emit observer event.
	if !wasPaused {
		o.observer.SafeEmit("pause")
	}
}

/**
 * Resume the RtpObserver.
 */
func (o *RtpObserver) Resume() {
	o.locker.Lock()
	defer o.locker.Unlock()

	o.logger.Debug("resume()")

	wasPaused := o.paused

	o.channel.Request("rtpObserver.resume", o.internal)

	o.paused = false

	// Emit observer event.
	if wasPaused {
		o.observer.SafeEmit("resume")
	}
}

/**
 * Add a Producer to the RtpObserver.
 */
func (o *RtpObserver) AddProducer(producerId string) {
	o.locker.Lock()
	defer o.locker.Unlock()

	o.logger.Debug("addProducer()")

	producer := o.getProducerById(producerId)
	internal := o.internal
	internal.ProducerId = producerId

	o.channel.Request("rtpObserver.addProducer", internal)

	// Emit observer event.
	o.observer.SafeEmit("addproducer", producer)
}

/**
 * Remove a Producer from the RtpObserver.
 */
func (o *RtpObserver) RemoveProducer(producerId string) {
	o.locker.Lock()
	defer o.locker.Unlock()

	o.logger.Debug("removeProducer()")

	producer := o.getProducerById(producerId)
	internal := o.internal
	internal.ProducerId = producerId

	o.channel.Request("rtpObserver.removeProducer", internal)

	// Emit observer event.
	o.observer.SafeEmit("removeproducer", producer)
}
