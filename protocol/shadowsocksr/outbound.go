package shadowsocksr

import (
	"context"
	"encoding/binary"
	"net"
	"net/netip"
	"strings"
	"time"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/adapter/outbound"
	"github.com/sagernet/sing-box/common/dialer"
	"github.com/sagernet/sing-box/common/ssr/cipher"
	"github.com/sagernet/sing-box/common/ssr/obfs"
	"github.com/sagernet/sing-box/common/ssr/protocol"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/buf"
	"github.com/sagernet/sing/common/bufio"
	E "github.com/sagernet/sing/common/exceptions"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
)

func RegisterOutbound(registry *outbound.Registry) {
	outbound.Register[option.ShadowsocksROutboundOptions](registry, C.TypeShadowsocksR, NewOutbound)
}

type Outbound struct {
	outbound.Adapter
	logger     log.ContextLogger
	dialer     N.Dialer
	method     *shadowstreamMethod
	serverAddr M.Socksaddr
	obfs       obfs.Obfs
	protocol   protocol.Protocol
}

// shadowstreamMethod adapts common/ssr/cipher to a dialer-like stream/packet wrapper.
type shadowstreamMethod struct {
	cipher cipher.Cipher
}

func (m *shadowstreamMethod) StreamConn(c net.Conn) net.Conn {
	return cipher.NewConn(c, m.cipher)
}

func (m *shadowstreamMethod) PacketConn(c N.NetPacketConn) N.NetPacketConn {
	return cipher.NewPacketConn(c, m.cipher)
}

func NewOutbound(ctx context.Context, router adapter.Router, logger log.ContextLogger, tag string, options option.ShadowsocksROutboundOptions) (adapter.Outbound, error) {
	// SSR protocol compatibility: "none" is an alias of the dummy cipher.
	method := strings.ToLower(options.Method)
	if method == "none" {
		method = "dummy"
	}

	outboundDialer, err := dialer.NewWithOptions(dialer.Options{
		Context:        ctx,
		Options:        options.DialerOptions,
		RemoteIsDomain: options.ServerIsDomain(),
	})
	if err != nil {
		return nil, err
	}

	outbound := &Outbound{
		Adapter:    outbound.NewAdapterWithDialerOptions(C.TypeShadowsocksR, tag, options.Network.Build(), options.DialerOptions),
		logger:     logger,
		dialer:     outboundDialer,
		serverAddr: options.ServerOptions.Build(),
	}

	var ivSize int
	var key []byte
	if method == "dummy" {
		ivSize = 0
		key = cipher.Kdf(options.Password, 16)
	} else {
		streamCipher, streamKey, err := cipher.Pick(strings.ToUpper(method), nil, options.Password)
		if err != nil {
			return nil, E.Cause(err, "initialize ", method, " cipher")
		}
		ivSize = streamCipher.IVSize()
		key = streamKey
		outbound.method = &shadowstreamMethod{cipher: streamCipher}
	}

	obfsObj, obfsOverhead, err := obfs.PickObfs(options.Obfs, &obfs.Base{
		Host:   options.Server,
		Port:   int(options.ServerPort),
		Key:    key,
		IVSize: ivSize,
		Param:  options.ObfsParam,
	})
	if err != nil {
		return nil, E.Cause(err, "initialize obfs")
	}
	outbound.obfs = obfsObj

	protocolObj, err := protocol.PickProtocol(options.Protocol, &protocol.Base{
		Key:      key,
		Overhead: obfsOverhead,
		Param:    options.ProtocolParam,
	})
	if err != nil {
		return nil, E.Cause(err, "initialize protocol")
	}
	outbound.protocol = protocolObj
	return outbound, nil
}

