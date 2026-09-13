package sip003

import (
	"context"
	"net"
	"strings"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/common/jls"
	E "github.com/sagernet/sing/common/exceptions"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
)

var _ Plugin = (*JLSPlugin)(nil)

func init() {
	RegisterPlugin("jls", newJLSPlugin)
}

func newJLSPlugin(ctx context.Context, pluginOpts Args, router adapter.Router, dialer N.Dialer, serverAddr M.Socksaddr) (Plugin, error) {
	var username, password string
	var alpn []string
	if value, loaded := pluginOpts.Get("username"); loaded {
		username = value
	}
	if value, loaded := pluginOpts.Get("password"); loaded {
		password = value
	}
	if value, loaded := pluginOpts.Get("alpn"); loaded {
		alpn = strings.Split(value, ",")
	}
	host := serverAddr.AddrString()
	if value, loaded := pluginOpts.Get("host"); loaded {
		host = value
	}
	clientConfig, err := jls.NewClientConfig(host, username, password, alpn)
	if err != nil {
		return nil, E.Cause(err, "initialize jls plugin")
	}
	return &JLSPlugin{
		dialer:     dialer,
		serverAddr: M.ParseSocksaddrHostPort(host, serverAddr.Port),
		config:     clientConfig,
	}, nil
}

type JLSPlugin struct {
	dialer     N.Dialer
	serverAddr M.Socksaddr
	config     *jls.ClientConfig
}

func (j *JLSPlugin) DialContext(ctx context.Context) (net.Conn, error) {
	conn, err := j.dialer.DialContext(ctx, N.NetworkTCP, j.serverAddr)
	if err != nil {
		return nil, err
	}
	conn, err = jls.NewClient(ctx, conn, j.config)
	if err != nil {
		conn.Close()
		return nil, E.Cause(err, "jls handshake")
	}
	return conn, nil
}
