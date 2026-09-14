package gost

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"net"
	"sync"
	"time"

	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
)

const (
	relayVersion1 = 0x01

	relayCmdConnect = 0x01
	relayFlagUDP    = 0x80

	relayStatusOK                  = 0x00
	relayStatusBadRequest          = 0x01
	relayStatusUnauthorized        = 0x02
	relayStatusForbidden           = 0x03
	relayStatusTimeout             = 0x04
	relayStatusServiceUnavailable  = 0x05
	relayStatusHostUnreachable     = 0x06
	relayStatusNetworkUnreachable  = 0x07
	relayStatusInternalServerError = 0x08

	relayFeatureUserAuth = 0x01
	relayFeatureAddr     = 0x02
	relayFeatureNetwork  = 0x04

	relayAddrIPv4   = 0x01
	relayAddrDomain = 0x03
	relayAddrIPv6   = 0x04

	relayNetworkTCP = 0x0000
	relayNetworkUDP = 0x0001
)

// RelayOption is the common gost relay transport configuration.
type RelayOption struct {
	Server   M.Socksaddr
	Forward  bool
	TLS      bool
	Mux      bool
	Username string
	Password string
}

// RelayDialer dials through a gost relay server.
type RelayDialer struct {
	base       N.Dialer
	option     RelayOption
	tlsWrapper TLSWrapper
}

// NewRelayDialer wraps an existing dialer with gost relay connect support.
func NewRelayDialer(base N.Dialer, option *RelayOption) *RelayDialer {
	if option == nil {
		return nil
	}
	return &RelayDialer{
		base:   base,
		option: *option,
	}
}

// NewRelayDialerTLS is NewRelayDialer with a TLS wrapper for the tls option.
func NewRelayDialerTLS(base N.Dialer, option *RelayOption, tlsWrapper TLSWrapper) *RelayDialer {
	if option == nil {
		return nil
	}
	return &RelayDialer{
		base:       base,
		option:     *option,
		tlsWrapper: tlsWrapper,
	}
}

func (d *RelayDialer) DialContext(ctx context.Context, network string, destination M.Socksaddr) (net.Conn, error) {
	if N.NetworkName(network) != N.NetworkTCP {
		return nil, fmt.Errorf("gost relay only supports tcp dial, got %s", network)
	}

	conn, err := d.dialRelayServer(ctx, destination)
	if err != nil {
		return nil, err
	}

	success := false
	defer func() {
		if !success {
			_ = conn.Close()
		}
	}()

	var targetAddress M.Socksaddr
	if !d.option.Forward {
		targetAddress = destination
	}
	err = writeRelayRequest(conn, relayCmdConnect, targetAddress, relayNetworkTCP, d.option.Username, d.option.Password)
	if err != nil {
		return nil, err
	}
	err = readRelayConnectResponse(conn)
	if err != nil {
		return nil, err
	}

	success = true
	return conn, nil
}

func (d *RelayDialer) ListenPacket(ctx context.Context, destination M.Socksaddr) (net.PacketConn, error) {
	if !d.option.Server.IsValid() && d.option.Forward {
		return nil, fmt.Errorf("gost relay udp target address is required")
	}

	conn, err := d.dialRelayServer(ctx, destination)
	if err != nil {
		return nil, err
	}

	success := false
	defer func() {
		if !success {
			_ = conn.Close()
		}
	}()

	var targetAddress M.Socksaddr
	if !d.option.Forward {
		targetAddress = destination
	}
	err = writeRelayRequest(conn, relayCmdConnect|relayFlagUDP, targetAddress, relayNetworkUDP, d.option.Username, d.option.Password)
	if err != nil {
		return nil, err
	}
	err = readRelayConnectResponse(conn)
	if err != nil {
		return nil, err
	}

	success = true
	return &relayPacketConn{
		conn:  conn,
		raddr: destination,
	}, nil
}

func (d *RelayDialer) dialRelayServer(ctx context.Context, fallbackAddress M.Socksaddr) (net.Conn, error) {
	relayAddress := d.option.Server
	if !relayAddress.IsValid() {
		if !d.option.Forward {
			return nil, fmt.Errorf("gost relay server and port are required")
		}
		relayAddress = fallbackAddress
	}
	if !relayAddress.IsValid() {
		return nil, fmt.Errorf("gost relay server and port are required")
	}

	conn, err := d.base.DialContext(ctx, "tcp", relayAddress)
	if err != nil {
		return nil, err
	}

	if d.option.TLS {
		if d.tlsWrapper == nil {
			_ = conn.Close()
			return nil, fmt.Errorf("gost relay tls is enabled but no tls configuration is provided")
		}
		tlsConn, err := d.tlsWrapper(ctx, conn, relayAddress)
		if err != nil {
			_ = conn.Close()
			return nil, err
		}
		conn = tlsConn
	}

	if d.option.Mux {
		conn, err = muxSession(conn)
		if err != nil {
			return nil, err
		}
	}

	return conn, nil
}

