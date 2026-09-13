package cipher

import (
	"net"
	"sync"

	"github.com/sagernet/sing/common/buf"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
)

var packetBufferPool = sync.Pool{
	New: func() any {
		return make([]byte, maxPacketSize)
	},
}

// PacketConn wraps an N.NetPacketConn with stream cipher encryption/decryption.
type PacketConn struct {
	N.NetPacketConn
	Cipher
}

// NewPacketConn wraps an N.NetPacketConn with stream cipher encryption/decryption.
func NewPacketConn(c N.NetPacketConn, ciph Cipher) *PacketConn {
	return &PacketConn{NetPacketConn: c, Cipher: ciph}
}

func (c *PacketConn) WriteTo(b []byte, addr net.Addr) (int, error) {
	buf := packetBufferPool.Get().([]byte)
	defer packetBufferPool.Put(buf)
	buf, err := Pack(buf, b, c.Cipher)
	if err != nil {
		return 0, err
	}
	_, err = c.NetPacketConn.WriteTo(buf, addr)
	return len(b), err
}

func (c *PacketConn) ReadFrom(b []byte) (int, net.Addr, error) {
	n, addr, err := c.NetPacketConn.ReadFrom(b)
	if err != nil {
		return n, addr, err
	}
	bb, err := UnpackInplace(b[:n], c.Cipher)
	if err != nil {
		return n, addr, err
	}
	copy(b, bb)
	return len(bb), addr, nil
}

func (c *PacketConn) WritePacket(buffer *buf.Buffer, destination M.Socksaddr) error {
	defer buffer.Release()
	packed := buf.NewSize(c.IVSize() + buffer.Len())
	defer packed.Release()
	_, err := Pack(packed.Extend(c.IVSize()+buffer.Len()), buffer.Bytes(), c.Cipher)
	if err != nil {
		return err
	}
	return c.NetPacketConn.WritePacket(packed, destination)
}

func (c *PacketConn) ReadPacket(buffer *buf.Buffer) (M.Socksaddr, error) {
	destination, err := c.NetPacketConn.ReadPacket(buffer)
	if err != nil {
		return M.Socksaddr{}, err
	}
	payload, err := UnpackInplace(buffer.Bytes(), c.Cipher)
	if err != nil {
		return M.Socksaddr{}, err
	}
	copy(buffer.Bytes(), payload)
	buffer.Truncate(len(payload))
	return destination, nil
}
