package zerotier

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/metacubex/zerotier-go"
	ZTIP "github.com/metacubex/zerotier-go/iplink"
	ZTTransport "github.com/metacubex/zerotier-go/transport"
	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/adapter/outbound"
	"github.com/sagernet/sing-box/common/dialer"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/control"
	E "github.com/sagernet/sing/common/exceptions"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
	"github.com/sagernet/sing/service"
	"github.com/sagernet/sing/service/filemanager"
)

func RegisterOutbound(registry *outbound.Registry) {
	outbound.Register[option.ZeroTierOutboundOptions](registry, C.TypeZeroTier, NewOutbound)
}

const (
	defaultStateDir = "zerotier"
	// frameQueueSize absorbs short multi-flow bursts so data frames do not
	// crowd handshake and window-update traffic out of the single ordered
	// bridge consumer.
	frameQueueSize = 2048
	// frameBatchSize amortizes runtime validation, lock acquisition, and
	// IP-stack delivery while retaining callback order.
	frameBatchSize              = 64
	frameDropLogInterval        = 10 * time.Second
	identityCollisionRetryDelay = 30 * time.Second
)

var (
	errClosed      = errors.New("ZeroTier outbound closed")
	errStaleConfig = errors.New("stale ZeroTier network configuration")
)

var _ adapter.InterfaceUpdateListener = (*Outbound)(nil)

type Outbound struct {
	outbound.Adapter
	ctx                  context.Context
	cancel               context.CancelFunc
	logger               log.ContextLogger
	options              option.ZeroTierOutboundOptions
	networkID            uint64
	stateStore           zerotier.StateStore
	configuredIdentity   zerotier.Identity
	planet               *zerotier.World
	orbits               []orbit
	remoteTraceTarget    zerotier.Address
	physicalDialer       ZTTransport.Dialer
	interfaceFinder      control.InterfaceFinder
	remoteDNS            bool
	configuredDNSServers []netip.AddrPort

	// Lock acquisition rules. Rows are locks already held; columns are locks to
	// acquire next. Any lock may be acquired when no ZeroTier lock is held.
	//
	//   +-------------+-------------+---------+
	//   | held / next | operationMu | stateMu |
	//   +-------------+-------------+---------+
	//   | operationMu |      -      |   YES   |
	//   | stateMu     |     NO      |    -    |
	//   +-------------+-------------+---------+
	//
	// Node state callbacks are serialized by zerotier-go and may run
	// synchronously while operationMu is write-locked. Data callbacks may run
	// concurrently. Both callback paths may take stateMu, but neither takes
	// operationMu; recovery that needs its write lock starts in another
	// goroutine. Control-plane operations take the write lock, while data-plane
	// use of mutable IP-link configuration takes the read lock. stateMu is only
	// a short-lived field lock and is never held while acquiring operationMu.
	operationMu              sync.RWMutex
	closed                   bool
	backgroundStarted        bool
	identityCollisionRetryAt time.Time

	frameCh  chan inboundFrame
	configCh chan struct{}

	// stateMu protects the current runtime pointer and network state below. A
	// runtime is immutable after publication except for its private worker
	// bookkeeping. stateMu is never held across calls into the node, IP link,
	// device, resolver, transport, or filesystem.
	stateMu              sync.RWMutex
	runtime              *runtime
	config               zerotier.NetworkConfigData
	tunDevice            ipStack
	remoteDNSServers     []netip.AddrPort
	networkErr           error
	stateCh              chan struct{}
	latestConfig         zerotier.NetworkConfigData
	haveLatestConfig     bool
	retryLatestConfig    bool
	networkRetrying      bool
	authURL              string
	loggedNetworkFailure string
	configGeneration     uint64
}

type orbit struct {
	world uint64
	seed  zerotier.Address
}

type inboundFrame struct {
	runtime *runtime
	frame   zerotier.Frame
}

// runtime owns one Node generation and all background work tied to it. Its
// lifecycle references do not change after publication; workers is used only
// by startBackgroundTasks and close under operationMu exclusion.
type runtime struct {
	ctx         context.Context
	cancel      context.CancelFunc
	node        *zerotier.Node
	nodeAddress zerotier.Address
	ipLink      *ZTIP.Link
	wire        *ZTTransport.Transport
	workers     sync.WaitGroup
}

type stateFS struct {
	directory string
	tag       string
	logger    log.ContextLogger
}

func (f stateFS) Open(name string) (fs.File, error) {
	return os.Open(filepath.Join(f.directory, filepath.FromSlash(name)))
}

func (f stateFS) WriteFile(name string, data []byte, perm fs.FileMode) error {
	path := filepath.Join(f.directory, filepath.FromSlash(name))
	err := os.MkdirAll(filepath.Dir(path), 0o700)
	if err == nil {
		err = os.WriteFile(path, data, perm)
	}
	if err != nil {
		f.logger.Warn("[ZeroTier](", f.tag, ") unable to write state object ", name, ": ", err)
	}
	return err
}

func (f stateFS) Remove(name string) error {
	return os.Remove(filepath.Join(f.directory, filepath.FromSlash(name)))
}

