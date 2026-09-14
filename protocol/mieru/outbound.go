package mieru

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/adapter/outbound"
	"github.com/sagernet/sing-box/common/dialer"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/bufio"
	E "github.com/sagernet/sing/common/exceptions"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
	"github.com/sagernet/sing/service"

	mieruclient "github.com/enfein/mieru/v3/apis/client"
	mierucommon "github.com/enfein/mieru/v3/apis/common"
	mierumodel "github.com/enfein/mieru/v3/apis/model"
	mierutp "github.com/enfein/mieru/v3/apis/trafficpattern"
	mierupb "github.com/enfein/mieru/v3/pkg/appctl/appctlpb"
	"google.golang.org/protobuf/proto"
)

func RegisterOutbound(registry *outbound.Registry) {
	outbound.Register[option.MieruOutboundOptions](registry, C.TypeMieru, NewOutbound)
}

var _ adapter.Outbound = (*Outbound)(nil)

type Outbound struct {
	outbound.Adapter
	dnsRouter adapter.DNSRouter
	logger    log.ContextLogger
	dialer    N.Dialer
	client    mieruclient.Client
	access    sync.Mutex
}

func NewOutbound(ctx context.Context, router adapter.Router, logger log.ContextLogger, tag string, options option.MieruOutboundOptions) (adapter.Outbound, error) {
	err := validateMieruOption(options)
	if err != nil {
		return nil, E.Cause(err, "mieru")
	}
	config, err := buildMieruClientConfig(tag, options)
	if err != nil {
		return nil, E.Cause(err, "build mieru client config")
	}
	client := mieruclient.NewClient()
	err = client.Store(config)
	if err != nil {
		return nil, E.Cause(err, "store mieru client config")
	}
	// The client is started lazily on the first use.
	outboundDialer, err := dialer.New(ctx, options.DialerOptions, options.ServerIsDomain())
	if err != nil {
		return nil, err
	}
	networkList := []string{N.NetworkTCP}
	if options.UDP {
		networkList = append(networkList, N.NetworkUDP)
	}
	return &Outbound{
		Adapter:   outbound.NewAdapterWithDialerOptions(C.TypeMieru, tag, networkList, options.DialerOptions),
		dnsRouter: service.FromContext[adapter.DNSRouter](ctx),
		logger:    logger,
		dialer:    outboundDialer,
		client:    client,
	}, nil
}

func (h *Outbound) DialContext(ctx context.Context, network string, destination M.Socksaddr) (net.Conn, error) {
	ctx, metadata := adapter.ExtendContext(ctx)
	metadata.Outbound = h.Tag()
	metadata.Destination = destination
	err := h.ensureClientIsRunning(ctx)
	if err != nil {
		return nil, err
	}
	addr := socksaddrToMieruNetAddrSpec(N.NetworkName(network), destination)
	h.logger.InfoContext(ctx, "outbound connection to ", destination)
	conn, err := h.client.DialContext(ctx, addr)
	if err != nil {
		return nil, E.Cause(err, "dial to ", destination)
	}
	return conn, nil
}

func (h *Outbound) ListenPacket(ctx context.Context, destination M.Socksaddr) (net.PacketConn, error) {
	ctx, metadata := adapter.ExtendContext(ctx)
	metadata.Outbound = h.Tag()
	metadata.Destination = destination
	err := h.ensureClientIsRunning(ctx)
	if err != nil {
		return nil, err
	}
	addr := socksaddrToMieruNetAddrSpec(N.NetworkUDP, destination)
	h.logger.InfoContext(ctx, "outbound packet connection to ", destination)
	conn, err := h.client.DialContext(ctx, addr)
	if err != nil {
		return nil, E.Cause(err, "dial to ", destination)
	}
	packetConn := mierucommon.NewUDPAssociateWrapper(mierucommon.NewPacketOverStreamTunnel(conn))
	return bufio.NewPacketConn(packetConn), nil
}

type mieruPacketDialer struct {
	dialer N.Dialer
}

var _ mierucommon.PacketDialer = (*mieruPacketDialer)(nil)

