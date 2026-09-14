package vlessenc

import (
	"bytes"
	"crypto/ecdh"
	"crypto/mlkem"
	"crypto/rand"
	"io"
	"net"
	"testing"

	E "github.com/sagernet/sing/common/exceptions"
)

// TestCommonConnRoundTrip proves the pooled Write path preserves wire
// behavior: handshake, chunked writes larger than the 8192 pool buffer, and
// reads on both ends.
func TestCommonConnRoundTrip(t *testing.T) {
	sk, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	mlk, err := mlkem.GenerateKey768()
	if err != nil {
		t.Fatal(err)
	}

	client := &ClientInstance{}
	if err = client.Init([][]byte{mlk.EncapsulationKey().Bytes(), sk.PublicKey().Bytes()}, 0, 0, ""); err != nil {
		t.Fatal(err)
	}
	server := &ServerInstance{}
	defer server.Close()
	if err = server.Init([][]byte{mlk.Bytes(), sk.Bytes()}, 0, 0, 0, ""); err != nil {
		t.Fatal(err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	serverDone := make(chan error, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			serverDone <- err
			return
		}
		defer conn.Close()
		s, err := server.Handshake(conn, nil)
		if err != nil {
			serverDone <- err
			return
		}
		defer s.Close()
		expected := append(bytes.Repeat([]byte("A"), 20000), bytes.Repeat([]byte("B"), 50000)...)
		got := make([]byte, len(expected))
		if _, err = io.ReadFull(s, got); err != nil {
			serverDone <- err
			return
		}
		if !bytes.Equal(expected, got) {
			serverDone <- E.New("server received corrupted data")
			return
		}
		// exercise the server-side pooled Write path too
		_, err = s.Write(bytes.Repeat([]byte("S"), 30000))
		serverDone <- err
	}()

	cConn, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer cConn.Close()
	c, err := client.Handshake(cConn)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	if _, err = c.Write(bytes.Repeat([]byte("A"), 20000)); err != nil {
		t.Fatal("client write: ", err)
	}
	if _, err = c.Write(bytes.Repeat([]byte("B"), 50000)); err != nil {
		t.Fatal("client write: ", err)
	}
	got := make([]byte, 30000)
	if _, err = io.ReadFull(c, got); err != nil {
		t.Fatal("client read: ", err)
	}
	if !bytes.Equal(bytes.Repeat([]byte("S"), 30000), got) {
		t.Fatal("client received corrupted data")
	}
	if err = <-serverDone; err != nil {
		t.Fatal("server side: ", err)
	}
}
