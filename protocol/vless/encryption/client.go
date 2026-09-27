package encryption

import (
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/mlkem"
	"crypto/rand"
	"encoding/base64"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	E "github.com/sagernet/sing/common/exceptions"

	"github.com/zeebo/blake3"
)

type ClientInstance struct {
	NfsPKeys      []any
	NfsPKeysBytes [][]byte
	Hash32s       [][32]byte
	RelaysLength  int
	XorMode       uint32
	Seconds       uint32
	PaddingLens   [][3]int
	PaddingGaps   [][3]int

	RWLock sync.RWMutex
	Expire time.Time
	PfsKey []byte
	Ticket []byte
}

func (i *ClientInstance) Init(nfsPKeysBytes [][]byte, xorMode, seconds uint32, padding string) (err error) {
	if i.NfsPKeys != nil {
		return E.New("already initialized")
	}
	l := len(nfsPKeysBytes)
	if l == 0 {
		return E.New("empty nfsPKeysBytes")
	}
	i.NfsPKeys = make([]any, l)
	i.NfsPKeysBytes = nfsPKeysBytes
	i.Hash32s = make([][32]byte, l)
	for j, k := range nfsPKeysBytes {
		if len(k) == 32 {
			if i.NfsPKeys[j], err = ecdh.X25519().NewPublicKey(k); err != nil {
				return
			}
			i.RelaysLength += 32 + 32
		} else {
			if i.NfsPKeys[j], err = mlkem.NewEncapsulationKey768(k); err != nil {
				return
			}
			i.RelaysLength += 1088 + 32
		}
		i.Hash32s[j] = blake3.Sum256(k)
	}
	i.RelaysLength -= 32
	i.XorMode = xorMode
	i.Seconds = seconds
	return ParsePadding(padding, &i.PaddingLens, &i.PaddingGaps)
}

func (i *ClientInstance) Handshake(conn net.Conn) (*CommonConn, error) {
	if i.NfsPKeys == nil {
		return nil, E.New("uninitialized")
	}
	c := NewCommonConn(conn, HasAESGCMHardwareSupport)

	ivAndRealysLength := 16 + i.RelaysLength
	pfsKeyExchangeLength := 18 + 1184 + 32 + 16
	paddingLength, paddingLens, paddingGaps := CreatPadding(i.PaddingLens, i.PaddingGaps)
	clientHello := make([]byte, ivAndRealysLength+pfsKeyExchangeLength+paddingLength)

	iv := clientHello[:16]
	rand.Read(iv)
	relays := clientHello[16:ivAndRealysLength]
	var nfsKey []byte
	var lastCTR cipher.Stream
	for j, k := range i.NfsPKeys {
		index := 32
		if k, ok := k.(*ecdh.PublicKey); ok {
			privateKey, _ := ecdh.X25519().GenerateKey(rand.Reader)
			copy(relays, privateKey.PublicKey().Bytes())
			var err error
			nfsKey, err = privateKey.ECDH(k)
			if err != nil {
				return nil, err
			}
		}
		if k, ok := k.(*mlkem.EncapsulationKey768); ok {
			var ciphertext []byte
			nfsKey, ciphertext = k.Encapsulate()
			copy(relays, ciphertext)
			index = 1088
		}
		if i.XorMode > 0 { // this xor can (others can't) be recovered by client's config, revealing an X25519 public key / ML-KEM-768 ciphertext, that's why "native" values
			NewCTR(i.NfsPKeysBytes[j], iv).XORKeyStream(relays, relays[:index]) // make X25519 public key / ML-KEM-768 ciphertext distinguishable from random bytes
		}
		if lastCTR != nil {
			lastCTR.XORKeyStream(relays, relays[:32]) // make this relay irreplaceable
		}
		if j == len(i.NfsPKeys)-1 {
			break
		}
		lastCTR = NewCTR(nfsKey, iv)
		lastCTR.XORKeyStream(relays[index:], i.Hash32s[j+1][:])
		relays = relays[index+32:]
	}
	nfsAEAD := NewAEAD(iv, nfsKey, c.UseAES)

	if i.Seconds > 0 {
		i.RWLock.RLock()
		if time.Now().Before(i.Expire) {
			c.Client = i
			c.UnitedKey = append(i.PfsKey, nfsKey...) // different unitedKey for each connection
			nfsAEAD.Seal(clientHello[:ivAndRealysLength], nil, EncodeLength(32), nil)
			nfsAEAD.Seal(clientHello[:ivAndRealysLength+18], nil, i.Ticket, nil)
			i.RWLock.RUnlock()
			c.PreWrite = clientHello[:ivAndRealysLength+18+32]
			c.AEAD = NewAEAD(clientHello[ivAndRealysLength+18:ivAndRealysLength+18+32], c.UnitedKey, c.UseAES)
			if i.XorMode == 2 {
				c.Conn = NewXorConn(conn, NewCTR(c.UnitedKey, iv), nil, len(c.PreWrite), 16)
			}
			return c, nil
		}
		i.RWLock.RUnlock()
	}

	pfsKeyExchange := clientHello[ivAndRealysLength : ivAndRealysLength+pfsKeyExchangeLength]
	nfsAEAD.Seal(pfsKeyExchange[:0], nil, EncodeLength(pfsKeyExchangeLength-18), nil)
	mlkem768DKey, _ := mlkem.GenerateKey768()
	x25519SKey, _ := ecdh.X25519().GenerateKey(rand.Reader)
	pfsPublicKey := append(mlkem768DKey.EncapsulationKey().Bytes(), x25519SKey.PublicKey().Bytes()...)
	nfsAEAD.Seal(pfsKeyExchange[:18], nil, pfsPublicKey, nil)

	padding := clientHello[ivAndRealysLength+pfsKeyExchangeLength:]
	nfsAEAD.Seal(padding[:0], nil, EncodeLength(paddingLength-18), nil)
	nfsAEAD.Seal(padding[:18], nil, padding[18:paddingLength-16], nil)

	paddingLens[0] = ivAndRealysLength + pfsKeyExchangeLength + paddingLens[0]
	for i, l := range paddingLens { // sends padding in a fragmented way, to create variable traffic pattern, before inner VLESS flow takes control
		if l > 0 {
			if _, err := conn.Write(clientHello[:l]); err != nil {
				return nil, err
			}
			clientHello = clientHello[l:]
		}
		if len(paddingGaps) > i {
			time.Sleep(paddingGaps[i])
		}
	}

	encryptedPfsPublicKey := make([]byte, 1088+32+16)
	if _, err := io.ReadFull(conn, encryptedPfsPublicKey); err != nil {
		return nil, err
	}
	nfsAEAD.Open(encryptedPfsPublicKey[:0], MaxNonce, encryptedPfsPublicKey, nil)
	mlkem768Key, err := mlkem768DKey.Decapsulate(encryptedPfsPublicKey[:1088])
	if err != nil {
		return nil, err
	}
	peerX25519PKey, err := ecdh.X25519().NewPublicKey(encryptedPfsPublicKey[1088 : 1088+32])
	if err != nil {
		return nil, err
	}
	x25519Key, err := x25519SKey.ECDH(peerX25519PKey)
	if err != nil {
		return nil, err
	}
	pfsKey := make([]byte, 32+32) // no more capacity
	copy(pfsKey, mlkem768Key)
	copy(pfsKey[32:], x25519Key)
	c.UnitedKey = append(pfsKey, nfsKey...)
	c.AEAD = NewAEAD(pfsPublicKey, c.UnitedKey, c.UseAES)
	c.PeerAEAD = NewAEAD(encryptedPfsPublicKey[:1088+32], c.UnitedKey, c.UseAES)

	encryptedTicket := make([]byte, 32)
	if _, err := io.ReadFull(conn, encryptedTicket); err != nil {
		return nil, err
	}
	if _, err := c.PeerAEAD.Open(encryptedTicket[:0], nil, encryptedTicket, nil); err != nil {
		return nil, err
	}
	seconds := DecodeLength(encryptedTicket)

	if i.Seconds > 0 && seconds > 0 {
		i.RWLock.Lock()
		i.Expire = time.Now().Add(time.Duration(seconds) * time.Second)
		i.PfsKey = pfsKey
		i.Ticket = encryptedTicket[:16]
		i.RWLock.Unlock()
	}

	encryptedLength := make([]byte, 18)
	if _, err := io.ReadFull(conn, encryptedLength); err != nil {
		return nil, err
	}
	if _, err := c.PeerAEAD.Open(encryptedLength[:0], nil, encryptedLength, nil); err != nil {
		return nil, err
	}
	length := DecodeLength(encryptedLength[:2])
	c.PeerPadding = make([]byte, length) // important: allows server sends padding slowly, eliminating 1-RTT's traffic pattern

	if i.XorMode == 2 {
		c.Conn = NewXorConn(conn, NewCTR(c.UnitedKey, iv), NewCTR(c.UnitedKey, encryptedTicket[:16]), 0, length)
	}
	return c, nil
}