func NewOutbound(ctx context.Context, router adapter.Router, logger log.ContextLogger, tag string, options option.ZeroTierOutboundOptions) (adapter.Outbound, error) {
	networkID, err := zerotier.ParseNetworkID(options.Network)
	if err != nil {
		return nil, err
	}
	var configuredIdentity zerotier.Identity
	if options.IdentitySecret != "" {
		configuredIdentity, err = zerotier.ParseIdentity(options.IdentitySecret)
		if err != nil {
			return nil, fmt.Errorf("parse ZeroTier identity_secret: %w", err)
		}
		if !configuredIdentity.HasPrivate() {
			return nil, fmt.Errorf("ZeroTier identity_secret must contain private keys: %w", zerotier.ErrPrivateKey)
		}
		if err = configuredIdentity.Validate(); err != nil {
			return nil, fmt.Errorf("validate ZeroTier identity_secret: %w", err)
		}
	}
	if options.MTU != 0 && (options.MTU < zerotier.MinNetworkMTU || options.MTU > zerotier.MaxNetworkMTU) {
		return nil, fmt.Errorf("ZeroTier MTU must be between %d and %d", zerotier.MinNetworkMTU, zerotier.MaxNetworkMTU)
	}
	if options.PhysicalMTU != 0 && (options.PhysicalMTU < zerotier.MinPhysicalMTU || options.PhysicalMTU > zerotier.MaxPhysicalMTU) {
		return nil, fmt.Errorf("ZeroTier physical_mtu must be between %d and %d", zerotier.MinPhysicalMTU, zerotier.MaxPhysicalMTU)
	}
	var planet *zerotier.World
	if options.Planet != "" {
		planetPath := filemanager.BasePath(ctx, os.ExpandEnv(options.Planet))
		planetPath, _ = filepath.Abs(planetPath)
		data, readErr := os.ReadFile(planetPath)
		if readErr != nil {
			return nil, fmt.Errorf("read ZeroTier planet: %w", readErr)
		}
		world, parseErr := zerotier.ParsePlanet(data)
		if parseErr != nil {
			return nil, fmt.Errorf("parse ZeroTier planet: %w", parseErr)
		}
		planet = &world
	}
	orbits := make([]orbit, 0, len(options.Orbit))
	seenWorlds := make(map[uint64]struct{}, len(options.Orbit))
	for _, orbitOption := range options.Orbit {
		world, seed, parseErr := zerotier.ParseOrbit(orbitOption.World, orbitOption.Seed)
		if parseErr != nil {
			return nil, parseErr
		}
		if _, exists := seenWorlds[world]; exists {
			return nil, fmt.Errorf("duplicate ZeroTier orbit world %016x", world)
		}
		seenWorlds[world] = struct{}{}
		orbits = append(orbits, orbit{world: world, seed: seed})
	}
	tcpFallbackMode, err := ZTTransport.ParseTCPFallbackMode(options.TCPFallbackMode)
	if err != nil {
		return nil, err
	}
	options.TCPFallbackMode = tcpFallbackMode.String()
	options.IPStack = normalizeIPStack(options.IPStack)
	if err = validateIPStack(options.IPStack); err != nil {
		return nil, err
	}
	if options.TCPFallbackRelay == "" {
		options.TCPFallbackRelay = ZTTransport.DefaultTCPFallbackRelay
	}
	var remoteTraceTarget zerotier.Address
	if options.RemoteTraceTarget != "" {
		remoteTraceTarget, err = zerotier.ParseAddress(options.RemoteTraceTarget)
		if err != nil || remoteTraceTarget.IsReserved() {
			return nil, errors.New("ZeroTier remote_trace_target must be a 10-digit node ID")
		}
	}
	if options.RemoteTraceLevel > zerotier.TraceLevelInsane {
		return nil, fmt.Errorf("ZeroTier remote_trace_level must be between %d and %d", zerotier.TraceLevelNormal, zerotier.TraceLevelInsane)
	}
	remoteDNS := options.RemoteDnsResolve
	var configuredDNSServers []netip.AddrPort
	if remoteDNS {
		configuredDNSServers, err = parseDNSServers(options.DNS)
		if err != nil {
			return nil, err
		}
	}
	if options.StateDir == "" {
		instance := sha256.Sum256([]byte(tag))
		options.StateDir = filepath.Join(defaultStateDir, fmt.Sprintf("%s-%x", options.Network, instance[:6]))
	}
	options.StateDir = filemanager.BasePath(ctx, os.ExpandEnv(options.StateDir))
	options.StateDir, _ = filepath.Abs(options.StateDir)
	outboundCtx, cancel := context.WithCancel(context.Background())
	stateStore, err := zerotier.NewFileStore(&stateFS{directory: options.StateDir, tag: tag, logger: logger})
	if err != nil {
		cancel()
		return nil, err
	}
	outboundDialer, err := dialer.NewWithOptions(dialer.Options{
		Context:        ctx,
		Options:        options.DialerOptions,
		RemoteIsDomain: true,
	})
	if err != nil {
		cancel()
		return nil, err
	}
	var interfaceFinder control.InterfaceFinder
	if networkManager := service.FromContext[adapter.NetworkManager](ctx); networkManager != nil {
		interfaceFinder = networkManager.InterfaceFinder()
	} else {
		interfaceFinder = control.NewDefaultInterfaceFinder()
	}
	outbound := &Outbound{
		Adapter:              outbound.NewAdapterWithDialerOptions(C.TypeZeroTier, tag, []string{N.NetworkTCP, N.NetworkUDP}, options.DialerOptions),
		ctx:                  outboundCtx,
		cancel:               cancel,
		options:              options,
		networkID:            networkID,
		stateStore:           stateStore,
		configuredIdentity:   configuredIdentity,
		planet:               planet,
		orbits:               orbits,
		remoteTraceTarget:    remoteTraceTarget,
		physicalDialer:       &wireDialer{dialer: outboundDialer},
		interfaceFinder:      interfaceFinder,
		remoteDNS:            remoteDNS,
		configuredDNSServers: configuredDNSServers,
		frameCh:              make(chan inboundFrame, frameQueueSize),
		configCh:             make(chan struct{}, 1),
		stateCh:              make(chan struct{}),
	}
	wireConfig, wireErr := outbound.wireTransportConfig()
	if wireErr == nil {
		wireErr = ZTTransport.ValidateConfig(wireConfig)
	}
	if wireErr != nil {
		cancel()
		return nil, wireErr
	}
	return outbound, nil
}

// wireDialer adapts a sing-box dialer to the zerotier-go physical transport.
type wireDialer struct {
	dialer N.Dialer
}

func (d *wireDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return d.dialer.DialContext(ctx, N.NetworkName(network), M.ParseSocksaddr(address))
}

func (d *wireDialer) ListenPacket(ctx context.Context, network, address string, _ netip.AddrPort) (net.PacketConn, error) {
	return d.dialer.ListenPacket(ctx, M.ParseSocksaddr(address))
}

