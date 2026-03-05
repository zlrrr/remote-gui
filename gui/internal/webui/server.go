package webui

import (
	"embed"
	"net/http"
)

//go:embed web/index.html
var webFS embed.FS

// NewServer creates and returns an http.ServeMux wired up with all
// web UI and API routes. configPath is the path to gui.yaml on disk.
func NewServer(configPath string) *http.ServeMux {
	h := &handler{configPath: configPath}
	mux := http.NewServeMux()

	// Serve the embedded SPA
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		data, _ := webFS.ReadFile("web/index.html")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(data)
	})

	// Config CRUD
	mux.HandleFunc("GET /api/config", h.getConfig)
	mux.HandleFunc("POST /api/config", h.saveConfig)

	// Execute proxy
	mux.HandleFunc("POST /api/execute", h.execute)

	// Scripts list proxy (for connection test in Settings)
	mux.HandleFunc("GET /api/scripts", h.listScripts)

	return mux
}
