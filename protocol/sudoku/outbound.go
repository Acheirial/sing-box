package sudoku

import (
	"context"
	"net"
	"strconv"
	"strings"
	"sync"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/adapter/outbound"
	"github.com/sagernet/sing-box/common/dialer"
	"github.com/sagernet/sing-box/common/sudoku"
	"github.com/sagernet/sing-box/common/sudoku/obfs/httpmask"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/bufio"
	E "github.com/sagernet/sing/common/exceptions"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
)

func RegisterOutbound(registry *outbound.Registry) {
	outbound.Register[option.SudokuOutboundOptions](registry, C.TypeSudoku, NewOutbound)
}

var _ adapter.Outbound = (*Outbound)(nil)

type Outbound struct {
	outbound.Adapter
	logger   log.ContextLogger
	dialer   N.Dialer
	baseConf sudoku.ProtocolConfig

	muxDialer *sudoku.MultiplexDialer

	httpMaskMu     sync.Mutex
	httpMaskClient *httpmask.TunnelClient
	httpMaskKey    string
}

func NewOutbound(ctx context.Context, router adapter.Router, logger log.ContextLogger, tag string, options option.SudokuOutboundOptions) (adapter.Outbound, error) {
	outbound := &Outbound{
		Adapter: outbound.NewAdapterWithDialerOptions(C.TypeSudoku, tag, []string{N.NetworkTCP, N.NetworkUDP}, options.DialerOptions),
		logger:  logger,
	}
	outboundDialer, err := dialer.NewWithOptions(dialer.Options{
		Context:        ctx,
		Options:        options.DialerOptions,
		RemoteIsDomain: options.ServerOptions.ServerIsDomain(),
	})
	if err != nil {
		return nil, err
	}
	outbound.dialer = outboundDialer

	serverAddress := options.ServerOptions.Build()
	if !serverAddress.IsValid() || serverAddress.Port == 0 {
		return nil, E.New("missing server address")
	}
	baseConf := *sudoku.DefaultConfig()
	baseConf.ServerAddress = serverAddress.String()
	tableType, err := sudoku.NormalizeTableType(options.TableType)
	if err != nil {
		return nil, err
	}
	paddingMin, paddingMax := sudoku.ResolvePadding(options.PaddingMin, options.PaddingMax, baseConf.PaddingMin, baseConf.PaddingMax)
	enablePureDownlink := baseConf.EnablePureDownlink
	if options.EnablePureDownlink != nil {
		enablePureDownlink = *options.EnablePureDownlink
	}

	disableHTTPMask := baseConf.DisableHTTPMask
	if options.HTTPMask != nil {
		disableHTTPMask = !*options.HTTPMask
	}
	httpMaskMode := baseConf.HTTPMaskMode
	if options.HTTPMaskMode != "" {
		httpMaskMode = options.HTTPMaskMode
	}
	httpMaskTLS := options.HTTPMaskTLS
	httpMaskHost := options.HTTPMaskHost
	pathRoot := strings.TrimSpace(options.PathRoot)
	multiplex := baseConf.Multiplex
	if mode := strings.TrimSpace(options.Multiplex); mode != "" {
		multiplex = mode
	}
	if mode := strings.TrimSpace(options.HTTPMaskMultiplex); mode != "" {
		multiplex = mode
	}

	if hm := options.HTTPMaskOptions; hm != nil {
		disableHTTPMask = hm.Disable
		if hm.Mode != "" {
			httpMaskMode = hm.Mode
		}
		httpMaskTLS = hm.TLS
		httpMaskHost = hm.Host
		if pr := strings.TrimSpace(hm.PathRoot); pr != "" {
			pathRoot = pr
		}
		if mux := strings.TrimSpace(hm.Multiplex); mux != "" {
			multiplex = mux
		}
	}
	if options.Key == "" {
		return nil, E.New("missing key")
	}
	baseConf.Key = options.Key
	baseConf.PaddingMin = paddingMin
	baseConf.PaddingMax = paddingMax
	baseConf.EnablePureDownlink = enablePureDownlink
	baseConf.DisableHTTPMask = disableHTTPMask
	baseConf.HTTPMaskMode = httpMaskMode
	baseConf.HTTPMaskTLSEnabled = httpMaskTLS
	baseConf.HTTPMaskHost = httpMaskHost
	baseConf.HTTPMaskPathRoot = pathRoot
	baseConf.Multiplex = multiplex

	tables, err := sudoku.NewClientTablesWithCustomPatterns(sudoku.ClientAEADSeed(options.Key), tableType, options.CustomTable, options.CustomTables)
	if err != nil {
		return nil, E.Cause(err, "build table(s)")
	}
	if len(tables) == 1 {
		baseConf.Table = tables[0]
	} else {
		baseConf.Tables = tables
	}
	if options.AEADMethod != "" {
		baseConf.AEADMethod = options.AEADMethod
	}
	if err := baseConf.Validate(); err != nil {
		return nil, E.Cause(err, "sudoku config")
	}
	outbound.baseConf = baseConf

	if baseConf.SessionMuxEnabled() {
		outbound.muxDialer, err = sudoku.NewMultiplexDialer(func(ctx context.Context) (net.Conn, error) {
			cfg := outbound.baseConf
			return outbound.dialAndHandshake(ctx, &cfg)
		})
		if err != nil {
			return nil, err
		}
	}
	return outbound, nil
}

