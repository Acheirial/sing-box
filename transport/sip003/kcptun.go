package sip003

import (
	"context"
	"crypto/sha1"
	"math/rand/v2"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/sagernet/sing-box/adapter"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
	"github.com/sagernet/smux"

	"github.com/golang/snappy"
	"github.com/metacubex/kcp-go"
	"golang.org/x/crypto/pbkdf2"
)

func init() {
	RegisterPlugin("kcptun", newKcpTunPlugin)
}

const (
	kcpTunSALT = "kcp-go"
	// maximum supported smux version
	kcpTunMaxSmuxVer = 2
	// scavenger check period
	kcpTunScavengePeriod = 5
)

type kcpTunConfig struct {
	Key          string
	Crypt        string
	Mode         string
	Conn         int
	AutoExpire   int
	ScavengeTTL  int
	MTU          int
	RateLimit    int
	SndWnd       int
	RcvWnd       int
	DataShard    int
	ParityShard  int
	DSCP         int
	NoComp       bool
	AckNodelay   bool
	NoDelay      int
	Interval     int
	Resend       int
	NoCongestion int
	SockBuf      int
	SmuxVer      int
	SmuxBuf      int
	FrameSize    int
	StreamBuf    int
	KeepAlive    int
}

func (config *kcpTunConfig) fillDefaults() {
	if config.Key == "" {
		config.Key = "it's a secrect"
	}
	if config.Crypt == "" {
		config.Crypt = "aes"
	}
	if config.Mode == "" {
		config.Mode = "fast"
	}
	if config.Conn == 0 {
		config.Conn = 1
	}
	if config.ScavengeTTL == 0 {
		config.ScavengeTTL = 600
	}
	if config.MTU == 0 {
		config.MTU = 1350
	}
	if config.SndWnd == 0 {
		config.SndWnd = 128
	}
	if config.RcvWnd == 0 {
		config.RcvWnd = 512
	}
	if config.DataShard == 0 {
		config.DataShard = 10
	}
	if config.ParityShard == 0 {
		config.ParityShard = 3
	}
	if config.Interval == 0 {
		config.Interval = 50
	}
	if config.SockBuf == 0 {
		config.SockBuf = 4194304
	}
	if config.SmuxVer == 0 {
		config.SmuxVer = 1
	}
	if config.SmuxBuf == 0 {
		config.SmuxBuf = 4194304
	}
	if config.FrameSize == 0 {
		config.FrameSize = 8192
	}
	if config.StreamBuf == 0 {
		config.StreamBuf = 2097152
	}
	if config.KeepAlive == 0 {
		config.KeepAlive = 10
	}
	switch config.Mode {
	case "normal":
		config.NoDelay, config.Interval, config.Resend, config.NoCongestion = 0, 40, 2, 1
	case "fast":
		config.NoDelay, config.Interval, config.Resend, config.NoCongestion = 0, 30, 2, 1
	case "fast2":
		config.NoDelay, config.Interval, config.Resend, config.NoCongestion = 1, 20, 2, 1
	case "fast3":
		config.NoDelay, config.Interval, config.Resend, config.NoCongestion = 1, 10, 2, 1
	}
	if config.SmuxVer > kcpTunMaxSmuxVer {
		config.SmuxVer = kcpTunMaxSmuxVer
	}
}

