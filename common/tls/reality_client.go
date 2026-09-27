//go:build with_utls

package tls

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	mRand "math/rand/v2"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/common/badversion"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/debug"
	E "github.com/sagernet/sing/common/exceptions"
	"github.com/sagernet/sing/common/logger"
	"github.com/sagernet/sing/common/ntp"
	aTLS "github.com/sagernet/sing/common/tls"

	"github.com/cloudflare/circl/sign/mldsa/mldsa65"
	utls "github.com/metacubex/utls"
	"golang.org/x/crypto/hkdf"
	"golang.org/x/net/http2"
)

var _ ConfigCompat = (*RealityClientConfig)(nil)

type RealityClientConfig struct {
	ctx           context.Context
	logger        logger.ContextLogger
	uClient       *UTLSClientConfig
	publicKey     []byte
	shortID       [8]byte
	mldsa65Verify []byte
	spiderX       string
	spiderY       realitySpiderY
}

func NewRealityClient(ctx context.Context, logger logger.ContextLogger, serverAddress string, options option.OutboundTLSOptions) (Config, error) {
	return newRealityClient(ctx, logger, serverAddress, options, false)
}

func newRealityClient(ctx context.Context, logger logger.ContextLogger, serverAddress string, options option.OutboundTLSOptions, allowEmptyServerName bool) (Config, error) {
	if options.UTLS == nil || !options.UTLS.Enabled {
		return nil, E.New("uTLS is required by reality client")
	}
	if options.Spoof != "" || options.SpoofMethod != "" {
		return nil, E.New("spoof is unsupported in reality")
	}

	uClient, err := newUTLSClient(ctx, logger, serverAddress, options, allowEmptyServerName)
	if err != nil {
		return nil, err
	}

	publicKey, err := base64.RawURLEncoding.DecodeString(options.Reality.PublicKey)
	if err != nil {
		return nil, E.Cause(err, "decode public_key")
	}
	if len(publicKey) != 32 {
		return nil, E.New("invalid public_key")
	}
	var shortID [8]byte
	decodedLen, err := hex.Decode(shortID[:], []byte(options.Reality.ShortID))
	if err != nil {
		return nil, E.Cause(err, "decode short_id")
	}
	if decodedLen > 8 {
		return nil, E.New("invalid short_id")
	}

	var mldsa65Verify []byte
	if options.Reality.Mldsa65Verify != "" {
		mldsa65Verify, err = base64.RawURLEncoding.DecodeString(options.Reality.Mldsa65Verify)
		if err != nil {
			return nil, E.Cause(err, "decode mldsa65_verify")
		}
		if len(mldsa65Verify) != 1952 {
			return nil, E.New("invalid mldsa65_verify")
		}
	}

	spiderX, spiderY, err := parseRealitySpiderX(options.Reality.SpiderX)
	if err != nil {
		return nil, E.Cause(err, "parse spider_x")
	}

	var config Config = &RealityClientConfig{
		ctx:           ctx,
		logger:        logger,
		uClient:       uClient.(*UTLSClientConfig),
		publicKey:     publicKey,
		shortID:       shortID,
		mldsa65Verify: mldsa65Verify,
		spiderX:       spiderX,
		spiderY:       spiderY,
	}
	if options.KernelRx || options.KernelTx {
		if !C.IsLinux {
			return nil, E.New("kTLS is only supported on Linux")
		}
		config = &KTLSClientConfig{
			Config:   config,
			logger:   logger,
			kernelTx: options.KernelTx,
			kernelRx: options.KernelRx,
		}
	}
	return config, nil
}

func (e *RealityClientConfig) ServerName() string {
	return e.uClient.ServerName()
}

func (e *RealityClientConfig) SetServerName(serverName string) {
	e.uClient.SetServerName(serverName)
}

func (e *RealityClientConfig) NextProtos() []string {
	return e.uClient.NextProtos()
}

func (e *RealityClientConfig) SetNextProtos(nextProto []string) {
	e.uClient.SetNextProtos(nextProto)
}

