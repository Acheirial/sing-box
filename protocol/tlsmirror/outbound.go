package tlsmirror

import (
	"context"
	"net"
	"os"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/adapter/outbound"
	"github.com/sagernet/sing-box/common/dialer"
	boxtls "github.com/sagernet/sing-box/common/tls"
	"github.com/sagernet/sing-box/common/tlsmirror"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common"
	E "github.com/sagernet/sing/common/exceptions"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
	"github.com/sagernet/sing/service"
)

func RegisterOutbound(registry *outbound.Registry) {
	outbound.Register[option.TLSMirrorOutboundOptions](registry, C.TypeTLSMirror, NewOutbound)
}

type Outbound struct {
	outbound.Adapter
	ctx                  context.Context
	logger               log.ContextLogger
	dialer               N.Dialer
	serverAddr           M.Socksaddr
	tlsConfig            boxtls.Config
	clientConfig         tlsmirror.ClientConfig
	outboundManager      adapter.OutboundManager
	enrolmentOutboundTag string
}

func NewOutbound(ctx context.Context, router adapter.Router, logger log.ContextLogger, tag string, options option.TLSMirrorOutboundOptions) (adapter.Outbound, error) {
	if _, err := tlsmirror.DecodePrimaryKey(options.PrimaryKey); err != nil {
		return nil, err
	}
	if options.TLS == nil || !options.TLS.Enabled {
		return nil, C.ErrTLSRequired
	}

	var dependencies []string
	if options.DialerOptions.Detour != "" {
		dependencies = append(dependencies, options.DialerOptions.Detour)
	}
	var enrolmentOutboundTag string
	if options.ConnectionEnrolment != nil {
		enrolmentOutboundTag = options.ConnectionEnrolment.PrimaryEgressOutbound
		if enrolmentOutboundTag == "" {
			enrolmentOutboundTag = options.ConnectionEnrolment.PrimaryIngressOutbound
		}
		if enrolmentOutboundTag == "" {
			return nil, E.New("connection_enrolment requires primary_ingress_outbound or primary_egress_outbound")
		}
		if enrolmentOutboundTag == tag {
			return nil, E.New("connection_enrolment outbound cannot reference itself")
		}
		dependencies = append(dependencies, enrolmentOutboundTag)
	}

	tlsConfig, err := boxtls.NewClient(ctx, logger, options.Server, common.PtrValueOrDefault(options.TLS))
	if err != nil {
		return nil, err
	}
	if tlsConfig == nil {
		return nil, C.ErrTLSRequired
	}

	outboundDialer, err := dialer.NewWithOptions(dialer.Options{
		Context:        ctx,
		Options:        options.DialerOptions,
		RemoteIsDomain: options.ServerIsDomain(),
	})
	if err != nil {
		return nil, err
	}

	clientConfig, err := buildClientConfig(options)
	if err != nil {
		return nil, err
	}
	if options.ConnectionEnrolment != nil {
		clientConfig.ConnectionEnrolment = &tlsmirror.ConnectionEnrolment{
			PrimaryIngressOutbound: options.ConnectionEnrolment.PrimaryIngressOutbound,
			PrimaryEgressOutbound:  options.ConnectionEnrolment.PrimaryEgressOutbound,
		}
	}

	outbound := &Outbound{
		Adapter:              outbound.NewAdapter(C.TypeTLSMirror, tag, []string{N.NetworkTCP}, dependencies),
		ctx:                  ctx,
		logger:               logger,
		dialer:               outboundDialer,
		serverAddr:           options.ServerOptions.Build(),
		tlsConfig:            tlsConfig,
		clientConfig:         clientConfig,
		outboundManager:      service.FromContext[adapter.OutboundManager](ctx),
		enrolmentOutboundTag: enrolmentOutboundTag,
	}
	if options.ConnectionEnrolment != nil {
		outbound.clientConfig.EnrollmentDialer = outbound.dialEnrolment
	}
	return outbound, nil
}

