package mediasoup_test

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/jiyeyuran/mediasoup-go/v2"
)

// The usual shape of an SFU: one worker, one router per room, and one transport
// per endpoint. Signalling between the browser and this code is left out; what
// crosses the wire are the transport parameters, the client's RTP capabilities,
// and the producer and consumer parameters.
func Example() {
	worker, err := mediasoup.NewWorker("/path/to/mediasoup-worker")
	if err != nil {
		log.Fatal(err)
	}
	defer worker.Close()

	router, err := worker.CreateRouter(&mediasoup.RouterOptions{
		MediaCodecs: []*mediasoup.RtpCodecCapability{
			{
				Kind:      mediasoup.MediaKindAudio,
				MimeType:  "audio/opus",
				ClockRate: 48000,
				Channels:  2,
			},
			{
				Kind:      mediasoup.MediaKindVideo,
				MimeType:  "video/VP8",
				ClockRate: 90000,
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	// Router.RtpCapabilities go to the client, which answers with its own.
	sendTransport, err := router.CreateWebRtcTransport(&mediasoup.WebRtcTransportOptions{
		ListenInfos: []mediasoup.TransportListenInfo{
			{
				Protocol:         mediasoup.TransportProtocolUDP,
				Ip:               "0.0.0.0",
				AnnouncedAddress: "203.0.113.7",
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	// Transport.Data holds the ICE and DTLS parameters the client needs. Once it
	// replies with its own DTLS parameters, connect the transport.
	if err := sendTransport.Connect(&mediasoup.TransportConnectOptions{
		DtlsParameters: clientDtlsParameters(),
	}); err != nil {
		log.Fatal(err)
	}

	producer, err := sendTransport.Produce(&mediasoup.ProducerOptions{
		Kind:          mediasoup.MediaKindAudio,
		RtpParameters: clientRtpParameters(),
	})
	if err != nil {
		log.Fatal(err)
	}

	// Every other endpoint in the room consumes that producer over its own
	// transport. Check first: a client that cannot decode the codec must not get
	// a consumer.
	recvTransport := transportOfSomeOtherPeer()
	clientCapabilities := rtpCapabilitiesOfSomeOtherPeer()

	if !router.CanConsume(producer.Id(), clientCapabilities) {
		return
	}

	// Start video consumers paused, let the client set up its receiver, and
	// resume once it reports back. Otherwise the first key frame can arrive
	// before the client can render it.
	consumer, err := recvTransport.Consume(&mediasoup.ConsumerOptions{
		ProducerId:      producer.Id(),
		RtpCapabilities: clientCapabilities,
		Paused:          true,
	})
	if err != nil {
		log.Fatal(err)
	}

	if err := consumer.Resume(); err != nil {
		log.Fatal(err)
	}
}

// A worker is a subprocess and can die. Nothing in Go can save the calls already
// in flight, but the rooms it was hosting can be rebuilt on another worker.
func ExampleWorker_OnDied() {
	worker, err := mediasoup.NewWorker("/path/to/mediasoup-worker")
	if err != nil {
		log.Fatal(err)
	}

	worker.OnDied(func(ctx context.Context, err error) {
		log.Printf("worker %d died: %v", worker.Pid(), err)
		// Routers and transports of this worker are about to be closed. Signal
		// the affected clients to renegotiate against a replacement worker.
	})

	// Close returns once shutdown has been requested. Wait for the process
	// itself if you are about to reuse its ports.
	subprocessClosed := make(chan struct{})
	worker.OnSubprocessClose(func(ctx context.Context) { close(subprocessClosed) })

	worker.Close()
	<-subprocessClosed
}

// One worker uses one CPU core. To spread a single room across cores, put a
// router on each worker and pipe producers between them on demand.
func ExampleRouter_PipeToRouter() {
	localRouter, remoteRouter := routerOnThisWorker(), routerOnAnotherWorker()

	result, err := localRouter.PipeToRouter(&mediasoup.PipeToRouterOptions{
		ProducerId: "the-producer-to-forward",
		Router:     remoteRouter,
		ListenInfo: mediasoup.TransportListenInfo{
			Protocol: mediasoup.TransportProtocolUDP,
			Ip:       "127.0.0.1",
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	// Consumers created on remoteRouter now receive the media, forwarded over the
	// pipe transport. result.PipeConsumer is the local end and lives as long as
	// the piping should.
	log.Println("piped as", result.PipeProducer.Id())
}

// Listeners can only be added, so one attached to a long-lived object from a
// short-lived operation has to be removed again.
func ExampleTransport_OnIceStateChange() {
	transport := transportOfSomeOtherPeer()

	connected := make(chan struct{})
	removeListener := transport.OnIceStateChange(func(state mediasoup.IceState) {
		if state == mediasoup.IceStateCompleted || state == mediasoup.IceStateConnected {
			close(connected)
		}
	})
	// Without this, every ICE attempt would leave a listener behind on a
	// transport that may outlive the wait by hours.
	defer removeListener()

	select {
	case <-connected:
	case <-time.After(10 * time.Second):
		log.Println("ICE never connected, closing transport")
		transport.Close()
	}
}

// A DirectTransport moves SCTP messages through this process instead of over the
// network, which is how server-side code joins a data channel.
func ExampleTransport_ConsumeData() {
	router := routerOnThisWorker()

	transport, err := router.CreateDirectTransport(&mediasoup.DirectTransportOptions{})
	if err != nil {
		log.Fatal(err)
	}

	dataConsumer, err := transport.ConsumeData(&mediasoup.DataConsumerOptions{
		DataProducerId: "a-data-producer-in-this-router",
	})
	if err != nil {
		log.Fatal(err)
	}

	dataConsumer.OnMessage(func(payload []byte, ppid mediasoup.SctpPayloadType) {
		// String and binary messages arrive on the same callback, told apart by
		// the SCTP payload protocol identifier.
		switch ppid {
		case mediasoup.SctpPayloadWebRTCString:
			log.Println("text:", string(payload))
		case mediasoup.SctpPayloadWebRTCBinary:
			log.Println("binary:", len(payload), "bytes")
		}
	})
}

// Requests are pipelined to the worker subprocess, so a slow or wedged worker
// shows up as a request that does not come back. Use the Context variants to
// bound it.
func ExampleTransport_ProduceContext() {
	transport := transportOfSomeOtherPeer()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	producer, err := transport.ProduceContext(ctx, &mediasoup.ProducerOptions{
		Kind:          mediasoup.MediaKindVideo,
		RtpParameters: clientRtpParameters(),
	})
	if err != nil {
		// Cancelling abandons the response, it does not undo the request: the
		// worker may well have created the producer. Closing the transport is
		// what cleans that up.
		if errors.Is(err, context.DeadlineExceeded) {
			transport.Close()
		}
		log.Fatal(err)
	}

	log.Println("producing", producer.Id())
}

func clientDtlsParameters() *mediasoup.DtlsParameters { return nil }
func clientRtpParameters() *mediasoup.RtpParameters   { return nil }
func transportOfSomeOtherPeer() *mediasoup.Transport  { return nil }
func routerOnThisWorker() *mediasoup.Router           { return nil }
func routerOnAnotherWorker() *mediasoup.Router        { return nil }

func rtpCapabilitiesOfSomeOtherPeer() *mediasoup.RtpCapabilities {
	return nil
}
