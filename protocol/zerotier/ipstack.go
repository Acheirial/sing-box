package zerotier

import (
	"context"
	"net"
	"net/netip"
	"strings"

	"github.com/sagernet/sing-tun"
	E "github.com/sagernet/sing/common/exceptions"
)

// IP stack modes, mirroring mihomo's IPStackOption.
const (
	ipStackAuto   = "auto"
	ipStackSystem = "system"
	ipStackGVisor = "gvisor"
	ipStackMixed  = "mixed"
)

func normalizeIPStack(mode string) string {
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode == "" {
		return ipStackAuto
	}
	return mode
}

func validateIPStack(mode string) error {
	switch mode {
	case ipStackAuto, ipStackSystem, ipStackGVisor, ipStackMixed:
		return nil
	default:
		return E.New("invalid ZeroTier ip_stack mode ", mode, "; expected auto, system, gvisor, or mixed")
	}
}

func newIPStack(mode string, localAddresses []netip.Prefix, mtu uint32) (ipStack, error) {
	if mode == ipStackAuto {
		if tun.WithGVisor {
			mode = ipStackGVisor
		} else {
			mode = ipStackSystem
		}
	}
	switch mode {
	case ipStackGVisor, ipStackMixed, ipStackSystem:
		if !tun.WithGVisor {
			return nil, E.New(`gVisor is not included in this build, rebuild with -tags with_gvisor`)
		}
		return newStackDevice(localAddresses, mtu)
	default:
		return nil, E.New("invalid ZeroTier IP stack mode")
	}
}

// ipStack is the packet and socket surface of one virtual ZeroTier link,
// adapted from sing-box's WireGuard gVisor stack device. ZeroTier virtual
// networks have no kernel TUN, so all IP stack modes route through the
// gVisor netstack; system and mixed modes only change TCP socket defaults.
type ipStack interface {
	Start() error
	DialTCP(ctx context.Context, network string, source, destination netip.AddrPort) (net.Conn, error)
	DialUDP(ctx context.Context, network string, source, destination netip.AddrPort) (net.Conn, error)
	ListenTCP(ctx context.Context, network string, local netip.AddrPort) (net.Listener, error)
	ListenUDP(ctx context.Context, network string, local netip.AddrPort) (net.PacketConn, error)
	Read(buffers [][]byte, sizes []int, offset int) (count int, err error)
	Write(buffers [][]byte, offset int) (count int, err error)
	MTU() (int, error)
	Name() (string, error)
	BatchSize() int
	Close() error
}