func (h *Outbound) DialContext(ctx context.Context, network string, destination M.Socksaddr) (net.Conn, error) {
	ctx, metadata := adapter.ExtendContext(ctx)
	metadata.Outbound = h.Tag()
	metadata.Destination = destination
	switch N.NetworkName(network) {
	case N.NetworkTCP:
		h.logger.InfoContext(ctx, "outbound connection to ", destination)
		conn, err := h.dialer.DialContext(ctx, N.NetworkTCP, h.serverAddr)
		if err != nil {
			return nil, err
		}
		return h.dialStreamConn(conn, destination)
	case N.NetworkUDP:
		h.logger.InfoContext(ctx, "outbound packet connection to ", destination)
		packetConn, err := h.ListenPacket(ctx, destination)
		if err != nil {
			return nil, err
		}
		return &packetConnAsConn{PacketConn: packetConn, destination: destination}, nil
	default:
		return nil, E.Extend(N.ErrUnknownNetwork, network)
	}
}

func (h *Outbound) dialStreamConn(conn net.Conn, destination M.Socksaddr) (net.Conn, error) {
	conn = h.obfs.StreamConn(conn)
	var iv []byte
	if h.method != nil {
		conn = h.method.StreamConn(conn)
		streamConn, ok := conn.(*cipher.Conn)
		if !ok {
			conn.Close()
			return nil, E.New("invalid connection type")
		}
		var err error
		iv, err = streamConn.ObtainWriteIV()
		if err != nil {
			conn.Close()
			return nil, err
		}
	}
	conn = h.protocol.StreamConn(conn, iv)
	_, err := conn.Write(destinationAddr(destination))
	if err != nil {
		conn.Close()
		return nil, err
	}
	return conn, nil
}

func (h *Outbound) ListenPacket(ctx context.Context, destination M.Socksaddr) (net.PacketConn, error) {
	ctx, metadata := adapter.ExtendContext(ctx)
	metadata.Outbound = h.Tag()
	metadata.Destination = destination
	h.logger.InfoContext(ctx, "outbound packet connection to ", destination)
	conn, err := h.dialer.DialContext(ctx, N.NetworkUDP, h.serverAddr)
	if err != nil {
		return nil, err
	}
	var packetConn N.NetPacketConn
	if h.method != nil {
		packetConn = h.method.PacketConn(bufio.NewUnbindPacketConn(conn))
	} else {
		packetConn = bufio.NewUnbindPacketConn(conn)
	}
	packetConn = h.protocol.PacketConn(packetConn)
	return &ssrPacketConn{
		NetPacketConn: packetConn,
		rAddr:         h.serverAddr,
		destination:   destination,
	}, nil
}

// destinationAddr serializes the target address in the SOCKS5 wire format used
// by SSR: ATYP + address + port.
func destinationAddr(destination M.Socksaddr) []byte {
	buffer := buf.NewSize(M.SocksaddrSerializer.AddrPortLen(destination))
	defer buffer.Release()
	if err := M.SocksaddrSerializer.WriteAddrPort(buffer, destination); err != nil {
		return nil
	}
	return append([]byte(nil), buffer.Bytes()...)
}

type ssrPacketConn struct {
	N.NetPacketConn
	rAddr       M.Socksaddr
	destination M.Socksaddr
}

func (spc *ssrPacketConn) WriteTo(b []byte, addr net.Addr) (int, error) {
	destination := M.SocksaddrFromNet(addr)
	packet := destinationAddr(destination)
	packet = append(packet, b...)
	_, err := spc.NetPacketConn.WriteTo(packet, spc.rAddr.UDPAddr())
	if err != nil {
		return 0, err
	}
	return len(b), nil
}

func (spc *ssrPacketConn) ReadFrom(b []byte) (int, net.Addr, error) {
	n, _, err := spc.NetPacketConn.ReadFrom(b)
	if err != nil {
		return 0, nil, err
	}
	return splitPacketPayload(b[:n])
}

