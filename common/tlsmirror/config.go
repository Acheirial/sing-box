package tlsmirror

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"math/big"
	"net"
	"sync"
	"time"

	boxtls "github.com/sagernet/sing-box/common/tls"
)

// CarrierConfig builds the TLS carrier connection over the injected mirror
// side of the internal pipe. Both the stdlib and uTLS client configs of
// sing-box common/tls satisfy this interface, so carrier fingerprinting and
// ECH follow the outbound's regular TLS options.
type CarrierConfig = boxtls.Config

type ClientConfig struct {
	Config

	ServerName         string
	ForwardAddressHint string
	CarrierConfig      CarrierConfig
	EnrollmentDialer   EnrollmentDialer
}

type ServerConfig = Config

// RecommendedExplicitNonceCipherSuites is the recommended TLS 1.2 cipher suite list for explicit nonce carriers.
var RecommendedExplicitNonceCipherSuites = []uint16{
	156, 157, 158, 159, 160, 161, 162, 163, 164, 165, 166, 167, 168, 169, 170, 171,
	172, 173, 49195, 49196, 49197, 49198, 49199, 49200, 49201, 49202, 49290,
	49291, 49293, 49316, 49317, 49318, 49319, 49320, 49321, 49322, 49323,
	49324, 49325, 49326, 49327, 52392, 52393, 52394, 52395, 52396, 52397,
	52398,
}

type Config struct {
	PrimaryKey                  string
	ExplicitNonceCipherSuites   []uint16
	DeferInstanceDerivedWrite   TimeSpec
	TransportLayerPadding       TransportLayerPadding
	ConnectionEnrolment         *ConnectionEnrolment
	SequenceWatermarkingEnabled bool
	EmbeddedTrafficGenerator    *TrafficGenerator
}

type ConnectionEnrolment struct {
	PrimaryIngressOutbound string
	PrimaryEgressOutbound  string
}

type EnrollmentDialer func(ctx context.Context, network, address string) (net.Conn, error)

type TrafficGenerator struct {
	Steps []TrafficStep
}

type TrafficStep struct {
	Name                         string
	Host                         string
	Path                         string
	Method                       string
	Headers                      []TrafficHeader
	NextStep                     []TrafficTransferCandidate
	ConnectionReady              bool
	ConnectionRecallExit         bool
	WaitTime                     TimeSpec
	H2DoNotWaitForDownloadFinish bool
}

type TrafficHeader struct {
	Name   string
	Value  string
	Values []string
}

type TrafficTransferCandidate struct {
	Weight       int32
	GotoLocation int
}

type TimeSpec struct {
	BaseNanoseconds                    uint64
	UniformRandomMultiplierNanoseconds uint64
}

func (s TimeSpec) Duration() (time.Duration, error) {
	delay := s.BaseNanoseconds
	if s.UniformRandomMultiplierNanoseconds > 0 {
		n, err := rand.Int(rand.Reader, new(big.Int).SetUint64(s.UniformRandomMultiplierNanoseconds))
		if err != nil {
			return 0, err
		}
		delay += n.Uint64()
	}
	return time.Duration(delay), nil
}

type TransportLayerPadding struct {
	Enabled bool
}

func GeneratePrimaryKey() string {
	key := make([]byte, 32)
	_, _ = rand.Read(key)
	return base64.StdEncoding.EncodeToString(key)
}

func DecodePrimaryKey(value string) ([]byte, error) {
	if value == "" {
		return nil, errors.New("missing tlsmirror primary key")
	}
	key, err := base64.StdEncoding.DecodeString(value)
	if err == nil && len(key) == 32 {
		return key, nil
	}
	return nil, errors.New("tlsmirror primary key must be standard base64 and decode to 32 bytes")
}

// pipeDeadline is a minimal copy of net.Pipe deadline handling for conns that
// wrap goroutine pipes instead of real sockets.
type pipeDeadline struct {
	mu     sync.Mutex
	timer  *time.Timer
	cancel chan struct{}
}

func makePipeDeadline() pipeDeadline {
	return pipeDeadline{cancel: make(chan struct{})}
}

func (d *pipeDeadline) Set(t time.Time) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.timer != nil && !d.timer.Stop() {
		<-d.cancel
	}
	d.timer = nil

	closed := isClosedChan(d.cancel)
	if t.IsZero() {
		if closed {
			d.cancel = make(chan struct{})
		}
		return
	}
	if dur := time.Until(t); dur > 0 {
		if closed {
			d.cancel = make(chan struct{})
		}
		d.timer = time.AfterFunc(dur, func() {
			close(d.cancel)
		})
		return
	}
	if !closed {
		close(d.cancel)
	}
}

func (d *pipeDeadline) Wait() chan struct{} {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.cancel
}

func isClosedChan(c <-chan struct{}) bool {
	select {
	case <-c:
		return true
	default:
		return false
	}
}
