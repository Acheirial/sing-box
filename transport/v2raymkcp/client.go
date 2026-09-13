package v2raymkcp

import (
	"context"
	"net"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/common/tls"
	"github.com/sagernet/sing-box/option"
	E "github.com/sagernet/sing/common/exceptions"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
)

// V2RayTransportTypeMKCP is the mkcp v2ray transport type.
const V2RayTransportTypeMKCP = "mkcp"

var _ adapter.V2RayClientTransport = (*Client)(nil)

type Client struct {
	dialer     N.Dialer
	serverAddr M.Socksaddr
	config     Config
}

func NewClient(ctx context.Context, dialer N.Dialer, serverAddr M.Socksaddr, options option.V2RayMKCPOptions, tlsConfig tls.Config) (adapter.V2RayClientTransport, error) {
	if tlsConfig != nil {
		return nil, E.New("TLS is not supported by mkcp transport")
	}
	return &Client{
		dialer:     dialer,
		serverAddr: serverAddr,
		config: Config{
			MTU:              options.MTU,
			TTI:              options.TTI,
			UplinkCapacity:   options.UplinkCapacity,
			DownlinkCapacity: options.DownlinkCapacity,
			Congestion:       options.Congestion,
			WriteBuffer:      options.WriteBuffer,
			ReadBuffer:       options.ReadBuffer,
			Seed:             options.Seed,
			Header:           options.Header,
		},
	}, nil
}

func (c *Client) DialContext(ctx context.Context) (net.Conn, error) {
	rawConn, err := c.dialer.DialContext(ctx, N.NetworkUDP, c.serverAddr)
	if err != nil {
		return nil, err
	}
	return Dial(ctx, rawConn, c.config)
}

func (c *Client) Close() error {
	return nil
}
