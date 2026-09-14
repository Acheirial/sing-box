package gost

import (
	"context"
	"net"
	"time"

	"github.com/sagernet/sing-box/common/tls"
	M "github.com/sagernet/sing/common/metadata"
	"github.com/sagernet/smux"
)

// TLSWrapper performs a TLS client handshake on an existing connection. The
// outbound builds the TLS configuration from its OutboundTLSOptionsContainer.
type TLSWrapper func(ctx context.Context, conn net.Conn, serverAddress M.Socksaddr) (net.Conn, error)

// NewTLSWrapper returns a TLSWrapper using the sing-box TLS client stack.
func NewTLSWrapper(tlsConfig tls.Config) TLSWrapper {
	return func(ctx context.Context, conn net.Conn, serverAddress M.Socksaddr) (net.Conn, error) {
		return tls.ClientHandshake(ctx, conn, tlsConfig)
	}
}

// muxSession wraps a connection in a smux session and opens a stream.
func muxSession(conn net.Conn) (net.Conn, error) {
	config := smux.DefaultConfig()
	config.KeepAliveDisabled = true

	session, err := smux.Client(conn, config)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	stream, err := session.OpenStream()
	if err != nil {
		_ = session.Close()
		return nil, err
	}

	return &muxConn{
		Conn:    stream,
		session: session,
	}, nil
}

// muxConn is a wrapper around smux.Stream that also closes the session when closed.
type muxConn struct {
	net.Conn
	session *smux.Session
}

func (m *muxConn) Close() error {
	streamErr := m.Conn.Close()
	sessionErr := m.session.Close()

	// Return stream error if there is one, otherwise return session error
	if streamErr != nil {
		return streamErr
	}
	return sessionErr
}

func (m *muxConn) SetDeadline(t time.Time) error {
	return m.Conn.SetDeadline(t)
}
