// Command holt-web serves the holt terminal in a web browser. It bridges a
// pseudo-terminal (PTY) running `holt` to the browser over a WebSocket and
// renders it with xterm.js, so the full TUI works from any browser.
//
// SECURITY: this spawns a real, interactive agent that can run shell commands
// and edit files with your provider API keys. Expose it only behind an auth
// token (HOLT_WEB_TOKEN / --token) and run it inside an isolated, disposable
// environment such as a container. Never point it at a directory you are not
// willing to hand to arbitrary visitors.
//
// It needs a persistent process and a real PTY, so it CANNOT run on serverless
// platforms such as Vercel. Deploy it to a container host (Railway, Fly.io,
// Render, or any Docker/VPS) using the provided Dockerfile.
package main

import (
	"context"
	"crypto/subtle"
	_ "embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
)

//go:embed index.html
var indexHTML []byte

type options struct {
	addr     string
	token    string
	holtBin  string
	workdir  string
	holtArgs []string
}

func parseOptions(args []string, getenv func(string) string) (options, error) {
	fs := flag.NewFlagSet("holt-web", flag.ContinueOnError)
	addr := fs.String("addr", "", "listen address (default :$PORT or :8080)")
	token := fs.String("token", "", "shared auth token required to open a session (default $HOLT_WEB_TOKEN)")
	holtBin := fs.String("holt", "", "path to the holt binary (default $HOLT_WEB_BIN or \"holt\" on PATH)")
	workdir := fs.String("workdir", "", "working directory holt runs in (default $HOLT_WEB_WORKDIR or the current directory)")
	if err := fs.Parse(args); err != nil {
		return options{}, err
	}

	opt := options{
		addr:     firstNonEmpty(*addr, portAddr(getenv("PORT")), ":8080"),
		token:    firstNonEmpty(*token, getenv("HOLT_WEB_TOKEN")),
		holtBin:  firstNonEmpty(*holtBin, getenv("HOLT_WEB_BIN"), "holt"),
		workdir:  firstNonEmpty(*workdir, getenv("HOLT_WEB_WORKDIR")),
		holtArgs: fs.Args(),
	}
	return opt, nil
}

func main() {
	opt, err := parseOptions(os.Args[1:], os.Getenv)
	if err != nil {
		os.Exit(2)
	}
	if err := run(opt); err != nil {
		log.Fatalf("holt-web: %v", err)
	}
}

func run(opt options) error {
	if opt.token == "" {
		log.Printf("WARNING: no auth token set (HOLT_WEB_TOKEN/--token). Anyone who reaches this server gets a live holt session. Set a token before exposing it publicly.")
	}

	srv := &server{opt: opt, upgrader: websocket.Upgrader{
		// The token check below is the access control; a browser same-origin
		// policy does not protect a shell, so we do not rely on Origin here.
		CheckOrigin: func(*http.Request) bool { return true },
	}}

	mux := http.NewServeMux()
	mux.HandleFunc("/", srv.handleIndex)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "ok")
	})
	mux.HandleFunc("/ws", srv.handleWS)

	httpServer := &http.Server{
		Addr:              opt.addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Printf("holt-web listening on %s (holt=%q workdir=%q auth=%v)", opt.addr, opt.holtBin, opt.workdir, opt.token != "")
	return httpServer.ListenAndServe()
}

type server struct {
	opt      options
	upgrader websocket.Upgrader
}

func (s *server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(indexHTML)
}

// authorized reports whether the request carries the configured token. When no
// token is configured every request is allowed (see the startup warning).
func (s *server) authorized(r *http.Request) bool {
	if s.opt.token == "" {
		return true
	}
	got := r.URL.Query().Get("token")
	if got == "" {
		got = strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(s.opt.token)) == 1
}

func (s *server) handleWS(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return // Upgrade already wrote the error response.
	}
	defer conn.Close()

	if err := s.bridge(r.Context(), conn); err != nil && !isExpectedClose(err) {
		log.Printf("session ended: %v", err)
	}
}

// bridge spawns holt in a PTY and pipes it to the WebSocket until either side
// closes. The PTY reader goroutine is the only writer to the connection.
func (s *server) bridge(ctx context.Context, conn *websocket.Conn) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	cmd := exec.CommandContext(ctx, s.opt.holtBin, s.opt.holtArgs...)
	cmd.Dir = s.opt.workdir
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	ptmx, err := pty.Start(cmd)
	if err != nil {
		return fmt.Errorf("start holt: %w", err)
	}
	defer func() { _ = ptmx.Close() }()

	var writeMu sync.Mutex
	writeErr := make(chan error, 1)
	go func() {
		buf := make([]byte, 32*1024)
		for {
			n, readErr := ptmx.Read(buf)
			if n > 0 {
				writeMu.Lock()
				err := conn.WriteMessage(websocket.BinaryMessage, buf[:n])
				writeMu.Unlock()
				if err != nil {
					writeErr <- err
					return
				}
			}
			if readErr != nil {
				writeErr <- readErr // io.EOF when holt exits
				return
			}
		}
	}()

	// Read loop: control frames (resize) and input from the browser.
	go func() {
		for {
			_, data, readErr := conn.ReadMessage()
			if readErr != nil {
				cancel()
				return
			}
			if len(data) == 0 {
				continue
			}
			switch data[0] {
			case msgInput:
				if _, err := ptmx.Write(data[1:]); err != nil {
					cancel()
					return
				}
			case msgResize:
				applyResize(ptmx, data[1:])
			}
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-writeErr:
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}
}

const (
	msgInput  = '0'
	msgResize = '1'
)

type resizeMessage struct {
	Cols uint16 `json:"cols"`
	Rows uint16 `json:"rows"`
}

func applyResize(ptmx *os.File, payload []byte) {
	var msg resizeMessage
	if err := json.Unmarshal(payload, &msg); err != nil || msg.Cols == 0 || msg.Rows == 0 {
		return
	}
	_ = pty.Setsize(ptmx, &pty.Winsize{Rows: msg.Rows, Cols: msg.Cols})
}

func isExpectedClose(err error) bool {
	if err == nil || errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) {
		return true
	}
	return websocket.IsCloseError(err,
		websocket.CloseNormalClosure,
		websocket.CloseGoingAway,
		websocket.CloseNoStatusReceived,
		websocket.CloseAbnormalClosure)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// portAddr turns a bare PORT value ("8080") into a listen address (":8080").
func portAddr(port string) string {
	if port == "" {
		return ""
	}
	if strings.Contains(port, ":") {
		return port
	}
	return ":" + port
}
