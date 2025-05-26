package mediasoup

import (
	"context"
	"log/slog"

	FbsDataProducer "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/DataProducer"
	FbsNotification "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/Notification"
	FbsRequest "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/Request"
	FbsTransport "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/Transport"

	"github.com/jiyeyuran/mediasoup-go/v2/internal/channel"
)

type dataProducerData struct {
	TransportId          string
	DataProducerId       string
	Type                 DataProducerType
	SctpStreamParameters *SctpStreamParameters
	Label                string
	Protocol             string
	AppData              H

	// changable fields
	Paused bool
}

type DataProducer struct {
	baseListener

	channel         *channel.Channel
	data            *dataProducerData
	closed          bool
	pauseListeners  []func(context.Context)
	resumeListeners []func(context.Context)
	logger          *slog.Logger
}

func newDataProducer(channel *channel.Channel, logger *slog.Logger, data *dataProducerData) *DataProducer {
	return &DataProducer{
		channel: channel,
		logger:  logger.With("dataProducerId", data.DataProducerId),
		data:    data,
	}
}

// Id returns DataProducer id
func (p *DataProducer) Id() string {
	return p.data.DataProducerId
}

// Type returns DataProducer type.
func (p *DataProducer) Type() DataProducerType {
	return p.data.Type
}

// SctpStreamParameters returns SCTP stream parameters.
func (p *DataProducer) SctpStreamParameters() *SctpStreamParameters {
	return clone(p.data.SctpStreamParameters)
}

// Label returns DataChannel label.
func (p *DataProducer) Label() string {
	return p.data.Label
}

// Protocol returns DataChannel protocol.
func (p *DataProducer) Protocol() string {
	return p.data.Protocol
}

func (p *DataProducer) Paused() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.data.Paused
}

// AppData returns app custom data.
func (p *DataProducer) AppData() H {
	return p.data.AppData
}

// Closed returns whether the DataProducer is closed.
func (p *DataProducer) Closed() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.closed
}

// Close the DataProducer.
func (p *DataProducer) Close() error {
	return p.CloseContext(context.Background())
}

func (p *DataProducer) CloseContext(ctx context.Context) error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.logger.DebugContext(ctx, "Close()")

	_, err := p.channel.Request(ctx, &FbsRequest.RequestT{
		Method:    FbsRequest.MethodTRANSPORT_CLOSE_DATAPRODUCER,
		HandlerId: p.data.TransportId,
		Body: &FbsRequest.BodyT{
			Type: FbsRequest.BodyTransport_CloseDataProducerRequest,
			Value: &FbsTransport.CloseDataProducerRequestT{
				DataProducerId: p.Id(),
			},
		},
	})
	if err != nil {
		p.mu.Unlock()
		return err
	}
	p.closed = true
	p.mu.Unlock()

	p.notifyClosed(ctx)

	return nil
}

// Dump DataConsumer.
func (p *DataProducer) Dump() (*DataProducerDump, error) {
	return p.DumpContext(context.Background())
}

func (p *DataProducer) DumpContext(ctx context.Context) (*DataProducerDump, error) {
	p.logger.DebugContext(ctx, "Dump()")

	msg, err := p.channel.Request(ctx, &FbsRequest.RequestT{
		HandlerId: p.Id(),
		Method:    FbsRequest.MethodDATAPRODUCER_DUMP,
	})
	if err != nil {
		return nil, err
	}
	resp := msg.(*FbsDataProducer.DumpResponseT)
	dump := &DataProducerDump{
		Id:       resp.Id,
		Paused:   resp.Paused,
		Type:     orElse(resp.Type == FbsDataProducer.TypeDIRECT, DataProducerDirect, DataProducerSctp),
		Label:    resp.Label,
		Protocol: resp.Protocol,
	}

	if resp.SctpStreamParameters != nil {
		dump.SctpStreamParameters = &SctpStreamParameters{
			StreamId:          resp.SctpStreamParameters.StreamId,
			Ordered:           resp.SctpStreamParameters.Ordered,
			MaxPacketLifeTime: resp.SctpStreamParameters.MaxPacketLifeTime,
			MaxRetransmits:    resp.SctpStreamParameters.MaxRetransmits,
		}
	}

	return dump, nil
}

// GetStats returns DataProducer stats.
func (p *DataProducer) GetStats() ([]*DataProducerStat, error) {
	return p.GetStatsContext(context.Background())
}

func (p *DataProducer) GetStatsContext(ctx context.Context) ([]*DataProducerStat, error) {
	p.logger.DebugContext(ctx, "GetStats()")

	msg, err := p.channel.Request(ctx, &FbsRequest.RequestT{
		Method:    FbsRequest.MethodDATAPRODUCER_GET_STATS,
		HandlerId: p.Id(),
	})
	if err != nil {
		return nil, err
	}
	resp := msg.(*FbsDataProducer.GetStatsResponseT)

	return []*DataProducerStat{
		{
			Type:             "data-producer",
			Timestamp:        resp.Timestamp,
			Label:            resp.Label,
			Protocol:         resp.Protocol,
			MessagesReceived: resp.MessagesReceived,
			BytesReceived:    resp.BytesReceived,
		},
	}, nil
}