func (e *RealityClientConfig) HandshakeTimeout() time.Duration {
	return e.uClient.HandshakeTimeout()
}

func (e *RealityClientConfig) SetHandshakeTimeout(timeout time.Duration) {
	e.uClient.SetHandshakeTimeout(timeout)
}

func (e *RealityClientConfig) STDConfig() (*STDConfig, error) {
	return nil, E.New("unsupported usage for reality")
}

func (e *RealityClientConfig) Client(conn net.Conn) (Conn, error) {
	return ClientHandshake(context.Background(), conn, e)
}

// realityClientVersion returns the REALITY client version sent in the ClientHello
// session ID. Xray-core encodes its own core version into these three bytes so
// that servers can enforce min_client_ver / max_client_ver, so we derive the
// value from the build version instead of the legacy hardcoded 1.8.1.
func realityClientVersion() [3]byte {
	version := badversion.Parse(C.Version)
	clamp := func(value int) byte {
		if value < 0 {
			return 0
		}
		if value > 255 {
			return 255
		}
		return byte(value)
	}
	return [3]byte{clamp(version.Major), clamp(version.Minor), clamp(version.Patch)}
}

// buildClientHello prepares the utls ClientHello for a REALITY handshake without
// sending it, so that the encrypted session ID can be verified and tested.
func (e *RealityClientConfig) buildClientHello(conn net.Conn) (*utls.UConn, *realityVerifier, error) {
	verifier := &realityVerifier{
		serverName:    e.uClient.ServerName(),
		mldsa65Verify: e.mldsa65Verify,
		logger:        e.logger,
	}
	uConfig := e.uClient.config.Clone()
	uConfig.InsecureSkipVerify = true
	uConfig.SessionTicketsDisabled = true
	uConfig.VerifyPeerCertificate = verifier.VerifyPeerCertificate
	uConn := utls.UClient(conn, uConfig, e.uClient.id)
	verifier.UConn = uConn
	err := uConn.BuildHandshakeState()
	if err != nil {
		return nil, nil, err
	}

	if len(uConfig.NextProtos) > 0 {
		for _, extension := range uConn.Extensions {
			if alpnExtension, isALPN := extension.(*utls.ALPNExtension); isALPN {
				alpnExtension.AlpnProtocols = uConfig.NextProtos
				break
			}
		}
	}

	hello := uConn.HandshakeState.Hello
	hello.SessionId = make([]byte, 32)
	copy(hello.Raw[39:], hello.SessionId)

	var nowTime time.Time
	if uConfig.Time != nil {
		nowTime = uConfig.Time()
	} else {
		nowTime = time.Now()
	}
	binary.BigEndian.PutUint64(hello.SessionId, uint64(nowTime.Unix()))

	clientVersion := realityClientVersion()
	copy(hello.SessionId[:3], clientVersion[:])
	binary.BigEndian.PutUint32(hello.SessionId[4:], uint32(time.Now().Unix()))
	copy(hello.SessionId[8:], e.shortID[:])
	if debug.Enabled {
		fmt.Printf("REALITY hello.sessionId[:16]: %v\n", hello.SessionId[:16])
	}
	publicKey, err := ecdh.X25519().NewPublicKey(e.publicKey)
	if err != nil {
		return nil, nil, err
	}
	keyShareKeys := uConn.HandshakeState.State13.KeyShareKeys
	if keyShareKeys == nil {
		return nil, nil, E.New("nil KeyShareKeys")
	}
	ecdheKey := keyShareKeys.Ecdhe
	if ecdheKey == nil {
		ecdheKey = keyShareKeys.MlkemEcdhe
	}
	if ecdheKey == nil {
		return nil, nil, E.New("Current fingerprint ", e.uClient.id.Client, e.uClient.id.Version, " does not support TLS 1.3, REALITY handshake cannot establish.")
	}
	authKey, err := ecdheKey.ECDH(publicKey)
	if err != nil {
		return nil, nil, err
	}
	if authKey == nil {
		return nil, nil, E.New("nil auth_key")
	}
	verifier.authKey = authKey
	_, err = hkdf.New(sha256.New, authKey, hello.Random[:20], []byte("REALITY")).Read(authKey)
	if err != nil {
		return nil, nil, err
	}
	return uConn, verifier, nil
}

