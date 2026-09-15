package tlsmirror

import (
	"bufio"
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"math/big"
	"net"
	"testing"
	"time"

	aTLS "github.com/sagernet/sing/common/tls"
)

func testPrimaryKeyFixed(t *testing.T) string {
	t.Helper()
	return GeneratePrimaryKey()
}

func selfSignedCert(t *testing.T) tls.Certificate {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		DNSNames:     []string{"localhost"},
	}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatal(err)
	}
	return cert
}

func tlsListener(t *testing.T, cert tls.Certificate, configure ...func(*tls.Config)) net.Listener {
	t.Helper()
	config := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13,
	}
	for _, configure := range configure {
		configure(config)
	}
	ln, err := tls.Listen("tcp", "127.0.0.1:0", config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	return ln
}

func startTestForwardTLS(t *testing.T, configure ...func(*tls.Config)) string {
	t.Helper()
	ln := tlsListener(t, selfSignedCert(t), configure...)
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				_, _ = io.Copy(io.Discard, conn)
			}()
		}
	}()
	return ln.Addr().String()
}

// stdCarrierConfig adapts a plain crypto/tls config into a CarrierConfig.
type stdCarrierConfig struct {
	config *tls.Config
}

func (c *stdCarrierConfig) ServerName() string {
	return c.config.ServerName
}

func (c *stdCarrierConfig) SetServerName(serverName string) {
	c.config.ServerName = serverName
}

func (c *stdCarrierConfig) NextProtos() []string {
	return c.config.NextProtos
}

func (c *stdCarrierConfig) SetNextProtos(nextProtos []string) {
	c.config.NextProtos = nextProtos
}

func (c *stdCarrierConfig) HandshakeTimeout() time.Duration {
	return 0
}

func (c *stdCarrierConfig) SetHandshakeTimeout(time.Duration) {}

func (c *stdCarrierConfig) STDConfig() (*tls.Config, error) {
	return c.config, nil
}

func (c *stdCarrierConfig) Client(conn net.Conn) (aTLS.Conn, error) {
	return tls.Client(conn, c.config), nil
}

func (c *stdCarrierConfig) Clone() CarrierConfig {
	return &stdCarrierConfig{config: c.config.Clone()}
}

func TestTLSMirrorRoundTrip(t *testing.T) {
	testTLSMirrorRoundTrip(t, Config{}, 0)
}

func TestTLSMirrorRoundTripWithPadding(t *testing.T) {
	testTLSMirrorRoundTrip(t, Config{TransportLayerPadding: TransportLayerPadding{Enabled: true}}, 0)
}

func TestTLSMirrorRoundTripWithWatermark(t *testing.T) {
	testTLSMirrorRoundTrip(t, Config{SequenceWatermarkingEnabled: true}, 0)
}

func TestTLSMirrorRoundTripWithFirstWriteDelay(t *testing.T) {
	testTLSMirrorRoundTrip(t, Config{DeferInstanceDerivedWrite: TimeSpec{BaseNanoseconds: uint64((30 * time.Millisecond).Nanoseconds())}}, 30*time.Millisecond)
}

func TestTLSMirrorRoundTripTLS12ExplicitNonce(t *testing.T) {
	testTLSMirrorRoundTrip(t, Config{ExplicitNonceCipherSuites: RecommendedExplicitNonceCipherSuites}, 0, func(config *tls.Config) {
		config.CipherSuites = []uint16{tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256}
		config.MinVersion = tls.VersionTLS12
		config.MaxVersion = tls.VersionTLS12
	})
}

func TestPaddingRoundTrip(t *testing.T) {
	for _, tc := range []struct {
		name     string
		data     []byte
		pad      int
		expected []byte
	}{
		{name: "no padding", data: []byte("hello"), pad: 0, expected: []byte("hello")},
		{name: "with padding", data: []byte("hello"), pad: 4, expected: []byte("hello")},
		{name: "empty", data: nil, pad: 0, expected: []byte{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			packed := packPadding(append([]byte(nil), tc.data...), tc.pad)
			unpacked, padding := unpackPadding(packed)
			if !bytes.Equal(unpacked, tc.expected) {
				t.Fatalf("expected %q, got %q", tc.expected, unpacked)
			}
			if padding != tc.pad {
				t.Fatalf("expected padding %d, got %d", tc.pad, padding)
			}
		})
	}
}

