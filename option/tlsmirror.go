package option

import "github.com/sagernet/sing/common/json/badoption"

type TLSMirrorOutboundOptions struct {
	DialerOptions
	ServerOptions
	OutboundTLSOptionsContainer
	PrimaryKey                    string                         `json:"primary_key"`
	ExplicitNonceCipherSuites     badoption.Listable[uint16]     `json:"explicit_nonce_ciphersuites,omitempty"`
	DeferInstanceDerivedWriteTime TLSMirrorTimeSpec              `json:"defer_instance_derived_write_time,omitempty"`
	TransportLayerPadding         TLSMirrorTransportLayerPadding `json:"transport_layer_padding,omitempty"`
	ConnectionEnrolment           *TLSMirrorConnectionEnrolment  `json:"connection_enrolment,omitempty"`
	EmbeddedTrafficGenerator      TLSMirrorTrafficGenerator      `json:"embedded_traffic_generator,omitempty"`
	SequenceWatermarkingEnabled   bool                           `json:"sequence_watermarking_enabled,omitempty"`
}

type TLSMirrorConnectionEnrolment struct {
	PrimaryIngressOutbound string `json:"primary_ingress_outbound,omitempty"`
	PrimaryEgressOutbound  string `json:"primary_egress_outbound,omitempty"`
}

type TLSMirrorTimeSpec struct {
	BaseNanoseconds                    uint64 `json:"base_nanoseconds,omitempty"`
	UniformRandomMultiplierNanoseconds uint64 `json:"uniform_random_multiplier_nanoseconds,omitempty"`
}

type TLSMirrorTransportLayerPadding struct {
	Enabled bool `json:"enabled,omitempty"`
}

type TLSMirrorTrafficGenerator struct {
	Steps []TLSMirrorTrafficStep `json:"steps,omitempty"`
}

type TLSMirrorTrafficStep struct {
	Name                         string                              `json:"name,omitempty"`
	Host                         string                              `json:"host,omitempty"`
	Path                         string                              `json:"path,omitempty"`
	Method                       string                              `json:"method,omitempty"`
	Headers                      []TLSMirrorTrafficHeader            `json:"headers,omitempty"`
	NextStep                     []TLSMirrorTrafficTransferCandidate `json:"next_step,omitempty"`
	ConnectionReady              bool                                `json:"connection_ready,omitempty"`
	ConnectionRecallExit         bool                                `json:"connection_recall_exit,omitempty"`
	WaitTime                     TLSMirrorTimeSpec                   `json:"wait_time,omitempty"`
	H2DoNotWaitForDownloadFinish bool                                `json:"h2_do_not_wait_for_download_finish,omitempty"`
}

type TLSMirrorTrafficHeader struct {
	Name   string   `json:"name,omitempty"`
	Value  string   `json:"value,omitempty"`
	Values []string `json:"values,omitempty"`
}

type TLSMirrorTrafficTransferCandidate struct {
	Weight       int32 `json:"weight,omitempty"`
	GotoLocation int   `json:"goto_location,omitempty"`
}
