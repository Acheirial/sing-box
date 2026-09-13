package tlsmirror

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
)

func Dial(ctx context.Context, rawConn net.Conn, cfg ClientConfig) (*Conn, error) {
	key, err := DecodePrimaryKey(cfg.PrimaryKey)
	if err != nil {
		return nil, err
	}
	if cfg.ConnectionEnrolment != nil {
		serverID, err := deriveEnrollmentServerIdentifier(key)
		if err != nil {
			_ = rawConn.Close()
			return nil, err
		}
		if IsLoopbackProtectionEnabled(ctx, serverID) {
			_ = rawConn.Close()
			return nil, fmt.Errorf("tlsmirror: loopback protection refused dialing to self")
		}
	}

	tlsSide, mirrorSide := net.Pipe()
	lifetimeCtx := context.Background()
	var hidden *Conn
	mirror := newMirrorConn(lifetimeCtx, mirrorSide, rawConn,
		cfg.Config,
		nil,
		func(rec *record) (bool, error) {
			return hidden.handleInboundRecord(rec)
		},
		nil,
		nil,
	)
	hidden, err = newHiddenConn(lifetimeCtx, mirror, key, false, cfg.Config)
	if err != nil {
		_ = mirror.Close()
		return nil, err
	}
	mirror.onC2SMessageTx = hidden.handleOutboundRecordTx
	mirror.start()

	if cfg.CarrierConfig == nil {
		_ = hidden.Close()
		return nil, errors.New("tlsmirror: missing carrier tls config")
	}
	carrierTLS, err := cfg.CarrierConfig.Client(tlsSide)
	if err != nil {
		_ = hidden.Close()
		return nil, err
	}
	if err := carrierTLS.HandshakeContext(ctx); err != nil {
		_ = hidden.Close()
		return nil, fmt.Errorf("%w: %w", errCarrierHandshake, err)
	}
	carrierALPN := carrierTLS.ConnectionState().NegotiatedProtocol

	ready := make(chan struct{})
	recall := make(chan struct{})
	var recallOnce sync.Once
	hidden.recallTrafficGenerator = func() {
		recallOnce.Do(func() {
			close(recall)
		})
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		runTrafficGenerator(hidden.ctx, carrierTLS, cfg.EmbeddedTrafficGenerator, carrierALPN, func() {
			close(ready)
		}, recall)
	}()
	if trafficGeneratorWaitsForReady(cfg.EmbeddedTrafficGenerator) {
		select {
		case <-ready:
		case <-done:
			_ = hidden.Close()
			return nil, fmt.Errorf("tlsmirror: carrier traffic generator exited before ready")
		case <-hidden.ctx.Done():
			return nil, hidden.ctx.Err()
		case <-ctx.Done():
			_ = hidden.Close()
			return nil, ctx.Err()
		}
	}
	if cfg.ConnectionEnrolment != nil && !isConnectionEnrollmentBypassed(ctx) {
		if err := hidden.verifyConnectionEnrollment(ctx, cfg); err != nil {
			_ = hidden.Close()
			return nil, err
		}
	}
	return hidden, nil
}
