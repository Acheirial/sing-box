package option

import (
	"reflect"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/schema"
	E "github.com/sagernet/sing/common/exceptions"
	"github.com/sagernet/sing/common/json"
	"github.com/sagernet/sing/common/json/badjson"
	"github.com/sagernet/sing/common/json/badoption"
)

const (
	V2RayTransportTypeMKCP  = "mkcp"
	V2RayTransportTypeMEKYA = "mekya"

	V2RayTransportTypeXHTTP = "xhttp"
)

type _V2RayTransportOptions struct {
	Type               string                  `json:"type" enum:"http,ws,quic,grpc,httpupgrade,mkcp,mekya,xhttp"`
	HTTPOptions        V2RayHTTPOptions        `json:"-"`
	WebsocketOptions   V2RayWebsocketOptions   `json:"-"`
	QUICOptions        V2RayQUICOptions        `json:"-"`
	GRPCOptions        V2RayGRPCOptions        `json:"-"`
	HTTPUpgradeOptions V2RayHTTPUpgradeOptions `json:"-"`
	MKCPOptions        V2RayMKCPOptions        `json:"-"`
	MEKYAOptions       V2RayMEKYAOptions       `json:"-"`
	XHTTPOptions       V2RayXHTTPOptions       `json:"-"`
}

type V2RayTransportOptions _V2RayTransportOptions

func (o V2RayTransportOptions) MarshalJSON() ([]byte, error) {
	var v any
	switch o.Type {
	case C.V2RayTransportTypeHTTP:
		v = o.HTTPOptions
	case C.V2RayTransportTypeWebsocket:
		v = o.WebsocketOptions
	case C.V2RayTransportTypeQUIC:
		v = o.QUICOptions
	case C.V2RayTransportTypeGRPC:
		v = o.GRPCOptions
	case V2RayTransportTypeMKCP:
		v = o.MKCPOptions
	case V2RayTransportTypeMEKYA:
		v = o.MEKYAOptions
	case C.V2RayTransportTypeHTTPUpgrade:
		v = o.HTTPUpgradeOptions
	case V2RayTransportTypeXHTTP:
		v = o.XHTTPOptions
	case "":
		return nil, E.New("missing transport type")
	default:
		return nil, E.New("unknown transport type: " + o.Type)
	}
	return badjson.MarshallObjects(_V2RayTransportOptions(o), v)
}

func (o *V2RayTransportOptions) UnmarshalJSON(bytes []byte) error {
	err := json.Unmarshal(bytes, (*_V2RayTransportOptions)(o))
	if err != nil {
		return err
	}
	var v any
	switch o.Type {
	case C.V2RayTransportTypeHTTP:
		v = &o.HTTPOptions
	case C.V2RayTransportTypeWebsocket:
		v = &o.WebsocketOptions
	case C.V2RayTransportTypeQUIC:
		v = &o.QUICOptions
	case C.V2RayTransportTypeGRPC:
		v = &o.GRPCOptions
	case V2RayTransportTypeMKCP:
		v = &o.MKCPOptions
	case V2RayTransportTypeMEKYA:
		v = &o.MEKYAOptions
	case C.V2RayTransportTypeHTTPUpgrade:
		v = &o.HTTPUpgradeOptions
	case V2RayTransportTypeXHTTP:
		v = &o.XHTTPOptions
	default:
		return E.New("unknown transport type: " + o.Type)
	}
	err = badjson.UnmarshallExcluded(bytes, (*_V2RayTransportOptions)(o), v)
	if err != nil {
		return err
	}
	return nil
}

func (o V2RayTransportOptions) DescribeSchema(builder schema.Builder) (*schema.Node, error) {
	return builder.Define("V2RayTransport", func() (*schema.Node, error) {
		return schema.DiscriminatedUnion(builder, "type", true, []schema.UnionVariant{
			{Value: C.V2RayTransportTypeHTTP, StructType: reflect.TypeFor[V2RayHTTPOptions]()},
			{Value: C.V2RayTransportTypeWebsocket, StructType: reflect.TypeFor[V2RayWebsocketOptions]()},
			{Value: C.V2RayTransportTypeQUIC, StructType: reflect.TypeFor[V2RayQUICOptions]()},
			{Value: V2RayTransportTypeMKCP, StructType: reflect.TypeFor[V2RayMKCPOptions]()},
			{Value: V2RayTransportTypeMEKYA, StructType: reflect.TypeFor[V2RayMEKYAOptions]()},
			{Value: C.V2RayTransportTypeGRPC, StructType: reflect.TypeFor[V2RayGRPCOptions]()},
			{Value: C.V2RayTransportTypeHTTPUpgrade, StructType: reflect.TypeFor[V2RayHTTPUpgradeOptions]()},
			{Value: V2RayTransportTypeXHTTP, StructType: reflect.TypeFor[V2RayXHTTPOptions]()},
		}, nil)
	})
}

type V2RayHTTPOptions struct {
	Host        badoption.Listable[string] `json:"host,omitempty"`
	Path        string                     `json:"path,omitempty"`
	Method      string                     `json:"method,omitempty"`
	Headers     badoption.HTTPHeader       `json:"headers,omitempty"`
	IdleTimeout badoption.Duration         `json:"idle_timeout,omitempty"`
	PingTimeout badoption.Duration         `json:"ping_timeout,omitempty"`
}

type V2RayWebsocketOptions struct {
	Path                string               `json:"path,omitempty"`
	Headers             badoption.HTTPHeader `json:"headers,omitempty"`
	MaxEarlyData        uint32               `json:"max_early_data,omitempty"`
	EarlyDataHeaderName string               `json:"early_data_header_name,omitempty"`
}

type V2RayQUICOptions struct{}

