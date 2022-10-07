package mediasoup

import (
	"sync"
	"sync/atomic"

	"github.com/go-logr/logr"
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

// RtpObserver is a base class inherited by ActiveSpeakerObserver and AudioLevelObserver.
//
// - @emits routerclose
// - @emits @close
type RtpObserver struct {
	IEventEmitter
	logger          logr.Logger
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

	logger.V(1).Info("constructor()", "internal", params.internal)

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

// Id returns RtpObserver id.
func (o *RtpObserver) Id() string {
	return o.internal.RtpObserverId
}

// Closed returns whether the RtpObserver is closed.
func (o *RtpObserver) Closed() bool {
	return atomic.LoadUint32(&o.closed) > 0
}

// Paused returns whether the RtpObserver is paused.
func (o *RtpObserver) Paused() bool {
	o.locker.Lock()
	defer o.locker.Unlock()

	return o.paused
}

// AppData returns app custom data.
func (o *RtpObserver) AppData() interface{} {
	return o.appData
}

// Observer.
//
// - @emits close
// - @emits pause
// - @emits resume
// - @emits addproducer - (producer *Producer)
// - @emits removeproducer - (producer *Producer)
func (o *RtpObserver) Observer() IEventEmitter {
	return o.observer
}

// Close the RtpObserver.
func (o *RtpObserver) Close() {
	if atomic.CompareAndSwapUint32(&o.closed, 0, 1) {
		o.logger.V(1).Info("close()")

		// Remove notification subscriptions.
		o.channel.Unsubscribe(o.internal.RtpObserverId)
		o.payloadChannel.Unsubscribe(o.internal.RtpObserverId)

		reqData := H{"rtpObserverId": o.internal.RtpObserverId}

		o.channel.Request("router.closeRtpObserver", o.internal, reqData)

		o.Emit("@close")
		o.RemoveAllListeners()

		// Emit observer event.
		o.observer.SafeEmit("close")
		o.observer.RemoveAllListeners()
	}
}

// routerClosed is called when router was closed.
func (o *RtpObserver) routerClosed() {
	if atomic.CompareAndSwapUint32(&o.closed, 0, 1) {
		o.logger.V(1).Info("routerClosed()")

		// Remove notification subscriptions.
		o.channel.Unsubscribe(o.internal.RtpObserverId)
		o.payloadChannel.Unsubscribe(o.internal.RtpObserverId)

		o.Emit("routerclose")
		o.RemoveAllListeners()

		// Emit observer event.
		o.observer.SafeEmit("close")
		o.observer.RemoveAllListeners()
	}
}

// Pause the RtpObserver.
func (o *RtpObserver) Pause() {
	o.locker.Lock()
	defer o.locker.Unlock()

	o.logger.V(1).Info("pause()")

	wasPaused := o.paused

	o.channel.Request("rtpObserver.pause", o.internal)

	o.paused = true

	// Emit observer event.
	if !wasPaused {
		o.observer.SafeEmit("pause")
	}
}

// Resume the RtpObserver.
func (o *RtpObserver) Resume() {
	o.locker.Lock()
	defer o.locker.Unlock()

	o.logger.V(1).Info("resume()")

	wasPaused := o.paused

	o.channel.Request("rtpObserver.resume", o.internal)

	o.paused = false

	// Emit observer event.
	if wasPaused {
		o.observer.SafeEmit("resume")
	}
}

// AddProducer add a Producer to the RtpObserver.
func (o *RtpObserver) AddProducer(producerId string) {
	o.locker.Lock()
	defer o.locker.Unlock()

	o.logger.V(1).Info("addProducer()")

	producer := o.getProducerById(producerId)
	internal := o.internal
	internal.ProducerId = producerId

	o.channel.Request("rtpObserver.addProducer", internal)

	// Emit observer event.
	o.observer.SafeEmit("addproducer", producer)
}

// RemoveProducer remove a Producer from the RtpObserver.
func (o *RtpObserver) RemoveProducer(producerId string) {
	o.locker.Lock()
	defer o.locker.Unlock()

	o.logger.V(1).Info("removeProducer()")

	producer := o.getProducerById(producerId)
	internal := o.internal
	internal.ProducerId = producerId

	o.channel.Request("rtpObserver.removeProducer", internal)

	// Emit observer event.
	o.observer.SafeEmit("removeproducer", producer)
}