// ParseClientEncryption parses the client-side "encryption" string.
//
// Grammar (matching Xray's infra/conf/vless.go:330-363):
//
//		mlkem768x25519plus.<mode>.<rtt>[.<padding>][.<base64url-key>...]
//
//	  - mode: "native" (no XOR), "xorpub" (XorMode 1), "random" (XorMode 2)
//	  - rtt:  "1rtt" (Seconds 0), "0rtt" (Seconds 1, enables ticket reuse)
//	  - padding: leading dot-separated components shorter than 20 bytes, passed
//	    through to ParsePadding (e.g. "100-111-1111.75-0-111")
//	  - keys: dot-separated base64.RawURLEncoding values decoding to 32 (X25519)
//	    or 1184 (ML-KEM-768 encapsulation key) bytes
func ParseClientEncryption(encryption string) (nfsPKeysBytes [][]byte, xorMode, seconds uint32, padding string, err error) {
	s := strings.Split(encryption, ".")
	if len(s) < 4 || s[0] != "mlkem768x25519plus" {
		return nil, 0, 0, "", E.New("unsupported encryption: ", encryption)
	}
	switch s[1] {
	case "native":
	case "xorpub":
		xorMode = 1
	case "random":
		xorMode = 2
	default:
		return nil, 0, 0, "", E.New("unsupported encryption: ", encryption)
	}
	switch s[2] {
	case "1rtt":
	case "0rtt":
		seconds = 1
	default:
		return nil, 0, 0, "", E.New("unsupported encryption: ", encryption)
	}
	rest := encryption[len(s[0])+1+len(s[1])+1+len(s[2])+1:]
	parts := strings.Split(rest, ".")
	index := 0
	for index < len(parts) && len(parts[index]) < 20 {
		padding += parts[index] + "."
		index++
	}
	if padding != "" {
		padding = strings.TrimSuffix(padding, ".")
	}
	for _, r := range parts[index:] {
		b, decodeErr := base64.RawURLEncoding.DecodeString(r)
		if decodeErr != nil || (len(b) != 32 && len(b) != 1184) {
			return nil, 0, 0, "", E.New("unsupported encryption: ", encryption)
		}
		nfsPKeysBytes = append(nfsPKeysBytes, b)
	}
	if len(nfsPKeysBytes) == 0 {
		return nil, 0, 0, "", E.New("no key in encryption: ", encryption)
	}
	return
}
