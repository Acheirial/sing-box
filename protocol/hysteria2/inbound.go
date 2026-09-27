package hysteria2

import (
	"context"
	stdTLS "crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/netip"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/adapter/inbound"
	"github.com/sagernet/sing-box/common/dialer"
	"github.com/sagernet/sing-box/common/listener"
	"github.com/sagernet/sing-box/common/tls"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	qtls "github.com/sagernet/sing-quic"
	"github.com/sagernet/sing-quic/hysteria"
	"github.com/sagernet/sing-quic/hysteria2"
	"github.com/sagernet/sing-quic/hysteria2/realm"
	"github.com/sagernet/sing/common"
	"github.com/sagernet/sing/common/auth"
	E "github.com/sagernet/sing/common/exceptions"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
	"github.com/sagernet/sing/service"
	"github.com/sagernet/sing/service/filemanager"
)

func RegisterInbound(registry *inbound.Registry) {
	inbound.Register[option.Hysteria2InboundOptions](registry, C.TypeHysteria2, NewInbound)
}

type Inbound struct {
	inbound.Adapter
	router       adapter.Router
	logger       log.ContextLogger
	listener     *listener.Listener
	tlsConfig    tls.ServerConfig
	service      *hysteria2.Service[int]
	userNameList []string

	masqueradeHandler     http.Handler
	masqueradeListenHTTP  string
	masqueradeListenHTTPS string
	masqueradeForceHTTPS  bool
	masqueradeServers     []*http.Server
	masqueradeListeners   []net.Listener
}

