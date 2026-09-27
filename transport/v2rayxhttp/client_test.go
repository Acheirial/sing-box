package v2rayxhttp

import (
	"context"
	"testing"

	"github.com/sagernet/sing-box/common/dialer"
	"github.com/sagernet/sing-box/option"
	M "github.com/sagernet/sing/common/metadata"

	"github.com/stretchr/testify/require"
)

func TestNewClientDownloadSettingsModes(t *testing.T) {
	for _, mode := range []string{"stream-one", "stream-up", "packet-up"} {
		t.Run(mode, func(t *testing.T) {
			options := option.V2RayXHTTPOptions{
				Path: "/xhttp",
				Mode: mode,
				DownloadSettings: &option.V2RayXHTTPDownloadSettings{
					ServerOptions:     option.ServerOptions{Server: "download.example", ServerPort: 443},
					V2RayXHTTPOptions: option.V2RayXHTTPOptions{Path: "/download", Mode: "packet-up"},
				},
			}
			clientDialer, err := dialer.NewDefault(context.Background(), option.DialerOptions{})
			require.NoError(t, err)
			client, err := NewClient(context.Background(), clientDialer, M.ParseSocksaddr("127.0.0.1:8080"), options, nil)
			require.NoError(t, err)
			defer client.Close()
			if mode == "stream-one" {
				// download_settings are ignored: the download stream shares the
				// bidirectional upload request.
				require.Nil(t, client.download)
			} else {
				require.NotNil(t, client.download)
			}
		})
	}
}
