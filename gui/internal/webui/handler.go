package webui

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/remote-gui/gui/internal/client"
	"github.com/remote-gui/gui/internal/config"
	"gopkg.in/yaml.v3"
)

type handler struct {
	configPath string
}

// jsonError writes a JSON error response.
func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// jsonOK writes a JSON success response.
func jsonOK(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

// loadConfig reads and parses the config file, returning an empty config on
// ENOENT so the UI can bootstrap a fresh setup.
func (h *handler) loadConfig() (*config.Config, error) {
	data, err := os.ReadFile(h.configPath)
	if os.IsNotExist(err) {
		return &config.Config{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("cannot read config: %w", err)
	}
	var cfg config.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid YAML: %w", err)
	}
	return &cfg, nil
}

// getConfig handles GET /api/config
func (h *handler) getConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.loadConfig()
	if err != nil {
		slog.Error("getConfig: load failed", "error", err)
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, cfg)
}

// saveConfig handles POST /api/config
func (h *handler) saveConfig(w http.ResponseWriter, r *http.Request) {
	var cfg config.Config
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		jsonError(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	const maxOps = 10
	const maxParams = 5
	if len(cfg.Operations) > maxOps {
		jsonError(w, fmt.Sprintf("too many operations: %d (max %d)", len(cfg.Operations), maxOps), http.StatusBadRequest)
		return
	}
	for _, op := range cfg.Operations {
		if len(op.Params) > maxParams {
			jsonError(w, fmt.Sprintf("operation %q has too many params: %d (max %d)", op.Alias, len(op.Params), maxParams), http.StatusBadRequest)
			return
		}
	}

	data, err := yaml.Marshal(&cfg)
	if err != nil {
		jsonError(w, "failed to serialize config: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if err := os.WriteFile(h.configPath, data, 0o644); err != nil {
		slog.Error("saveConfig: write failed", "path", h.configPath, "error", err)
		jsonError(w, "failed to write config file: "+err.Error(), http.StatusInternalServerError)
		return
	}
	slog.Info("config saved", "path", h.configPath)
	jsonOK(w, map[string]bool{"ok": true})
}

// executeRequest is the body expected by POST /api/execute.
type executeRequest struct {
	Script string            `json:"script"`
	Params map[string]string `json:"params"`
}

// execute handles POST /api/execute
func (h *handler) execute(w http.ResponseWriter, r *http.Request) {
	var req executeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.Script == "" {
		jsonError(w, "script is required", http.StatusBadRequest)
		return
	}

	// Re-read config on every execute so newly saved settings take effect
	// without restarting the process.
	cfg, err := h.loadConfig()
	if err != nil {
		jsonError(w, "config error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	slog.Debug("execute: creating executor client", "endpoint", cfg.Executor.Endpoint, "script", req.Script)

	ec, err := client.NewExecutorClient(client.ExecutorClientConfig{
		Endpoint:   cfg.Executor.Endpoint,
		CACert:     cfg.Executor.TLS.CACert,
		ClientCert: cfg.Executor.TLS.ClientCert,
		ClientKey:  cfg.Executor.TLS.ClientKey,
	})
	if err != nil {
		slog.Error("execute: client init failed", "error", err)
		jsonError(w, "failed to create executor client: "+err.Error(), http.StatusInternalServerError)
		return
	}

	result, err := ec.Execute(req.Script, req.Params)
	if err != nil {
		slog.Error("execute: execution failed", "script", req.Script, "error", err)
		jsonError(w, err.Error(), http.StatusBadGateway)
		return
	}

	slog.Info("execute: success", "script", req.Script, "exit_code", result.ExitCode, "duration_ms", result.DurationMs)
	jsonOK(w, result)
}

// listScripts handles GET /api/scripts (proxies to executor)
func (h *handler) listScripts(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.loadConfig()
	if err != nil {
		jsonError(w, "config error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if cfg.Executor.Endpoint == "" {
		jsonError(w, "executor endpoint not configured", http.StatusBadRequest)
		return
	}

	ec, err := client.NewExecutorClient(client.ExecutorClientConfig{
		Endpoint:   cfg.Executor.Endpoint,
		CACert:     cfg.Executor.TLS.CACert,
		ClientCert: cfg.Executor.TLS.ClientCert,
		ClientKey:  cfg.Executor.TLS.ClientKey,
	})
	if err != nil {
		jsonError(w, "failed to create executor client: "+err.Error(), http.StatusInternalServerError)
		return
	}

	scripts, err := ec.ListScripts()
	if err != nil {
		jsonError(w, "executor error: "+err.Error(), http.StatusBadGateway)
		return
	}
	jsonOK(w, map[string]any{"scripts": scripts})
}
