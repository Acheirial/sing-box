package gost

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sagernet/sing-box/common/tls"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing/common/buf"
	"github.com/sagernet/sing/common/bufio"
	"github.com/sagernet/sing/common/bufio/deadline"
	E "github.com/sagernet/sing/common/exceptions"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"

	"github.com/sagernet/ws"
)

// WebsocketOption is options of gost websocket transport.
type WebsocketOption struct {
	Host      string
	Path      string
	Headers   http.Header
	TLS       bool
	TLSConfig tls.Config
	Mux       bool
}

// DialWebsocket returns a gost websocket connection to the destination
// through the given dialer.
func DialWebsocket(ctx context.Context, dialer N.Dialer, serverAddress M.Socksaddr, option *WebsocketOption) (net.Conn, error) {
	underlying := dialer
	if option.TLS {
		if len(option.TLSConfig.NextProtos()) == 0 {
			option.TLSConfig.SetNextProtos([]string{"http/1.1"})
		}
		underlying = tls.NewDialer(dialer, option.TLSConfig)
	}
	conn, err := underlying.DialContext(ctx, N.NetworkTCP, serverAddress)
	if err != nil {
		return nil, err
	}
	wsConn, err := dialWebsocket(ctx, conn, serverAddress, option)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	if option.Mux {
		return muxSession(wsConn)
	}
	return wsConn, nil
}

func dialWebsocket(ctx context.Context, conn net.Conn, serverAddress M.Socksaddr, option *WebsocketOption) (net.Conn, error) {
	requestURL := url.URL{
		Scheme: "ws",
		Host:   serverAddress.String(),
		Path:   option.Path,
	}
	if !strings.HasPrefix(requestURL.Path, "/") {
		requestURL.Path = "/" + requestURL.Path
	}
	if option.TLS {
		requestURL.Scheme = "wss"
	}
	headers := option.Headers.Clone()
	if headers == nil {
		headers = http.Header{}
	}
	if host := headers.Get("Host"); host != "" {
		headers.Del("Host")
		requestURL.Host = host
	} else if option.Host != "" {
		requestURL.Host = option.Host
	}
	if headers.Get("User-Agent") == "" {
		headers.Set("User-Agent", "Go-http-client/1.1")
	}

	var deadlineConn net.Conn
	if deadline.NeedAdditionalReadDeadline(conn) {
		deadlineConn = deadline.NewConn(conn)
	} else {
		deadlineConn = conn
	}
	deadlineConn.SetDeadline(time.Now().Add(C.TCPTimeout))
	reader, _, err := ws.Dialer{Header: ws.HandshakeHeaderHTTP(headers)}.Upgrade(deadlineConn, &requestURL)
	deadlineConn.SetDeadline(time.Time{})
	if err != nil {
		return nil, E.Cause(err, "gost websocket handshake")
	}
	if reader != nil {
		buffer := buf.NewSize(reader.Buffered())
		_, err = buffer.ReadFullFrom(reader, buffer.Len())
		if err != nil {
			return nil, err
		}
		conn = bufio.NewCachedConn(conn, buffer)
	}
	return NewWebsocketConn(conn, nil, ws.StateClientSide), nil
}
