package mediasoup

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/jiyeyuran/mediasoup-go/h264"
)

var DYNAMIC_PAYLOAD_TYPES = [...]byte{
	100, 101, 102, 103, 104, 105, 106, 107, 108, 109, 110, 111, 112, 113, 114, 115,
	116, 117, 118, 119, 120, 121, 122, 123, 124, 125, 126, 127, 96, 97, 98, 99, 77,
	78, 79, 80, 81, 82, 83, 84, 85, 86, 87, 88, 89, 90, 91, 92, 93, 94, 95, 35, 36,
	37, 38, 39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50, 51, 52, 53, 54, 55, 56,
	57, 58, 59, 60, 61, 62, 63, 64, 65, 66, 67, 68, 69, 70, 71,
}

type matchOptions struct {
	strict bool
	modify bool
}

type RtpMapping struct {
	Codecs    []RtpMappingCodec    `json:"codecs,omitempty"`
	Encodings []RtpMappingEncoding `json:"encodings,omitempty"`
}

type RtpMappingCodec struct {
	PayloadType       byte `json:"payloadType"`
	MappedPayloadType byte `json:"mappedPayloadType"`
}

type RtpMappingEncoding struct {
	Ssrc            uint32 `json:"ssrc,omitempty"`
	Rid             string `json:"rid,omitempty"`
	ScalabilityMode string `json:"scalabilityMode,omitempty"`
	MappedSsrc      uint32 `json:"mappedSsrc"`
}

/**
 * Validates RtpCapabilities. It may modify given data by adding missing
 * fields with default values.
 */
func validateRtpCapabilities(params *RtpCapabilities) (err error) {
	for _, codec := range params.Codecs {
		if err = validateRtpCodecCapability(codec); err != nil {
			return
		}
	}

	for _, ext := range params.HeaderExtensions {
		if err = validateRtpHeaderExtension(ext); err != nil {
			return
		}
	}

	return
}

/**
 * Validates RtpCodecCapability. It may modify given data by adding missing
 * fields with default values.
 */
func validateRtpCodecCapability(code *RtpCodecCapability) (err error) {
	mimeType := strings.ToLower(code.MimeType)

	//  mimeType is mandatory.
	if !strings.HasPrefix(mimeType, "audio/") && !strings.HasPrefix(mimeType, "video/") {
		return NewTypeError("invalid codec.mimeType")
	}

	code.Kind = MediaKind(strings.Split(mimeType, "/")[0])

	// clockRate is mandatory.
	if code.ClockRate == 0 {
		return NewTypeError("missing codec.clockRate")
	}

	// channels is optional. If unset, set it to 1 (just if audio).
	if code.Kind == MediaKind_Audio && code.Channels == 0 {
		code.Channels = 1
	}

	for _, fb := range code.RtcpFeedback {
		if err = validateRtcpFeedback(fb); err != nil {
			return
		}
	}

	return
}

/**
 * Validates RtcpFeedback. It may modify given data by adding missing
 * fields with default values.
 */
func validateRtcpFeedback(fb RtcpFeedback) error {
	if len(fb.Type) == 0 {
		return NewTypeError("missing fb.type")
	}
	return nil
}

/**
 * Validates RtpHeaderExtension. It may modify given data by adding missing
 * fields with default values.
 */
func validateRtpHeaderExtension(ext *RtpHeaderExtension) (err error) {
	if len(ext.Kind) > 0 && ext.Kind != MediaKind_Audio && ext.Kind != MediaKind_Video {
		return NewTypeError("invalid ext.kind")
	}

	// uri is mandatory.
	if len(ext.Uri) == 0 {
		return NewTypeError("missing ext.uri")
	}

	// preferredId is mandatory.
	if ext.PreferredId == 0 {
		return NewTypeError("missing ext.preferredId")
	}

	// direction is optional. If unset set it to sendrecv.
	if len(ext.Direction) == 0 {
		ext.Direction = Direction_Sendrecv
	}

	return
}

/**
 * Validates RtpParameters. It may modify given data by adding missing
 * fields with default values.
 */
func validateRtpParameters(params *RtpParameters) (err error) {
	for _, codec := range params.Codecs {
		if err = validateRtpCodecParameters(codec); err != nil {
			return
		}
	}

	for _, ext := range params.HeaderExtensions {
		if err = validateRtpHeaderExtensionParameters(ext); err != nil {
			return
		}
	}

	return validateRtcpParameters(&params.Rtcp)
}

