package shadowsocksr

import (
	"bytes"
	"context"
	"net"
	"testing"

	"github.com/sagernet/sing-box/common/ssr/cipher"
	"github.com/sagernet/sing-box/common/ssr/obfs"
	"github.com/sagernet/sing-box/common/ssr/protocol"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
)

// TestOutboundConstruction verifies every supported method resolves through
// the full obfs/protocol pick chain, and unsupported methods error out.
func TestOutboundConstruction(t *testing.T) {
	for _, method := range []string{"none", "dummy", "rc4-md5", "aes-128-cfb", "aes-192-cfb", "aes-256-cfb", "aes-128-ctr", "aes-192-ctr", "aes-256-ctr", "chacha20", "chacha20-ietf", "xchacha20"} {
		_, err := NewOutbound(context.Background(), nil, log.NewNOPFactory().Logger(), "ssr-test", option.ShadowsocksROutboundOptions{
			ServerOptions: option.ServerOptions{Server: "127.0.0.1", ServerPort: 8388},
			Method:        method,
			Password:      "password",
			Obfs:          "plain",
			Protocol:      "origin",
		})
		if err != nil {
			t.Fatalf("method %s: %v", method, err)
		}
	}
	for _, method := range []string{"aes-256-gcm", "unknown-cipher"} {
		_, err := NewOutbound(context.Background(), nil, log.NewNOPFactory().Logger(), "ssr-test", option.ShadowsocksROutboundOptions{
			ServerOptions: option.ServerOptions{Server: "127.0.0.1", ServerPort: 8388},
			Method:        method,
			Password:      "password",
			Obfs:          "plain",
			Protocol:      "origin",
		})
		if err == nil {
			t.Fatalf("expected error for method %s", method)
		}
	}
	_, err := NewOutbound(context.Background(), nil, log.NewNOPFactory().Logger(), "ssr-test", option.ShadowsocksROutboundOptions{
		ServerOptions: option.ServerOptions{Server: "127.0.0.1", ServerPort: 8388},
		Method:        "dummy",
		Password:      "password",
		Obfs:          "unknown-obfs",
		Protocol:      "origin",
	})
	if err == nil {
		t.Fatal("expected error for unknown obfs")
	}
	_, err = NewOutbound(context.Background(), nil, log.NewNOPFactory().Logger(), "ssr-test", option.ShadowsocksROutboundOptions{
		ServerOptions: option.ServerOptions{Server: "127.0.0.1", ServerPort: 8388},
		Method:        "dummy",
		Password:      "password",
		Obfs:          "plain",
		Protocol:      "unknown-protocol",
	})
	if err == nil {
		t.Fatal("expected error for unknown protocol")
	}
}

// TestOriginPlainRoundTrip exercises the simplest chain end-to-end over a
// net.Pipe: protocol-encode on the client, protocol-decode on the server.
func TestOriginPlainRoundTrip(t *testing.T) {
	key := cipher.Kdf("password", 16)
	protoObj, err := protocol.PickProtocol("origin", &protocol.Base{Key: key})
	if err != nil {
		t.Fatal(err)
	}

	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	payload := []byte{0x01, 0x02, 0x03, 0x04}

	go func() {
		conn := protoObj.StreamConn(clientConn, nil)
		conn.Write(payload)
	}()

	buf := make([]byte, 64)
	n, err := serverConn.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(buf[:n], payload) {
		t.Fatalf("round-trip mismatch: %v", buf[:n])
	}
}

// TestObfsPicksAllRegistered verifies every obfs plugin constructs and wraps
// a conn without error.
func TestObfsPicksAllRegistered(t *testing.T) {
	for _, name := range []string{"plain", "http_simple", "http_post", "random_head", "tls1.2_ticket_auth", "tls1.2_ticket_fastauth"} {
		obfsObj, _, err := obfs.PickObfs(name, &obfs.Base{
			Host:   "127.0.0.1",
			Port:   8080,
			Key:    cipher.Kdf("password", 16),
			IVSize: 16,
			Param:  "",
		})
		if err != nil {
			t.Fatalf("obfs %s: %v", name, err)
		}
		clientConn, serverConn := net.Pipe()
		clientConn.Close()
		serverConn.Close()
		wrapped := obfsObj.StreamConn(clientConn)
		if wrapped == nil {
			t.Fatalf("obfs %s returned nil conn", name)
		}
	}
}

// TestProtocolPicksAllRegistered verifies every protocol plugin constructs.
func TestProtocolPicksAllRegistered(t *testing.T) {
	for _, name := range []string{"origin", "auth_sha1_v4", "auth_aes128_md5", "auth_aes128_sha1", "auth_chain_a", "auth_chain_b"} {
		protoObj, err := protocol.PickProtocol(name, &protocol.Base{Key: cipher.Kdf("password", 16)})
		if err != nil {
			t.Fatalf("protocol %s: %v", name, err)
		}
		clientConn, serverConn := net.Pipe()
		clientConn.Close()
		serverConn.Close()
		wrapped := protoObj.StreamConn(clientConn, make([]byte, 16))
		if wrapped == nil {
			t.Fatalf("protocol %s returned nil conn", name)
		}
		if name == "auth_aes128_md5" || name == "auth_aes128_sha1" || name == "auth_chain_a" || name == "auth_chain_b" {
			packet := protoObj.PacketConn(nil)
			if packet == nil {
				t.Fatalf("protocol %s returned nil packet conn", name)
			}
		}
	}
}
