package mediasoup

import (
	"errors"
	"strings"

	FbsRtpPacket "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/RtpPacket"
	FbsRtpParameters "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/RtpParameters"
	FbsSctpAssociation "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/SctpAssociation"
	FbsSctpParameters "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/SctpParameters"
	FbsSrtpParameters "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/SrtpParameters"
	FbsTransport "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/Transport"
	FbsWebRtcTransport "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/WebRtcTransport"
)

func parseTransportTuple(tuple *FbsTransport.TupleT) *TransportTuple {
	if tuple == nil {
		return nil
	}
	return &TransportTuple{
		LocalAddress: tuple.LocalAddress,
		LocalPort:    tuple.LocalPort,
		Protocol:     TransportProtocol(strings.ToLower(tuple.Protocol.String())),
		RemoteIp:     tuple.RemoteIp,
		RemotePort:   tuple.RemotePort,
	}
}

func parseRtpPacketDump(packet *FbsRtpPacket.DumpT) H {
	if packet == nil {
		return nil
	}
	return H{
		"payloadType":        packet.PayloadType,
		"sequenceNumber":     packet.SequenceNumber,
		"timestamp":          packet.Timestamp,
		"marker":             packet.Marker,
		"ssrc":               packet.Ssrc,
		"isKeyFrame":         packet.IsKeyFrame,
		"size":               packet.Size,
		"payloadSize":        packet.PayloadSize,
		"spatialLayer":       packet.SpatialLayer,
		"temporalLayer":      packet.TemporalLayer,
		"mid":                packet.Mid,
		"rid":                packet.Rid,
		"rrid":               packet.Rrid,
		"wideSequenceNumber": packet.WideSequenceNumber,
	}
}

func parseSctpParameters(sctpParameters *FbsSctpParameters.SctpParametersT) *SctpParameters {
	if sctpParameters == nil {
		return nil
	}
	return &SctpParameters{
		Port:                        sctpParameters.Port,
		MaxSendMessageSize:          sctpParameters.MaxSendMessageSize,
		MaxReceiveMessageSize:       sctpParameters.MaxReceiveMessageSize,
		SendBufferSize:              sctpParameters.SendBufferSize,
		PerStreamSendQueueLimit:     sctpParameters.PerStreamSendQueueLimit,
		MaxReceiverWindowBufferSize: sctpParameters.MaxReceiverWindowBufferSize,
		IsDataChannel:               sctpParameters.IsDataChannel,
	}
}

// parseSctpNegotiatedCapabilities returns nil until the SCTP association has been negotiated,
// since the worker reports zeroed capabilities while the association is not connected.
func parseSctpNegotiatedCapabilities(capabilities *FbsSctpAssociation.SctpNegotiatedCapabilitiesT) *SctpNegotiatedCapabilities {
	if capabilities == nil ||
		(capabilities.NegotiatedMaxOutboundStreams == 0 && capabilities.NegotiatedMaxInboundStreams == 0) {
		return nil
	}
	return &SctpNegotiatedCapabilities{
		NegotiatedMaxOutboundStreams: capabilities.NegotiatedMaxOutboundStreams,
		NegotiatedMaxInboundStreams:  capabilities.NegotiatedMaxInboundStreams,
	}
}

// convertTransportOptions builds the base transport options.
func convertTransportOptions(
	bwe BweOptions,
	defaultSctpZeroChecksum SctpZeroChecksum,
	enableSctp bool,
	options SctpOptions,
	isDataChannel bool,
) (*FbsTransport.OptionsT, error) {
	o := options.withDefaults(defaultSctpZeroChecksum)
	sctpZeroChecksum, err := serializeSctpZeroChecksum(o.SctpZeroChecksum)
	if err != nil {
		return nil, err
	}
	bwe = bwe.withDefaults()

	return &FbsTransport.OptionsT{
		EnableBwe:                                   *bwe.EnableBwe,
		InitialAvailableOutgoingBitrate:             positiveUint64(bwe.InitialAvailableOutgoingBitrate),
		MaxIncomingBitrate:                          positiveUint64(bwe.MaxIncomingBitrate),
		MaxOutgoingBitrate:                          positiveUint64(bwe.MaxOutgoingBitrate),
		MinOutgoingBitrate:                          positiveUint64(bwe.MinOutgoingBitrate),
		EnableSctp:                                  enableSctp,
		MaxSendMessageSize:                          o.MaxSendMessageSize,
		MaxReceiveMessageSize:                       o.MaxReceiveMessageSize,
		SctpSendBufferSize:                          o.SctpSendBufferSize,
		SctpPerStreamSendQueueLimit:                 o.SctpPerStreamSendQueueLimit,
		SctpMaxReceiverWindowBufferSize:             o.SctpMaxReceiverWindowBufferSize,
		SctpDefaultStreamBufferedAmountLowThreshold: o.SctpDefaultStreamBufferedAmountLowThreshold,
		SctpZeroChecksum:                            sctpZeroChecksum,
		IsDataChannel:                               isDataChannel,
	}, nil
}

func serializeSctpZeroChecksum(value SctpZeroChecksum) (FbsTransport.SctpZeroChecksum, error) {
	switch value {
	case SctpZeroChecksumNone:
		return FbsTransport.SctpZeroChecksumNONE, nil
	case SctpZeroChecksumSctpOverDtls:
		return FbsTransport.SctpZeroChecksumSCTP_OVER_DTLS, nil
	case SctpZeroChecksumTrustedNetwork:
		return FbsTransport.SctpZeroChecksumTRUSTED_NETWORK, nil
	default:
		return 0, errors.New("invalid SctpZeroChecksum: " + string(value))
	}
}