/**
 * Validates RtpCodecParameters. It may modify given data by adding missing
 * fields with default values.
 */
func validateRtpCodecParameters(code *RtpCodecParameters) (err error) {
	mimeType := strings.ToLower(code.MimeType)

	//  mimeType is mandatory.
	if !strings.HasPrefix(mimeType, "audio/") && !strings.HasPrefix(mimeType, "video/") {
		return NewTypeError("invalid codec.mimeType")
	}

	// clockRate is mandatory.
	if code.ClockRate == 0 {
		return NewTypeError("missing codec.clockRate")
	}

	kind := MediaKind(strings.Split(mimeType, "/")[0])

	// channels is optional. If unset, set it to 1 (just if audio).
	if kind == MediaKind_Audio && code.Channels == 0 {
		code.Channels = 1
	}

	for _, fb := range code.RtcpFeedback {
		if err = validateRtcpFeedback(fb); err != nil {
			return
		}
	}

	return
}

/**
 * Validates RtpHeaderExtension. It may modify given data by adding missing
 * fields with default values.
 */
func validateRtpHeaderExtensionParameters(ext RtpHeaderExtensionParameters) (err error) {
	// uri is mandatory.
	if len(ext.Uri) == 0 {
		return NewTypeError("missing ext.uri")
	}

	// preferredId is mandatory.
	if ext.Id == 0 {
		return NewTypeError("missing ext.id")
	}

	return
}

/**
 * Validates RtcpParameters. It may modify given data by adding missing
 * fields with default values.
 */
func validateRtcpParameters(rtcp *RtcpParameters) (err error) {
	// reducedSize is optional. If unset set it to true.
	if rtcp.ReducedSize == nil {
		rtcp.ReducedSize = Bool(true)
	}

	return
}

/**
 * Validates SctpCapabilities. It may modify given data by adding missing
 * fields with default values.
 */
func validateSctpCapabilities(caps SctpCapabilities) (err error) {
	// numStreams is mandatory.
	if reflect.DeepEqual(caps.NumStreams, NumSctpStreams{}) {
		return NewTypeError("missing caps.numStreams")
	}

	return validateNumSctpStreams(caps.NumStreams)
}

/**
 * Validates NumSctpStreams. It may modify given data by adding missing
 * fields with default values.
 */
func validateNumSctpStreams(numStreams NumSctpStreams) (err error) {
	// OS is mandatory.
	if numStreams.OS == 0 {
		return NewTypeError("missing numStreams.OS")
	}
	// MIS is mandatory.
	if numStreams.MIS == 0 {
		return NewTypeError("missing numStreams.MIS")
	}

	return
}

/**
 * Validates SctpParameters. It may modify given data by adding missing
 * fields with default values.
 * It throws if invalid.
 */
func validateSctpParameters(params SctpParameters) (err error) {
	// port is mandatory.
	if params.Port == 0 {
		return NewTypeError("missing params.port")
	}

	// OS is mandatory.
	if params.OS == 0 {
		return NewTypeError("missing params.OS")
	}
	// MIS is mandatory.
	if params.MIS == 0 {
		return NewTypeError("missing params.MIS")
	}

	// maxMessageSize is mandatory.
	if params.MaxMessageSize == 0 {
		return NewTypeError("missing params.maxMessageSize")
	}

	return
}

/**
 * Validates SctpStreamParameters. It may modify given data by adding missing
 * fields with default values.
 */
func validateSctpStreamParameters(params *SctpStreamParameters) (err error) {
	if params == nil {
		return NewTypeError("params is nil")
	}
	orderedGiven := params.Ordered != nil

	if params.Ordered == nil {
		params.Ordered = Bool(true)
	}

	if params.MaxPacketLifeTime > 0 && params.MaxRetransmits > 0 {
		return NewTypeError("cannot provide both maxPacketLifeTime and maxRetransmits")
	}

	if orderedGiven && *params.Ordered &&
		(params.MaxPacketLifeTime > 0 || params.MaxRetransmits > 0) {
		return NewTypeError("cannot be ordered with maxPacketLifeTime or maxRetransmits")
	} else if !orderedGiven && (params.MaxPacketLifeTime > 0 || params.MaxRetransmits > 0) {
		params.Ordered = Bool(false)
	}

	return
}