func (z *Outbound) wireTransportConfig() (ZTTransport.Config, error) {
	fallbackMode, err := ZTTransport.ParseTCPFallbackMode(z.options.TCPFallbackMode)
	if err != nil {
		return ZTTransport.Config{}, err
	}
	return ZTTransport.Config{
		Dialer:           z.physicalDialer,
		Interfaces:       z.transportInterfaces,
		InterfaceName:    z.options.BindInterface,
		SharedUDP:        z.options.Detour == "",
		PrimaryPort:      int(z.options.PrimaryPort),
		SecondaryPort:    int(z.options.SecondaryPort),
		TCPFallbackMode:  fallbackMode,
		TCPFallbackRelay: z.options.TCPFallbackRelay,
		Log: func(level ZTTransport.LogLevel, format string, arguments ...any) {
			message := fmt.Sprintf(format, arguments...)
			if level == ZTTransport.LogInfo {
				z.logger.Info("[ZeroTier](", z.Tag(), ") ", message)
			} else {
				z.logger.Debug("[ZeroTier](", z.Tag(), ") ", message)
			}
		},
	}, nil
}

func (z *Outbound) transportInterfaces() ([]ZTTransport.Interface, error) {
	interfaces := z.interfaceFinder.Interfaces()
	result := make([]ZTTransport.Interface, 0, len(interfaces))
	for _, networkInterface := range interfaces {
		result = append(result, ZTTransport.Interface{
			Name:      networkInterface.Name,
			Flags:     networkInterface.Flags,
			Addresses: networkInterface.Addresses,
		})
	}
	return result, nil
}

func (z *Outbound) InterfaceUpdated(ctx context.Context) {
	z.retryNetwork()
}

func (z *Outbound) Start(stage adapter.StartStage) error {
	return nil
}

func (z *Outbound) Close() error {
	// Cancel and close the physical transport before waiting for operationMu so
	// an in-flight startup dial or wire write cannot prevent its own shutdown.
	z.cancel()
	z.stateMu.RLock()
	rt := z.runtime
	z.stateMu.RUnlock()
	if rt != nil {
		_ = rt.wire.Close()
	}
	z.operationMu.Lock()
	defer z.operationMu.Unlock()
	if z.closed {
		return nil
	}
	z.closed = true
	z.stateMu.Lock()
	rt, device := z.detachRuntimeLocked()
	z.resetNetworkStateLocked(errClosed)
	z.stateMu.Unlock()
	if rt != nil {
		return rt.close(device)
	}
	if device != nil {
		return device.Close()
	}
	return nil
}

func (z *Outbound) detachStackLocked() ipStack {
	device := z.tunDevice
	z.tunDevice = nil
	return device
}

func (z *Outbound) resetNetworkStateLocked(networkErr error) {
	z.config = zerotier.NetworkConfigData{}
	z.latestConfig = zerotier.NetworkConfigData{}
	z.haveLatestConfig = false
	z.retryLatestConfig = false
	z.networkErr = networkErr
	z.authURL = ""
	z.loggedNetworkFailure = ""
	z.configGeneration++
	z.notifyStateLocked()
}

func (z *Outbound) setNetworkFailureLocked(err error, authURL string, retryConfig bool) {
	z.networkErr = err
	z.authURL = authURL
	z.retryLatestConfig = retryConfig && z.haveLatestConfig
	z.notifyStateLocked()
}

func (z *Outbound) clearLoggedNetworkFailure(source *runtime) bool {
	z.stateMu.Lock()
	if z.runtime != source {
		z.stateMu.Unlock()
		return false
	}
	z.loggedNetworkFailure = ""
	z.stateMu.Unlock()
	return true
}

func (z *Outbound) detachRuntimeLocked() (rt *runtime, device ipStack) {
	rt = z.runtime
	device = z.detachStackLocked()
	z.runtime = nil
	return
}

// startBackgroundTasks starts the workers owned by this runtime. The caller
// holds operationMu, so the runtime cannot be detached until Add completes.
func (r *runtime) startBackgroundTasks() {
	r.workers.Add(2)
	go func() {
		defer r.workers.Done()
		r.node.RunBackgroundTasks(r.ctx)
	}()
	go func() {
		defer r.workers.Done()
		r.ipLink.RunBackgroundTasks(r.ctx)
	}()
}

// close stops one retired runtime before its identity or persistent state can
// be reused. Physical receive workers stop before the Node and its background
// tasks, so no transport callback can race Node.Close.
func (r *runtime) close(device ipStack) error {
	r.cancel()
	_ = r.wire.Close()
	r.wire.Wait()
	r.workers.Wait()
	_ = r.node.Close()
	if device != nil {
		return device.Close()
	}
	return nil
}

func (z *Outbound) start() error {
	z.operationMu.Lock()
	defer z.operationMu.Unlock()
	return z.startLocked()
}

