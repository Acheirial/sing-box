package cipher

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"crypto/rc4"
	"errors"
	"io"
	"net"
	"strconv"

	"github.com/metacubex/chacha"
)

// ErrShortPacket means the packet is too short to be a valid encrypted packet.
var ErrShortPacket = errors.New("short packet")

// KeySizeError reports a wrong key size for a stream cipher.
type KeySizeError int

func (e KeySizeError) Error() string {
	return "key size error: need " + strconv.Itoa(int(e)) + " bytes"
}

// Cipher generates a pair of stream ciphers for encryption and decryption.
type Cipher interface {
	IVSize() int
	Encrypter(iv []byte) cipher.Stream
	Decrypter(iv []byte) cipher.Stream
}

var streamList = map[string]struct {
	KeySize int
	New     func(key []byte) (Cipher, error)
}{
	"RC4-MD5":       {16, RC4MD5},
	"AES-128-CTR":   {16, AESCTR},
	"AES-192-CTR":   {24, AESCTR},
	"AES-256-CTR":   {32, AESCTR},
	"AES-128-CFB":   {16, AESCFB},
	"AES-192-CFB":   {24, AESCFB},
	"AES-256-CFB":   {32, AESCFB},
	"CHACHA20":      {32, ChaCha20},
	"CHACHA20-IETF": {32, Chacha20IETF},
	"XCHACHA20":     {32, Xchacha20},
}

// List returns the supported stream cipher names.
func List() []string {
	var l []string
	for k := range streamList {
		l = append(l, k)
	}
	return l
}

// Pick returns a stream cipher of the given name, deriving the key from the
// password with the original Shadowsocks key derivation if needed.
func Pick(name string, key []byte, password string) (Cipher, []byte, error) {
	choice, ok := streamList[name]
	if !ok {
		return nil, nil, errors.New("cipher not supported: " + name)
	}
	if len(key) == 0 {
		key = Kdf(password, choice.KeySize)
	}
	if len(key) != choice.KeySize {
		return nil, nil, KeySizeError(choice.KeySize)
	}
	ciph, err := choice.New(key)
	return ciph, key, err
}

