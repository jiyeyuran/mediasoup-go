package mediasoup

import (
	"encoding/json"
	"strings"
)

type internalData struct {
	RouterId       string `json:"routerId,omitempty"`
	TransportId    string `json:"transportId,omitempty"`
	ProducerId     string `json:"producerId,omitempty"`
	ConsumerId     string `json:"consumerId,omitempty"`
	DataProducerId string `json:"dataProducerId,omitempty"`
	DataConsumerId string `json:"dataConsumerId,omitempty"`
	RtpObserverId  string `json:"rtpObserverId,omitempty"`
	WebRtcServerId string `json:"webRtcServerId,omitempty"`
}

func (i internalData) HandlerID(method string) string {
	switch strings.Split(method, ".")[0] {
	case "router":
		return i.RouterId

	case "transport":
		return i.TransportId

	case "producer":
		return i.ProducerId

	case "consumer":
		return i.ConsumerId

	case "dataProducer":
		return i.DataProducerId

	case "dataConsumer":
		return i.DataConsumerId

	case "rtpObserver":
		return i.RtpObserverId

	case "webRtcServer":
		return i.WebRtcServerId

	default:
		return "undefined"
	}
}

const (
	NS_MESSAGE_MAX_LEN = 4194308
	NS_PAYLOAD_MAX_LEN = 4194304
)

// workerRequest represents the json request sent to the worker
type workerRequest struct {
	Id       int64           `json:"id,omitempty"`
	Method   string          `json:"method,omitempty"`
	Internal internalData    `json:"internal,omitempty"`
	Data     json.RawMessage `json:"data,omitempty"`
}

// workerResponse represents the json response returned from the worker
type workerResponse struct {
	data json.RawMessage
	err  error
}

func (r workerResponse) Unmarshal(v interface{}) error {
	if r.err != nil {
		return r.err
	}
	if len(r.data) == 0 {
		return nil
	}
	return json.Unmarshal([]byte(r.data), v)
}

func (r workerResponse) Data() []byte {
	return []byte(r.data)
}

func (r workerResponse) Err() error {
	return r.err
}

// sentInfo includes rpc info
type sentInfo struct {
	method  string              // method name
	request []byte              // request json data
	payload []byte              // payload json data, used by payload channel
	respCh  chan workerResponse // channel to hold response
}

// workerNotification is the notification meta info sent to worker
type workerNotification struct {
	Event    string       `json:"event,omitempty"`
	Internal internalData `json:"internal,omitempty"`
	Data     interface{}  `json:"data,omitempty"`
}

// notification represents a notification of the specified target from worker
type notification struct {
	TargetId string          `json:"targetId,omitempty"`
	Event    string          `json:"event,omitempty"`
	Data     json.RawMessage `json:"data,omitempty"`
}