func (e *RealityClientConfig) ClientHandshake(ctx context.Context, conn net.Conn) (aTLS.Conn, error) {
	uConn, verifier, err := e.buildClientHello(conn)
	if err != nil {
		return nil, err
	}
	hello := uConn.HandshakeState.Hello
	aesBlock, _ := aes.NewCipher(verifier.authKey)
	aesGcmCipher, _ := cipher.NewGCM(aesBlock)
	aesGcmCipher.Seal(hello.SessionId[:0], hello.Random[20:], hello.SessionId[:16], hello.Raw)
	copy(hello.Raw[39:], hello.SessionId)
	if debug.Enabled {
		fmt.Printf("REALITY hello.sessionId: %v\n", hello.SessionId)
		fmt.Printf("REALITY uConn.AuthKey: %v\n", verifier.authKey)
	}

	err = uConn.HandshakeContext(ctx)
	if err != nil {
		return nil, err
	}

	if debug.Enabled {
		fmt.Printf("REALITY Conn.Verified: %v\n", verifier.verified)
	}

	if !verifier.verified {
		go realityClientFallback(e.ctx, uConn, e.uClient.ServerName(), e.uClient.id, e.spiderX, e.spiderY)
		return nil, E.New("reality verification failed")
	}

	return &realityClientConnWrapper{uConn}, nil
}

func realityClientFallback(ctx context.Context, uConn net.Conn, serverName string, fingerprint utls.ClientHelloID, spiderX string, spiderY realitySpiderY) {
	client := &http.Client{
		Transport: &http2.Transport{
			DialTLSContext: func(ctx context.Context, network, addr string, config *tls.Config) (net.Conn, error) {
				return uConn, nil
			},
			TLSClientConfig: &tls.Config{
				Time:    ntp.TimeFuncFromContext(ctx),
				RootCAs: adapter.RootPoolFromContext(ctx),
			},
		},
	}
	// The connection is intentionally left open: the spider reuses it for every
	// request, mirroring Xray-core.
	newRealitySpider(serverName, spiderX, fingerprint.Client, spiderY).crawl(client.Do)
}

type realitySpiderY [10]int64

// parseRealitySpiderX normalizes spider_x and extracts the SpiderY parameters
// encoded in its query string, mirroring Xray-core.
func parseRealitySpiderX(spiderX string) (string, realitySpiderY, error) {
	var spiderY realitySpiderY
	if spiderX == "" {
		spiderX = "/"
	}
	if spiderX[0] != '/' {
		return "", spiderY, E.New("spider_x must start with /: ", spiderX)
	}
	parsed, err := url.Parse(spiderX)
	if err != nil {
		return "", spiderY, E.Cause(err, "parse spider_x")
	}
	query := parsed.Query()
	parse := func(param string, index int) {
		value := query.Get(param)
		query.Del(param)
		if value == "" {
			return
		}
		bounds := strings.SplitN(value, "-", 2)
		first, _ := strconv.ParseInt(bounds[0], 10, 64)
		spiderY[index] = first
		if len(bounds) == 1 {
			spiderY[index+1] = first
		} else {
			second, _ := strconv.ParseInt(bounds[1], 10, 64)
			spiderY[index+1] = second
		}
	}
	parse("p", 0) // padding
	parse("c", 2) // concurrency
	parse("t", 4) // times
	parse("i", 6) // interval
	parse("r", 8) // return
	parsed.RawQuery = query.Encode()
	return parsed.String(), spiderY, nil
}

type realitySpider struct {
	prefix    string
	userAgent string
	paths     map[string]struct{}
	spiderY   realitySpiderY
}

