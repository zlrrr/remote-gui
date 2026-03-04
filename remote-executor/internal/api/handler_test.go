package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/remote-gui/remote-executor/internal/record"
	"github.com/remote-gui/remote-executor/internal/runner"
	"github.com/remote-gui/remote-executor/internal/script"
)

// ── Mocks ────────────────────────────────────────────────────────────────────

type mockRunner struct {
	result *runner.RunResult
	err    error
}

func (m *mockRunner) Run(_ context.Context, _ runner.RunRequest) (*runner.RunResult, error) {
	return m.result, m.err
}

type mockStore struct {
	records map[string]*record.Record
	saveErr error
}

func newMockStore() *mockStore {
	return &mockStore{records: make(map[string]*record.Record)}
}

func (m *mockStore) Save(rec record.Record) (string, error) {
	if m.saveErr != nil {
		return "", m.saveErr
	}
	id := "rec-test-001"
	rec.ID = id
	m.records[id] = &rec
	return id, nil
}

func (m *mockStore) Get(id string) (*record.Record, error) {
	rec, ok := m.records[id]
	if !ok {
		return nil, nil
	}
	return rec, nil
}

func (m *mockStore) List(page, pageSize int) (*record.ListResult, error) {
	records := make([]*record.Record, 0, len(m.records))
	for _, r := range m.records {
		records = append(records, r)
	}
	return &record.ListResult{
		Total:    len(records),
		Page:     page,
		PageSize: pageSize,
		Records:  records,
	}, nil
}

// ── Test fixtures ─────────────────────────────────────────────────────────────

func buildRegistry() script.Registry {
	return script.Registry{
		"query-rocketmq-msg": {
			Name:           "query-rocketmq-msg",
			Description:    "test script",
			TimeoutSeconds: 60,
			ScriptPath:     "/fake/run.sh",
			Params: []script.ParamSpec{
				{
					Name:     "topic",
					Type:     "string",
					Required: true,
					Rules:    script.ParamRule{Pattern: `^[a-zA-Z0-9_\-]{1,64}$`, MinLength: 1, MaxLength: 64},
				},
				{
					Name:     "message_id",
					Type:     "string",
					Required: true,
					Rules:    script.ParamRule{Pattern: `^[A-F0-9]{32,40}$`, MinLength: 32, MaxLength: 40},
				},
			},
		},
	}
}

func buildHandler() (*Handler, *mockRunner, *mockStore) {
	mr := &mockRunner{result: &runner.RunResult{ExitCode: 0, Stdout: "ok", DurationMs: 100}}
	ms := newMockStore()
	h := NewHandler(buildRegistry(), mr, ms)
	return h, mr, ms
}

// ── Tests: ListScripts ───────────────────────────────────────────────────────

func TestListScriptsHandler(t *testing.T) {
	h, _, _ := buildHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/scripts", nil)
	w := httptest.NewRecorder()

	h.ListScripts(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp ListScriptsResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Len(t, resp.Scripts, 1)
	assert.Equal(t, "query-rocketmq-msg", resp.Scripts[0].Name)
	assert.Len(t, resp.Scripts[0].Params, 2)
}

// ── Tests: Execute ──────────────────────────────────────────────────────────

func TestExecuteHandler_Success(t *testing.T) {
	h, _, _ := buildHandler()

	body := `{"script":"query-rocketmq-msg","params":{"topic":"test-topic","message_id":"A1B2C3D4E5F6A1B2C3D4E5F6A1B2C3D4"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/execute", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Execute(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp ExecuteResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "success", resp.Status)
	assert.Equal(t, 0, resp.ExitCode)
	assert.Equal(t, "ok", resp.Stdout)
	assert.NotEmpty(t, resp.RecordID)
}

func TestExecuteHandler_ValidationFail(t *testing.T) {
	h, _, _ := buildHandler()

	// message_id format is wrong
	body := `{"script":"query-rocketmq-msg","params":{"topic":"test-topic","message_id":"invalid!!"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/execute", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Execute(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	var errResp ErrorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &errResp))
	assert.Equal(t, "validation_failed", errResp.Error)
}

func TestExecuteHandler_UnknownScript(t *testing.T) {
	h, _, _ := buildHandler()

	body := `{"script":"unknown-script","params":{}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/execute", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Execute(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	var errResp ErrorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &errResp))
	assert.Equal(t, "script_not_found", errResp.Error)
}

func TestExecuteHandler_MissingRequiredParam(t *testing.T) {
	h, _, _ := buildHandler()

	// topic is required but missing
	body := `{"script":"query-rocketmq-msg","params":{"message_id":"A1B2C3D4E5F6A1B2C3D4E5F6A1B2C3D4"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/execute", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Execute(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestExecuteHandler_BadRequestBody(t *testing.T) {
	h, _, _ := buildHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/execute", strings.NewReader("{bad json}"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Execute(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestExecuteHandler_ScriptFailed(t *testing.T) {
	h, mr, _ := buildHandler()
	mr.result = &runner.RunResult{ExitCode: 1, Stderr: "oops", DurationMs: 50}

	body := `{"script":"query-rocketmq-msg","params":{"topic":"test-topic","message_id":"A1B2C3D4E5F6A1B2C3D4E5F6A1B2C3D4"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/execute", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Execute(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp ExecuteResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "failed", resp.Status)
	assert.Equal(t, 1, resp.ExitCode)
}

func TestExecuteHandler_Timeout(t *testing.T) {
	h, mr, _ := buildHandler()
	mr.result = nil
	mr.err = runner.ErrTimeout

	body := `{"script":"query-rocketmq-msg","params":{"topic":"test-topic","message_id":"A1B2C3D4E5F6A1B2C3D4E5F6A1B2C3D4"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/execute", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Execute(w, req)

	assert.Equal(t, http.StatusRequestTimeout, w.Code)
	var errResp ErrorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &errResp))
	assert.Equal(t, "execution_timeout", errResp.Error)
}

// ── Tests: Records ───────────────────────────────────────────────────────────

func TestListRecordsHandler(t *testing.T) {
	h, _, ms := buildHandler()
	// Pre-populate a record
	ms.Save(record.Record{Script: "query-rocketmq-msg", Status: "success", ExecutedAt: time.Now()})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/records", nil)
	w := httptest.NewRecorder()

	h.ListRecords(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp ListRecordsResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 1, resp.Total)
}

func TestGetRecordHandler_NotFound(t *testing.T) {
	h, _, _ := buildHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/records/rec-nonexistent", nil)
	// Set chi URL param
	req = req.WithContext(withRecordID(req.Context(), "rec-nonexistent"))
	w := httptest.NewRecorder()

	h.GetRecord(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
