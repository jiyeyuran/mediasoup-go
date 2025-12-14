package mediasoup

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/jiyeyuran/mediasoup-go/v2/internal/h264"
)

var availablePayloadTypes = [...]uint8{
	100, 101, 102, 103, 104, 105, 106, 107, 108, 109, 110, 111, 112, 113, 114, 115,
	116, 117, 118, 119, 120, 121, 122, 123, 124, 125, 126, 127, 96, 97, 98, 99, 77,
	78, 79, 80, 81, 82, 83, 84, 85, 86, 87, 88, 89, 90, 91, 92, 93, 94, 95, 35, 36,
	37, 38, 39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50, 51, 52, 53, 54, 55, 56,
	57, 58, 59, 60, 61, 62, 63, 64, 65, 66, 67, 68, 69, 70, 71,
}

// TODO: Remove this if we switch to 'sendrecv' in Dependency-Descriptor header
// extension.
var dependencyDescriptorHeaderExtensionParametersForPipeConsumer *RtpHeaderExtensionParameters

func init() {
	// TODO: Remove this if we switch to 'sendrecv' in Dependency-Descriptor header
	// extension.
	//
	// We need to create and store this Dependency-Descriptor header extension to
	// leter be used by `getPipeConsumerRtpParameters()` function.
	for _, ext := range supportedRtpCapabilities.HeaderExtensions {
		if ext.Uri == "https://aomediacodec.github.io/av1-rtp-spec/#dependency-descriptor-rtp-header-extension" &&
			ext.Direction != MediaDirectionSendrecv {
			dependencyDescriptorHeaderExtensionParametersForPipeConsumer = &RtpHeaderExtensionParameters{
				Uri:     ext.Uri,
				Id:      ext.PreferredId,
				Encrypt: ext.PreferredEncrypt,
			}
			break
		}
	}
}

type matchOptions struct {
	strict bool
	modify bool
}

// validateRtpCapabilities validates RtpCapabilities. It may modify given data by adding missing
// fields with default values.
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

// validateRtpCodecCapability validates RtpCodecCapability. It may modify given data by adding
// missing fields with default values.
func validateRtpCodecCapability(codec *RtpCodecCapability) (err error) {
	mimeType := strings.ToLower(codec.MimeType)

	//  mimeType is mandatory.
	if !strings.HasPrefix(mimeType, "audio/") && !strings.HasPrefix(mimeType, "video/") {
		return errors.New("invalid codec.mimeType")
	}

	if len(codec.Kind) == 0 {
		codec.Kind = MediaKind(strings.Split(mimeType, "/")[0])
	}

	// clockRate is mandatory.
	if codec.ClockRate == 0 {
		return errors.New("missing codec.clockRate")
	}

	// channels is optional. If unset, set it to 1 (just if audio).
	if codec.Kind == MediaKindAudio && codec.Channels == 0 {
		codec.Channels = 1
	}

	for _, fb := range codec.RtcpFeedback {
		if err = validateRtcpFeedback(fb); err != nil {
			return
		}
	}

	return
}

// validateRtcpFeedback validates RtcpFeedback. It may modify given data by adding missing
// fields with default values.
func validateRtcpFeedback(fb *RtcpFeedback) error {
	if len(fb.Type) == 0 {
		return errors.New("missing fb.type")
	}
	return nil
}

// validateRtpHeaderExtension validates RtpHeaderExtension. It may modify given data by adding
// missing fields with default values.
func validateRtpHeaderExtension(ext *RtpHeaderExtension) (err error) {
	if len(ext.Kind) > 0 && ext.Kind != MediaKindAudio && ext.Kind != MediaKindVideo {
		return errors.New("invalid ext.kind")
	}

	// uri is mandatory.
	if len(ext.Uri) == 0 {
		return errors.New("missing ext.uri")
	}

	// preferredId is mandatory.
	if ext.PreferredId == 0 {
		return errors.New("missing ext.preferredId")
	}

	// direction is optional. If unset set it to sendrecv.
	if len(ext.Direction) == 0 {
		ext.Direction = MediaDirectionSendrecv
	}

	return
}

