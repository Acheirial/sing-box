package option

import (
	"encoding/base64"
	"strconv"
	"strings"

	E "github.com/sagernet/sing/common/exceptions"
	"github.com/sagernet/sing/common/json/badoption"
)

type VLESSInboundOptions struct {
	ListenOptions
	Users []VLESSUser `json:"users,omitempty"`
	InboundTLSOptionsContainer
	Multiplex  *InboundMultiplexOptions `json:"multiplex,omitempty"`
	Transport  *V2RayTransportOptions   `json:"transport,omitempty"`
	Decryption string                   `json:"decryption,omitempty"`
}

type VLESSUser struct {
	Name       string                     `json:"name"`
	UUID       string                     `json:"uuid"`
	Flow       string                     `json:"flow,omitempty"`
	Encryption string                     `json:"encryption,omitempty"`
	Testseed   badoption.Listable[uint32] `json:"testseed,omitempty"`
}

type VLESSOutboundOptions struct {
	DialerOptions
	ServerOptions
	UUID     string                     `json:"uuid"`
	Flow     string                     `json:"flow,omitempty"`
	Testseed badoption.Listable[uint32] `json:"testseed,omitempty"`
	Network  NetworkList                `json:"network,omitempty"`
	OutboundTLSOptionsContainer
	Multiplex      *OutboundMultiplexOptions `json:"multiplex,omitempty"`
	Transport      *V2RayTransportOptions    `json:"transport,omitempty"`
	PacketEncoding *string                   `json:"packet_encoding,omitempty"`
	Encryption     string                    `json:"encryption,omitempty"`
}

// VLESSEncryptionConfig is the normalized form of a parsed VLESS
// encryption/decryption string (Xray's "mlkem768x25519plus" scheme),
// shared by the client and server instances.
type VLESSEncryptionConfig struct {
	// Enabled reports whether encryption/decryption is active.
	Enabled bool
	// XorMode is 0 (native), 1 (xorpub) or 2 (random).
	XorMode uint32
	// SecondsFrom/SecondsTo are the server-side session lifetime range.
	SecondsFrom int64
	SecondsTo   int64
	// Seconds is the client-side ticket lifetime (0 = 1-RTT only).
	Seconds uint32
	// Keys holds the decoded X25519 (32 bytes) or ML-KEM-768 key material,
	// in the order given by the configuration string.
	Keys [][]byte
	// Padding is the reassembled padding parameter string.
	Padding string
}

// parseVLESSKeysAndPadding extracts key entries and the reassembled
// padding string from the trailing segments of an encryption string,
// exactly mirroring Xray's infra/conf/vless.go: every segment shorter
// than 20 characters is a padding fragment (its length plus one dot is
// added to the padding prefix), longer segments must base64-decode to
// an accepted key length.
func parseVLESSKeysAndPadding(rest []string, keyLengths [2]int) (keys [][]byte, padding string, err error) {
	paddingLength := 0
	for _, r := range rest {
		if len(r) < 20 {
			paddingLength += len(r) + 1
			continue
		}
		b, _ := base64.RawURLEncoding.DecodeString(r)
		if len(b) != keyLengths[0] && len(b) != keyLengths[1] {
			return nil, "", E.New("invalid key length: ", len(b))
		}
		keys = append(keys, b)
	}
	if paddingLength > 0 {
		joined := strings.Join(rest, ".")
		padding = joined[:paddingLength-1]
	}
	return
}

// ParseDecryption parses a VLESS inbound "decryption" string, mirroring
// Xray infra/conf/vless.go: "none"/"" disables decryption, otherwise the
// format is "mlkem768x25519plus.<mode>.<seconds>[.<key or padding fragment>...]",
// where <seconds> is "N" or "N-M" with an optional trailing "s", and each
// remaining segment is either a base64 RawURL key (32 or 64 bytes) or a
// short padding fragment.
func (o *VLESSInboundOptions) ParseDecryption() (*VLESSEncryptionConfig, error) {
	if o.Decryption == "" || o.Decryption == "none" {
		return &VLESSEncryptionConfig{}, nil
	}
	s := strings.Split(o.Decryption, ".")
	if len(s) < 4 || s[0] != "mlkem768x25519plus" {
		return nil, E.New(`unsupported "decryption": `, o.Decryption)
	}
	config := new(VLESSEncryptionConfig)
	config.Enabled = true
	switch s[1] {
	case "native":
	case "xorpub":
		config.XorMode = 1
	case "random":
		config.XorMode = 2
	default:
		return nil, E.New(`unsupported "decryption": `, o.Decryption)
	}
	t := strings.SplitN(strings.TrimSuffix(s[2], "s"), "-", 2)
	i, err := strconv.Atoi(t[0])
	if err != nil {
		return nil, E.Cause(err, `unsupported "decryption": `, o.Decryption)
	}
	config.SecondsFrom = int64(i)
	if len(t) == 2 {
		i, err := strconv.Atoi(t[1])
		if err != nil {
			return nil, E.Cause(err, `unsupported "decryption": `, o.Decryption)
		}
		config.SecondsTo = int64(i)
	}
	keys, padding, err := parseVLESSKeysAndPadding(s[3:], [2]int{32, 64})
	if err != nil {
		return nil, E.Cause(err, `unsupported "decryption": `, o.Decryption)
	}
	config.Keys = keys
	config.Padding = padding
	return config, nil
}

// ParseEncryption parses a VLESS outbound "encryption" string, mirroring
// Xray infra/conf/vless.go: "none"/"" disables encryption, otherwise the
// format is "mlkem768x25519plus.<mode>.<1rtt|0rtt>[.<key or padding fragment>...]",
// where each remaining segment is either a base64 RawURL key (32 or 1184
// bytes) or a short padding fragment.
func (o *VLESSOutboundOptions) ParseEncryption() (*VLESSEncryptionConfig, error) {
	if o.Encryption == "" || o.Encryption == "none" {
		return &VLESSEncryptionConfig{}, nil
	}
	s := strings.Split(o.Encryption, ".")
	if len(s) < 4 || s[0] != "mlkem768x25519plus" {
		return nil, E.New(`unsupported "encryption": `, o.Encryption)
	}
	config := new(VLESSEncryptionConfig)
	config.Enabled = true
	switch s[1] {
	case "native":
	case "xorpub":
		config.XorMode = 1
	case "random":
		config.XorMode = 2
	default:
		return nil, E.New(`unsupported "encryption": `, o.Encryption)
	}
	switch s[2] {
	case "1rtt":
	case "0rtt":
		config.Seconds = 1
	default:
		return nil, E.New(`unsupported "encryption": `, o.Encryption)
	}
	keys, padding, err := parseVLESSKeysAndPadding(s[3:], [2]int{32, 1184})
	if err != nil {
		return nil, E.Cause(err, `unsupported "encryption": `, o.Encryption)
	}
	config.Keys = keys
	config.Padding = padding
	return config, nil
}
