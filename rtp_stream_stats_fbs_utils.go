package mediasoup

import (
	FbsRtpParameters "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/RtpParameters"
	FbsRtpStream "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/RtpStream"
)

func parseRtpStreamStats(stats *FbsRtpStream.StatsT) *RtpStreamStats {
	switch stats.Data.Type {
	case FbsRtpStream.StatsDataRecvStats:
		return parseRtpStreamRecvStats(stats.Data.Value.(*FbsRtpStream.RecvStatsT))

	case FbsRtpStream.StatsDataSendStats:
		return parseSendStreamStats(stats.Data.Value.(*FbsRtpStream.SendStatsT))

	case FbsRtpStream.StatsDataBaseStats:
		return parseBaseStreamStats(stats.Data.Value.(*FbsRtpStream.BaseStatsT))

	default:
		return nil
	}
}

func parseRtpStreamRecvStats(recvStats *FbsRtpStream.RecvStatsT) *RtpStreamStats {
	stats := parseBaseStreamStats(recvStats.Base.Data.Value.(*FbsRtpStream.BaseStatsT))
	stats.Type = "inbound-rtp"
	stats.Jitter = recvStats.Jitter
	stats.PacketCount = recvStats.PacketCount
	stats.ByteCount = recvStats.ByteCount
	stats.Bitrate = recvStats.Bitrate
	stats.BitrateByLayer = parseBitrateByLayer(recvStats)

	return stats
}

func parseSendStreamStats(sendStats *FbsRtpStream.SendStatsT) *RtpStreamStats {
	stats := parseBaseStreamStats(sendStats.Base.Data.Value.(*FbsRtpStream.BaseStatsT))
	stats.Type = "outbound-rtp"
	stats.PacketCount = sendStats.PacketCount
	stats.ByteCount = sendStats.ByteCount
	stats.Bitrate = sendStats.Bitrate

	return stats
}

func parseBaseStreamStats(stats *FbsRtpStream.BaseStatsT) *RtpStreamStats {
	return &RtpStreamStats{
		Timestamp:            stats.Timestamp,
		Ssrc:                 stats.Ssrc,
		Kind:                 orElse(stats.Kind == FbsRtpParameters.MediaKindVIDEO, MediaKindVideo, MediaKindAudio),
		MimeType:             stats.MimeType,
		PacketsLost:          stats.PacketsLost,
		FractionLost:         stats.FractionLost,
		PacketsDiscarded:     stats.PacketsDiscarded,
		PacketsRetransmitted: stats.PacketsRetransmitted,
		PacketsRepaired:      stats.PacketsRepaired,
		NackCount:            stats.NackCount,
		NackPacketCount:      stats.NackPacketCount,
		PliCount:             stats.PliCount,
		FirCount:             stats.FirCount,
		Score:                stats.Score,
		Rid:                  stats.Rid,
		RtxSsrc:              stats.RtxSsrc,
		RoundTripTime:        stats.RoundTripTime,
		RtxPacketsDiscarded:  stats.RtxPacketsDiscarded,
	}
}

func parseBitrateByLayer(stats *FbsRtpStream.RecvStatsT) map[string]uint32 {
	if stats == nil {
		return nil
	}
	bitrateByLayer := make(map[string]uint32)

	for _, layer := range stats.BitrateByLayer {
		bitrateByLayer[layer.Layer] = layer.Bitrate
	}

	return bitrateByLayer
}