func positiveUint64(value uint64) *uint64 {
	if value == 0 {
		return nil
	}
	return ref(value)
}

func parseSrtpParameters(srtpParameters *FbsSrtpParameters.SrtpParametersT) *SrtpParameters {
	if srtpParameters == nil {
		return nil
	}
	return &SrtpParameters{
		CryptoSuite: SrtpCryptoSuite(srtpParameters.CryptoSuite.String()),
		KeyBase64:   srtpParameters.KeyBase64,
	}
}

func convertSrtpParameters(srtpParameters *SrtpParameters) (*FbsSrtpParameters.SrtpParametersT, error) {
	if srtpParameters == nil {
		return nil, nil
	}
	key := strings.ToUpper(string(srtpParameters.CryptoSuite))
	cryptoSuite, ok := FbsSrtpParameters.EnumValuesSrtpCryptoSuite[key]
	if !ok {
		return nil, errors.New("invalid SrtpCryptoSuite: " + key)
	}
	keyBase64 := srtpParameters.KeyBase64
	if len(keyBase64) == 0 {
		return nil, errors.New("invalid SrtpParameters: missing KeyBase64")
	}
	return &FbsSrtpParameters.SrtpParametersT{
		CryptoSuite: cryptoSuite,
		KeyBase64:   srtpParameters.KeyBase64,
	}, nil
}

func convertRtpEncodingParameters(encoding *RtpEncodingParameters) *FbsRtpParameters.RtpEncodingParametersT {
	return &FbsRtpParameters.RtpEncodingParametersT{
		Ssrc:             orElse(encoding.Ssrc > 0, ref(encoding.Ssrc), nil),
		Rid:              encoding.Rid,
		CodecPayloadType: encoding.CodecPayloadType,
		Rtx: ifElse(encoding.Rtx != nil, func() *FbsRtpParameters.RtxT {
			return &FbsRtpParameters.RtxT{Ssrc: encoding.Rtx.Ssrc}
		}),
		Dtx:             encoding.Dtx,
		ScalabilityMode: encoding.ScalabilityMode,
		MaxBitrate:      orElse(encoding.MaxBitrate > 0, ref(encoding.MaxBitrate), nil),
	}
}

func convertTransportListenInfo(info TransportListenInfo) *FbsTransport.ListenInfoT {
	return &FbsTransport.ListenInfoT{
		Protocol:         orElse(info.Protocol == TransportProtocolTCP, FbsTransport.ProtocolTCP, FbsTransport.ProtocolUDP),
		Ip:               info.Ip,
		AnnouncedAddress: info.AnnouncedAddress,
		ExposeInternalIp: info.ExposeInternalIp,
		Port:             info.Port,
		PortRange: &FbsTransport.PortRangeT{
			Min: info.PortRange.Min,
			Max: info.PortRange.Max,
		},
		Flags: &FbsTransport.SocketFlagsT{
			Ipv6Only:     info.Flags.IPv6Only,
			UdpReusePort: info.Flags.UDPReusePort,
		},
		SendBufferSize: info.SendBufferSize,
		RecvBufferSize: info.RecvBufferSize,
	}
}

func convertDtlsFingerprint(item DtlsFingerprint) *FbsWebRtcTransport.FingerprintT {
	var algorithm FbsWebRtcTransport.FingerprintAlgorithm

	switch item.Algorithm {
	case "sha-1":
		algorithm = FbsWebRtcTransport.FingerprintAlgorithmSHA1

	case "sha-224":
		algorithm = FbsWebRtcTransport.FingerprintAlgorithmSHA224

	case "sha-256":
		algorithm = FbsWebRtcTransport.FingerprintAlgorithmSHA256

	case "sha-384":
		algorithm = FbsWebRtcTransport.FingerprintAlgorithmSHA384

	case "sha-512":
		algorithm = FbsWebRtcTransport.FingerprintAlgorithmSHA512

	default:
		key := strings.ToUpper(strings.Join(strings.Split(item.Algorithm, "-"), ""))
		algorithm = FbsWebRtcTransport.EnumValuesFingerprintAlgorithm[key]
	}

	// avoid mediasoup crash
	if len(item.Value) == 0 {
		item.Value = "unknown"
	}

	return &FbsWebRtcTransport.FingerprintT{
		Algorithm: algorithm,
		Value:     item.Value,
	}
}

func parseDtlsFingerprint(item *FbsWebRtcTransport.FingerprintT) DtlsFingerprint {
	var algorithm string

	switch item.Algorithm {
	case FbsWebRtcTransport.FingerprintAlgorithmSHA1:
		algorithm = "sha-1"

	case FbsWebRtcTransport.FingerprintAlgorithmSHA224:
		algorithm = "sha-224"

	case FbsWebRtcTransport.FingerprintAlgorithmSHA256:
		algorithm = "sha-256"

	case FbsWebRtcTransport.FingerprintAlgorithmSHA384:
		algorithm = "sha-384"

	case FbsWebRtcTransport.FingerprintAlgorithmSHA512:
		algorithm = "sha-512"

	default:
		algorithm = "sha-" + strings.TrimPrefix(item.Algorithm.String(), "SHA")
	}

	return DtlsFingerprint{
		Algorithm: algorithm,
		Value:     item.Value,
	}
}