func NewInbound(ctx context.Context, router adapter.Router, logger log.ContextLogger, tag string, options option.Hysteria2InboundOptions) (adapter.Inbound, error) {
	if options.TLS == nil || !options.TLS.Enabled {
		return nil, C.ErrTLSRequired
	}
	tlsConfig, err := tls.NewServer(ctx, logger, common.PtrValueOrDefault(options.TLS))
	if err != nil {
		return nil, err
	}
	var salamanderPassword string
	var geckoPassword string
	var geckoMinPacketSize, geckoMaxPacketSize int
	if options.Obfs != nil {
		if options.Obfs.Password == "" {
			return nil, E.New("missing obfs password")
		}
		switch options.Obfs.Type {
		case hysteria2.ObfsTypeSalamander:
			salamanderPassword = options.Obfs.Password
		case hysteria2.ObfsTypeGecko:
			geckoPassword = options.Obfs.Password
			geckoMinPacketSize = options.Obfs.GeckoOptions.MinPacketSize
			geckoMaxPacketSize = options.Obfs.GeckoOptions.MaxPacketSize
		default:
			return nil, E.New("unknown obfs type: ", options.Obfs.Type)
		}
	}
	var masqueradeHandler http.Handler
	if options.Masquerade != nil && options.Masquerade.Type != "" {
		switch options.Masquerade.Type {
		case C.Hysterai2MasqueradeTypeFile:
			masqueradeDirectory := filemanager.BasePath(ctx, os.ExpandEnv(options.Masquerade.FileOptions.Directory))
			_, err = filemanager.ReadDir(ctx, masqueradeDirectory)
			if err != nil && !os.IsNotExist(err) {
				return nil, E.Cause(err, "read masquerade directory")
			}
			masqueradeHandler = http.FileServer(http.Dir(masqueradeDirectory))
		case C.Hysterai2MasqueradeTypeProxy:
			masqueradeURL, err := url.Parse(options.Masquerade.ProxyOptions.URL)
			if err != nil {
				return nil, E.Cause(err, "parse masquerade URL")
			}
			proxyRewrite := func(r *httputil.ProxyRequest) {
				r.SetURL(masqueradeURL)
				if !options.Masquerade.ProxyOptions.RewriteHost {
					r.Out.Host = r.In.Host
				}
				if options.Masquerade.ProxyOptions.XForwarded {
					r.SetXForwarded()
				}
			}
			masqueradeHandler = &httputil.ReverseProxy{
				Rewrite: proxyRewrite,
				ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
					w.WriteHeader(http.StatusBadGateway)
				},
			}
			if options.Masquerade.ProxyOptions.Insecure {
				masqueradeHandler.(*httputil.ReverseProxy).Transport = &http.Transport{
					TLSClientConfig: &stdTLS.Config{InsecureSkipVerify: true},
				}
			}
		case C.Hysterai2MasqueradeTypeString:
			masqueradeHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if options.Masquerade.StringOptions.StatusCode != 0 {
					w.WriteHeader(options.Masquerade.StringOptions.StatusCode)
				}
				for key, values := range options.Masquerade.StringOptions.Headers {
					for _, value := range values {
						w.Header().Add(key, value)
					}
				}
				w.Write([]byte(options.Masquerade.StringOptions.Content))
			})
		default:
			return nil, E.New("unknown masquerade type: ", options.Masquerade.Type)
		}
	}
	inbound := &Inbound{
		Adapter: inbound.NewAdapter(C.TypeHysteria2, tag),
		router:  router,
		logger:  logger,
		listener: listener.New(listener.Options{
			Context: ctx,
			Logger:  logger,
			Listen:  options.ListenOptions,
		}),
		tlsConfig:            tlsConfig,
		masqueradeHandler:    masqueradeHandler,
		masqueradeForceHTTPS: options.Masquerade != nil && options.Masquerade.ForceHTTPS,
	}
	if options.Masquerade != nil {
		inbound.masqueradeListenHTTP = options.Masquerade.ListenHTTP
		inbound.masqueradeListenHTTPS = options.Masquerade.ListenHTTPS
		if options.Masquerade.ForceHTTPS && (options.Masquerade.ListenHTTP == "" || options.Masquerade.ListenHTTPS == "") {
			return nil, E.New("masquerade force_https requires listen_http and listen_https")
		}
		if (options.Masquerade.ListenHTTP != "" || options.Masquerade.ListenHTTPS != "") && masqueradeHandler == nil {
			return nil, E.New("masquerade listen_http and listen_https require a masquerade type")
		}
	}
	var udpTimeout time.Duration
	if options.UDPTimeout != 0 {
		udpTimeout = time.Duration(options.UDPTimeout)
	} else {
		udpTimeout = C.UDPTimeout
	}
	var realmOptions *realm.Options
	if options.Realm != nil {
		if options.Realm.IPVersion != 0 && options.ListenOptions.Listen != nil {
			listenAddr := netip.Addr(*options.ListenOptions.Listen).Unmap()
			if options.Realm.IPVersion == 6 && listenAddr.Is4() {
				return nil, E.New("realm.ip_version 6 conflicts with listen address ", listenAddr)
			}
			if options.Realm.IPVersion == 4 && listenAddr.Is6() && !listenAddr.IsUnspecified() {
				return nil, E.New("realm.ip_version 4 conflicts with listen address ", listenAddr)
			}
		}
		var queryOptions adapter.DNSQueryOptions
		if options.Realm.STUNServersIsDomain() {
			queryOptions, err = dialer.NewDNSQueryOptions(ctx, options.Realm.STUNDomainResolver, true)
			if err != nil {
				return nil, E.Cause(err, "create realm STUN domain resolver")
			}
		}
		var httpClientTransport adapter.HTTPTransport
		httpClientTransport, err = service.FromContext[adapter.HTTPClientManager](ctx).ResolveTransport(ctx, logger, common.PtrValueOrDefault(options.Realm.HTTPClient))
		if err != nil {
			return nil, E.Cause(err, "create realm http client")
		}
		dnsRouter := service.FromContext[adapter.DNSRouter](ctx)
		realmOptions = &realm.Options{
			ServerURL:   options.Realm.ServerURL,
			Token:       options.Realm.Token,
			RealmID:     options.Realm.RealmID,
			STUNServers: options.Realm.STUNServers,
			HTTPClient:  &http.Client{Transport: httpClientTransport},
			Resolver: func(ctx context.Context, host string, ipv4, ipv6 bool) ([]netip.Addr, error) {
				dnsOptions := queryOptions
				switch {
				case ipv4 && !ipv6:
					dnsOptions.Strategy = C.DomainStrategyIPv4Only
				case !ipv4 && ipv6:
					dnsOptions.Strategy = C.DomainStrategyIPv6Only
				}
				return dnsRouter.Lookup(ctx, host, dnsOptions)
			},
			Logger:    logger,
			IPVersion: options.Realm.IPVersion,
		}
		if options.Realm.PortMapping != nil && options.Realm.PortMapping.Enabled {
			realmOptions.PortMapping = &realm.PortMappingOptions{
				Timeout:  time.Duration(options.Realm.PortMapping.Timeout),
				Lifetime: time.Duration(options.Realm.PortMapping.Lifetime),
			}
		}
	}
	hysteriaService, err := hysteria2.NewService[int](hysteria2.ServiceOptions{
		Context:            ctx,
		Logger:             logger,
		BrutalDebug:        options.BrutalDebug,
		SendBPS:            uint64(options.UpMbps * hysteria.MbpsToBps),
		ReceiveBPS:         uint64(options.DownMbps * hysteria.MbpsToBps),
		SalamanderPassword: salamanderPassword,
		GeckoPassword:      geckoPassword,
		GeckoMinPacketSize: geckoMinPacketSize,
		GeckoMaxPacketSize: geckoMaxPacketSize,
		TLSConfig:          tlsConfig,
		QUICOptions: qtls.QUICOptions{
			IdleTimeout:             options.IdleTimeout.Build(),
			KeepAlivePeriod:         options.KeepAlivePeriod.Build(),
			StreamReceiveWindow:     options.StreamReceiveWindow.Value(),
			ConnectionReceiveWindow: options.ConnectionReceiveWindow.Value(),
			MaxConcurrentStreams:    options.MaxConcurrentStreams,
			InitialPacketSize:       options.InitialPacketSize,
			DisablePathMTUDiscovery: options.DisablePathMTUDiscovery,
		},
		IgnoreClientBandwidth: options.IgnoreClientBandwidth,
		UDPDisabled:           options.DisableUDP,
		UDPTimeout:            udpTimeout,
		Handler:               inbound,
		MasqueradeHandler:     masqueradeHandler,
		BBRProfile:            options.BBRProfile,
		RealmOptions:          realmOptions,
	})
	if err != nil {
		return nil, err
	}
	userList := make([]int, 0, len(options.Users))
	userNameList := make([]string, 0, len(options.Users))
	userPasswordList := make([]string, 0, len(options.Users))
	for index, user := range options.Users {
		userList = append(userList, index)
		userNameList = append(userNameList, user.Name)
		userPasswordList = append(userPasswordList, user.Password)
	}
	hysteriaService.UpdateUsers(userList, userPasswordList)
	inbound.service = hysteriaService
	inbound.userNameList = userNameList
	return inbound, nil
}

