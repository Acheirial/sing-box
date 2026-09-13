package v2raymekya

import (
	"context"
	"net"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/common/tls"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing-box/transport/v2raymkcp"
	E "github.com/sagernet/sing/common/exceptions"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
)

// V2RayTransportTypeMEKYA is the mekya v2ray transport type.
const V2RayTransportTypeMEKYA = "mekya"

var _ adapter.V2RayClientTransport = (*Transport)(nil)

type Transport struct {
	client    *Client
	tlsConfig tls.Config
	dialer    N.Dialer
}

func NewClient(ctx context.Context, dialer N.Dialer, serverAddr M.Socksaddr, options option.V2RayMEKYAOptions, tlsConfig tls.Config) (adapter.V2RayClientTransport, error) {
	if tlsConfig == nil {
		return nil, E.New("mekya transport requires TLS")
	}
	url := options.URL
	if url == "" {
		url = "https://" + serverAddr.String()
	}
	if len(tlsConfig.NextProtos()) == 0 {
		tlsConfig.SetNextProtos([]string{"h2", "http/1.1"})
	}
	dial := func(ctx context.Context) (net.Conn, error) {
		conn, err := dialer.DialContext(ctx, N.NetworkTCP, serverAddr)
		if err != nil {
			return nil, err
		}
		tlsConn, err := tls.ClientHandshake(ctx, conn, tlsConfig)
		if err != nil {
			_ = conn.Close()
			return nil, err
		}
		return tlsConn, nil
	}
	client, err := NewHTTPClient(context.Background(), dial, Config{
		KCP: v2raymkcp.Config{
			MTU:              options.KCP.MTU,
			TTI:              options.KCP.TTI,
			UplinkCapacity:   options.KCP.UplinkCapacity,
			DownlinkCapacity: options.KCP.DownlinkCapacity,
			Congestion:       options.KCP.Congestion,
			WriteBuffer:      options.KCP.WriteBuffer,
			ReadBuffer:       options.KCP.ReadBuffer,
			Seed:             options.KCP.Seed,
			Header:           options.KCP.Header,
		},
		URL:                            url,
		H2PoolSize:                     options.H2PoolSize,
		MaxWriteDelay:                  options.MaxWriteDelay,
		MaxRequestSize:                 options.MaxRequestSize,
		PollingIntervalInitial:         options.PollingIntervalInitial,
		MaxWriteSize:                   options.MaxWriteSize,
		MaxWriteDurationMs:             options.MaxWriteDurationMs,
		MaxSimultaneousWriteConnection: options.MaxSimultaneousWriteConnection,
		PacketWritingBuffer:            options.PacketWritingBuffer,
	})
	if err != nil {
		return nil, err
	}
	return &Transport{client: client}, nil
}

func (t *Transport) DialContext(ctx context.Context) (net.Conn, error) {
	return t.client.Dial(ctx)
}

func (t *Transport) Close() error {
	return t.client.Close()
}
