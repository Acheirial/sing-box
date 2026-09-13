package protocol

import (
	"bytes"
	"net"

	"github.com/sagernet/sing/common/buf"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
)

// PacketConn is a protocol-wrapped N.NetPacketConn.
type PacketConn struct {
	N.NetPacketConn
	Protocol
}

func (c *PacketConn) WriteTo(b []byte, addr net.Addr) (int, error) {
	packet := bytes.Buffer{}
	err := c.EncodePacket(&packet, b)
	if err != nil {
		return 0, err
	}
	_, err = c.NetPacketConn.WriteTo(packet.Bytes(), addr)
	return len(b), err
}

func (c *PacketConn) ReadFrom(b []byte) (int, net.Addr, error) {
	n, addr, err := c.NetPacketConn.ReadFrom(b)
	if err != nil {
		return n, addr, err
	}
	decoded, err := c.DecodePacket(b[:n])
	if err != nil {
		return n, addr, err
	}
	copy(b, decoded)
	return len(decoded), addr, nil
}

func (c *PacketConn) WritePacket(buffer *buf.Buffer, destination M.Socksaddr) error {
	defer buffer.Release()
	packet := bytes.Buffer{}
	err := c.EncodePacket(&packet, buffer.Bytes())
	if err != nil {
		return err
	}
	packed := buf.As(packet.Bytes())
	return c.NetPacketConn.WritePacket(packed, destination)
}

func (c *PacketConn) ReadPacket(buffer *buf.Buffer) (M.Socksaddr, error) {
	destination, err := c.NetPacketConn.ReadPacket(buffer)
	if err != nil {
		return M.Socksaddr{}, err
	}
	decoded, err := c.DecodePacket(buffer.Bytes())
	if err != nil {
		return M.Socksaddr{}, err
	}
	copy(buffer.Bytes(), decoded)
	buffer.Truncate(len(decoded))
	return destination, nil
}