func writeRelayRequest(w io.Writer, command byte, address M.Socksaddr, network uint16, username, password string) error {
	features := make([][]byte, 0, 3)
	if username != "" || password != "" {
		if len(username) > 0xFF || len(password) > 0xFF {
			return fmt.Errorf("gost relay username or password too long")
		}
		authFeature, err := encodeRelayFeature(relayFeatureUserAuth, encodeRelayUserAuth(username, password))
		if err != nil {
			return err
		}
		features = append(features, authFeature)
	}

	if address.IsValid() {
		addrPayload, err := encodeRelayAddr(address)
		if err != nil {
			return err
		}
		addrFeature, err := encodeRelayFeature(relayFeatureAddr, addrPayload)
		if err != nil {
			return err
		}
		features = append(features, addrFeature)
	}

	networkFeature, err := encodeRelayFeature(relayFeatureNetwork, []byte{byte(network >> 8), byte(network)})
	if err != nil {
		return err
	}
	features = append(features, networkFeature)

	payloadLen := 0
	for _, feature := range features {
		payloadLen += len(feature)
	}
	if payloadLen > 0xFFFF {
		return fmt.Errorf("gost relay feature list too large")
	}

	header := []byte{
		relayVersion1,
		command,
		byte(payloadLen >> 8),
		byte(payloadLen),
	}
	if _, err := w.Write(header); err != nil {
		return err
	}
	for _, feature := range features {
		if _, err := w.Write(feature); err != nil {
			return err
		}
	}
	return nil
}

type relayPacketConn struct {
	conn  net.Conn
	raddr M.Socksaddr
	wmu   sync.Mutex
}

func (c *relayPacketConn) ReadFrom(p []byte) (int, net.Addr, error) {
	var header [2]byte
	if _, err := io.ReadFull(c.conn, header[:]); err != nil {
		return 0, nil, err
	}

	packetLen := int(binary.BigEndian.Uint16(header[:]))
	if packetLen <= len(p) {
		n, err := io.ReadFull(c.conn, p[:packetLen])
		return n, c.raddr, err
	}

	buf := make([]byte, packetLen)
	if _, err := io.ReadFull(c.conn, buf); err != nil {
		return 0, nil, err
	}
	return copy(p, buf), c.raddr, nil
}

func (c *relayPacketConn) WriteTo(p []byte, addr net.Addr) (int, error) {
	if len(p) > math.MaxUint16 {
		return 0, fmt.Errorf("gost relay udp packet too large: %d", len(p))
	}
	if addr != nil {
		raddr := c.raddr.UDPAddr()
		if udpAddr, ok := addr.(*net.UDPAddr); ok && udpAddr.String() != raddr.String() {
			return 0, fmt.Errorf("gost relay udp association is bound to %s, got %s", raddr, udpAddr)
		}
	}

	c.wmu.Lock()
	defer c.wmu.Unlock()

	var header [2]byte
	binary.BigEndian.PutUint16(header[:], uint16(len(p)))
	if _, err := c.conn.Write(header[:]); err != nil {
		return 0, err
	}
	if _, err := c.conn.Write(p); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (c *relayPacketConn) Close() error {
	return c.conn.Close()
}

func (c *relayPacketConn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *relayPacketConn) SetDeadline(t time.Time) error {
	return c.conn.SetDeadline(t)
}

func (c *relayPacketConn) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

func (c *relayPacketConn) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}

func readRelayConnectResponse(r io.Reader) error {
	var header [4]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return err
	}
	if header[0] != relayVersion1 {
		return fmt.Errorf("gost relay bad version: %d", header[0])
	}
	if header[1] != relayStatusOK {
		return fmt.Errorf("gost relay connect failed with status 0x%02x (%s)", header[1], relayStatusText(header[1]))
	}

	featureLen := int(header[2])<<8 | int(header[3])
	if featureLen == 0 {
		return nil
	}

	_, err := io.CopyN(io.Discard, r, int64(featureLen))
	return err
}

func encodeRelayFeature(featureType byte, payload []byte) ([]byte, error) {
	if len(payload) > 0xFFFF {
		return nil, fmt.Errorf("gost relay feature payload too large")
	}

	out := make([]byte, 3+len(payload))
	out[0] = featureType
	out[1] = byte(len(payload) >> 8)
	out[2] = byte(len(payload))
	copy(out[3:], payload)
	return out, nil
}

func encodeRelayUserAuth(username, password string) []byte {
	out := make([]byte, 0, 2+len(username)+len(password))
	out = append(out, byte(len(username)))
	out = append(out, username...)
	out = append(out, byte(len(password)))
	out = append(out, password...)
	return out
}

func encodeRelayAddr(address M.Socksaddr) ([]byte, error) {
	port := address.Port
	out := make([]byte, 0, 1+1+len(address.AddrString())+2)
	if !address.IsDomain() {
		ip := address.Addr.AsSlice()
		if address.IsIPv4() {
			out = append(out, relayAddrIPv4)
			out = append(out, ip...)
		} else {
			out = append(out, relayAddrIPv6)
			out = append(out, ip...)
		}
	} else {
		host := address.Fqdn
		if len(host) > 0xFF {
			return nil, fmt.Errorf("relay target host too long")
		}
		out = append(out, relayAddrDomain, byte(len(host)))
		out = append(out, host...)
	}

	out = append(out, byte(port>>8), byte(port))
	return out, nil
}

func relayStatusText(status byte) string {
	switch status {
	case relayStatusOK:
		return "ok"
	case relayStatusBadRequest:
		return "bad request"
	case relayStatusUnauthorized:
		return "unauthorized"
	case relayStatusForbidden:
		return "forbidden"
	case relayStatusTimeout:
		return "timeout"
	case relayStatusServiceUnavailable:
		return "service unavailable"
	case relayStatusHostUnreachable:
		return "host unreachable"
	case relayStatusNetworkUnreachable:
		return "network unreachable"
	case relayStatusInternalServerError:
		return "internal server error"
	default:
		return "unknown"
	}
}
