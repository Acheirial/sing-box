package vless

// VisionConn ports sing-vmess github.com/sagernet/sing-vmess/vless/vision.go
// (v0.2.8, itself derived from the Xray-core XTLS Vision flow) with the
// padding sizes parameterized through testseed, mirroring Xray-core
// proxy.XtlsPadding / NewVisionWriter (proxy/proxy.go):
//
//	testseed[0]: content length threshold for long padding (sing-vmess: 900)
//	testseed[1]: random range of the long padding size (sing-vmess: 500)
//	testseed[2]: base size of the long padding (sing-vmess: 900)
//	testseed[3]: random range of the short padding size (sing-vmess: 256)
//
// A testseed with fewer than 4 elements falls back to the Xray-core default
// {900, 500, 900, 256}, which makes NewVisionConnWithSeed behave exactly
// like sing-vmess NewVisionConn when testseed is nil. As in Xray, no other
// validation is applied (uint32 values are inherently non-negative).
//
// The UUID prefix write, padding frame format, direct-copy switching, TLS
// record filtering and unpadding are kept byte-identical to sing-vmess,
// which is Xray-compatible on the wire. Direct TLS connections are accepted
// in the same way as sing-vmess: *tls.Conn always, *utls.UConn / *utls.Conn
// when built with the with_utls tag.

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"io"
	"math/big"
	"net"
	"reflect"
	"sync"
	"time"
	"unsafe"

	"github.com/sagernet/sing/common"
	"github.com/sagernet/sing/common/buf"
	"github.com/sagernet/sing/common/bufio"
	E "github.com/sagernet/sing/common/exceptions"
	"github.com/sagernet/sing/common/logger"
	N "github.com/sagernet/sing/common/network"
)

var tlsRegistry []func(conn net.Conn) (loaded bool, netConn net.Conn, reflectType reflect.Type, reflectPointer uintptr)

func init() {
	tlsRegistry = append(tlsRegistry, func(conn net.Conn) (loaded bool, netConn net.Conn, reflectType reflect.Type, reflectPointer uintptr) {
		tlsConn, loaded := N.CastReader[*tls.Conn](conn)
		if !loaded {
			return
		}
		return true, tlsConn.NetConn(), reflect.TypeOf(tlsConn).Elem(), uintptr(unsafe.Pointer(tlsConn))
	})
}

const xrayChunkSize = 8192

var (
	tls13SupportedVersions  = []byte{0x00, 0x2b, 0x00, 0x02, 0x03, 0x04}
	tlsClientHandShakeStart = []byte{0x16, 0x03}
	tlsApplicationDataStart = []byte{0x17, 0x03, 0x03}
)

const (
	commandPaddingContinue byte = iota
	commandPaddingEnd
	commandPaddingDirect
)

var tls13CipherSuiteDic = map[uint16]string{
	0x1301: "TLS_AES_128_GCM_SHA256",
	0x1302: "TLS_AES_256_GCM_SHA384",
	0x1303: "TLS_CHACHA20_POLY1305_SHA256",
	0x1304: "TLS_AES_128_CCM_SHA256",
	0x1305: "TLS_AES_128_CCM_8_SHA256",
}

func reshapeBuffer(b []byte) []*buf.Buffer {
	const bufferLimit = 8192 - 21
	if len(b) < bufferLimit {
		return []*buf.Buffer{buf.As(b)}
	}
	var buffers []*buf.Buffer
	for len(b) >= bufferLimit {
		index := int32(bytes.LastIndex(b, tlsApplicationDataStart))
		if index < 32 || index > bufferLimit {
			index = 8192 / 2
		}
		buffers = append(buffers, buf.As(b[:index]))
		b = b[index:]
	}
	buffers = append(buffers, buf.As(b))
	return buffers
}

type VisionConn struct {
	net.Conn
	reader      *bufio.ChunkReader
	writer      N.VectorisedWriter
	writeAccess sync.Mutex
	input       *bytes.Reader
	rawInput    *bytes.Buffer
	netConn     net.Conn
	logger      logger.Logger

	userUUID               [16]byte
	testseed               []uint32
	isTLS                  bool
	numberOfPacketToFilter int
	isTLS12orAbove         bool
	remainingServerHello   int32
	cipher                 uint16
	enableXTLS             bool
	isPadding              bool
	directWrite            bool
	writeUUID              bool
	withinPaddingBuffers   bool
	remainingContent       int
	remainingPadding       int
	currentCommand         byte
	directRead             bool
	remainingBuffers       []*buf.Buffer
}

