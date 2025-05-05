package mediasoup

import (
	"errors"
	"strings"

	FbsRtpParameters "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/RtpParameters"
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

func parseSctpParameters(sctpParameters *FbsSctpParameters.SctpParametersT) *SctpParameters {
	if sctpParameters == nil {
		return nil
	}
	return &SctpParameters{
		Port:               sctpParameters.Port,
		OS:                 sctpParameters.Os,
		MIS:                sctpParameters.Mis,
		MaxMessageSize:     sctpParameters.MaxMessageSize,
		SendBufferSize:     sctpParameters.SendBufferSize,
		SctpBufferedAmount: sctpParameters.SctpBufferedAmount,
		IsDataChannel:      sctpParameters.IsDataChannel,
	}
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
