package mediasoup

import (
	"strings"

	"github.com/jiyeyuran/mediasoup-go/h264"
)

/**
 * The RTP capabilities define what mediasoup or an endpoint can receive at
 * media level.
 */
type RtpCapabilities struct {
	/**
	 * Supported media and RTX codecs.
	 */
	Codecs []*RtpCodecCapability `json:"codecs,omitempty"`

	/**
	 * Supported RTP header extensions.
	 */
	HeaderExtensions []*RtpHeaderExtension `json:"headerExtensions,omitempty"`

	/**
	 * Supported FEC mechanisms.
	 */
	FecMechanisms []string `json:"fecMechanisms,omitempty"`
}

/**
 * Media kind ('audio' or 'video').
 */
type MediaKind string

const (
	MediaKind_Audio MediaKind = "audio"
	MediaKind_Video           = "video"
)

/**
 * Provides information on the capabilities of a codec within the RTP
 * capabilities. The list of media codecs supported by mediasoup and their
 * settings is defined in the supportedRtpCapabilities.ts file.
 *
 * Exactly one RtpCodecCapability will be present for each supported combination
 * of parameters that requires a distinct value of preferredPayloadType. For
 * example
 *
 * - Multiple H264 codecs, each with their own distinct 'packetization-mode' and
 *   'profile-level-id' values.
 * - Multiple VP9 codecs, each with their own distinct 'profile-id' value.
 *
 * RtpCodecCapability entries in the mediaCodecs array of RouterOptions do not
 * require preferredPayloadType field (if unset, mediasoup will choose a random
 * one). If given, make sure it's in the 96-127 range.
 */
type RtpCodecCapability struct {
	/**
	 * Media kind.
	 */
	Kind MediaKind `json:"kind"`

	/**
	 * The codec MIME media type/subtype (e.g. 'audio/opus', 'video/VP8').
	 */
	MimeType string `json:"mimeType"`

	/**
	 * The preferred RTP payload type.
	 */
	PreferredPayloadType byte `json:"preferredPayloadType,omitempty"`

	/**
	 * Codec clock rate expressed in Hertz.
	 */
	ClockRate int `json:"clockRate"`

	/**
	 * The int of channels supported (e.g. two for stereo). Just for audio.
	 * Default 1.
	 */
	Channels int `json:"channels,omitempty"`

	/**
	 * Codec specific parameters. Some parameters (such as 'packetization-mode'
	 * and 'profile-level-id' in H264 or 'profile-id' in VP9) are critical for
	 * codec matching.
	 */
	Parameters RtpCodecSpecificParameters `json:"parameters,omitempty"`

	/**
	 * Transport layer and codec-specific feedback messages for this codec.
	 */
	RtcpFeedback []RtcpFeedback `json:"rtcpFeedback,omitempty"`
}

func (r RtpCodecCapability) isRtxCodec() bool {
	return strings.HasSuffix(strings.ToLower(r.MimeType), "/rtx")
}

/**
 * Direction of RTP header extension.
 */
type RtpHeaderExtensionDirection string

const (
	Direction_Sendrecv RtpHeaderExtensionDirection = "sendrecv"
	Direction_Sendonly                             = "sendonly"
	Direction_Recvonly                             = "recvonly"
	Direction_Inactive                             = "inactive"
)

/**
 * Provides information relating to supported header extensions. The list of
 * RTP header extensions supported by mediasoup is defined in the
 * supportedRtpCapabilities.ts file.
 *
 * mediasoup does not currently support encrypted RTP header extensions. The
 * direction field is just present in mediasoup RTP capabilities (retrieved via
 * router.rtpCapabilities or mediasoup.getSupportedRtpCapabilities()). It's
 * ignored if present in endpoints' RTP capabilities.
 */
type RtpHeaderExtension struct {
	/**
	 * Media kind. If empty string, it's valid for all kinds.
	 * Default any media kind.
	 */
	Kind MediaKind `json:"kind"`

	/*
	 * The URI of the RTP header extension, as defined in RFC 5285.
	 */
	Uri string `json:"uri"`

	/**
	 * The preferred numeric identifier that goes in the RTP packet. Must be
	 * unique.
	 */
	PreferredId int `json:"preferredId"`

	/**
	 * If true, it is preferred that the value in the header be encrypted as per
	 * RFC 6904. Default false.
	 */
	PreferredEncrypt bool `json:"preferredEncrypt,omitempty"`

	/**
	 * If 'sendrecv', mediasoup supports sending and receiving this RTP extension.
	 * 'sendonly' means that mediasoup can send (but not receive) it. 'recvonly'
	 * means that mediasoup can receive (but not send) it.
	 */
	Direction RtpHeaderExtensionDirection `json:"direction,omitempty"`
}