func (z *Outbound) startLocked() error {
	if z.closed || z.ctx.Err() != nil {
		return errClosed
	}
	if !z.identityCollisionRetryAt.IsZero() {
		retryAfter := time.Until(z.identityCollisionRetryAt)
		if retryAfter > 0 {
			retryAfter = max(retryAfter.Round(time.Second), time.Second)
			return fmt.Errorf("configured ZeroTier identity %s can retry in %s: %w", z.configuredIdentity.Address(), retryAfter, zerotier.ErrIdentityCollision)
		}
		z.identityCollisionRetryAt = time.Time{}
	}
	z.stateMu.RLock()
	started := z.runtime != nil
	z.stateMu.RUnlock()
	if started {
		return nil
	}
	if err := os.MkdirAll(z.options.StateDir, 0o700); err != nil {
		return err
	}
	wireConfig, err := z.wireTransportConfig()
	if err != nil {
		return err
	}
	wireTransport, err := ZTTransport.New(wireConfig)
	if err != nil {
		return err
	}
	rt := &runtime{}
	var frameDrops struct {
		sync.Mutex
		count uint64
		last  time.Time
	}
	node, err := zerotier.NewNode(zerotier.NodeConfig{
		Identity: z.configuredIdentity,
		Store:    z.stateStore,
		Sender:   wireTransport,
		Planet:   z.planet,
		OnEvent: func(event zerotier.Event) {
			z.handleNodeEvent(rt, event)
		},
		PhysicalMTU:       int(z.options.PhysicalMTU),
		RemoteTraceTarget: z.remoteTraceTarget,
		RemoteTraceLevel:  z.options.RemoteTraceLevel,
		LowBandwidth:      z.options.LowBandwidth,
		EncryptedHello:    z.options.EncryptedHello,
		OnNetworkConfig: func(config zerotier.NetworkConfigData) {
			z.enqueueNetworkConfig(rt, config)
		},
		OnFrame: func(frame zerotier.Frame) {
			select {
			case z.frameCh <- inboundFrame{runtime: rt, frame: frame}:
			default:
				now := time.Now()
				frameDrops.Lock()
				frameDrops.count++
				var dropped uint64
				if frameDrops.last.IsZero() || now.Sub(frameDrops.last) >= frameDropLogInterval {
					dropped = frameDrops.count
					frameDrops.count = 0
					frameDrops.last = now
				}
				frameDrops.Unlock()
				if dropped != 0 {
					z.logger.Warn("[ZeroTier](", z.Tag(), ") dropped ", dropped, " inbound frames because the bridge queue is full")
				}
			}
		},
		DirectPaths: wireTransport.DirectPaths,
	})
	if err != nil {
		_ = wireTransport.Close()
		return err
	}
	ipLink, err := ZTIP.New(z.networkID, node)
	if err != nil {
		_ = wireTransport.Close()
		_ = node.Close()
		return err
	}
	runtimeCtx, runtimeCancel := context.WithCancel(z.ctx)
	rt.ctx = runtimeCtx
	rt.cancel = runtimeCancel
	rt.node = node
	rt.nodeAddress = node.Address()
	rt.ipLink = ipLink
	rt.wire = wireTransport
	z.stateMu.Lock()
	z.runtime = rt
	z.resetNetworkStateLocked(nil)
	z.stateMu.Unlock()
	cleanup := func(startErr error) error {
		z.stateMu.Lock()
		if z.runtime == rt {
			z.runtime = nil
		}
		z.stateMu.Unlock()
		_ = rt.close(nil)
		return startErr
	}
	if err = wireTransport.Start(runtimeCtx, node); err != nil {
		return cleanup(err)
	}
	for _, orbit := range z.orbits {
		if err = node.Orbit(orbit.world, orbit.seed); err != nil {
			return cleanup(fmt.Errorf("orbit ZeroTier world %016x: %w", orbit.world, err))
		}
	}
	if err = node.Join(z.networkID); err != nil {
		return cleanup(err)
	}
	if !z.backgroundStarted {
		z.backgroundStarted = true
		go z.runNetworkConfig()
		go z.runInboundFrames()
	}
	rt.startBackgroundTasks()
	return nil
}

func (z *Outbound) enqueueNetworkConfig(source *runtime, config zerotier.NetworkConfigData) {
	if z.ctx.Err() != nil || source == nil || source.node == nil {
		return
	}
	network, ok := source.node.Network(z.networkID)
	if !ok || network.Status != zerotier.NetworkStatusOK || !network.Config.Equal(config) {
		return
	}
	z.stateMu.Lock()
	if z.ctx.Err() != nil || z.runtime != source {
		z.stateMu.Unlock()
		return
	}
	z.latestConfig = config
	z.haveLatestConfig = true
	z.stateMu.Unlock()
	select {
	case z.configCh <- struct{}{}:
	default:
	}
}

func (z *Outbound) ensureStarted(ctx context.Context) error {
	if err := z.start(); err != nil {
		return err
	}
	z.retryNetwork()
	for {
		z.stateMu.RLock()
		networkErr := z.networkErr
		device := z.tunDevice
		stateCh := z.stateCh
		z.stateMu.RUnlock()
		if device != nil && networkErr == nil {
			return nil
		}
		if networkErr != nil {
			return networkErr
		}
		select {
		case <-stateCh:
		case <-ctx.Done():
			return ctx.Err()
		case <-z.ctx.Done():
			return errClosed
		}
	}
}

func (z *Outbound) notifyStateLocked() {
	close(z.stateCh)
	z.stateCh = make(chan struct{})
}

func (z *Outbound) runNetworkConfig() {
	for {
		select {
		case <-z.configCh:
			z.stateMu.RLock()
			rt := z.runtime
			config := z.latestConfig
			haveConfig := z.haveLatestConfig
			generation := z.configGeneration
			z.stateMu.RUnlock()
			if !haveConfig {
				continue
			}
			if err := z.applyNetworkConfig(config, generation); err != nil && !errors.Is(err, errStaleConfig) {
				if z.recordConfigFailure(rt, config, generation, err) {
					z.logger.Error("[ZeroTier](", z.Tag(), ") apply network configuration: ", err)
				}
			}
		case <-z.ctx.Done():
			return
		}
	}
}

