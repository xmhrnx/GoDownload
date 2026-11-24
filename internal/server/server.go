// Package server is GoDownload's local HTTP server on 127.0.0.1. It takes
// downloads from the browser extension and serves downloads to media
// players while they download.
package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/godownload/godownload/internal/manager"
)

// Version is reported to the extension.
const Version = "1.0.0"

// Config sets up the server.
type Config struct {
	Port int
	// Token, when set, must be sent by the extension as a bearer token.
	Token string
	// Intercept tells the extension whether to capture browser downloads.
	Intercept func() bool
	// OnAdd receives links from the browser. batch is true for "download all
	// links", which the UI shows as a picker.
	OnAdd func(items []Link, batch bool)
}

// Server is the running local server.
type Server struct {
	cfg   Config
	m     *manager.Manager
	srv   *http.Server
	ln    net.Listener
	token string // in stream URLs, so other local pages can't guess them

	mu sync.Mutex
}

// Start listens on 127.0.0.1:Port.
func Start(cfg Config, m *manager.Manager) (*Server, error) {
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", cfg.Port))
	if err != nil {
		return nil, fmt.Errorf("port %d is in use; change it in Settings › Browser: %w", cfg.Port, err)
	}
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		ln.Close()
		return nil, err
	}
	s := &Server{cfg: cfg, m: m, ln: ln, token: hex.EncodeToString(b)}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/", s.api)
	mux.HandleFunc("GET /stream/{token}/{id}/{file}/{name...}", s.stream)
	mux.HandleFunc("GET /status/{token}/{id}/{file}", s.status)
	s.srv = &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go s.srv.Serve(ln)
	return s, nil
}

// SetToken changes the pairing token.
func (s *Server) SetToken(t string) {
	s.mu.Lock()
	s.cfg.Token = t
	s.mu.Unlock()
}

func (s *Server) pairingToken() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cfg.Token
}

// Port is the port the server listens on.
func (s *Server) Port() int { return s.ln.Addr().(*net.TCPAddr).Port }

// Close stops the server.
func (s *Server) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err := s.srv.Shutdown(ctx)
	if errors.Is(err, context.DeadlineExceeded) {
		return s.srv.Close()
	}
	return err
}
