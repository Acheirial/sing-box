//go:build with_easytier

package easytier

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"sync"

	corehost "github.com/easytier/easytier/easytier-go"

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
	"github.com/sagernet/sing/service/filemanager"
)

func RegisterOutbound(registry *outbound.Registry) {
	outbound.Register[option.EasyTierOutboundOptions](registry, C.TypeEasyTier, NewOutbound)
}

const (
	easyTierDefaultStateDirectory = "easytier"
	easyTierInstanceIDFile        = "instance_id"
)

var errEasyTierClosed = errors.New("easytier outbound closed")

type Outbound struct {
	outbound.Adapter
	logger     log.ContextLogger
	options    option.EasyTierOutboundOptions
	configTOML string
	stateDir   string
	zone       string
	dialer     N.Dialer
	dnsRouter  adapter.DNSRouter
	ctx        context.Context
	cancel     context.CancelFunc
	startOnce  sync.Once
	startErr   error
	mu         sync.Mutex
	host       *corehost.Host
	instance   *corehost.Instance
	instanceID string
}

func NewOutbound(ctx context.Context, router adapter.Router, logger log.ContextLogger, tag string, options option.EasyTierOutboundOptions) (adapter.Outbound, error) {
	if strings.TrimSpace(options.NetworkName) == "" {
		return nil, E.New("easytier: missing network name")
	}
	configTOML, err := renderConfigTOML(options)
	if err != nil {
		return nil, E.Cause(err, "render EasyTier configuration")
	}
	stateDirectory := options.StateDirectory
	if stateDirectory == "" {
		stateDirectory = filepath.Join(easyTierDefaultStateDirectory, tag)
	}
	stateDirectory = filemanager.BasePath(ctx, os.ExpandEnv(stateDirectory))
	stateDirectory, _ = filepath.Abs(stateDirectory)
	outboundDialer, err := dialer.NewWithOptions(dialer.Options{
		Context:        ctx,
		Options:        options.DialerOptions,
		RemoteIsDomain: true,
	})
	if err != nil {
		return nil, err
	}
	outboundCtx, cancel := context.WithCancel(context.Background())
	return &Outbound{
		Adapter:    outbound.NewAdapterWithDialerOptions(C.TypeEasyTier, tag, networkList(options), options.DialerOptions),
		logger:     logger,
		options:    options,
		configTOML: configTOML,
		stateDir:   stateDirectory,
		zone:       normalizeZone(options.TLDDNSZone),
		dialer:     outboundDialer,
		dnsRouter:  service.FromContext[adapter.DNSRouter](ctx),
		ctx:        outboundCtx,
		cancel:     cancel,
	}, nil
}

func networkList(options option.EasyTierOutboundOptions) []string {
	if options.UDP {
		return []string{N.NetworkTCP, N.NetworkUDP}
	}
	return []string{N.NetworkTCP}
}

func (h *Outbound) start() error {
	h.startOnce.Do(func() {
		if err := h.init(); err != nil {
			h.startErr = err
			_ = h.shutdown()
		}
	})
	return h.startErr
}

