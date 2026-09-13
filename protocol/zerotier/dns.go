package zerotier

import (
	"context"
	"fmt"
	"net/netip"
	"time"

	mDNS "github.com/miekg/dns"
)

const dnsTimeout = 10 * time.Second

// parseDNSServers parses the configured remote DNS server addresses. Entries
// without a port default to 53.
func parseDNSServers(servers []string) ([]netip.AddrPort, error) {
	result := make([]netip.AddrPort, 0, len(servers))
	for _, server := range servers {
		addrPort, err := netip.ParseAddrPort(server)
		if err == nil {
			result = append(result, addrPort)
			continue
		}
		address, addrErr := netip.ParseAddr(server)
		if addrErr != nil {
			return nil, fmt.Errorf("parse ZeroTier DNS server %q: %w", server, addrErr)
		}
		result = append(result, netip.AddrPortFrom(address, 53))
	}
	return result, nil
}

// remoteLookup resolves a domain through DNS servers reachable over the
// ZeroTier virtual network, preferring the configured servers and falling back
// to the controller-provided ones.
func (z *Outbound) remoteLookup(ctx context.Context, domain string) ([]netip.Addr, error) {
	z.stateMu.RLock()
	device := z.tunDevice
	servers := z.configuredDNSServers
	if len(servers) == 0 {
		servers = z.remoteDNSServers
	}
	z.stateMu.RUnlock()
	if device == nil {
		return nil, fmt.Errorf("ZeroTier stack is not ready")
	}
	if len(servers) == 0 {
		return nil, fmt.Errorf("no ZeroTier remote DNS servers configured")
	}
	var lookupErr error
	for _, server := range servers {
		addresses, err := lookupDNS(ctx, device, server, domain)
		if err == nil && len(addresses) > 0 {
			return addresses, nil
		}
		if err != nil {
			lookupErr = err
		}
	}
	if lookupErr == nil {
		lookupErr = fmt.Errorf("no addresses returned for %s", domain)
	}
	return nil, lookupErr
}

// lookupDNS exchanges one A/AAAA pair of queries with the server over the
// virtual network through the gVisor netstack UDP socket.
func lookupDNS(ctx context.Context, device ipStack, server netip.AddrPort, domain string) ([]netip.Addr, error) {
	packetConn, err := device.ListenUDP(ctx, "udp", netip.AddrPort{})
	if err != nil {
		return nil, err
	}
	defer packetConn.Close()

	client := &mDNS.Client{
		UDPSize: mDNS.DefaultMsgSize,
		Timeout: dnsTimeout,
	}
	var addresses []netip.Addr
	for _, qtype := range []uint16{mDNS.TypeA, mDNS.TypeAAAA} {
		message := &mDNS.Msg{}
		message.SetQuestion(mDNS.Fqdn(domain), qtype)
		response, _, err := client.ExchangeContext(ctx, message, server.String())
		if err != nil {
			return nil, err
		}
		if response.Rcode != mDNS.RcodeSuccess {
			return nil, fmt.Errorf("DNS query %s %s: rcode %d", domain, mDNS.TypeToString[qtype], response.Rcode)
		}
		for _, record := range response.Answer {
			switch answer := record.(type) {
			case *mDNS.A:
				addresses = append(addresses, netip.AddrFrom4([4]byte(answer.A)))
			case *mDNS.AAAA:
				addresses = append(addresses, netip.AddrFrom16([16]byte(answer.AAAA)))
			}
		}
	}
	return addresses, nil
}
