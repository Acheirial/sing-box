//go:build with_gvisor

package zerotier

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"os"
	"sync"
	"time"

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

var _ ipStack = (*stackDevice)(nil)

type stackDevice struct {
	stack         *stack.Stack
	endpoint      *deviceEndpoint
	inet4Address  netip.Addr
	inet6Address  netip.Addr
	localPrefixes []netip.Prefix
	mtu           uint32
	outbound      chan *stack.PacketBuffer
	done          chan struct{}
	closeOnce     sync.Once
}

func newStackDevice(localAddresses []netip.Prefix, mtu uint32) (*stackDevice, error) {
	device := &stackDevice{
		localPrefixes: localAddresses,
		mtu:           mtu,
		outbound:      make(chan *stack.PacketBuffer, 256),
		done:          make(chan struct{}),
	}
	endpoint := &deviceEndpoint{
		mtu:      mtu,
		done:     device.done,
		outbound: device.outbound,
	}
	ipStack, err := tun.NewGVisorStackWithOptions(endpoint, stack.NICOptions{}, true)
	if err != nil {
		return nil, err
	}
	for _, prefix := range localAddresses {
		addr := tun.AddressFromAddr(prefix.Addr())
		protoAddr := tcpip.ProtocolAddress{
			AddressWithPrefix: tcpip.AddressWithPrefix{
				Address:   addr,
				PrefixLen: prefix.Bits(),
			},
		}
		if prefix.Addr().Is4() {
			device.inet4Address = prefix.Addr()
			protoAddr.Protocol = ipv4.ProtocolNumber
		} else {
			device.inet6Address = prefix.Addr()
			protoAddr.Protocol = ipv6.ProtocolNumber
		}
		gErr := ipStack.AddProtocolAddress(tun.DefaultNIC, protoAddr, stack.AddressProperties{})
		if gErr != nil {
			return nil, E.New("parse local address ", protoAddr.AddressWithPrefix, ": ", gErr.String())
		}
	}
	device.stack = ipStack
	device.endpoint = endpoint
	return device, nil
}

func (w *stackDevice) Start() error {
	return nil
}

func (w *stackDevice) File() *os.File {
	return nil
}

func (w *stackDevice) Read(bufs [][]byte, sizes []int, offset int) (count int, err error) {
	select {
	case packet, ok := <-w.outbound:
		if !ok {
			return 0, os.ErrClosed
		}
		defer packet.DecRef()
		var copyN int
		for _, view := range packet.AsSlices() {
			copyN += copy(bufs[0][offset+copyN:], view)
		}
		sizes[0] = copyN
		return 1, nil
	case <-w.done:
		return 0, os.ErrClosed
	}
}

func (w *stackDevice) Write(bufs [][]byte, offset int) (count int, err error) {
	for _, b := range bufs {
		b = b[offset:]
		if len(b) == 0 {
			continue
		}
		var networkProtocol tcpip.NetworkProtocolNumber
		switch header.IPVersion(b) {
		case header.IPv4Version:
			networkProtocol = header.IPv4ProtocolNumber
		case header.IPv6Version:
			networkProtocol = header.IPv6ProtocolNumber
		}
		packetBuffer := stack.NewPacketBuffer(stack.PacketBufferOptions{
			Payload: buffer.MakeWithData(b),
		})
		w.endpoint.dispatcher.DeliverNetworkPacket(networkProtocol, packetBuffer)
		packetBuffer.DecRef()
		count++
	}
	return
}

func (w *stackDevice) Flush() error {
	return nil
}

func (w *stackDevice) MTU() (int, error) {
	return int(w.mtu), nil
}

func (w *stackDevice) Name() (string, error) {
	return "zerotier", nil
}

func (w *stackDevice) Events() <-chan struct{} {
	return w.done
}

func (w *stackDevice) Close() error {
	w.closeOnce.Do(func() {
		close(w.done)
		w.stack.Close()
		for _, endpoint := range w.stack.CleanupEndpoints() {
			endpoint.Abort()
		}
		w.stack.Wait()
	})
	return nil
}

func (w *stackDevice) BatchSize() int {
	return 1
}

var _ stack.LinkEndpoint = (*deviceEndpoint)(nil)

type deviceEndpoint struct {
	mtu        uint32
	done       chan struct{}
	outbound   chan *stack.PacketBuffer
	dispatcher stack.NetworkDispatcher
}

func (ep *deviceEndpoint) MTU() uint32 {
	return ep.mtu
}

func (ep *deviceEndpoint) SetMTU(mtu uint32) {
}

func (ep *deviceEndpoint) MaxHeaderLength() uint16 {
	return 0
}