func (h *Outbound) ensureStarted(ctx context.Context) error {
	done := make(chan error, 1)
	go func() {
		done <- h.start()
	}()
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (h *Outbound) init() error {
	if err := os.MkdirAll(h.stateDir, 0o755); err != nil {
		return fmt.Errorf("easytier: create state directory: %w", err)
	}
	instanceID := loadInstanceID(h.stateDir)
	instanceName := h.options.InstanceName
	if instanceName == "" {
		instanceName = h.Tag()
	}
	host, err := corehost.New(h.ctx, corehost.Options{
		Platform: services(h.dialer, h.lookupControlPlane),
	})
	if err != nil {
		return err
	}
	h.mu.Lock()
	h.host = host
	h.mu.Unlock()
	instance, err := host.CreateInstanceTOML(h.ctx, instanceName, instanceID, h.configTOML)
	if err != nil {
		return err
	}
	h.mu.Lock()
	h.instance = instance
	h.instanceID = instance.ID()
	h.mu.Unlock()
	if err := writeInstanceID(h.stateDir, instance.ID()); err != nil {
		return err
	}
	if err := instance.Start(h.ctx); err != nil {
		return err
	}
	h.logger.Info("EasyTier instance ", instance.ID(), " running")
	return nil
}

func (h *Outbound) currentInstance() (*corehost.Instance, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.instance == nil {
		return nil, errors.New("easytier instance is not ready")
	}
	return h.instance, nil
}

func (h *Outbound) overlayNodes(ctx context.Context) ([]Node, error) {
	instance, err := h.currentInstance()
	if err != nil {
		return nil, err
	}
	var nodes []Node
	info, err := instance.ShowNodeInfo(ctx)
	if err == nil && info != nil {
		node := Node{Hostname: info.GetHostname()}
		if ip, parseErr := parseNodeIPv4(info.GetIpv4Addr()); parseErr == nil {
			node.IPv4 = ip
		}
		if node.Hostname != "" || node.IPv4.IsValid() {
			nodes = append(nodes, node)
		}
	}
	routes, err := instance.ListRoute(ctx)
	if err != nil {
		if len(nodes) == 0 {
			return nil, err
		}
		return nodes, nil
	}
	for _, route := range routes {
		if route == nil {
			continue
		}
		inet := route.GetIpv4Addr()
		if inet == nil || inet.GetAddress() == nil {
			continue
		}
		nodes = append(nodes, Node{
			Hostname: route.GetHostname(),
			IPv4:     ipv4FromUint32(inet.GetAddress().Addr),
		})
	}
	return nodes, nil
}

func (h *Outbound) resolveIPv4(ctx context.Context, host string) (netip.Addr, error) {
	if ip, err := netip.ParseAddr(host); err == nil {
		ip = ip.Unmap()
		if !ip.Is4() {
			return netip.Addr{}, E.New("easytier: overlay dial supports IPv4 only")
		}
		return ip, nil
	}
	nodes, err := h.overlayNodes(ctx)
	if err != nil {
		return netip.Addr{}, err
	}
	if ip, ok := lookupOverlayHost(host, h.zone, nodes); ok {
		return ip, nil
	}
	if isMagicDNS(host, h.zone) {
		return netip.Addr{}, E.New("easytier: overlay hostname ", host, " was not found")
	}
	addresses, err := h.lookupControlPlane(ctx, host, true, false)
	if err != nil {
		return netip.Addr{}, err
	}
	if len(addresses) == 0 {
		return netip.Addr{}, E.New("easytier: resolve ", host, ": no IPv4 address")
	}
	return addresses[0], nil
}

// lookupControlPlane resolves EasyTier control-plane names through the router
// DNS, falling back to the system resolver.
func (h *Outbound) lookupControlPlane(ctx context.Context, host string, ipv4, ipv6 bool) ([]netip.Addr, error) {
	if h.dnsRouter != nil {
		queryOptions := adapter.DNSQueryOptions{}
		switch {
		case ipv4 && !ipv6:
			queryOptions.Strategy = C.DomainStrategyIPv4Only
		case !ipv4 && ipv6:
			queryOptions.Strategy = C.DomainStrategyIPv6Only
		}
		return h.dnsRouter.Lookup(ctx, host, queryOptions)
	}
	var network string
	switch {
	case ipv4 && !ipv6:
		network = "ip4"
	case !ipv4 && ipv6:
		network = "ip6"
	default:
		network = "ip"
	}
	addresses, err := net.DefaultResolver.LookupNetIP(ctx, network, host)
	if err != nil {
		return nil, err
	}
	result := make([]netip.Addr, 0, len(addresses))
	for _, address := range addresses {
		result = append(result, address.Unmap())
	}
	return result, nil
}

func (h *Outbound) DialContext(ctx context.Context, network string, destination M.Socksaddr) (net.Conn, error) {
	switch N.NetworkName(network) {
	case N.NetworkTCP:
	default:
		return nil, os.ErrInvalid
	}
	if err := h.ensureStarted(ctx); err != nil {
		return nil, err
	}
	host := destination.AddrString()
	if destination.IsIP() {
		host = destination.Addr.String()
	}
	ip, err := h.resolveIPv4(ctx, host)
	if err != nil {
		return nil, err
	}
	instance, err := h.currentInstance()
	if err != nil {
		return nil, err
	}
	conn, err := instance.Dial(ctx, "tcp4", net.JoinHostPort(ip.String(), fmt.Sprintf("%d", destination.Port)))
	if err != nil {
		return nil, err
	}
	if conn == nil {
		return nil, errors.New("easytier: conn is nil")
	}
	return conn, nil
}

func (h *Outbound) ListenPacket(ctx context.Context, destination M.Socksaddr) (net.PacketConn, error) {
	if err := h.ensureStarted(ctx); err != nil {
		return nil, err
	}
	var remoteIP netip.Addr
	if destination.IsIP() {
		remoteIP = destination.Addr.Unmap()
	} else {
		ip, err := h.resolveIPv4(ctx, destination.Fqdn)
		if err != nil {
			return nil, fmt.Errorf("can't resolve ip: %w", err)
		}
		remoteIP = ip
	}
	if !remoteIP.Is4() {
		return nil, E.New("easytier: overlay dial supports IPv4 only")
	}
	instance, err := h.currentInstance()
	if err != nil {
		return nil, err
	}
	pc, err := instance.ListenPacket("udp4", ":0")
	if err != nil {
		return nil, err
	}
	if pc == nil {
		return nil, errors.New("easytier: packetConn is nil")
	}
	return bufio.NewBindPacketConn(pc, M.SocksaddrFrom(remoteIP, destination.Port)), nil
}

func (h *Outbound) Close() error {
	h.cancel()
	h.startOnce.Do(func() {
		h.startErr = errEasyTierClosed
	})
	return h.shutdown()
}

func (h *Outbound) shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), C.TCPTimeout)
	defer cancel()
	h.mu.Lock()
	instance := h.instance
	host := h.host
	h.instance = nil
	h.host = nil
	h.mu.Unlock()
	var err error
	if instance != nil {
		err = instance.Close(ctx)
	}
	if host != nil {
		if hostErr := host.Close(ctx); err == nil {
			err = hostErr
		}
	}
	return err
}