func (z *Outbound) handleNodeEvent(source *runtime, event zerotier.Event) {
	if !z.acceptNodeEvent(source, event) {
		return
	}
	switch event.Type {
	case zerotier.EventNodeDown:
		z.logger.Debug("[ZeroTier](", z.Tag(), ") node ", event.NodeAddress, " shut down")
	case zerotier.EventNodeUp:
		if zerotier.IsAdHocNetworkID(z.networkID) {
			z.logger.Info("[ZeroTier](", z.Tag(), ") node ", event.NodeAddress, " initialized")
		} else {
			z.logger.Info("[ZeroTier](", z.Tag(), ") node ", event.NodeAddress, " initialized; authorize this ID on network ", fmt.Sprintf("%016x", z.networkID))
		}
	case zerotier.EventNodeOnline:
		z.logger.Info("[ZeroTier](", z.Tag(), ") node ", event.NodeAddress, " is online via ", event.Endpoint)
	case zerotier.EventNodeOffline:
		z.logger.Warn("[ZeroTier](", z.Tag(), ") node ", event.NodeAddress, " is offline")
	case zerotier.EventNodeIdentityCollision:
		go z.recoverIdentityCollision(source, event.NodeAddress)
	case zerotier.EventPeerIdentityLearned:
		if event.PeerRole != zerotier.PeerRoleLeaf {
			z.logger.Debug("[ZeroTier](", z.Tag(), ") loaded ", event.PeerRole, " root identity ", event.PeerAddress)
		} else {
			z.logger.Debug("[ZeroTier](", z.Tag(), ") loaded peer identity ", event.PeerAddress)
		}
	case zerotier.EventPeerPathLearned:
		if event.PeerRole != zerotier.PeerRoleLeaf {
			z.logger.Debug("[ZeroTier](", z.Tag(), ") ", event.PeerRole, " root ", event.PeerAddress, " authenticated path ", event.Endpoint)
		} else {
			z.logger.Debug("[ZeroTier](", z.Tag(), ") peer ", event.PeerAddress, " authenticated path ", event.Endpoint)
		}
	case zerotier.EventPeerRouteChanged:
		switch event.Route {
		case zerotier.PeerRouteDirect:
			if event.Endpoint.IsValid() {
				z.logger.Debug("[ZeroTier](", z.Tag(), ") peer ", event.PeerAddress, " selected direct route via ", event.Endpoint)
			} else {
				z.logger.Debug("[ZeroTier](", z.Tag(), ") peer ", event.PeerAddress, " selected ", event.PathCount, " direct paths")
			}
		case zerotier.PeerRouteRelayed:
			z.logger.Debug("[ZeroTier](", z.Tag(), ") peer ", event.PeerAddress, " selected upstream relay route")
		default:
			z.logger.Debug("[ZeroTier](", z.Tag(), ") peer ", event.PeerAddress, " selected unknown route ", event.Route)
		}
	case zerotier.EventLocalSurfaceChanged:
		z.logger.Debug("[ZeroTier](", z.Tag(), ") external surface changed ", event.PreviousEndpoint, " -> ", event.Endpoint,
			" after report from ", event.PeerRole, " root ", event.ReporterAddress, "; revalidating ", event.PathCount, " paths")
	case zerotier.EventNetworkConfigPending:
		z.logger.Debug("[ZeroTier](", z.Tag(), ") requesting configuration for network ", fmt.Sprintf("%016x", event.NetworkID))
	case zerotier.EventNetworkConfigReady:
		if !z.clearLoggedNetworkFailure(source) {
			return
		}
		if zerotier.IsAdHocNetworkID(event.NetworkID) {
			z.logger.Info("[ZeroTier](", z.Tag(), ") network ", fmt.Sprintf("%016x", event.NetworkID), " ad-hoc configuration created")
		} else {
			z.logger.Info("[ZeroTier](", z.Tag(), ") network ", fmt.Sprintf("%016x", event.NetworkID), " controller configuration accepted")
		}
	case zerotier.EventNetworkConfigChanged:
		if !z.clearLoggedNetworkFailure(source) {
			return
		}
		if zerotier.IsAdHocNetworkID(event.NetworkID) {
			z.logger.Debug("[ZeroTier](", z.Tag(), ") network ", fmt.Sprintf("%016x", event.NetworkID), " ad-hoc configuration refresh accepted")
		} else {
			z.logger.Debug("[ZeroTier](", z.Tag(), ") network ", fmt.Sprintf("%016x", event.NetworkID), " controller configuration update accepted")
		}
	case zerotier.EventNetworkAccessDenied:
		networkErr := errors.New("ZeroTier network access denied")
		if shouldLog := z.invalidateNetwork(source, networkErr, "", false); shouldLog {
			z.logger.Warn("[ZeroTier](", z.Tag(), ") network ", fmt.Sprintf("%016x", event.NetworkID), " access denied")
		}
	case zerotier.EventNetworkNotFound:
		var networkErr error
		if zerotier.IsAdHocNetworkID(event.NetworkID) {
			networkErr = errors.New("unsupported ZeroTier ad-hoc network ID")
		} else {
			networkErr = errors.New("ZeroTier network not found or controller unsupported")
		}
		if shouldLog := z.invalidateNetwork(source, networkErr, "", false); shouldLog {
			z.logger.Warn("[ZeroTier](", z.Tag(), ") network ", fmt.Sprintf("%016x", event.NetworkID), ": ", networkErr)
		}
	case zerotier.EventNetworkAuthenticationRequired:
		authURL, err := event.Authentication.LoginURL()
		if err != nil {
			networkErr := fmt.Errorf("ZeroTier network authentication required: %w", err)
			if shouldLog := z.invalidateNetwork(source, networkErr, "", false); shouldLog {
				z.logger.Warn("[ZeroTier](", z.Tag(), ") network ", fmt.Sprintf("%016x", event.NetworkID), ": ", networkErr)
			}
			return
		}
		if shouldLog := z.invalidateNetwork(source, nil, authURL, false); shouldLog {
			z.logger.Info("[ZeroTier](", z.Tag(), ") network ", fmt.Sprintf("%016x", event.NetworkID), " authentication required; complete login at ", authURL)
		}
	case zerotier.EventNetworkLeft:
		if shouldLog := z.invalidateNetwork(source, errors.New("ZeroTier network was left"), "", false); shouldLog {
			z.logger.Warn("[ZeroTier](", z.Tag(), ") network ", fmt.Sprintf("%016x", event.NetworkID), " was left")
		}
	default:
		z.logger.Debug("[ZeroTier](", z.Tag(), ") ignored unknown node event ", event.Type)
	}
}

// acceptNodeEvent rejects callbacks from retired runtimes and network events
// superseded before delivery. NodeDown remains useful after detach. Constructor
// callbacks use an allocated but not yet published runtime whose Node is nil.
func (z *Outbound) acceptNodeEvent(source *runtime, event zerotier.Event) bool {
	if event.Type == zerotier.EventNodeDown {
		return true
	}
	if z.ctx.Err() != nil {
		return false
	}
	if source == nil {
		return false
	}
	if source.node == nil {
		return true
	}
	z.stateMu.RLock()
	current := z.runtime == source
	z.stateMu.RUnlock()
	return current && event.MatchesNodeState(source.node)
}

func (z *Outbound) recoverIdentityCollision(source *runtime, address zerotier.Address) {
	z.operationMu.Lock()
	defer z.operationMu.Unlock()
	if z.closed || z.ctx.Err() != nil {
		return
	}
	z.stateMu.Lock()
	if z.runtime != source || source == nil || source.nodeAddress != address {
		z.stateMu.Unlock()
		return
	}
	rt, device := z.detachRuntimeLocked()
	var collisionErr error
	if !z.configuredIdentity.Address().IsZero() {
		collisionErr = fmt.Errorf("configured ZeroTier identity %s: %w", address, zerotier.ErrIdentityCollision)
		z.identityCollisionRetryAt = time.Now().Add(identityCollisionRetryDelay)
	}
	z.resetNetworkStateLocked(collisionErr)
	z.stateMu.Unlock()

	_ = rt.close(device)
	if collisionErr != nil {
		z.logger.Warn("[ZeroTier](", z.Tag(), ") ", collisionErr, "; the next use can retry in ", identityCollisionRetryDelay)
		return
	}
	if err := zerotier.RotateIdentityState(z.stateStore); err != nil {
		z.logger.Warn("[ZeroTier](", z.Tag(), ") unable to rotate collided identity: ", err)
	}
	z.logger.Warn("[ZeroTier](", z.Tag(), ") node ", address, " has an identity collision; generating a new identity")
	startErr := z.startLocked()
	if startErr != nil && !errors.Is(startErr, errClosed) {
		z.logger.Error("[ZeroTier](", z.Tag(), ") restart after identity collision: ", startErr)
		z.invalidateNetwork(nil, startErr, "", false)
	}
}