/**
 * Generate RTP capabilities for the Router based on the given media codecs and
 * mediasoup supported RTP capabilities.
 */
func generateRouterRtpCapabilities(mediaCodecs []*RtpCodecCapability) (caps RtpCapabilities, err error) {
	if len(mediaCodecs) == 0 {
		err = NewTypeError("mediaCodecs must be an Array")
		return
	}

	clonedSupportedRtpCapabilities := GetSupportedRtpCapabilities()
	supportedCodecs := clonedSupportedRtpCapabilities.Codecs

	caps.HeaderExtensions = clonedSupportedRtpCapabilities.HeaderExtensions

	dynamicPayloadTypes := make([]byte, len(DYNAMIC_PAYLOAD_TYPES))
	copy(dynamicPayloadTypes, DYNAMIC_PAYLOAD_TYPES[:])

	for _, mediaCodec := range mediaCodecs {
		if err = validateRtpCodecCapability(mediaCodec); err != nil {
			return
		}
		matchedSupportedCodec, matched := findMatchedCodec(mediaCodec, supportedCodecs, matchOptions{})

		if !matched {
			err = NewUnsupportedError(`media codec not supported [mimeType:%s]`, mediaCodec.MimeType)
			return
		}
		codec := &RtpCodecCapability{}

		if err = clone(matchedSupportedCodec, codec); err != nil {
			return
		}

		if mediaCodec.PreferredPayloadType > 0 {
			codec.PreferredPayloadType = mediaCodec.PreferredPayloadType

			idx := bytes.IndexByte(dynamicPayloadTypes, codec.PreferredPayloadType)

			if idx > -1 {
				dynamicPayloadTypes = append(dynamicPayloadTypes[:idx], dynamicPayloadTypes[idx+1:]...)
			}
		} else if codec.PreferredPayloadType == 0 {
			if len(dynamicPayloadTypes) == 0 {
				err = errors.New("cannot allocate more dynamic codec payload types")
				return
			}
			codec.PreferredPayloadType = dynamicPayloadTypes[0]
			dynamicPayloadTypes = dynamicPayloadTypes[1:]
		}

		for _, capCodec := range caps.Codecs {
			if capCodec.PreferredPayloadType == codec.PreferredPayloadType {
				err = NewTypeError("duplicated codec.preferredPayloadType")
				return
			}
		}

		// Merge the media codec parameters.
		override(&codec.Parameters, mediaCodec.Parameters)

		// Append to the codec list.
		caps.Codecs = append(caps.Codecs, codec)

		// Add a RTX video codec if video.
		if codec.Kind == MediaKind_Video {
			if len(dynamicPayloadTypes) == 0 {
				err = errors.New("cannot allocate more dynamic codec payload types")
				return
			}
			pt := dynamicPayloadTypes[0]
			dynamicPayloadTypes = dynamicPayloadTypes[1:]

			rtxCodec := &RtpCodecCapability{
				Kind:                 codec.Kind,
				MimeType:             fmt.Sprintf("%s/rtx", codec.Kind),
				PreferredPayloadType: pt,
				ClockRate:            codec.ClockRate,
				Parameters: RtpCodecSpecificParameters{
					Apt: codec.PreferredPayloadType,
				},
				RtcpFeedback: []RtcpFeedback{},
			}

			// Append to the codec list.
			caps.Codecs = append(caps.Codecs, rtxCodec)
		}
	}

	return
}

/**
 * Get a mapping of the codec payload, RTP header extensions and encodings from
 * the given Producer RTP parameters to the values expected by the Router.
 *
 */
