package shadowquic

import (
	"context"
	stdtls "crypto/tls"
	"net"
	"net/netip"
	"strings"
	"time"

	"github.com/sagernet/quic-go"
	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/adapter/outbound"
	"github.com/sagernet/sing-box/common/dialer"
	"github.com/sagernet/sing-box/common/quiccongestion"
	"github.com/sagernet/sing-box/common/shadowquic"
	"github.com/sagernet/sing-box/common/tls"
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
	outbound.Register[option.ShadowQUICOutboundOptions](registry, C.TypeShadowQUIC, NewOutbound)
}

var _ adapter.Outbound = (*Outbound)(nil)

type Outbound struct {
	outbound.Adapter
	logger     log.ContextLogger
	dialer     N.Dialer
	serverAddr M.Socksaddr
	client     *shadowquic.Client
}

func NewOutbound(ctx context.Context, router adapter.Router, logger log.ContextLogger, tag string, options option.ShadowQUICOutboundOptions) (adapter.Outbound, error) {
	if options.Username != "" || options.Password != "" {
		// JLS authentication requires the jls-quic-go QUIC stack, which is not
		// available on sagernet/quic-go; refuse instead of silently connecting
		// unauthenticated.
		return nil, E.New("shadowquic JLS authentication (username/password) is not supported in this build")
	}
	serverAddr := options.ServerOptions.Build()
	if serverAddr.Port == 0 {
		serverAddr.Port = 443
	}
	if options.TLS == nil || !options.TLS.Enabled {
		return nil, C.ErrTLSRequired
	}
	tlsServerAddress := options.Server
	if options.TLS.ServerName != "" {
		tlsServerAddress = options.TLS.ServerName
	}
	tlsConfig, err := tls.NewClient(ctx, logger, tlsServerAddress, common.PtrValueOrDefault(options.TLS))
	if err != nil {
		return nil, err
	}
	tlsConfig.SetNextProtos(options.ALPN)
	if len(tlsConfig.NextProtos()) == 0 {
		tlsConfig.SetNextProtos([]string{"h3"})
	}
	stdTLSConfig, err := tlsConfig.STDConfig()
	if err != nil {
		return nil, E.Cause(err, "shadowquic requires a standard TLS engine")
	}
	if stdTLSConfig.MinVersion == 0 {
		stdTLSConfig.MinVersion = stdtls.VersionTLS13
	}
	if options.ZeroRTT {
		// DialEarly can only send 0-RTT data after TLS has cached a session
		// ticket from an earlier connection to this server.
		stdTLSConfig.ClientSessionCache = stdtls.NewLRUClientSessionCache(1)
	}

	if options.MaxDatagramFrameSize == 0 {
		options.MaxDatagramFrameSize = 1400
	}
	if options.MaxOpenStreams == 0 {
		options.MaxOpenStreams = 1024
	}
	if options.CWND == 0 {
		options.CWND = 32
	}

	quicVersions := shadowquic.DefaultQUICVersions()
	if len(options.QUICVersions) > 0 {
		quicVersions, err = shadowquic.ParseQUICVersions(options.QUICVersions)
		if err != nil {
			return nil, err
		}
	}

	quicConfig := &quic.Config{
		Versions:                       quicVersions,
		InitialStreamReceiveWindow:     uint64(options.ReceiveWindowConn),
		MaxStreamReceiveWindow:         uint64(options.ReceiveWindowConn),
		InitialConnectionReceiveWindow: uint64(options.ReceiveWindow),
		MaxConnectionReceiveWindow:     uint64(options.ReceiveWindow),
		MaxIncomingStreams:             int64(options.MaxOpenStreams),
		MaxIncomingUniStreams:          int64(options.MaxOpenStreams),
		DisablePathMTUDiscovery:        options.DisableMTUDiscovery,
		MaxDatagramFrameSize:           int64(options.MaxDatagramFrameSize),
		EnableDatagrams:                true,
	}
	if options.KeepAliveInterval > 0 {
		quicConfig.KeepAlivePeriod = time.Duration(options.KeepAliveInterval)
	}
	if options.ReceiveWindowConn == 0 {
		quicConfig.InitialStreamReceiveWindow = quiccongestion.DefaultStreamReceiveWindow / 10
		quicConfig.MaxStreamReceiveWindow = quiccongestion.DefaultStreamReceiveWindow
	}
	if options.ReceiveWindow == 0 {
		quicConfig.InitialConnectionReceiveWindow = quiccongestion.DefaultConnectionReceiveWindow / 10
		quicConfig.MaxConnectionReceiveWindow = quiccongestion.DefaultConnectionReceiveWindow
	}

	outboundDialer, err := dialer.NewWithOptions(dialer.Options{
		Context:        ctx,
		Options:        options.DialerOptions,
		RemoteIsDomain: options.ServerIsDomain(),
	})
	if err != nil {
		return nil, err
	}

	client := shadowquic.NewClient(&shadowquic.ClientOption{
		UDPOverStream:        options.UDPOverStream,
		CongestionController: options.CongestionController,
		SendBPS:              parseBps(options.Up),
		ReceiveBPS:           parseBps(options.Down),
		CWND:                 options.CWND,
		BBRProfile:           options.BBRProfile,
		Dial: func(dialCtx context.Context) (*quic.Conn, error) {
			destination, err := udpDestination(serverAddr)
			if err != nil {
				return nil, err
			}
			_, quicConn, err := shadowquic.DialQuic(dialCtx, destination, outboundDialer, stdTLSConfig, quicConfig, shadowquic.DialQuicOption{
				Early: options.ZeroRTT,
			})
			return quicConn, err
		},
	})

	return &Outbound{
		Adapter:    outbound.NewAdapterWithDialerOptions(C.TypeShadowQUIC, tag, []string{N.NetworkTCP, N.NetworkUDP}, options.DialerOptions),
		logger:     logger,
		dialer:     outboundDialer,
		serverAddr: serverAddr,
		client:     client,
	}, nil
}