func TestRecordRoundTrip(t *testing.T) {
	rec := &record{
		recordType: recordTypeApplicationData,
		version:    [2]byte{0x03, 0x03},
		fragment:   []byte("payload"),
	}
	var buf bytes.Buffer
	writer := bufio.NewWriter(&buf)
	if err := writeRecord(writer, rec); err != nil {
		t.Fatal(err)
	}
	parsed, raw, err := readRecord(bufio.NewReader(&buf))
	if err != nil {
		t.Fatal(err)
	}
	if parsed.recordType != rec.recordType || parsed.version != rec.version || !bytes.Equal(parsed.fragment, rec.fragment) {
		t.Fatalf("mismatch: %+v vs %+v", parsed, rec)
	}
	if len(raw) != 5+len(rec.fragment) {
		t.Fatalf("raw length mismatch: %d", len(raw))
	}
}

func TestDecodePrimaryKey(t *testing.T) {
	if _, err := DecodePrimaryKey(""); err == nil {
		t.Fatal("expected error for empty key")
	}
	if _, err := DecodePrimaryKey("not-base64!!!"); err == nil {
		t.Fatal("expected error for invalid base64")
	}
	if _, err := DecodePrimaryKey("AAAA"); err == nil {
		t.Fatal("expected error for wrong key size")
	}
	key := GeneratePrimaryKey()
	decoded, err := DecodePrimaryKey(key)
	if err != nil {
		t.Fatal(err)
	}
	if len(decoded) != 32 {
		t.Fatalf("expected 32-byte key, got %d", len(decoded))
	}
}
func TestNonceGenerators(t *testing.T) {
	nonce := newNonceGenerator()
	first := append([]byte(nil), nonce.Next()...)
	second := nonce.Next()
	if bytes.Equal(first, second) {
		t.Fatal("nonce generator repeated value")
	}
	explicit := explicitNonceGenerator{}
	explicitFirst := append([]byte(nil), explicit.Next()...)
	for i, b := range explicitFirst[:7] {
		if b != 0 {
			t.Fatalf("explicit nonce byte %d should start at zero, got %x", i, b)
		}
	}
	if explicitFirst[7] != 1 {
		t.Fatalf("explicit nonce should start at one, got %x", explicitFirst[7])
	}
	explicitSecond := explicit.Next()
	if explicitSecond[7] != 2 {
		t.Fatalf("explicit nonce should increment, got %x", explicitSecond[7])
	}
}

func testTLSMirrorRoundTrip(t *testing.T, cfg Config, firstWriteDelayAtLeast time.Duration, configureForwardTLS ...func(*tls.Config)) {
	t.Helper()
	cfg.PrimaryKey = testPrimaryKeyFixed(t)

	forwardAddr := startTestForwardTLS(t, configureForwardTLS...)
	serverLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = serverLn.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	serverDone := make(chan error, 1)
	go func() {
		carrier, err := serverLn.Accept()
		if err != nil {
			serverDone <- err
			return
		}
		dialer := net.Dialer{}
		forward, err := dialer.DialContext(ctx, "tcp", forwardAddr)
		if err != nil {
			serverDone <- err
			return
		}
		conn, err := ServeConnReady(ctx, carrier, forward, cfg)
		if err != nil {
			serverDone <- err
			return
		}
		for i := range 8 {
			size := 4 + i*8192
			buf := make([]byte, size)
			if _, err := io.ReadFull(conn, buf); err != nil {
				serverDone <- err
				return
			}
			if !bytes.Equal(buf, bytes.Repeat([]byte{byte(i)}, size)) {
				serverDone <- bytes.ErrTooLarge
				return
			}
			if _, err := conn.Write(bytes.Repeat([]byte{byte(255 - i)}, size)); err != nil {
				serverDone <- err
				return
			}
		}
		serverDone <- nil
	}()

	raw, err := net.Dial("tcp", serverLn.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	carrierConfig := &tls.Config{
		ServerName:         "localhost",
		InsecureSkipVerify: true,
	}
	client, err := Dial(ctx, raw, ClientConfig{
		Config:        cfg,
		CarrierConfig: &stdCarrierConfig{config: carrierConfig},
	})
	if err != nil {
		t.Fatal(err)
	}
	for i := range 8 {
		size := 4 + i*8192
		start := time.Now()
		if _, err := client.Write(bytes.Repeat([]byte{byte(i)}, size)); err != nil {
			t.Fatal(err)
		}
		if i == 0 && firstWriteDelayAtLeast > 0 {
			if elapsed := time.Since(start); elapsed < firstWriteDelayAtLeast {
				t.Fatalf("first write delay too short: %v", elapsed)
			}
		}
		buf := make([]byte, size)
		if _, err := io.ReadFull(client, buf); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(buf, bytes.Repeat([]byte{byte(255 - i)}, size)) {
			t.Fatal("payload mismatch")
		}
	}

	select {
	case err := <-serverDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("server handler timeout")
	}
}