func NewVisionConnWithSeed(conn net.Conn, tlsConn net.Conn, userUUID [16]byte, logger logger.Logger, testseed []uint32) (*VisionConn, error) {
	if len(testseed) < 4 {
		// Xray-core proxy/proxy.go NewVisionWriter default.
		testseed = []uint32{900, 500, 900, 256}
	}
	var (
		loaded         bool
		reflectType    reflect.Type
		reflectPointer uintptr
		netConn        net.Conn
	)
	for _, tlsCreator := range tlsRegistry {
		loaded, netConn, reflectType, reflectPointer = tlsCreator(tlsConn)
		if loaded {
			break
		}
	}
	if !loaded {
		return nil, E.New("vision: not a valid supported TLS connection: ", reflect.TypeOf(tlsConn))
	}
	input, _ := reflectType.FieldByName("input")
	rawInput, _ := reflectType.FieldByName("rawInput")
	return &VisionConn{
		Conn:     conn,
		reader:   bufio.NewChunkReader(conn, xrayChunkSize),
		writer:   bufio.NewVectorisedWriter(conn),
		input:    (*bytes.Reader)(unsafe.Pointer(reflectPointer + input.Offset)),
		rawInput: (*bytes.Buffer)(unsafe.Pointer(reflectPointer + rawInput.Offset)),
		netConn:  netConn,
		logger:   logger,

		userUUID:               userUUID,
		testseed:               testseed,
		numberOfPacketToFilter: 8,
		remainingServerHello:   -1,
		isPadding:              true,
		writeUUID:              true,
		withinPaddingBuffers:   true,
		remainingContent:       -1,
		remainingPadding:       -1,
	}, nil
}

func (c *VisionConn) Read(p []byte) (n int, err error) {
	for len(c.remainingBuffers) > 0 {
		newN, _ := c.remainingBuffers[0].Read(p[n:])
		if c.remainingBuffers[0].IsEmpty() {
			c.remainingBuffers[0].Release()
			c.remainingBuffers = c.remainingBuffers[1:]
		}
		n += newN
		if n == len(p) {
			break
		}
	}
	if n > 0 {
		return
	}
	if c.directRead {
		return c.netConn.Read(p)
	}
	var bufferBytes []byte
	var chunkBuffer *buf.Buffer
	if len(p) > xrayChunkSize {
		n, err = c.Conn.Read(p)
		if err != nil {
			return
		}
		bufferBytes = p[:n]
	} else {
		chunkBuffer, err = c.reader.ReadChunk()
		if err != nil {
			return 0, err
		}
		bufferBytes = chunkBuffer.Bytes()
	}
	if c.withinPaddingBuffers || c.numberOfPacketToFilter > 0 {
		buffers := c.unPadding(bufferBytes)
		if chunkBuffer != nil {
			chunkBuffer.Reset()
		}
		if c.remainingContent == 0 && c.remainingPadding == 0 {
			switch c.currentCommand {
			case commandPaddingEnd:
				c.withinPaddingBuffers = false
				c.remainingContent = -1
				c.remainingPadding = -1
			case commandPaddingDirect:
				c.withinPaddingBuffers = false
				c.directRead = true

				inputBuffer, err := io.ReadAll(c.input)
				if err != nil {
					return 0, err
				}
				buffers = append(buffers, buf.As(inputBuffer))

				rawInputBuffer, err := io.ReadAll(c.rawInput)
				if err != nil {
					return 0, err
				}

				buffers = append(buffers, buf.As(rawInputBuffer))

				c.logger.Trace("XtlsRead readV")
			case commandPaddingContinue:
				c.withinPaddingBuffers = true
			default:
				return 0, E.New("unknown command ", c.currentCommand)
			}
		} else if c.remainingContent > 0 || c.remainingPadding > 0 {
			c.withinPaddingBuffers = true
		} else {
			c.withinPaddingBuffers = false
		}
		if c.numberOfPacketToFilter > 0 {
			c.filterTLS(buf.ToSliceMulti(buffers))
		}
		c.remainingBuffers = buffers
		return c.Read(p)
	} else {
		if c.numberOfPacketToFilter > 0 {
			c.filterTLS([][]byte{bufferBytes})
		}
		if chunkBuffer != nil {
			c.remainingBuffers = append(c.remainingBuffers, buf.As(chunkBuffer.Bytes()))
			chunkBuffer.Reset() // chunkBuffer should not be release and only reused after c.remainingBuffers be emptied, so must reset at here
			return c.Read(p)
		}
		return
	}
}