func getProducerRtpParametersMapping(params RtpParameters, caps RtpCapabilities) (rtpMapping RtpMapping, err error) {
	// Match parameters media codecs to capabilities media codecs.
	codecToCapCodec := map[*RtpCodecParameters]*RtpCodecCapability{}

	for _, codec := range params.Codecs {
		if codec.isRtxCodec() {
			continue
		}
		matchedCapCodec, matched := findMatchedCodec(codec, caps.Codecs, matchOptions{strict: true, modify: true})

		if !matched {
			err = NewUnsupportedError("unsupported codec [mimeType:%s, payloadType:%d]", codec.MimeType, codec.PayloadType)
			return
		}

		codecToCapCodec[codec] = matchedCapCodec
	}

	for _, codec := range params.Codecs {
		if !codec.isRtxCodec() {
			continue
		}
		var associatedMediaCodec *RtpCodecParameters

		// Search for the associated media codec.
		for _, mediaCodec := range params.Codecs {
			if mediaCodec.PayloadType == codec.Parameters.Apt {
				associatedMediaCodec = mediaCodec
				break
			}
		}

		if associatedMediaCodec == nil {
			err = NewTypeError(`missing media codec found for RTX PT %d`, codec.PayloadType)
			return
		}
		capMediaCodec := codecToCapCodec[associatedMediaCodec]

		var associatedCapRtxCodec *RtpCodecCapability

		// Ensure that the capabilities media codec has a RTX codec.
		for _, capCodec := range caps.Codecs {
			if !capCodec.isRtxCodec() {
				continue
			}
			if capCodec.Parameters.Apt == capMediaCodec.PreferredPayloadType {
				associatedCapRtxCodec = capCodec
				break
			}
		}

		if associatedCapRtxCodec == nil {
			err = NewUnsupportedError("no RTX codec for capability codec PT %d", capMediaCodec.PreferredPayloadType)
			return
		}

		codecToCapCodec[codec] = associatedCapRtxCodec
	}

	// Generate codecs mapping.
	for codec, capCodec := range codecToCapCodec {
		rtpMapping.Codecs = append(rtpMapping.Codecs, RtpMappingCodec{
			PayloadType:       codec.PayloadType,
			MappedPayloadType: capCodec.PreferredPayloadType,
		})
	}

	// Generate encodings mapping.
	mappedSsrc := generateRandomNumber()

	for _, encoding := range params.Encodings {
		mappedEncoding := RtpMappingEncoding{
			Rid:             encoding.Rid,
			Ssrc:            encoding.Ssrc,
			MappedSsrc:      mappedSsrc,
			ScalabilityMode: encoding.ScalabilityMode,
		}
		mappedSsrc++

		rtpMapping.Encodings = append(rtpMapping.Encodings, mappedEncoding)
	}

	return
}

/**
 * Generate RTP parameters for Consumers given the RTP parameters of a Producer
 * and the RTP capabilities of the Router.
 *
 */
func getConsumableRtpParameters(
	kind MediaKind,
	params RtpParameters,
	caps RtpCapabilities,
	rtpMapping RtpMapping,
) (consumableParams RtpParameters, err error) {
	for _, codec := range params.Codecs {
		if codec.isRtxCodec() {
			continue
		}
		var consumableCodecPt byte

		for _, entry := range rtpMapping.Codecs {
			if entry.PayloadType == codec.PayloadType {
				consumableCodecPt = entry.MappedPayloadType
				break
			}
		}
		var matchedCapCodec *RtpCodecCapability

		for _, capCodec := range caps.Codecs {
			if capCodec.PreferredPayloadType == consumableCodecPt {
				matchedCapCodec = capCodec
				break
			}
		}
		consumableCodec := &RtpCodecParameters{
			MimeType:     matchedCapCodec.MimeType,
			ClockRate:    matchedCapCodec.ClockRate,
			Channels:     matchedCapCodec.Channels,
			RtcpFeedback: matchedCapCodec.RtcpFeedback,
			Parameters:   codec.Parameters, // Keep the Producer parameters.
			PayloadType:  matchedCapCodec.PreferredPayloadType,
		}
		consumableCodec.Parameters = codec.Parameters // Keep the Producer parameters.

		consumableParams.Codecs = append(consumableParams.Codecs, consumableCodec)

		var consumableCapRtxCodec *RtpCodecCapability

		for _, capRtxCodec := range caps.Codecs {
			if capRtxCodec.isRtxCodec() &&
				capRtxCodec.Parameters.Apt == consumableCodec.PayloadType {
				consumableCapRtxCodec = capRtxCodec
				break
			}
		}

		if consumableCapRtxCodec != nil {
			consumableRtxCodec := &RtpCodecParameters{
				MimeType:     consumableCapRtxCodec.MimeType,
				ClockRate:    consumableCapRtxCodec.ClockRate,
				Channels:     consumableCapRtxCodec.Channels,
				RtcpFeedback: consumableCapRtxCodec.RtcpFeedback,
				Parameters:   consumableCapRtxCodec.Parameters,
				PayloadType:  consumableCapRtxCodec.PreferredPayloadType,
			}

			consumableParams.Codecs = append(consumableParams.Codecs, consumableRtxCodec)
		}
	}

	for _, capExt := range caps.HeaderExtensions {
		if capExt.Kind != kind ||
			(capExt.Direction != Direction_Sendrecv && capExt.Direction != Direction_Sendonly) {
			continue
		}
		consumableExt := RtpHeaderExtensionParameters{
			Uri:     capExt.Uri,
			Id:      capExt.PreferredId,
			Encrypt: capExt.PreferredEncrypt,
		}

		consumableParams.HeaderExtensions = append(consumableParams.HeaderExtensions, consumableExt)
	}

	for i, encoding := range params.Encodings {
		// Remove useless fields.
		encoding.Rid = ""
		encoding.Rtx = nil
		encoding.CodecPayloadType = 0

		// Set the mapped ssrc.
		encoding.Ssrc = rtpMapping.Encodings[i].MappedSsrc

		consumableParams.Encodings = append(consumableParams.Encodings, encoding)
	}

	consumableParams.Rtcp = RtcpParameters{
		Cname:       params.Rtcp.Cname,
		ReducedSize: Bool(true),
		Mux:         Bool(true),
	}

	return
}

