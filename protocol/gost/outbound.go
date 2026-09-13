package gost

import (
	"context"
	"net"
	"os"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/adapter/outbound"
	"github.com/sagernet/sing-box/common/dialer"
	"github.com/sagernet/sing-box/common/gost"
	"github.com/sagernet/sing-box/common/mux"
	"github.com/sagernet/sing-box/common/tls"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
	"github.com/sagernet/sing/common/uot"
)

func RegisterOutbound(registry *outbound.Registry) {
	outbound.Register[option.GostOutboundOptions](registry, C.TypeGost, NewOutbound)
}

var (
	_ adapter.OutboundWithMultiplex   = (*Outbound)(nil)
	_ adapter.InterfaceUpdateListener = (*Outbound)(nil)
	_ adapter.IdleConnectionKeeper    = (*Outbound)(nil)
)

type Outbound struct {
	outbound.Adapter
	logger          log.ContextLogger
	dialer          N.Dialer
	serverAddr      M.Socksaddr
	muxEnabled      bool
	multiplexDialer *mux.Client
	uotClient       *uot.Client
}

func NewOutbound(ctx context.Context, router adapter.Router, logger log.ContextLogger, tag string, options option.GostOutboundOptions) (adapter.Outbound, error) {
	outboundDialer, err := dialer.NewWithOptions(dialer.Options{
		Context:        ctx,
		Options:        options.DialerOptions,
		RemoteIsDomain: options.ServerIsDomain(),
	})
	if err != nil {
		return nil, err
	}

	networks := []string{N.NetworkTCP}
	if options.UDP {
		networks = append(networks, N.NetworkUDP)
	}

	outbound := &Outbound{
		Adapter:    outbound.NewAdapterWithDialerOptions(C.TypeGost, tag, networks, options.DialerOptions),
		logger:     logger,
		serverAddr: options.ServerOptions.Build(),
		muxEnabled: options.Mux,
	}

	relayOption := &gost.RelayOption{
		Server:   outbound.serverAddr,
		Forward:  options.Forward,
		TLS:      options.TLS != nil && options.TLS.Enabled,
		Username: options.Username,
		Password: options.Password,
	}

	var relayDialer N.Dialer
	if relayOption.TLS {
		tlsConfig, err := tls.NewClient(ctx, logger, options.Server, common.PtrValueOrDefault(options.TLS))
		if err != nil {
			return nil, err
		}
		relayDialer = gost.NewRelayDialerTLS(outboundDialer, relayOption, gost.NewTLSWrapper(tlsConfig))
	} else {
		relayDialer = gost.NewRelayDialer(outboundDialer, relayOption)
	}
	outbound.dialer = relayDialer

	if options.UDP {
		outbound.uotClient = &uot.Client{
			Dialer:  relayDialer,
			Version: uot.Version,
		}
	}

	if options.Mux {
		outbound.multiplexDialer, err = mux.NewClientWithOptions(relayDialer, logger, option.OutboundMultiplexOptions{Enabled: true})
		if err != nil {
			return nil, err
		}
	}

	return outbound, nil
}

func (h *Outbound) Start(stage adapter.StartStage) error {
	return nil
}

func (h *Outbound) MultiplexEnabled() bool {
	return h.muxEnabled
}

func (h *Outbound) InterfaceUpdated(ctx context.Context) {
	if h.multiplexDialer != nil {
		h.multiplexDialer.Reset()
	}
}

func (h *Outbound) SetKeepIdleConnections(keep bool) {
	if h.multiplexDialer != nil {
		h.multiplexDialer.SetKeepIdleConnections(keep)
	}
}

func (h *Outbound) CloseIdleConnections() {
	if h.multiplexDialer != nil {
		h.multiplexDialer.CloseIdleConnections()
	}
}

func (h *Outbound) Close() error {
	return common.Close(common.PtrOrNil(h.multiplexDialer))
}

func (h *Outbound) DialContext(ctx context.Context, network string, destination M.Socksaddr) (net.Conn, error) {
	ctx, metadata := adapter.ExtendContext(ctx)
	metadata.Outbound = h.Tag()
	metadata.Destination = destination
	if h.multiplexDialer != nil {
		switch N.NetworkName(network) {
		case N.NetworkTCP:
			h.logger.InfoContext(ctx, "outbound multiplex connection to ", destination)
			return h.multiplexDialer.DialContext(ctx, network, destination)
		case N.NetworkUDP:
			h.logger.InfoContext(ctx, "outbound multiplex packet connection to ", destination)
			return h.multiplexDialer.DialContext(ctx, network, destination)
		}
		return nil, os.ErrInvalid
	}
	switch N.NetworkName(network) {
	case N.NetworkTCP:
		h.logger.InfoContext(ctx, "outbound connection to ", destination)
		return h.dialer.DialContext(ctx, network, destination)
	case N.NetworkUDP:
		h.logger.InfoContext(ctx, "outbound packet connection to ", destination)
		if h.uotClient != nil {
			return h.uotClient.DialContext(ctx, network, destination)
		}
	}
	return nil, os.ErrInvalid
}

func (h *Outbound) ListenPacket(ctx context.Context, destination M.Socksaddr) (net.PacketConn, error) {
	ctx, metadata := adapter.ExtendContext(ctx)
	metadata.Outbound = h.Tag()
	metadata.Destination = destination
	if h.multiplexDialer != nil {
		h.logger.InfoContext(ctx, "outbound multiplex packet connection to ", destination)
		return h.multiplexDialer.ListenPacket(ctx, destination)
	}
	if !h.udpEnabled() {
		return nil, os.ErrInvalid
	}
	if relayDialer, isRelayDialer := h.dialer.(*gost.RelayDialer); isRelayDialer && destination.IsValid() {
		h.logger.InfoContext(ctx, "outbound packet connection to ", destination)
		return relayDialer.ListenPacket(ctx, destination)
	}
	return nil, os.ErrInvalid
}

func (h *Outbound) udpEnabled() bool {
	return h.uotClient != nil
}