/**
 * The RTP send parameters describe a media stream received by mediasoup from
 * an endpoint through its corresponding mediasoup Producer. These parameters
 * may include a mid value that the mediasoup transport will use to match
 * received RTP packets based on their MID RTP extension value.
 *
 * mediasoup allows RTP send parameters with a single encoding and with multiple
 * encodings (simulcast). In the latter case, each entry in the encodings array
 * must include a ssrc field or a rid field (the RID RTP extension value). Check
 * the Simulcast and SVC sections for more information.
 *
 * The RTP receive parameters describe a media stream as sent by mediasoup to
 * an endpoint through its corresponding mediasoup Consumer. The mid value is
 * unset (mediasoup does not include the MID RTP extension into RTP packets
 * being sent to endpoints).
 *
 * There is a single entry in the encodings array (even if the corresponding
 * producer uses simulcast). The consumer sends a single and continuous RTP
 * stream to the endpoint and spatial/temporal layer selection is possible via
 * consumer.setPreferredLayers().
 *
 * As an exception, previous bullet is not true when consuming a stream over a
 * PipeTransport, in which all RTP streams from the associated producer are
 * forwarded verbatim through the consumer.
 *
 * The RTP receive parameters will always have their ssrc values randomly
 * generated for all of its  encodings (and optional rtx { ssrc XXXX } if the
 * endpoint supports RTX), regardless of the original RTP send parameters in
 * the associated producer. This applies even if the producer's encodings have
 * rid set.
 */
type RtpParameters struct {
	/**
	 * The MID RTP extension value as defined in the BUNDLE specification.
	 */
	Mid string `json:"mid,omitempty"`

	/**
	 * Media and RTX codecs in use.
	 */
	Codecs []*RtpCodecParameters `json:"codecs"`

	/**
	 * RTP header extensions in use.
	 */
	HeaderExtensions []RtpHeaderExtensionParameters `json:"headerExtensions,omitempty"`

	/**
	 * Transmitted RTP streams and their settings.
	 */
	Encodings []RtpEncodingParameters `json:"encodings,omitempty"`

	/**
	 * Parameters used for RTCP.
	 */
	Rtcp RtcpParameters `json:"rtcp,omitempty"`
}

/**
 * Provides information on codec settings within the RTP parameters. The list
 * of media codecs supported by mediasoup and their settings is defined in the
 * supportedRtpCapabilities.ts file.
 */
type RtpCodecParameters struct {
	/**
	 * The codec MIME media type/subtype (e.g. 'audio/opus', 'video/VP8').
	 */
	MimeType string `json:"mimeType"`

	/**
	 * The value that goes in the RTP Payload Type Field. Must be unique.
	 */
	PayloadType byte `json:"payloadType"`

	/**
	 * Codec clock rate expressed in Hertz.
	 */
	ClockRate int `json:"clockRate"`

	/**
	 * The int of channels supported (e.g. two for stereo). Just for audio.
	 * Default 1.
	 */
	Channels int `json:"channels,omitempty"`

	/**
	 * Codec-specific parameters available for signaling. Some parameters (such
	 * as 'packetization-mode' and 'profile-level-id' in H264 or 'profile-id' in
	 * VP9) are critical for codec matching.
	 */
	Parameters RtpCodecSpecificParameters `json:"parameters,omitempty"`

	/**
	 * Transport layer and codec-specific feedback messages for this codec.
	 */
	RtcpFeedback []RtcpFeedback `json:"rtcpFeedback,omitempty"`
}

func (r RtpCodecParameters) isRtxCodec() bool {
	return strings.HasSuffix(strings.ToLower(r.MimeType), "/rtx")
}

/**
 * RtpCodecSpecificParameters the Codec-specific parameters available for signaling. Some parameters (such
 * as 'packetization-mode' and 'profile-level-id' in H264 or 'profile-id' in
 * VP9) are critical for codec matching.
 */