func (d mieruPacketDialer) ListenPacket(ctx context.Context, network, laddr, raddr string) (net.PacketConn, error) {
	rDestination := M.ParseSocksaddr(raddr)
	return d.dialer.ListenPacket(ctx, rDestination)
}

type mieruDNSResolver struct {
	dnsRouter adapter.DNSRouter
}

var _ mierucommon.DNSResolver = (*mieruDNSResolver)(nil)

func (r mieruDNSResolver) LookupIP(ctx context.Context, network, host string) ([]net.IP, error) {
	dnsOptions := adapter.DNSQueryOptions{}
	switch network {
	case "ip4":
		dnsOptions.Strategy = C.DomainStrategyIPv4Only
	case "ip6":
		dnsOptions.Strategy = C.DomainStrategyIPv6Only
	}
	addresses, err := r.dnsRouter.Lookup(ctx, host, dnsOptions)
	if err != nil {
		return nil, E.Cause(err, "look up ip of ", host)
	}
	ips := make([]net.IP, 0, len(addresses))
	for _, address := range addresses {
		ips = append(ips, net.IP(address.AsSlice()))
	}
	if len(ips) == 0 {
		return nil, E.New("no ip address found for ", host)
	}
	return ips, nil
}

func (h *Outbound) ensureClientIsRunning(ctx context.Context) error {
	h.access.Lock()
	defer h.access.Unlock()

	if h.client.IsRunning() {
		return nil
	}

	// Set the dialer, packet dialer, and resolver before starting the client.
	config, err := h.client.Load()
	if err != nil {
		return err
	}
	config.Dialer = &mieruDialer{dialer: h.dialer, dnsRouter: h.dnsRouter}
	config.PacketDialer = mieruPacketDialer{dialer: h.dialer}
	config.Resolver = mieruDNSResolver{dnsRouter: h.dnsRouter}
	err = h.client.Store(config)
	if err != nil {
		return err
	}

	err = h.client.Start()
	if err != nil {
		return E.Cause(err, "start mieru client")
	}
	return nil
}

func (h *Outbound) Close() error {
	h.access.Lock()
	defer h.access.Unlock()
	if h.client != nil && h.client.IsRunning() {
		return h.client.Stop()
	}
	return nil
}

type mieruDialer struct {
	dialer    N.Dialer
	dnsRouter adapter.DNSRouter
}

var _ mierucommon.Dialer = (*mieruDialer)(nil)

func (d *mieruDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	destination := M.ParseSocksaddr(address)
	if destination.IsDomain() && d.dnsRouter != nil {
		destinationAddresses, err := d.dnsRouter.Lookup(ctx, destination.Fqdn, adapter.DNSQueryOptions{})
		if err != nil {
			return nil, E.Cause(err, "look up ip of ", destination.Fqdn)
		}
		if len(destinationAddresses) > 0 {
			destination.Addr = destinationAddresses[0]
			destination.Fqdn = ""
		}
	}
	return d.dialer.DialContext(ctx, network, destination)
}

func socksaddrToMieruNetAddrSpec(network string, destination M.Socksaddr) mierumodel.NetAddrSpec {
	spec := mierumodel.NetAddrSpec{
		Net: network,
	}
	if destination.IsDomain() {
		spec.AddrSpec = mierumodel.AddrSpec{
			FQDN: destination.Fqdn,
			Port: int(destination.Port),
		}
	} else {
		spec.AddrSpec = mierumodel.AddrSpec{
			IP:   net.IP(destination.Addr.AsSlice()),
			Port: int(destination.Port),
		}
	}
	return spec
}

