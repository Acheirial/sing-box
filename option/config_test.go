package option

import (
	"context"
	"strings"
	"testing"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing/service"

	"github.com/stretchr/testify/require"
)

type stubInboundOptionsRegistry struct{}

func (stubInboundOptionsRegistry) OptionTypes() []string {
	return []string{C.TypeDirect}
}

func (stubInboundOptionsRegistry) CreateOptions(inboundType string) (any, bool) {
	if inboundType == C.TypeDirect {
		return new(DirectInboundOptions), true
	}
	return nil, false
}

type stubOutboundOptionsRegistry struct{}

func (stubOutboundOptionsRegistry) OptionTypes() []string {
	return []string{C.TypeDirect}
}

func (stubOutboundOptionsRegistry) CreateOptions(outboundType string) (any, bool) {
	if outboundType == C.TypeDirect {
		return new(DirectOutboundOptions), true
	}
	return nil, false
}

func TestDetectConfigFormat(t *testing.T) {
	cases := []struct {
		name     string
		path     string
		content  string
		expected ConfigFormat
	}{
		{"json ext", "config.json", "", ConfigFormatJSON},
		{"yaml ext", "config.yaml", "", ConfigFormatYAML},
		{"yml ext", "config.yml", "", ConfigFormatYAML},
		{"uppercase yaml", "CONFIG.YAML", "", ConfigFormatYAML},
		{"stdin json", "stdin", "  {\"log\": {}}", ConfigFormatJSON},
		{"stdin yaml", "stdin", "log:\n  level: info\n", ConfigFormatYAML},
		{"stdin empty", "stdin", "", ConfigFormatJSON},
		{"stdin bom json", "stdin", "\xef\xbb\xbf{\"log\": {}}", ConfigFormatJSON},
		{"stdin comment json", "stdin", "// comment\n{\"log\": {}}", ConfigFormatJSON},
		{"no ext json", "/etc/sing-box/config", "{\"log\": {}}", ConfigFormatJSON},
		{"no ext yaml", "/etc/sing-box/config", "log:\n  disabled: false\n", ConfigFormatYAML},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, DetectConfigFormat(tc.path, []byte(tc.content)))
		})
	}
}

func TestIsConfigFile(t *testing.T) {
	require.True(t, IsConfigFile("a.json"))
	require.True(t, IsConfigFile("a.yaml"))
	require.True(t, IsConfigFile("a.yml"))
	require.True(t, IsConfigFile("a.YML"))
	require.False(t, IsConfigFile("a.txt"))
	require.False(t, IsConfigFile("a.srs"))
	require.False(t, IsConfigFile("json"))
}

func TestYAMLToJSONScalars(t *testing.T) {
	cases := []struct {
		name     string
		yaml     string
		expected string
	}{
		{
			"primitives",
			"str: hello\nnum: 1080\nhex: 0x10\noct: 0o10\nbin: 0b1010\nbool_true: true\nbool_false: false\nnull_val: null\n",
			`{"str":"hello","num":1080,"hex":16,"oct":8,"bin":10,"bool_true":true,"bool_false":false,"null_val":null}`,
		},
		{
			"numeric string preserved",
			"port: \"1080\"\nbool_str: \"true\"\nnull_str: \"null\"\n",
			`{"port":"1080","bool_str":"true","null_str":"null"}`,
		},
		{
			"literal timestamp as string",
			"date: 2026-09-25\ntime: 2026-09-25T12:00:00Z\n",
			`{"date":"2026-09-25","time":"2026-09-25T12:00:00Z"}`,
		},
		{
			"empty document",
			"",
			`{}`,
		},
		{
			"null document",
			"null\n",
			`null`,
		},
		{
			"anchors and aliases",
			"base: &base\n  a: 1\nref: *base\n",
			`{"base":{"a":1},"ref":{"a":1}}`,
		},
		{
			"merge keys",
			"base: &base\n  shared: 1\n  override: old\nderived:\n  <<: *base\n  override: new\n  extra: 2\n",
			`{"base":{"shared":1,"override":"old"},"derived":{"override":"new","extra":2,"shared":1}}`,
		},
		{
			"merge sequence of maps",
			"m1: &m1\n  a: 1\nm2: &m2\n  b: 2\nmerged:\n  <<: [*m1, *m2]\n  c: 3\n",
			`{"m1":{"a":1},"m2":{"b":2},"merged":{"c":3,"a":1,"b":2}}`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			jsonBytes, err := yamlToJSON([]byte(tc.yaml))
			require.NoError(t, err)
			require.JSONEq(t, tc.expected, string(jsonBytes))
		})
	}
}

func TestYAMLToJSONErrors(t *testing.T) {
	// Duplicate keys must be rejected.
	_, err := yamlToJSON([]byte("a: 1\na: 2\n"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "duplicate mapping key")

	// Multiple documents must be rejected.
	_, err = yamlToJSON([]byte("a: 1\n---\nb: 2\n"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "multiple YAML documents")
}

func TestUnmarshalConfigYAML(t *testing.T) {
	yamlConfig := `
log:
  disabled: false
  level: trace
  timestamp: true

endpoints: []

inbounds:
  - type: direct
    tag: direct-in
    listen: "::"
    listen_port: 1080

outbounds:
  - type: direct
    tag: direct-out
`
	ctx := context.Background()
	ctx = service.ContextWith[InboundOptionsRegistry](ctx, stubInboundOptionsRegistry{})
	ctx = service.ContextWith[OutboundOptionsRegistry](ctx, stubOutboundOptionsRegistry{})
	options, err := UnmarshalConfig(ctx, []byte(yamlConfig), ConfigFormatYAML)
	require.NoError(t, err)
	require.NotNil(t, options.Log)
	require.Equal(t, "trace", options.Log.Level)
	require.True(t, options.Log.Timestamp)
	require.Len(t, options.Inbounds, 1)
	require.Equal(t, C.TypeDirect, options.Inbounds[0].Type)
	require.Equal(t, "direct-in", options.Inbounds[0].Tag)
	require.Len(t, options.Outbounds, 1)
	require.Equal(t, C.TypeDirect, options.Outbounds[0].Type)
	require.NotEmpty(t, options.RawMessage)
	require.True(t, strings.HasPrefix(strings.TrimSpace(string(options.RawMessage)), "{"))
}

func TestUnmarshalConfigYAMLUnknownField(t *testing.T) {
	yamlConfig := `
log:
  level: info
unknown_field_xyz: true
`
	_, err := UnmarshalConfig(context.Background(), []byte(yamlConfig), ConfigFormatYAML)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown field")
}

func TestJSONToYAMLRoundTrip(t *testing.T) {
	jsonInput := `{"big":9007199254740993,"bool_str":"true","flag":true,"list":[1,2,3],"name":"sing-box","nested":{"inner":"val"},"port":1080}`
	yamlBytes, err := JSONToYAML([]byte(jsonInput))
	require.NoError(t, err)

	// Reparse YAML to JSON and ensure values are preserved without loss.
	jsonRoundTrip, err := yamlToJSON(yamlBytes)
	require.NoError(t, err)
	require.JSONEq(t, jsonInput, string(jsonRoundTrip))
}