func (c *VisionConn) Write(p []byte) (n int, err error) {
	c.writeAccess.Lock()
	defer c.writeAccess.Unlock()
	if c.numberOfPacketToFilter > 0 {
		c.filterTLS([][]byte{p})
	}
	if c.isPadding {
		inputLen := len(p)
		buffers := reshapeBuffer(p)
		var specIndex int
		for i, buffer := range buffers {
			if c.isTLS && buffer.Len() > 6 && bytes.Equal(tlsApplicationDataStart, buffer.To(3)) {
				var command = commandPaddingEnd
				if c.enableXTLS {
					c.directWrite = true
					specIndex = i
					command = commandPaddingDirect
				}
				c.isPadding = false
				buffers[i] = c.padding(buffer, command)
				break
			} else if !c.isTLS12orAbove && c.numberOfPacketToFilter <= 1 {
				c.isPadding = false
				buffers[i] = c.padding(buffer, commandPaddingEnd)
				break
			}
			buffers[i] = c.padding(buffer, commandPaddingContinue)
		}
		if c.directWrite {
			encryptedBuffer := buffers[:specIndex+1]
			err = c.writer.WriteVectorised(encryptedBuffer)
			if err != nil {
				return
			}
			buffers = buffers[specIndex+1:]
			c.writer = bufio.NewVectorisedWriter(c.netConn)
			if len(buffers) > 0 {
				c.logger.Trace("XtlsWrite writeV ", specIndex, " ", buf.LenMulti(encryptedBuffer), " ", len(buffers))
				time.Sleep(5 * time.Millisecond) // wtf
			}
		}
		if len(buffers) > 0 {
			err = c.writer.WriteVectorised(buffers)
		}
		if err == nil {
			n = inputLen
		}
		return
	}
	if c.directWrite {
		return c.netConn.Write(p)
	} else {
		return c.Conn.Write(p)
	}
}

func (c *VisionConn) filterTLS(buffers [][]byte) {
	for _, buffer := range buffers {
		c.numberOfPacketToFilter--
		if len(buffer) > 6 {
			if buffer[0] == 22 && buffer[1] == 3 && buffer[2] == 3 {
				c.isTLS = true
				if buffer[5] == 2 {
					c.isTLS12orAbove = true
					c.remainingServerHello = (int32(buffer[3])<<8 | int32(buffer[4])) + 5
					if len(buffer) >= 79 && c.remainingServerHello >= 79 {
						sessionIdLen := int32(buffer[43])
						cipherSuite := buffer[43+sessionIdLen+1 : 43+sessionIdLen+3]
						c.cipher = uint16(cipherSuite[0])<<8 | uint16(cipherSuite[1])
					} else {
						c.logger.Trace("XtlsFilterTls short server hello, tls 1.2 or older? ", len(buffer), " ", c.remainingServerHello)
					}
				}
			} else if bytes.Equal(tlsClientHandShakeStart, buffer[:2]) && buffer[5] == 1 {
				c.isTLS = true
				c.logger.Trace("XtlsFilterTls found tls client hello! ", len(buffer))
			}
		}
		if c.remainingServerHello > 0 {
			end := int(c.remainingServerHello)
			if end > len(buffer) {
				end = len(buffer)
			}
			c.remainingServerHello -= int32(end)
			if bytes.Contains(buffer[:end], tls13SupportedVersions) {
				cipher, ok := tls13CipherSuiteDic[c.cipher]
				if ok && cipher != "TLS_AES_128_CCM_8_SHA256" {
					c.enableXTLS = true
				}
				c.logger.Trace("XtlsFilterTls found tls 1.3! ", len(buffer), " ", c.cipher, " ", c.enableXTLS)
				c.numberOfPacketToFilter = 0
				return
			} else if c.remainingServerHello == 0 {
				c.logger.Trace("XtlsFilterTls found tls 1.2! ", len(buffer))
				c.numberOfPacketToFilter = 0
				return
			}
		}
		if c.numberOfPacketToFilter == 0 {
			c.logger.Trace("XtlsFilterTls stop filtering ", len(buffer))
		}
	}
}