// udpDestination converts the server address to a UDP address port. The
// dialer already resolved domain servers during construction because
// RemoteIsDomain was set, so the address is valid here.
func udpDestination(serverAddr M.Socksaddr) (netip.AddrPort, error) {
	destination := serverAddr.AddrPort()
	if !destination.IsValid() {
		return netip.AddrPort{}, E.New("invalid server address")
	}
	return destination, nil
}

func (h *Outbound) DialContext(ctx context.Context, network string, destination M.Socksaddr) (net.Conn, error) {
	switch N.NetworkName(network) {
	case N.NetworkTCP:
		h.logger.InfoContext(ctx, "outbound connection to ", destination)
		return h.client.DialContext(ctx, destination)
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
	h.logger.InfoContext(ctx, "outbound packet connection to ", destination)
	return h.client.ListenPacket(ctx)
}

func (h *Outbound) Close() error {
	return h.client.Close()
}

func parseBps(value string) uint64 {
	if value == "" {
		return 0
	}
	// Minimal mihomo compatible parser: "<number>[KMGT][b]ps".
	// Numbers without a unit are treated as Mbps.
	numEnd := len(value)
	for i, c := range value {
		if c < '0' || c > '9' {
			numEnd = i
			break
		}
	}
	if numEnd == 0 {
		return 0
	}
	var num uint64
	for _, c := range value[:numEnd] {
		num = num*10 + uint64(c-'0')
	}
	rest := strings.TrimSpace(value[numEnd:])
	if rest == "" {
		rest = "Mbps"
	}
	var n uint64 = 1000 * 1000 // default Mbps multiplier
	for _, c := range rest {
		switch c {
		case 'T':
			n *= 1000
		case 'G':
			n *= 1000
		case 'M':
		case 'K':
			n /= 1000
		case 'b':
			n /= 8 // bits, convert to bytes
		case 'B', 'p', 's':
		default:
			return 0
		}
	}
	return n * num
}
