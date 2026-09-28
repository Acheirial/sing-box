package option

type VLESSInboundOptions struct {
	ListenOptions
	Users []VLESSUser `json:"users,omitempty"`
	InboundTLSOptionsContainer
	Multiplex *InboundMultiplexOptions `json:"multiplex,omitempty"`
	Transport *V2RayTransportOptions   `json:"transport,omitempty"`
	// Decryption enables VLESS Encryption on this inbound. Empty or "none"
	// leaves the connection untouched. Otherwise it uses the
	// "mlkem768x25519plus" grammar, matching Xray's `decryption`.
	Decryption string `json:"decryption,omitempty"`
}

type VLESSUser struct {
	Name string `json:"name"`
	UUID string `json:"uuid"`
	Flow string `json:"flow,omitempty"`
}

type VLESSOutboundOptions struct {
	DialerOptions
	ServerOptions
	UUID       string      `json:"uuid"`
	Flow       string      `json:"flow,omitempty"`
	VlessRoute uint16      `json:"vless_route,omitempty"`
	Network    NetworkList `json:"network,omitempty"`
	// Encryption enables VLESS Encryption on this outbound. Empty or "none"
	// leaves the connection untouched. Otherwise it uses the
	// "mlkem768x25519plus" grammar, matching Xray's account.Encryption.
	Encryption string `json:"encryption,omitempty"`
	OutboundTLSOptionsContainer
	Multiplex      *OutboundMultiplexOptions `json:"multiplex,omitempty"`
	Transport      *V2RayTransportOptions    `json:"transport,omitempty"`
	PacketEncoding *string                   `json:"packet_encoding,omitempty"`
}
