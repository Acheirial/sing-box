//go:build with_quic

package trusttunnel

import (
	"context"
	"runtime"

	cryptoTLS "crypto/tls"

	"github.com/sagernet/quic-go"
	"github.com/sagernet/quic-go/http3"
	"github.com/sagernet/sing-box/common/quiccongestion"
	"github.com/sagernet/sing-box/common/tls"
	M "github.com/sagernet/sing/common/metadata"
)

const (
	// DefaultQuicStreamReceiveWindow is Chrome's default.
	DefaultQuicStreamReceiveWindow = 131072
	DefaultQuicMaxIdleTimeout      = 2 * (DefaultConnectionTimeout + DefaultHealthCheckTimeout)
)

func (c *Client) quicRoundTripper(tlsConfig tls.Config, congestionControlName string, cwnd int, bbrProfile string) error {
	stdConfig, err := tlsConfig.STDConfig()
	if err != nil {
		return err
	}
	c.roundTripper = &http3.Transport{
		TLSClientConfig: stdConfig,
		QUICConfig: &quic.Config{
			Versions:                   []quic.Version{quic.Version1},
			MaxIdleTimeout:             DefaultQuicMaxIdleTimeout,
			InitialStreamReceiveWindow: DefaultQuicStreamReceiveWindow,
			DisablePathMTUDiscovery:    !(runtime.GOOS == "windows" || runtime.GOOS == "linux" || runtime.GOOS == "android" || runtime.GOOS == "darwin"),
			Allow0RTT:                  false,
		},
		Dial: func(ctx context.Context, addr string, tlsCfg *cryptoTLS.Config, cfg *quic.Config) (*quic.Conn, error) {
			destination := M.ParseSocksaddr(addr)
			udpConn, err := c.dialer.ListenPacket(ctx, destination)
			if err != nil {
				return nil, err
			}
			transport := &quic.Transport{
				Conn: udpConn,
			}
			transport.SetCreatedConn(true) // auto close conn
			transport.SetSingleUse(true)   // auto close transport
			quicConn, err := transport.DialEarly(ctx, destination.UDPAddr(), tlsCfg, cfg)
			if err != nil {
				_ = udpConn.Close()
				return nil, err
			}
			quiccongestion.SetCongestionController(quicConn, congestionControlName, cwnd, bbrProfile)
			return quicConn, nil
		},
	}
	return nil
}
