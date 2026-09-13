package masque

import (
	mquic "github.com/metacubex/quic-go"
	squic "github.com/sagernet/quic-go"
	shttp3 "github.com/sagernet/quic-go/http3"

	connectip "github.com/metacubex/connect-ip-go"
)

// proxiedStream adapts a sagernet/quic-go HTTP/3 request stream to the
// metacubex/quic-go based stream interface expected by connect-ip-go. The
// two forks share the same wire protocol; only the StreamErrorCode named
// types differ, so the adapter converts between them.
type proxiedStream struct {
	*shttp3.RequestStream
}

func (s *proxiedStream) CancelRead(code mquic.StreamErrorCode) {
	s.RequestStream.CancelRead(squic.StreamErrorCode(code))
}

var _ connectip.Http3Stream = (*proxiedStream)(nil)