/**
 * Check whether the given RTP capabilities can consume the given Producer.
 *
 */
func canConsume(consumableParams RtpParameters, caps RtpCapabilities) (ok bool, err error) {
	if err = validateRtpCapabilities(&caps); err != nil {
		return
	}

	var matchingCodecs []*RtpCodecCapability

	for _, codec := range consumableParams.Codecs {
		matchedCodec, matched := findMatchedCodec(codec, caps.Codecs, matchOptions{strict: true})

		if !matched {
			continue
		}

		matchingCodecs = append(matchingCodecs, matchedCodec)
	}

	// Ensure there is at least one media codec.
	if len(matchingCodecs) == 0 || matchingCodecs[0].isRtxCodec() {
		return
	}

	return true, nil
}

/**
 * Generate RTP parameters for a specific Consumer.
 *
 * It reduces encodings to just one and takes into account given RTP capabilities
 * to reduce codecs, codecs" RTCP feedback and header extensions, and also enables
 * or disabled RTX.
 *
 */
func getConsumerRtpParameters(consumableParams RtpParameters, caps RtpCapabilities, pipe bool) (consumerParams RtpParameters, err error) {
	for _, capCodec := range caps.Codecs {
		if err = validateRtpCodecCapability(capCodec); err != nil {
			return
		}
	}

	consumableCodecs := []*RtpCodecParameters{}
	rtxSupported := false

	clone(consumableParams.Codecs, &consumableCodecs)

	for _, codec := range consumableCodecs {
		matchedCapCodec, matched := findMatchedCodec(codec, caps.Codecs, matchOptions{strict: true})

		if !matched {
			continue
		}

		codec.RtcpFeedback = matchedCapCodec.RtcpFeedback

		consumerParams.Codecs = append(consumerParams.Codecs, codec)
	}

	codecs := consumerParams.Codecs[:0]

	// Must sanitize the list of matched codecs by removing useless RTX codecs.
	for _, codec := range consumerParams.Codecs {
		if codec.isRtxCodec() {
			// Search for the associated media codec.
			for _, mediaCodec := range consumerParams.Codecs {
				if mediaCodec.PayloadType == codec.Parameters.Apt {
					rtxSupported = true
					codecs = append(codecs, codec)
					break
				}
			}
		} else {
			codecs = append(codecs, codec)
		}
	}

	consumerParams.Codecs = codecs

	// Ensure there is at least one media codec.
	if len(consumerParams.Codecs) == 0 || consumerParams.Codecs[0].isRtxCodec() {
		err = NewUnsupportedError("no compatible media codecs")
		return
	}

	for _, ext := range consumableParams.HeaderExtensions {
		for _, capExt := range caps.HeaderExtensions {
			if capExt.PreferredId == ext.Id && capExt.Uri == ext.Uri {
				consumerParams.HeaderExtensions = append(consumerParams.HeaderExtensions, ext)
				break
			}
		}
	}

	// Reduce codecs' RTCP feedback. Use Transport-CC if available, REMB otherwise.
	if matchHeaderExtensionUri(consumerParams.HeaderExtensions,
		"http://www.ietf.org/id/draft-holmer-rmcat-transport-wide-cc-extensions-01") {
		for i := range consumerParams.Codecs {
			consumerParams.Codecs[i].RtcpFeedback = filterRtcpFeedback(
				consumerParams.Codecs[i].RtcpFeedback,
				func(fb RtcpFeedback) bool {
					return fb.Type != "goog-remb"
				},
			)
		}
	} else if matchHeaderExtensionUri(consumerParams.HeaderExtensions,
		"http://www.webrtc.org/experiments/rtp-hdrext/abs-send-time") {
		for i := range consumerParams.Codecs {
			consumerParams.Codecs[i].RtcpFeedback = filterRtcpFeedback(
				consumerParams.Codecs[i].RtcpFeedback,
				func(fb RtcpFeedback) bool {
					return fb.Type != "transport-cc"
				},
			)
		}
	} else {
		for i := range consumerParams.Codecs {
			consumerParams.Codecs[i].RtcpFeedback = filterRtcpFeedback(
				consumerParams.Codecs[i].RtcpFeedback,
				func(fb RtcpFeedback) bool {
					return fb.Type != "transport-cc" && fb.Type != "goog-remb"
				},
			)
		}
	}

	if pipe {
		var consumableEncodings []RtpEncodingParameters

		clone(consumableParams.Encodings, &consumableEncodings)

		baseSsrc := generateRandomNumber()
		baseRtxSsrc := generateRandomNumber()

		for i := 0; i < len(consumableEncodings); i++ {
			encoding := consumableEncodings[i]

			encoding.Ssrc = baseSsrc + uint32(i)
			if rtxSupported {
				encoding.Rtx = &RtpEncodingRtx{Ssrc: baseRtxSsrc + uint32(i)}
			} else {
				encoding.Rtx = nil
			}

			consumerParams.Encodings = append(consumerParams.Encodings, encoding)
		}

		return
	}

	consumerEncoding := RtpEncodingParameters{
		Ssrc: generateRandomNumber(),
	}

	if rtxSupported {
		consumerEncoding.Rtx = &RtpEncodingRtx{
			Ssrc: generateRandomNumber(),
		}
	}

	var scalabilityMode string

	// If any of the consumableParams.encodings has scalabilityMode, process it
	// (assume all encodings have the same value).
	for _, encoding := range consumableParams.Encodings {
		if len(encoding.ScalabilityMode) > 0 {
			scalabilityMode = encoding.ScalabilityMode
			break
		}
	}

	// If there is simulast, mangle spatial layers in scalabilityMode.
	if len(consumableParams.Encodings) > 1 {
		temporalLayers := ParseScalabilityMode(scalabilityMode).TemporalLayers
		scalabilityMode = fmt.Sprintf("S%dT%d", len(consumableParams.Encodings), temporalLayers)
	}

	consumerEncoding.ScalabilityMode = scalabilityMode

	maxEncodingMaxBitrate := 0

	// Use the maximum maxBitrate in any encoding and honor it in the Consumer's encoding.
	for _, encoding := range consumableParams.Encodings {
		if encoding.MaxBitrate > maxEncodingMaxBitrate {
			maxEncodingMaxBitrate = encoding.MaxBitrate
		}
	}

	if maxEncodingMaxBitrate > 0 {
		consumerEncoding.MaxBitrate = maxEncodingMaxBitrate
	}

	// Set a single encoding for the Consumer.
	consumerParams.Encodings = append(consumerParams.Encodings, consumerEncoding)

	// Copy verbatim.
	consumerParams.Rtcp = consumableParams.Rtcp

	return
}