type RtpCodecSpecificParameters struct {
	h264.RtpParameter          // used by h264 codec
	ProfileId           string `json:"profile-id,omitempty"`   // used by vp9
	Apt                 byte   `json:"apt,omitempty"`          // used by rtx codec
	SpropStereo         uint8  `json:"sprop-stereo,omitempty"` // used by audio, 1 or 0
	Useinbandfec        uint8  `json:"useinbandfec,omitempty"` // used by audio, 1 or 0
	Usedtx              uint8  `json:"usedtx,omitempty"`       // used by audio, 1 or 0
	Maxplaybackrate     uint32 `json:"maxplaybackrate,omitempty"`
	XGoogleMinBitrate   uint32 `json:"x-google-min-bitrate,omitempty"`
	XGoogleMaxBitrate   uint32 `json:"x-google-max-bitrate,omitempty"`
	XGoogleStartBitrate uint32 `json:"x-google-start-bitrate,omitempty"`
}

/**
 * Provides information on RTCP feedback messages for a specific codec. Those
 * messages can be transport layer feedback messages or codec-specific feedback
 * messages. The list of RTCP feedbacks supported by mediasoup is defined in the
 * supportedRtpCapabilities.ts file.
 */
type RtcpFeedback struct {
	/**
	 * RTCP feedback type.
	 */
	Type string `json:"type"`

	/**
	 * RTCP feedback parameter.
	 */
	Parameter string `json:"parameter,omitempty"`
}

/**
 * Provides information relating to an encoding, which represents a media RTP
 * stream and its associated RTX stream (if any).
 */
type RtpEncodingParameters struct {
	/**
	 * The media SSRC.
	 */
	Ssrc uint32 `json:"ssrc,omitempty"`

	/**
	 * The RID RTP extension value. Must be unique.
	 */
	Rid string `json:"rid,omitempty"`

	/**
	 * Codec payload type this encoding affects. If unset, first media codec is
	 * chosen.
	 */
	CodecPayloadType byte `json:"codecPayloadType,omitempty"`

	/**
	 * RTX stream information. It must contain a numeric ssrc field indicating
	 * the RTX SSRC.
	 */
	Rtx *RtpEncodingRtx `json:"rtx,omitempty"`

	/**
	 * It indicates whether discontinuous RTP transmission will be used. Useful
	 * for audio (if the codec supports it) and for video screen sharing (when
	 * static content is being transmitted, this option disables the RTP
	 * inactivity checks in mediasoup). Default false.
	 */
	Dtx bool `json:"dtx,omitempty"`

	/**
	 * int of spatial and temporal layers in the RTP stream (e.g. 'L1T3').
	 * See webrtc-svc.
	 */
	ScalabilityMode string `json:"scalabilityMode,omitempty"`

	/**
	 * Others.
	 */
	ScaleResolutionDownBy int `json:"scaleResolutionDownBy,omitempty"`
	MaxBitrate            int `json:"maxBitrate,omitempty"`
}

// RtpEncodingRtx represents the associated RTX stream for RTP stream.
type RtpEncodingRtx struct {
	Ssrc uint32 `json:"ssrc"`
}

/**
 * Defines a RTP header extension within the RTP parameters. The list of RTP
 * header extensions supported by mediasoup is defined in the
 * supportedRtpCapabilities.ts file.
 *
 * mediasoup does not currently support encrypted RTP header extensions and no
 * parameters are currently considered.
 */
type RtpHeaderExtensionParameters struct {
	/**
	 * The URI of the RTP header extension, as defined in RFC 5285.
	 */
	Uri string `json:"uri"`

	/**
	 * The numeric identifier that goes in the RTP packet. Must be unique.
	 */
	Id int `json:"id"`

	/**
	 * If true, the value in the header is encrypted as per RFC 6904. Default false.
	 */
	Encrypt bool `json:"encrypt,omitempty"`

	/**
	 * Configuration parameters for the header extension.
	 */
	Parameters *RtpCodecSpecificParameters `json:"parameters,omitempty"`
}

/**
 * Provides information on RTCP settings within the RTP parameters.
 *
 * If no cname is given in a producer's RTP parameters, the mediasoup transport
 * will choose a random one that will be used into RTCP SDES messages sent to
 * all its associated consumers.
 *
 * mediasoup assumes reducedSize to always be true.
 */
type RtcpParameters struct {
	/**
	 * The Canonical Name (CNAME) used by RTCP (e.g. in SDES messages).
	 */
	Cname string `json:"cname,omitempty"`

	/**
	 * Whether reduced size RTCP RFC 5506 is configured (if true) or compound RTCP
	 * as specified in RFC 3550 (if false). Default true.
	 */
	ReducedSize *bool `json:"reducedSize,omitempty"`

	/**
	 * Whether RTCP-mux is used. Default true.
	 */
	Mux *bool `json:"mux,omitempty"`
}
