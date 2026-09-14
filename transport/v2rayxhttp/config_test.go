package v2rayxhttp

import (
	"testing"

	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/json"
)

func TestResolveExtraOptions(t *testing.T) {
	base := option.V2RayXHTTPOptions{
		Path:               "/basepath",
		Host:               "base.example.com",
		Mode:               "packet-up",
		XPaddingBytes:      "100-200",
		NoGRPCHeader:       true,
		UplinkHTTPMethod:   "PUT",
		ScMaxEachPostBytes: "500000",
	}
	// extra overrides some fields, leaves others unset, and tries to
	// override host/path/mode which must always lose to the base.
	base.Extra = json.RawMessage(`{
		"host": "evil.example.com",
		"path": "/evilpath",
		"mode": "stream-one",
		"x_padding_bytes": "10-20",
		"uplink_http_method": "POST",
		"no_grpc_header": false,
		"sc_min_posts_interval_ms": "50",
		"unknown_future_field": {"anything": 1}
	}`)
	resolved, err := resolveExtraOptions(base)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Host != "base.example.com" {
		t.Error("host must win from base, got", resolved.Host)
	}
	if resolved.Path != "/basepath" {
		t.Error("path must win from base, got", resolved.Path)
	}
	if resolved.Mode != "packet-up" {
		t.Error("mode must win from base, got", resolved.Mode)
	}
	if resolved.XPaddingBytes != "10-20" {
		t.Error("extra should override x_padding_bytes, got", resolved.XPaddingBytes)
	}
	if resolved.UplinkHTTPMethod != "POST" {
		t.Error("extra should override uplink_http_method, got", resolved.UplinkHTTPMethod)
	}
	if resolved.ScMaxEachPostBytes != "500000" {
		t.Error("absent extra field must keep base, got", resolved.ScMaxEachPostBytes)
	}
	if resolved.ScMinPostsIntervalMs != "50" {
		t.Error("extra should set sc_min_posts_interval_ms, got", resolved.ScMinPostsIntervalMs)
	}
	// bools merge by zero value: an extra "false" cannot override a base true
	// (indistinguishable from unset), matching the non-zero override rule.
	if resolved.NoGRPCHeader != true {
		t.Error("extra false must not override base true, got", resolved.NoGRPCHeader)
	}
}

func TestResolveExtraInvalidJSON(t *testing.T) {
	base := option.V2RayXHTTPOptions{
		Extra: json.RawMessage(`{"x_padding_bytes": `),
	}
	_, err := resolveExtraOptions(base)
	if err == nil {
		t.Fatal("expected error for invalid extra JSON")
	}
}

func TestGetNormalizedServerMaxHeaderBytes(t *testing.T) {
	// default
	c := &Config{}
	v, err := c.GetNormalizedServerMaxHeaderBytes()
	if err != nil {
		t.Fatal(err)
	}
	if v != 8192 {
		t.Error("default should be 8192, got", v)
	}
	// positive value
	c = &Config{ServerMaxHeaderBytes: "16384"}
	v, err = c.GetNormalizedServerMaxHeaderBytes()
	if err != nil {
		t.Fatal(err)
	}
	if v != 16384 {
		t.Error("explicit value should be used, got", v)
	}
	// zero / negative -> default (Xray: <= 0 -> 8192)
	c = &Config{ServerMaxHeaderBytes: "0"}
	v, err = c.GetNormalizedServerMaxHeaderBytes()
	if err != nil {
		t.Fatal(err)
	}
	if v != 8192 {
		t.Error("0 should fall back to default, got", v)
	}
	// invalid
	c = &Config{ServerMaxHeaderBytes: "abc"}
	_, err = c.GetNormalizedServerMaxHeaderBytes()
	if err == nil {
		t.Error("expected error for invalid range")
	}
}

func TestEffectiveModeMatrix(t *testing.T) {
	// auto + reality + no download -> stream-one
	c := &Config{Mode: "auto"}
	if m := c.EffectiveMode(true); m != "stream-one" {
		t.Error("auto+reality -> stream-one, got", m)
	}
	// auto + reality + download -> stream-up
	c = &Config{Mode: "auto", DownloadConfig: &Config{}}
	if m := c.EffectiveMode(true); m != "stream-up" {
		t.Error("auto+reality+download -> stream-up, got", m)
	}
	// auto + no reality -> packet-up
	c = &Config{Mode: ""}
	if m := c.EffectiveMode(false); m != "packet-up" {
		t.Error("auto -> packet-up, got", m)
	}
	// explicit mode passes through
	c = &Config{Mode: "stream-one"}
	if m := c.EffectiveMode(false); m != "stream-one" {
		t.Error("explicit mode, got", m)
	}
}
