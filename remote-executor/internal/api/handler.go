package api

import "net/http"

// Handler holds dependencies for HTTP request handlers.
type Handler struct{}

// NewHandler creates a new Handler.
func NewHandler() *Handler {
	// TODO: implement in Phase 3.1
	return &Handler{}
}

// ListScripts handles GET /api/v1/scripts.
func (h *Handler) ListScripts(w http.ResponseWriter, r *http.Request) {
	// TODO: implement in Phase 3.1
	w.WriteHeader(http.StatusNotImplemented)
}

// Execute handles POST /api/v1/execute.
func (h *Handler) Execute(w http.ResponseWriter, r *http.Request) {
	// TODO: implement in Phase 3.1
	w.WriteHeader(http.StatusNotImplemented)
}

// ListRecords handles GET /api/v1/records.
func (h *Handler) ListRecords(w http.ResponseWriter, r *http.Request) {
	// TODO: implement in Phase 3.1
	w.WriteHeader(http.StatusNotImplemented)
}

// GetRecord handles GET /api/v1/records/{record_id}.
func (h *Handler) GetRecord(w http.ResponseWriter, r *http.Request) {
	// TODO: implement in Phase 3.1
	w.WriteHeader(http.StatusNotImplemented)
}
