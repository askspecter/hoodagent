package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func newTestServer(t *testing.T, token string) (*httptest.Server, string) {
	t.Helper()
	// /bin/cat echoes stdin to stdout through the PTY, which is enough to prove
	// the input -> PTY -> output path end to end.
	srv := &server{
		opt:      options{holtBin: "/bin/cat", token: token},
		upgrader: websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", srv.handleIndex)
	mux.HandleFunc("/ws", srv.handleWS)
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws"
	return ts, wsURL
}

func TestIndexServesFrontend(t *testing.T) {
	ts, _ := newTestServer(t, "")
	resp, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer resp.Body.Close()
	buf := make([]byte, len(indexHTML))
	_, _ = resp.Body.Read(buf)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if !strings.Contains(string(indexHTML), "HOLT") || !strings.Contains(string(indexHTML), "/ws") {
		t.Fatal("embedded index.html missing HOLT branding or /ws client")
	}
}

func TestWebSocketBridgesPTYInputToOutput(t *testing.T) {
	_, wsURL := newTestServer(t, "")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	// A resize control frame must be accepted without tearing down the session.
	if err := conn.WriteMessage(websocket.TextMessage, []byte(string(msgResize)+`{"cols":80,"rows":24}`)); err != nil {
		t.Fatalf("write resize: %v", err)
	}
	if err := conn.WriteMessage(websocket.TextMessage, []byte(string(msgInput)+"holt-web-smoke\n")); err != nil {
		t.Fatalf("write input: %v", err)
	}

	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	var got strings.Builder
	for got.Len() < 4096 {
		_, data, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("read: %v (got so far: %q)", err, got.String())
		}
		got.Write(data)
		if strings.Contains(got.String(), "holt-web-smoke") {
			return // echoed back through the PTY: bridge works.
		}
	}
	t.Fatalf("did not observe echoed input; got %q", got.String())
}

func TestWebSocketRequiresTokenWhenConfigured(t *testing.T) {
	_, wsURL := newTestServer(t, "s3cret")

	_, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err == nil {
		t.Fatal("expected handshake to be rejected without a token")
	}
	if resp == nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %v, want 401", resp)
	}

	conn, _, err := websocket.DefaultDialer.Dial(wsURL+"?token=s3cret", nil)
	if err != nil {
		t.Fatalf("dial with token: %v", err)
	}
	conn.Close()
}

func TestParseOptionsDefaults(t *testing.T) {
	env := map[string]string{"PORT": "9000"}
	opt, err := parseOptions(nil, func(k string) string { return env[k] })
	if err != nil {
		t.Fatalf("parseOptions: %v", err)
	}
	if opt.addr != ":9000" {
		t.Fatalf("addr = %q, want :9000", opt.addr)
	}
	if opt.holtBin != "holt" {
		t.Fatalf("holtBin = %q, want holt", opt.holtBin)
	}
}
