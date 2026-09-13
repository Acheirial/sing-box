package option

type MieruOutboundOptions struct {
	DialerOptions
	ServerOptions
	PortRange      string `json:"port_range,omitempty"`
	Transport      string `json:"transport,omitempty"`
	UDP            bool   `json:"udp,omitempty"`
	Username       string `json:"username,omitempty"`
	Password       string `json:"password,omitempty"`
	Multiplexing   string `json:"multiplexing,omitempty"`
	HandshakeMode  string `json:"handshake_mode,omitempty"`
	TrafficPattern string `json:"traffic_pattern,omitempty"`
}