func (h *Inbound) NewConnectionEx(ctx context.Context, conn net.Conn, source M.Socksaddr, destination M.Socksaddr, onClose N.CloseHandlerFunc) {
	ctx = log.ContextWithNewID(ctx)
	var metadata adapter.InboundContext
	metadata.Inbound = h.Tag()
	metadata.InboundType = h.Type()
	//nolint:staticcheck
	metadata.InboundDetour = h.listener.ListenOptions().Detour
	//nolint:staticcheck
	metadata.OriginDestination = h.listener.UDPAddr()
	metadata.Source = source
	metadata.Destination = destination
	h.logger.InfoContext(ctx, "inbound connection from ", metadata.Source)
	userID, _ := auth.UserFromContext[int](ctx)
	if userName := h.userNameList[userID]; userName != "" {
		metadata.User = userName
		h.logger.InfoContext(ctx, "[", userName, "] inbound connection to ", metadata.Destination)
	} else {
		h.logger.InfoContext(ctx, "inbound connection to ", metadata.Destination)
	}
	h.router.RouteConnectionEx(ctx, conn, metadata, onClose)
}

func (h *Inbound) NewPacketConnectionEx(ctx context.Context, conn N.PacketConn, source M.Socksaddr, destination M.Socksaddr, onClose N.CloseHandlerFunc) {
	ctx = log.ContextWithNewID(ctx)
	var metadata adapter.InboundContext
	metadata.Inbound = h.Tag()
	metadata.InboundType = h.Type()
	//nolint:staticcheck
	metadata.InboundDetour = h.listener.ListenOptions().Detour
	//nolint:staticcheck
	metadata.OriginDestination = h.listener.UDPAddr()
	metadata.Source = source
	metadata.Destination = destination
	h.logger.InfoContext(ctx, "inbound packet connection from ", metadata.Source)
	userID, _ := auth.UserFromContext[int](ctx)
	if userName := h.userNameList[userID]; userName != "" {
		metadata.User = userName
		h.logger.InfoContext(ctx, "[", userName, "] inbound packet connection to ", metadata.Destination)
	} else {
		h.logger.InfoContext(ctx, "inbound packet connection to ", metadata.Destination)
	}
	h.router.RoutePacketConnectionEx(ctx, conn, metadata, onClose)
}