// validateRtpParameters validates RtpParameters. It may modify given data by adding missing
// fields with default values.
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

	return validateRtcpParameters(params.Rtcp)
}

// validateRtpCodecParameters validates RtpCodecParameters. It may modify given data by adding
// missing fields with default values.
func validateRtpCodecParameters(code *RtpCodecParameters) (err error) {
	mimeType := strings.ToLower(code.MimeType)

	//  mimeType is mandatory.
	if !strings.HasPrefix(mimeType, "audio/") && !strings.HasPrefix(mimeType, "video/") {
		return errors.New("invalid codec.mimeType")
	}

	// clockRate is mandatory.
	if code.ClockRate == 0 {
		return errors.New("missing codec.clockRate")
	}

	kind := MediaKind(strings.Split(mimeType, "/")[0])

	// channels is optional. If unset, set it to 1 (just if audio).
	if kind == MediaKindAudio && code.Channels == 0 {
		code.Channels = 1
	}

	for _, fb := range code.RtcpFeedback {
		if err = validateRtcpFeedback(fb); err != nil {
			return
		}
	}

	return
}

// validateRtpHeaderExtensionParameters validates RtpHeaderExtension. It may modify given data by
// adding missing fields with default values.
func validateRtpHeaderExtensionParameters(ext *RtpHeaderExtensionParameters) (err error) {
	// uri is mandatory.
	if len(ext.Uri) == 0 {
		return errors.New("missing ext.uri")
	}

	// preferredId is mandatory.
	if ext.Id == 0 {
		return errors.New("missing ext.id")
	}

	return
}

// validateRtcpParameters validates RtcpParameters. It may modify given data by adding missing
// fields with default values.
func validateRtcpParameters(rtcp *RtcpParameters) (err error) {
	// reducedSize is optional. If unset set it to true.
	if rtcp.ReducedSize == nil {
		rtcp.ReducedSize = ref(true)
	}

	return
}

// validateSctpStreamParameters validates SctpStreamParameters. It may modify given data by adding
// missing fields with default values.
func validateSctpStreamParameters(params *SctpStreamParameters) (err error) {
	if params == nil {
		return ErrMissSctpStreamParameters
	}
	orderedGiven := params.Ordered != nil

	if params.Ordered == nil {
		params.Ordered = ref(true)
	}

	if params.MaxPacketLifeTime != nil && params.MaxRetransmits != nil {
		return errors.New("cannot provide both maxPacketLifeTime and maxRetransmits")
	}

	if orderedGiven && *params.Ordered &&
		(params.MaxPacketLifeTime != nil || params.MaxRetransmits != nil) {
		return errors.New("cannot be ordered with maxPacketLifeTime or maxRetransmits")
	} else if !orderedGiven && (params.MaxPacketLifeTime != nil || params.MaxRetransmits != nil) {
		params.Ordered = ref(false)
	}

	return
}

