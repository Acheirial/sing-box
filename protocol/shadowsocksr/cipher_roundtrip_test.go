package shadowsocksr

import (
	"bytes"
	"net"
	"testing"

	"github.com/sagernet/sing-box/common/ssr/cipher"
)

// TestCipherConnRoundTrip verifies a full-duplex encrypted net.Pipe round trip
// for every stream cipher: client writes encrypted, server decrypts.
func ioReadFull(r net.Conn, buf []byte) (int, error) {
	total := 0
	for total < len(buf) {
		n, err := r.Read(buf[total:])
		total += n
		if err != nil {
			return total, err
		}
	}
	return total, nil
}

func TestCipherConnRoundTrip(t *testing.T) {
	for _, method := range []string{"RC4-MD5", "AES-128-CTR", "AES-192-CTR", "AES-256-CTR", "AES-128-CFB", "AES-192-CFB", "AES-256-CFB", "CHACHA20", "CHACHA20-IETF", "XCHACHA20"} {
		t.Run(method, func(t *testing.T) {
			clientCipher, key, err := cipher.Pick(method, nil, "password")
			if err != nil {
				t.Fatal(err)
			}
			serverCipher, _, err := cipher.Pick(method, key, "")
			if err != nil {
				t.Fatal(err)
			}

			clientConn, serverConn := net.Pipe()

			payload := bytes.Repeat([]byte("sing-box ssr cipher round trip "), 100)

			go func() {
				conn := cipher.NewConn(clientConn, clientCipher)
				conn.Write(payload)
			}()

			server := cipher.NewConn(serverConn, serverCipher)
			got := make([]byte, len(payload))
			_, err = ioReadFull(server, got)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, payload) {
				t.Fatalf("cipher %s round-trip mismatch", method)
			}
		})
	}
}

// TestCipherPacketRoundTrip verifies Pack/UnpackInplace symmetry.
func TestCipherPacketRoundTrip(t *testing.T) {
	for _, method := range []string{"RC4-MD5", "AES-128-CTR", "CHACHA20-IETF", "XCHACHA20"} {
		c, _, err := cipher.Pick(method, nil, "password")
		if err != nil {
			t.Fatal(err)
		}
		payload := []byte("packet payload")
		buf := make([]byte, c.IVSize()+len(payload))
		packed, err := cipher.Pack(buf, payload, c)
		if err != nil {
			t.Fatal(err)
		}
		unpacked, err := cipher.UnpackInplace(packed, c)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(unpacked, payload) {
			t.Fatalf("cipher %s packet round-trip mismatch", method)
		}
	}
}