func newRealitySpider(serverName string, spiderX string, userAgent string, spiderY realitySpiderY) *realitySpider {
	paths := make(map[string]struct{})
	if spiderX != "" {
		paths[spiderX] = struct{}{}
	}
	return &realitySpider{
		prefix:    "https://" + serverName,
		userAgent: userAgent,
		paths:     paths,
		spiderY:   spiderY,
	}
}

var realitySpiderHref = regexp.MustCompile(`href="([/h].*?)"`)

func realityRandBetween(first int64, second int64) int64 {
	if second <= first {
		return first
	}
	return first + mRand.Int64N(second-first+1)
}

func (s *realitySpider) randomPath() string {
	if len(s.paths) == 0 {
		return "/"
	}
	stopAt := mRand.IntN(len(s.paths))
	index := 0
	for path := range s.paths {
		if index == stopAt {
			return path
		}
		index++
	}
	return "/"
}

func (s *realitySpider) paddingLength() int {
	length := int(realityRandBetween(s.spiderY[0], s.spiderY[1]))
	if length <= 0 {
		length = mRand.IntN(32) + 30
	}
	return length
}

func (s *realitySpider) addPaths(body []byte) {
	for _, match := range realitySpiderHref.FindAllSubmatch(body, -1) {
		path := bytes.TrimPrefix(match[1], []byte(s.prefix))
		if !bytes.Contains(path, []byte(".")) {
			s.paths[string(path)] = struct{}{}
		}
	}
}

// crawl walks the fallback server with at least two requests, chaining the
// Referer header, so that a blocked REALITY handshake blends into ordinary
// browsing traffic.
func (s *realitySpider) crawl(do func(*http.Request) (*http.Response, error)) {
	followUps := max(realityRandBetween(s.spiderY[2], s.spiderY[3]), 1)
	repeat := max(realityRandBetween(s.spiderY[4], s.spiderY[5]), 1)
	total := 1 + int(followUps*repeat)
	requestURL := s.prefix + s.randomPath()
	referer := ""
	for i := range total {
		request, err := http.NewRequest(http.MethodGet, requestURL, nil)
		if err != nil {
			return
		}
		request.Header.Set("User-Agent", s.userAgent)
		if referer != "" {
			request.Header.Set("Referer", referer)
		}
		request.AddCookie(&http.Cookie{Name: "padding", Value: strings.Repeat("0", s.paddingLength())})
		response, err := do(request)
		if err != nil {
			return
		}
		body, err := io.ReadAll(response.Body)
		response.Body.Close()
		if err != nil {
			return
		}
		s.addPaths(body)
		referer = request.URL.String()
		requestURL = s.prefix + s.randomPath()
		if i > 0 {
			time.Sleep(time.Duration(realityRandBetween(s.spiderY[6], s.spiderY[7])) * time.Millisecond)
		}
	}
}

func (e *RealityClientConfig) Clone() Config {
	return &RealityClientConfig{
		ctx:           e.ctx,
		logger:        e.logger,
		uClient:       e.uClient.Clone().(*UTLSClientConfig),
		publicKey:     e.publicKey,
		shortID:       e.shortID,
		mldsa65Verify: e.mldsa65Verify,
		spiderX:       e.spiderX,
		spiderY:       e.spiderY,
	}
}

type realityVerifier struct {
	*utls.UConn
	serverName    string
	authKey       []byte
	verified      bool
	mldsa65Verify []byte
	logger        logger.ContextLogger
}