/**
 * Generate RTP parameters for a pipe Consumer.
 *
 * It keeps all original consumable encodings and removes support for BWE. If
 * enableRtx is false, it also removes RTX and NACK support.
 */
func getPipeConsumerRtpParameters(consumableParams RtpParameters, enableRtx bool) (consumerParams RtpParameters) {
	consumerParams.Rtcp = consumableParams.Rtcp

	consumableCodecs := []*RtpCodecParameters{}
	clone(consumableParams.Codecs, &consumableCodecs)

	for _, codec := range consumableCodecs {
		if !enableRtx && codec.isRtxCodec() {
			continue
		}

		codec.RtcpFeedback = filterRtcpFeedback(codec.RtcpFeedback, func(fb RtcpFeedback) bool {
			return (fb.Type == "nack" && fb.Parameter == "pli") ||
				(fb.Type == "ccm" && fb.Parameter == "fir") ||
				(enableRtx && fb.Type == "nack" && len(fb.Parameter) == 0)
		})
		consumerParams.Codecs = append(consumerParams.Codecs, codec)
	}

	// Reduce RTP extensions by disabling transport MID and BWE related ones.
	for _, ext := range consumableParams.HeaderExtensions {
		if ext.Uri != "urn:ietf:params:rtp-hdrext:sdes:mid" &&
			ext.Uri != "http://www.webrtc.org/experiments/rtp-hdrext/abs-send-time" &&
			ext.Uri != "http://www.ietf.org/id/draft-holmer-rmcat-transport-wide-cc-extensions-01" {
			consumerParams.HeaderExtensions = append(consumerParams.HeaderExtensions, ext)
		}
	}

	consumableEncodings := []RtpEncodingParameters{}
	clone(consumableParams.Encodings, &consumableEncodings)

	baseSsrc := generateRandomNumber()
	baseRtxSsrc := generateRandomNumber()

	for i, encoding := range consumableEncodings {
		encoding.Ssrc = baseSsrc + uint32(i)

		if enableRtx {
			encoding.Rtx = &RtpEncodingRtx{Ssrc: baseRtxSsrc + uint32(i)}
		} else {
			encoding.Rtx = nil
		}

		consumerParams.Encodings = append(consumerParams.Encodings, encoding)
	}

	return
}

