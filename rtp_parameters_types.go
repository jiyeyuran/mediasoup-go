package mediasoup

import (
	"strings"
)

// RtpCapabilities define what mediasoup or an endpoint can receive at media level.
type RtpCapabilities struct {
	// Codecs is the supported media and RTX codecs.
	Codecs []*RtpCodecCapability `json:"codecs,omitempty"`

	// HeaderExtensions is the supported RTP header extensions.
	HeaderExtensions []*RtpHeaderExtension `json:"headerExtensions,omitempty"`

	// FecMechanisms is the supported FEC mechanisms.
	FecMechanisms []string `json:"fecMechanisms,omitempty"`
}

// Media kind ("audio" or "video").
type MediaKind string

const (
	MediaKindAudio MediaKind = "audio"
	MediaKindVideo MediaKind = "video"
)

// RtpCodecCapability provides information on the capabilities of a codec within the RTP
// capabilities. The list of media codecs supported by mediasoup and their
// settings is defined in the supported_rtp_capabilities.go file.
//
// Exactly one RtpCodecCapability will be present for each supported combination
// of parameters that requires a distinct value of preferredPayloadType. For
// example
//
//   - Multiple H264 codecs, each with their own distinct 'packetization-mode' and
//     'profile-level-id' values.
//   - Multiple VP9 codecs, each with their own distinct 'profile-id' value.
//
// RtpCodecCapability entries in the mediaCodecs array of RouterOptions do not
// require preferredPayloadType field (if unset, mediasoup will choose a random
// one). If given, make sure it's in the 96-127 range.
type RtpCodecCapability struct {
	// Kind is the media kind.
	Kind MediaKind `json:"kind"`

	// MimeType is the codec MIME media type/subtype (e.g. 'audio/opus', 'video/VP8').
	MimeType string `json:"mimeType"`

	// PreferredPayloadType is the preferred RTP payload type.
	PreferredPayloadType uint8 `json:"preferredPayloadType,omitempty"`

	// ClockRate is the codec clock rate expressed in Hertz.
	ClockRate uint32 `json:"clockRate"`

	// Channels is the int of channels supported (e.g. 2 for stereo). Just for audio.
	// Default 1.
	Channels uint8 `json:"channels,omitempty"`

	// Parameters is the codec specific parameters. Some parameters (such as
	// 'packetization-mode' and 'profile-level-id' in H264 or 'profile-id' in VP9)
	// are critical for codec matching.
	Parameters RtpCodecSpecificParameters `json:"parameters,omitempty"`

	// RtcpFeedback is the transport layer and codec-specific feedback messages for this codec.
	RtcpFeedback []*RtcpFeedback `json:"rtcpFeedback,omitempty"`
}

func (r RtpCodecCapability) clone() RtpCodecCapability {
	clone := r
	clone.RtcpFeedback = make([]*RtcpFeedback, len(r.RtcpFeedback))
	for i, feedback := range r.RtcpFeedback {
		clone.RtcpFeedback[i] = &RtcpFeedback{
			Type:      feedback.Type,
			Parameter: feedback.Parameter,
		}
	}
	return clone
}

func (r RtpCodecCapability) isRtxCodec() bool {
	return strings.HasSuffix(strings.ToLower(r.MimeType), "/rtx")
}

// Direction of RTP header extension.
type MediaDirection string

const (
	MediaDirectionSendrecv MediaDirection = "sendrecv"
	MediaDirectionSendonly MediaDirection = "sendonly"
	MediaDirectionRecvonly MediaDirection = "recvonly"
	MediaDirectionInactive MediaDirection = "inactive"
)