func (c *realityVerifier) VerifyPeerCertificate(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	if c.logger != nil && c.HandshakeState.ServerHello != nil && c.HandshakeState.ServerHello.ServerShare.Group != 0 {
		c.logger.Trace("REALITY: is using X25519MLKEM768 for TLS communication: ", c.HandshakeState.ServerHello.ServerShare.Group == utls.X25519MLKEM768)
		c.logger.Trace("REALITY: is using ML-DSA-65 for cert's extra verification: ", len(c.mldsa65Verify) > 0)
	}
	if debug.Enabled && c.HandshakeState.ServerHello != nil && c.HandshakeState.ServerHello.ServerShare.Group != 0 {
		fmt.Printf("REALITY: is using X25519MLKEM768 for TLS communication: %v\n", c.HandshakeState.ServerHello.ServerShare.Group == utls.X25519MLKEM768)
		fmt.Printf("REALITY: is using ML-DSA-65 for cert's extra verification: %v\n", len(c.mldsa65Verify) > 0)
	}
	p, _ := reflect.TypeFor[utls.Conn]().FieldByName("peerCertificates")
	certs := *(*([]*x509.Certificate))(unsafe.Add(unsafe.Pointer(c.Conn), p.Offset))
	if pub, ok := certs[0].PublicKey.(ed25519.PublicKey); ok {
		h := hmac.New(sha512.New, c.authKey)
		h.Write(pub)
		if bytes.Equal(h.Sum(nil), certs[0].Signature) {
			if len(c.mldsa65Verify) > 0 {
				if c.logger != nil {
					c.logger.Trace("REALITY: verifying certificate with ML-DSA-65")
				}
				if debug.Enabled {
					fmt.Printf("REALITY: verifying certificate with ML-DSA-65\n")
				}
				if len(certs[0].Extensions) > 0 {
					h.Write(c.HandshakeState.Hello.Raw)
					h.Write(c.HandshakeState.ServerHello.Raw)
					verify, err := mldsa65.Scheme().UnmarshalBinaryPublicKey(c.mldsa65Verify)
					if err != nil {
						if c.logger != nil {
							c.logger.Trace("REALITY: failed to unmarshal ML-DSA-65 public key")
						}
						return E.Cause(err, "unmarshal ML-DSA-65 public key")
					}
					if mldsa65.Verify(verify.(*mldsa65.PublicKey), h.Sum(nil), nil, certs[0].Extensions[0].Value) {
						if c.logger != nil {
							c.logger.Trace("REALITY: ML-DSA-65 verification succeeded")
						}
						if debug.Enabled {
							fmt.Printf("REALITY: ML-DSA-65 verification succeeded\n")
						}
						c.verified = true
						return nil
					} else {
						if c.logger != nil {
							c.logger.Trace("REALITY: ML-DSA-65 verification failed")
						}
					}
				} else {
					if c.logger != nil {
						c.logger.Trace("REALITY: certificate has no extensions for ML-DSA-65 signature")
					}
				}
			} else {
				c.verified = true
				return nil
			}
		}
	}
	opts := x509.VerifyOptions{
		DNSName:       c.serverName,
		Intermediates: x509.NewCertPool(),
	}
	for _, cert := range certs[1:] {
		opts.Intermediates.AddCert(cert)
	}
	if _, err := certs[0].Verify(opts); err != nil {
		return err
	}
	return nil
}

type realityClientConnWrapper struct {
	*utls.UConn
}

func (c *realityClientConnWrapper) ConnectionState() tls.ConnectionState {
	state := c.Conn.ConnectionState()
	//nolint:staticcheck
	return tls.ConnectionState{
		Version:                     state.Version,
		HandshakeComplete:           state.HandshakeComplete,
		DidResume:                   state.DidResume,
		CipherSuite:                 state.CipherSuite,
		NegotiatedProtocol:          state.NegotiatedProtocol,
		NegotiatedProtocolIsMutual:  state.NegotiatedProtocolIsMutual,
		ServerName:                  state.ServerName,
		PeerCertificates:            state.PeerCertificates,
		VerifiedChains:              state.VerifiedChains,
		SignedCertificateTimestamps: state.SignedCertificateTimestamps,
		OCSPResponse:                state.OCSPResponse,
		TLSUnique:                   state.TLSUnique,
	}
}

func (c *realityClientConnWrapper) Upstream() any {
	return c.UConn
}

// Due to low implementation quality, the reality server intercepted half close and caused memory leaks.
// We fixed it by calling Close() directly.
func (c *realityClientConnWrapper) CloseWrite() error {
	return c.Close()
}

func (c *realityClientConnWrapper) ReaderReplaceable() bool {
	return true
}

func (c *realityClientConnWrapper) WriterReplaceable() bool {
	return false
}
