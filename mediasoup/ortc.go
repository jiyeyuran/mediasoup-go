package mediasoup

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jiyeyuran/mediasoup-go/mediasoup/h264profile"
)

var DYNAMIC_PAYLOAD_TYPES = [...]int{
	100, 101, 102, 103, 104, 105, 106, 107, 108, 109, 110, 111, 112, 113, 114, 115,
	116, 117, 118, 119, 120, 121, 122, 123, 124, 125, 126, 127, 96, 97, 98, 99, 77,
	78, 79, 80, 81, 82, 83, 84, 85, 86, 87, 88, 89, 90, 91, 92, 93, 94, 95, 35, 36,
	37, 38, 39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50, 51, 52, 53, 54, 55, 56,
	57, 58, 59, 60, 61, 62, 63, 64, 65, 66, 67, 68, 69, 70, 71,
}

type codecMatchMode int

const (
	codecMatchNormal          = 0
	codecMatchStrict          = 0x01
	codecMatchModify          = 0x10
	codecMatchStrictAndModify = codecMatchStrict | codecMatchModify
)

/**
 * Generate RTP capabilities for the Router based on the given media codecs and
 * mediasoup supported RTP capabilities.
 *
 */
func GenerateRouterRtpCapabilities(mediaCodecs []RtpCodecCapability) (caps RtpCapabilities, err error) {
	if len(mediaCodecs) == 0 {
		err = NewTypeError("mediaCodecs cannot be empty")
		return
	}

	supportedRtpCapabilities := GetSupportedRtpCapabilities()
	supportedCodecs := supportedRtpCapabilities.Codecs

	caps.HeaderExtensions = supportedRtpCapabilities.HeaderExtensions
	caps.FecMechanisms = supportedRtpCapabilities.FecMechanisms

	dynamicPayloadTypeIdx := 0

	for _, mediaCodec := range mediaCodecs {
		if err = checkCodecCapability(&mediaCodec); err != nil {
			return
		}

		codec, matched := selectMatchedCodecs(
			mediaCodec, supportedCodecs, codecMatchNormal)

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
				Parameters: &RtpParameter{
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
func GetProducerRtpParametersMapping(
	params RtpRemoteCapabilities,
	caps RtpCapabilities,
) (rtpMapping RtpRemoteCapabilities, err error) {
	// Match parameters media codecs to capabilities media codecs.
	codecToCapCodec := map[RtpMappingCodec]RtpCodecCapability{}

	for _, codec := range params.Codecs {
		if err = checkCodecParameters(codec.RtpCodecCapability); err != nil {
			return
		}

		if strings.HasSuffix(strings.ToLower(codec.MimeType), "/rtx") {
			continue
		}

		matchedCapCodec, matched := selectMatchedCodecs(
			*codec.RtpCodecCapability, caps.Codecs, codecMatchStrictAndModify)

		if !matched {
			err = fmt.Errorf(
				"unsupported codec [mimeType:%s, payloadType:%d]",
				codec.MimeType, codec.PreferredPayloadType,
			)
		}

		codecToCapCodec[codec] = matchedCapCodec
	}

	for _, codec := range params.Codecs {
		if !strings.HasSuffix(strings.ToLower(codec.MimeType), "/rtx") {
			continue
		}

		if codec.Parameters == nil {
			err = NewTypeError("missing parameters in RTX codec")
			return
		}

		var associatedMediaCodec RtpMappingCodec

		for _, mediaCodec := range params.Codecs {
			if mediaCodec.PayloadType == codec.Parameters.Apt {
				associatedMediaCodec = mediaCodec
				break
			}
		}

		if associatedMediaCodec.RtpCodecCapability == nil {
			err = NewTypeError(`missing media codec found for RTX PT %d`, codec.PayloadType)
			return
		}

		capMediaCodec := codecToCapCodec[associatedMediaCodec]

		var associatedCapRtxCodec RtpCodecCapability

		// Ensure that the capabilities media codec has a RTX codec.
		for _, capCodec := range caps.Codecs {
			if !strings.HasSuffix(strings.ToLower(capCodec.MimeType), "/rtx") {
				continue
			}
			if capCodec.Parameters.Apt == capMediaCodec.PreferredPayloadType {
				associatedCapRtxCodec = capCodec
			}
		}

		if associatedCapRtxCodec.PreferredPayloadType == 0 {
			err = NewUnsupportedError(
				"no RTX codec for capability codec PT %d",
				capMediaCodec.PreferredPayloadType,
			)
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

	// Generate header extensions mapping.
	for _, ext := range params.HeaderExtensions {
		var matchedCapExt *RtpHeaderExtension

		for _, capExt := range caps.HeaderExtensions {
			if matchHeaderExtensions(*ext.RtpHeaderExtension, capExt) {
				matchedCapExt = &capExt
			}
		}

		if matchedCapExt == nil {
			err = NewUnsupportedError(
				`unsupported header extensions [uri:"%s", id:%d]`,
				ext.Uri, ext.Id,
			)

			return
		}

		rtpMapping.HeaderExtensions = append(
			rtpMapping.HeaderExtensions,
			RtpMappingHeaderExt{
				Id:       ext.Id,
				MappedId: matchedCapExt.PreferredId,
			},
		)
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
func GetConsumableRtpParameters(
	kind string,
	params RtpRemoteCapabilities,
	caps RtpCapabilities,
	rtpMapping RtpRemoteCapabilities,
) (consumableParams RtpRemoteCapabilities, err error) {
	for _, codec := range params.Codecs {
		if err = checkCodecParameters(codec.RtpCodecCapability); err != nil {
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
			}
		}

		consumableCodec := RtpMappingCodec{
			RtpCodecCapability: &matchedCapCodec,
			PayloadType:        matchedCapCodec.PreferredPayloadType,
		}
		consumableCodec.Parameters = codec.Parameters // Keep the Producer parameters.

		consumableParams.Codecs = append(consumableParams.Codecs, consumableCodec)

		var consumableCapRtxCodec *RtpCodecCapability

		for _, capRtxCodec := range caps.Codecs {
			if strings.HasSuffix(strings.ToLower(capRtxCodec.MimeType), "/rtx") &&
				capRtxCodec.Parameters.Apt == consumableCodec.PayloadType {
				consumableCapRtxCodec = &capRtxCodec
			}
		}

		if consumableCapRtxCodec != nil {
			consumableRtxCodec := RtpMappingCodec{
				RtpCodecCapability: consumableCapRtxCodec,
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

		consumableExt := RtpMappingHeaderExt{
			RtpHeaderExtension: &RtpHeaderExtension{
				Uri: capExt.Uri,
			},
			Id: capExt.PreferredId,
		}

		consumableParams.HeaderExtensions = append(
			consumableParams.HeaderExtensions, consumableExt)
	}

	for i := range params.Encodings {
		if i < len(rtpMapping.Encodings) {
			consumableParams.Encodings = append(
				consumableParams.Encodings,
				RtpMappingEncoding{
					Ssrc: rtpMapping.Encodings[i].MappedSsrc,
				},
			)
		}
	}

	consumableParams.Rtcp = RtcpConfiguation{
		Cname:       params.Rtcp.Cname,
		ReducedSize: true,
		Mux:         true,
	}

	return
}

/**
 * Check whether the given RTP capabilities can consume the given Producer.
 *
 */
func CanConsume(consumableParams, caps RtpRemoteCapabilities) bool {
	capCodecs := []RtpCodecCapability{}

	for _, capCodec := range caps.Codecs {
		if checkCodecCapability(capCodec.RtpCodecCapability) != nil {
			return false
		}
		capCodecs = append(capCodecs, *capCodec.RtpCodecCapability)
	}

	var matchingCodecs []RtpCodecCapability

	for _, codec := range consumableParams.Codecs {
		codec, matched := selectMatchedCodecs(
			*codec.RtpCodecCapability, capCodecs, codecMatchStrict)

		if !matched {
			continue
		}

		matchingCodecs = append(matchingCodecs, codec)
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
func GetConsumerRtpParameters(
	consumableParams RtpRemoteCapabilities, caps RtpRemoteCapabilities,
) (consumerParams RtpRemoteCapabilities, err error) {
	capCodecs := []RtpCodecCapability{}

	for _, capCodec := range caps.Codecs {
		if err = checkCodecCapability(capCodec.RtpCodecCapability); err != nil {
			return
		}
		capCodecs = append(capCodecs, *capCodec.RtpCodecCapability)
	}

	consumableCodecs := consumableParams.Codecs
	rtxSupported := false

	for _, codec := range consumableCodecs {
		matchedCapCodec, matched := selectMatchedCodecs(
			*codec.RtpCodecCapability, capCodecs, codecMatchStrict)

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

	for _, ext := range consumerParams.HeaderExtensions {
		for _, capExt := range consumableParams.HeaderExtensions {
			if capExt.PreferredId == ext.Id {
				consumerParams.HeaderExtensions =
					append(consumerParams.HeaderExtensions, ext)
			}
		}
	}

	consumerEncoding := RtpMappingEncoding{
		Ssrc: generateRandomNumber(),
	}

	if rtxSupported {
		consumerEncoding.Rtx = &RtpMappingEncoding{
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
func GetPipeConsumerRtpParameters(
	consumableParams RtpRemoteCapabilities,
) (consumerParams RtpRemoteCapabilities) {
	consumerParams.Rtcp = consumableParams.Rtcp

	for _, codec := range consumableParams.Codecs {
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

	for _, encoding := range consumableParams.Encodings {
		consumerParams.Encodings = append(
			consumerParams.Encodings, RtpMappingEncoding{
				Ssrc: encoding.Ssrc,
			},
		)
	}

	return
}

func checkCodecCapability(codec *RtpCodecCapability) (err error) {
	if codec == nil || len(codec.MimeType) == 0 || codec.ClockRate == 0 {
		return NewTypeError("invalid RTCRtpCodecCapability")
	}

	// Add kind if not present.
	if len(codec.Kind) == 0 {
		codec.Kind = strings.ToLower(strings.Split(codec.MimeType, "/")[0])
	}

	return
}

func checkCodecParameters(codec *RtpCodecCapability) error {
	if codec == nil || len(codec.MimeType) > 0 || codec.ClockRate == 0 {
		return NewTypeError("invalid RTCRtpCodecParameters")
	}
	return nil
}

func selectMatchedCodecs(
	aCodec RtpCodecCapability,
	bCodecs []RtpCodecCapability,
	mode codecMatchMode,
) (codec RtpCodecCapability, matched bool) {
	aMimeType := strings.ToLower(aCodec.MimeType)

	for _, bCodec := range bCodecs {
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
			if aCodec.Parameters.PacketizationMode !=
				bCodec.Parameters.PacketizationMode {
				return
			}

			if mode&codecMatchStrict > 0 {
				selectedProfileLevelId, err := h264profile.GenerateProfileLevelIdForAnswer(
					aCodec.Parameters.RtpH264Parameter, bCodec.Parameters.RtpH264Parameter)
				if err != nil {
					return
				}

				if mode&codecMatchModify > 0 {
					aCodec.Parameters.ProfileLevelId = selectedProfileLevelId
				}
			}
		}

		return bCodec, true
	}

	return
}

func matchHeaderExtensions(aExt, bExt RtpHeaderExtension) bool {
	if len(aExt.Kind) > 0 &&
		len(bExt.Kind) > 0 &&
		aExt.Kind != bExt.Kind {
		return false
	}

	return aExt.Uri == bExt.Uri
}
