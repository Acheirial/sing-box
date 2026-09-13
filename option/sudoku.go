package option

type SudokuHTTPOptions struct {
	Disable   bool   `json:"disable,omitempty"`
	Mode      string `json:"mode,omitempty"`
	TLS       bool   `json:"tls,omitempty"`
	Host      string `json:"host,omitempty"`
	PathRoot  string `json:"path_root,omitempty"`
	Multiplex string `json:"multiplex,omitempty"`
}

type SudokuOutboundOptions struct {
	DialerOptions
	ServerOptions
	Key                string             `json:"key"`
	AEADMethod         string             `json:"aead_method,omitempty"`
	PaddingMin         *int               `json:"padding_min,omitempty"`
	PaddingMax         *int               `json:"padding_max,omitempty"`
	TableType          string             `json:"table_type,omitempty"`
	EnablePureDownlink *bool              `json:"enable_pure_downlink,omitempty"`
	HTTPMask           *bool              `json:"http_mask,omitempty"`
	HTTPMaskMode       string             `json:"http_mask_mode,omitempty"`
	HTTPMaskTLS        bool               `json:"http_mask_tls,omitempty"`
	HTTPMaskHost       string             `json:"http_mask_host,omitempty"`
	PathRoot           string             `json:"path_root,omitempty"`
	Multiplex          string             `json:"multiplex,omitempty"`
	HTTPMaskMultiplex  string             `json:"http_mask_multiplex,omitempty"`
	HTTPMaskOptions    *SudokuHTTPOptions `json:"httpmask,omitempty"`
	CustomTable        string             `json:"custom_table,omitempty"`
	CustomTables       []string           `json:"custom_tables,omitempty"`
}
