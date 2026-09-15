//go:build with_gvisor

package masque

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"sync"

	"github.com/sagernet/gvisor/pkg/buffer"
	"github.com/sagernet/gvisor/pkg/tcpip"
	"github.com/sagernet/gvisor/pkg/tcpip/adapters/gonet"
	"github.com/sagernet/gvisor/pkg/tcpip/header"
	"github.com/sagernet/gvisor/pkg/tcpip/network/ipv4"
	"github.com/sagernet/gvisor/pkg/tcpip/network/ipv6"
	"github.com/sagernet/gvisor/pkg/tcpip/stack"
	"github.com/sagernet/gvisor/pkg/tcpip/transport/tcp"
	"github.com/sagernet/gvisor/pkg/waiter"
	"github.com/sagernet/sing-tun"
	E "github.com/sagernet/sing/common/exceptions"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
)

// packetStack is a gVisor based userspace IP stack fed by a MASQUE Connect-IP
// tunnel. It implements the surface needed by the masque outbound: dialing TCP
// and UDP flows into the tunnel, and injecting packets received from the
// tunnel back into the stack.
type PacketStack struct {
	stack        *stack.Stack
	endpoint     *packetEndpoint
	inet4Address netip.Addr
	inet6Address netip.Addr
	closeOnce    sync.Once
	closeDone    chan struct{}
}

func NewPacketStack(localPrefixes []netip.Prefix, mtu uint32) (*PacketStack, error) {
	if len(localPrefixes) == 0 {
		return nil, E.New("missing local address")
	}
	s := &PacketStack{
		closeDone: make(chan struct{}),
		endpoint: &packetEndpoint{
			mtu:  mtu,
			done: make(chan struct{}),
		},
	}
	ipStack, err := tun.NewGVisorStackWithOptions(s.endpoint, stack.NICOptions{}, true)
	if err != nil {
		return nil, err
	}
	for _, prefix := range localPrefixes {
		protoAddr := tcpip.ProtocolAddress{
			AddressWithPrefix: tcpip.AddressWithPrefix{
				Address:   tun.AddressFromAddr(prefix.Addr()),
				PrefixLen: prefix.Bits(),
			},
		}
		if prefix.Addr().Is4() {
			s.inet4Address = prefix.Addr()
			protoAddr.Protocol = ipv4.ProtocolNumber
		} else {
			s.inet6Address = prefix.Addr()
			protoAddr.Protocol = ipv6.ProtocolNumber
		}
		gErr := ipStack.AddProtocolAddress(tun.DefaultNIC, protoAddr, stack.AddressProperties{})
		if gErr != nil {
			return nil, E.New("parse local address ", protoAddr.AddressWithPrefix, ": ", gErr.String())
		}
	}
	s.stack = ipStack
	return s, nil
}

// InjectPacket feeds an IP packet received from the tunnel into the stack.
func (s *PacketStack) InjectPacket(packet []byte) {
	if len(packet) == 0 {
		return
	}
	var networkProtocol tcpip.NetworkProtocolNumber
	switch header.IPVersion(packet) {
	case header.IPv4Version:
		networkProtocol = header.IPv4ProtocolNumber
	case header.IPv6Version:
		networkProtocol = header.IPv6ProtocolNumber
	default:
		return
	}
	packetBuffer := stack.NewPacketBuffer(stack.PacketBufferOptions{
		Payload: buffer.MakeWithData(packet),
	})
	s.endpoint.dispatcher.DeliverNetworkPacket(networkProtocol, packetBuffer)
	packetBuffer.DecRef()
}

// Outbound returns the channel of IP packets the stack wants to send into the
// tunnel.
func (s *PacketStack) Outbound() <-chan []byte {
	return s.endpoint.outbound
}

func (s *PacketStack) DialContext(ctx context.Context, network string, destination M.Socksaddr) (net.Conn, error) {
	addr := tcpip.FullAddress{
		NIC:  tun.DefaultNIC,
		Port: destination.Port,
		Addr: tun.AddressFromAddr(destination.Addr),
	}
	bind := tcpip.FullAddress{
		NIC: tun.DefaultNIC,
	}
	var networkProtocol tcpip.NetworkProtocolNumber
	if destination.IsIPv4() {
		if !s.inet4Address.IsValid() {
			return nil, E.New("missing IPv4 local address")
		}
		networkProtocol = header.IPv4ProtocolNumber
		bind.Addr = tun.AddressFromAddr(s.inet4Address)
	} else {
		if !s.inet6Address.IsValid() {
			return nil, E.New("missing IPv6 local address")
		}
		networkProtocol = header.IPv6ProtocolNumber
		bind.Addr = tun.AddressFromAddr(s.inet6Address)
	}
	switch N.NetworkName(network) {
	case N.NetworkTCP:
		tcpConn, err := dialTCPWithBind(ctx, s.stack, bind, addr, networkProtocol)
		if err != nil {
			return nil, err
		}
		return tcpConn, nil
	case N.NetworkUDP:
		udpConn, err := gonet.DialUDP(s.stack, &bind, &addr, networkProtocol)
		if err != nil {
			return nil, err
		}
		return udpConn, nil
	default:
		return nil, E.Extend(N.ErrUnknownNetwork, network)
	}
}

