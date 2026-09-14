// Package quiccongestion provides the mihomo congestion controllers (cubic,
// new reno, BBR v1/v2, brutal) for sagernet/quic-go connections. The
// implementations live in the meta1 and meta2 subpackages, ported from
// mihomo transport/tuic/congestion and transport/tuic/congestion_v2.
package quiccongestion

import (
	"github.com/sagernet/quic-go"
	"github.com/sagernet/quic-go/congestion"
	"github.com/sagernet/sing-box/common/quiccongestion/meta1"
	"github.com/sagernet/sing-box/common/quiccongestion/meta2"
)

const (
	// DefaultStreamReceiveWindow is the default stream receive window used by mihomo QUIC transports.
	DefaultStreamReceiveWindow = 15728640 // 15 MB/s
	// DefaultConnectionReceiveWindow is the default connection receive window used by mihomo QUIC transports.
	DefaultConnectionReceiveWindow = 67108864 // 64 MB/s
)

// SetCongestionController installs the named congestion controller on a QUIC
// connection. Supported names: cubic, new_reno, bbr_meta_v1, bbr_meta_v2/bbr.
// Unknown names are ignored and the QUIC default is kept.
func SetCongestionController(quicConn *quic.Conn, cc string, cwnd int, profile string) {
	if cwnd == 0 {
		cwnd = 32
	}
	initialPacketSize := quicConn.InitialPacketSize()
	switch cc {
	case "cubic":
		quicConn.SetCongestionControl(
			meta1.NewCubicSender(
				initialPacketSize,
				false,
			),
		)
	case "new_reno":
		quicConn.SetCongestionControl(
			meta1.NewCubicSender(
				initialPacketSize,
				true,
			),
		)
	case "bbr_meta_v1":
		quicConn.SetCongestionControl(
			meta1.NewBBRSender(
				initialPacketSize,
				congestion.ByteCount(cwnd)*meta1.InitialMaxDatagramSize,
				meta1.DefaultBBRMaxCongestionWindow*meta1.InitialMaxDatagramSize,
			),
		)
	case "bbr_meta_v2", "bbr":
		quicConn.SetCongestionControl(
			meta2.NewBbrSender(
				initialPacketSize,
				congestion.ByteCount(cwnd),
				meta2.Profile(profile),
			),
		)
	}
}

// SetBrutalCongestionController installs the fixed-rate brutal congestion
// controller. It is a no-op when bps is zero.
func SetBrutalCongestionController(quicConn *quic.Conn, bps uint64) {
	if bps == 0 {
		return
	}
	quicConn.SetCongestionControl(
		meta2.NewBrutalSender(bps),
	)
}
