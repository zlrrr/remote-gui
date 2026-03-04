package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/remote-gui/remote-executor/internal/record"
	"github.com/remote-gui/remote-executor/internal/runner"
	"github.com/remote-gui/remote-executor/internal/script"
)

// ── Request / Response types ──────────────────────────────────────────────────

// ExecuteRequest is the body for POST /api/v1/execute.
type ExecuteRequest struct {
	Script string            `json:"script"`
	Params map[string]string `json:"params"`
}

// ExecuteResponse is the success body for POST /api/v1/execute.
type ExecuteResponse struct {
	RecordID   string    `json:"record_id"`
	Script     string    `json:"script"`
	Status     string    `json:"status"`
	ExitCode   int       `json:"exit_code"`
	Stdout     string    `json:"stdout"`
	Stderr     string    `json:"stderr"`
	DurationMs int64     `json:"duration_ms"`
	ExecutedAt time.Time `json:"executed_at"`
}

// ErrorResponse is the body for error responses.
type ErrorResponse struct {
	Error   string      `json:"error"`
	Details interface{} `json:"details,omitempty"`
}

// ValidationDetail describes a single parameter validation failure.
type ValidationDetail struct {
	Param  string `json:"param"`
	Reason string `json:"reason"`
}

// ScriptSummary is the per-script entry in ListScriptsResponse.
type ScriptSummary struct {
	Name           string              `json:"name"`
	Description    string              `json:"description"`
	TimeoutSeconds int                 `json:"timeout_seconds"`
	Params         []script.ParamSpec  `json:"params"`
}

// ListScriptsResponse is the body for GET /api/v1/scripts.
type ListScriptsResponse struct {
	Scripts []ScriptSummary `json:"scripts"`
}

// ListRecordsResponse is the body for GET /api/v1/records.
type ListRecordsResponse struct {
	Total    int              `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"page_size"`
	Records  []*record.Record `json:"records"`
}

// ── Handler ───────────────────────────────────────────────────────────────────

// Handler holds dependencies for HTTP request handlers.
type Handler struct {
	registry script.Registry
	runner   runner.Runner
	store    record.Store
}

// NewHandler creates a new Handler with the given dependencies.
func NewHandler(registry script.Registry, r runner.Runner, s record.Store) *Handler {
	return &Handler{
		registry: registry,
		runner:   r,
		store:    s,
	}
}

// writeJSON writes v as JSON to w with the given status code.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// ListScripts handles GET /api/v1/scripts.
func (h *Handler) ListScripts(w http.ResponseWriter, r *http.Request) {
	scripts := make([]ScriptSummary, 0, len(h.registry))
	for _, spec := range h.registry {
		scripts = append(scripts, ScriptSummary{
			Name:           spec.Name,
			Description:    spec.Description,
			TimeoutSeconds: spec.TimeoutSeconds,
			Params:         spec.Params,
		})
	}
	writeJSON(w, http.StatusOK, ListScriptsResponse{Scripts: scripts})
}

