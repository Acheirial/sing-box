package option

type TrustTunnelOutboundOptions struct {
	DialerOptions
	ServerOptions
	OutboundTLSOptionsContainer
	Username    string `json:"username,omitempty"`
	Password    string `json:"password,omitempty"`
	UDP         bool   `json:"udp,omitempty"`
	HealthCheck bool   `json:"health_check,omitempty"`
	// QUIC options.
	Quic                 bool   `json:"quic,omitempty"`
	CongestionController string `json:"congestion_controller,omitempty"`
	CWND                 int    `json:"cwnd,omitempty"`
	BBRProfile           string `json:"bbr_profile,omitempty" enum:"standard,conservative,aggressive"`
	// Reuse options.
	MaxConnections int `json:"max_connections,omitempty"`
	MinStreams     int `json:"min_streams,omitempty"`
	MaxStreams     int `json:"max_streams,omitempty"`
}