// generateRouterRtpCapabilities generate RTP capabilities for the Router based on the given media
// codecs and mediasoup supported RTP capabilities.
func generateRouterRtpCapabilities(mediaCodecs []*RtpCodecCapability) (caps *RtpCapabilities, err error) {
	caps = &RtpCapabilities{
		HeaderExtensions: supportedRtpCapabilities.HeaderExtensions,
	}

	availablePayloadTypesCopy := availablePayloadTypes
	dynamicPayloadTypes := availablePayloadTypesCopy[:]

	for _, mediaCodec := range mediaCodecs {
		if err = validateRtpCodecCapability(mediaCodec); err != nil {
			return nil, err
		}
		matchedSupportedCodec, matched := findMatchedCodec(
			supportedRtpCapabilities.Codecs,
			mediaCodec,
			matchOptions{},
		)

		if !matched {
			return nil, fmt.Errorf(`media codec not supported [mimeType:%s]`, mediaCodec.MimeType)
		}
		codec := matchedSupportedCodec.clone()

		if mediaCodec.PreferredPayloadType > 0 {
			codec.PreferredPayloadType = mediaCodec.PreferredPayloadType

			idx := -1
			for i, pt := range dynamicPayloadTypes {
				if pt == codec.PreferredPayloadType {
					idx = i
					break
				}
			}
			if idx > -1 {
				dynamicPayloadTypes = append(
					dynamicPayloadTypes[:idx],
					dynamicPayloadTypes[idx+1:]...)
			}
		} else if codec.PreferredPayloadType == 0 {
			if len(dynamicPayloadTypes) == 0 {
				return nil, errors.New("cannot allocate more dynamic codec payload types")
			}
			codec.PreferredPayloadType = dynamicPayloadTypes[0]
			dynamicPayloadTypes = dynamicPayloadTypes[1:]
		}

		for _, capCodec := range caps.Codecs {
			if capCodec.PreferredPayloadType == codec.PreferredPayloadType {
				return nil, errors.New("duplicated codec.preferredPayloadType")
			}
		}

		// Merge the media codec parameters.
		data, _ := json.Marshal(mediaCodec.Parameters)
		_ = json.Unmarshal(data, &codec.Parameters)

		// Append to the codec list.
		caps.Codecs = append(caps.Codecs, &codec)

		// Add a RTX video codec if video.
		if codec.Kind == MediaKindVideo {
			if len(dynamicPayloadTypes) == 0 {
				return nil, errors.New("cannot allocate more dynamic codec payload types")
			}
			pt := dynamicPayloadTypes[0]
			dynamicPayloadTypes = dynamicPayloadTypes[1:]

			rtxCodec := &RtpCodecCapability{
				Kind:                 codec.Kind,
				MimeType:             fmt.Sprintf("%s/rtx", codec.Kind),
				PreferredPayloadType: pt,
				ClockRate:            codec.ClockRate,
				Parameters:           RtpCodecSpecificParameters{Apt: codec.PreferredPayloadType},
				RtcpFeedback:         []*RtcpFeedback{},
			}

			// Append to the codec list.
			caps.Codecs = append(caps.Codecs, rtxCodec)
		}
	}

	return caps, nil
}

// getProducerRtpParametersMapping get a mapping of the codec payload, RTP header extensions and
// encodings from the given Producer RTP parameters to the values expected by the Router.
func getProducerRtpParametersMapping(params *RtpParameters, caps *RtpCapabilities) (*RtpMapping, error) {
	// Match parameters media codecs to capabilities media codecs.
	codecToCapCodec := map[*RtpCodecParameters]*RtpCodecCapability{}
	rtpMapping := &RtpMapping{}

	for _, codec := range params.Codecs {
		if codec.isRtxCodec() {
			continue
		}
		matchedCapCodec, matched := findMatchedCodec(
			caps.Codecs,
			codec,
			matchOptions{strict: true, modify: true},
		)

		if !matched {
			return nil, fmt.Errorf(
				"unsupported codec [mimeType:%s, payloadType:%d]",
				codec.MimeType,
				codec.PayloadType,
			)
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
			return nil, fmt.Errorf(`missing media codec found for RTX PT %d`, codec.PayloadType)
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
			return nil, fmt.Errorf(
				"no RTX codec for capability codec PT %d",
				capMediaCodec.PreferredPayloadType,
			)
		}

		codecToCapCodec[codec] = associatedCapRtxCodec
	}

	// Generate codecs mapping.
	for codec, capCodec := range codecToCapCodec {
		rtpMapping.Codecs = append(rtpMapping.Codecs, &RtpMappingCodec{
			PayloadType:       codec.PayloadType,
			MappedPayloadType: capCodec.PreferredPayloadType,
		})
	}

	// Generate encodings mapping.
	mappedSsrc := generateSsrc()

	for _, encoding := range params.Encodings {
		mappedEncoding := &RtpMappingEncoding{
			Rid:             encoding.Rid,
			Ssrc:            orElse(encoding.Ssrc > 0, ref(encoding.Ssrc), nil),
			MappedSsrc:      mappedSsrc,
			ScalabilityMode: encoding.ScalabilityMode,
		}
		mappedSsrc++
		rtpMapping.Encodings = append(rtpMapping.Encodings, mappedEncoding)
	}

	return rtpMapping, nil
}

