package gost

import (
	"io"
	"net"
	"os"
	"time"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing/common"
	E "github.com/sagernet/sing/common/exceptions"
	"github.com/sagernet/ws"
	"github.com/sagernet/ws/wsutil"
)

// WebsocketConn is a stream-oriented websocket connection carrying binary frames.
type WebsocketConn struct {
	net.Conn
	state          ws.State
	reader         *wsutil.Reader
	controlHandler wsutil.FrameHandlerFunc
	remoteAddr     net.Addr
}

func NewWebsocketConn(conn net.Conn, remoteAddr net.Addr, state ws.State) *WebsocketConn {
	controlHandler := wsutil.ControlFrameHandler(conn, state)
	return &WebsocketConn{
		Conn:  conn,
		state: state,
		reader: &wsutil.Reader{
			Source:          conn,
			State:           state,
			SkipHeaderCheck: true,
			CheckUTF8:       false,
			OnIntermediate:  controlHandler,
		},
		controlHandler: controlHandler,
		remoteAddr:     remoteAddr,
	}
}

func (c *WebsocketConn) Read(b []byte) (n int, err error) {
	var header ws.Header
	for {
		n, err = c.reader.Read(b)
		if n > 0 {
			err = nil
			return
		}
		if err != nil && err != io.EOF && err != wsutil.ErrNoFrameAdvance {
			err = wrapWsError(err)
			return
		}
		header, err = wrapWsError0(c.reader.NextFrame())
		if err != nil {
			return
		}
		if header.OpCode.IsControl() {
			if header.Length > 128 {
				err = wsutil.ErrFrameTooLarge
				return
			}
			err = wrapWsError(c.controlHandler(header, c.reader))
			if err != nil {
				return
			}
			continue
		}
		if header.OpCode&ws.OpBinary == 0 {
			err = wrapWsError(c.reader.Discard())
			if err != nil {
				return
			}
			continue
		}
	}
}

func (c *WebsocketConn) Write(p []byte) (n int, err error) {
	err = wrapWsError(wsutil.WriteMessage(c.Conn, c.state, ws.OpBinary, p))
	if err != nil {
		return
	}
	n = len(p)
	return
}

func (c *WebsocketConn) Close() error {
	c.Conn.SetWriteDeadline(time.Now().Add(C.TCPTimeout))
	frame := ws.NewCloseFrame(ws.NewCloseFrameBody(
		ws.StatusNormalClosure, "",
	))
	if c.state == ws.StateClientSide {
		frame = ws.MaskFrameInPlace(frame)
	}
	ws.WriteFrame(c.Conn, frame)
	c.Conn.Close()
	return nil
}

func (c *WebsocketConn) RemoteAddr() net.Addr {
	if c.remoteAddr != nil {
		return c.remoteAddr
	}
	return c.Conn.RemoteAddr()
}

func (c *WebsocketConn) SetDeadline(t time.Time) error {
	return os.ErrInvalid
}

func (c *WebsocketConn) SetReadDeadline(t time.Time) error {
	return os.ErrInvalid
}

func (c *WebsocketConn) SetWriteDeadline(t time.Time) error {
	return os.ErrInvalid
}

func (c *WebsocketConn) NeedAdditionalReadDeadline() bool {
	return true
}

func (c *WebsocketConn) Upstream() any {
	return c.Conn
}

func wrapWsError(err error) error {
	if err == nil {
		return nil
	}
	if _, ok := err.(net.Error); ok {
		return err
	}
	return E.Cause(err, "websocket")
}

func wrapWsError0[T any](value T, err error) (T, error) {
	return value, wrapWsError(err)
}

var _ = common.Close
