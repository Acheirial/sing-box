//go:build !with_gvisor

package zerotier

import (
	"net/netip"

	"github.com/sagernet/sing-tun"
)

func newStackDevice(localAddresses []netip.Prefix, mtu uint32) (ipStack, error) {
	return nil, tun.ErrGVisorNotIncluded
}
