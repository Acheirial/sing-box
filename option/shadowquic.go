package option

import "github.com/sagernet/sing/common/json/badoption"

type ShadowQUICOutboundOptions struct {
	DialerOptions
	ServerOptions
	Username             string                     `json:"username,omitempty"`
	Password             string                     `json:"password,omitempty"`
	ALPN                 badoption.Listable[string] `json:"alpn,omitempty"`
	QUICVersions         badoption.Listable[string] `json:"quic_versions,omitempty"`
	UDPOverStream        bool                       `json:"udp_over_stream,omitempty"`
	ZeroRTT              bool                       `json:"zero_rtt,omitempty"`
	KeepAliveInterval    badoption.Duration         `json:"keep_alive_interval,omitempty"`
	CongestionController string                     `json:"congestion_controller,omitempty" enum:"cubic,new_reno,bbr_meta_v1,bbr_meta_v2,bbr"`
	Up                   string                     `json:"up,omitempty"`
	Down                 string                     `json:"down,omitempty"`
	CWND                 int                        `json:"cwnd,omitempty"`
	BBRProfile           string                     `json:"bbr_profile,omitempty" enum:"standard,conservative,aggressive"`
	ReceiveWindowConn    uint32                     `json:"receive_window_conn,omitempty"`
	ReceiveWindow        uint32                     `json:"receive_window,omitempty"`
	DisableMTUDiscovery  bool                       `json:"disable_mtu_discovery,omitempty"`
	MaxDatagramFrameSize int                        `json:"max_datagram_frame_size,omitempty"`
	MaxOpenStreams       int                        `json:"max_open_streams,omitempty"`
	OutboundTLSOptionsContainer
}