func (c *VisionConn) padding(buffer *buf.Buffer, command byte) *buf.Buffer {
	contentLen := 0
	paddingLen := 0
	if buffer != nil {
		contentLen = buffer.Len()
	}
	if contentLen < int(c.testseed[0]) && c.isTLS {
		l, _ := rand.Int(rand.Reader, big.NewInt(int64(c.testseed[1])))
		paddingLen = int(l.Int64()) + int(c.testseed[2]) - contentLen
	} else {
		l, _ := rand.Int(rand.Reader, big.NewInt(int64(c.testseed[3])))
		paddingLen = int(l.Int64())
	}
	var bufferLen int
	if c.writeUUID {
		bufferLen += 16
	}
	bufferLen += 5
	if buffer != nil {
		bufferLen += buffer.Len()
	}
	bufferLen += paddingLen
	newBuffer := buf.NewSize(bufferLen)
	if c.writeUUID {
		common.Must1(newBuffer.Write(c.userUUID[:]))
		c.writeUUID = false
	}
	common.Must1(newBuffer.Write([]byte{command, byte(contentLen >> 8), byte(contentLen), byte(paddingLen >> 8), byte(paddingLen)}))
	if buffer != nil {
		common.Must1(newBuffer.Write(buffer.Bytes()))
		buffer.Release()
	}
	newBuffer.Extend(paddingLen)
	c.logger.Trace("XtlsPadding ", contentLen, " ", paddingLen, " ", command)
	return newBuffer
}

func (c *VisionConn) unPadding(buffer []byte) []*buf.Buffer {
	var bufferIndex int
	if c.remainingContent == -1 && c.remainingPadding == -1 {
		if len(buffer) >= 21 && bytes.Equal(c.userUUID[:], buffer[:16]) {
			bufferIndex = 16
			c.remainingContent = 0
			c.remainingPadding = 0
			c.currentCommand = 0
		}
	}
	if c.remainingContent == -1 && c.remainingPadding == -1 {
		return []*buf.Buffer{buf.As(buffer).ToOwned()}
	}
	var buffers []*buf.Buffer
	for bufferIndex < len(buffer) {
		if c.remainingContent <= 0 && c.remainingPadding <= 0 {
			if c.currentCommand == 1 {
				buffers = append(buffers, buf.As(buffer[bufferIndex:]).ToOwned())
				break
			} else {
				paddingInfo := buffer[bufferIndex : bufferIndex+5]
				c.currentCommand = paddingInfo[0]
				c.remainingContent = int(paddingInfo[1])<<8 | int(paddingInfo[2])
				c.remainingPadding = int(paddingInfo[3])<<8 | int(paddingInfo[4])
				bufferIndex += 5
				c.logger.Trace("Xtls Unpadding new block ", bufferIndex, " ", c.remainingContent, " padding ", c.remainingPadding, " ", c.currentCommand)
			}
		} else if c.remainingContent > 0 {
			end := c.remainingContent
			if end > len(buffer)-bufferIndex {
				end = len(buffer) - bufferIndex
			}
			buffers = append(buffers, buf.As(buffer[bufferIndex:bufferIndex+end]).ToOwned())
			c.remainingContent -= end
			bufferIndex += end
		} else {
			end := c.remainingPadding
			if end > len(buffer)-bufferIndex {
				end = len(buffer) - bufferIndex
			}
			c.remainingPadding -= end
			bufferIndex += end
		}
		if bufferIndex == len(buffer) {
			break
		}
	}
	return buffers
}

func (c *VisionConn) NeedAdditionalReadDeadline() bool {
	return true
}

func (c *VisionConn) Upstream() any {
	return c.Conn
}