func (z *Outbound) retryNetwork() {
	z.stateMu.Lock()
	if z.ctx.Err() != nil {
		z.stateMu.Unlock()
		return
	}
	if z.tunDevice != nil && z.networkErr == nil {
		z.stateMu.Unlock()
		return
	}
	if z.networkRetrying {
		z.stateMu.Unlock()
		return
	}
	z.networkRetrying = true
	z.networkErr = nil
	retryConfig := z.retryLatestConfig && z.haveLatestConfig
	haveConfig := z.haveLatestConfig
	config := z.latestConfig
	rt := z.runtime
	generation := z.configGeneration
	z.stateMu.Unlock()
	defer func() {
		z.stateMu.Lock()
		z.networkRetrying = false
		z.stateMu.Unlock()
	}()
	if retryConfig {
		if err := z.applyNetworkConfig(config, generation); err == nil {
			return
		} else if !errors.Is(err, errStaleConfig) {
			z.recordConfigFailure(rt, config, generation, err)
		}
	}
	if rt == nil {
		return
	}
	operation := "refresh network configuration"
	var err error
	if _, joined := rt.node.Network(z.networkID); joined {
		err = rt.node.RefreshNetwork(z.networkID)
	} else {
		operation = "rejoin network"
		err = rt.node.Join(z.networkID)
	}
	if err != nil {
		recorded := false
		z.stateMu.Lock()
		if z.ctx.Err() == nil && z.configSnapshotCurrentLocked(rt, config, haveConfig, generation) {
			z.setNetworkFailureLocked(fmt.Errorf("%s: %w", operation, err), z.authURL, z.retryLatestConfig)
			recorded = true
		}
		z.stateMu.Unlock()
		if recorded {
			if retryConfig {
				z.logger.Debug("[ZeroTier](", z.Tag(), ") ", operation, " after local apply failure: ", err)
			} else {
				z.logger.Debug("[ZeroTier](", z.Tag(), ") ", operation, ": ", err)
			}
		}
	}
}

func (z *Outbound) recordConfigFailure(rt *runtime, config zerotier.NetworkConfigData, generation uint64, err error) bool {
	z.stateMu.Lock()
	defer z.stateMu.Unlock()
	if z.ctx.Err() != nil || !z.configSnapshotCurrentLocked(rt, config, true, generation) {
		return false
	}
	z.setNetworkFailureLocked(err, z.authURL, true)
	return true
}

// invalidateNetwork detaches the virtual stack and publishes a network
// failure. A non-nil source limits the mutation to that runtime generation. It
// reports whether this failure differs from the last logged failure.
func (z *Outbound) invalidateNetwork(source *runtime, err error, authURL string, retryConfig bool) (shouldLog bool) {
	z.stateMu.Lock()
	if z.ctx.Err() != nil || source != nil && z.runtime != source {
		z.stateMu.Unlock()
		return false
	}
	device := z.detachStackLocked()
	if !retryConfig {
		z.latestConfig = zerotier.NetworkConfigData{}
		z.haveLatestConfig = false
	}
	failure := ""
	if authURL != "" {
		failure = "authentication-required:" + authURL
	} else if err != nil {
		failure = err.Error()
	}
	shouldLog = z.loggedNetworkFailure != failure
	z.loggedNetworkFailure = failure
	z.configGeneration++
	z.setNetworkFailureLocked(err, authURL, retryConfig)
	z.stateMu.Unlock()
	if device != nil {
		go func() { _ = device.Close() }()
	}
	return shouldLog
}

func (z *Outbound) invalidateDevice(device ipStack, err error) bool {
	z.stateMu.Lock()
	if z.ctx.Err() != nil || z.tunDevice != device {
		z.stateMu.Unlock()
		return false
	}
	z.detachStackLocked()
	z.setNetworkFailureLocked(err, z.authURL, true)
	z.stateMu.Unlock()
	go func() { _ = device.Close() }()
	return true
}

func (z *Outbound) applyNetworkConfig(config zerotier.NetworkConfigData, generation uint64) error {
	z.operationMu.Lock()
	defer z.operationMu.Unlock()
	if z.ctx.Err() != nil {
		return errClosed
	}
	z.stateMu.RLock()
	rt := z.runtime
	if !z.networkConfigCurrentLocked(rt, config, generation) {
		z.stateMu.RUnlock()
		return errStaleConfig
	}
	oldDevice := z.tunDevice
	oldConfig := z.config
	z.stateMu.RUnlock()
	if rt == nil {
		return errors.New("ZeroTier core is not started")
	}
	ipLink := rt.ipLink
	if len(config.Assigned) == 0 {
		return errors.New("ZeroTier controller assigned no managed addresses")
	}
	mtu := z.effectiveMTU(config)
	replaceDevice := oldDevice == nil || !oldConfig.ManagedAddressesEqual(config) || z.effectiveMTU(oldConfig) != mtu
	var device ipStack
	if replaceDevice {
		var err error
		device, err = newIPStack(z.options.IPStack, config.Assigned, mtu)
		if err != nil {
			return fmt.Errorf("create ZeroTier stack device: %w", err)
		}
		if err = device.Start(); err != nil {
			_ = device.Close()
			return err
		}
		if z.ctx.Err() != nil {
			_ = device.Close()
			return errClosed
		}
	}
	z.stateMu.RLock()
	current := z.tunDevice == oldDevice && z.networkConfigCurrentLocked(rt, config, generation)
	z.stateMu.RUnlock()
	if !current {
		if replaceDevice {
			_ = device.Close()
		}
		return errStaleConfig
	}
	linkConfig := config
	linkConfig.MTU = mtu
	if err := ipLink.ApplyNetworkConfig(linkConfig); err != nil {
		if replaceDevice {
			_ = device.Close()
		}
		return err
	}
	z.stateMu.Lock()
	if z.tunDevice != oldDevice || !z.networkConfigCurrentLocked(rt, config, generation) {
		z.stateMu.Unlock()
		var rollbackErr error
		if oldConfig.NetworkID == z.networkID && len(oldConfig.Assigned) != 0 {
			oldLinkConfig := oldConfig
			oldLinkConfig.MTU = z.effectiveMTU(oldConfig)
			rollbackErr = ipLink.ApplyNetworkConfig(oldLinkConfig)
		}
		if replaceDevice {
			_ = device.Close()
		}
		if rollbackErr != nil {
			z.logger.Warn("[ZeroTier](", z.Tag(), ") restore IP link after stale configuration: ", rollbackErr)
		}
		return errStaleConfig
	}
	replacedDevice := z.tunDevice
	z.config = config
	z.tunDevice = device
	if z.remoteDNS && len(z.configuredDNSServers) == 0 {
		z.remoteDNSServers = config.DNSServers
	}
	z.networkErr = nil
	z.authURL = ""
	z.retryLatestConfig = false
	z.configGeneration++
	z.notifyStateLocked()
	z.stateMu.Unlock()
	if !replaceDevice {
		return nil
	}
	go z.runStackPackets(rt, device)
	action := "joined"
	if replacedDevice != nil {
		_ = replacedDevice.Close()
		action = "updated"
	}
	z.logger.Info("[ZeroTier](", z.Tag(), ") ", action, " network ", fmt.Sprintf("%016x", config.NetworkID),
		" (", config.Name, "), addresses=", config.Assigned, " routes=", config.Routes, " mtu=", mtu)
	return nil
}

