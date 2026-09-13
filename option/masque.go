package option

type MasqueOutboundOptions struct {
	DialerOptions
	ServerOptions
	PrivateKey        string   `json:"private_key"`
	PublicKey         string   `json:"public_key"`
	IPv4Address       string   `json:"ip,omitempty"`
	IPv6Address       string   `json:"ipv6,omitempty"`
	URI               string   `json:"uri,omitempty"`
	MTU               int      `json:"mtu,omitempty"`
	UDP               bool     `json:"udp,omitempty"`
	HandshakeTimeout  int      `json:"handshake_timeout,omitempty"`
	SkipCertVerify    bool     `json:"skip_cert_verify,omitempty"`
	Network           string   `json:"network,omitempty" enum:"h3,h3-l4proxy"`
	CongestionControl string   `json:"congestion_controller,omitempty"`
	CWND              int      `json:"cwnd,omitempty"`
	BBRProfile        string   `json:"bbr_profile,omitempty" enum:"standard,conservative,aggressive"`
	RemoteDnsResolve  bool     `json:"remote_dns_resolve,omitempty"`
	DNS               []string `json:"dns,omitempty"`
}

// LocalPrefixes returns the local IP prefixes assigned to the MASQUE tunnel.
func (o MasqueOutboundOptions) LocalPrefixes() []string {
	var prefixes []string
	if o.IPv4Address != "" {
		prefixes = append(prefixes, o.IPv4Address)
	}
	if o.IPv6Address != "" {
		prefixes = append(prefixes, o.IPv6Address)
	}
	return prefixes
}
