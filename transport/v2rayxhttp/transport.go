package v2rayxhttp

import (
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/sagernet/quic-go"
	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/common/tls"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing-quic"
	"github.com/sagernet/sing/common"
	E "github.com/sagernet/sing/common/exceptions"
	"github.com/sagernet/sing/common/logger"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
	aTLS "github.com/sagernet/sing/common/tls"

	"github.com/sagernet/sing/common/baderror"
)

var (
	_ adapter.V2RayClientTransport = (*XHTTPClientTransport)(nil)
	_ adapter.V2RayServerTransport = (*Server)(nil)
)

const V2RayTransportTypeXHTTP = "xhttp"

type XHTTPClientTransport struct {
	ctx         context.Context
	dialer      N.Dialer
	serverAddr  M.Socksaddr
	tlsConfig   tls.Config
	hasReality  bool
	xhttpClient *Client
	requestHost string
	alpn        []string
	closeOnce   sync.Once
}

func NewClient(ctx context.Context, dialer N.Dialer, serverAddr M.Socksaddr, options option.V2RayXHTTPOptions, tlsConfig tls.Config) (adapter.V2RayClientTransport, error) {
	return newXHTTPClient(ctx, dialer, serverAddr, options, tlsConfig)
}

func newXHTTPClient(ctx context.Context, dialer N.Dialer, serverAddr M.Socksaddr, options option.V2RayXHTTPOptions, tlsConfig tls.Config) (adapter.V2RayClientTransport, error) {
	options, err := resolveExtraOptions(options)
	if err != nil {
		return nil, err
	}
	requestHost := options.Host
	if requestHost == "" {
		if tlsConfig != nil && tlsConfig.ServerName() != "" {
			requestHost = tlsConfig.ServerName()
		} else {
			requestHost = serverAddr.AddrString()
		}
	}

	var hKeepAlivePeriod time.Duration
	if options.ReuseSettings != nil {
		hKeepAlivePeriod = time.Duration(options.ReuseSettings.HKeepAlivePeriod) * time.Second
	}

	cfg, err := newConfig(options)
	if err != nil {
		return nil, err
	}

	hasReality := isRealityConfig(tlsConfig)

	alpn := options.ALPN
	if tlsConfig != nil && len(alpn) == 0 {
		alpn = tlsConfig.NextProtos()
	}
	alpn = filterALPN(alpn)

	makeTransport := func() http.RoundTripper {
		return NewTransport(
			func(ctx context.Context) (net.Conn, error) {
				return dialer.DialContext(ctx, N.NetworkTCP, serverAddr)
			},
			func(ctx context.Context, raw net.Conn, isH2 bool) (net.Conn, error) {
				if tlsConfig == nil {
					return raw, nil
				}
				if isH2 {
					nextProtos := tlsConfig.NextProtos()
					if !common.Contains(nextProtos, "h2") {
						tlsConfig.SetNextProtos(append([]string{"h2"}, nextProtos...))
					}
				} else {
					tlsConfig.SetNextProtos([]string{"http/1.1"})
				}
				return tls.ClientHandshake(ctx, raw, tlsConfig)
			},
			func(ctx context.Context, cfg *quic.Config) (*quic.Conn, error) {
				if tlsConfig == nil {
					return nil, E.New("xhttp HTTP/3 requires TLS")
				}
				if len(alpn) > 0 && alpn[0] != "h3" {
					tlsConfig.SetNextProtos([]string{"h3"})
				} else {
					tlsConfig.SetNextProtos([]string{"h3"})
				}
				udpConn, err := dialer.DialContext(ctx, N.NetworkUDP, serverAddr)
				if err != nil {
					return nil, err
				}
				quicConn, err := qtls.DialEarly(ctx, udpConn, tlsConfig, cfg)
				if err != nil {
					udpConn.Close()
					return nil, err
				}
				go func() {
					<-quicConn.Context().Done()
					udpConn.Close()
				}()
				return quicConn, nil
			},
			alpn,
			hKeepAlivePeriod,
		)
	}

	var makeDownloadTransport func() http.RoundTripper
	if ds := options.DownloadSettings; ds != nil {
		if cfg.Mode == "stream-one" {
			return nil, E.New(`xhttp mode "stream-one" cannot be used with download-settings`)
		}

		downloadServerAddr := serverAddr
		if ds.Server != "" || ds.ServerPort != 0 {
			downloadServerAddr = M.ParseSocksaddrHostPort(ds.Server, uint16(ds.ServerPort))
			if ds.Server == "" {
				downloadServerAddr = M.SocksaddrFrom(serverAddr.Addr, ds.ServerPort)
			}
		}

		downloadTLSConfig := tlsConfig
		if ds.TLS != nil {
			if !ds.TLS.Enabled {
				downloadTLSConfig = nil
			}
		}
		if downloadTLSConfig != nil && ds.TLS != nil {
			cloned := downloadTLSConfig.Clone()
			if ds.TLS.ServerName != "" {
				cloned.SetServerName(ds.TLS.ServerName)
			}
			if len(ds.ALPN) > 0 {
				cloned.SetNextProtos(ds.ALPN)
			}
			downloadTLSConfig = cloned
		}

		downloadALPN := ds.ALPN
		if downloadTLSConfig != nil && len(downloadALPN) == 0 {
			downloadALPN = filterALPN(downloadTLSConfig.NextProtos())
		}

		downloadHost := ds.Host
		if downloadHost == "" {
			if downloadTLSConfig != nil && downloadTLSConfig.ServerName() != "" {
				downloadHost = downloadTLSConfig.ServerName()
			} else {
				downloadHost = downloadServerAddr.AddrString()
			}
		}

		var downloadHKeepAlivePeriod = hKeepAlivePeriod

		downloadCfg := *cfg // make a copy
		downloadCfg.Host = downloadHost
		downloadCfg.Path = ds.Path
		if ds.Path == "" {
			downloadCfg.Path = cfg.Path
		}
		if ds.Headers != nil {
			downloadCfg.Headers = ds.Headers.Build()
		}

		if ds.ReuseSettings != nil {
			downloadCfg.ReuseConfig = &ReuseConfig{
				MaxConcurrency:   ds.ReuseSettings.MaxConcurrency,
				MaxConnections:   ds.ReuseSettings.MaxConnections,
				CMaxReuseTimes:   ds.ReuseSettings.CMaxReuseTimes,
				HMaxRequestTimes: ds.ReuseSettings.HMaxRequestTimes,
				HMaxReusableSecs: ds.ReuseSettings.HMaxReusableSecs,
			}
			downloadHKeepAlivePeriod = time.Duration(ds.ReuseSettings.HKeepAlivePeriod) * time.Second
		}

		cfg.DownloadConfig = &downloadCfg

		makeDownloadTransport = func() http.RoundTripper {
			return NewTransport(
				func(ctx context.Context) (net.Conn, error) {
					return dialer.DialContext(ctx, N.NetworkTCP, downloadServerAddr)
				},
				func(ctx context.Context, raw net.Conn, isH2 bool) (net.Conn, error) {
					if downloadTLSConfig == nil {
						return raw, nil
					}
					if isH2 {
						nextProtos := downloadTLSConfig.NextProtos()
						if !common.Contains(nextProtos, "h2") {
							downloadTLSConfig.SetNextProtos(append([]string{"h2"}, nextProtos...))
						}
					} else {
						downloadTLSConfig.SetNextProtos([]string{"http/1.1"})
					}
					return tls.ClientHandshake(ctx, raw, downloadTLSConfig)
				},
				func(ctx context.Context, cfg *quic.Config) (*quic.Conn, error) {
					if downloadTLSConfig == nil {
						return nil, E.New("xhttp HTTP/3 requires TLS")
					}
					downloadTLSConfig.SetNextProtos([]string{"h3"})
					udpConn, err := dialer.DialContext(ctx, N.NetworkUDP, downloadServerAddr)
					if err != nil {
						return nil, err
					}
					quicConn, err := qtls.DialEarly(ctx, udpConn, downloadTLSConfig, cfg)
					if err != nil {
						udpConn.Close()
						return nil, err
					}
					go func() {
						<-quicConn.Context().Done()
						udpConn.Close()
					}()
					return quicConn, nil
				},
				downloadALPN,
				downloadHKeepAlivePeriod,
			)
		}
	}

	xhttpClient, err := NewClientFromConfig(cfg, makeTransport, makeDownloadTransport, hasReality)
	if err != nil {
		return nil, err
	}
	return &XHTTPClientTransport{
		ctx:         ctx,
		dialer:      dialer,
		serverAddr:  serverAddr,
		tlsConfig:   tlsConfig,
		hasReality:  hasReality,
		xhttpClient: xhttpClient,
		requestHost: requestHost,
		alpn:        alpn,
	}, nil
}

