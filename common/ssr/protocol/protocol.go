package protocol

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	cr "crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"math/rand/v2"
	"net"
	"sync"
	"time"

	N "github.com/sagernet/sing/common/network"
)

var (
	errAuthSHA1V4CRC32Error   = errors.New("auth_sha1_v4 decode data wrong crc32")
	errAuthSHA1V4LengthError  = errors.New("auth_sha1_v4 decode data wrong length")
	errAuthSHA1V4Adler32Error = errors.New("auth_sha1_v4 decode data wrong adler32")
	errAuthAES128MACError     = errors.New("auth_aes128 decode data wrong mac")
	errAuthAES128LengthError  = errors.New("auth_aes128 decode data wrong length")
	errAuthAES128ChksumError  = errors.New("auth_aes128 decode data wrong checksum")
	errAuthChainLengthError   = errors.New("auth_chain decode data wrong length")
	errAuthChainChksumError   = errors.New("auth_chain decode data wrong checksum")
)

type Protocol interface {
	StreamConn(net.Conn, []byte) net.Conn
	PacketConn(N.NetPacketConn) N.NetPacketConn
	Decode(dst, src *bytes.Buffer) error
	Encode(buf *bytes.Buffer, b []byte) error
	DecodePacket([]byte) ([]byte, error)
	EncodePacket(buf *bytes.Buffer, b []byte) error
}

type protocolCreator func(b *Base) Protocol

var protocolList = make(map[string]struct {
	overhead int
	new      protocolCreator
})

func register(name string, c protocolCreator, o int) {
	protocolList[name] = struct {
		overhead int
		new      protocolCreator
	}{overhead: o, new: c}
}

// PickProtocol returns a Protocol of the given name, accumulating its
// overhead into the base.
func PickProtocol(name string, b *Base) (Protocol, error) {
	if choice, ok := protocolList[name]; ok {
		b.Overhead += choice.overhead
		return choice.new(b), nil
	}
	return nil, fmt.Errorf("protocol %s not supported", name)
}

func getHeadSize(b []byte, defaultValue int) int {
	if len(b) < 2 {
		return defaultValue
	}
	headType := b[0] & 7
	switch headType {
	case 1:
		return 7
	case 4:
		return 19
	case 3:
		return 4 + int(b[1])
	}
	return defaultValue
}

func getDataLength(b []byte) int {
	bLength := len(b)
	dataLength := getHeadSize(b, 30) + rand.IntN(32)
	if bLength < dataLength {
		return bLength
	}
	return dataLength
}

type Base struct {
	Key      []byte
	Overhead int
	Param    string
}

type userData struct {
	userKey []byte
	userID  [4]byte
}

type authData struct {
	clientID     [4]byte
	connectionID uint32
	mutex        sync.Mutex
}

func (a *authData) next() *authData {
	r := &authData{}
	a.mutex.Lock()
	defer a.mutex.Unlock()
	if a.connectionID > 0xff000000 || a.connectionID == 0 {
		cr.Read(a.clientID[:])
		a.connectionID = rand.Uint32() & 0xffffff
	}
	a.connectionID++
	copy(r.clientID[:], a.clientID[:])
	r.connectionID = a.connectionID
	return r
}

func (a *authData) putAuthData(buf *bytes.Buffer) {
	binary.Write(buf, binary.LittleEndian, uint32(time.Now().Unix()))
	buf.Write(a.clientID[:])
	binary.Write(buf, binary.LittleEndian, a.connectionID)
}

func (a *authData) putEncryptedData(b *bytes.Buffer, userKey []byte, paddings [2]int, salt string) error {
	encrypt := make([]byte, 16)
	binary.LittleEndian.PutUint32(encrypt, uint32(time.Now().Unix()))
	copy(encrypt[4:], a.clientID[:])
	binary.LittleEndian.PutUint32(encrypt[8:], a.connectionID)
	binary.LittleEndian.PutUint16(encrypt[12:], uint16(paddings[0]))
	binary.LittleEndian.PutUint16(encrypt[14:], uint16(paddings[1]))

	cipherKey := kdf(base64.StdEncoding.EncodeToString(userKey)+salt, 16)
	block, err := aes.NewCipher(cipherKey)
	if err != nil {
		return err
	}
	iv := bytes.Repeat([]byte{0}, 16)
	cbcCipher := cipher.NewCBCEncrypter(block, iv)

	cbcCipher.CryptBlocks(encrypt, encrypt)

	b.Write(encrypt)
	return nil
}

// kdf is the key-derivation function from the original Shadowsocks.
func kdf(password string, keyLen int) []byte {
	var b, prev []byte
	h := md5.New()
	for len(b) < keyLen {
		h.Write(prev)
		h.Write([]byte(password))
		b = h.Sum(b)
		prev = b[len(b)-h.Size():]
		h.Reset()
	}
	return b[:keyLen]
}
