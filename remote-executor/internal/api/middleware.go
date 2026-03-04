package api

import "net/http"

// MTLSMiddleware enforces mTLS client certificate verification.
// The actual TLS handshake is handled at the server level;
// this middleware provides additional checks if needed.
func MTLSMiddleware(next http.Handler) http.Handler {
	// TODO: implement in Phase 3.2
	return next
}
