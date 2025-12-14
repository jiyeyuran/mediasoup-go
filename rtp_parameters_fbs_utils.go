package mediasoup

import (
	"encoding/json"
	"math"

	FbsRtpParameters "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/RtpParameters"
)

func convertRtpParameters(rtpParameters *RtpParameters) *FbsRtpParameters.RtpParametersT {
	return &FbsRtpParameters.RtpParametersT{
		Mid: rtpParameters.Mid,
		Codecs: collect(rtpParameters.Codecs,
			func(item *RtpCodecParameters) *FbsRtpParameters.RtpCodecParametersT {
				return &FbsRtpParameters.RtpCodecParametersT{
					MimeType:    item.MimeType,
					PayloadType: item.PayloadType,
					ClockRate:   item.ClockRate,
					Channels:    orElse(item.Channels > 0, ref(item.Channels), nil),
					Parameters:  convertRtpCodecSpecificParameters(&item.Parameters),
					RtcpFeedback: collect(item.RtcpFeedback,
						func(item *RtcpFeedback) *FbsRtpParameters.RtcpFeedbackT {
							return &FbsRtpParameters.RtcpFeedbackT{
								Type:      item.Type,
								Parameter: item.Parameter,
							}
						}),
				}
			}),
		HeaderExtensions: collect(rtpParameters.HeaderExtensions,
			func(item *RtpHeaderExtensionParameters) *FbsRtpParameters.RtpHeaderExtensionParametersT {
				return &FbsRtpParameters.RtpHeaderExtensionParametersT{
					Uri:        convertHeaderExtensionUri(item.Uri),
					Id:         item.Id,
					Encrypt:    item.Encrypt,
					Parameters: convertRtpCodecSpecificParameters(&item.Parameters),
				}
			}),
		Encodings: collect(rtpParameters.Encodings,
			func(item *RtpEncodingParameters) *FbsRtpParameters.RtpEncodingParametersT {
				return &FbsRtpParameters.RtpEncodingParametersT{
					Ssrc:             orElse(item.Ssrc > 0, ref(item.Ssrc), nil),
					Rid:              item.Rid,
					CodecPayloadType: item.CodecPayloadType,
					Rtx: ifElse(item.Rtx != nil, func() *FbsRtpParameters.RtxT {
						return &FbsRtpParameters.RtxT{
							Ssrc: item.Rtx.Ssrc,
						}
					}),
					Dtx:             item.Dtx,
					ScalabilityMode: item.ScalabilityMode,
				}
			},
		),
		Rtcp: ifElse(rtpParameters.Rtcp != nil, func() *FbsRtpParameters.RtcpParametersT {
			return &FbsRtpParameters.RtcpParametersT{
				Cname:       rtpParameters.Rtcp.Cname,
				ReducedSize: unref(rtpParameters.Rtcp.ReducedSize, true),
			}
		}, func() *FbsRtpParameters.RtcpParametersT {
			return &FbsRtpParameters.RtcpParametersT{
				ReducedSize: true,
			}
		}),
		Msid: rtpParameters.Msid,
	}
}

func parseRtpParameters(rtpParameters *FbsRtpParameters.RtpParametersT) *RtpParameters {
	return &RtpParameters{
		Mid:              rtpParameters.Mid,
		Codecs:           collect(rtpParameters.Codecs, parseRtpCodecParameters),
		HeaderExtensions: collect(rtpParameters.HeaderExtensions, parseRtpHeaderExtensionParameters),
		Encodings:        collect(rtpParameters.Encodings, parseRtpEncodingParameters),
		Rtcp:             parseRtcpParameters(rtpParameters.Rtcp),
		Msid:             rtpParameters.Msid,
	}
}

func parseRtpCodecParameters(codec *FbsRtpParameters.RtpCodecParametersT) *RtpCodecParameters {
	h := H{}

	for _, param := range codec.Parameters {
		switch param.Value.Type {
		case FbsRtpParameters.ValueBoolean:
			h[param.Name] = param.Value.Value.(*FbsRtpParameters.BooleanT).Value

		case FbsRtpParameters.ValueInteger32:
			h[param.Name] = param.Value.Value.(*FbsRtpParameters.Integer32T).Value

		case FbsRtpParameters.ValueDouble:
			h[param.Name] = param.Value.Value.(*FbsRtpParameters.DoubleT).Value

		case FbsRtpParameters.ValueString:
			h[param.Name] = param.Value.Value.(*FbsRtpParameters.StringT).Value

		case FbsRtpParameters.ValueInteger32Array:
			h[param.Name] = param.Value.Value.(*FbsRtpParameters.Integer32ArrayT).Value
		}
	}

	parameters := RtpCodecSpecificParameters{}

	if len(h) > 0 {
		data, _ := json.Marshal(h)
		json.Unmarshal(data, &parameters)
	}

	return &RtpCodecParameters{
		MimeType:     codec.MimeType,
		PayloadType:  codec.PayloadType,
		ClockRate:    codec.ClockRate,
		Channels:     unref(codec.Channels),
		Parameters:   parameters,
		RtcpFeedback: collect(codec.RtcpFeedback, parseRtcpFeedback),
	}
}