func (z *Outbound) networkConfigCurrentLocked(rt *runtime, config zerotier.NetworkConfigData, generation uint64) bool {
	return z.configSnapshotCurrentLocked(rt, config, true, generation)
}

func (z *Outbound) configSnapshotCurrentLocked(rt *runtime, config zerotier.NetworkConfigData, haveConfig bool, generation uint64) bool {
	return z.configGeneration == generation && z.runtime == rt && z.haveLatestConfig == haveConfig && (!haveConfig || z.latestConfig.Equal(config))
}

func (z *Outbound) effectiveMTU(config zerotier.NetworkConfigData) uint32 {
	mtu := config.MTU
	switch {
	case mtu == 0:
		mtu = zerotier.DefaultNetworkMTU
	case mtu < zerotier.MinNetworkMTU:
		mtu = zerotier.MinNetworkMTU
	case mtu > zerotier.MaxNetworkMTU:
		mtu = zerotier.MaxNetworkMTU
	}
	if z.options.MTU != 0 && z.options.MTU < mtu {
		mtu = z.options.MTU
	}
	return mtu
}

func (z *Outbound) runStackPackets(rt *runtime, device ipStack) {
	mtu, err := device.MTU()
	if err != nil || mtu < 1 {
		mtu = 64 * 1024
	}
	batchSize := max(device.BatchSize(), 1)
	storage := make([]byte, mtu*batchSize)
	buffers := make([][]byte, batchSize)
	for index := range buffers {
		start := index * mtu
		buffers[index] = storage[start : start+mtu : start+mtu]
	}
	sizes := make([]int, batchSize)
	writeErrors := make([]error, 0, batchSize)
	for z.ctx.Err() == nil {
		count, readErr := device.Read(buffers, sizes, 0)
		if readErr != nil {
			if z.ctx.Err() == nil {
				invalidated := z.invalidateDevice(device, fmt.Errorf("ZeroTier stack read failed: %w", readErr))
				if invalidated && !errors.Is(readErr, net.ErrClosed) && !errors.Is(readErr, os.ErrClosed) {
					z.logger.Error("[ZeroTier](", z.Tag(), ") stack read: ", readErr)
				}
			}
			return
		}
		z.operationMu.RLock()
		z.stateMu.RLock()
		current := z.runtime == rt && z.tunDevice == device
		z.stateMu.RUnlock()
		if !current {
			z.operationMu.RUnlock()
			return
		}
		writeErrors = writeErrors[:0]
		for index := range count {
			if writeErr := rt.ipLink.WritePacket(buffers[index][:sizes[index]]); writeErr != nil {
				writeErrors = append(writeErrors, writeErr)
			}
		}
		z.operationMu.RUnlock()
		for _, writeErr := range writeErrors {
			z.logger.Debug("[ZeroTier](", z.Tag(), ") send IP packet: ", writeErr)
		}
	}
}

func (z *Outbound) runInboundFrames() {
	frames := make([]inboundFrame, frameBatchSize)
	packets := make([][]byte, 0, frameBatchSize)
	frameErrors := make([]error, 0, frameBatchSize)
	for {
		var first inboundFrame
		select {
		case first = <-z.frameCh:
		case <-z.ctx.Done():
			return
		}
		frames[0] = first
		count := 1
	drain:
		for count < len(frames) {
			select {
			case frames[count] = <-z.frameCh:
				count++
			default:
				break drain
			}
		}
		device, output, processingErrors := z.processInboundFrames(frames[:count], packets[:0], frameErrors[:0])
		for _, err := range processingErrors {
			z.logger.Debug("[ZeroTier](", z.Tag(), ") process inbound frame: ", err)
		}
		if device != nil && len(output) != 0 {
			if _, writeErr := device.Write(output, 0); writeErr != nil && z.ctx.Err() == nil {
				invalidated := z.invalidateDevice(device, fmt.Errorf("ZeroTier stack write failed: %w", writeErr))
				if invalidated && !errors.Is(writeErr, net.ErrClosed) && !errors.Is(writeErr, os.ErrClosed) {
					z.logger.Debug("[ZeroTier](", z.Tag(), ") stack write: ", writeErr)
				}
			}
		}
		for index := range count {
			frames[index] = inboundFrame{}
		}
		packets, frameErrors = output, processingErrors
	}
}

