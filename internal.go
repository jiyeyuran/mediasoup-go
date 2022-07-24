package mediasoup

import "encoding/json"

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

const (
	NS_MESSAGE_MAX_LEN = 4194308
	NS_PAYLOAD_MAX_LEN = 4194304
)

type Marshaler interface {
}

// workerRequest represents the json request sent to the worker
type workerRequest struct {
	Id       int64        `json:"id,omitempty"`
	Method   string       `json:"method,omitempty"`
	Internal internalData `json:"internal,omitempty"`
	Data     interface{}  `json:"data,omitempty"`
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
	request workerRequest
	payload []byte // payload data
	respCh  chan workerResponse
}

// workerNotification is the notification meta info sent to worker
type workerNotification struct {
	Event    string       `json:"event,omitempty"`
	Internal internalData `json:"internal,omitempty"`
	Data     interface{}  `json:"data,omitempty"`
}

// pendingNotification represents the meta data of a payload
// notification from worker
type pendingNotification struct {
	TargetId string          `json:"targetId,omitempty"`
	Event    string          `json:"event,omitempty"`
	Data     json.RawMessage `json:"data,omitempty"`
}
