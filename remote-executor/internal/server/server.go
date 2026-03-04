package server

// Server wraps the HTTP server with mTLS configuration.
type Server struct{}

// Config holds server startup configuration.
type Config struct {
	Addr       string
	CACert     string
	ServerCert string
	ServerKey  string
}

// New creates a new Server.
func New(cfg Config) *Server {
	// TODO: implement in Phase 3.2
	return &Server{}
}

// Start launches the mTLS HTTP server and blocks until it exits.
func (s *Server) Start() error {
	// TODO: implement in Phase 3.2
	return nil
}
