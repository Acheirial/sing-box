// Package shadowquic is a port of mihomo's ShadowQUIC client transport for
// sagernet/quic-go.
package shadowquic

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"

	M "github.com/sagernet/sing/common/metadata"
)

const (
	CommandConnect           byte = 0x01
	CommandBind              byte = 0x02
	CommandAssociateDatagram byte = 0x03
	CommandAssociateStream   byte = 0x04
	CommandAuthenticate      byte = 0x05
	CommandExtension         byte = 0xff
)

const maxUDPPacketSize = 0xffff

var (
	errInvalidAddress  = errors.New("shadowquic: invalid address")
	errInvalidDatagram = errors.New("shadowquic: invalid datagram")
	errPacketTooLarge  = errors.New("shadowquic: packet too large")
)

// serializer encodes addresses in the SOCKS5 wire format used by the
// ShadowQUIC protocol: family(1) | addr | port(2), with family 1 = IPv4,
// 4 = IPv6 and 3 = domain (with a one byte length prefix).
var serializer = M.NewSerializer(
	M.AddressFamilyByte(0x01, M.AddressFamilyIPv4),
	M.AddressFamilyByte(0x04, M.AddressFamilyIPv6),
	M.AddressFamilyByte(0x03, M.AddressFamilyFqdn),
)

// socksAddr is a SOCKS5 encoded address: family(1) | address | port(2),
// with domains encoded as length(1) | host.
type socksAddr []byte

// MetadataAddr encodes a destination as a SOCKS5 address.
func MetadataAddr(destination M.Socksaddr) (socksAddr, error) {
	buffer := make([]byte, 0, serializer.AddrPortLen(destination))
	writer := &byteSliceWriter{buffer}
	err := serializer.WriteAddrPort(writer, destination)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", errInvalidAddress, destination)
	}
	return socksAddr(writer.buffer), nil
}

// UnspecifiedAddr returns the wildcard IPv4 address with port 0.
func UnspecifiedAddr() socksAddr {
	return socksAddr([]byte{0x01, 0, 0, 0, 0, 0, 0})
}

func parseAddr(r io.Reader) (socksAddr, error) {
	destination, err := serializer.ReadAddrPort(r)
	if err != nil {
		return nil, err
	}
	addr, err := MetadataAddr(destination)
	if err != nil {
		return nil, err
	}
	return addr, nil
}

// Destination decodes the address back to a sing Socksaddr.
func (a socksAddr) Destination() (M.Socksaddr, error) {
	if len(a) < 1 {
		return M.Socksaddr{}, errInvalidAddress
	}
	reader := &byteSliceReader{data: a}
	destination, err := serializer.ReadAddrPort(reader)
	if err != nil {
		return M.Socksaddr{}, err
	}
	return destination, nil
}

func (a socksAddr) String() string {
	destination, err := a.Destination()
	if err != nil {
		return ""
	}
	return destination.String()
}

// AddrFromNetAddr converts a net.Addr into the wire address form.
func AddrFromNetAddr(addr net.Addr) (socksAddr, error) {
	if addr == nil {
		return nil, errInvalidAddress
	}
	destination := M.SocksaddrFromNet(addr)
	if !destination.IsValid() && destination.Port == 0 && destination.Addr.IsValid() {
		destination.Port = 0
	}
	return MetadataAddr(destination)
}

// AddrToNetAddr converts the wire address back to a net.Addr.
func AddrToNetAddr(addr socksAddr) net.Addr {
	destination, err := addr.Destination()
	if err != nil {
		return socksNetAddr{addr: addr}
	}
	return M.SocksaddrFromNetIP(destination.AddrPort()).UDPAddr()
}

func WriteRequest(w io.Writer, command byte, addr socksAddr) error {
	if addr == nil {
		return errInvalidAddress
	}
	if _, err := w.Write([]byte{command}); err != nil {
		return err
	}
	_, err := w.Write(addr)
	return err
}

func ReadCommand(r io.Reader) (byte, error) {
	var buf [1]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return 0, err
	}
	return buf[0], nil
}

func ReadRequestAddr(r io.Reader) (socksAddr, error) {
	return parseAddr(r)
}

func ReadRequest(r io.Reader) (byte, socksAddr, error) {
	command, err := ReadCommand(r)
	if err != nil {
		return 0, nil, err
	}
	addr, err := ReadRequestAddr(r)
	if err != nil {
		return 0, nil, err
	}
	return command, addr, nil
}

func WriteUDPControl(w io.Writer, addr socksAddr, id uint16) error {
	if addr == nil {
		return errInvalidAddress
	}
	if _, err := w.Write(addr); err != nil {
		return err
	}
	var buf [2]byte
	binary.BigEndian.PutUint16(buf[:], id)
	_, err := w.Write(buf[:])
	return err
}

func ReadUDPControl(r io.Reader) (socksAddr, uint16, error) {
	addr, err := parseAddr(r)
	if err != nil {
		return nil, 0, err
	}
	id, err := ReadUint16(r)
	if err != nil {
		return nil, 0, err
	}
	return addr, id, nil
}

func EncodeDatagram(id uint16, payload []byte) ([]byte, error) {
	if len(payload) > maxUDPPacketSize {
		return nil, errPacketTooLarge
	}
	packet := make([]byte, 2+len(payload))
	binary.BigEndian.PutUint16(packet[:2], id)
	copy(packet[2:], payload)
	return packet, nil
}

func DecodeDatagram(packet []byte) (uint16, []byte, error) {
	if len(packet) < 2 {
		return 0, nil, errInvalidDatagram
	}
	return binary.BigEndian.Uint16(packet[:2]), packet[2:], nil
}

func WritePacketStreamHeader(w io.Writer, id uint16) error {
	var buf [2]byte
	binary.BigEndian.PutUint16(buf[:], id)
	_, err := w.Write(buf[:])
	return err
}

func WritePacketStreamPayload(w io.Writer, payload []byte) error {
	if len(payload) > maxUDPPacketSize {
		return errPacketTooLarge
	}
	var buf [2]byte
	binary.BigEndian.PutUint16(buf[:], uint16(len(payload)))
	if _, err := w.Write(buf[:]); err != nil {
		return err
	}
	_, err := w.Write(payload)
	return err
}

func ReadUint16(r io.Reader) (uint16, error) {
	var buf [2]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint16(buf[:]), nil
}

type socksNetAddr struct {
	addr socksAddr
}

func (a socksNetAddr) Network() string {
	return "udp"
}

func (a socksNetAddr) String() string {
	return a.addr.String()
}

type byteSliceWriter struct {
	buffer []byte
}

func (w *byteSliceWriter) Write(p []byte) (int, error) {
	w.buffer = append(w.buffer, p...)
	return len(p), nil
}

type byteSliceReader struct {
	data   []byte
	offset int
}

func (r *byteSliceReader) Read(p []byte) (int, error) {
	if r.offset >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.offset:])
	r.offset += n
	return n, nil
}