func (s *PacketStack) ListenPacket(ctx context.Context, destination M.Socksaddr) (net.PacketConn, error) {
	bind := tcpip.FullAddress{
		NIC: tun.DefaultNIC,
	}
	var networkProtocol tcpip.NetworkProtocolNumber
	if destination.IsIPv4() {
		if !s.inet4Address.IsValid() {
			return nil, E.New("missing IPv4 local address")
		}
		networkProtocol = header.IPv4ProtocolNumber
		bind.Addr = tun.AddressFromAddr(s.inet4Address)
	} else {
		if !s.inet6Address.IsValid() {
			return nil, E.New("missing IPv6 local address")
		}
		networkProtocol = header.IPv6ProtocolNumber
		bind.Addr = tun.AddressFromAddr(s.inet6Address)
	}
	udpConn, err := gonet.DialUDP(s.stack, &bind, nil, networkProtocol)
	if err != nil {
		return nil, err
	}
	return udpConn, nil
}

func (s *PacketStack) Close() error {
	s.closeOnce.Do(func() {
		close(s.endpoint.done)
		s.stack.Close()
		for _, endpoint := range s.stack.CleanupEndpoints() {
			endpoint.Abort()
		}
		s.stack.Wait()
		close(s.closeDone)
	})
	return nil
}

func dialTCPWithBind(ctx context.Context, s *stack.Stack, localAddr, remoteAddr tcpip.FullAddress, network tcpip.NetworkProtocolNumber) (*gonet.TCPConn, error) {
	var wq waiter.Queue
	ep, err := s.NewEndpoint(tcp.ProtocolNumber, network, &wq)
	if err != nil {
		return nil, errors.New(err.String())
	}
	waitEntry, notifyCh := waiter.NewChannelEntry(waiter.WritableEvents)
	wq.EventRegister(&waitEntry)
	defer wq.EventUnregister(&waitEntry)

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if localAddr != (tcpip.FullAddress{}) {
		if err = ep.Bind(localAddr); err != nil {
			return nil, errors.New(err.String())
		}
	}

	err = ep.Connect(remoteAddr)
	if _, ok := err.(*tcpip.ErrConnectStarted); ok {
		select {
		case <-ctx.Done():
			ep.Close()
			return nil, ctx.Err()
		case <-notifyCh:
		}
		err = ep.LastError()
	}
	if err != nil {
		ep.Close()
		return nil, errors.New(err.String())
	}
	return gonet.NewTCPConn(&wq, ep), nil
}

var _ stack.LinkEndpoint = (*packetEndpoint)(nil)

type packetEndpoint struct {
	mtu        uint32
	done       chan struct{}
	outbound   chan []byte
	dispatcher stack.NetworkDispatcher
}

func (ep *packetEndpoint) MTU() uint32 {
	return ep.mtu
}

func (ep *packetEndpoint) SetMTU(mtu uint32) {
}

func (ep *packetEndpoint) MaxHeaderLength() uint16 {
	return 0
}

func (ep *packetEndpoint) LinkAddress() tcpip.LinkAddress {
	return ""
}

func (ep *packetEndpoint) SetLinkAddress(addr tcpip.LinkAddress) {
}

func (ep *packetEndpoint) Capabilities() stack.LinkEndpointCapabilities {
	return stack.CapabilityRXChecksumOffload
}

func (ep *packetEndpoint) Attach(dispatcher stack.NetworkDispatcher) {
	ep.dispatcher = dispatcher
}

func (ep *packetEndpoint) IsAttached() bool {
	return ep.dispatcher != nil
}

func (ep *packetEndpoint) Wait() {
}

func (ep *packetEndpoint) ARPHardwareType() header.ARPHardwareType {
	return header.ARPHardwareNone
}

func (ep *packetEndpoint) AddHeader(buffer *stack.PacketBuffer) {
}

func (ep *packetEndpoint) ParseHeader(ptr *stack.PacketBuffer) bool {
	return true
}

func (ep *packetEndpoint) WritePackets(list stack.PacketBufferList) (int, tcpip.Error) {
	for _, packetBuffer := range list.AsSlice() {
		packet := make([]byte, 0, packetBuffer.Size())
		for _, view := range packetBuffer.AsSlices() {
			packet = append(packet, view...)
		}
		select {
		case ep.outbound <- packet:
		case <-ep.done:
			return list.Len(), nil
		}
	}
	return list.Len(), nil
}

func (ep *packetEndpoint) Close() {
}

func (ep *packetEndpoint) SetOnCloseAction(f func()) {
}