func buildMieruClientConfig(profileName string, options option.MieruOutboundOptions) (*mieruclient.ClientConfig, error) {
	transportProtocol := mierupb.TransportProtocol_UNKNOWN_TRANSPORT_PROTOCOL.Enum()
	switch options.Transport {
	case "tcp":
		transportProtocol = mierupb.TransportProtocol_TCP.Enum()
	case "udp":
		transportProtocol = mierupb.TransportProtocol_UDP.Enum()
	}
	var server *mierupb.ServerEndpoint
	if net.ParseIP(options.Server) != nil {
		server = buildMieruServerEndpoint(&mierupb.ServerEndpoint{
			IpAddress: proto.String(options.Server),
		}, options, transportProtocol)
	} else {
		server = buildMieruServerEndpoint(&mierupb.ServerEndpoint{
			DomainName: proto.String(options.Server),
		}, options, transportProtocol)
	}
	config := &mieruclient.ClientConfig{
		Profile: &mierupb.ClientProfile{
			ProfileName: proto.String(profileName),
			User: &mierupb.User{
				Name:     proto.String(options.Username),
				Password: proto.String(options.Password),
			},
			Servers: []*mierupb.ServerEndpoint{server},
		},
		DNSConfig: &mierucommon.ClientDNSConfig{
			BypassDialerDNS: true,
		},
	}
	if multiplexing, ok := mierupb.MultiplexingLevel_value[options.Multiplexing]; ok {
		config.Profile.Multiplexing = &mierupb.MultiplexingConfig{
			Level: mierupb.MultiplexingLevel(multiplexing).Enum(),
		}
	}
	if handshakeMode, ok := mierupb.HandshakeMode_value[options.HandshakeMode]; ok {
		config.Profile.HandshakeMode = (*mierupb.HandshakeMode)(&handshakeMode)
	}
	if options.TrafficPattern != "" {
		trafficPattern, _ := mierutp.Decode(options.TrafficPattern)
		config.Profile.TrafficPattern = trafficPattern
	}
	return config, nil
}

func buildMieruServerEndpoint(server *mierupb.ServerEndpoint, options option.MieruOutboundOptions, transportProtocol *mierupb.TransportProtocol) *mierupb.ServerEndpoint {
	if options.PortRange != "" {
		server.PortBindings = []*mierupb.PortBinding{
			{
				PortRange: proto.String(options.PortRange),
				Protocol:  transportProtocol,
			},
		}
	} else {
		server.PortBindings = []*mierupb.PortBinding{
			{
				Port:     proto.Int32(int32(options.ServerPort)),
				Protocol: transportProtocol,
			},
		}
	}
	return server
}

func validateMieruOption(options option.MieruOutboundOptions) error {
	if options.Server == "" {
		return E.New("server is empty")
	}
	if options.ServerPort == 0 && options.PortRange == "" {
		return E.New("either server_port or port_range must be set")
	}
	if options.ServerPort != 0 && options.PortRange != "" {
		return E.New("server_port and port_range cannot be set at the same time")
	}
	if options.PortRange != "" {
		beginPort, endPort, err := beginAndEndPortFromPortRange(options.PortRange)
		if err != nil {
			return E.New("invalid port_range format")
		}
		if beginPort < 1 || beginPort > 65535 {
			return E.New("begin port must be between 1 and 65535")
		}
		if endPort < 1 || endPort > 65535 {
			return E.New("end port must be between 1 and 65535")
		}
		if beginPort > endPort {
			return E.New("begin port must be less than or equal to end port")
		}
	}
	if options.Transport != "tcp" && options.Transport != "udp" {
		return E.New("transport must be tcp or udp")
	}
	if options.Username == "" {
		return E.New("username is empty")
	}
	if options.Password == "" {
		return E.New("password is empty")
	}
	if options.Multiplexing != "" {
		if _, ok := mierupb.MultiplexingLevel_value[options.Multiplexing]; !ok {
			return E.New("invalid multiplexing level: ", options.Multiplexing)
		}
	}
	if options.HandshakeMode != "" {
		if _, ok := mierupb.HandshakeMode_value[options.HandshakeMode]; !ok {
			return E.New("invalid handshake mode: ", options.HandshakeMode)
		}
	}
	if options.TrafficPattern != "" {
		trafficPattern, err := mierutp.Decode(options.TrafficPattern)
		if err != nil {
			return E.Cause(err, "decode traffic pattern ", options.TrafficPattern)
		}
		err = mierutp.Validate(trafficPattern)
		if err != nil {
			return E.Cause(err, "invalid traffic pattern ", options.TrafficPattern)
		}
	}
	return nil
}

func beginAndEndPortFromPortRange(portRange string) (int, int, error) {
	var beginPort, endPort int
	_, err := fmt.Sscanf(portRange, "%d-%d", &beginPort, &endPort)
	return beginPort, endPort, err
}
