package sip003

import (
	"context"
	"net"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/common/restls"
	E "github.com/sagernet/sing/common/exceptions"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
)

var _ Plugin = (*RestlsPlugin)(nil)

func init() {
	RegisterPlugin("restls", newRestlsPlugin)
}

func newRestlsPlugin(ctx context.Context, pluginOpts Args, router adapter.Router, dialer N.Dialer, serverAddr M.Socksaddr) (Plugin, error) {
	var password, versionHint, restlsScript string
	var skipCertVerify, forceTLS12 bool
	var fingerprint, nameCertVerify string
	if value, loaded := pluginOpts.Get("password"); loaded {
		password = value
	}
	if value, loaded := pluginOpts.Get("version_hint"); loaded {
		versionHint = value
	}
	if value, loaded := pluginOpts.Get("restls_script"); loaded {
		restlsScript = value
	}
	if value, loaded := pluginOpts.Get("skip_cert_verify"); loaded {
		skipCertVerify = value == "true" || value == "1"
	}
	if value, loaded := pluginOpts.Get("force_tls12"); loaded {
		forceTLS12 = value == "true" || value == "1"
	}
	if value, loaded := pluginOpts.Get("fingerprint"); loaded {
		fingerprint = value
	}
	if value, loaded := pluginOpts.Get("name_cert_verify"); loaded {
		nameCertVerify = value
	}
	host := serverAddr.AddrString()
	if value, loaded := pluginOpts.Get("host"); loaded {
		host = value
	}
	config, err := restls.NewRestlsConfig(host, password, versionHint, restlsScript, "")
	if err != nil {
		return nil, E.Cause(err, "initialize restls plugin")
	}
	config.InsecureSkipVerify = skipCertVerify
	if fingerprint != "" {
		err = restls.SetFingerprint(config, fingerprint, nameCertVerify)
		if err != nil {
			return nil, E.Cause(err, "initialize restls plugin")
		}
	} else if nameCertVerify != "" {
		restls.SetNameCertVerify(config, nameCertVerify)
	}
	config.ForceTLS12 = forceTLS12
	return &RestlsPlugin{
		dialer:     dialer,
		serverAddr: M.ParseSocksaddrHostPort(host, serverAddr.Port),
		config:     config,
	}, nil
}

type RestlsPlugin struct {
	dialer     N.Dialer
	serverAddr M.Socksaddr
	config     *restls.Config
}

func (r *RestlsPlugin) DialContext(ctx context.Context) (net.Conn, error) {
	conn, err := r.dialer.DialContext(ctx, N.NetworkTCP, r.serverAddr)
	if err != nil {
		return nil, err
	}
	conn, err = restls.NewRestls(ctx, conn, r.config)
	if err != nil {
		conn.Close()
		return nil, E.Cause(err, "restls handshake")
	}
	return conn, nil
}
