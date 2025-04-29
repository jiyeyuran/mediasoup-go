package mediasoup

import (
	"log/slog"

	FbsDataProducer "github.com/jiyeyuran/mediasoup-go/internal/FBS/DataProducer"
	FbsNotification "github.com/jiyeyuran/mediasoup-go/internal/FBS/Notification"
	FbsRequest "github.com/jiyeyuran/mediasoup-go/internal/FBS/Request"
	FbsTransport "github.com/jiyeyuran/mediasoup-go/internal/FBS/Transport"

	"github.com/jiyeyuran/mediasoup-go/internal/channel"
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
	baseNotifier

	channel         *channel.Channel
	data            *dataProducerData
	closed          bool
	pausedListeners []func(bool)
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
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.closed
}

// Close the DataProducer.
func (p *DataProducer) Close() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.logger.Debug("Close()")

	_, err := p.channel.Request(&FbsRequest.RequestT{
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

	p.notifyClosed()

	return nil
}

// Dump DataConsumer.
func (p *DataProducer) Dump() (*DataProducerDump, error) {
	p.logger.Debug("Dump()")

	msg, err := p.channel.Request(&FbsRequest.RequestT{
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
	p.logger.Debug("GetStats()")

	msg, err := p.channel.Request(&FbsRequest.RequestT{
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
	p.logger.Debug("Pause()")

	p.mu.Lock()

	_, err := p.channel.Request(&FbsRequest.RequestT{
		Method:    FbsRequest.MethodDATAPRODUCER_PAUSE,
		HandlerId: p.Id(),
	})
	if err != nil {
		p.mu.Unlock()
		return err
	}
	wasPaused := p.data.Paused
	listeners := p.pausedListeners
	p.data.Paused = true
	p.mu.Unlock()

	if !wasPaused {
		for _, listener := range listeners {
			listener(true)
		}
	}

	return err
}

// Resume the DataProducer.
func (p *DataProducer) Resume() error {
	p.logger.Debug("Resume()")

	p.mu.Lock()

	_, err := p.channel.Request(&FbsRequest.RequestT{
		Method:    FbsRequest.MethodDATAPRODUCER_RESUME,
		HandlerId: p.Id(),
	})
	if err != nil {
		p.mu.Unlock()
		return err
	}
	wasPaused := p.data.Paused
	listeners := p.pausedListeners
	p.data.Paused = false

	p.mu.Unlock()

	if wasPaused {
		for _, listener := range listeners {
			listener(false)
		}
	}

	return err
}

func (p *DataProducer) OnPaused(f func(bool)) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.pausedListeners = append(p.pausedListeners, f)
}

// Send send binary data.
func (p *DataProducer) Send(data []byte) (err error) {
	p.logger.Debug("Send()")

	ppid := SctpPayloadWebRTCBinary

	if len(data) == 0 {
		ppid, data = SctpPayloadWebRTCBinaryEmpty, make([]byte, 1)
	}
	return p.send(data, ppid)
}

// SendText send text.
func (p *DataProducer) SendText(message string) error {
	p.logger.Debug("SendText()")

	ppid, payload := SctpPayloadWebRTCString, []byte(message)

	if len(payload) == 0 {
		ppid, payload = SctpPayloadWebRTCStringEmpty, []byte{' '}
	}
	return p.send(payload, ppid)
}

func (p *DataProducer) send(data []byte, ppid SctpPayloadType) error {
	return p.channel.Notify(&FbsNotification.NotificationT{
		Event:     FbsNotification.EventDATAPRODUCER_SEND,
		HandlerId: p.Id(),
		Body: &FbsNotification.BodyT{
			Type: FbsNotification.BodyDataProducer_SendNotification,
			Value: &FbsDataProducer.SendNotificationT{
				Data: data,
				Ppid: uint32(ppid),
			},
		},
	})
}

func (p *DataProducer) transportClosed() {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.closed = true
	p.mu.Unlock()
	p.notifyClosed()
}