// Pause the DataProducer.
func (p *DataProducer) Pause() error {
	return p.PauseContext(context.Background())
}

func (p *DataProducer) PauseContext(ctx context.Context) error {
	p.logger.DebugContext(ctx, "Pause()")

	p.mu.Lock()

	_, err := p.channel.Request(ctx, &FbsRequest.RequestT{
		Method:    FbsRequest.MethodDATAPRODUCER_PAUSE,
		HandlerId: p.Id(),
	})
	if err != nil {
		p.mu.Unlock()
		return err
	}
	wasPaused := p.data.Paused
	listeners := p.pauseListeners
	p.data.Paused = true
	p.mu.Unlock()

	if !wasPaused {
		for _, listener := range listeners {
			listener(ctx)
		}
	}

	return err
}

// Resume the DataProducer.
func (p *DataProducer) Resume() error {
	return p.ResumeContext(context.Background())
}

func (p *DataProducer) ResumeContext(ctx context.Context) error {
	p.logger.DebugContext(ctx, "Resume()")

	p.mu.Lock()

	_, err := p.channel.Request(ctx, &FbsRequest.RequestT{
		Method:    FbsRequest.MethodDATAPRODUCER_RESUME,
		HandlerId: p.Id(),
	})
	if err != nil {
		p.mu.Unlock()
		return err
	}
	wasPaused := p.data.Paused
	listeners := p.resumeListeners
	p.data.Paused = false

	p.mu.Unlock()

	if wasPaused {
		for _, listener := range listeners {
			listener(ctx)
		}
	}

	return err
}

// OnPause add listener on "pause" event.
func (p *DataProducer) OnPause(f func(context.Context)) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.pauseListeners = append(p.pauseListeners, f)
}

// OnResume add listener on "resume" event.
func (p *DataProducer) OnResume(f func(context.Context)) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.resumeListeners = append(p.resumeListeners, f)
}

// Send send binary data.
func (p *DataProducer) Send(data []byte, options ...DataProducerSendOption) (err error) {
	return p.SendContext(context.Background(), data, options...)
}

func (p *DataProducer) SendContext(ctx context.Context, data []byte, options ...DataProducerSendOption) (err error) {
	p.logger.DebugContext(ctx, "Send()")

	ppid := SctpPayloadWebRTCBinary

	if len(data) == 0 {
		ppid, data = SctpPayloadWebRTCBinaryEmpty, make([]byte, 1)
	}
	return p.send(ctx, data, ppid, options...)
}

// SendText send text.
func (p *DataProducer) SendText(message string, options ...DataProducerSendOption) error {
	return p.SendTextContext(context.Background(), message, options...)
}

func (p *DataProducer) SendTextContext(ctx context.Context, message string, options ...DataProducerSendOption) error {
	p.logger.DebugContext(ctx, "SendText()")

	ppid, payload := SctpPayloadWebRTCString, []byte(message)

	if len(payload) == 0 {
		ppid, payload = SctpPayloadWebRTCStringEmpty, []byte{' '}
	}
	return p.send(ctx, payload, ppid, options...)
}

func (p *DataProducer) send(ctx context.Context, data []byte, ppid SctpPayloadType, options ...DataProducerSendOption) error {
	opts := &DataProducerSendOptions{
		Subchannels:        []uint16{},
		RequiredSubchannel: nil,
	}

	for _, option := range options {
		option(opts)
	}

	return p.channel.Notify(ctx, &FbsNotification.NotificationT{
		Event:     FbsNotification.EventDATAPRODUCER_SEND,
		HandlerId: p.Id(),
		Body: &FbsNotification.BodyT{
			Type: FbsNotification.BodyDataProducer_SendNotification,
			Value: &FbsDataProducer.SendNotificationT{
				Data:               data,
				Ppid:               uint32(ppid),
				Subchannels:        opts.Subchannels,
				RequiredSubchannel: opts.RequiredSubchannel,
			},
		},
	})
}

func (p *DataProducer) transportClosed(ctx context.Context) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.closed = true
	p.mu.Unlock()
	p.logger.DebugContext(ctx, "transportClosed()")

	p.notifyClosed(ctx)
}

func (p *DataProducer) syncState(ctx context.Context, other *DataProducer) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.data.Paused != other.Paused() {
		if p.data.Paused {
			return other.PauseContext(ctx)
		} else {
			return other.ResumeContext(ctx)
		}
	}

	return nil
}