func (config *kcpTunConfig) newBlock() (block kcp.BlockCrypt) {
	pass := pbkdf2.Key([]byte(config.Key), []byte(kcpTunSALT), 4096, 32, sha1.New)
	switch config.Crypt {
	case "null":
		block = nil
	case "tea":
		block, _ = kcp.NewTEABlockCrypt(pass[:16])
	case "xor":
		block, _ = kcp.NewSimpleXORBlockCrypt(pass)
	case "none":
		block, _ = kcp.NewNoneBlockCrypt(pass)
	case "aes-128":
		block, _ = kcp.NewAESBlockCrypt(pass[:16])
	case "aes-192":
		block, _ = kcp.NewAESBlockCrypt(pass[:24])
	case "blowfish":
		block, _ = kcp.NewBlowfishBlockCrypt(pass)
	case "twofish":
		block, _ = kcp.NewTwofishBlockCrypt(pass)
	case "cast5":
		block, _ = kcp.NewCast5BlockCrypt(pass[:16])
	case "3des":
		block, _ = kcp.NewTripleDESBlockCrypt(pass[:24])
	case "xtea":
		block, _ = kcp.NewXTEABlockCrypt(pass[:16])
	case "salsa20":
		block, _ = kcp.NewSalsa20BlockCrypt(pass)
	case "aes-128-gcm":
		block, _ = kcp.NewAESGCMCrypt(pass[:16])
	default:
		config.Crypt = "aes"
		block, _ = kcp.NewAESBlockCrypt(pass)
	}
	return
}

var _ Plugin = (*KcpTunPlugin)(nil)

type KcpTunPlugin struct {
	once   sync.Once
	config kcpTunConfig
	block  kcp.BlockCrypt

	ctx    context.Context
	cancel context.CancelFunc

	numconn uint16
	muxes   []timedSession
	rr      uint16
	connMu  sync.Mutex

	chScavenger chan timedSession

	dialFn DialFn
}

func newKcpTunPlugin(ctx context.Context, pluginOpts Args, router adapter.Router, dialer N.Dialer, serverAddr M.Socksaddr) (Plugin, error) {
	config := &kcpTunConfig{}
	if value, loaded := pluginOpts.Get("key"); loaded {
		config.Key = value
	}
	if value, loaded := pluginOpts.Get("crypt"); loaded {
		config.Crypt = value
	}
	if value, loaded := pluginOpts.Get("mode"); loaded {
		config.Mode = value
	}
	if value, loaded := pluginOpts.Get("conn"); loaded {
		if n, atoiErr := strconv.Atoi(value); atoiErr == nil {
			config.Conn = n
		}
	}
	if value, loaded := pluginOpts.Get("autoexpire"); loaded {
		if n, atoiErr := strconv.Atoi(value); atoiErr == nil {
			config.AutoExpire = n
		}
	}
	if value, loaded := pluginOpts.Get("scavengettl"); loaded {
		if n, atoiErr := strconv.Atoi(value); atoiErr == nil {
			config.ScavengeTTL = n
		}
	}
	if value, loaded := pluginOpts.Get("mtu"); loaded {
		if n, atoiErr := strconv.Atoi(value); atoiErr == nil {
			config.MTU = n
		}
	}
	if value, loaded := pluginOpts.Get("ratelimit"); loaded {
		if n, atoiErr := strconv.Atoi(value); atoiErr == nil {
			config.RateLimit = n
		}
	}
	if value, loaded := pluginOpts.Get("sndwnd"); loaded {
		if n, atoiErr := strconv.Atoi(value); atoiErr == nil {
			config.SndWnd = n
		}
	}
	if value, loaded := pluginOpts.Get("rcvwnd"); loaded {
		if n, atoiErr := strconv.Atoi(value); atoiErr == nil {
			config.RcvWnd = n
		}
	}
	if value, loaded := pluginOpts.Get("datashard"); loaded {
		if n, atoiErr := strconv.Atoi(value); atoiErr == nil {
			config.DataShard = n
		}
	}
	if value, loaded := pluginOpts.Get("parityshard"); loaded {
		if n, atoiErr := strconv.Atoi(value); atoiErr == nil {
			config.ParityShard = n
		}
	}
	if value, loaded := pluginOpts.Get("dscp"); loaded {
		if n, atoiErr := strconv.Atoi(value); atoiErr == nil {
			config.DSCP = n
		}
	}
	if value, loaded := pluginOpts.Get("nocomp"); loaded && value != "false" {
		config.NoComp = true
	}
	if value, loaded := pluginOpts.Get("acknodelay"); loaded && value != "false" {
		config.AckNodelay = true
	}
	if value, loaded := pluginOpts.Get("nodelay"); loaded {
		if n, atoiErr := strconv.Atoi(value); atoiErr == nil {
			config.NoDelay = n
		}
	}
	if value, loaded := pluginOpts.Get("interval"); loaded {
		if n, atoiErr := strconv.Atoi(value); atoiErr == nil {
			config.Interval = n
		}
	}
	if value, loaded := pluginOpts.Get("resend"); loaded {
		if n, atoiErr := strconv.Atoi(value); atoiErr == nil {
			config.Resend = n
		}
	}
	if value, loaded := pluginOpts.Get("nc"); loaded {
		if n, atoiErr := strconv.Atoi(value); atoiErr == nil {
			config.NoCongestion = n
		}
	}
	if value, loaded := pluginOpts.Get("sockbuf"); loaded {
		if n, atoiErr := strconv.Atoi(value); atoiErr == nil {
			config.SockBuf = n
		}
	}
	if value, loaded := pluginOpts.Get("smuxver"); loaded {
		if n, atoiErr := strconv.Atoi(value); atoiErr == nil {
			config.SmuxVer = n
		}
	}
	if value, loaded := pluginOpts.Get("smuxbuf"); loaded {
		if n, atoiErr := strconv.Atoi(value); atoiErr == nil {
			config.SmuxBuf = n
		}
	}
	if value, loaded := pluginOpts.Get("framesize"); loaded {
		if n, atoiErr := strconv.Atoi(value); atoiErr == nil {
			config.FrameSize = n
		}
	}
	if value, loaded := pluginOpts.Get("streambuf"); loaded {
		if n, atoiErr := strconv.Atoi(value); atoiErr == nil {
			config.StreamBuf = n
		}
	}
	if value, loaded := pluginOpts.Get("keepalive"); loaded {
		if n, atoiErr := strconv.Atoi(value); atoiErr == nil {
			config.KeepAlive = n
		}
	}
	config.fillDefaults()

	pluginCtx, cancel := context.WithCancel(context.Background())
	plugin := &KcpTunPlugin{
		config: *config,
		block:  config.newBlock(),
		ctx:    pluginCtx,
		cancel: cancel,
	}
	plugin.dialFn = func(ctx context.Context) (net.PacketConn, net.Addr, error) {
		packetConn, err := dialer.ListenPacket(ctx, serverAddr)
		if err != nil {
			return nil, nil, err
		}
		return packetConn, serverAddr.UDPAddr(), nil
	}
	return plugin, nil
}