// RtpHeaderExtension provides information relating to supported header extensions.
// The list of RTP header extensions supported by mediasoup is defined in the
// supported_rtp_capabilities.go file.
//
// mediasoup does not currently support encrypted RTP header extensions. The
// direction field is just present in mediasoup RTP capabilities (retrieved via
// router.RtpCapabilities() or mediasoup.GetSupportedRtpCapabilities()). It's
// ignored if present in endpoints' RTP capabilities.
type RtpHeaderExtension struct {
	// Kind is media kind. If empty string, it's valid for all kinds.
	// Default any media kind.
	Kind MediaKind `json:"kind"`

	// URI of the RTP header extension, as defined in RFC 5285.
	Uri string `json:"uri"`

	// PreferredId is the preferred numeric identifier that goes in the RTP packet.
	// Must be unique.
	PreferredId uint8 `json:"preferredId"`

	// PreferredEncrypt if true, it is preferred that the value in the header be
	// encrypted as per RFC 6904. Default false.
	PreferredEncrypt bool `json:"preferredEncrypt,omitempty"`

	// Direction if "sendrecv", mediasoup supports sending and receiving this RTP extension.
	// "sendonly" means that mediasoup can send (but not receive) it. "recvonly"
	// means that mediasoup can receive (but not send) it.
	Direction MediaDirection `json:"direction,omitempty"`
}

// RtpParameters describe a media stream received by mediasoup from
// an endpoint through its corresponding mediasoup Producer. These parameters
// may include a mid value that the mediasoup transport will use to match
// received RTP packets based on their MID RTP extension value.
//
// mediasoup allows RTP send parameters with a single encoding and with multiple
// encodings (simulcast). In the latter case, each entry in the encodings array
// must include a ssrc field or a rid field (the RID RTP extension value). Check
// the Simulcast and SVC sections for more information.
//
// The RTP receive parameters describe a media stream as sent by mediasoup to
// an endpoint through its corresponding mediasoup Consumer. The mid value is
// unset (mediasoup does not include the MID RTP extension into RTP packets
// being sent to endpoints).
//
// There is a single entry in the encodings array (even if the corresponding
// producer uses simulcast). The consumer sends a single and continuous RTP
// stream to the endpoint and spatial/temporal layer selection is possible via
// consumer.setPreferredLayers().
//
// As an exception, previous bullet is not true when consuming a stream over a
// PipeTransport, in which all RTP streams from the associated producer are
// forwarded verbatim through the consumer.
//
// The RTP receive parameters will always have their ssrc values randomly
// generated for all of its  encodings (and optional rtx { ssrc XXXX } if the
// endpoint supports RTX), regardless of the original RTP send parameters in
// the associated producer. This applies even if the producer's encodings have
// rid set.
type RtpParameters struct {
	// MID RTP extension value as defined in the BUNDLE specification.
	Mid string `json:"mid,omitempty"`

	// Codecs defines media and RTX codecs in use.
	Codecs []*RtpCodecParameters `json:"codecs"`

	// HeaderExtensions is the RTP header extensions in use.
	HeaderExtensions []*RtpHeaderExtensionParameters `json:"headerExtensions,omitempty"`

	// Encodings is the transmitted RTP streams and their settings.
	Encodings []*RtpEncodingParameters `json:"encodings,omitempty"`

	// Rtcp is the parameters used for RTCP.
	Rtcp *RtcpParameters `json:"rtcp,omitempty"`
}

// RtpCodecParameters provides information on codec settings within the RTP parameters.
// The list of media codecs supported by mediasoup and their settings is defined in the
// supported_rtp_capabilities.go file.
type RtpCodecParameters struct {
	// MimeType is the codec MIME media type/subtype (e.g. 'audio/opus', 'video/VP8').
	MimeType string `json:"mimeType"`

	// PayloadType is the value that goes in the RTP Payload Type Field. Must be unique.
	PayloadType uint8 `json:"payloadType"`

	// ClockRate is codec clock rate expressed in Hertz.
	ClockRate uint32 `json:"clockRate"`

	// Channels is the int of channels supported (e.g. 2 for stereo). Just for audio.
	// Default 1.
	Channels uint8 `json:"channels,omitempty"`

	// Parameters is Codec-specific parameters available for signaling. Some parameters
	// (such as 'packetization-mode' and 'profile-level-id' in H264 or 'profile-id' in
	// VP9) are critical for codec matching.
	Parameters RtpCodecSpecificParameters `json:"parameters,omitempty"`

	// RtcpFeedback is transport layer and codec-specific feedback messages for this codec.
	RtcpFeedback []*RtcpFeedback `json:"rtcpFeedback,omitempty"`
}