func (s *Outbound) DialContext(ctx context.Context, network string, destination M.Socksaddr) (net.Conn, error) {
	ctx, metadata := adapter.ExtendContext(ctx)
	metadata.Outbound = s.Tag()
	metadata.Destination = destination
	switch N.NetworkName(network) {
	case N.NetworkTCP:
		s.logger.InfoContext(ctx, "outbound connection to ", destination)
	case N.NetworkUDP:
		s.logger.InfoContext(ctx, "outbound UoT packet connection to ", destination)
		conn, err := s.ListenPacket(ctx, destination)
		if err != nil {
			return nil, err
		}
		return bufio.NewBindPacketConn(conn, destination), nil
	default:
		return nil, E.New("unsupported network: ", network)
	}

	cfg, err := s.buildConfig(destination)
	if err != nil {
		return nil, err
	}

	if cfg.SessionMuxEnabled() {
		return s.muxDialer.Dial(ctx, cfg.TargetAddress)
	}

	c, err := s.dialAndHandshake(ctx, cfg)
	if err != nil {
		return nil, err
	}

	addrBuf, err := sudoku.EncodeAddress(cfg.TargetAddress)
	if err != nil {
		_ = c.Close()
		return nil, E.Cause(err, "encode target address")
	}

	if err = sudoku.WriteKIPMessage(c, sudoku.KIPTypeOpenTCP, addrBuf); err != nil {
		_ = c.Close()
		return nil, E.Cause(err, "send target address")
	}
	return c, nil
}

func (s *Outbound) ListenPacket(ctx context.Context, destination M.Socksaddr) (net.PacketConn, error) {
	ctx, metadata := adapter.ExtendContext(ctx)
	metadata.Outbound = s.Tag()
	metadata.Destination = destination
	s.logger.InfoContext(ctx, "outbound UoT packet connection to ", destination)

	cfg, err := s.buildConfig(destination)
	if err != nil {
		return nil, err
	}

	c, err := s.dialAndHandshake(ctx, cfg)
	if err != nil {
		return nil, err
	}

	if err = sudoku.WriteKIPMessage(c, sudoku.KIPTypeStartUoT, nil); err != nil {
		_ = c.Close()
		return nil, E.Cause(err, "start uot")
	}
	return sudoku.NewUoTPacketConn(c), nil
}