type DialFn func(ctx context.Context) (net.PacketConn, net.Addr, error)

func (c *KcpTunPlugin) Close() error {
	c.cancel()
	return nil
}

func (c *KcpTunPlugin) createConn(ctx context.Context, dial DialFn) (*smux.Session, error) {
	conn, addr, err := dial(ctx)
	if err != nil {
		return nil, err
	}

	config := c.config
	convid := rand.Uint32()
	kcpconn, err := kcp.NewConn4(convid, addr, c.block, config.DataShard, config.ParityShard, true, conn)
	if err != nil {
		return nil, err
	}
	kcpconn.SetStreamMode(true)
	kcpconn.SetWriteDelay(false)
	kcpconn.SetNoDelay(config.NoDelay, config.Interval, config.Resend, config.NoCongestion)
	kcpconn.SetWindowSize(config.SndWnd, config.RcvWnd)
	kcpconn.SetMtu(config.MTU)
	kcpconn.SetACKNoDelay(config.AckNodelay)
	kcpconn.SetRateLimit(uint32(config.RateLimit))

	_ = kcpconn.SetDSCP(config.DSCP)
	_ = kcpconn.SetReadBuffer(config.SockBuf)
	_ = kcpconn.SetWriteBuffer(config.SockBuf)
	smuxConfig := smux.DefaultConfig()
	smuxConfig.Version = config.SmuxVer
	smuxConfig.MaxReceiveBuffer = config.SmuxBuf
	smuxConfig.MaxStreamBuffer = config.StreamBuf
	smuxConfig.MaxFrameSize = config.FrameSize
	smuxConfig.KeepAliveInterval = time.Duration(config.KeepAlive) * time.Second
	if smuxConfig.KeepAliveInterval >= smuxConfig.KeepAliveTimeout {
		smuxConfig.KeepAliveTimeout = 3 * smuxConfig.KeepAliveInterval
	}

	if err := smux.VerifyConfig(smuxConfig); err != nil {
		return nil, err
	}

	var netConn net.Conn = kcpconn
	if !config.NoComp {
		netConn = NewCompStream(netConn)
	}
	// stream multiplex
	return smux.Client(netConn, smuxConfig)
}