func (r RtpCodecParameters) isRtxCodec() bool {
	return strings.HasSuffix(strings.ToLower(r.MimeType), "/rtx")
}

// RtcpFeedback provides information on RTCP feedback messages for a specific codec.
// Those messages can be transport layer feedback messages or codec-specific feedback
// messages. The list of RTCP feedbacks supported by mediasoup is defined in the
// supported_rtp_capabilities.go file.
type RtcpFeedback struct {
	// Type is RTCP feedback type.
	Type string `json:"type"`

	// Parameter is RTCP feedback parameter.
	Parameter string `json:"parameter,omitempty"`
}

// RtpEncodingParameters provides information relating to an encoding, which represents
// a media RTP stream and its associated RTX stream (if any).
type RtpEncodingParameters struct {
	// SSRC of media.
	Ssrc uint32 `json:"ssrc,omitempty"`

	// RID RTP extension value. Must be unique.
	Rid string `json:"rid,omitempty"`

	// CodecPayloadType is the codec payload type this encoding affects.
	// If unset, first media codec is chosen.
	CodecPayloadType *uint8 `json:"codecPayloadType,omitempty"`

	// RTX stream information. It must contain a numeric ssrc field indicating
	// the RTX SSRC.
	Rtx *RtpEncodingRtx `json:"rtx,omitempty"`

	// Dtx indicates whether discontinuous RTP transmission will be used. Useful
	// for audio (if the codec supports it) and for video screen sharing (when
	// static content is being transmitted, this option disables the RTP
	// inactivity checks in mediasoup). Default false.
	Dtx bool `json:"dtx,omitempty"`

	// ScalabilityMode defines spatial and temporal layers in the RTP stream (e.g. 'L1T3').
	// See webrtc-svc.
	ScalabilityMode string `json:"scalabilityMode,omitempty"`

	// Others.
	ScaleResolutionDownBy int    `json:"scaleResolutionDownBy,omitempty"`
	MaxBitrate            uint32 `json:"maxBitrate,omitempty"`
}

// RtpEncodingRtx represents the associated RTX stream for RTP stream.
type RtpEncodingRtx struct {
	// SSRC of media.
	Ssrc uint32 `json:"ssrc"`
}

// RtpHeaderExtensionParameters defines a RTP header extension within the RTP parameters.
// The list of RTP header extensions supported by mediasoup is defined in the
// supported_rtp_capabilities.go file.
//
// mediasoup does not currently support encrypted RTP header extensions and no
// parameters are currently considered.
type RtpHeaderExtensionParameters struct {
	// URI of the RTP header extension, as defined in RFC 5285.
	Uri string `json:"uri"`

	// Id is the numeric identifier that goes in the RTP packet. Must be unique.
	Id uint8 `json:"id"`

	// Encrypt if true, the value in the header is encrypted as per RFC 6904. Default false.
	Encrypt bool `json:"encrypt,omitempty"`

	// Parameters is the configuration parameters for the header extension.
	Parameters RtpCodecSpecificParameters `json:"parameters,omitempty"`
}