func (ep *deviceEndpoint) LinkAddress() tcpip.LinkAddress {
	return ""
}

func (ep *deviceEndpoint) SetLinkAddress(addr tcpip.LinkAddress) {
}

func (ep *deviceEndpoint) Capabilities() stack.LinkEndpointCapabilities {
	return stack.CapabilityRXChecksumOffload
}

func (ep *deviceEndpoint) Attach(dispatcher stack.NetworkDispatcher) {
	ep.dispatcher = dispatcher
}

func (ep *deviceEndpoint) IsAttached() bool {
	return ep.dispatcher != nil
}

func (ep *deviceEndpoint) Wait() {
}

func (ep *deviceEndpoint) ARPHardwareType() header.ARPHardwareType {
	return header.ARPHardwareNone
}

func (ep *deviceEndpoint) AddHeader(buffer *stack.PacketBuffer) {
}

func (ep *deviceEndpoint) ParseHeader(ptr *stack.PacketBuffer) bool {
	return true
}

func (ep *deviceEndpoint) WritePackets(list stack.PacketBufferList) (int, tcpip.Error) {
	for _, packetBuffer := range list.AsSlice() {
		packetBuffer.IncRef()
		select {
		case <-ep.done:
			return 0, &tcpip.ErrClosedForSend{}
		case ep.outbound <- packetBuffer:
		}
	}
	return list.Len(), nil
}

func (ep *deviceEndpoint) Close() {
}

func (ep *deviceEndpoint) SetOnCloseAction(f func()) {
}

func (w *stackDevice) dialTCP(ctx context.Context, network string, source, destination netip.AddrPort) (net.Conn, error) {
	bind := tcpip.FullAddress{
		NIC: tun.DefaultNIC,
	}
	if source.Port() != 0 {
		bind.Port = source.Port()
	}
	addr := tcpip.FullAddress{
		NIC:  tun.DefaultNIC,
		Port: destination.Port(),
		Addr: tun.AddressFromAddr(destination.Addr()),
	}
	var networkProtocol tcpip.NetworkProtocolNumber
	if destination.Addr().Is4() {
		if !w.inet4Address.IsValid() {
			return nil, E.New("missing IPv4 local address")
		}
		networkProtocol = header.IPv4ProtocolNumber
		if bind.Port == 0 {
			bind.Addr = tun.AddressFromAddr(w.inet4Address)
		}
	} else {
		if !w.inet6Address.IsValid() {
			return nil, E.New("missing IPv6 local address")
		}
		networkProtocol = header.IPv6ProtocolNumber
		if bind.Port == 0 {
			bind.Addr = tun.AddressFromAddr(w.inet6Address)
		}
	}
	switch N.NetworkName(network) {
	case N.NetworkTCP:
		return dialTCPWithBind(ctx, w.stack, bind, addr, networkProtocol)
	default:
		return nil, E.Extend(N.ErrUnknownNetwork, network)
	}
}

func (w *stackDevice) DialTCP(ctx context.Context, network string, source, destination netip.AddrPort) (net.Conn, error) {
	destination = netip.AddrPortFrom(destination.Addr().Unmap(), destination.Port())
	return w.dialTCP(ctx, network, source, destination)
}

func (w *stackDevice) DialUDP(ctx context.Context, network string, source, destination netip.AddrPort) (net.Conn, error) {
	destination = netip.AddrPortFrom(destination.Addr().Unmap(), destination.Port())
	bind := tcpip.FullAddress{
		NIC: tun.DefaultNIC,
	}
	if source.IsValid() && (source.Addr().IsValid() || source.Port() != 0) {
		bind.Port = source.Port()
	}
	addr := tcpip.FullAddress{
		NIC:  tun.DefaultNIC,
		Port: destination.Port(),
		Addr: tun.AddressFromAddr(destination.Addr()),
	}
	var networkProtocol tcpip.NetworkProtocolNumber
	if destination.Addr().Is4() {
		if !w.inet4Address.IsValid() {
			return nil, E.New("missing IPv4 local address")
		}
		networkProtocol = header.IPv4ProtocolNumber
		if bind.Port == 0 {
			bind.Addr = tun.AddressFromAddr(w.inet4Address)
		}
	} else {
		if !w.inet6Address.IsValid() {
			return nil, E.New("missing IPv6 local address")
		}
		networkProtocol = header.IPv6ProtocolNumber
		if bind.Port == 0 {
			bind.Addr = tun.AddressFromAddr(w.inet6Address)
		}
	}
	switch N.NetworkName(network) {
	case N.NetworkUDP:
		return gonet.DialUDP(w.stack, &bind, &addr, networkProtocol)
	default:
		return nil, E.Extend(N.ErrUnknownNetwork, network)
	}
}

