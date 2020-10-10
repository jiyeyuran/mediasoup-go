package mediasoup

import (
	"errors"
	"fmt"
	"strings"

	"github.com/imdario/mergo"
	"github.com/jinzhu/copier"
	"github.com/jiyeyuran/mediasoup/h264"
)

var DYNAMIC_PAYLOAD_TYPES = [...]int{
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
	PayloadType       int `json:"payloadType"`
	MappedPayloadType int `json:"mappedPayloadType"`
}

type RtpMappingEncoding struct {
	Ssrc            uint32 `json:"ssrc,omitempty"`
	Rid             string `json:"rid,omitempty"`
	ScalabilityMode string `json:"scalabilityMode,omitempty"`
	MappedSsrc      uint32 `json:"mappedSsrc"`
}

/**
 * Generate RTP capabilities for the Router based on the given media codecs and
 * mediasoup supported RTP capabilities.
 */
func generateRouterRtpCapabilities(mediaCodecs []RtpCodecCapability) (caps RtpCapabilities, err error) {
	supportedRtpCapabilities := GetSupportedRtpCapabilities()
	supportedCodecs := supportedRtpCapabilities.Codecs

	caps.HeaderExtensions = supportedRtpCapabilities.HeaderExtensions
	caps.FecMechanisms = supportedRtpCapabilities.FecMechanisms

	dynamicPayloadTypeIdx := 0

	for _, mediaCodec := range mediaCodecs {
		if err = checkCodecCapability(&mediaCodec); err != nil {
			return
		}

		codec, matched := findMatchedCodec(&mediaCodec, supportedCodecs, matchOptions{})

		if !matched {
			err = NewUnsupportedError(
				fmt.Sprintf(`media codec not supported [mimeType:%s]`, mediaCodec.MimeType))
			return
		}

		// Normalize channels.
		if codec.Kind != "audio" {
			codec.Channels = 0
		} else if codec.Channels == 0 {
			codec.Channels = 1
		}

		// Merge the media codec parameters.
		if codec.Parameters == nil {
			codec.Parameters = &RtpCodecSpecificParameters{}
		}
		if mediaCodec.Parameters != nil {
			mergo.Merge(codec.Parameters, mediaCodec.Parameters, mergo.WithOverride)
		}

		// Make rtcpFeedback an array.
		if codec.RtcpFeedback == nil {
			codec.RtcpFeedback = []RtcpFeedback{}
		}

		// Assign a payload type.
		if codec.PreferredPayloadType == 0 {
			if dynamicPayloadTypeIdx >= len(DYNAMIC_PAYLOAD_TYPES) {
				err = errors.New("cannot allocate more dynamic codec payload types")
				return
			}

			codec.PreferredPayloadType = DYNAMIC_PAYLOAD_TYPES[dynamicPayloadTypeIdx]

			dynamicPayloadTypeIdx++
		}

		// Append to the codec list.
		caps.Codecs = append(caps.Codecs, codec)

		// Add a RTX video codec if video.
		if codec.Kind == "video" {
			if dynamicPayloadTypeIdx >= len(DYNAMIC_PAYLOAD_TYPES) {
				err = errors.New("cannot allocate more dynamic codec payload types")
				return
			}
			pt := DYNAMIC_PAYLOAD_TYPES[dynamicPayloadTypeIdx]

			rtxCodec := RtpCodecCapability{
				Kind:                 codec.Kind,
				MimeType:             fmt.Sprintf("%s/rtx", codec.Kind),
				PreferredPayloadType: pt,
				ClockRate:            codec.ClockRate,
				RtcpFeedback:         []RtcpFeedback{},
				Parameters: &RtpCodecSpecificParameters{
					Apt: codec.PreferredPayloadType,
				},
			}

			dynamicPayloadTypeIdx++

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
	codecToCapCodec := map[*RtpCodecParameters]RtpCodecCapability{}

	for i, codec := range params.Codecs {
		if err = checkCodecParameters(codec); err != nil {
			return
		}

		if strings.HasSuffix(strings.ToLower(codec.MimeType), "/rtx") {
			continue
		}

		matchedCapCodec, matched := findMatchedCodec(
			&codec, caps.Codecs, matchOptions{strict: true, modify: true})

		if !matched {
			err = NewUnsupportedError(
				"unsupported codec [mimeType:%s, payloadType:%d]",
				codec.MimeType, codec.PayloadType,
			)
		}

		codecToCapCodec[&params.Codecs[i]] = matchedCapCodec
	}

	for i, codec := range params.Codecs {
		if !strings.HasSuffix(strings.ToLower(codec.MimeType), "/rtx") {
			continue
		}

		if codec.Parameters == nil {
			err = NewTypeError("missing parameters in RTX codec")
			return
		}

		var associatedMediaCodec *RtpCodecParameters

		for i, mediaCodec := range params.Codecs {
			if mediaCodec.PayloadType == codec.Parameters.Apt {
				associatedMediaCodec = &params.Codecs[i]
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
			if !strings.HasSuffix(strings.ToLower(capCodec.MimeType), "/rtx") {
				continue
			}
			if capCodec.Parameters.Apt == capMediaCodec.PreferredPayloadType {
				associatedCapRtxCodec = &capCodec
				break
			}
		}

		if associatedCapRtxCodec == nil {
			err = NewUnsupportedError(
				"no RTX codec for capability codec PT %d",
				capMediaCodec.PreferredPayloadType,
			)
			return
		}

		codecToCapCodec[&params.Codecs[i]] = *associatedCapRtxCodec
	}

	// Generate codecs mapping.
	for codec, capCodec := range codecToCapCodec {
		rtpMapping.Codecs = append(rtpMapping.Codecs, RtpMappingCodec{
			PayloadType:       codec.PayloadType,
			MappedPayloadType: capCodec.PreferredPayloadType,
		})
	}

	// Generate encodings mapping.
	for _, encoding := range params.Encodings {
		mappedEncoding := RtpMappingEncoding{
			Rid:        encoding.Rid,
			Ssrc:       encoding.Ssrc,
			MappedSsrc: generateRandomNumber(),
		}

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
		if err = checkCodecParameters(codec); err != nil {
			return
		}

		if strings.HasSuffix(strings.ToLower(codec.MimeType), "/rtx") {
			continue
		}

		consumableCodecPt := 0

		for _, entry := range rtpMapping.Codecs {
			if entry.PayloadType == codec.PayloadType {
				consumableCodecPt = entry.MappedPayloadType
				break
			}
		}

		var matchedCapCodec RtpCodecCapability

		for _, capCodec := range caps.Codecs {
			if capCodec.PreferredPayloadType == consumableCodecPt {
				matchedCapCodec = capCodec
				break
			}
		}
		consumableCodec := RtpCodecParameters{
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
			if strings.HasSuffix(strings.ToLower(capRtxCodec.MimeType), "/rtx") &&
				capRtxCodec.Parameters.Apt == consumableCodec.PayloadType {
				consumableCapRtxCodec = &capRtxCodec
				break
			}
		}

		if consumableCapRtxCodec != nil {
			consumableRtxCodec := RtpCodecParameters{
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
			capExt.Uri == "urn:ietf:params:rtp-hdrext:sdes:mid" ||
			capExt.Uri == "urn:ietf:params:rtp-hdrext:sdes:rtp-stream-id" ||
			capExt.Uri == "urn:ietf:params:rtp-hdrext:sdes:repaired-rtp-stream-id" {
			continue
		}

		consumableExt := RtpHeaderExtensionParameters{
			Uri: capExt.Uri,
			Id:  capExt.PreferredId,
		}

		consumableParams.HeaderExtensions = append(
			consumableParams.HeaderExtensions, consumableExt)
	}

	for i, encoding := range params.Encodings {
		encoding.Rid = ""
		encoding.Rtx = nil
		encoding.CodecPayloadType = 0
		encoding.Ssrc = rtpMapping.Encodings[i].MappedSsrc

		consumableParams.Encodings = append(consumableParams.Encodings, encoding)
	}

	consumableParams.Rtcp = &RtcpParameters{
		Cname:       params.Rtcp.Cname,
		ReducedSize: true,
		Mux:         newBool(true),
	}

	return
}

/**
 * Check whether the given RTP capabilities can consume the given Producer.
 *
 */
func canConsume(consumableParams RtpParameters, caps RtpCapabilities) bool {
	capCodecs := []RtpCodecCapability{}

	for _, capCodec := range caps.Codecs {
		if checkCodecCapability(&capCodec) != nil {
			return false
		}
		capCodecs = append(capCodecs, capCodec)
	}

	var matchingCodecs []RtpCodecCapability

	for _, codec := range consumableParams.Codecs {
		matchedCodec, matched := findMatchedCodec(&codec, capCodecs, matchOptions{strict: true})

		if !matched {
			continue
		}

		matchingCodecs = append(matchingCodecs, matchedCodec)
	}

	// Ensure there is at least one media codec.
	if len(matchingCodecs) == 0 ||
		strings.HasSuffix(matchingCodecs[0].MimeType, "/rtx") {
		return false
	}

	return true
}

/**
 * Generate RTP parameters for a specific Consumer.
 *
 * It reduces encodings to just one and takes into account given RTP capabilities
 * to reduce codecs, codecs" RTCP feedback and header extensions, and also enables
 * or disabled RTX.
 *
 */
func getConsumerRtpParameters(consumableParams RtpParameters, caps RtpCapabilities) (consumerParams RtpParameters, err error) {
	consumerParams.HeaderExtensions = []RtpHeaderExtensionParameters{}

	for _, capCodec := range caps.Codecs {
		if err = checkCodecCapability(&capCodec); err != nil {
			return
		}
	}

	consumableCodecs, rtxSupported := []RtpCodecParameters{}, false

	clone(consumableParams.Codecs, &consumableCodecs)

	for _, codec := range consumableCodecs {
		matchedCapCodec, matched := findMatchedCodec(&codec, caps.Codecs, matchOptions{strict: true})

		if !matched {
			continue
		}

		codec.RtcpFeedback = matchedCapCodec.RtcpFeedback

		consumerParams.Codecs = append(consumerParams.Codecs, codec)

		if !rtxSupported && strings.HasSuffix(codec.MimeType, "/rtx") {
			rtxSupported = true
		}
	}

	// Ensure there is at least one media codec.
	if len(consumerParams.Codecs) == 0 ||
		strings.HasSuffix(consumerParams.Codecs[0].MimeType, "/rtx") {
		err = NewUnsupportedError("no compatible media codecs")
		return
	}

	for _, ext := range consumableParams.HeaderExtensions {
		for _, capExt := range caps.HeaderExtensions {
			if capExt.PreferredId == ext.Id {
				consumerParams.HeaderExtensions =
					append(consumerParams.HeaderExtensions, ext)
				break
			}
		}
	}

	consumerEncoding := RtpEncodingParameters{
		Ssrc: generateRandomNumber(),
	}

	if rtxSupported {
		consumerEncoding.Rtx = &RtpEncodingRtx{
			Ssrc: generateRandomNumber(),
		}
	}

	consumerParams.Encodings = append(consumerParams.Encodings, consumerEncoding)
	consumerParams.Rtcp = consumableParams.Rtcp

	return
}

/**
 * Generate RTP parameters for a pipe Consumer.
 *
 * It keeps all original consumable encodings, removes RTX support and also
 * other features such as NACK.
 *
 * @param {RTCRtpParameters} consumableParams - Consumable RTP parameters.
 *
 * @returns {RTCRtpParameters}
 * @throws {TypeError} if wrong arguments.
 */
func getPipeConsumerRtpParameters(consumableParams RtpParameters) (consumerParams RtpParameters) {
	consumerParams.Rtcp = consumableParams.Rtcp

	consumableCodecs := []RtpCodecParameters{}
	copier.Copy(&consumableCodecs, &consumableParams.Codecs)

	for _, codec := range consumableCodecs {
		if strings.HasSuffix(strings.ToLower(codec.MimeType), "/rtx") {
			continue
		}

		var rtcpFeedback []RtcpFeedback

		for _, fb := range codec.RtcpFeedback {
			if (fb.Type == "nack" && fb.Parameter == "pli") ||
				(fb.Type == "ccm" && fb.Parameter == "fir") {
				rtcpFeedback = append(rtcpFeedback, fb)
			}
		}

		codec.RtcpFeedback = rtcpFeedback
		consumerParams.Codecs = append(consumerParams.Codecs, codec)
	}

	// Reduce RTP header extensions.
	for _, ext := range consumableParams.HeaderExtensions {
		if ext.Uri != "http://www.webrtc.org/experiments/rtp-hdrext/abs-send-time" {
			consumerParams.HeaderExtensions = append(consumerParams.HeaderExtensions, ext)
		}
	}

	consumableEncodings := []RtpEncodingParameters{}
	copier.Copy(&consumableEncodings, &consumableParams.Encodings)

	for _, encoding := range consumableEncodings {
		encoding.Rtx = nil

		consumerParams.Encodings = append(consumerParams.Encodings, encoding)
	}

	return
}

func checkCodecCapability(codec *RtpCodecCapability) (err error) {
	if len(codec.MimeType) == 0 || codec.ClockRate == 0 {
		return NewTypeError("invalid RTCRtpCodecCapability")
	}

	// Add kind if not present.
	if len(codec.Kind) == 0 {
		codec.Kind = MediaKind(strings.ToLower(strings.Split(codec.MimeType, "/")[0]))
	}

	return
}

func checkCodecParameters(codec RtpCodecParameters) error {
	if len(codec.MimeType) == 0 || codec.ClockRate == 0 {
		return NewTypeError("invalid RTCRtpCodecParameters")
	}
	return nil
}

func findMatchedCodec(aCodec interface{}, bCodecs []RtpCodecCapability, options matchOptions) (codec RtpCodecCapability, matched bool) {
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

func matchedCodecs(aCodec *RtpCodecParameters, bCodec RtpCodecCapability, options matchOptions) (matched bool) {
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
		if aParameters == nil {
			aParameters = &RtpCodecSpecificParameters{}
		}
		if bParameters == nil {
			bParameters = &RtpCodecSpecificParameters{}
		}

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