func convertRtpCodecSpecificParameters(params *RtpCodecSpecificParameters) []*FbsRtpParameters.ParameterT {
	h := H{}
	data, _ := json.Marshal(params)
	json.Unmarshal(data, &h)

	parameters := make([]*FbsRtpParameters.ParameterT, 0, len(h))

	for k, v := range h {
		switch val := v.(type) {
		case string:
			parameters = append(parameters, &FbsRtpParameters.ParameterT{
				Name: k,
				Value: &FbsRtpParameters.ValueT{
					Type: FbsRtpParameters.ValueString,
					Value: &FbsRtpParameters.StringT{
						Value: val,
					},
				},
			})

		case bool:
			parameters = append(parameters, &FbsRtpParameters.ParameterT{
				Name: k,
				Value: &FbsRtpParameters.ValueT{
					Type: FbsRtpParameters.ValueBoolean,
					Value: &FbsRtpParameters.BooleanT{
						Value: orElse[byte](val, 1, 0),
					},
				},
			})

		case float64:
			// handle Integer.
			if math.Floor(val) == val {
				parameters = append(parameters, &FbsRtpParameters.ParameterT{
					Name: k,
					Value: &FbsRtpParameters.ValueT{
						Type: FbsRtpParameters.ValueInteger32,
						Value: &FbsRtpParameters.Integer32T{
							Value: int32(val),
						},
					},
				})
			} else {
				parameters = append(parameters, &FbsRtpParameters.ParameterT{
					Name: k,
					Value: &FbsRtpParameters.ValueT{
						Type: FbsRtpParameters.ValueDouble,
						Value: &FbsRtpParameters.DoubleT{
							Value: val,
						},
					},
				})
			}

		case []int32:
			parameters = append(parameters, &FbsRtpParameters.ParameterT{
				Name: k,
				Value: &FbsRtpParameters.ValueT{
					Type: FbsRtpParameters.ValueInteger32Array,
					Value: &FbsRtpParameters.Integer32ArrayT{
						Value: v.([]int32),
					},
				},
			})
		}
	}

	return parameters
}

func parseRtpHeaderExtensionParameters(headerExtension *FbsRtpParameters.RtpHeaderExtensionParametersT) *RtpHeaderExtensionParameters {
	uri := ""
	switch headerExtension.Uri {
	case FbsRtpParameters.RtpHeaderExtensionUriMid:
		uri = "urn:ietf:params:rtp-hdrext:sdes:mid"

	case FbsRtpParameters.RtpHeaderExtensionUriRtpStreamId:
		uri = "urn:ietf:params:rtp-hdrext:sdes:rtp-stream-id"

	case FbsRtpParameters.RtpHeaderExtensionUriRepairRtpStreamId:
		uri = "urn:ietf:params:rtp-hdrext:sdes:repaired-rtp-stream-id"

	case FbsRtpParameters.RtpHeaderExtensionUriAbsSendTime:
		uri = "http://www.webrtc.org/experiments/rtp-hdrext/abs-send-time"

	case FbsRtpParameters.RtpHeaderExtensionUriTransportWideCcDraft01:
		uri = "http://www.ietf.org/id/draft-holmer-rmcat-transport-wide-cc-extensions-01"

	case FbsRtpParameters.RtpHeaderExtensionUriSsrcAudioLevel:
		uri = "urn:ietf:params:rtp-hdrext:ssrc-audio-level"

	case FbsRtpParameters.RtpHeaderExtensionUriDependencyDescriptor:
		uri = "https://aomediacodec.github.io/av1-rtp-spec/#dependency-descriptor-rtp-header-extension"

	case FbsRtpParameters.RtpHeaderExtensionUriVideoOrientation:
		uri = "urn:3gpp:video-orientation"

	case FbsRtpParameters.RtpHeaderExtensionUriTimeOffset:
		uri = "urn:ietf:params:rtp-hdrext:toffset"

	case FbsRtpParameters.RtpHeaderExtensionUriAbsCaptureTime:
		uri = "http://www.webrtc.org/experiments/rtp-hdrext/abs-capture-time"

	case FbsRtpParameters.RtpHeaderExtensionUriPlayoutDelay:
		uri = "urn:ietf:params:rtp-hdrext:playout-delay"

	case FbsRtpParameters.RtpHeaderExtensionUriMediasoupPacketId:
		uri = "urn:mediasoup:params:rtp-hdrext:packet-id"
	}

	return &RtpHeaderExtensionParameters{
		Uri:     uri,
		Id:      headerExtension.Id,
		Encrypt: headerExtension.Encrypt,
	}
}

