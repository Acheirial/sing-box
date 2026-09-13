package tls

import (
	"context"
	"net"

	"github.com/sagernet/sing-box/common/jls"
	"github.com/sagernet/sing-box/common/restls"
	"github.com/sagernet/sing-box/option"
	E "github.com/sagernet/sing/common/exceptions"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
)

// NewJLSClientConfig builds a JLS client configuration, or returns nil when JLS is not enabled.
// The enclosing outbound must have tls enabled.
func NewJLSClientConfig(serverAddress string, tlsOptions option.OutboundTLSOptions, jlsOptions *option.JLSOptions) (*jls.ClientConfig, error) {
	if jlsOptions == nil {
		return nil, nil
	}
	if !tlsOptions.Enabled {
		return nil, E.New("jls requires tls")
	}
	serverName := tlsOptions.ServerName
	if serverName == "" {
		serverName = serverAddress
	}
	if serverName == "" {
		return nil, E.New("missing server_name for jls")
	}
	clientConfig, err := jls.NewClientConfig(serverName, jlsOptions.Username, jlsOptions.Password, tlsOptions.ALPN)
	if err != nil {
		return nil, E.Cause(err, "initialize jls")
	}
	if tlsOptions.UTLS != nil {
		clientConfig.ClientFingerprint = tlsOptions.UTLS.Fingerprint
	}
	return clientConfig, nil
}

// NewRestlsConfig builds a RestLS client configuration, or returns nil when RestLS is not enabled.
// The enclosing outbound must have tls enabled.
func NewRestlsConfig(serverAddress string, tlsOptions option.OutboundTLSOptions, restlsOptions *option.RestLSOptions) (*restls.Config, error) {
	if restlsOptions == nil {
		return nil, nil
	}
	if !tlsOptions.Enabled {
		return nil, E.New("restls requires tls")
	}
	serverName := tlsOptions.ServerName
	if serverName == "" {
		serverName = serverAddress
	}
	if serverName == "" {
		return nil, E.New("missing server_name for restls")
	}
	config, err := restls.NewRestlsConfig(serverName, restlsOptions.Password, restlsOptions.VersionHint, restlsOptions.RestlsScript, "")
	if err != nil {
		return nil, E.Cause(err, "initialize restls")
	}
	config.InsecureSkipVerify = tlsOptions.Insecure
	config.NextProtos = tlsOptions.ALPN
	return config, nil
}

// WrapJLSRestls wraps an established raw connection with RestLS and JLS layers,
// in the order RestLS first, then JLS, mirroring mihomo's TLS → Restls → JLS stacking.
// Configs may be nil to skip the corresponding layer.
func WrapJLSRestls(ctx context.Context, conn net.Conn, restlsConfig *restls.Config, jlsConfig *jls.ClientConfig) (net.Conn, error) {
	if restlsConfig != nil {
		var err error
		conn, err = restls.NewRestls(ctx, conn, restlsConfig)
		if err != nil {
			return nil, E.Cause(err, "restls handshake")
		}
	}
	if jlsConfig != nil {
		var err error
		conn, err = jls.NewClient(ctx, conn, jlsConfig)
		if err != nil {
			return nil, E.Cause(err, "jls handshake")
		}
	}
	return conn, nil
}

// JLSRestlsDialer wraps a dialer to apply RestLS and JLS layers on every TCP connection.
// It is a no-op wrapper when neither layer is configured.
type JLSRestlsDialer struct {
	N.Dialer
	restlsConfig *restls.Config
	jlsConfig    *jls.ClientConfig
}

func NewJLSRestlsDialer(dialer N.Dialer, restlsConfig *restls.Config, jlsConfig *jls.ClientConfig) N.Dialer {
	if restlsConfig == nil && jlsConfig == nil {
		return dialer
	}
	return &JLSRestlsDialer{
		Dialer:       dialer,
		restlsConfig: restlsConfig,
		jlsConfig:    jlsConfig,
	}
}

func (d *JLSRestlsDialer) DialContext(ctx context.Context, network string, destination M.Socksaddr) (net.Conn, error) {
	conn, err := d.Dialer.DialContext(ctx, network, destination)
	if err != nil {
		return nil, err
	}
	conn, err = WrapJLSRestls(ctx, conn, d.restlsConfig, d.jlsConfig)
	if err != nil {
		conn.Close()
		return nil, err
	}
	return conn, nil
}

func (d *JLSRestlsDialer) ListenPacket(ctx context.Context, destination M.Socksaddr) (net.PacketConn, error) {
	return d.Dialer.ListenPacket(ctx, destination)
}

func (d *JLSRestlsDialer) Upstream() any {
	return d.Dialer
}
