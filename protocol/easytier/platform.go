//go:build with_easytier

package easytier

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"strings"

	coreplatform "github.com/easytier/easytier/easytier-go/platform"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
)

// services wraps a sing-box dialer as EasyTier platform capabilities.
func services(dialer N.Dialer, dnsRouter dnsRouterFunc) coreplatform.Services {
	return coreplatform.Services{
		Sockets:     socketFactory{dialer: dialer},
		DNS:         platformDNSResolver{dnsRouter: dnsRouter},
		Environment: connectorEnvironment{dialer: dialer},
	}
}

type dnsRouterFunc func(ctx context.Context, host string, ipv4, ipv6 bool) ([]netip.Addr, error)

// socketFactory creates sockets through the sing-box dialer.
//
// EasyTier BindDevice/SocketMark/reuse options are ignored so hole punching
// can bind local UDP ports. Interface, routing mark, and dialer detour stay on
// the sing-box dialer. FakeTCP is not supported.
type socketFactory struct {
	dialer N.Dialer
}

func (s socketFactory) ConnectTCP(ctx context.Context, options coreplatform.TCPConnectOptions) (net.Conn, error) {
	if options.Purpose == coreplatform.TCPConnectFake {
		return nil, fmt.Errorf("easytier: FakeTCP is not supported")
	}
	if options.RemoteAddr == nil {
		return nil, fmt.Errorf("easytier: TCP connect is missing a remote address")
	}
	network := "tcp"
	if options.Bind.Context.IPVersion == coreplatform.IPVersionV4 || options.Bind.LocalAddr != nil && options.Bind.LocalAddr.IP.To4() != nil {
		network = "tcp4"
	} else if options.Bind.OnlyV6 || options.Bind.Context.IPVersion == coreplatform.IPVersionV6 {
		network = "tcp6"
	}
	return s.dialer.DialContext(ctx, network, M.SocksaddrFromNet(options.RemoteAddr))
}

func (s socketFactory) BindUDP(ctx context.Context, options coreplatform.UDPBindOptions) (net.PacketConn, error) {
	destination := M.Socksaddr{}
	if options.LocalAddr != nil {
		destination = M.SocksaddrFromNet(options.LocalAddr)
	}
	return s.dialer.ListenPacket(ctx, destination)
}

func (s socketFactory) ListenTCP(ctx context.Context, options coreplatform.TCPListenOptions) (net.Listener, error) {
	if options.Bind.Context.NetNS != nil {
		return nil, fmt.Errorf("easytier: network namespaces are not supported")
	}
	if externalTCPListen(options.Purpose) {
		return nil, fmt.Errorf("easytier: TCP listeners are unavailable through a proxy or custom dialer")
	}
	var lc net.ListenConfig
	address := ":0"
	if options.Bind.LocalAddr != nil {
		address = options.Bind.LocalAddr.String()
	}
	return lc.Listen(ctx, "tcp", address)
}

func externalTCPListen(purpose coreplatform.TCPListenPurpose) bool {
	switch purpose {
	case coreplatform.TCPListenDirect, coreplatform.TCPListenManual:
		return true
	default:
		return false
	}
}

// platformDNSResolver resolves EasyTier control-plane names through the sing-box DNS router.
type platformDNSResolver struct {
	dnsRouter dnsRouterFunc
}

func (r platformDNSResolver) LookupIP(ctx context.Context, query coreplatform.DNSQuery) ([]netip.Addr, error) {
	if address, err := netip.ParseAddr(query.Host); err == nil {
		return []netip.Addr{address.Unmap()}, nil
	}
	if r.dnsRouter == nil {
		addresses, err := net.DefaultResolver.LookupNetIP(ctx, "ip", query.Host)
		if err != nil {
			return nil, err
		}
		result := make([]netip.Addr, 0, len(addresses))
		for _, address := range addresses {
			result = append(result, address.Unmap())
		}
		return result, nil
	}
	var ipv4, ipv6 bool
	switch query.IPVersion {
	case 4:
		ipv4, ipv6 = true, false
	case 6:
		ipv4, ipv6 = false, true
	default:
		ipv4, ipv6 = true, true
	}
	return r.dnsRouter(ctx, query.Host, ipv4, ipv6)
}

func (r platformDNSResolver) LookupTXT(ctx context.Context, query coreplatform.DNSQuery) (string, error) {
	records, err := net.LookupTXT(query.Host)
	if err != nil {
		return "", err
	}
	if len(records) == 0 {
		return "", fmt.Errorf("easytier: DNS TXT query for %q returned no records", query.Host)
	}
	return strings.Join(records, ""), nil
}

func (r platformDNSResolver) LookupSRV(ctx context.Context, query coreplatform.DNSQuery) ([]*net.SRV, error) {
	_, srvs, err := net.DefaultResolver.LookupSRV(ctx, "", "", query.Host)
	return srvs, err
}

// connectorEnvironment reports the local address used toward a remote UDP peer.
type connectorEnvironment struct {
	dialer N.Dialer
}

func (e connectorEnvironment) LocalAddrForRemote(ctx context.Context, remote *net.UDPAddr, _ coreplatform.SocketContext) (net.Addr, error) {
	if remote == nil {
		return nil, fmt.Errorf("easytier: missing remote address")
	}
	network := "udp"
	if remote.IP.To4() != nil {
		network = "udp4"
	} else if remote.IP.To16() != nil {
		network = "udp6"
	}
	conn, err := e.dialer.DialContext(ctx, network, M.SocksaddrFromNet(remote))
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	return conn.LocalAddr(), nil
}