func buildClientConfig(options option.TLSMirrorOutboundOptions) (tlsmirror.ClientConfig, error) {
	config := tlsmirror.Config{
		PrimaryKey:                  options.PrimaryKey,
		SequenceWatermarkingEnabled: options.SequenceWatermarkingEnabled,
		TransportLayerPadding: tlsmirror.TransportLayerPadding{
			Enabled: options.TransportLayerPadding.Enabled,
		},
		DeferInstanceDerivedWrite: buildTimeSpec(options.DeferInstanceDerivedWriteTime),
	}
	if len(options.ExplicitNonceCipherSuites) > 0 {
		config.ExplicitNonceCipherSuites = append(config.ExplicitNonceCipherSuites, options.ExplicitNonceCipherSuites...)
	}
	if len(options.EmbeddedTrafficGenerator.Steps) > 0 {
		steps := make([]tlsmirror.TrafficStep, 0, len(options.EmbeddedTrafficGenerator.Steps))
		for _, step := range options.EmbeddedTrafficGenerator.Steps {
			steps = append(steps, buildTrafficStep(step))
		}
		config.EmbeddedTrafficGenerator = &tlsmirror.TrafficGenerator{Steps: steps}
	}
	return tlsmirror.ClientConfig{Config: config}, nil
}

func buildTimeSpec(spec option.TLSMirrorTimeSpec) tlsmirror.TimeSpec {
	return tlsmirror.TimeSpec{
		BaseNanoseconds:                    spec.BaseNanoseconds,
		UniformRandomMultiplierNanoseconds: spec.UniformRandomMultiplierNanoseconds,
	}
}

func buildTrafficStep(step option.TLSMirrorTrafficStep) tlsmirror.TrafficStep {
	headers := make([]tlsmirror.TrafficHeader, 0, len(step.Headers))
	for _, header := range step.Headers {
		headers = append(headers, tlsmirror.TrafficHeader{
			Name:   header.Name,
			Value:  header.Value,
			Values: header.Values,
		})
	}
	nextStep := make([]tlsmirror.TrafficTransferCandidate, 0, len(step.NextStep))
	for _, candidate := range step.NextStep {
		nextStep = append(nextStep, tlsmirror.TrafficTransferCandidate{
			Weight:       candidate.Weight,
			GotoLocation: candidate.GotoLocation,
		})
	}
	return tlsmirror.TrafficStep{
		Name:                         step.Name,
		Host:                         step.Host,
		Path:                         step.Path,
		Method:                       step.Method,
		Headers:                      headers,
		NextStep:                     nextStep,
		ConnectionReady:              step.ConnectionReady,
		ConnectionRecallExit:         step.ConnectionRecallExit,
		WaitTime:                     buildTimeSpec(step.WaitTime),
		H2DoNotWaitForDownloadFinish: step.H2DoNotWaitForDownloadFinish,
	}
}

func (h *Outbound) dialEnrolment(ctx context.Context, network, address string) (net.Conn, error) {
	detour, loaded := h.outboundManager.Outbound(h.enrolmentOutboundTag)
	if !loaded {
		return nil, E.New("enrolment outbound not found: ", h.enrolmentOutboundTag)
	}
	return detour.DialContext(ctx, network, M.ParseSocksaddr(address))
}

func (h *Outbound) dialOut(ctx context.Context) (net.Conn, error) {
	conn, err := h.dialer.DialContext(ctx, N.NetworkTCP, h.serverAddr)
	if err != nil {
		return nil, err
	}
	tlsConn, err := boxtls.ClientHandshake(ctx, conn, h.tlsConfig)
	if err != nil {
		_ = conn.Close()
		return nil, E.Cause(err, "carrier tls handshake")
	}
	hiddenConn, err := tlsmirror.Dial(ctx, tlsConn, h.clientConfig)
	if err != nil {
		_ = tlsConn.Close()
		return nil, err
	}
	return hiddenConn, nil
}

func (h *Outbound) DialContext(ctx context.Context, network string, destination M.Socksaddr) (net.Conn, error) {
	if N.NetworkName(network) != N.NetworkTCP {
		return nil, os.ErrInvalid
	}
	ctx, metadata := adapter.ExtendContext(ctx)
	metadata.Outbound = h.Tag()
	metadata.Destination = destination
	h.logger.InfoContext(ctx, "outbound connection to ", destination)
	return h.dialOut(ctx)
}

func (h *Outbound) ListenPacket(ctx context.Context, destination M.Socksaddr) (net.PacketConn, error) {
	return nil, os.ErrInvalid
}
