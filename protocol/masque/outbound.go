package masque

import (
	"context"
	"crypto/ecdsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"io"
	"net"
	"net/netip"
	"strings"
	"time"

	"github.com/sagernet/quic-go"
	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/adapter/outbound"
	"github.com/sagernet/sing-box/common/dialer"
	"github.com/sagernet/sing-box/common/masque"
	"github.com/sagernet/sing-box/common/quiccongestion"
	"github.com/sagernet/sing-box/common/shadowquic"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing-tun"
	E "github.com/sagernet/sing/common/exceptions"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"

	"golang.org/x/sync/semaphore"
)

func RegisterOutbound(registry *outbound.Registry) {
	outbound.Register[option.MasqueOutboundOptions](registry, C.TypeMasque, NewOutbound)
}

var (
	_ adapter.Outbound = (*Outbound)(nil)
)

type Outbound struct {
	outbound.Adapter
	logger        log.ContextLogger
	dialer        N.Dialer
	serverAddr    M.Socksaddr
	tlsConfig     *tls.Config
	quicConfig    *quic.Config
	stack         *packetStackHandle
	uri           string
	h2DialConn    func(ctx context.Context) (net.Conn, error)
	l4Client      *masque.L4Client
	handshakeTime time.Duration

	runCtx    context.Context
	runCancel context.CancelFunc
	runLock   *semaphore.Weighted
	running   bool

	option option.MasqueOutboundOptions
}

func NewOutbound(ctx context.Context, router adapter.Router, logger log.ContextLogger, tag string, options option.MasqueOutboundOptions) (adapter.Outbound, error) {
	if options.HandshakeTimeout < 0 {
		return nil, E.New("masque handshake timeout must be non-negative")
	}
	if !tun.WithGVisor {
		return nil, E.New("masque requires the with_gvisor build tag for its userspace IP stack")
	}

	privKey, err := parsePrivateKey(options.PrivateKey)
	if err != nil {
		return nil, E.Cause(err, "parse private key")
	}
	peerPubKey, err := parsePublicKey(options.PublicKey)
	if err != nil {
		return nil, E.Cause(err, "parse public key")
	}

	l4proxy := options.Network == "h3-l4proxy"

	uri := options.URI
	if uri == "" {
		uri = masque.ConnectURI
	}

	sni := options.Server
	if !l4proxy && sni == "" {
		sni = masque.ConnectSNI
	}
	if l4proxy {
		// L4 proxy mode uses a distinct SNI by default in mihomo; the server
		// option is authoritative when set.
	}

	tlsConfig, err := masque.PrepareTlsConfig(privKey, peerPubKey, sni, options.SkipCertVerify)
	if err != nil {
		return nil, E.Cause(err, "prepare tls config")
	}

	quicConfig := &quic.Config{
		EnableDatagrams:   true,
		InitialPacketSize: 1242,
		KeepAlivePeriod:   30 * time.Second,
	}

	outboundDialer, err := dialer.NewWithOptions(dialer.Options{
		Context:        ctx,
		Options:        options.DialerOptions,
		RemoteIsDomain: options.ServerIsDomain(),
	})
	if err != nil {
		return nil, err
	}

	serverAddr := options.ServerOptions.Build()
	if serverAddr.Port == 0 {
		serverAddr.Port = 443
	}

	o := &Outbound{
		Adapter:       outbound.NewAdapterWithDialerOptions(C.TypeMasque, tag, []string{N.NetworkTCP, N.NetworkUDP}, options.DialerOptions),
		logger:        logger,
		dialer:        outboundDialer,
		serverAddr:    serverAddr,
		tlsConfig:     tlsConfig,
		quicConfig:    quicConfig,
		uri:           uri,
		handshakeTime: time.Duration(options.HandshakeTimeout) * time.Second,
		option:        options,
		runLock:       semaphore.NewWeighted(1),
	}
	runCtx, runCancel := context.WithCancel(context.Background())
	o.runCtx = runCtx
	o.runCancel = runCancel

	if l4proxy {
		o.l4Client = masque.NewL4Client(runCtx, o.dialQuic)
	} else {
		localPrefixes, err := parseLocalPrefixes(options)
		if err != nil {
			return nil, err
		}
		stack, err := masque.NewPacketStack(localPrefixes, uint32(mtuOrDefault(options.MTU)))
		if err != nil {
			return nil, E.Cause(err, "create ip stack")
		}
		o.stack = &packetStackHandle{stack: stack}
	}
	return o, nil
}

func parsePrivateKey(value string) (*ecdsa.PrivateKey, error) {
	der, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return nil, E.Cause(err, "decode base64")
	}
	key, err := x509.ParseECPrivateKey(der)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func parsePublicKey(value string) (*ecdsa.PublicKey, error) {
	der, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return nil, E.Cause(err, "decode base64")
	}
	key, err := x509.ParsePKIXPublicKey(der)
	if err != nil {
		return nil, err
	}
	pubKey, isECDSA := key.(*ecdsa.PublicKey)
	if !isECDSA {
		return nil, E.New("endpoint public key is not ECDSA")
	}
	return pubKey, nil
}

func parseLocalPrefixes(options option.MasqueOutboundOptions) ([]netip.Prefix, error) {
	var prefixes []netip.Prefix
	for _, value := range options.LocalPrefixes() {
		if !strings.Contains(value, "/") {
			if strings.Contains(value, ":") {
				value += "/128"
			} else {
				value += "/32"
			}
		}
		prefix, err := netip.ParsePrefix(value)
		if err != nil {
			return nil, E.Cause(err, "parse local prefix")
		}
		prefixes = append(prefixes, prefix.Masked())
	}
	if len(prefixes) == 0 {
		return nil, E.New("missing local address")
	}
	return prefixes, nil
}

