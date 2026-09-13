package option

import "github.com/sagernet/sing/common/json/badoption"

type ZeroTierOutboundOptions struct {
	DialerOptions
	Network           string                     `json:"network"`
	StateDir          string                     `json:"state_dir,omitempty"`
	IdentitySecret    string                     `json:"identity_secret,omitempty"`
	Planet            string                     `json:"planet,omitempty"`
	MTU               uint32                     `json:"mtu,omitempty"`
	IPStack           string                     `json:"ip_stack,omitempty"`
	PhysicalMTU       uint32                     `json:"physical_mtu,omitempty"`
	UDP               bool                       `json:"udp,omitempty"`
	RemoteDnsResolve  bool                       `json:"remote_dns_resolve,omitempty"`
	DNS               badoption.Listable[string] `json:"dns,omitempty"`
	LowBandwidth      bool                       `json:"low_bandwidth,omitempty"`
	EncryptedHello    bool                       `json:"encrypted_hello,omitempty"`
	PrimaryPort       uint16                     `json:"primary_port,omitempty"`
	SecondaryPort     uint16                     `json:"secondary_port,omitempty"`
	TCPFallbackMode   string                     `json:"tcp_fallback_mode,omitempty"`
	TCPFallbackRelay  string                     `json:"tcp_fallback_relay,omitempty"`
	Orbit             []ZeroTierOrbitOptions     `json:"orbit,omitempty"`
	RemoteTraceTarget string                     `json:"remote_trace_target,omitempty"`
	RemoteTraceLevel  uint64                     `json:"remote_trace_level,omitempty"`
}

type ZeroTierOrbitOptions struct {
	World string `json:"world"`
	Seed  string `json:"seed"`
}
