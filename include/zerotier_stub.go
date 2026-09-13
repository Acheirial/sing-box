//go:build !with_zerotier

package include

import (
	"context"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/adapter/outbound"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	E "github.com/sagernet/sing/common/exceptions"
)

func registerZeroTierOutbound(registry *outbound.Registry) {
	outbound.Register[option.ZeroTierOutboundOptions](registry, C.TypeZeroTier, func(ctx context.Context, router adapter.Router, logger log.ContextLogger, tag string, options option.ZeroTierOutboundOptions) (adapter.Outbound, error) {
		return nil, E.New("missing build tag: with_zerotier")
	})
}
