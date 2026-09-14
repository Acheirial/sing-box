package v2raymekya

import (
	"bytes"
	"context"
	"crypto/rand"
	stdtls "crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/sagernet/sing-box/common/tls"
	"github.com/sagernet/sing-box/transport/v2raymkcp"

	"golang.org/x/net/http2"
)

type DialFunc func(ctx context.Context) (net.Conn, error)

const http2NextProtoTLS = "h2"

type Client struct {
	ctx    context.Context
	cancel context.CancelFunc
	cfg    Config
	url    string
	rt     http.RoundTripper
	once   sync.Once
}

func NewHTTPClient(ctx context.Context, dial DialFunc, cfg Config) (*Client, error) {
	ctx, cancel := context.WithCancel(ctx)
	roundTripURL, err := normalizeURL(cfg.URL)
	if err != nil {
		cancel()
		return nil, err
	}
	c := &Client{
		ctx:    ctx,
		cancel: cancel,
		cfg:    cfg,
		url:    roundTripURL,
		rt:     newRoundTripper(dial, cfg.H2PoolSize),
	}
	return c, nil
}

func (c *Client) Dial(ctx context.Context) (net.Conn, error) {
	if err := c.ctx.Err(); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	raw, err := c.newSession()
	if err != nil {
		return nil, err
	}
	conn, err := v2raymkcp.Dial(ctx, raw, c.cfg.KCP)
	if err != nil {
		_ = raw.Close()
		return nil, err
	}
	return &clientConn{Conn: conn, raw: raw}, nil
}

func (c *Client) Close() error {
	c.once.Do(func() {
		c.cancel()
		closeTransport(c.rt)
	})
	return nil
}

func closeTransport(rt any) {
	if closer, ok := rt.(interface{ Close() error }); ok {
		_ = closer.Close()
	}
}

func newRoundTripper(dial DialFunc, h2PoolSize int) http.RoundTripper {
	rt := &alpnAwareRoundTripper{
		dial:          dial,
		connectWithH1: make(map[string]bool),
		pendingConn:   make(map[pendingConnKey]*pendingConn),
	}
	rt.h1 = &http.Transport{
		DialTLSContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return rt.dialOrGetTLSWithExpectedALPN(ctx, addr, false)
		},
		DisableCompression: true,
		ForceAttemptHTTP2:  false,
	}
	rt.h2 = newH2RoundTripper(h2PoolSize, func(ctx context.Context, network, addr string) (net.Conn, error) {
		return rt.dialOrGetTLSWithExpectedALPN(ctx, addr, true)
	})
	return rt
}

func newH2RoundTripper(h2PoolSize int, dialTLSContext func(context.Context, string, string) (net.Conn, error)) http.RoundTripper {
	newTransport := func() http.RoundTripper {
		// h2c mode: the TLS conn already negotiated h2 via ALPN; mask tls.Conn
		// so net/http does not retry the handshake (see golang/go#79293).
		return &http2.Transport{
			DialTLSContext: func(ctx context.Context, network, addr string, _ *stdtls.Config) (net.Conn, error) {
				return dialTLSContext(ctx, network, addr)
			},
			DisableCompression: true,
		}
	}
	if h2PoolSize >= 2 {
		pool := &roundTripperPool{roundTrippers: make([]http.RoundTripper, h2PoolSize)}
		for i := range pool.roundTrippers {
			pool.roundTrippers[i] = newTransport()
		}
		return pool
	}
	return newTransport()
}

var (
	errUnexpectedALPN        = errors.New("mekya: incorrect ALPN negotiated, try again")
	errUnexpectedALPNTooMany = errors.New("mekya: incorrect ALPN negotiated")
)

type alpnAwareRoundTripper struct {
	mu            sync.Mutex
	connectWithH1 map[string]bool
	pendingConn   map[pendingConnKey]*pendingConn

	dial DialFunc
	h1   http.RoundTripper
	h2   http.RoundTripper
}

type pendingConnKey struct {
	addr string
	h2   bool
}

func (r *alpnAwareRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Scheme != "https" {
		return nil, fmt.Errorf("mekya: unsupported url scheme %q", req.URL.Scheme)
	}
	addr := roundTripAddr(req)
	for range 5 {
		var rt http.RoundTripper
		if r.shouldConnectWithH1(addr) {
			rt = r.h1
		} else {
			rt = r.h2
		}
		resp, err := rt.RoundTrip(req)
		if errors.Is(err, errUnexpectedALPN) {
			continue
		}
		return resp, err
	}
	return nil, errUnexpectedALPNTooMany
}

func (r *alpnAwareRoundTripper) Close() error {
	r.mu.Lock()
	pending := r.pendingConn
	r.pendingConn = nil
	r.mu.Unlock()

	for _, conn := range pending {
		conn.close()
	}
	closeTransport(r.h1)
	closeTransport(r.h2)
	return nil
}