func (c *KcpTunPlugin) DialContext(ctx context.Context) (net.Conn, error) {
	c.once.Do(func() {
		// start scavenger if autoexpire is set
		c.chScavenger = make(chan timedSession, 128)
		if c.config.AutoExpire > 0 {
			go scavenger(c.ctx, c.chScavenger, &c.config)
		}

		c.numconn = uint16(c.config.Conn)
		c.muxes = make([]timedSession, c.config.Conn)
		c.rr = uint16(0)
	})

	c.connMu.Lock()
	idx := c.rr % c.numconn

	// do auto expiration && reconnection
	if c.muxes[idx].session == nil || c.muxes[idx].session.IsClosed() ||
		(c.config.AutoExpire > 0 && time.Now().After(c.muxes[idx].expiryDate)) {
		var err error
		c.muxes[idx].session, err = c.createConn(ctx, c.dialFn)
		if err != nil {
			c.connMu.Unlock()
			return nil, err
		}
		c.muxes[idx].expiryDate = time.Now().Add(time.Duration(c.config.AutoExpire) * time.Second)
		if c.config.AutoExpire > 0 { // only when autoexpire set
			c.chScavenger <- c.muxes[idx]
		}
	}
	c.rr++
	session := c.muxes[idx].session
	c.connMu.Unlock()

	return session.OpenStream()
}

// timedSession is a wrapper for smux.Session with expiry date
type timedSession struct {
	session    *smux.Session
	expiryDate time.Time
}

// scavenger goroutine is used to close expired sessions
func scavenger(ctx context.Context, ch chan timedSession, config *kcpTunConfig) {
	ticker := time.NewTicker(kcpTunScavengePeriod * time.Second)
	defer ticker.Stop()
	var sessionList []timedSession
	for {
		select {
		case item := <-ch:
			sessionList = append(sessionList, timedSession{
				item.session,
				item.expiryDate.Add(time.Duration(config.ScavengeTTL) * time.Second),
			})
		case <-ticker.C:
			var newList []timedSession
			for k := range sessionList {
				s := sessionList[k]
				if s.session.IsClosed() {
					continue
				} else if time.Now().After(s.expiryDate) {
					s.session.Close()
				} else {
					newList = append(newList, sessionList[k])
				}
			}
			sessionList = newList
		case <-ctx.Done():
			return
		}
	}
}

// CompStream is a net.Conn wrapper that compresses data using snappy
type CompStream struct {
	conn net.Conn
	w    *snappy.Writer
	r    *snappy.Reader
}

func (c *CompStream) Read(p []byte) (n int, err error) {
	return c.r.Read(p)
}

func (c *CompStream) Write(p []byte) (n int, err error) {
	if _, err := c.w.Write(p); err != nil {
		return 0, err
	}

	if err := c.w.Flush(); err != nil {
		return 0, err
	}
	return len(p), err
}

func (c *CompStream) Close() error {
	return c.conn.Close()
}

func (c *CompStream) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *CompStream) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *CompStream) SetDeadline(t time.Time) error {
	return c.conn.SetDeadline(t)
}

func (c *CompStream) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

func (c *CompStream) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}

// NewCompStream creates a new stream that compresses data using snappy
func NewCompStream(conn net.Conn) *CompStream {
	c := new(CompStream)
	c.conn = conn
	c.w = snappy.NewBufferedWriter(conn)
	c.r = snappy.NewReader(conn)
	return c
}
