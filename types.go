package mediasoup

import "encoding/json"

type H map[string]interface{}

func Bool(b bool) *bool {
	return &b
}

type DumpResult struct {
	Data []byte
	Err  error
}

func NewDumpResult(data []byte, err error) DumpResult {
	return DumpResult{
		Data: data,
		Err:  err,
	}
}

func (r DumpResult) Unmarshal(v interface{}) error {
	if r.Err != nil {
		return r.Err
	}
	return json.Unmarshal(r.Data, v)
}

type internalData struct {
	RouterId       string `json:"routerId,omitempty"`
	TransportId    string `json:"transportId,omitempty"`
	ProducerId     string `json:"producerId,omitempty"`
	ConsumerId     string `json:"consumerId,omitempty"`
	DataProducerId string `json:"dataProducerId,omitempty"`
	DataConsumerId string `json:"dataConsumerId,omitempty"`
	RtpObserverId  string `json:"rtpObserverId,omitempty"`
}
