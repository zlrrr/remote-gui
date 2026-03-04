package client

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testCACert     = "testdata/certs/ca.crt"
	testClientCert = "testdata/certs/client.crt"
	testClientKey  = "testdata/certs/client.key"
)

// mockExecutorServer builds a simple httptest server (no TLS) with predefined responses.
func mockExecutorServer(handler http.Handler) *httptest.Server {
	return httptest.NewServer(handler)
}

// newTestClient creates a client pointed at the given URL (plain HTTP for tests).
func newTestClient(url string) ExecutorClient {
	return &httpExecutorClient{
		cfg: ExecutorClientConfig{Endpoint: url},
		httpClient: &http.Client{},
	}
}

// ── ListScripts ────────────────────────────────────────────────────────────────

func TestClient_ListScripts(t *testing.T) {
	srv := mockExecutorServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/scripts", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"scripts": []map[string]interface{}{
				{"name": "query-rocketmq-msg", "description": "test script"},
			},
		})
	}))
	defer srv.Close()

	client := newTestClient(srv.URL)
	scripts, err := client.ListScripts()
	require.NoError(t, err)
	require.Len(t, scripts, 1)
	assert.Equal(t, "query-rocketmq-msg", scripts[0].Name)
}

func TestClient_ListScripts_ServerError(t *testing.T) {
	srv := mockExecutorServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := newTestClient(srv.URL)
	_, err := client.ListScripts()
	assert.Error(t, err)
}

// ── Execute ────────────────────────────────────────────────────────────────────

func TestClient_Execute_Success(t *testing.T) {
	srv := mockExecutorServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/execute", r.URL.Path)

		// Verify request body contains the script name
		var body map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "query-rocketmq-msg", body["script"])

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"record_id":   "rec-001",
			"script":      "query-rocketmq-msg",
			"status":      "success",
			"exit_code":   0,
			"stdout":      "result output",
			"stderr":      "",
			"duration_ms": 500,
		})
	}))
	defer srv.Close()

	client := newTestClient(srv.URL)
	result, err := client.Execute("query-rocketmq-msg", map[string]string{"topic": "t1"})
	require.NoError(t, err)
	assert.Equal(t, "success", result.Status)
	assert.Equal(t, 0, result.ExitCode)
	assert.Equal(t, "result output", result.Stdout)
}

func TestClient_Execute_ValidationError(t *testing.T) {
	srv := mockExecutorServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "validation_failed",
			"details": []map[string]string{
				{"param": "message_id", "reason": "格式不匹配"},
			},
		})
	}))
	defer srv.Close()

	client := newTestClient(srv.URL)
	_, err := client.Execute("query-rocketmq-msg", map[string]string{"message_id": "invalid"})
	require.Error(t, err)

	var valErr *ValidationError
	assert.ErrorAs(t, err, &valErr)
	assert.Len(t, valErr.Details, 1)
	assert.Equal(t, "message_id", valErr.Details[0].Param)
}

func TestClient_Execute_ScriptNotFound(t *testing.T) {
	srv := mockExecutorServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "script_not_found",
			"details": "script 'no-such' is not registered",
		})
	}))
	defer srv.Close()

	client := newTestClient(srv.URL)
	_, err := client.Execute("no-such", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "script_not_found")
}

// ── mTLS wiring test (build only, requires real TLS) ──────────────────────────

func TestNewExecutorClient_MTLSConfig(t *testing.T) {
	// Verify that the client can be constructed with mTLS config
	// (doesn't make a real request here)
	cfg := ExecutorClientConfig{
		Endpoint:   "https://localhost:8443",
		CACert:     testCACert,
		ClientCert: testClientCert,
		ClientKey:  testClientKey,
	}
	c, err := NewExecutorClient(cfg)
	require.NoError(t, err)
	assert.NotNil(t, c)
}

func TestNewExecutorClient_InvalidCert(t *testing.T) {
	cfg := ExecutorClientConfig{
		Endpoint:   "https://localhost:8443",
		CACert:     "testdata/certs/nonexistent.crt",
		ClientCert: testClientCert,
		ClientKey:  testClientKey,
	}
	_, err := NewExecutorClient(cfg)
	assert.Error(t, err)
}

// ── Helper: builds mTLS httptest server for executor ──────────────────────────

func buildMTLSExecutorServer(t *testing.T, handler http.Handler) (*httptest.Server, *http.Client) {
	t.Helper()

	// Load server certs (same as executor)
	serverCert, err := tls.LoadX509KeyPair(
		"../../remote-executor/internal/server/testdata/certs/executor.crt",
		"../../remote-executor/internal/server/testdata/certs/executor.key",
	)
	if err != nil {
		t.Skip("executor test certs not available")
		return nil, nil
	}

	caCertPEM, err := os.ReadFile(testCACert)
	require.NoError(t, err)
	caPool := x509.NewCertPool()
	require.True(t, caPool.AppendCertsFromPEM(caCertPEM))

	srv := httptest.NewUnstartedServer(handler)
	srv.TLS = &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    caPool,
		MinVersion:   tls.VersionTLS13,
	}
	srv.StartTLS()
	t.Cleanup(srv.Close)

	clientCert, err := tls.LoadX509KeyPair(testClientCert, testClientKey)
	require.NoError(t, err)

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				Certificates: []tls.Certificate{clientCert},
				RootCAs:      caPool,
				MinVersion:   tls.VersionTLS13,
			},
		},
	}

	return srv, httpClient
}

func TestClient_Execute_OverMTLS(t *testing.T) {
	srv, httpClient := buildMTLSExecutorServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"record_id": "rec-mtls-001",
			"status":    "success",
			"exit_code": 0,
			"stdout":    "mTLS ok",
		})
	}))
	if srv == nil {
		return
	}

	c := &httpExecutorClient{
		cfg:        ExecutorClientConfig{Endpoint: srv.URL},
		httpClient: httpClient,
	}

	result, err := c.Execute("test-script", nil)
	require.NoError(t, err)
	assert.Equal(t, "success", result.Status)
}
