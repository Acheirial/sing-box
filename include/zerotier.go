//go:build with_zerotier

package include

import (
	"github.com/sagernet/sing-box/adapter/outbound"
	"github.com/sagernet/sing-box/protocol/zerotier"
)

func registerZeroTierOutbound(registry *outbound.Registry) {
	zerotier.RegisterOutbound(registry)
}
