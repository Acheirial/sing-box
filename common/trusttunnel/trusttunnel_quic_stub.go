//go:build !with_quic

package trusttunnel

import (
	"errors"

	"github.com/sagernet/sing-box/common/tls"
)

func (c *Client) quicRoundTripper(tlsConfig tls.Config, congestionControlName string, cwnd int, bbrProfile string) error {
	return errors.New(`trusttunnel: QUIC is not included in this build, rebuild with -tags with_quic`)
}