func loadInstanceID(stateDir string) string {
	contents, err := os.ReadFile(filepath.Join(stateDir, easyTierInstanceIDFile))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(contents))
}

func writeInstanceID(stateDir, id string) error {
	if id == "" {
		return nil
	}
	path := filepath.Join(stateDir, easyTierInstanceIDFile)
	return os.WriteFile(path, []byte(id+"\n"), 0o600)
}

// RenderConfigTOML converts structured options to EasyTier TOML.
func renderConfigTOML(options option.EasyTierOutboundOptions) (string, error) {
	if err := validateConfig(options); err != nil {
		return "", err
	}
	var encoded strings.Builder
	if options.InstanceName != "" {
		writeTOMLStringField(&encoded, "instance_name", options.InstanceName)
	}
	if options.Hostname != "" {
		writeTOMLStringField(&encoded, "hostname", options.Hostname)
	}
	if strings.TrimSpace(options.IPv4) != "" {
		writeTOMLStringField(&encoded, "ipv4", options.IPv4)
	}
	if options.DHCP || strings.TrimSpace(options.IPv4) == "" {
		writeTOMLBoolField(&encoded, "dhcp", true)
	}
	writeTOMLStringArrayField(&encoded, "listeners", listeners(options.NoListener, options.Listeners))
	if len(options.MappedListeners) > 0 {
		writeTOMLStringArrayField(&encoded, "mapped_listeners", options.MappedListeners)
	}
	if len(options.ExitNodes) > 0 {
		writeTOMLStringArrayField(&encoded, "exit_nodes", options.ExitNodes)
	}
	encoded.WriteByte('\n')
	encoded.WriteString("[network_identity]\n")
	writeTOMLStringField(&encoded, "network_name", options.NetworkName)
	writeTOMLStringField(&encoded, "network_secret", options.NetworkSecret)

	peers, err := parsedPeers(options.Peers)
	if err != nil {
		return "", err
	}

	if secureModeEnabled(options.SecureMode, options.LocalPrivateKey, options.LocalPublicKey, options.Peers) {
		encoded.WriteString("\n[secure_mode]\n")
		writeTOMLBoolField(&encoded, "enabled", true)
		if options.LocalPrivateKey != "" {
			writeTOMLStringField(&encoded, "local_private_key", options.LocalPrivateKey)
		}
		if options.LocalPublicKey != "" {
			writeTOMLStringField(&encoded, "local_public_key", options.LocalPublicKey)
		}
	}

	for _, peer := range peers {
		encoded.WriteString("\n[[peer]]\n")
		writeTOMLStringField(&encoded, "uri", peer.URI)
		if peer.PeerPublicKey != "" {
			writeTOMLStringField(&encoded, "peer_public_key", peer.PeerPublicKey)
		}
	}
	for _, network := range options.ProxyNetworks {
		encoded.WriteString("\n[[proxy_network]]\n")
		writeTOMLStringField(&encoded, "cidr", network)
	}

	encoded.WriteString("\n[flags]\n")
	writeTOMLBoolField(&encoded, "no_tun", true)
	writeTOMLBoolField(&encoded, "bind_device", false)
	writeOptionalBoolField(&encoded, "accept_dns", options.AcceptDNS)
	writeOptionalBoolField(&encoded, "enable_exit_node", options.EnableExitNode)
	writeOptionalBoolField(&encoded, "enable_encryption", options.EnableEncryption)
	if options.EncryptionAlgorithm != "" {
		writeTOMLStringField(&encoded, "encryption_algorithm", options.EncryptionAlgorithm)
	}
	writeOptionalBoolField(&encoded, "private_mode", options.PrivateMode)
	writeOptionalBoolField(&encoded, "latency_first", options.LatencyFirst)
	writeOptionalBoolField(&encoded, "disable_p2p", options.DisableP2P)
	writeOptionalBoolField(&encoded, "enable_kcp_proxy", options.EnableKCPProxy)
	writeOptionalBoolField(&encoded, "disable_kcp_input", options.DisableKCPInput)
	writeOptionalBoolField(&encoded, "enable_quic_proxy", options.EnableQUICProxy)
	writeOptionalBoolField(&encoded, "disable_quic_input", options.DisableQUICInput)
	if options.MTU > 0 {
		fmt.Fprintf(&encoded, "mtu = %d\n", options.MTU)
	}
	if options.TLDDNSZone != "" {
		writeTOMLStringField(&encoded, "tld_dns_zone", options.TLDDNSZone)
	}
	return encoded.String(), nil
}

func validateConfig(options option.EasyTierOutboundOptions) error {
	if strings.TrimSpace(options.NetworkName) == "" {
		return errors.New("easytier: network name is required")
	}
	if options.NoListener != nil && *options.NoListener && len(options.Listeners) > 0 {
		return errors.New("easytier: no-listener cannot be combined with listeners")
	}
	if len(options.Peers) == 0 && len(listeners(options.NoListener, options.Listeners)) == 0 {
		return errors.New("easytier: peers is required when listeners are empty; implicit public.easytier.top is disabled")
	}
	if _, err := parsedPeers(options.Peers); err != nil {
		return err
	}
	if options.LocalPublicKey != "" && options.LocalPrivateKey == "" {
		return errors.New("easytier: local-public-key requires local-private-key")
	}
	if options.SecureMode != nil && !*options.SecureMode && hasSecureModeMaterial(options.LocalPrivateKey, options.LocalPublicKey, options.Peers) {
		return errors.New("easytier: local keys and peer-public-key require secure-mode")
	}
	return nil
}
