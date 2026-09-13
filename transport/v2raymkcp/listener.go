package v2raymkcp

import (
	"context"
	"crypto/cipher"
	"net"
	"sync"
)

type Listener struct {
	ctx       context.Context
	cancel    context.CancelFunc
	pc        net.PacketConn
	security  packetSecurity
	cfg       Config
	packets   chan packetPayload
	sessions  map[sessionID]*Conn
	accepts   chan *Conn
	closeOnce sync.Once
	done      chan struct{}
}

type packetSecurity struct {
	aead   cipher.AEAD
	header packetHeader
}

type sessionID struct {
	conv uint16
	addr string
}

type packetPayload struct {
	payload []byte
	addr    net.Addr
}

func Listen(ctx context.Context, pc net.PacketConn, cfg Config) (*Listener, error) {
	security, err := cfg.security()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(ctx)
	l := &Listener{
		ctx:      ctx,
		cancel:   cancel,
		pc:       pc,
		security: packetSecurity{aead: security, header: cfg.packetHeader()},
		cfg:      cfg,
		packets:  make(chan packetPayload, 1024),
		sessions: make(map[sessionID]*Conn),
		accepts:  make(chan *Conn, 1024),
		done:     make(chan struct{}),
	}
	reader := packetReader{security: security, header: cfg.packetHeader()}
	go l.readLoop(reader)
	go l.packetLoop()
	return l, nil
}

func (l *Listener) readLoop(reader packetReader) {
	defer close(l.done)
	buf := make([]byte, 65536)
	for {
		n, addr, err := l.pc.ReadFrom(buf)
		if err != nil {
			return
		}
		payload := make([]byte, n)
		copy(payload, buf[:n])
		select {
		case <-l.ctx.Done():
			return
		case l.packets <- packetPayload{payload: payload, addr: addr}:
		default:
		}
		_ = reader
	}
}

func (l *Listener) packetLoop() {
	reader := packetReader{security: l.security.aead, header: l.security.header}
	for {
		select {
		case <-l.ctx.Done():
			return
		case payload := <-l.packets:
			l.onReceive(reader.read(payload.payload), payload.addr)
		}
	}
}

func (l *Listener) onReceive(segments []segment, addr net.Addr) {
	for _, seg := range segments {
		id := sessionID{conv: seg.conversation(), addr: addr.String()}
		conn, ok := l.sessions[id]
		if !ok {
			writer := packetWriter{
				security: l.security.aead,
				header:   l.security.header,
				writer:   &packetConnWriter{pc: l.pc, addr: addr},
			}
			conn = newConn(addr, addr, seg.conversation(), writer, &packetConnWriter{pc: l.pc, addr: addr}, l.cfg)
			l.sessions[id] = conn
			select {
			case l.accepts <- conn:
			default:
			}
		}
		conn.Input([]segment{seg})
	}
}

func (l *Listener) remove(id sessionID) {
	delete(l.sessions, id)
}

func (l *Listener) Accept() (net.Conn, error) {
	select {
	case <-l.ctx.Done():
		return nil, net.ErrClosed
	case conn := <-l.accepts:
		go func() {
			select {
			case <-l.ctx.Done():
			case <-conn.done:
				l.remove(sessionID{conv: conn.conv, addr: conn.remoteAddr.String()})
			}
		}()
		return conn, nil
	}
}

func (l *Listener) Close() error {
	l.closeOnce.Do(func() {
		l.cancel()
		_ = l.pc.Close()
	})
	return nil
}

func (l *Listener) Addr() net.Addr {
	return l.pc.LocalAddr()
}

type packetConnWriter struct {
	pc   net.PacketConn
	addr net.Addr
}

func (w *packetConnWriter) Write(b []byte) (int, error) {
	return w.pc.WriteTo(b, w.addr)
}

func (w *packetConnWriter) Close() error {
	return nil
}

var _ net.Listener = (*Listener)(nil)