// RtcpParameters provides information on RTCP settings within the RTP parameters.
//
// If no cname is given in a producer's RTP parameters, the mediasoup transport
// will choose a random one that will be used into RTCP SDES messages sent to
// all its associated consumers.
//
// mediasoup assumes reducedSize to always be true.
type RtcpParameters struct {
	// Cname is the Canonical Name (CNAME) used by RTCP (e.g. in SDES messages).
	Cname string `json:"cname,omitempty"`

	// ReducedSize defines whether reduced size RTCP RFC 5506 is configured (if true) or
	// compound RTCP as specified in RFC 3550 (if false). Default true.
	ReducedSize *bool `json:"reducedSize,omitempty"`
}

// RtpCodecSpecificParameters is the Codec-specific parameters available for signaling.
// Some parameters (such as 'packetization-mode' and 'profile-level-id' in H264 or
// 'profile-id' in VP9) are critical for codec matching.
// RtpCodecSpecificParameters represents codec-specific parameters for RTP transmission.
// It includes settings for various codecs:
//
// H.264 specific parameters:
//   - PacketizationMode: Indicates the packetization mode
//   - ProfileLevelId: Specifies the H.264 profile level ID
//   - LevelAsymmetryAllowed: Indicates if level asymmetry is allowed
//
// VP9/AV1 specific parameters:
//   - ProfileId: Profile ID for VP9 or AV1 codec
//
// RTX specific parameters:
//   - Apt: Associated payload type for RTX
//
// OPUS specific parameters:
//   - SpropStereo: Enables stereo audio
//   - Useinbandfec: Enables in-band forward error correction
//   - Usedtx: Enables discontinuous transmission
//   - Maxplaybackrate: Maximum playback rate
//   - Maxaveragebitrate: Maximum average bitrate
//   - Ptime: Preferred packet duration
//   - ChannelMapping: Defines audio channel organization
//   - NumStreams: Number of streams for multichannel audio
//   - CoupledStreams: Number of coupled streams
//
// WebRTC specific parameters (for libwebrtc browsers):
//   - XGoogleStartBitrate: Initial video bitrate
//   - XGoogleMaxBitrate: Maximum video bitrate
//   - XGoogleMinBitrate: Minimum video bitrate
type RtpCodecSpecificParameters struct {
	// PacketizationMode controls how H.264 video frames are split into RTP packets
	// when sending over the network.
	//
	// PacketizationMode = 0:
	//   - Simple: 1 NAL unit per RTP packet.
	//   - No fragmentation allowed.
	//   - Large frames must fit into a single RTP packet (limited by MTU, ~1200 bytes usually).
	//   - Easier for simple hardware.
	//   - Inefficient for high-resolution video.
	// PacketizationMode = 1:
	// 	 - Flexible: allows fragmenting a single large NAL into multiple RTP packets (using Fragmentation Units, FU-A).
	// 	 - Also allows packing multiple small NALs into one RTP packet (STAP-A aggregation).
	// 	 - Better for high-bitrate / large frames (HD, 4K video).
	// 	 - Widely used in WebRTC.
	// 	 - Slightly more complicated RTP parsing.
	// PacketizationMode = 2:
	// 	 - Interleaving allows out-of-order delivery with extra reordering at the receiver.
	// 	 - Very rare.
	// 	 - WebRTC almost never supports mode 2.
	// 	 - Mostly ignored in modern real-time video systems.
	PacketizationMode uint32 `json:"packetization-mode,omitempty"`

	// ProfileLevelId specifies the H.264 profile level ID.
	// It’s a 6-character hexadecimal string (24 bits) that encodes:
	// | Field            | Bits   | Meaning                            |
	// |:-----------------|:-------|:-----------------------------------|
	// | Profile_idc      | 8 bits | H.264 Profile (Baseline, Main, etc.) |
	// | Profile_iop      | 8 bits | Profile compatibility flags       |
	// | Level_idc        | 8 bits | H.264 Level (resolution, bitrate) |
	ProfileLevelId string `json:"profile-level-id,omitempty"`

	// LevelAsymmetryAllowed indicates whether the encoder and decoder are allowed
	// to operate at different H.264 levels. In practice, Mobile devices (weaker CPU/GPU)
	// can decode high-bitrate streams but send lower-quality streams.
	// Almost always set to 1 in WebRTC.
	LevelAsymmetryAllowed uint32 `json:"level-asymmetry-allowed,omitempty"`

	// ProfileId is the profile id used by VP9 or AV1 codec.
	ProfileId string `json:"profile-id,omitempty"`

	// Apt is the apt value used by rtx codec.
	Apt uint8 `json:"apt,omitempty"`

	// SpropStereo enable OPUS stereo (if the audio source is stereo).
	SpropStereo uint8 `json:"sprop-stereo,omitempty"`

	// Useinbandfec enable OPUS in band FEC.
	Useinbandfec uint8 `json:"useinbandfec,omitempty"`

	// Usedtx enable OPUS discontinuous transmission.
	Usedtx uint8 `json:"usedtx,omitempty"`

	// Maxplaybackrate set OPUS maximum playback rate.
	Maxplaybackrate uint32 `json:"maxplaybackrate,omitempty"`

	// Maxaveragebitrate set OPUS maximum average bitrate.
	Maxaveragebitrate uint32 `json:"maxaveragebitrate,omitempty"`

	// Ptime set OPUS preferred duration of media represented by a packet. default 20ms.
	Ptime uint32 `json:"ptime,omitempty"`

	// ChannelMapping is a mechanism Opus uses to describe how audio channels are organized inside
	// the bitstream, especially for multichannel audio (more than 2 channels, like surround sound).
	//
	// How Channel Mapping Works
	//
	// The Opus header contains:
	//   - channel count (how many channels)
	//   - channel mapping family (an integer saying which mapping rule to apply)
	//   - Optionally, a channel mapping table.
	//
	// The mapping family defines the meaning of each audio channel:
	//   - Family 0: Mono or stereo (1 or 2 channels). No mapping needed.
	//   - Family 1: Multichannel surround sound (standard layouts like 5.1).
	//   - Family 2: Ambisonics (for 3D audio / VR).
	//
	// Family 1 example (5.1 surround):
	// +-----------------+            +----------------+
	// | Encoded Opus    |  --->      | Output Speakers |
	// | Channel 0       |  --->      | Front Left      |
	// | Channel 1       |  --->      | Front Right     |
	// | Channel 2       |  --->      | Center          |
	// | Channel 3       |  --->      | Subwoofer (LFE) |
	// | Channel 4       |  --->      | Back Left       |
	// | Channel 5       |  --->      | Back Right      |
	// +-----------------+            +----------------+
	//
	// Important
	//   - If you’re using 1 or 2 channels, you don’t need to care — Opus will automatically
	//     decode it right (no special mapping).
	//   - For 3+ channels, you must set the correct channel mapping during encoding.
	//   - If the mapping is wrong, sound channels will go to the wrong speakers
	//     (e.g., voice comes out from the subwoofer).
	ChannelMapping string `json:"channel_mapping,omitempty"`

	// NumStreams is the number of independent Opus streams inside one encoded packet.
	NumStreams uint32 `json:"num_streams,omitempty"`

	// CoupledStreams specifics how many streams are coupled (stereo). Coupled Streams are
	// streams where two audio channels are encoded together as a stereo pair.
	CoupledStreams uint32 `json:"coupled_streams,omitempty"`

	// XGoogleStartBitrate is just for libwebrtc based browsers. Set video initial bitrate.
	XGoogleStartBitrate uint32 `json:"x-google-start-bitrate,omitempty"` // used by video

	// XGoogleMaxBitrate is just for libwebrtc based browsers. Set video maximum bitrate.
	XGoogleMaxBitrate uint32 `json:"x-google-max-bitrate,omitempty"` // used by video

	// XGoogleMinBitrate is just for libwebrtc based browsers. Set video initial bitrate.
	XGoogleMinBitrate uint32 `json:"x-google-min-bitrate,omitempty"` // used by video
}
