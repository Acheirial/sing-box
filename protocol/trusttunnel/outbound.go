package trusttunnel

import (
	"context"
	"net"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/adapter/outbound"
	"github.com/sagernet/sing-box/common/dialer"
	"github.com/sagernet/sing-box/common/tls"
	"github.com/sagernet/sing-box/common/trusttunnel"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common"
	"github.com/sagernet/sing/common/bufio"
	E "github.com/sagernet/sing/common/exceptions"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
)

func RegisterOutbound(registry *outbound.Registry) {
	outbound.Register[option.TrustTunnelOutboundOptions](registry, C.TypeTrustTunnel, NewOutbound)
}

var (
	_ adapter.InterfaceUpdateListener = (*Outbound)(nil)
	_ adapter.OutboundWithMultiplex   = (*Outbound)(nil)
)

type Outbound struct {
	outbound.Adapter
	logger     log.ContextLogger
	udpEnabled bool
	poolClient *trusttunnel.PoolClient
}

func NewOutbound(ctx context.Context, router adapter.Router, logger log.ContextLogger, tag string, options option.TrustTunnelOutboundOptions) (adapter.Outbound, error) {
	if options.TLS == nil || !options.TLS.Enabled {
		return nil, C.ErrTLSRequired
	}
	if options.TLS.ECH != nil && options.TLS.ECH.Enabled && options.TLS.UTLS != nil && options.TLS.UTLS.Enabled {
		return nil, E.New("trust-tunnel: ech is incompatible with utls engine")
	}
	tlsConfig, err := tls.NewClient(ctx, logger, options.Server, common.PtrValueOrDefault(options.TLS))
	if err != nil {
		return nil, err
	}

	outboundDialer, err := dialer.NewWithOptions(dialer.Options{
		Context:        ctx,
		Options:        options.DialerOptions,
		RemoteIsDomain: options.ServerIsDomain(),
	})
	if err != nil {
		return nil, err
	}

	server := options.ServerOptions.Build()
	if !server.IsValid() {
		return nil, E.New("missing server address")
	}

	client, err := trusttunnel.NewPoolClient(ctx, trusttunnel.ClientOptions{
		Dialer:                outboundDialer,
		Server:                server.String(),
		Username:              options.Username,
		Password:              options.Password,
		TLSConfig:             tlsConfig,
		QUIC:                  options.Quic,
		QUICCongestionControl: options.CongestionController,
		QUICCwnd:              options.CWND,
		QUICBBRProfile:        options.BBRProfile,
		HealthCheck:           options.HealthCheck,
		MaxConnections:        options.MaxConnections,
		MinStreams:            options.MinStreams,
		MaxStreams:            options.MaxStreams,
	})
	if err != nil {
		return nil, err
	}

	networkList := []string{N.NetworkTCP}
	if options.UDP {
		networkList = append(networkList, N.NetworkUDP)
	}
	return &Outbound{
		Adapter:    outbound.NewAdapterWithDialerOptions(C.TypeTrustTunnel, tag, networkList, options.DialerOptions),
		logger:     logger,
		udpEnabled: options.UDP,
		poolClient: client,
	}, nil
}

func (h *Outbound) DialContext(ctx context.Context, network string, destination M.Socksaddr) (net.Conn, error) {
	ctx, metadata := adapter.ExtendContext(ctx)
	metadata.Outbound = h.Tag()
	metadata.Destination = destination
	switch N.NetworkName(network) {
	case N.NetworkTCP:
		h.logger.InfoContext(ctx, "outbound connection to ", destination)
		return h.poolClient.Dial(ctx, destination.String())
	case N.NetworkUDP:
		conn, err := h.ListenPacket(ctx, destination)
		if err != nil {
			return nil, err
		}
		return bufio.NewBindPacketConn(conn, destination), nil
	default:
		return nil, E.New("unsupported network: ", network)
	}
}

func (h *Outbound) ListenPacket(ctx context.Context, destination M.Socksaddr) (net.PacketConn, error) {
	ctx, metadata := adapter.ExtendContext(ctx)
	metadata.Outbound = h.Tag()
	metadata.Destination = destination
	h.logger.InfoContext(ctx, "outbound packet connection to ", destination)
	return h.poolClient.ListenPacket(ctx)
}

func (h *Outbound) InterfaceUpdated(ctx context.Context) {
	h.poolClient.ResetConnections()
}

func (h *Outbound) MultiplexEnabled() bool {
	return true
}

func (h *Outbound) Close() error {
	return common.Close(h.poolClient)
}
