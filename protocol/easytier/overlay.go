//go:build with_easytier

package easytier

import (
	"encoding/binary"
	"fmt"
	"net/netip"
	"strings"
)

// DefaultTLDDNSZone is the default MagicDNS zone used by EasyTier.
const DefaultTLDDNSZone = "et.net"

// Node describes one overlay IPv4 hostname mapping.
type Node struct {
	Hostname string
	IPv4     netip.Addr
}

// normalizeDNSName lowercases a name and strips a trailing dot.
func normalizeDNSName(name string) string {
	return strings.ToLower(strings.TrimSuffix(strings.TrimSpace(name), "."))
}

// normalizeZone returns a MagicDNS zone without a trailing dot.
func normalizeZone(zone string) string {
	zone = normalizeDNSName(zone)
	if zone == "" {
		return DefaultTLDDNSZone
	}
	return zone
}

// overlayNames returns hostname forms that MagicDNS may use.
func overlayNames(hostname, zone string) []string {
	hostname = normalizeDNSName(hostname)
	if hostname == "" {
		return nil
	}
	zone = normalizeZone(zone)
	names := []string{hostname}
	if hostname != zone && !strings.HasSuffix(hostname, "."+zone) {
		names = append(names, hostname+"."+zone)
	}
	return names
}

// isMagicDNS reports whether host should stay on the overlay resolver.
func isMagicDNS(host, zone string) bool {
	host = normalizeDNSName(host)
	if host == "" {
		return false
	}
	zone = normalizeZone(zone)
	return host == zone || strings.HasSuffix(host, "."+zone)
}

// lookupOverlayHost finds an overlay IPv4 address for host.
func lookupOverlayHost(host, zone string, nodes []Node) (netip.Addr, bool) {
	host = normalizeDNSName(host)
	if host == "" {
		return netip.Addr{}, false
	}
	for _, node := range nodes {
		if !node.IPv4.IsValid() || !node.IPv4.Is4() {
			continue
		}
		for _, name := range overlayNames(node.Hostname, zone) {
			if host == name {
				return node.IPv4, true
			}
		}
	}
	return netip.Addr{}, false
}

// parseNodeIPv4 parses a node IPv4 address or CIDR such as "10.144.0.1/24".
func parseNodeIPv4(value string) (netip.Addr, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return netip.Addr{}, fmt.Errorf("empty overlay IPv4 address")
	}
	if prefix, err := netip.ParsePrefix(value); err == nil {
		return prefix.Addr().Unmap(), nil
	}
	ip, err := netip.ParseAddr(value)
	if err != nil {
		return netip.Addr{}, err
	}
	return ip.Unmap(), nil
}

// ipv4FromUint32 converts a big-endian IPv4 integer to an address.
func ipv4FromUint32(addr uint32) netip.Addr {
	var bytes [4]byte
	binary.BigEndian.PutUint32(bytes[:], addr)
	return netip.AddrFrom4(bytes)
}