// Execute handles POST /api/v1/execute.
func (h *Handler) Execute(w http.ResponseWriter, r *http.Request) {
	var req ExecuteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Details: "failed to decode request body",
		})
		return
	}

	slog.Info("execute request", "script", req.Script, "remote_addr", r.RemoteAddr)

	// Look up script
	spec, ok := h.registry[req.Script]
	if !ok {
		slog.Warn("script not found", "script", req.Script)
		writeJSON(w, http.StatusNotFound, ErrorResponse{
			Error:   "script_not_found",
			Details: "script '" + req.Script + "' is not registered",
		})
		return
	}

	// Validate parameters
	var validationErrors []ValidationDetail
	for _, paramSpec := range spec.Params {
		value := req.Params[paramSpec.Name]
		if err := script.ValidateParam(value, paramSpec.Rules); err != nil {
			// Also enforce required check here in case rules.Required is set but ParamRule.Required is not
			validationErrors = append(validationErrors, ValidationDetail{
				Param:  paramSpec.Name,
				Reason: err.Error(),
			})
		}
		if paramSpec.Required && value == "" {
			// Might already be caught by ValidateParam, but ensure it is recorded
			alreadyRecorded := false
			for _, ve := range validationErrors {
				if ve.Param == paramSpec.Name {
					alreadyRecorded = true
					break
				}
			}
			if !alreadyRecorded {
				validationErrors = append(validationErrors, ValidationDetail{
					Param:  paramSpec.Name,
					Reason: "parameter is required but empty",
				})
			}
		}
	}
	if len(validationErrors) > 0 {
		slog.Warn("validation failed", "script", req.Script, "error_count", len(validationErrors))
		writeJSON(w, http.StatusUnprocessableEntity, ErrorResponse{
			Error:   "validation_failed",
			Details: validationErrors,
		})
		return
	}

	// Build env-var params (PARAM_{UPPER_NAME} = value)
	envParams := make(map[string]string, len(req.Params))
	for k, v := range req.Params {
		envParams["PARAM_"+toUpperSnake(k)] = v
	}

	executedAt := time.Now().UTC()

	// Run the script
	result, err := h.runner.Run(r.Context(), runner.RunRequest{
		ScriptPath: spec.ScriptPath,
		Params:     envParams,
		TimeoutSec: spec.TimeoutSeconds,
	})
	if err != nil {
		if errors.Is(err, runner.ErrTimeout) {
			slog.Warn("script timed out", "script", req.Script)
			writeJSON(w, http.StatusRequestTimeout, ErrorResponse{
				Error:   "execution_timeout",
				Details: "script exceeded timeout",
			})
			return
		}
		slog.Error("internal error running script", "script", req.Script, "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Details: err.Error(),
		})
		return
	}

	status := "success"
	if result.ExitCode != 0 {
		status = "failed"
	}

	rec := record.Record{
		Script:     req.Script,
		Params:     req.Params,
		Status:     status,
		ExitCode:   result.ExitCode,
		Stdout:     result.Stdout,
		Stderr:     result.Stderr,
		DurationMs: result.DurationMs,
		ExecutedAt: executedAt,
	}

	recordID, err := h.store.Save(rec)
	if err != nil {
		// Record storage failure is non-fatal; still return result
		recordID = ""
	}

	slog.Info("execute success", "script", req.Script, "status", status, "exit_code", result.ExitCode, "duration_ms", result.DurationMs)
	writeJSON(w, http.StatusOK, ExecuteResponse{
		RecordID:   recordID,
		Script:     req.Script,
		Status:     status,
		ExitCode:   result.ExitCode,
		Stdout:     result.Stdout,
		Stderr:     result.Stderr,
		DurationMs: result.DurationMs,
		ExecutedAt: executedAt,
	})
}

// ListRecords handles GET /api/v1/records.
func (h *Handler) ListRecords(w http.ResponseWriter, r *http.Request) {
	page := queryInt(r, "page", 1)
	pageSize := queryInt(r, "page_size", 20)
	if pageSize > 100 {
		pageSize = 100
	}

	result, err := h.store.List(page, pageSize)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Details: err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, ListRecordsResponse{
		Total:    result.Total,
		Page:     result.Page,
		PageSize: result.PageSize,
		Records:  result.Records,
	})
}

// GetRecord handles GET /api/v1/records/{record_id}.
func (h *Handler) GetRecord(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "record_id")
	if id == "" {
		// Fallback: check context value set by tests
		if v := r.Context().Value(recordIDKey{}); v != nil {
			id = v.(string)
		}
	}

	rec, err := h.store.Get(id)
	if err != nil || rec == nil {
		writeJSON(w, http.StatusNotFound, ErrorResponse{
			Error:   "record_not_found",
			Details: "record '" + id + "' not found",
		})
		return
	}

	writeJSON(w, http.StatusOK, rec)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func queryInt(r *http.Request, key string, defaultVal int) int {
	s := r.URL.Query().Get(key)
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil || v < 1 {
		return defaultVal
	}
	return v
}

// toUpperSnake converts "message_id" → "MESSAGE_ID".
func toUpperSnake(s string) string {
	result := make([]byte, len(s))
	for i := range s {
		c := s[i]
		if c >= 'a' && c <= 'z' {
			result[i] = c - 32
		} else {
			result[i] = c
		}
	}
	return string(result)
}

// recordIDKey is used to pass the record ID via context in tests.
type recordIDKey struct{}

// withRecordID injects a record ID into the context (used in tests without chi).
func withRecordID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, recordIDKey{}, id)
}