func (r *alpnAwareRoundTripper) shouldConnectWithH1(addr string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.connectWithH1[addr]
}

func (r *alpnAwareRoundTripper) dialOrGetTLSWithExpectedALPN(ctx context.Context, addr string, expectedH2 bool) (net.Conn, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.pendingConn == nil {
		return nil, net.ErrClosed
	}
	if r.connectWithH1[addr] == expectedH2 {
		return nil, errUnexpectedALPN
	}
	if conn := r.getPendingConnLocked(addr, expectedH2); conn != nil {
		return conn, nil
	}

	conn, err := r.dial(ctx)
	if err != nil {
		return nil, err
	}
	tlsState := getTLSConnectionState(conn)
	protocolIsH2 := tlsState.NegotiatedProtocol == http2NextProtoTLS

	if !tlsState.HandshakeComplete || protocolIsH2 == expectedH2 {
		return conn, nil
	}

	r.putPendingConnLocked(addr, protocolIsH2, conn)
	r.connectWithH1[addr] = !protocolIsH2
	return nil, errUnexpectedALPN
}

func (r *alpnAwareRoundTripper) getPendingConnLocked(addr string, h2 bool) net.Conn {
	if r.pendingConn == nil {
		return nil
	}
	key := pendingConnKey{addr: addr, h2: h2}
	pending := r.pendingConn[key]
	if pending == nil {
		return nil
	}
	delete(r.pendingConn, key)
	return pending.claim()
}

func (r *alpnAwareRoundTripper) putPendingConnLocked(addr string, h2 bool, conn net.Conn) {
	if r.pendingConn == nil {
		_ = conn.Close()
		return
	}
	key := pendingConnKey{addr: addr, h2: h2}
	if old := r.pendingConn[key]; old != nil {
		old.close()
	}
	r.pendingConn[key] = newPendingConn(conn)
}

func roundTripAddr(req *http.Request) string {
	port := req.URL.Port()
	if port == "" {
		port = "443"
	}
	return net.JoinHostPort(req.URL.Hostname(), port)
}

func getTLSConnectionState(conn net.Conn) tls.ConnectionState {
	type connectionStater interface {
		ConnectionState() tls.ConnectionState
	}
	for {
		if cs, ok := conn.(connectionStater); ok {
			return cs.ConnectionState()
		}
		type netConner interface {
			NetConn() net.Conn
		}
		if nc, ok := conn.(netConner); ok {
			conn = nc.NetConn()
			continue
		}
		return tls.ConnectionState{}
	}
}

type pendingConn struct {
	conn    net.Conn
	timer   *time.Timer
	mu      sync.Mutex
	claimed bool
}

func newPendingConn(conn net.Conn) *pendingConn {
	p := &pendingConn{conn: conn}
	p.timer = time.AfterFunc(time.Minute, p.close)
	return p
}

func (p *pendingConn) claim() net.Conn {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.claimed {
		return nil
	}
	p.claimed = true
	if p.timer != nil {
		p.timer.Stop()
	}
	return p.conn
}

func (p *pendingConn) close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.claimed {
		return
	}
	p.claimed = true
	if p.timer != nil {
		p.timer.Stop()
	}
	_ = p.conn.Close()
}

type roundTripperPool struct {
	mu            sync.Mutex
	next          int
	roundTrippers []http.RoundTripper
}

func (p *roundTripperPool) RoundTrip(req *http.Request) (*http.Response, error) {
	p.mu.Lock()
	rt := p.roundTrippers[p.next]
	p.next = (p.next + 1) % len(p.roundTrippers)
	p.mu.Unlock()
	return rt.RoundTrip(req)
}

func (p *roundTripperPool) Close() error {
	for _, rt := range p.roundTrippers {
		closeTransport(rt)
	}
	return nil
}

func (c *Client) newSession() (*requestClientSession, error) {
	sessionID := make([]byte, 16)
	if _, err := rand.Read(sessionID); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(c.ctx)
	session := &requestClientSession{
		ctx:                    ctx,
		cancel:                 cancel,
		sessionID:              sessionID,
		url:                    c.url,
		currentPollingInterval: c.cfg.PollingIntervalInitial,
		maxRequestSize:         c.cfg.MaxRequestSize,
		maxWriteDelay:          c.cfg.MaxWriteDelay,
		writerChan:             make(chan []byte, 256),
		readerChan:             make(chan []byte, 256),
		deadlines:              newPipeDeadlines(),
		rt:                     c.rt,
	}
	go session.keepRunning()
	return session, nil
}

type clientConn struct {
	net.Conn
	once sync.Once

	raw *requestClientSession
}

func (c *clientConn) Close() error {
	c.once.Do(func() {
		_ = c.raw.Close()
	})
	return c.Conn.Close()
}

func (c *clientConn) LocalAddr() net.Addr {
	if addr := c.raw.LocalAddr(); addr != nil {
		return addr
	}
	return c.Conn.LocalAddr()
}

