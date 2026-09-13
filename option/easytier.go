package option

import "github.com/sagernet/sing/common/json/badoption"

type EasyTierOutboundOptions struct {
	DialerOptions
	NetworkName         string                     `json:"network_name,omitempty"`
	NetworkSecret       string                     `json:"network_secret,omitempty"`
	Hostname            string                     `json:"hostname,omitempty"`
	IPv4                string                     `json:"ipv4,omitempty"`
	DHCP                bool                       `json:"dhcp,omitempty"`
	Peers               badoption.Listable[string] `json:"peers,omitempty"`
	Listeners           badoption.Listable[string] `json:"listeners,omitempty"`
	NoListener          *bool                      `json:"no_listener,omitempty"`
	MappedListeners     badoption.Listable[string] `json:"mapped_listeners,omitempty"`
	ExitNodes           badoption.Listable[string] `json:"exit_nodes,omitempty"`
	ProxyNetworks       badoption.Listable[string] `json:"proxy_networks,omitempty"`
	InstanceName        string                     `json:"instance_name,omitempty"`
	StateDirectory      string                     `json:"state_directory,omitempty"`
	UDP                 bool                       `json:"udp,omitempty"`
	AcceptDNS           *bool                      `json:"accept_dns,omitempty"`
	EnableExitNode      *bool                      `json:"enable_exit_node,omitempty"`
	EnableEncryption    *bool                      `json:"enable_encryption,omitempty"`
	EncryptionAlgorithm string                     `json:"encryption_algorithm,omitempty"`
	PrivateMode         *bool                      `json:"private_mode,omitempty"`
	LatencyFirst        *bool                      `json:"latency_first,omitempty"`
	DisableP2P          *bool                      `json:"disable_p2p,omitempty"`
	EnableKCPProxy      *bool                      `json:"enable_kcp_proxy,omitempty"`
	DisableKCPInput     *bool                      `json:"disable_kcp_input,omitempty"`
	EnableQUICProxy     *bool                      `json:"enable_quic_proxy,omitempty"`
	DisableQUICInput    *bool                      `json:"disable_quic_input,omitempty"`
	MTU                 int                        `json:"mtu,omitempty"`
	TLDDNSZone          string                     `json:"tld_dns_zone,omitempty"`
	SecureMode          *bool                      `json:"secure_mode,omitempty"`
	LocalPrivateKey     string                     `json:"local_private_key,omitempty"`
	LocalPublicKey      string                     `json:"local_public_key,omitempty"`
}