func (s *Outbound) buildConfig(destination M.Socksaddr) (*sudoku.ProtocolConfig, error) {
	if !destination.IsValid() || destination.Port == 0 {
		return nil, E.New("invalid destination for sudoku outbound")
	}

	cfg := s.baseConf
	cfg.TargetAddress = destination.String()

	if err := cfg.ValidateClient(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func httpTunnelModeEnabled(mode string) bool {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "stream", "poll", "auto", "ws":
		return true
	default:
		return false
	}
}

func (s *Outbound) dialDialer(ctx context.Context, network, addr string) (net.Conn, error) {
	return s.dialer.DialContext(ctx, network, M.ParseSocksaddr(addr))
}

func (s *Outbound) dialAndHandshake(ctx context.Context, cfg *sudoku.ProtocolConfig) (_ net.Conn, err error) {
	if cfg == nil {
		return nil, E.New("config is required")
	}

	handshakeCfg := *cfg
	if !handshakeCfg.DisableHTTPMask && httpTunnelModeEnabled(handshakeCfg.HTTPMaskMode) {
		handshakeCfg.DisableHTTPMask = true
	}

	upgrade := func(raw net.Conn) (net.Conn, error) {
		return sudoku.ClientHandshake(raw, &handshakeCfg)
	}

	var (
		c             net.Conn
		handshakeDone bool
	)
	if !cfg.DisableHTTPMask && httpTunnelModeEnabled(cfg.HTTPMaskMode) {
		muxMode := cfg.MultiplexMode()
		if muxMode == "auto" && strings.ToLower(strings.TrimSpace(cfg.HTTPMaskMode)) != "ws" {
			if client, cerr := s.getOrCreateHTTPMaskClient(cfg); cerr == nil && client != nil {
				c, err = client.DialTunnel(ctx, httpmask.TunnelDialOptions{
					Mode:         cfg.HTTPMaskMode,
					TLSEnabled:   cfg.HTTPMaskTLSEnabled,
					HostOverride: cfg.HTTPMaskHost,
					PathRoot:     cfg.HTTPMaskPathRoot,
					AuthKey:      sudoku.ClientAEADSeed(cfg.Key),
					Upgrade:      upgrade,
					DialContext:  s.dialDialer,
				})
				if err != nil {
					s.resetHTTPMaskClient()
				}
			}
		}
		if c == nil && err == nil {
			c, err = sudoku.DialHTTPMaskTunnel(ctx, cfg.ServerAddress, cfg, s.dialDialer, upgrade)
		}
		if err == nil && c != nil {
			handshakeDone = true
		}
	}
	if c == nil && err == nil {
		c, err = s.dialer.DialContext(ctx, "tcp", M.ParseSocksaddr(cfg.ServerAddress))
	}
	if err != nil {
		return nil, E.Cause(err, "connect to ", cfg.ServerAddress)
	}

	defer func() {
		if err != nil {
			_ = c.Close()
		}
	}()

	if !handshakeDone {
		c, err = sudoku.ClientHandshake(c, &handshakeCfg)
		if err != nil {
			return nil, err
		}
	}

	return c, nil
}

func (s *Outbound) resetHTTPMaskClient() {
	s.httpMaskMu.Lock()
	defer s.httpMaskMu.Unlock()
	if s.httpMaskClient != nil {
		// A failed session must not tear down the shared transport used by
		// concurrent sessions. Close only idle connections here; the adapter's
		// final Close owns the permanent client shutdown.
		s.httpMaskClient.CloseIdleConnections()
		s.httpMaskClient = nil
		s.httpMaskKey = ""
	}
}

func (s *Outbound) getOrCreateHTTPMaskClient(cfg *sudoku.ProtocolConfig) (*httpmask.TunnelClient, error) {
	if s == nil || cfg == nil {
		return nil, E.New("nil adapter or config")
	}

	key := cfg.ServerAddress + "|" + strconv.FormatBool(cfg.HTTPMaskTLSEnabled) + "|" + strings.TrimSpace(cfg.HTTPMaskHost)

	s.httpMaskMu.Lock()
	if s.httpMaskClient != nil && s.httpMaskKey == key {
		client := s.httpMaskClient
		s.httpMaskMu.Unlock()
		return client, nil
	}
	s.httpMaskMu.Unlock()

	client, err := httpmask.NewTunnelClient(cfg.ServerAddress, httpmask.TunnelClientOptions{
		TLSEnabled:   cfg.HTTPMaskTLSEnabled,
		HostOverride: cfg.HTTPMaskHost,
		DialContext:  s.dialDialer,
		MaxIdleConns: 32,
	})
	if err != nil {
		return nil, err
	}

	s.httpMaskMu.Lock()
	defer s.httpMaskMu.Unlock()
	if s.httpMaskClient != nil && s.httpMaskKey == key {
		client.CloseIdleConnections()
		return s.httpMaskClient, nil
	}
	if s.httpMaskClient != nil {
		s.httpMaskClient.CloseIdleConnections()
	}
	s.httpMaskClient = client
	s.httpMaskKey = key
	return client, nil
}

func (s *Outbound) Close() error {
	if s.muxDialer != nil {
		_ = s.muxDialer.Close()
	}
	s.resetHTTPMaskClient()
	if s.httpMaskClient != nil {
		s.httpMaskClient.Close()
		s.httpMaskClient = nil
	}
	return nil
}
