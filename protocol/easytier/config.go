//go:build with_easytier

package easytier

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

const defaultListener = "tcp://0.0.0.0:11010"

// Peer is one EasyTier [[peer]] table after URI query parameters are extracted.
type Peer struct {
	URI           string
	PeerPublicKey string
}

func listeners(noListener *bool, listeners []string) []string {
	if noListener != nil && *noListener {
		return []string{}
	}
	if len(listeners) > 0 {
		return listeners
	}
	if noListener != nil && !*noListener {
		return []string{defaultListener}
	}
	return []string{}
}

func parsedPeers(peers []string) ([]Peer, error) {
	out := make([]Peer, 0, len(peers))
	for i, raw := range peers {
		peer, err := parsePeerURI(raw)
		if err != nil {
			return nil, fmt.Errorf("peers[%d]: %w", i, err)
		}
		out = append(out, peer)
	}
	return out, nil
}

func parsePeerURI(raw string) (Peer, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Peer{}, fmt.Errorf("uri is required")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return Peer{}, fmt.Errorf("invalid uri: %w", err)
	}
	if u.RawQuery == "" {
		return Peer{URI: raw}, nil
	}
	var kept []string
	key := ""
	stripped := false
	for _, pair := range strings.Split(u.RawQuery, "&") {
		if pair == "" {
			continue
		}
		name, value, _ := strings.Cut(pair, "=")
		decodedName, err := url.PathUnescape(name)
		if err != nil {
			return Peer{}, fmt.Errorf("invalid uri query: %w", err)
		}
		switch decodedName {
		case "peer-public-key", "peer_public_key":
			decodedValue, err := url.PathUnescape(value)
			if err != nil {
				return Peer{}, fmt.Errorf("invalid peer-public-key: %w", err)
			}
			key = decodedValue
			stripped = true
		default:
			kept = append(kept, pair)
		}
	}
	if !stripped {
		return Peer{URI: raw}, nil
	}
	u.RawQuery = strings.Join(kept, "&")
	return Peer{URI: u.String(), PeerPublicKey: key}, nil
}

func hasSecureModeMaterial(localPrivateKey, localPublicKey string, peers []string) bool {
	if localPrivateKey != "" || localPublicKey != "" {
		return true
	}
	parsed, err := parsedPeers(peers)
	if err != nil {
		return false
	}
	for _, peer := range parsed {
		if peer.PeerPublicKey != "" {
			return true
		}
	}
	return false
}

func secureModeEnabled(secureMode *bool, localPrivateKey, localPublicKey string, peers []string) bool {
	if secureMode != nil {
		return *secureMode
	}
	return hasSecureModeMaterial(localPrivateKey, localPublicKey, peers)
}

func writeOptionalBoolField(encoded *strings.Builder, name string, value *bool) {
	if value != nil {
		writeTOMLBoolField(encoded, name, *value)
	}
}

func writeTOMLStringField(encoded *strings.Builder, name string, value string) {
	fmt.Fprintf(encoded, "%s = %s\n", name, quoteTOMLString(value))
}

func writeTOMLBoolField(encoded *strings.Builder, name string, value bool) {
	fmt.Fprintf(encoded, "%s = %t\n", name, value)
}

func writeTOMLStringArrayField(encoded *strings.Builder, name string, values []string) {
	fmt.Fprintf(encoded, "%s = [", name)
	for i, value := range values {
		if i != 0 {
			encoded.WriteString(", ")
		}
		encoded.WriteString(quoteTOMLString(value))
	}
	encoded.WriteString("]\n")
}

func quoteTOMLString(value string) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic("encoding a validated Go string as JSON cannot fail")
	}
	return string(encoded)
}