// getConsumableRtpParameters generate RTP parameters for Consumers given the RTP parameters of a
// Producer and the RTP capabilities of the Router.
func getConsumableRtpParameters(
	kind MediaKind,
	params *RtpParameters,
	caps *RtpCapabilities,
	rtpMapping *RtpMapping,
) *RtpParameters {
	consumableRtpParameters := &RtpParameters{}

	for _, codec := range params.Codecs {
		if codec.isRtxCodec() {
			continue
		}
		var consumableCodecPt uint8

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
		consumableRtpParameters.Codecs = append(consumableRtpParameters.Codecs, consumableCodec)

		var consumableCapRtxCodec *RtpCodecCapability

		for _, capRtxCodec := range caps.Codecs {
			if capRtxCodec.isRtxCodec() &&
				capRtxCodec.Parameters.Apt == consumableCodec.PayloadType {
				consumableCapRtxCodec = capRtxCodec
				break
			}
		}

		if consumableCapRtxCodec != nil {
			consumableRtpParameters.Codecs = append(consumableRtpParameters.Codecs, &RtpCodecParameters{
				MimeType:     consumableCapRtxCodec.MimeType,
				ClockRate:    consumableCapRtxCodec.ClockRate,
				Channels:     consumableCapRtxCodec.Channels,
				RtcpFeedback: consumableCapRtxCodec.RtcpFeedback,
				Parameters:   consumableCapRtxCodec.Parameters,
				PayloadType:  consumableCapRtxCodec.PreferredPayloadType,
			})
		}
	}

	for _, capExt := range caps.HeaderExtensions {
		if capExt.Kind != kind ||
			(capExt.Direction != MediaDirectionSendrecv && capExt.Direction != MediaDirectionSendonly) {
			continue
		}
		consumableExt := &RtpHeaderExtensionParameters{
			Uri:     capExt.Uri,
			Id:      capExt.PreferredId,
			Encrypt: capExt.PreferredEncrypt,
		}
		consumableRtpParameters.HeaderExtensions = append(consumableRtpParameters.HeaderExtensions, consumableExt)
	}

	for i, encoding := range params.Encodings {
		// Set the mapped ssrc.
		consumableRtpParameters.Encodings = append(consumableRtpParameters.Encodings, &RtpEncodingParameters{
			Ssrc:                  rtpMapping.Encodings[i].MappedSsrc,
			Dtx:                   encoding.Dtx,
			ScalabilityMode:       encoding.ScalabilityMode,
			ScaleResolutionDownBy: encoding.ScaleResolutionDownBy,
			MaxBitrate:            encoding.MaxBitrate,
		})
	}

	consumableRtpParameters.Rtcp = &RtcpParameters{
		Cname:       params.Rtcp.Cname,
		ReducedSize: ref(true),
	}

	return consumableRtpParameters
}

