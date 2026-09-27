package v2rayxhttp

import (
	"bytes"
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/json/badoption"
	"github.com/sagernet/sing/common/logger"

	"github.com/stretchr/testify/require"
)

func TestServeHTTPHostIgnoresPort(t *testing.T) {
	server, err := NewServer(context.Background(), logger.NOP(), option.V2RayXHTTPOptions{Path: "/xhttp", Host: "example.com"}, nil, echoHandler{})
	require.NoError(t, err)
	t.Cleanup(func() { _ = server.Close() })
	for _, testCase := range []struct {
		name   string
		host   string
		status int
	}{
		{"port-suffix", "example.com:8443", http.StatusOK},
		{"bare", "example.com", http.StatusOK},
		{"empty-port", "example.com:", http.StatusNotFound},
		{"out-of-range-port", "example.com:99999", http.StatusNotFound},
		{"other-host", "example.org:8443", http.StatusNotFound},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodOptions, "http://"+testCase.host+"/xhttp", nil)
			recorder := httptest.NewRecorder()
			server.ServeHTTP(recorder, request)
			require.Equal(t, testCase.status, recorder.Code)
		})
	}
}

func TestServeHTTPSessionlessModes(t *testing.T) {
	for _, testCase := range []struct {
		mode   string
		status int
	}{
		{"packet-up", http.StatusBadRequest},
		{"stream-one", http.StatusOK},
		{"stream-up", http.StatusOK},
		{"auto", http.StatusOK},
	} {
		t.Run(testCase.mode, func(t *testing.T) {
			server, err := NewServer(context.Background(), logger.NOP(), option.V2RayXHTTPOptions{Path: "/xhttp", Mode: testCase.mode}, nil, echoHandler{})
			require.NoError(t, err)
			t.Cleanup(func() { _ = server.Close() })
			// A session-less request carries padding in the request URL, so it
			// passes the padding check without a Referer header. The streaming
			// path only returns once the client disconnects, so cancel the
			// context to emulate that.
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			request := httptest.NewRequest(http.MethodGet, "http://example.com/xhttp/?x_padding="+strings.Repeat("X", 200), nil)
			recorder := httptest.NewRecorder()
			server.ServeHTTP(recorder, request.WithContext(ctx))
			require.Equal(t, testCase.status, recorder.Code)
		})
	}
}

func TestServeHTTPUploadStatusCodes(t *testing.T) {
	newServer := func(t *testing.T, options option.V2RayXHTTPOptions) *Server {
		t.Helper()
		server, err := NewServer(context.Background(), logger.NOP(), options, nil, echoHandler{})
		require.NoError(t, err)
		t.Cleanup(func() { _ = server.Close() })
		return server
	}
	// paddedUpload builds an upload request carrying a valid padding value and
	// the session/sequence metadata expected by the base options below.
	paddedUpload := func(server *Server, session, sequence string) *http.Request {
		request := httptest.NewRequest(http.MethodPost, "http://example.com/xhttp/", nil)
		server.config.applyPadding(request)
		request.Header.Set("X-Session", session)
		request.Header.Set("X-Seq", sequence)
		return request
	}
	baseOptions := option.V2RayXHTTPOptions{
		Path:               "/xhttp",
		Mode:               "packet-up",
		SessionIDPlacement: placementHeader,
		SessionIDKey:       "X-Session",
		SeqPlacement:       placementHeader,
		SeqKey:             "X-Seq",
	}

	t.Run("unreadable-payload", func(t *testing.T) {
		options := baseOptions
		options.UplinkDataPlacement = placementHeader
		server := newServer(t, options)
		request := paddedUpload(server, "unreadable", "0")
		request.Header.Set(server.config.dataKey+"-0", "!!!not-base64!!!")
		recorder := httptest.NewRecorder()
		server.ServeHTTP(recorder, request)
		require.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("payload-too-large", func(t *testing.T) {
		options := baseOptions
		options.UplinkDataPlacement = placementHeader
		options.SCMaxEachPostBytes = option.V2RayXHTTPRange{From: 8, To: 8}
		server := newServer(t, options)
		request := paddedUpload(server, "large", "0")
		request.Header.Set(server.config.dataKey+"-0", base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte("a"), 16)))
		recorder := httptest.NewRecorder()
		server.ServeHTTP(recorder, request)
		require.Equal(t, http.StatusRequestEntityTooLarge, recorder.Code)
	})

	t.Run("unparsable-sequence", func(t *testing.T) {
		server := newServer(t, baseOptions)
		request := paddedUpload(server, "sequence", "not-a-number")
		recorder := httptest.NewRecorder()
		server.ServeHTTP(recorder, request)
		require.Equal(t, http.StatusInternalServerError, recorder.Code)
	})

	t.Run("failed-push", func(t *testing.T) {
		server := newServer(t, baseOptions)
		// An unbuffered session whose download side is already gone makes the
		// push fail deterministically instead of racing the packet buffer.
		server.config.maxBufferedPosts = 0
		session := server.session("closed-session")
		session.close()
		request := paddedUpload(server, "closed-session", "0")
		recorder := httptest.NewRecorder()
		server.ServeHTTP(recorder, request)
		require.Equal(t, http.StatusInternalServerError, recorder.Code)
	})
}

