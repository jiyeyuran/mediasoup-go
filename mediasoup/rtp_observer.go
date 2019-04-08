package mediasoup

import (
	"github.com/sirupsen/logrus"
)

type RtpObserver interface {
	EventEmitter

	Id() string
	Closed() bool
	Paused() bool
	Close()
	routerClosed()
	Pause()
	Resume()
	AddProducer(producerId string)
	RemoveProducer(producerId string)
}

type baseRtpObserver struct {
	EventEmitter
	logger   logrus.FieldLogger
	internal Internal
	channel  *Channel
	closed   bool
	paused   bool
}

func newRtpObserver(internal Internal, channel *Channel) *baseRtpObserver {
	logger := TypeLogger("RtpObserver")

	logger.Debug("constructor()")

	return &baseRtpObserver{
		EventEmitter: NewEventEmitter(logger),
		logger:       logger,
		// - .RouterId
		// - .RtpObserverId
		internal: internal,
		channel:  channel,
	}
}

func (rtpObserver baseRtpObserver) Id() string {
	return rtpObserver.internal.RtpObserverId
}

func (rtpObserver baseRtpObserver) Closed() bool {
	return rtpObserver.closed
}

func (rtpObserver baseRtpObserver) Paused() bool {
	return rtpObserver.paused
}

func (rtpObserver *baseRtpObserver) Close() {
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
func (rtpObserver *baseRtpObserver) routerClosed() {
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
func (rtpObserver *baseRtpObserver) Pause() {
	if rtpObserver.paused {
		return
	}

	rtpObserver.logger.Debug("pause()")

	rtpObserver.channel.Request("rtpObserver.pause", rtpObserver.internal, nil)

	rtpObserver.paused = true
}

// Resume the RtpObserver.
func (rtpObserver *baseRtpObserver) Resume() {
	if !rtpObserver.paused {
		return

	}
	rtpObserver.logger.Debug("resume()")

	rtpObserver.channel.Request("rtpObserver.resume", rtpObserver.internal, nil)

	rtpObserver.paused = false
}

// Add a Producer to the RtpObserver.
func (rtpObserver *baseRtpObserver) AddProducer(producerId string) {
	rtpObserver.logger.Debug("addProducer()")

	internal := rtpObserver.internal
	internal.ProducerId = producerId

	rtpObserver.channel.Request("rtpObserver.addProducer", internal, nil)
}

// Remove a Producer from the RtpObserver.
func (rtpObserver *baseRtpObserver) RemoveProducer(producerId string) {
	rtpObserver.logger.Debug("removeProducer()")

	internal := rtpObserver.internal
	internal.ProducerId = producerId

	rtpObserver.channel.Request("rtpObserver.removeProducer", internal, nil)
}
