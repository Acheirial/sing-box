package option

type GostOutboundOptions struct {
	DialerOptions
	ServerOptions
	OutboundTLSOptionsContainer
	Forward  bool   `json:"forward,omitempty"`
	UDP      bool   `json:"udp,omitempty"`
	Mux      bool   `json:"mux,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}