func TestServeHTTPSecondUploadStreamRefused(t *testing.T) {
	server, err := NewServer(context.Background(), logger.NOP(), option.V2RayXHTTPOptions{
		Path:               "/xhttp",
		Mode:               "stream-up",
		SessionIDPlacement: placementHeader,
		SessionIDKey:       "X-Session",
	}, nil, echoHandler{})
	require.NoError(t, err)
	t.Cleanup(func() { _ = server.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	first := httptest.NewRequest(http.MethodPost, "http://example.com/xhttp/?x_padding="+strings.Repeat("X", 200), nil)
	first.Header.Set("X-Session", "shared")
	firstRecorder := httptest.NewRecorder()
	firstDone := make(chan struct{})
	go func() {
		defer close(firstDone)
		server.ServeHTTP(firstRecorder, first.WithContext(ctx))
	}()
	waitForLifecycle(t, 5*time.Second, func() bool {
		session, loaded := server.sessions.Load("shared")
		return loaded && session.(*serverSession).streamStarted.Load()
	})

	second := httptest.NewRequest(http.MethodPost, "http://example.com/xhttp/?x_padding="+strings.Repeat("X", 200), nil)
	second.Header.Set("X-Session", "shared")
	secondRecorder := httptest.NewRecorder()
	server.ServeHTTP(secondRecorder, second)
	require.Equal(t, http.StatusConflict, secondRecorder.Code)

	cancel()
	<-firstDone
	require.Equal(t, http.StatusOK, firstRecorder.Code)
}

func TestSourceAddressTrustedForwardedFor(t *testing.T) {
	baseOptions := option.V2RayXHTTPOptions{Path: "/xhttp"}
	newRequest := func() *http.Request {
		request := httptest.NewRequest(http.MethodGet, "http://example.com/xhttp/", nil)
		request.RemoteAddr = "9.9.9.9:1234"
		request.Header.Set("X-Forwarded-For", "1.2.3.4")
		return request
	}
	t.Run("untrusted-ignored", func(t *testing.T) {
		options := baseOptions
		options.TrustedXForwardedFor = badoption.Listable[string]{"X-Real-IP"}
		server, err := NewServer(context.Background(), logger.NOP(), options, nil, echoHandler{})
		require.NoError(t, err)
		t.Cleanup(func() { _ = server.Close() })
		address := server.sourceAddress(newRequest())
		require.Equal(t, "9.9.9.9", address.Addr.String())
		require.Equal(t, uint16(1234), address.Port)
	})
	t.Run("trusted-honoured", func(t *testing.T) {
		options := baseOptions
		options.TrustedXForwardedFor = badoption.Listable[string]{"X-Real-IP"}
		server, err := NewServer(context.Background(), logger.NOP(), options, nil, echoHandler{})
		require.NoError(t, err)
		t.Cleanup(func() { _ = server.Close() })
		request := newRequest()
		request.Header.Set("X-Real-IP", "203.0.113.7")
		address := server.sourceAddress(request)
		require.Equal(t, "1.2.3.4", address.Addr.String())
		require.Equal(t, uint16(1234), address.Port)
	})
	t.Run("empty-trusted-header-ignored", func(t *testing.T) {
		options := baseOptions
		options.TrustedXForwardedFor = badoption.Listable[string]{"X-Real-IP"}
		server, err := NewServer(context.Background(), logger.NOP(), options, nil, echoHandler{})
		require.NoError(t, err)
		t.Cleanup(func() { _ = server.Close() })
		request := newRequest()
		request.Header.Set("X-Real-IP", "")
		address := server.sourceAddress(request)
		require.Equal(t, "9.9.9.9", address.Addr.String())
	})
	t.Run("no-trusted-headers-configured", func(t *testing.T) {
		server, err := NewServer(context.Background(), logger.NOP(), baseOptions, nil, echoHandler{})
		require.NoError(t, err)
		t.Cleanup(func() { _ = server.Close() })
		address := server.sourceAddress(newRequest())
		require.Equal(t, "9.9.9.9", address.Addr.String())
	})
}