func convertHeaderExtensionUri(uri string) FbsRtpParameters.RtpHeaderExtensionUri {
	switch uri {
	case "urn:ietf:params:rtp-hdrext:sdes:mid":
		return FbsRtpParameters.RtpHeaderExtensionUriMid

	case "urn:ietf:params:rtp-hdrext:sdes:rtp-stream-id":
		return FbsRtpParameters.RtpHeaderExtensionUriRtpStreamId

	case "urn:ietf:params:rtp-hdrext:sdes:repaired-rtp-stream-id":
		return FbsRtpParameters.RtpHeaderExtensionUriRepairRtpStreamId

	case "http://www.webrtc.org/experiments/rtp-hdrext/abs-send-time":
		return FbsRtpParameters.RtpHeaderExtensionUriAbsSendTime

	case "http://www.ietf.org/id/draft-holmer-rmcat-transport-wide-cc-extensions-01":
		return FbsRtpParameters.RtpHeaderExtensionUriTransportWideCcDraft01

	case "urn:ietf:params:rtp-hdrext:ssrc-audio-level":
		return FbsRtpParameters.RtpHeaderExtensionUriSsrcAudioLevel

	case "https://aomediacodec.github.io/av1-rtp-spec/#dependency-descriptor-rtp-header-extension":
		return FbsRtpParameters.RtpHeaderExtensionUriDependencyDescriptor

	case "urn:3gpp:video-orientation":
		return FbsRtpParameters.RtpHeaderExtensionUriVideoOrientation

	case "urn:ietf:params:rtp-hdrext:toffset":
		return FbsRtpParameters.RtpHeaderExtensionUriTimeOffset

	case "http://www.webrtc.org/experiments/rtp-hdrext/abs-capture-time":
		return FbsRtpParameters.RtpHeaderExtensionUriAbsCaptureTime

	case "urn:ietf:params:rtp-hdrext:playout-delay":
		return FbsRtpParameters.RtpHeaderExtensionUriPlayoutDelay

	case "urn:mediasoup:params:rtp-hdrext:packet-id":
		return FbsRtpParameters.RtpHeaderExtensionUriMediasoupPacketId

	default:
		return FbsRtpParameters.EnumValuesRtpHeaderExtensionUri[uri]
	}
}

func parseRtcpFeedback(rtcpFeedback *FbsRtpParameters.RtcpFeedbackT) *RtcpFeedback {
	return &RtcpFeedback{
		Type:      rtcpFeedback.Type,
		Parameter: rtcpFeedback.Parameter,
	}
}

func parseRtpEncodingParameters(encoding *FbsRtpParameters.RtpEncodingParametersT) *RtpEncodingParameters {
	return &RtpEncodingParameters{
		Ssrc:             unref(encoding.Ssrc),
		Rid:              encoding.Rid,
		CodecPayloadType: encoding.CodecPayloadType,
		Rtx: ifElse(encoding.Rtx != nil, func() *RtpEncodingRtx {
			return &RtpEncodingRtx{Ssrc: encoding.Rtx.Ssrc}
		}),
		Dtx:             encoding.Dtx,
		ScalabilityMode: encoding.ScalabilityMode,
		MaxBitrate:      unref(encoding.MaxBitrate),
	}
}

func parseRtcpParameters(rtcp *FbsRtpParameters.RtcpParametersT) *RtcpParameters {
	if rtcp == nil {
		return nil
	}
	return &RtcpParameters{
		Cname:       rtcp.Cname,
		ReducedSize: &rtcp.ReducedSize,
	}
}
