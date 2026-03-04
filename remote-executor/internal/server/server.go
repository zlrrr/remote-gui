package server

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/remote-gui/remote-executor/internal/api"
	"github.com/remote-gui/remote-executor/internal/record"
	"github.com/remote-gui/remote-executor/internal/runner"
	"github.com/remote-gui/remote-executor/internal/script"
)

// Config holds server startup configuration.
type Config struct {
	Addr       string
	CACert     string
	ServerCert string
	ServerKey  string
	Registry   script.Registry
	Runner     runner.Runner
	Store      record.Store
}

// Server wraps the HTTP server with mTLS configuration.
type Server struct {
	cfg     Config
	handler *api.Handler
}

// New creates a new Server.
func New(cfg Config) *Server {
	h := api.NewHandler(cfg.Registry, cfg.Runner, cfg.Store)
	return &Server{cfg: cfg, handler: h}
}

// buildTLSConfig creates a TLS config requiring mTLS (client certificate verification).
func buildTLSConfig(caCertPath, certPath, keyPath string) (*tls.Config, error) {
	caCert, err := os.ReadFile(caCertPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA cert %q: %w", caCertPath, err)
	}

	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse CA cert")
	}

	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load server cert/key: %w", err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    caPool,
		MinVersion:   tls.VersionTLS13,
	}, nil
}

// buildRouter wires up all API routes.
func (s *Server) buildRouter() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/scripts", s.handler.ListScripts)
		r.Post("/execute", s.handler.Execute)
		r.Get("/records", s.handler.ListRecords)
		r.Get("/records/{record_id}", s.handler.GetRecord)
	})

	return r
}

// Start launches the mTLS HTTP server and blocks until it exits.
func (s *Server) Start() error {
	tlsCfg, err := buildTLSConfig(s.cfg.CACert, s.cfg.ServerCert, s.cfg.ServerKey)
	if err != nil {
		return fmt.Errorf("failed to build TLS config: %w", err)
	}

	httpSrv := &http.Server{
		Addr:      s.cfg.Addr,
		Handler:   s.buildRouter(),
		TLSConfig: tlsCfg,
	}

	// cert/key are already loaded into TLSConfig; pass empty strings to ListenAndServeTLS
	return httpSrv.ListenAndServeTLS("", "")
}

// BuildRouterForTest returns the router without TLS for use with httptest servers.
func (s *Server) BuildRouterForTest() http.Handler {
	return s.buildRouter()
}
