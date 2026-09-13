// Package trusttunnel is a port of mihomo's transport/trusttunnel, originally
// adopted from https://github.com/xchacha20-poly1305/sing-trusttunnel.
// The TCP transport is a plain HTTP/2 CONNECT-style tunnel implemented with
// net/http and golang.org/x/net/http2. The QUIC transport (HTTP/3 over
// QUIC, protocol option `quic`) is gated behind the with_quic build tag in
// trusttunnel_quic.go / trusttunnel_quic_stub.go.
package trusttunnel

import (
	"bytes"
	"encoding/base64"
	"errors"
	"io"
	"net"
	"net/http"
	"net/netip"
	"runtime"
	"sync"
	"time"
)

const (
	UDPMagicAddress         = "_udp2"
	ICMPMagicAddress        = "_icmp"
	HealthCheckMagicAddress = "_check"

	DefaultConnectionTimeout  = 30 * time.Second
	DefaultHealthCheckTimeout = 7 * time.Second
	DefaultSessionTimeout     = 30 * time.Second
)

var (
	// Version is the application version.
	Version = "unknown"

	// AppName is the application name advertised in UDP packets.
	AppName = "sing-box"

	// TCPUserAgent is the user agent for TCP connections.
	// Format: <platform> sing-box/<version>.
	TCPUserAgent = runtime.GOOS + " sing-box/" + Version

	// UDPUserAgent is the user agent for UDP multiplexing.
	// Format: <platform> _udp2.
	UDPUserAgent = runtime.GOOS + " " + UDPMagicAddress

	// ICMPUserAgent is the user agent for ICMP multiplexing.
	// Format: <platform> _icmp.
	ICMPUserAgent = runtime.GOOS + " " + ICMPMagicAddress

	HealthCheckUserAgent = runtime.GOOS
)

func buildAuth(username string, password string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(username+":"+password))
}

// parse16BytesIP parses a 16-byte padded IP address. IPv4 addresses are
// carried in the last 4 bytes with zero padding.
func parse16BytesIP(buffer [16]byte) netip.Addr {
	var zeroPrefix [12]byte
	isIPv4 := bytes.HasPrefix(buffer[:], zeroPrefix[:])
	// Special: check ::1
	isIPv4 = isIPv4 && !(buffer[12] == 0 && buffer[13] == 0 && buffer[14] == 0 && buffer[15] == 1)
	if isIPv4 {
		return netip.AddrFrom4([4]byte(buffer[12:16]))
	}
	return netip.AddrFrom16(buffer)
}

// buildPaddingIP pads an IP address to 16 bytes.
func buildPaddingIP(addr netip.Addr) (buffer [16]byte) {
	if addr.Is6() {
		return addr.As16()
	}
	ipv4 := addr.As4()
	copy(buffer[12:16], ipv4[:])
	return buffer
}

// httpConn is the HTTP/2 (or HTTP/3) stream-backed half of a tunnel
// connection. The request body (an io.Pipe) carries writes, the response
// body carries reads.
type httpConn struct {
	writer    io.Writer
	body      io.ReadCloser
	setupOnce sync.Once
	created   chan struct{}
	createErr error
	cancelFn  func()
	closeFn   func()

	remoteAddr net.Addr
	localAddr  net.Addr

	// deadlines
	deadline *time.Timer
}

func (h *httpConn) setup(body io.ReadCloser, err error) {
	h.setupOnce.Do(func() {
		h.body = body
		h.createErr = err
		close(h.created)
	})
	if h.createErr != nil && body != nil { // conn already closed before setup
		_ = body.Close()
	}
}

func (h *httpConn) waitCreated() error {
	<-h.created
	if h.body != nil {
		return nil
	}
	return h.createErr
}

func (h *httpConn) Close() error {
	var errorArr []error
	h.setup(nil, net.ErrClosed)
	if closer, ok := h.writer.(io.Closer); ok {
		errorArr = append(errorArr, closer.Close())
	}
	if h.body != nil {
		errorArr = append(errorArr, h.body.Close())
	}
	if h.cancelFn != nil {
		h.cancelFn()
	}
	if h.closeFn != nil {
		h.closeFn()
	}
	return errors.Join(errorArr...)
}

func (h *httpConn) writeFlush(p []byte) (n int, err error) {
	n, err = h.writer.Write(p)
	if flusher, ok := h.writer.(http.Flusher); ok {
		flusher.Flush()
	}
	return n, err
}

func (h *httpConn) SetReadDeadline(t time.Time) error  { return h.SetDeadline(t) }
func (h *httpConn) SetWriteDeadline(t time.Time) error { return h.SetDeadline(t) }

func (h *httpConn) SetDeadline(t time.Time) error {
	if t.IsZero() {
		if h.deadline != nil {
			h.deadline.Stop()
			h.deadline = nil
		}
		return nil
	}
	d := time.Until(t)
	if h.deadline != nil {
		h.deadline.Reset(d)
		return nil
	}
	h.deadline = time.AfterFunc(d, func() {
		h.Close()
	})
	return nil
}

func (h *httpConn) LocalAddr() net.Addr {
	return h.localAddr
}

func (h *httpConn) RemoteAddr() net.Addr {
	return h.remoteAddr
}

var _ net.Conn = (*tcpConn)(nil)

type tcpConn struct {
	httpConn
}

func (t *tcpConn) Read(b []byte) (n int, err error) {
	err = t.waitCreated()
	if err != nil {
		return 0, err
	}
	n, err = t.body.Read(b)
	return
}

func (t *tcpConn) Write(b []byte) (int, error) {
	return t.writeFlush(b)
}
