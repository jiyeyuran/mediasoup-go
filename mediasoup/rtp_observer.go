package mediasoup

import (
	"github.com/sirupsen/logrus"
)

type RtpObserver struct {
	EventEmitter
	logger          logrus.FieldLogger
	internal        Internal
	channel         *Channel
	closed          bool
	paused          bool
	getProducerById FetchProducerFunc
}

//
func NewRtpObserver(internal Internal, channel *Channel, getProducerById FetchProducerFunc) *RtpObserver {
	logger := TypeLogger("RtpObserver")

	logger.Debug("constructor()")

	return &RtpObserver{
		EventEmitter: NewEventEmitter(logger),
		logger:       logger,
		// - .RouterId
		// - .RtpObserverId
		internal:        internal,
		channel:         channel,
		getProducerById: getProducerById,
	}
}

func (rtpObserver RtpObserver) Id() string {
	return rtpObserver.internal.RtpObserverId
}

func (rtpObserver RtpObserver) Closed() bool {
	return rtpObserver.closed
}

func (rtpObserver RtpObserver) Paused() bool {
	return rtpObserver.paused
}

func (rtpObserver *RtpObserver) Close() {
	if rtpObserver.closed {
		return
	}

	rtpObserver.closed = true

	// Remove notification subscriptions.
	rtpObserver.channel.RemoveAllListeners(rtpObserver.internal.RtpObserverId)

	rtpObserver.channel.Request("rtpObserver.close", rtpObserver.internal, nil)

	rtpObserver.Emit("@close")
}

// Router was closed.
func (rtpObserver *RtpObserver) RouterClosed() {
	if rtpObserver.closed {
		return
	}

	rtpObserver.logger.Debug("routerClosed()")

	rtpObserver.closed = true

	// Remove notification subscriptions.
	rtpObserver.channel.RemoveAllListeners(rtpObserver.internal.RtpObserverId)

	rtpObserver.SafeEmit("routerclose")
}

// Pause the RtpObserver.
func (rtpObserver *RtpObserver) Pause() {
	if rtpObserver.paused {
		return
	}

	rtpObserver.logger.Debug("pause()")

	rtpObserver.channel.Request("rtpObserver.pause", rtpObserver.internal, nil)

	rtpObserver.paused = true
}

// Resume the RtpObserver.
func (rtpObserver *RtpObserver) Resume() {
	if !rtpObserver.paused {
		return

	}
	rtpObserver.logger.Debug("resume()")

	rtpObserver.channel.Request("rtpObserver.resume", rtpObserver.internal, nil)

	rtpObserver.paused = false
}

// Add a Producer to the RtpObserver.
func (rtpObserver *RtpObserver) AddProducer(producerId string) {
	rtpObserver.logger.Debug("addProducer()")

	internal := rtpObserver.internal
	internal.ProducerId = producerId

	rtpObserver.channel.Request("rtpObserver.addProducer", internal, nil)
}

// Remove a Producer from the RtpObserver.
func (rtpObserver *RtpObserver) removeProducer(producerId string) {
	rtpObserver.logger.Debug("removeProducer()")

	internal := rtpObserver.internal
	internal.ProducerId = producerId

	rtpObserver.channel.Request("rtpObserver.removeProducer", internal, nil)
}
