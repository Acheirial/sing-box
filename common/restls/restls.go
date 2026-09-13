package restls

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/sagernet/sing-box/adapter"

	tls "github.com/metacubex/restls-client-go"
)

const (
	Mode string = "restls"
)

type Restls struct {
	*tls.UConn
}

func (r *Restls) Upstream() any {
	return r.UConn.NetConn()
}

type Config = tls.Config

func NewRestlsConfig(serverName, password, versionHint, restlsScript, clientID string) (*Config, error) {
	config, err := tls.NewRestlsConfig(serverName, password, versionHint, restlsScript, clientID)
	if err != nil {
		return nil, err
	}
	config.RootCAs = adapter.RootPoolFromContext(context.Background())
	config.Time = time.Now
	return config, nil
}

func SetFingerprint(config *Config, fingerprint string, nameCertVerify string) (err error) {
	verifier, err := NewFingerprintVerifier(fingerprint, config.Time)
	if err != nil {
		return err
	}
	config.InsecureSkipVerify = true
	config.VerifyConnection = func(state tls.ConnectionState) error {
		serverName := state.ServerName
		if nameCertVerify != "" {
			serverName = nameCertVerify
		}
		return verifier(state.PeerCertificates, serverName)
	}
	return nil
}

func SetNameCertVerify(config *Config, dnsName string) {
	verifier := NewNameCertVerifier(dnsName, config.RootCAs, config.Time)
	config.InsecureSkipVerify = true
	config.VerifyConnection = func(state tls.ConnectionState) error {
		return verifier(state.PeerCertificates)
	}
}

// NewFingerprintVerifier returns a function that verifies whether a certificate's
// SHA-256 fingerprint matches the given one.
func NewFingerprintVerifier(fingerprint string, now func() time.Time) (func(certs []*x509.Certificate, serverName string) error, error) {
	fingerprint = strings.TrimSpace(strings.Replace(fingerprint, ":", "", -1))
	fpByte, err := hex.DecodeString(fingerprint)
	if err != nil {
		return nil, fmt.Errorf("fingerprint string decode error: %w", err)
	}
	if len(fpByte) != 32 {
		return nil, fmt.Errorf("fingerprint string length error,need sha256 fingerprint")
	}
	return func(certs []*x509.Certificate, serverName string) error {
		// ssl pining
		for i, cert := range certs {
			hash := sha256.Sum256(cert.Raw)
			if bytes.Equal(fpByte, hash[:]) {
				if i > 0 {
					// When the fingerprint matches a non-leaf certificate,
					// the certificate chain validity is verified using the certificate as the trusted root certificate.
					opts := x509.VerifyOptions{
						Roots:         x509.NewCertPool(),
						Intermediates: x509.NewCertPool(),
						DNSName:       serverName,
					}
					if now != nil {
						opts.CurrentTime = now()
					}
					opts.Roots.AddCert(certs[i])
					for _, cert := range certs[1 : i+1] { // stop at i
						opts.Intermediates.AddCert(cert)
					}
					_, err := certs[0].Verify(opts)
					return err
				}
				return nil
			}
		}
		return errors.New("tls: no certificate matches the given fingerprint")
	}, nil
}

// NewNameCertVerifier returns a verifier for a certificate chain and an explicit DNSName.
func NewNameCertVerifier(dnsName string, roots *x509.CertPool, now func() time.Time) func([]*x509.Certificate) error {
	return func(certificates []*x509.Certificate) error {
		if len(certificates) == 0 {
			return errors.New("tls: no peer certificates")
		}
		intermediates := x509.NewCertPool()
		for _, certificate := range certificates[1:] {
			intermediates.AddCert(certificate)
		}
		verifyOptions := x509.VerifyOptions{
			Roots:         roots,
			Intermediates: intermediates,
			DNSName:       dnsName,
		}
		if now != nil {
			verifyOptions.CurrentTime = now()
		}
		_, err := certificates[0].Verify(verifyOptions)
		return err
	}
}

// NewRestls return a Restls Connection
func NewRestls(ctx context.Context, conn net.Conn, config *Config) (net.Conn, error) {
	clientHellowID := tls.HelloChrome_Auto
	if config != nil {
		clientIDPtr := config.ClientID.Load()
		if clientIDPtr != nil {
			clientHellowID = *clientIDPtr
		}
		config = config.Clone() // avoid race condition in HandshakeContext
	}
	restls := &Restls{
		UConn: tls.UClient(conn, config, clientHellowID),
	}
	if err := restls.HandshakeContext(ctx); err != nil {
		return nil, err
	}
	return restls, nil
}