func findMatchedCodec(aCodec interface{}, bCodecs []*RtpCodecCapability, options matchOptions) (codec *RtpCodecCapability, matched bool) {
	var rtpCodecParameters *RtpCodecParameters

	switch aCodec.(type) {
	case *RtpCodecCapability:
		cap := aCodec.(*RtpCodecCapability)
		rtpCodecParameters = &RtpCodecParameters{
			MimeType:   cap.MimeType,
			ClockRate:  cap.ClockRate,
			Channels:   cap.Channels,
			Parameters: cap.Parameters,
		}
	case *RtpCodecParameters:
		rtpCodecParameters = aCodec.(*RtpCodecParameters)
	}

	for _, bCodec := range bCodecs {
		if matchedCodecs(rtpCodecParameters, bCodec, options) {
			return bCodec, true
		}
	}
	return
}

func matchedCodecs(aCodec *RtpCodecParameters, bCodec *RtpCodecCapability, options matchOptions) (matched bool) {
	aMimeType := strings.ToLower(aCodec.MimeType)
	bMimeType := strings.ToLower(bCodec.MimeType)

	if aMimeType != bMimeType {
		return
	}

	if aCodec.ClockRate != bCodec.ClockRate {
		return
	}

	if strings.HasPrefix(aMimeType, "audio/") &&
		aCodec.Channels > 0 &&
		bCodec.Channels > 0 &&
		aCodec.Channels != bCodec.Channels {
		return
	}

	switch aMimeType {
	case "video/h264":
		aParameters, bParameters := aCodec.Parameters, bCodec.Parameters

		if aParameters.PacketizationMode != bParameters.PacketizationMode {
			return
		}

		if options.strict {
			selectedProfileLevelId, err := h264.GenerateProfileLevelIdForAnswer(
				aParameters.RtpParameter, bParameters.RtpParameter)
			if err != nil {
				return
			}

			if options.modify {
				aParameters.ProfileLevelId = selectedProfileLevelId
				aCodec.Parameters = aParameters
			}
		}
	}

	return true
}

func matchHeaderExtensionUri(exts []RtpHeaderExtensionParameters, uri string) bool {
	for _, ext := range exts {
		if ext.Uri == uri {
			return true
		}
	}

	return false
}

func filterRtcpFeedback(arr []RtcpFeedback, cond func(RtcpFeedback) bool) []RtcpFeedback {
	newArr := arr[:0]

	for _, x := range arr {
		if cond(x) {
			newArr = append(newArr, x)
		}
	}

	return newArr
}