func mtuOrDefault(mtu int) int {
	if mtu == 0 {
		return 1280
	}
	return mtu
}

func (w *Outbound) dialQuic(ctx context.Context) (net.PacketConn, *quic.Conn, error) {
	destination, err := w.resolveDestination(ctx)
	if err != nil {
		return nil, nil, err
	}
	// without ConnectionIDLength set, the server occasionally throws
	// PROTOCOL_VIOLATION and that closes our connection
	packetConn, quicConn, err := shadowquic.DialQuic(ctx, destination, w.dialer, w.tlsConfig, w.quicConfig, shadowquic.DialQuicOption{ConnectionIDLength: 20})
	if err != nil {
		return nil, nil, err
	}
	quiccongestion.SetCongestionController(quicConn, w.option.CongestionControl, w.option.CWND, w.option.BBRProfile)
	return packetConn, quicConn, nil
}

func (w *Outbound) resolveDestination(ctx context.Context) (netip.AddrPort, error) {
	destination := w.serverAddr.AddrPort()
	if !destination.IsValid() {
		return netip.AddrPort{}, E.New("invalid server address")
	}
	return destination, nil
}

func (w *Outbound) run(ctx context.Context) error {
	w.runLock.Acquire(context.Background(), 1)
	defer w.runLock.Release(1)
	if w.running {
		return nil
	}
	if w.runCtx.Err() != nil {
		return w.runCtx.Err()
	}

	startCtx := w.runCtx
	var cancel context.CancelFunc
	if w.handshakeTime > 0 {
		startCtx, cancel = context.WithTimeout(startCtx, w.handshakeTime)
		defer cancel()
	}

	if err := w.startLocked(startCtx); err != nil {
		return err
	}
	w.running = true
	return nil
}

func (w *Outbound) startLocked(ctx context.Context) error {

	packetConn, quicConn, err := w.dialQuic(ctx)
	if err != nil {
		return err
	}
	tunnelConn, ipConn, err := masque.ConnectTunnel(ctx, quicConn, w.uri)
	if err != nil {
		_ = packetConn.Close()
		return err
	}
	w.monitorTunnel(tunnelConn, ipConn)
	return nil
}

func (w *Outbound) monitorTunnel(closer io.Closer, ipConn masque.IpConn) {
	runCtx, runCancel := context.WithCancel(w.runCtx)
	go func() {
		<-runCtx.Done()
		w.running = false
		_ = ipConn.Close()
		_ = closer.Close()
	}()
	go func() {
		defer runCancel()
		for runCtx.Err() == nil {
			packet, err := ipConn.ReadPacket()
			if err != nil {
				w.logger.ErrorContext(runCtx, "read packet from masque tunnel: ", err)
				return
			}
			w.stack.InjectPacket(packet)
		}
	}()
	go func() {
		defer runCancel()
		outbound := w.stack.Outbound()
		for {
			select {
			case <-runCtx.Done():
				return
			case packet := <-outbound:
				_, err := ipConn.WritePacket(packet)
				if err != nil {
					w.logger.ErrorContext(runCtx, "write packet to masque tunnel: ", err)
					return
				}
			}
		}
	}()
}

func (w *Outbound) DialContext(ctx context.Context, network string, destination M.Socksaddr) (net.Conn, error) {
	if w.l4Client != nil {
		return w.dialContextL4(ctx, destination)
	}
	if err := w.run(ctx); err != nil {
		return nil, err
	}
	return w.stack.DialContext(ctx, network, destination)
}

func (w *Outbound) dialContextL4(ctx context.Context, destination M.Socksaddr) (net.Conn, error) {
	address := destination.AddrPort()
	if !destination.Addr.IsValid() {
		return nil, E.New("masque l4 proxy mode requires resolved addresses; enable a domain_resolver")
	}
	return w.l4Client.DialContext(ctx, "tcp", M.SocksaddrFromNetIP(address).String())
}

func (w *Outbound) ListenPacket(ctx context.Context, destination M.Socksaddr) (net.PacketConn, error) {
	if w.l4Client != nil {
		return nil, E.New("masque l4 proxy mode is not supported for UDP")
	}
	if err := w.run(ctx); err != nil {
		return nil, err
	}
	return w.stack.ListenPacket(ctx, destination)
}

func (w *Outbound) Close() error {
	w.runCancel()
	if w.stack != nil {
		_ = w.stack.Close()
	}
	if w.l4Client != nil {
		_ = w.l4Client.Close()
	}
	return nil
}

// packetStackHandle defers construction cost and hides the gvisor-typed stack.
type packetStackHandle struct {
	stack *masque.PacketStack
}

func (h *packetStackHandle) InjectPacket(packet []byte) {
	h.stack.InjectPacket(packet)
}

func (h *packetStackHandle) Outbound() <-chan []byte {
	return h.stack.Outbound()
}

func (h *packetStackHandle) DialContext(ctx context.Context, network string, destination M.Socksaddr) (net.Conn, error) {
	return h.stack.DialContext(ctx, network, destination)
}

func (h *packetStackHandle) ListenPacket(ctx context.Context, destination M.Socksaddr) (net.PacketConn, error) {
	return h.stack.ListenPacket(ctx, destination)
}

func (h *packetStackHandle) Close() error {
	return h.stack.Close()
}