func (c *clientConn) RemoteAddr() net.Addr {
	if addr := c.raw.RemoteAddr(); addr != nil {
		return addr
	}
	return c.Conn.RemoteAddr()
}

type requestClientSession struct {
	ctx                    context.Context
	cancel                 context.CancelFunc
	sessionID              []byte
	rt                     http.RoundTripper
	url                    string
	currentPollingInterval int
	maxRequestSize         int
	maxWriteDelay          int
	writerChan             chan []byte
	readerChan             chan []byte
	nextWrite              []byte
	deadlines              pipeDeadlines
	addrMu                 sync.RWMutex
	localAddr              net.Addr
	remoteAddr             net.Addr
}

func (s *requestClientSession) keepRunning() {
	for s.ctx.Err() == nil {
		s.runOnce()
	}
}

func (s *requestClientSession) runOnce() {
	requestBody := bytes.NewBuffer(nil)
	waitTimer := time.NewTimer(time.Duration(s.currentPollingInterval) * time.Millisecond)
	defer waitTimer.Stop()
	seenPacket := false

	if s.nextWrite != nil {
		seenPacket = true
		if !waitTimer.Stop() {
			select {
			case <-waitTimer.C:
			default:
			}
		}
		waitTimer.Reset(time.Duration(s.maxWriteDelay) * time.Millisecond)
		if !s.writePacket(requestBody, s.nextWrite) {
			return
		}
		s.nextWrite = nil
	}

copyFromChan:
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-waitTimer.C:
			break copyFromChan
		case packet := <-s.writerChan:
			if !seenPacket {
				seenPacket = true
				if !waitTimer.Stop() {
					select {
					case <-waitTimer.C:
					default:
					}
				}
				waitTimer.Reset(time.Duration(s.maxWriteDelay) * time.Millisecond)
			}
			if !s.writePacket(requestBody, packet) {
				break copyFromChan
			}
		}
	}

	go s.roundTrip(requestBody.Bytes())
}

func (s *requestClientSession) writePacket(requestBody *bytes.Buffer, packet []byte) bool {
	sizeOffset := packetBundleOverhead + len(packet)
	if s.maxRequestSize > 0 && requestBody.Len()+sizeOffset > s.maxRequestSize {
		s.nextWrite = packet
		return false
	}
	if err := writePacketBundle(requestBody, packet); err != nil {
		return false
	}
	return true
}

func (s *requestClientSession) roundTrip(body []byte) {
	req, err := http.NewRequestWithContext(s.ctx, http.MethodPost, s.url, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("X-Session-ID", base64.RawURLEncoding.EncodeToString(s.sessionID))
	resp, err := s.rt.RoundTrip(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	for {
		packet, err := readPacketBundle(resp.Body)
		if err != nil {
			return
		}
		select {
		case <-s.ctx.Done():
			return
		case s.readerChan <- packet:
		}
	}
}

func (s *requestClientSession) Read(p []byte) (int, error) {
	select {
	case <-s.ctx.Done():
		return 0, s.ctx.Err()
	case <-s.deadlines.read.Wait():
		return 0, os.ErrDeadlineExceeded
	case packet := <-s.readerChan:
		return copy(p, packet), nil
	}
}

func (s *requestClientSession) Write(p []byte) (int, error) {
	packet := append([]byte(nil), p...)
	select {
	case <-s.ctx.Done():
		return 0, s.ctx.Err()
	case <-s.deadlines.write.Wait():
		return 0, os.ErrDeadlineExceeded
	case s.writerChan <- packet:
		return len(p), nil
	}
}

func (s *requestClientSession) Close() error {
	s.cancel()
	return nil
}

func (s *requestClientSession) LocalAddr() net.Addr {
	s.addrMu.RLock()
	defer s.addrMu.RUnlock()
	return s.localAddr
}

func (s *requestClientSession) RemoteAddr() net.Addr {
	s.addrMu.RLock()
	defer s.addrMu.RUnlock()
	return s.remoteAddr
}

func (s *requestClientSession) SetDeadline(t time.Time) error {
	return s.deadlines.SetDeadline(t)
}

func (s *requestClientSession) SetReadDeadline(t time.Time) error {
	return s.deadlines.SetReadDeadline(t)
}

func (s *requestClientSession) SetWriteDeadline(t time.Time) error {
	return s.deadlines.SetWriteDeadline(t)
}

var _ net.Conn = (*requestClientSession)(nil)

func normalizeURL(raw string) (string, error) {
	if raw == "" {
		return "", fmt.Errorf("mekya: empty url")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	if u.Scheme == "" {
		u.Scheme = "https"
	}
	if u.Scheme != "https" {
		return "", fmt.Errorf("mekya: unsupported url scheme %q", u.Scheme)
	}
	if u.Host == "" {
		return "", fmt.Errorf("mekya: empty url host")
	}
	return u.String(), nil
}
