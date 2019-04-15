package mediasoup

import (
	"encoding/json"

	"github.com/sirupsen/logrus"
)

var _ Transport = (*WebRtcTransport)(nil)

type WebRtcTransport struct {
	*baseTransport
	logger logrus.FieldLogger
	data   WebRtcTransportData
}

func NewWebRtcTransport(data WebRtcTransportData, params createTransportParams) *WebRtcTransport {
	logger := TypeLogger("WebRtcTransportData")

	logger.Debug("constructor()")

	t := &WebRtcTransport{
		baseTransport: newTransport(params),
		logger:        logger,
		data:          data,
	}

	t.handleWorkerNotifications()

	return t
}

func (t *WebRtcTransport) IceRole() string {
	return t.data.IceRole
}

func (t *WebRtcTransport) IceParameters() IceParameters {
	return t.data.IceParameters
}

func (t *WebRtcTransport) IceCandidates() []IceCandidate {
	return t.data.IceCandidates
}

func (t *WebRtcTransport) IceState() string {
	return t.data.IceState
}

func (t *WebRtcTransport) IceSelectedTuple() *TransportTuple {
	return t.data.IceSelectedTuple
}

func (t *WebRtcTransport) DtlsParameters() DtlsParameters {
	return t.data.DtlsParameters
}

func (t *WebRtcTransport) DtlsState() string {
	return t.data.DtlsState
}

func (t *WebRtcTransport) DtlsRemoteCert() string {
	return t.data.DtlsRemoteCert
}

/**
 * Observer.
 *
 * @override
 * @type {EventEmitter}
 *
 * @emits close
 * @emits {producer: Producer} newproducer
 * @emits {consumer: Consumer} newconsumer
 * @emits {iceState: String} icestatechange
 * @emits {iceSelectedTuple: Object} iceselectedtuplechange
 * @emits {dtlsState: String} dtlsstatechange
 */
func (t *WebRtcTransport) Observer() EventEmitter {
	return t.observer
}

/**
 * Close the WebRtcTransport.
 *
 * @override
 */
func (t *WebRtcTransport) Close() (err error) {
	if t.closed {
		return
	}

	t.data.IceState = "closed"
	t.data.IceSelectedTuple = nil
	t.data.DtlsState = "closed"

	return t.baseTransport.Close()
}

/**
 * Router was closed.
 *
 * @private
 * @override
 */
func (t *WebRtcTransport) routerClosed() {
	if t.closed {
		return
	}

	t.data.IceState = "closed"
	t.data.IceSelectedTuple = nil
	t.data.DtlsState = "closed"

	t.baseTransport.routerClosed()
}

/**
 * Provide the WebRtcTransport remote parameters.
 *
 * @param {RTCDtlsParameters} dtlsParameters - Remote DTLS parameters.
 *
 * @override
 */
func (t *WebRtcTransport) Connect(params transportConnectParams) (err error) {
	t.logger.Debug("connect()")

	resp := t.channel.Request("transport.connect", t.internal, params)

	var data struct {
		DtlsLocalRole string
	}

	if err = resp.Result(&data); err != nil {
		return
	}

	// Update data.
	t.data.DtlsParameters.Role = data.DtlsLocalRole

	return
}

/**
 * Set maximum incoming bitrate for receiving media.
 *
 * @param {Number} bitrate - In bps.
 */
func (t *WebRtcTransport) SetMaxIncomingBitrate(bitrate int) error {
	t.logger.Debugf(`setMaxIncomingBitrate() [bitrate:%d]`, bitrate)

	reqData := map[string]int{
		"bitrate": bitrate,
	}

	resp := t.channel.Request(
		"transport.setMaxIncomingBitrate", t.internal, reqData)

	return resp.Err()
}

/**
 * Restart ICE.
 *
 * @returns {RTCIceParameters}
 */
func (t *WebRtcTransport) RestartIce() (iceParameters IceParameters, err error) {
	t.logger.Debug("restartIce()")

	resp := t.channel.Request("transport.RestartIce", t.internal, nil)

	var data struct {
		IceParameters IceParameters
	}
	if err = resp.Result(&data); err != nil {
		return
	}

	t.data.IceParameters = data.IceParameters

	return iceParameters, nil
}

/**
 * @private
 */
func (t *WebRtcTransport) handleWorkerNotifications() {
	t.channel.On(t.internal.TransportId, func(event string, rawData json.RawMessage) {
		var data WebRtcTransportData
		json.Unmarshal([]byte(rawData), &data)

		switch event {
		case "icestatechange":
			iceState := data.IceState

			t.data.IceState = iceState

			t.SafeEmit("icestatechange", iceState)

			// Emit observer event.
			t.observer.SafeEmit("icestatechange", iceState)

		case "iceselectedtuplechange":
			iceSelectedTuple := *data.IceSelectedTuple

			t.data.IceSelectedTuple = &iceSelectedTuple

			t.SafeEmit("iceselectedtuplechange", iceSelectedTuple)

			// Emit observer event.
			t.observer.SafeEmit("iceselectedtuplechange", iceSelectedTuple)

		case "dtlsstatechange":
			dtlsState, dtlsRemoteCert := data.DtlsState, data.DtlsRemoteCert

			t.data.DtlsState = dtlsState

			if dtlsState == "connected" {
				t.data.DtlsRemoteCert = dtlsRemoteCert
			}

			t.SafeEmit("dtlsstatechange", dtlsState)

			// Emit observer event.
			t.observer.SafeEmit("dtlsstatechange", dtlsState)

		default:
			t.logger.Errorf(`ignoring unknown event "%s"`, event)
		}
	})
}
