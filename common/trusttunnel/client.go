package trusttunnel

import (
	"context"
	cryptoTLS "crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/net/http2"

	"github.com/sagernet/sing-box/common/tls"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
)

type ClientOptions struct {
	Dialer                N.Dialer
	Server                string
	Username              string
	Password              string
	TLSConfig             tls.Config
	QUIC                  bool
	QUICCongestionControl string
	QUICCwnd              int
	QUICBBRProfile        string
	HealthCheck           bool
	MaxConnections        int
	MinStreams            int
	MaxStreams            int
}

type Client struct {
	ctx          context.Context
	dialer       N.Dialer
	server       string
	auth         string
	tlsConfig    tls.Config
	roundTripper http.RoundTripper
	startOnce    sync.Once
	healthCheck  bool
	healthTimer  *time.Timer
	count        atomic.Int64
}

func NewClient(ctx context.Context, options ClientOptions) (*Client, error) {
	client := &Client{
		ctx:       ctx,
		dialer:    options.Dialer,
		server:    options.Server,
		auth:      buildAuth(options.Username, options.Password),
		tlsConfig: options.TLSConfig,
	}
	if options.QUIC {
		nextProtos := options.TLSConfig.NextProtos()
		if len(nextProtos) == 0 {
			options.TLSConfig.SetNextProtos([]string{"h3"})
		} else if !containsString(nextProtos, "h3") {
			return nil, errors.New("require alpn h3")
		}
		err := client.quicRoundTripper(options.TLSConfig, options.QUICCongestionControl, options.QUICCwnd, options.QUICBBRProfile)
		if err != nil {
			return nil, err
		}
	} else {
		nextProtos := options.TLSConfig.NextProtos()
		if len(nextProtos) == 0 {
			options.TLSConfig.SetNextProtos([]string{"h2"})
		} else if !containsString(nextProtos, "h2") {
			return nil, errors.New("require alpn h2")
		}
		client.h2RoundTripper()
	}
	if options.HealthCheck {
		client.healthCheck = true
	}
	return client, nil
}

func containsString(array []string, target string) bool {
	return slices.Contains(array, target)
}

func (c *Client) h2RoundTripper() {
	// Use an explicit TLS handshake in DialTLSContext so the h2 connection is
	// established over our own dialer and TLS config (uTLS / ECH aware), and
	// net/http never falls back to HTTP/1.1.
	c.roundTripper = &http2.Transport{
		IdleConnTimeout: DefaultSessionTimeout,
		DialTLSContext: func(ctx context.Context, network, addr string, _ *cryptoTLS.Config) (net.Conn, error) {
			conn, err := c.dialer.DialContext(ctx, N.NetworkTCP, M.ParseSocksaddr(addr))
			if err != nil {
				return nil, err
			}
			tlsConn, err := tls.ClientHandshake(ctx, conn, c.tlsConfig)
			if err != nil {
				_ = conn.Close()
				return nil, err
			}
			return tlsConn, nil
		},
	}
}

func (c *Client) start() {
	if c.healthCheck {
		c.healthTimer = time.NewTimer(DefaultHealthCheckTimeout)
		go c.loopHealthCheck()
	}
}

func (c *Client) loopHealthCheck() {
	for {
		select {
		case <-c.healthTimer.C:
		case <-c.ctx.Done():
			c.healthTimer.Stop()
			return
		}
		ctx, cancel := context.WithTimeout(c.ctx, DefaultHealthCheckTimeout)
		_ = c.HealthCheck(ctx)
		cancel()
	}
}

func (c *Client) resetHealthCheckTimer() {
	if c.healthTimer == nil {
		return
	}
	c.healthTimer.Reset(DefaultHealthCheckTimeout)
}