func isRealityConfig(tlsConfig tls.Config) bool {
	// REALITY configs cannot produce a std config.
	if tlsConfig == nil {
		return false
	}
	_, err := tlsConfig.STDConfig()
	return err != nil && strings.Contains(err.Error(), "reality")
}

func filterALPN(alpn []string) []string {
	switch {
	case len(alpn) == 0:
		return nil
	case len(alpn) == 1 && (alpn[0] == "h3" || alpn[0] == "http/1.1"):
		return alpn
	case len(alpn) == 1 && alpn[0] == "h2":
		return nil
	default:
		// multiple protocols: keep only known ones, default to h2 handling
		var filtered []string
		for _, proto := range alpn {
			switch proto {
			case "h2", "http/1.1":
				filtered = append(filtered, proto)
			}
		}
		return filtered
	}
}

func (c *XHTTPClientTransport) DialContext(ctx context.Context) (net.Conn, error) {
	return c.xhttpClient.Dial(ctx)
}

func (c *XHTTPClientTransport) Close() error {
	c.closeOnce.Do(func() {
		_ = c.xhttpClient.Close()
	})
	return nil
}

type Server struct {
	ctx        context.Context
	logger     logger.ContextLogger
	tlsConfig  tls.ServerConfig
	handler    adapter.V2RayServerTransportHandler
	httpServer *http.Server
	handler2   *requestHandler
}

