//go:build with_utls

package tls

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"

	utls "github.com/metacubex/utls"
	"github.com/stretchr/testify/require"
)

func newRealityTestUTLSClient() *UTLSClientConfig {
	return &UTLSClientConfig{
		ctx:        context.Background(),
		config:     &utls.Config{ServerName: "example.com", InsecureSkipVerify: true},
		serverName: "example.com",
		id:         utls.HelloChrome_Auto,
	}
}

func newRealityTestPublicKey() []byte {
	publicKey := make([]byte, 32)
	for i := range publicKey {
		publicKey[i] = byte(i + 1)
	}
	return publicKey
}

func newRealityTestPrivateKey() string {
	return base64.RawURLEncoding.EncodeToString(make([]byte, 32))
}

// TestRealityClientHelloVersion guards the version byte triple sent in the
// REALITY session ID: it must be derived from the build version instead of the
// legacy hardcoded 1.8.1.
func TestRealityClientHelloVersion(t *testing.T) {
	previousVersion := C.Version
	C.Version = "2.5.9"
	t.Cleanup(func() {
		C.Version = previousVersion
	})

	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	config := &RealityClientConfig{
		ctx:       context.Background(),
		uClient:   newRealityTestUTLSClient(),
		publicKey: newRealityTestPublicKey(),
		shortID:   [8]byte{1, 2, 3, 4, 5, 6, 7, 8},
	}
	uConn, verifier, err := config.buildClientHello(client)
	require.NoError(t, err)
	require.NotNil(t, verifier)

	hello := uConn.HandshakeState.Hello
	require.Len(t, hello.SessionId, 32)
	require.Equal(t, []byte{2, 5, 9}, hello.SessionId[:3])
	require.Equal(t, byte(0), hello.SessionId[3])
	require.Equal(t, config.shortID[:], hello.SessionId[8:16])
	require.NotEqual(t, []byte{1, 8, 1}, hello.SessionId[:3], "legacy hardcoded client version must be gone")
}

func TestParseRealitySpiderX(t *testing.T) {
	t.Parallel()

	path, spiderY, err := parseRealitySpiderX("/?p=10-20&c=2-3&t=4&i=5-6&r=7")
	require.NoError(t, err)
	require.Equal(t, "/", path)
	require.Equal(t, realitySpiderY{10, 20, 2, 3, 4, 4, 5, 6, 7, 7}, spiderY)

	path, _, err = parseRealitySpiderX("")
	require.NoError(t, err)
	require.Equal(t, "/", path)

	_, _, err = parseRealitySpiderX("relative/path")
	require.ErrorContains(t, err, "spider_x")
}

// TestRealitySpiderCrawl verifies that a failed verification crawls the
// fallback server with at least two requests, chaining the Referer header.
func TestRealitySpiderCrawl(t *testing.T) {
	t.Parallel()

	spiderX, spiderY, err := parseRealitySpiderX("/")
	require.NoError(t, err)
	spider := newRealitySpider("example.com", spiderX, "Mozilla/5.0 (Test)", spiderY)

	var requests []*http.Request
	do := func(request *http.Request) (*http.Response, error) {
		requests = append(requests, request)
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`<a href="/next">next</a>`)),
		}, nil
	}
	spider.crawl(do)

	require.GreaterOrEqual(t, len(requests), 2, "spider must issue at least two requests")
	require.Empty(t, requests[0].Header.Get("Referer"))
	for i := range requests {
		request := requests[i]
		require.Equal(t, "https", request.URL.Scheme)
		require.Equal(t, "example.com", request.URL.Host)
		require.Equal(t, "Mozilla/5.0 (Test)", request.Header.Get("User-Agent"))
		require.NotEmpty(t, request.Cookies())
		if i == 0 {
			continue
		}
		require.Equal(t, requests[i-1].URL.String(), request.Header.Get("Referer"))
	}
}

func TestRealityOptionsJSONRoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value any
	}{
		{
			name: "inbound",
			value: &option.InboundTLSOptions{
				Enabled:    true,
				ServerName: "example.com",
				Reality: &option.InboundRealityOptions{
					Enabled:               true,
					PrivateKey:            newRealityTestPrivateKey(),
					ShortID:               []string{"0123456789abcdef"},
					ServerNames:           []string{"www.example.com", "example.org"},
					MinClientVer:          "1.2.3",
					MaxClientVer:          "2.3.4",
					Xver:                  2,
					Show:                  true,
					MasterKeyLog:          "/tmp/reality-master-key.log",
					LimitFallbackUpload:   &option.InboundRealityLimitFallbackOptions{AfterBytes: 100, BytesPerSec: 200, BurstBytesPerSec: 300},
					LimitFallbackDownload: &option.InboundRealityLimitFallbackOptions{AfterBytes: 400, BytesPerSec: 500, BurstBytesPerSec: 600},
				},
			},
		},
		{
			name: "outbound",
			value: &option.OutboundTLSOptions{
				Enabled:    true,
				ServerName: "example.com",
				Reality: &option.OutboundRealityOptions{
					Enabled:   true,
					PublicKey: newRealityTestPrivateKey(),
					ShortID:   "0123",
					SpiderX:   "/?c=1-2",
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			content, err := json.Marshal(test.value)
			require.NoError(t, err)
			decoded := reflect.New(reflect.TypeOf(test.value).Elem()).Interface()
			err = json.Unmarshal(content, decoded)
			require.NoError(t, err)
			require.Equal(t, test.value, decoded)
		})
	}
}