// CanConsume check whether the given RTP capabilities can consume the given Producer.
func CanConsume(consumableRtpParameters *RtpParameters, caps *RtpCapabilities) (ok bool, err error) {
	if err = validateRtpCapabilities(caps); err != nil {
		return
	}

	var matchingCodecs []*RtpCodecCapability

	for _, codec := range consumableRtpParameters.Codecs {
		matchedCodec, matched := findMatchedCodec(caps.Codecs, codec, matchOptions{strict: true})

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

// getConsumerRtpParameters generate RTP parameters for a specific Consumer.
//
// It reduces encodings to just one and takes into account given RTP capabilities
// to reduce codecs, codecs" RTCP feedback and header extensions, and also enables
// or disabled RTX.
func getConsumerRtpParameters(
	consumableRtpParameters *RtpParameters,
	remoteRtpCapabilities *RtpCapabilities,
	pipe, enableRtx bool,
) (*RtpParameters, error) {
	for _, capCodec := range remoteRtpCapabilities.Codecs {
		if err := validateRtpCodecCapability(capCodec); err != nil {
			return nil, err
		}
	}

	consumerParams := &RtpParameters{}
	rtxSupported := false

	for _, codec := range consumableRtpParameters.Codecs {
		codec = ref(*codec)
		if !enableRtx && codec.isRtxCodec() {
			continue
		}

		matchedCapCodec, matched := findMatchedCodec(remoteRtpCapabilities.Codecs, codec, matchOptions{strict: true})

		if !matched {
			continue
		}

		codec.RtcpFeedback = filterRtcpFeedback(matchedCapCodec.RtcpFeedback, func(fb *RtcpFeedback) bool {
			return (enableRtx || fb.Type != "nack" || fb.Parameter != "")
		})

		consumerParams.Codecs = append(consumerParams.Codecs, codec)
	}

	// Must sanitize the list of matched codecs by removing useless RTX codecs.
	for i := len(consumerParams.Codecs) - 1; i >= 0; i-- {
		if codec := consumerParams.Codecs[i]; codec.isRtxCodec() {
			found := false
			// Search for the associated media codec.
			for _, mediaCodec := range consumerParams.Codecs {
				if mediaCodec.PayloadType == codec.Parameters.Apt {
					rtxSupported = true
					found = true
					break
				}
			}
			if !found {
				consumerParams.Codecs = append(consumerParams.Codecs[0:i], consumerParams.Codecs[i+1:]...)
			}
		}
	}

	// Ensure there is at least one media codec.
	if len(consumerParams.Codecs) == 0 || consumerParams.Codecs[0].isRtxCodec() {
		return nil, errors.New("no compatible media codecs")
	}

	for _, ext := range consumableRtpParameters.HeaderExtensions {
		for _, capExt := range remoteRtpCapabilities.HeaderExtensions {
			if capExt.PreferredId == ext.Id && capExt.Uri == ext.Uri {
				consumerParams.HeaderExtensions = append(consumerParams.HeaderExtensions, ext)
				break
			}
		}
	}

	// Reduce codecs' RTCP feedback. Use Transport-CC if available, REMB otherwise.
	if matchHeaderExtensionUri(consumerParams.HeaderExtensions,
		"http://www.ietf.org/id/draft-holmer-rmcat-transport-wide-cc-extensions-01") {
		for _, codec := range consumerParams.Codecs {
			codec.RtcpFeedback = filterRtcpFeedback(codec.RtcpFeedback, func(fb *RtcpFeedback) bool {
				return fb.Type != "goog-remb"
			})
		}
	} else if matchHeaderExtensionUri(consumerParams.HeaderExtensions,
		"http://www.webrtc.org/experiments/rtp-hdrext/abs-send-time") {
		for _, codec := range consumerParams.Codecs {
			codec.RtcpFeedback = filterRtcpFeedback(codec.RtcpFeedback, func(fb *RtcpFeedback) bool {
				return fb.Type != "transport-cc"
			})
		}
	} else {
		for _, codec := range consumerParams.Codecs {
			codec.RtcpFeedback = filterRtcpFeedback(codec.RtcpFeedback, func(fb *RtcpFeedback) bool {
				return fb.Type != "transport-cc" && fb.Type != "goog-remb"
			})
		}
	}

	if pipe {
		baseSsrc := generateSsrc()
		baseRtxSsrc := generateSsrc()

		for i, encoding := range consumableRtpParameters.Encodings {
			encoding = ref(*encoding)

			encoding.Ssrc = baseSsrc + uint32(i)
			if rtxSupported {
				encoding.Rtx = &RtpEncodingRtx{Ssrc: baseRtxSsrc + uint32(i)}
			} else {
				encoding.Rtx = nil
			}

			consumerParams.Encodings = append(consumerParams.Encodings, encoding)
		}
	} else {
		consumerEncoding := &RtpEncodingParameters{
			Ssrc: generateSsrc(),
		}

		if rtxSupported {
			consumerEncoding.Rtx = &RtpEncodingRtx{
				Ssrc: generateSsrc(),
			}
		}

		var scalabilityMode string

		// If any of the consumableRtpParameters.encodings has scalabilityMode, process it
		// (assume all encodings have the same value).
		for _, encoding := range consumableRtpParameters.Encodings {
			if len(encoding.ScalabilityMode) > 0 {
				scalabilityMode = encoding.ScalabilityMode
				break
			}
		}

		// If there is simulast, mangle spatial layers in scalabilityMode.
		if len(consumableRtpParameters.Encodings) > 1 {
			temporalLayers := parseScalabilityMode(scalabilityMode).TemporalLayers
			scalabilityMode = fmt.Sprintf("L%dT%d", len(consumableRtpParameters.Encodings), temporalLayers)
		}

		consumerEncoding.ScalabilityMode = scalabilityMode

		maxEncodingMaxBitrate := uint32(0)

		// Use the maximum maxBitrate in any encoding and honor it in the Consumer's encoding.
		for _, encoding := range consumableRtpParameters.Encodings {
			if maxBitrate := encoding.MaxBitrate; maxBitrate > maxEncodingMaxBitrate {
				maxEncodingMaxBitrate = maxBitrate
			}
		}

		if maxEncodingMaxBitrate > 0 {
			consumerEncoding.MaxBitrate = maxEncodingMaxBitrate
		}

		// Set a single encoding for the Consumer.
		consumerParams.Encodings = append(consumerParams.Encodings, consumerEncoding)
	}

	return consumerParams, nil
}

// getPipeConsumerRtpParameters generate RTP parameters for a pipe Consumer.
//
// It keeps all original consumable encodings and removes support for BWE. If
// enableRtx is false, it also removes RTX and NACK support.
func getPipeConsumerRtpParameters(consumableRtpParameters *RtpParameters, enableRtx bool) *RtpParameters {
	consumerParams := &RtpParameters{}
	consumerParams.Rtcp = consumableRtpParameters.Rtcp

	for _, codec := range consumableRtpParameters.Codecs {
		codec = ref(*codec)

		if !enableRtx && codec.isRtxCodec() {
			continue
		}

		codec.RtcpFeedback = filterRtcpFeedback(codec.RtcpFeedback, func(fb *RtcpFeedback) bool {
			return (fb.Type == "nack" && fb.Parameter == "pli") ||
				(fb.Type == "ccm" && fb.Parameter == "fir") ||
				(enableRtx && fb.Type == "nack" && len(fb.Parameter) == 0)
		})
		consumerParams.Codecs = append(consumerParams.Codecs, codec)
	}

	// Reduce RTP extensions by disabling transport MID and BWE related ones.
	for _, ext := range consumableRtpParameters.HeaderExtensions {
		if ext.Uri != "urn:ietf:params:rtp-hdrext:sdes:mid" &&
			ext.Uri != "http://www.webrtc.org/experiments/rtp-hdrext/abs-send-time" &&
			ext.Uri != "http://www.ietf.org/id/draft-holmer-rmcat-transport-wide-cc-extensions-01" {
			consumerParams.HeaderExtensions = append(consumerParams.HeaderExtensions, ext)
		}
	}

	// TODO: Remove this if we switch to 'sendrecv' in Dependency-Descriptor header
	// extension.
	//
	// We need to add Dependency-Descriptor header extension manually since it's
	// 'recvonly' so it's not present in received `consumableRtpParameters`.
	if dependencyDescriptorHeaderExtensionParametersForPipeConsumer != nil {
		consumerParams.HeaderExtensions = append(consumerParams.HeaderExtensions, dependencyDescriptorHeaderExtensionParametersForPipeConsumer)
		// Sort header extensions by ID.
		sort.Slice(consumerParams.HeaderExtensions, func(i, j int) bool {
			return consumerParams.HeaderExtensions[i].Id < consumerParams.HeaderExtensions[j].Id
		})
	}

	baseSsrc := generateSsrc()
	baseRtxSsrc := generateSsrc()

	for i, encoding := range consumableRtpParameters.Encodings {
		encoding = ref(*encoding)
		encoding.Ssrc = baseSsrc + uint32(i)

		if enableRtx {
			encoding.Rtx = &RtpEncodingRtx{Ssrc: baseRtxSsrc + uint32(i)}
		} else {
			encoding.Rtx = nil
		}

		consumerParams.Encodings = append(consumerParams.Encodings, encoding)
	}

	return consumerParams
}

func findMatchedCodec(
	bCodecs []*RtpCodecCapability,
	aCodec any,
	options matchOptions,
) (codec *RtpCodecCapability, matched bool) {
	var rtpCodecParameters *RtpCodecParameters

	switch aCodec := aCodec.(type) {
	case *RtpCodecCapability:
		rtpCodecParameters = &RtpCodecParameters{
			MimeType:   aCodec.MimeType,
			ClockRate:  aCodec.ClockRate,
			Channels:   aCodec.Channels,
			Parameters: aCodec.Parameters,
		}
	case *RtpCodecParameters:
		rtpCodecParameters = aCodec
	}

	for _, bCodec := range bCodecs {
		if matchCodecs(rtpCodecParameters, bCodec, options) {
			return bCodec, true
		}
	}
	return nil, false
}

func matchCodecs(
	aCodec *RtpCodecParameters,
	bCodec *RtpCodecCapability,
	options matchOptions,
) (matched bool) {
	aMimeType := strings.ToLower(aCodec.MimeType)
	bMimeType := strings.ToLower(bCodec.MimeType)

	if aMimeType != bMimeType {
		return false
	}

	if aCodec.ClockRate != bCodec.ClockRate {
		return false
	}

	if strings.HasPrefix(aMimeType, "audio/") &&
		aCodec.Channels > 0 &&
		bCodec.Channels > 0 &&
		aCodec.Channels != bCodec.Channels {
		return false
	}

	aParameters, bParameters := aCodec.Parameters, bCodec.Parameters

	switch aMimeType {
	case "audio/multiopus":
		aNumStreams := aParameters.NumStreams
		bNumstreams := bParameters.NumStreams

		if aNumStreams != bNumstreams {
			return false
		}

		aCoupledStreams := aParameters.CoupledStreams
		bCoupledStreams := bParameters.CoupledStreams

		if aCoupledStreams != bCoupledStreams {
			return false
		}

	case "video/h264":
		if options.strict {
			if aParameters.PacketizationMode != bParameters.PacketizationMode {
				return false
			}

			if !h264.IsSameProfile(aParameters.ProfileLevelId, bParameters.ProfileLevelId) {
				return false
			}

			selectedProfileLevelId, err := h264.GenerateProfileLevelIdForAnswer(
				h264.RtpParameter{
					PacketizationMode:     aParameters.PacketizationMode,
					ProfileLevelId:        aParameters.ProfileLevelId,
					LevelAsymmetryAllowed: aParameters.LevelAsymmetryAllowed,
				}, h264.RtpParameter{
					PacketizationMode:     bParameters.PacketizationMode,
					ProfileLevelId:        bParameters.ProfileLevelId,
					LevelAsymmetryAllowed: bParameters.LevelAsymmetryAllowed,
				})
			if err != nil {
				return false
			}

			if options.modify {
				aParameters.ProfileLevelId = selectedProfileLevelId
				aCodec.Parameters = aParameters
			}
		}

	case "video/vp9":
		if options.strict && aParameters.ProfileId != bParameters.ProfileId {
			return false
		}
	}

	return true
}

func matchHeaderExtensionUri(exts []*RtpHeaderExtensionParameters, uri string) bool {
	for _, ext := range exts {
		if ext.Uri == uri {
			return true
		}
	}

	return false
}

func filterRtcpFeedback(arr []*RtcpFeedback, cond func(*RtcpFeedback) bool) []*RtcpFeedback {
	newArr := arr[:0]

	for _, x := range arr {
		if cond(x) {
			newArr = append(newArr, x)
		}
	}

	return newArr
}
