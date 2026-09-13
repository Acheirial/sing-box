package tools

import (
	"bytes"
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha1"
	"io"
	"sync"
)

const RelayBufferSize = 32768

var relayBufferPool = sync.Pool{
	New: func() any {
		return make([]byte, RelayBufferSize)
	},
}

// GetRelayBuffer returns a pooled buffer for stream relaying.
func GetRelayBuffer() []byte {
	return relayBufferPool.Get().([]byte)
}

// PutRelayBuffer returns a relay buffer to the pool.
func PutRelayBuffer(buffer []byte) {
	relayBufferPool.Put(buffer)
}

func HmacMD5(key, data []byte) []byte {
	hmacMD5 := hmac.New(md5.New, key)
	hmacMD5.Write(data)
	return hmacMD5.Sum(nil)
}

func HmacSHA1(key, data []byte) []byte {
	hmacSHA1 := hmac.New(sha1.New, key)
	hmacSHA1.Write(data)
	return hmacSHA1.Sum(nil)
}

func MD5Sum(b []byte) []byte {
	h := md5.New()
	h.Write(b)
	return h.Sum(nil)
}

func SHA1Sum(b []byte) []byte {
	h := sha1.New()
	h.Write(b)
	return h.Sum(nil)
}

func AppendRandBytes(b *bytes.Buffer, length int) {
	b.ReadFrom(io.LimitReader(rand.Reader, int64(length)))
}