func NewServer(ctx context.Context, logger logger.ContextLogger, options option.V2RayXHTTPOptions, tlsConfig tls.ServerConfig, handler adapter.V2RayServerTransportHandler) (*Server, error) {
	options, err := resolveExtraOptions(options)
	if err != nil {
		return nil, err
	}
	cfg, err := newConfig(options)
	if err != nil {
		return nil, err
	}
	serverMaxHeaderBytes, err := cfg.GetNormalizedServerMaxHeaderBytes()
	if err != nil {
		return nil, err
	}
	xhandler, err := NewServerHandler(ServerOption{
		Config: *cfg,
		ConnHandler: func(conn net.Conn) {
			handler.NewConnectionEx(DupContext(context.Background()), conn, M.Socksaddr{}, M.Socksaddr{}, nil)
		},
	})
	if err != nil {
		return nil, err
	}
	server := &Server{
		ctx:       ctx,
		logger:    logger,
		tlsConfig: tlsConfig,
		handler:   handler,
		handler2:  xhandler,
	}
	server.httpServer = &http.Server{
		Handler:           xhandler,
		ReadHeaderTimeout: C.TCPTimeout,
		MaxHeaderBytes:    serverMaxHeaderBytes,
		BaseContext: func(net.Listener) context.Context {
			return ctx
		},
		ConnContext: func(ctx context.Context, c net.Conn) context.Context {
			return log.ContextWithNewID(ctx)
		},
		TLSNextProto: make(map[string]func(*http.Server, *tls.STDConn, http.Handler)),
	}
	return server, nil
}

func (s *Server) Network() []string {
	return []string{N.NetworkTCP}
}

func (s *Server) Serve(listener net.Listener) error {
	if s.tlsConfig != nil {
		if len(s.tlsConfig.NextProtos()) == 0 {
			s.tlsConfig.SetNextProtos([]string{"h2", "http/1.1"})
		}
		listener = aTLS.NewListener(listener, s.tlsConfig)
	}
	return s.httpServer.Serve(listener)
}

func (s *Server) ServePacket(listener net.PacketConn) error {
	return os.ErrInvalid
}

func (s *Server) Close() error {
	return common.Close(common.PtrOrNil(s.httpServer))
}

type httpServerConn struct {
	mu      sync.Mutex
	w       http.ResponseWriter
	flusher http.Flusher
	reader  io.ReadCloser
	closed  bool
	done    chan struct{}
	once    sync.Once
}

func newHTTPServerConn(w http.ResponseWriter, r io.ReadCloser) *httpServerConn {
	flusher, _ := w.(http.Flusher)
	return &httpServerConn{
		w:       w,
		flusher: flusher,
		reader:  r,
		done:    make(chan struct{}),
	}
}

func (c *httpServerConn) Read(b []byte) (int, error) {
	n, err := c.reader.Read(b)
	return n, baderror.WrapH2(err)
}

func (c *httpServerConn) Write(b []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return 0, net.ErrClosed
	}

	n, err := c.w.Write(b)
	if err == nil && c.flusher != nil {
		c.flusher.Flush()
	}
	return n, err
}

func (c *httpServerConn) Close() error {
	c.once.Do(func() {
		c.mu.Lock()
		c.closed = true
		c.mu.Unlock()
		close(c.done)
	})
	return c.reader.Close()
}

func (c *httpServerConn) Wait() <-chan struct{} {
	return c.done
}

func DupContext(ctx context.Context) context.Context {
	id, loaded := log.IDFromContext(ctx)
	if !loaded {
		return context.Background()
	}
	return log.ContextWithID(context.Background(), id)
}