// roundTrip issues a CONNECT request and returns as soon as the underlying
// connection is established, without waiting for the HTTP response: the
// returned conn reads block until the response headers arrive.
func (c *Client) roundTrip(ctx context.Context, request *http.Request, conn *httpConn) error {
	c.startOnce.Do(c.start)
	pipeReader, pipeWriter := io.Pipe()
	request.Body = pipeReader
	*conn = httpConn{
		writer:  pipeWriter,
		created: make(chan struct{}),
	}
	c.count.Add(1)
	conn.closeFn = sync.OnceFunc(func() {
		c.count.Add(-1)
	})
	requestCtx, cancel := context.WithCancel(c.ctx) // requestCtx must alive during conn not closed
	conn.cancelFn = cancel                          // cancel ctx when conn closed

	// Use gotConn to detect when the underlying connection is established, so
	// we can return the conn immediately without waiting for the HTTP response.
	gotConn := make(chan bool, 1)
	var requestErr atomic.Value
	streamCtx := httptrace.WithClientTrace(requestCtx, &httptrace.ClientTrace{
		GotConn: func(info httptrace.GotConnInfo) {
			conn.remoteAddr = info.Conn.RemoteAddr()
			conn.localAddr = info.Conn.LocalAddr()
			select {
			case gotConn <- true:
			default: // GotConn maybe called multiple times, ignore the second and later calls
			}
		},
	})

	go func() {
		request = request.WithContext(streamCtx)
		response, err := c.roundTripper.RoundTrip(request)
		if err != nil {
			requestErr.Store(err)
			close(gotConn)
			_ = pipeWriter.CloseWithError(err)
			_ = pipeReader.CloseWithError(err)
			conn.setup(nil, err)
		} else if response.StatusCode != http.StatusOK {
			_ = response.Body.Close()
			err = fmt.Errorf("unexpected status code: %d", response.StatusCode)
			requestErr.Store(err)
			_ = pipeWriter.CloseWithError(err)
			_ = pipeReader.CloseWithError(err)
			conn.setup(nil, err)
		} else {
			c.resetHealthCheckTimer()
			conn.setup(response.Body, nil)
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-gotConn:
		if err, ok := requestErr.Load().(error); ok {
			return err
		}
		return nil
	}
}

func (c *Client) newConnectRequest(host, userAgent string) *http.Request {
	request := &http.Request{
		Method: http.MethodConnect,
		URL: &url.URL{
			Scheme: "https",
			Host:   c.server, // Use the proxy server authority so the pool keys reuse against the actual proxy endpoint.
		},
		Header: make(http.Header),
		Host:   host, // Send the actual CONNECT target as the Host header (:authority).
	}
	request.Header.Add("User-Agent", userAgent)
	request.Header.Add("Proxy-Authorization", c.auth)
	return request
}

func (c *Client) Dial(ctx context.Context, host string) (net.Conn, error) {
	request := c.newConnectRequest(host, TCPUserAgent)
	conn := &tcpConn{}
	err := c.roundTrip(ctx, request, &conn.httpConn)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	return conn, nil
}

func (c *Client) ListenPacket(ctx context.Context) (*clientPacketConn, error) {
	request := c.newConnectRequest(UDPMagicAddress, UDPUserAgent)
	conn := &clientPacketConn{}
	err := c.roundTrip(ctx, request, &conn.httpConn)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	return conn, nil
}

func (c *Client) Close() error {
	if closer, ok := c.roundTripper.(io.Closer); ok {
		_ = closer.Close()
	}
	if closeIdle, ok := c.roundTripper.(interface{ CloseIdleConnections() }); ok {
		closeIdle.CloseIdleConnections()
	}
	if c.healthTimer != nil {
		c.healthTimer.Stop()
	}
	return nil
}

func (c *Client) ResetConnections() {
	if closeIdle, ok := c.roundTripper.(interface{ CloseIdleConnections() }); ok {
		closeIdle.CloseIdleConnections()
	}
	c.resetHealthCheckTimer()
}

func (c *Client) HealthCheck(ctx context.Context) error {
	defer c.resetHealthCheckTimer()
	request := c.newConnectRequest(HealthCheckMagicAddress, HealthCheckUserAgent)
	response, err := c.roundTripper.RoundTrip(request.WithContext(ctx))
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}
	return nil
}

type PoolClient struct {
	mutex          sync.Mutex
	maxConnections int
	minStreams     int
	maxStreams     int
	ctx            context.Context
	options        ClientOptions
	clients        []*Client
}

func NewPoolClient(ctx context.Context, options ClientOptions) (*PoolClient, error) {
	maxConnections := options.MaxConnections
	minStreams := options.MinStreams
	maxStreams := options.MaxStreams
	if maxConnections == 0 && minStreams == 0 && maxStreams == 0 {
		maxConnections = 8
		minStreams = 5
	}
	client, err := NewClient(ctx, options) // reserve one client and verify the configuration
	if err != nil {
		return nil, err
	}
	return &PoolClient{
		maxConnections: maxConnections,
		minStreams:     minStreams,
		maxStreams:     maxStreams,
		ctx:            ctx,
		options:        options,
		clients:        []*Client{client},
	}, nil
}

func (c *PoolClient) ResetConnections() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	for _, client := range c.clients {
		client.ResetConnections()
	}
}

func (c *PoolClient) Dial(ctx context.Context, host string) (net.Conn, error) {
	transport, err := c.getClient()
	if err != nil {
		return nil, err
	}
	return transport.Dial(ctx, host)
}

func (c *PoolClient) ListenPacket(ctx context.Context) (*clientPacketConn, error) {
	transport, err := c.getClient()
	if err != nil {
		return nil, err
	}
	return transport.ListenPacket(ctx)
}

func (c *PoolClient) Close() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	var errs []error
	for _, t := range c.clients {
		if err := t.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	c.clients = nil
	return errors.Join(errs...)
}

func (c *PoolClient) getClient() (*Client, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	var transport *Client
	for _, t := range c.clients {
		if transport == nil || t.count.Load() < transport.count.Load() {
			transport = t
		}
	}
	if transport == nil {
		return c.newTransportLocked()
	}
	numStreams := int(transport.count.Load())
	if numStreams == 0 {
		return transport, nil
	}
	if c.maxConnections > 0 {
		if len(c.clients) >= c.maxConnections || numStreams < c.minStreams {
			return transport, nil
		}
	} else {
		if c.maxStreams > 0 && numStreams < c.maxStreams {
			return transport, nil
		}
	}
	return c.newTransportLocked()
}

func (c *PoolClient) newTransportLocked() (*Client, error) {
	transport, err := NewClient(c.ctx, c.options)
	if err != nil {
		return nil, err
	}
	c.clients = append(c.clients, transport)
	return transport, nil
}