func (w *stackDevice) ListenTCP(ctx context.Context, network string, local netip.AddrPort) (net.Listener, error) {
	bind := tcpip.FullAddress{
		NIC:  tun.DefaultNIC,
		Port: local.Port(),
	}
	if local.Addr().IsValid() && !local.Addr().IsUnspecified() {
		bind.Addr = tun.AddressFromAddr(local.Addr())
	}
	var networkProtocol tcpip.NetworkProtocolNumber
	if local.Addr().Is4() || (!local.Addr().IsValid() && w.inet4Address.IsValid() && !w.inet6Address.IsValid()) {
		if !w.inet4Address.IsValid() {
			return nil, E.New("missing IPv4 local address")
		}
		networkProtocol = header.IPv4ProtocolNumber
	} else {
		if !w.inet6Address.IsValid() {
			return nil, E.New("missing IPv6 local address")
		}
		networkProtocol = header.IPv6ProtocolNumber
	}
	switch N.NetworkName(network) {
	case N.NetworkTCP:
		var waitQueue waiter.Queue
		ep, err := w.stack.NewEndpoint(tcp.ProtocolNumber, networkProtocol, &waitQueue)
		if err != nil {
			return nil, gonet.TranslateNetstackError(err)
		}
		if err = ep.Bind(bind); err != nil {
			ep.Close()
			return nil, gonet.TranslateNetstackError(err)
		}
		if err = ep.Listen(4096); err != nil {
			ep.Close()
			return nil, gonet.TranslateNetstackError(err)
		}
		return gonet.NewTCPListener(w.stack, &waitQueue, ep), nil
	default:
		return nil, E.Extend(N.ErrUnknownNetwork, network)
	}
}

func (w *stackDevice) ListenUDP(ctx context.Context, network string, local netip.AddrPort) (net.PacketConn, error) {
	bind := tcpip.FullAddress{
		NIC: tun.DefaultNIC,
	}
	var networkProtocol tcpip.NetworkProtocolNumber
	if local.Addr().Is4() || (!local.Addr().IsValid() && w.inet4Address.IsValid()) {
		if !w.inet4Address.IsValid() {
			return nil, E.New("missing IPv4 local address")
		}
		networkProtocol = header.IPv4ProtocolNumber
		bind.Addr = tun.AddressFromAddr(w.inet4Address)
	} else {
		if !w.inet6Address.IsValid() {
			return nil, E.New("missing IPv6 local address")
		}
		networkProtocol = header.IPv6ProtocolNumber
		bind.Addr = tun.AddressFromAddr(w.inet6Address)
	}
	switch N.NetworkName(network) {
	case N.NetworkUDP:
		udpConn, err := gonet.DialUDP(w.stack, &bind, nil, networkProtocol)
		if err != nil {
			return nil, err
		}
		return udpConn, nil
	default:
		return nil, E.Extend(N.ErrUnknownNetwork, network)
	}
}

// dialTCPWithBind is adapted from sing-box's WireGuard stack device; it sets
// the same keepalive defaults on outbound connections.
func dialTCPWithBind(ctx context.Context, s *stack.Stack, localAddr, remoteAddr tcpip.FullAddress, network tcpip.NetworkProtocolNumber) (*gonet.TCPConn, error) {
	var waitQueue waiter.Queue
	ep, err := s.NewEndpoint(tcp.ProtocolNumber, network, &waitQueue)
	if err != nil {
		return nil, errors.New(err.String())
	}
	waitEntry, notifyCh := waiter.NewChannelEntry(waiter.WritableEvents)
	waitQueue.EventRegister(&waitEntry)
	defer waitQueue.EventUnregister(&waitEntry)

	select {
	case <-ctx.Done():
		ep.Close()
		return nil, ctx.Err()
	default:
	}

	if localAddr != (tcpip.FullAddress{}) {
		if err = ep.Bind(localAddr); err != nil {
			ep.Close()
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
		return nil, &net.OpError{
			Op:   "connect",
			Net:  "tcp",
			Addr: M.SocksaddrFromNetIP(netip.AddrPortFrom(tun.AddrFromAddress(remoteAddr.Addr), remoteAddr.Port)).TCPAddr(),
			Err:  errors.New(err.String()),
		}
	}

	ep.SocketOptions().SetKeepAlive(true)
	keepAliveIdle := tcpip.KeepaliveIdleOption(15 * time.Second)
	ep.SetSockOpt(&keepAliveIdle)
	keepAliveInterval := tcpip.KeepaliveIntervalOption(15 * time.Second)
	ep.SetSockOpt(&keepAliveInterval)

	return gonet.NewTCPConn(&waitQueue, ep), nil
}