func (spc *ssrPacketConn) ReadPacket(buffer *buf.Buffer) (M.Socksaddr, error) {
	_, err := spc.NetPacketConn.ReadPacket(buffer)
	if err != nil {
		return M.Socksaddr{}, err
	}
	payload := buffer.Bytes()
	destination, consumed, err := parseSocksAddr(payload)
	if err != nil {
		return M.Socksaddr{}, err
	}
	copy(buffer.Bytes(), payload[consumed:])
	buffer.Truncate(len(payload) - consumed)
	return destination, nil
}

func (spc *ssrPacketConn) WritePacket(buffer *buf.Buffer, destination M.Socksaddr) error {
	defer buffer.Release()
	packet := buf.NewSize(M.SocksaddrSerializer.AddrPortLen(destination) + buffer.Len())
	defer packet.Release()
	if err := M.SocksaddrSerializer.WriteAddrPort(packet, destination); err != nil {
		return err
	}
	packet.Write(buffer.Bytes())
	return spc.NetPacketConn.WritePacket(packet, spc.rAddr)
}

// splitPacketPayload strips the serialized SOCKS address prefix from a decoded
// UDP payload and returns the payload length and source address.
func splitPacketPayload(b []byte) (int, net.Addr, error) {
	destination, consumed, err := parseSocksAddr(b)
	if err != nil {
		return 0, nil, err
	}
	copy(b, b[consumed:])
	return len(b) - consumed, destination.UDPAddr(), nil
}

// parseSocksAddr parses an SSR UDP payload header: the destination serialized
// in the SOCKS5 wire format. It returns the address and the number of
// consumed bytes.
func parseSocksAddr(b []byte) (M.Socksaddr, int, error) {
	if len(b) < 1 {
		return M.Socksaddr{}, 0, E.New("parse addr error")
	}
	switch b[0] {
	case 0x01:
		if len(b) < 7 {
			return M.Socksaddr{}, 0, E.New("parse addr error")
		}
		return M.SocksaddrFrom(netip.AddrFrom4([4]byte(b[1:5])), binary.BigEndian.Uint16(b[5:7])), 7, nil
	case 0x04:
		if len(b) < 19 {
			return M.Socksaddr{}, 0, E.New("parse addr error")
		}
		return M.SocksaddrFrom(netip.AddrFrom16([16]byte(b[1:17])).Unmap(), binary.BigEndian.Uint16(b[17:19])), 19, nil
	case 0x03:
		if len(b) < 4 || len(b) < 4+int(b[1]) {
			return M.Socksaddr{}, 0, E.New("parse addr error")
		}
		return M.ParseSocksaddrHostPort(string(b[2:2+int(b[1])]), binary.BigEndian.Uint16(b[2+int(b[1]):4+int(b[1])])), 4 + int(b[1]), nil
	default:
		return M.Socksaddr{}, 0, E.New("parse addr error")
	}
}

// packetConnAsConn adapts a bound net.PacketConn into a net.Conn for
// DialContext(N.NetworkUDP).
type packetConnAsConn struct {
	net.PacketConn
	destination M.Socksaddr
}

func (c *packetConnAsConn) Read(b []byte) (int, error) {
	n, _, err := c.PacketConn.ReadFrom(b)
	return n, err
}

func (c *packetConnAsConn) Write(b []byte) (int, error) {
	return c.PacketConn.WriteTo(b, c.destination.UDPAddr())
}

func (c *packetConnAsConn) RemoteAddr() net.Addr {
	return c.destination.UDPAddr()
}

func (c *packetConnAsConn) LocalAddr() net.Addr {
	return c.destination.UDPAddr()
}

func (c *packetConnAsConn) SetDeadline(t time.Time) error {
	return c.PacketConn.SetDeadline(t)
}

func (c *packetConnAsConn) SetReadDeadline(t time.Time) error {
	return c.PacketConn.SetReadDeadline(t)
}

func (c *packetConnAsConn) SetWriteDeadline(t time.Time) error {
	return c.PacketConn.SetWriteDeadline(t)
}

var _ adapter.Outbound = (*Outbound)(nil)