func (h *Inbound) Start(stage adapter.StartStage) error {
	if stage != adapter.StartStateStart {
		return nil
	}
	if h.tlsConfig != nil {
		err := h.tlsConfig.Start()
		if err != nil {
			return err
		}
	}
	packetConn, err := h.listener.ListenUDP()
	if err != nil {
		return err
	}
	if err = h.startMasqueradeServers(); err != nil {
		return err
	}
	return h.service.Start(packetConn)
}

// masqueradeAltSvcWriter advertises the QUIC endpoint on masquerade TCP
// responses, mirroring the reference implementation.
type masqueradeAltSvcWriter struct {
	http.ResponseWriter
	port uint16
}

func (w masqueradeAltSvcWriter) WriteHeader(statusCode int) {
	w.Header().Set("Alt-Svc", fmt.Sprintf(`h3=":%d"; ma=2592000`, w.port))
	w.ResponseWriter.WriteHeader(statusCode)
}

func (h *Inbound) startMasqueradeServers() error {
	if h.masqueradeHandler == nil {
		return nil
	}
	port := uint16(0)
	if address := h.listener.UDPConn().LocalAddr(); address != nil {
		if udpAddress, loaded := address.(*net.UDPAddr); loaded {
			port = uint16(udpAddress.Port)
		}
	}
	serve := func(listener net.Listener, handler http.Handler) {
		server := &http.Server{Handler: handler}
		h.masqueradeServers = append(h.masqueradeServers, server)
		h.masqueradeListeners = append(h.masqueradeListeners, listener)
		go func() {
			if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
				h.logger.ErrorContext(context.Background(), "masquerade server: ", err)
			}
		}()
	}
	if h.masqueradeListenHTTP != "" {
		tcpListener, err := net.Listen("tcp", h.masqueradeListenHTTP)
		if err != nil {
			return E.Cause(err, "listen masquerade HTTP")
		}
		httpsPort := uint16(0)
		if h.masqueradeListenHTTPS != "" {
			if _, portString, splitErr := net.SplitHostPort(h.masqueradeListenHTTPS); splitErr == nil {
				parsedPort, parseErr := strconv.ParseUint(portString, 10, 16)
				if parseErr == nil {
					httpsPort = uint16(parsedPort)
				}
			}
		}
		serve(tcpListener, http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			if h.masqueradeForceHTTPS {
				target := "https://" + request.Host + request.RequestURI
				if httpsPort != 0 && httpsPort != 443 {
					target = fmt.Sprintf("https://%s:%d%s", request.Host, httpsPort, request.RequestURI)
				}
				http.Redirect(writer, request, target, http.StatusMovedPermanently)
				return
			}
			h.masqueradeHandler.ServeHTTP(masqueradeAltSvcWriter{ResponseWriter: writer, port: port}, request)
		}))
	}
	if h.masqueradeListenHTTPS != "" {
		tcpListener, err := net.Listen("tcp", h.masqueradeListenHTTPS)
		if err != nil {
			return E.Cause(err, "listen masquerade HTTPS")
		}
		stdConfig, err := h.tlsConfig.STDConfig()
		if err != nil {
			return E.Cause(err, "masquerade HTTPS requires standard TLS")
		}
		serve(stdTLS.NewListener(tcpListener, stdConfig), http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			h.masqueradeHandler.ServeHTTP(masqueradeAltSvcWriter{ResponseWriter: writer, port: port}, request)
		}))
	}
	return nil
}

func (h *Inbound) InterfaceUpdated(ctx context.Context) {
	h.service.Reset()
}

func (h *Inbound) Close() error {
	errList := make([]error, 0, 4)
	for _, server := range h.masqueradeServers {
		errList = append(errList, server.Close())
	}
	for _, masqueradeListener := range h.masqueradeListeners {
		errList = append(errList, masqueradeListener.Close())
	}
	closeErr := common.Close(
		h.listener,
		h.tlsConfig,
		common.PtrOrNil(h.service),
	)
	return E.Errors(append(errList, closeErr)...)
}
