//go:build !with_utls

package jls

import (
	"context"
	"net"
)

func newUTLSClient(ctx context.Context, conn net.Conn, config *ClientConfig) (net.Conn, bool, error) {
	return nil, false, nil
}