// processInboundFrames converts one callback-order batch while preventing a
// concurrent configuration update from mutating its IP link. Stack delivery
// follows after the read lock is released.
func (z *Outbound) processInboundFrames(frames []inboundFrame, packets [][]byte, frameErrors []error) (ipStack, [][]byte, []error) {
	z.operationMu.RLock()
	z.stateMu.RLock()
	rt := z.runtime
	device := z.tunDevice
	z.stateMu.RUnlock()
	if rt == nil {
		z.operationMu.RUnlock()
		return nil, packets, frameErrors
	}
	for _, inbound := range frames {
		if inbound.runtime != rt {
			continue
		}
		packet, err := rt.ipLink.HandleFrame(inbound.frame)
		if err != nil {
			frameErrors = append(frameErrors, err)
		}
		if len(packet) != 0 {
			packets = append(packets, packet)
		}
	}
	z.operationMu.RUnlock()
	return device, packets, frameErrors
}

func (z *Outbound) networkStackFor(destination netip.Addr) (*ZTIP.Link, ipStack, error) {
	z.operationMu.RLock()
	defer z.operationMu.RUnlock()
	z.stateMu.RLock()
	rt := z.runtime
	device := z.tunDevice
	networkErr := z.networkErr
	z.stateMu.RUnlock()
	if rt == nil {
		return nil, nil, errors.New("ZeroTier core is not ready")
	}
	ipLink := rt.ipLink
	if networkErr != nil {
		return nil, nil, networkErr
	}
	if device == nil {
		return nil, nil, errors.New("ZeroTier stack is not ready")
	}
	if err := ipLink.ValidateDestination(destination); err != nil {
		return nil, nil, err
	}
	z.stateMu.RLock()
	current := z.runtime == rt && z.tunDevice == device && z.networkErr == nil
	z.stateMu.RUnlock()
	if !current {
		return nil, nil, errors.New("ZeroTier stack changed while validating destination")
	}
	return ipLink, device, nil
}

func (z *Outbound) DialContext(ctx context.Context, network string, destination M.Socksaddr) (net.Conn, error) {
	network = N.NetworkName(network)
	switch network {
	case N.NetworkTCP, N.NetworkUDP:
	default:
		return nil, E.Extend(N.ErrUnknownNetwork, network)
	}
	z.logger.InfoContext(ctx, "outbound connection to ", destination)
	if err := z.ensureStarted(ctx); err != nil {
		return nil, err
	}
	if destination.IsDomain() {
		destinationAddress, err := z.lookup(ctx, destination.Fqdn)
		if err != nil {
			return nil, err
		}
		destination = M.SocksaddrFrom(destinationAddress, destination.Port)
	}
	_, device, err := z.networkStackFor(destination.Addr)
	if err != nil {
		return nil, err
	}
	destinationAddrPort := destination.AddrPort()
	switch network {
	case N.NetworkTCP:
		return device.DialTCP(ctx, network, netip.AddrPort{}, destinationAddrPort)
	case N.NetworkUDP:
		return device.DialUDP(ctx, network, netip.AddrPort{}, destinationAddrPort)
	default:
		return nil, E.Extend(N.ErrUnknownNetwork, network)
	}
}

func (z *Outbound) ListenPacket(ctx context.Context, destination M.Socksaddr) (net.PacketConn, error) {
	z.logger.InfoContext(ctx, "outbound packet connection to ", destination)
	if err := z.ensureStarted(ctx); err != nil {
		return nil, err
	}
	if destination.IsDomain() {
		destinationAddress, err := z.lookup(ctx, destination.Fqdn)
		if err != nil {
			return nil, err
		}
		destination = M.SocksaddrFrom(destinationAddress, destination.Port)
	}
	ipLink, device, err := z.networkStackFor(destination.Addr)
	if err != nil {
		return nil, err
	}
	// The ipStack contract guarantees that a generic UDP wildcard supports
	// both address families.
	packetConn, err := device.ListenUDP(ctx, N.NetworkUDP, netip.AddrPort{})
	if err != nil {
		return nil, err
	}
	if packetConn == nil {
		return nil, errors.New("packetConn is nil")
	}
	sourceAddress, err := ipLink.SourceFor(destination.Addr)
	if err != nil {
		sourceAddress = destination.Addr
	}
	return &zeroTierPacketConn{
		PacketConn: packetConn,
		source:     M.SocksaddrFrom(sourceAddress, 0),
		validateDestination: func(destination netip.Addr) error {
			currentLink, currentDevice, validateErr := z.networkStackFor(destination)
			if validateErr != nil {
				return validateErr
			}
			if currentLink != ipLink || currentDevice != device {
				return errors.New("ZeroTier stack changed while packet connection was active")
			}
			return nil
		},
	}, nil
}

func (z *Outbound) lookup(ctx context.Context, domain string) (netip.Addr, error) {
	if z.remoteDNS {
		addresses, lookupErr := z.remoteLookup(ctx, domain)
		if lookupErr != nil {
			return netip.Addr{}, lookupErr
		}
		return pickAddress(addresses), nil
	}
	dnsRouter := service.FromContext[adapter.DNSRouter](z.ctx)
	if dnsRouter == nil {
		addresses, err := net.DefaultResolver.LookupNetIP(ctx, "ip", domain)
		if err != nil {
			return netip.Addr{}, err
		}
		return pickAddress(addresses), nil
	}
	addresses, err := dnsRouter.Lookup(ctx, domain, adapter.DNSQueryOptions{})
	if err != nil {
		return netip.Addr{}, err
	}
	return pickAddress(addresses), nil
}

func pickAddress(addresses []netip.Addr) netip.Addr {
	for _, address := range addresses {
		if address.Is4() {
			return address
		}
	}
	if len(addresses) > 0 {
		return addresses[0]
	}
	return netip.Addr{}
}

type zeroTierPacketConn struct {
	net.PacketConn
	source              M.Socksaddr
	validateDestination func(netip.Addr) error
}

func (c *zeroTierPacketConn) LocalAddr() net.Addr {
	return c.source.UDPAddr()
}

func (c *zeroTierPacketConn) WriteTo(packet []byte, destination net.Addr) (int, error) {
	address := M.SocksaddrFromNet(destination).Unwrap()
	if !address.IsIP() {
		return 0, fmt.Errorf("invalid ZeroTier UDP destination %v", destination)
	}
	if err := c.validateDestination(address.Addr); err != nil {
		return 0, err
	}
	return c.PacketConn.WriteTo(packet, destination)
}

var _ N.Dialer = (*Outbound)(nil)
