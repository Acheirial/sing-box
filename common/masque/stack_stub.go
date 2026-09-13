//go:build !with_gvisor

package masque

import (
	"context"
	"net"
	"net/netip"

	"github.com/sagernet/sing/common/exceptions"
	M "github.com/sagernet/sing/common/metadata"
)

// packetStack is a stub for builds without the gVisor IP stack.
type PacketStack struct{}

func NewPacketStack(localPrefixes []netip.Prefix, mtu uint32) (*PacketStack, error) {
	return nil, exceptions.New(`masque requires the with_gvisor build tag for its userspace IP stack`)
}

func (s *PacketStack) InjectPacket(packet []byte) {}

func (s *PacketStack) Outbound() <-chan []byte {
	return nil
}

func (s *PacketStack) DialContext(ctx context.Context, network string, destination M.Socksaddr) (net.Conn, error) {
	return nil, exceptions.New(`masque requires the with_gvisor build tag for its userspace IP stack`)
}

func (s *PacketStack) ListenPacket(ctx context.Context, destination M.Socksaddr) (net.PacketConn, error) {
	return nil, exceptions.New(`masque requires the with_gvisor build tag for its userspace IP stack`)
}

func (s *PacketStack) Close() error {
	return nil
}