// TestRealityServerOptionsApplied verifies that the server options are mapped
// onto the utls configuration used by the handshake.
func TestRealityServerOptionsApplied(t *testing.T) {
	t.Parallel()

	serverConfig, err := NewRealityServer(context.Background(), nil, option.InboundTLSOptions{
		Enabled:    true,
		ServerName: "example.com",
		Reality: &option.InboundRealityOptions{
			Enabled:               true,
			PrivateKey:            newRealityTestPrivateKey(),
			ShortID:               []string{"0123456789abcdef"},
			ServerNames:           []string{"www.example.com", "example.org"},
			MinClientVer:          "1.2.3",
			MaxClientVer:          "2.3.4",
			Xver:                  2,
			LimitFallbackUpload:   &option.InboundRealityLimitFallbackOptions{AfterBytes: 100, BytesPerSec: 200, BurstBytesPerSec: 300},
			LimitFallbackDownload: &option.InboundRealityLimitFallbackOptions{AfterBytes: 400, BytesPerSec: 500, BurstBytesPerSec: 600},
		},
	})
	require.NoError(t, err)
	realityConfig := serverConfig.(*RealityServerConfig)
	require.Equal(t, map[string]bool{"www.example.com": true, "example.org": true}, realityConfig.config.ServerNames)
	require.Equal(t, []byte{1, 2, 3}, realityConfig.config.MinClientVer)
	require.Equal(t, []byte{2, 3, 4}, realityConfig.config.MaxClientVer)
	require.Equal(t, byte(2), realityConfig.config.Xver)
	require.Equal(t, utls.RealityLimitFallback{AfterBytes: 100, BytesPerSec: 200, BurstBytesPerSec: 300}, realityConfig.config.LimitFallbackUpload)
	require.Equal(t, utls.RealityLimitFallback{AfterBytes: 400, BytesPerSec: 500, BurstBytesPerSec: 600}, realityConfig.config.LimitFallbackDownload)
}

// TestRealityServerDefaultServerNames keeps the previous behaviour when
// server_names is not configured.
func TestRealityServerDefaultServerNames(t *testing.T) {
	t.Parallel()

	serverConfig, err := NewRealityServer(context.Background(), nil, option.InboundTLSOptions{
		Enabled:    true,
		ServerName: "example.com",
		Reality: &option.InboundRealityOptions{
			Enabled:    true,
			PrivateKey: newRealityTestPrivateKey(),
			ShortID:    []string{"0123456789abcdef"},
		},
	})
	require.NoError(t, err)
	require.Equal(t, map[string]bool{"example.com": true}, serverConfig.(*RealityServerConfig).config.ServerNames)
}

func TestRealityServerMasterKeyLog(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "reality-master-key.log")
	serverConfig, err := NewRealityServer(context.Background(), nil, option.InboundTLSOptions{
		Enabled:    true,
		ServerName: "example.com",
		Reality: &option.InboundRealityOptions{
			Enabled:      true,
			PrivateKey:   newRealityTestPrivateKey(),
			ShortID:      []string{"0123456789abcdef"},
			MasterKeyLog: path,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, serverConfig.(*RealityServerConfig).config.KeyLogWriter)
	require.NoError(t, serverConfig.Close())
	_, err = os.Stat(path)
	require.NoError(t, err)
}

func TestRealityServerOptionsValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		reality option.InboundRealityOptions
		err     string
	}{
		{
			name:    "xver",
			reality: option.InboundRealityOptions{Enabled: true, PrivateKey: newRealityTestPrivateKey(), Xver: 3},
			err:     "xver",
		},
		{
			name:    "min_client_ver",
			reality: option.InboundRealityOptions{Enabled: true, PrivateKey: newRealityTestPrivateKey(), MinClientVer: "1.2.3.4"},
			err:     "min_client_ver",
		},
		{
			name:    "max_client_ver",
			reality: option.InboundRealityOptions{Enabled: true, PrivateKey: newRealityTestPrivateKey(), MaxClientVer: "a.b.c"},
			err:     "max_client_ver",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := NewRealityServer(context.Background(), nil, option.InboundTLSOptions{
				Enabled:    true,
				ServerName: "example.com",
				Reality:    &test.reality,
			})
			require.ErrorContains(t, err, test.err)
		})
	}
}
