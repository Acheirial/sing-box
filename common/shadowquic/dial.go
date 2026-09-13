package shadowquic

import (
	"context"
	"crypto/tls"
	"net"
	"net/netip"
	"time"

	"github.com/sagernet/quic-go"
	M "github.com/sagernet/sing/common/metadata"
)

type PacketDialer interface {
	ListenPacket(ctx context.Context, destination M.Socksaddr) (net.PacketConn, error)
}

type DialQuicOption struct {
	Early              bool
	ConnectionIDLength int
}

// DialQuic dials a QUIC connection over UDP through the given packet dialer.
// It returns the packet connection used for the handshake and the QUIC
// connection itself.
func DialQuic(ctx context.Context, destination netip.AddrPort, packetDialer PacketDialer, tlsConf *tls.Config, quicConfig *quic.Config, option DialQuicOption) (net.PacketConn, *quic.Conn, error) {
	packetConn, err := packetDialer.ListenPacket(ctx, M.SocksaddrFromNetIP(destination))
	if err != nil {
		return nil, nil, err
	}
	udpAddr := net.UDPAddrFromAddrPort(destination)
	transport := &quic.Transport{
		Conn:               packetConn,
		ConnectionIDLength: option.ConnectionIDLength,
	}
	transport.SetCreatedConn(true)
	transport.SetSingleUse(true)

	var quicConn *quic.Conn
	if option.Early {
		quicConn, err = transport.DialEarly(ctx, udpAddr, tlsConf, quicConfig)
	} else {
		quicConn, err = transport.Dial(ctx, udpAddr, tlsConf, quicConfig)
	}
	if err != nil {
		_ = packetConn.Close()
		return nil, nil, err
	}
	return packetConn, quicConn, nil
}

type quicNetConn struct {
	*quic.Conn
	pc net.PacketConn
}

func (q quicNetConn) Close() error {
	err := q.Conn.CloseWithError(0, "")
	_ = q.pc.Close()
	return err
}

func (q quicNetConn) Read([]byte) (int, error) {
	panic("should not call Read on quicNetConn")
}

func (q quicNetConn) Write([]byte) (int, error) {
	panic("should not call Write on quicNetConn")
}

func (q quicNetConn) SetDeadline(time.Time) error {
	panic("should not call SetDeadline on quicNetConn")
}

func (q quicNetConn) SetReadDeadline(time.Time) error {
	panic("should not call SetReadDeadline on quicNetConn")
}

func (q quicNetConn) SetWriteDeadline(time.Time) error {
	panic("should not call SetWriteDeadline on quicNetConn")
}

var _ net.Conn = quicNetConn{}