// Kdf is the key-derivation function from the original Shadowsocks.
func Kdf(password string, keyLen int) []byte {
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

// CTR mode

type ctrStream struct{ cipher.Block }

func (b *ctrStream) IVSize() int                       { return b.BlockSize() }
func (b *ctrStream) Decrypter(iv []byte) cipher.Stream { return b.Encrypter(iv) }
func (b *ctrStream) Encrypter(iv []byte) cipher.Stream { return cipher.NewCTR(b.Block, iv) }

func AESCTR(key []byte) (Cipher, error) {
	blk, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return &ctrStream{blk}, nil
}

// CFB mode

type cfbStream struct{ cipher.Block }

func (b *cfbStream) IVSize() int                       { return b.BlockSize() }
func (b *cfbStream) Decrypter(iv []byte) cipher.Stream { return cipher.NewCFBDecrypter(b.Block, iv) }
func (b *cfbStream) Encrypter(iv []byte) cipher.Stream { return cipher.NewCFBEncrypter(b.Block, iv) }

func AESCFB(key []byte) (Cipher, error) {
	blk, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return &cfbStream{blk}, nil
}

// RC4-MD5

type rc4Md5Key []byte

func (k rc4Md5Key) IVSize() int {
	return 16
}

func (k rc4Md5Key) Encrypter(iv []byte) cipher.Stream {
	h := md5.New()
	h.Write([]byte(k))
	h.Write(iv)
	rc4key := h.Sum(nil)
	c, _ := rc4.NewCipher(rc4key)
	return c
}

func (k rc4Md5Key) Decrypter(iv []byte) cipher.Stream {
	return k.Encrypter(iv)
}

func RC4MD5(key []byte) (Cipher, error) {
	return rc4Md5Key(key), nil
}

// ChaCha20 family

func newChaCha20(nonce, key []byte) cipher.Stream {
	c, err := chacha.NewChaCha20IgnoreCounterOverflow(nonce, key)
	if err != nil {
		panic(err) // should never happen
	}
	return c
}

type chacha20key []byte

func (k chacha20key) IVSize() int                       { return chacha.NonceSize }
func (k chacha20key) Encrypter(iv []byte) cipher.Stream { return newChaCha20(iv, k) }
func (k chacha20key) Decrypter(iv []byte) cipher.Stream { return k.Encrypter(iv) }

func ChaCha20(key []byte) (Cipher, error) {
	if len(key) != chacha.KeySize {
		return nil, KeySizeError(chacha.KeySize)
	}
	return chacha20key(key), nil
}

// IETF-variant of chacha20

type chacha20ietfkey []byte

func (k chacha20ietfkey) IVSize() int                       { return chacha.INonceSize }
func (k chacha20ietfkey) Decrypter(iv []byte) cipher.Stream { return k.Encrypter(iv) }
func (k chacha20ietfkey) Encrypter(iv []byte) cipher.Stream { return newChaCha20(iv, k) }

func Chacha20IETF(key []byte) (Cipher, error) {
	if len(key) != chacha.KeySize {
		return nil, KeySizeError(chacha.KeySize)
	}
	return chacha20ietfkey(key), nil
}

type xchacha20key []byte

func (k xchacha20key) IVSize() int                       { return chacha.XNonceSize }
func (k xchacha20key) Decrypter(iv []byte) cipher.Stream { return k.Encrypter(iv) }
func (k xchacha20key) Encrypter(iv []byte) cipher.Stream { return newChaCha20(iv, k) }

func Xchacha20(key []byte) (Cipher, error) {
	if len(key) != chacha.KeySize {
		return nil, KeySizeError(chacha.KeySize)
	}
	return xchacha20key(key), nil
}

// Stream conn

const bufSize = 2048

type writer struct {
	io.Writer
	cipher.Stream
	buf [bufSize]byte
}

func newWriter(w io.Writer, s cipher.Stream) *writer { return &writer{Writer: w, Stream: s} }

func (w *writer) Write(p []byte) (n int, err error) {
	buf := w.buf[:]
	for nw := 0; n < len(p) && err == nil; n += nw {
		end := n + len(buf)
		if end > len(p) {
			end = len(p)
		}
		w.XORKeyStream(buf, p[n:end])
		nw, err = w.Writer.Write(buf[:end-n])
	}
	return
}

type reader struct {
	io.Reader
	cipher.Stream
	buf [bufSize]byte
}

func newReader(r io.Reader, s cipher.Stream) *reader { return &reader{Reader: r, Stream: s} }

func (r *reader) Read(p []byte) (n int, err error) {
	n, err = r.Reader.Read(p)
	if err != nil {
		return 0, err
	}
	r.XORKeyStream(p, p[:n])
	return
}

// Conn is a stream-cipher encrypted net.Conn.
type Conn struct {
	net.Conn
	Cipher
	r       *reader
	w       *writer
	readIV  []byte
	writeIV []byte
}

// NewConn wraps a stream-oriented net.Conn with stream cipher encryption/decryption.
func NewConn(c net.Conn, ciph Cipher) *Conn { return &Conn{Conn: c, Cipher: ciph} }

func (c *Conn) initReader() error {
	if c.r == nil {
		iv, err := c.ObtainReadIV()
		if err != nil {
			return err
		}
		c.r = newReader(c.Conn, c.Decrypter(iv))
	}
	return nil
}

func (c *Conn) Read(b []byte) (int, error) {
	if c.r == nil {
		if err := c.initReader(); err != nil {
			return 0, err
		}
	}
	return c.r.Read(b)
}

func (c *Conn) WriteTo(w io.Writer) (int64, error) {
	if c.r == nil {
		if err := c.initReader(); err != nil {
			return 0, err
		}
	}
	buf := c.r.buf[:]
	var n int64
	for {
		nr, er := c.Conn.Read(buf)
		if nr > 0 {
			c.r.XORKeyStream(buf, buf[:nr])
			nw, ew := w.Write(buf[:nr])
			n += int64(nw)
			if ew != nil {
				return n, ew
			}
		}
		if er != nil {
			if er != io.EOF {
				return n, er
			}
			return n, nil
		}
	}
}

func (c *Conn) initWriter() error {
	if c.w == nil {
		iv, err := c.ObtainWriteIV()
		if err != nil {
			return err
		}
		if _, err := c.Conn.Write(iv); err != nil {
			return err
		}
		c.w = newWriter(c.Conn, c.Encrypter(iv))
	}
	return nil
}

func (c *Conn) Write(b []byte) (int, error) {
	if c.w == nil {
		if err := c.initWriter(); err != nil {
			return 0, err
		}
	}
	return c.w.Write(b)
}

func (c *Conn) ReadFrom(r io.Reader) (int64, error) {
	if c.w == nil {
		if err := c.initWriter(); err != nil {
			return 0, err
		}
	}
	buf := c.w.buf[:]
	var n int64
	for {
		nr, er := r.Read(buf)
		n += int64(nr)
		b := buf[:nr]
		c.w.XORKeyStream(b, b)
		if _, err := c.Conn.Write(b); err != nil {
			return n, err
		}
		if er != nil {
			if er != io.EOF {
				return n, er
			}
			return n, nil
		}
	}
}

// ObtainWriteIV returns the random IV generated for the write stream.
func (c *Conn) ObtainWriteIV() ([]byte, error) {
	if len(c.writeIV) == c.IVSize() {
		return c.writeIV, nil
	}

	iv := make([]byte, c.IVSize())
	if _, err := rand.Read(iv); err != nil {
		return nil, err
	}
	c.writeIV = iv
	return iv, nil
}

// ObtainReadIV reads the peer IV for the read stream.
func (c *Conn) ObtainReadIV() ([]byte, error) {
	if len(c.readIV) == c.IVSize() {
		return c.readIV, nil
	}

	iv := make([]byte, c.IVSize())
	if _, err := io.ReadFull(c.Conn, iv); err != nil {
		return nil, err
	}
	c.readIV = iv
	return iv, nil
}

// Packet conn

// Pack encrypts plaintext using stream cipher s and a random IV.
// Returns a slice of dst containing random IV and ciphertext.
// Ensure len(dst) >= s.IVSize() + len(plaintext).
func Pack(dst, plaintext []byte, s Cipher) ([]byte, error) {
	if len(dst) < s.IVSize()+len(plaintext) {
		return nil, io.ErrShortBuffer
	}
	iv := dst[:s.IVSize()]
	_, err := rand.Read(iv)
	if err != nil {
		return nil, err
	}
	s.Encrypter(iv).XORKeyStream(dst[len(iv):], plaintext)
	return dst[:len(iv)+len(plaintext)], nil
}

// UnpackInplace decrypts pkt using stream cipher s.
// Returns a slice of pkt containing decrypted plaintext.
// Note: the data in the input pkt will be changed.
func UnpackInplace(pkt []byte, s Cipher) ([]byte, error) {
	if len(pkt) < s.IVSize() {
		return nil, ErrShortPacket
	}
	iv, dst := pkt[:s.IVSize()], pkt[s.IVSize():]
	s.Decrypter(iv).XORKeyStream(dst, dst)
	return dst, nil
}

const maxPacketSize = 64 * 1024