type V2RayGRPCOptions struct {
	ServiceName         string             `json:"service_name,omitempty"`
	IdleTimeout         badoption.Duration `json:"idle_timeout,omitempty"`
	PingTimeout         badoption.Duration `json:"ping_timeout,omitempty"`
	PermitWithoutStream bool               `json:"permit_without_stream,omitempty"`
	ForceLite           bool               `json:"-"` // for test
}

type V2RayHTTPUpgradeOptions struct {
	Host    string               `json:"host,omitempty"`
	Path    string               `json:"path,omitempty"`
	Headers badoption.HTTPHeader `json:"headers,omitempty"`
}

type V2RayMKCPOptions struct {
	MTU              uint32 `json:"mtu,omitempty"`
	TTI              uint32 `json:"tti,omitempty"`
	UplinkCapacity   uint32 `json:"uplink_capacity,omitempty"`
	DownlinkCapacity uint32 `json:"downlink_capacity,omitempty"`
	Congestion       bool   `json:"congestion,omitempty"`
	WriteBuffer      uint32 `json:"write_buffer,omitempty"`
	ReadBuffer       uint32 `json:"read_buffer,omitempty"`
	Seed             string `json:"seed,omitempty"`
	Header           string `json:"header,omitempty"`
}

type V2RayMEKYAOptions struct {
	URL                            string           `json:"url,omitempty"`
	H2PoolSize                     int              `json:"h2_pool_size,omitempty"`
	MaxWriteDelay                  int              `json:"max_write_delay,omitempty"`
	MaxRequestSize                 int              `json:"max_request_size,omitempty"`
	PollingIntervalInitial         int              `json:"polling_interval_initial,omitempty"`
	MaxWriteSize                   int              `json:"max_write_size,omitempty"`
	MaxWriteDurationMs             int              `json:"max_write_duration_ms,omitempty"`
	MaxSimultaneousWriteConnection int              `json:"max_simultaneous_write_connection,omitempty"`
	PacketWritingBuffer            int              `json:"packet_writing_buffer,omitempty"`
	KCP                            V2RayMKCPOptions `json:"kcp,omitempty"`
}

type V2RayXHTTPOptions struct {
	Path                 string                     `json:"path,omitempty"`
	Host                 string                     `json:"host,omitempty"`
	Mode                 string                     `json:"mode,omitempty"`
	Headers              badoption.HTTPHeader       `json:"headers,omitempty"`
	ALPN                 badoption.Listable[string] `json:"alpn,omitempty"`
	NoGRPCHeader         bool                       `json:"no_grpc_header,omitempty"`
	XPaddingBytes        string                     `json:"x_padding_bytes,omitempty"`
	XPaddingObfsMode     bool                       `json:"x_padding_obfs_mode,omitempty"`
	XPaddingKey          string                     `json:"x_padding_key,omitempty"`
	XPaddingHeader       string                     `json:"x_padding_header,omitempty"`
	XPaddingPlacement    string                     `json:"x_padding_placement,omitempty"`
	XPaddingMethod       string                     `json:"x_padding_method,omitempty"`
	UplinkHTTPMethod     string                     `json:"uplink_http_method,omitempty"`
	SessionPlacement     string                     `json:"session_placement,omitempty"`
	SessionKey           string                     `json:"session_key,omitempty"`
	SessionTable         string                     `json:"session_table,omitempty"`
	SessionLength        string                     `json:"session_length,omitempty"`
	SeqPlacement         string                     `json:"seq_placement,omitempty"`
	SeqKey               string                     `json:"seq_key,omitempty"`
	UplinkDataPlacement  string                     `json:"uplink_data_placement,omitempty"`
	UplinkDataKey        string                     `json:"uplink_data_key,omitempty"`
	UplinkChunkSize      string                     `json:"uplink_chunk_size,omitempty"`
	ScMaxEachPostBytes   string                     `json:"sc_max_each_post_bytes,omitempty"`
	ScMinPostsIntervalMs string                     `json:"sc_min_posts_interval_ms,omitempty"`
	ReuseSettings        *V2RayXHTTPReuseSettings   `json:"reuse_settings,omitempty"`           // aka xmux
	NoSSEHeader          bool                       `json:"no_sse_header,omitempty"`            // server only
	ScStreamUpServerSecs string                     `json:"sc_stream_up_server_secs,omitempty"` // server only
	ScMaxBufferedPosts   string                     `json:"sc_max_buffered_posts,omitempty"`    // server only
	DownloadSettings     *V2RayXHTTPDownloadOptions `json:"download_settings,omitempty"`
}

type V2RayXHTTPReuseSettings struct {
	MaxConcurrency   string `json:"max_concurrency,omitempty"`
	MaxConnections   string `json:"max_connections,omitempty"`
	CMaxReuseTimes   string `json:"c_max_reuse_times,omitempty"`
	HMaxRequestTimes string `json:"h_max_request_times,omitempty"`
	HMaxReusableSecs string `json:"h_max_reusable_secs,omitempty"`
	HKeepAlivePeriod int    `json:"h_keep_alive_period,omitempty"`
}

type V2RayXHTTPDownloadOptions struct {
	// xhttp part
	Path          string                   `json:"path,omitempty"`
	Host          string                   `json:"host,omitempty"`
	Headers       badoption.HTTPHeader     `json:"headers,omitempty"`
	ReuseSettings *V2RayXHTTPReuseSettings `json:"reuse_settings,omitempty"` // aka xmux
	// proxy part: transport and TLS options for the download connection
	Server     string                     `json:"server,omitempty"`
	ServerPort uint16                     `json:"server_port,omitempty"`
	TLS        *OutboundTLSOptions        `json:"tls,omitempty"`
	ALPN       badoption.Listable[string] `json:"alpn,omitempty"`
}
