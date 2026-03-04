package server

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/remote-gui/remote-executor/internal/api"
	"github.com/remote-gui/remote-executor/internal/record"
	"github.com/remote-gui/remote-executor/internal/runner"
	"github.com/remote-gui/remote-executor/internal/script"
)

const (
	testCACert     = "testdata/certs/ca.crt"
	testServerCert = "testdata/certs/executor.crt"
	testServerKey  = "testdata/certs/executor.key"
	testClientCert = "testdata/certs/client.crt"
	testClientKey  = "testdata/certs/client.key"
)

// ── Minimal mocks ──────────────────────────────────────────────────────────────

type noopRunner struct{}

func (r *noopRunner) Run(_ interface{ Done() <-chan struct{} }, _ runner.RunRequest) (*runner.RunResult, error) {
	return &runner.RunResult{ExitCode: 0, Stdout: "ok"}, nil
}

type noopStore struct{}

func (s *noopStore) Save(_ record.Record) (string, error)           { return "rec-001", nil }
func (s *noopStore) Get(_ string) (*record.Record, error)           { return nil, nil }
func (s *noopStore) List(_, _ int) (*record.ListResult, error) {
	return &record.ListResult{Records: []*record.Record{}}, nil
}

func buildTestServer() *Server {
	registry := script.Registry{
		"test-script": {
			Name:       "test-script",
			ScriptPath: "/fake/run.sh",
			Params:     []script.ParamSpec{},
		},
	}
	return New(Config{
		CACert:     testCACert,
		ServerCert: testServerCert,
		ServerKey:  testServerKey,
		Registry:   registry,
		Runner:     runner.NewRunner(),
		Store:      record.NewFileStore(os.TempDir()),
	})
}

// buildMTLSClient creates an http.Client configured with the given client cert and CA.
func buildMTLSClient(t *testing.T, clientCert, clientKey, caCert string) *http.Client {
	t.Helper()

	cert, err := tls.LoadX509KeyPair(clientCert, clientKey)
	require.NoError(t, err)

	caCertPEM, err := os.ReadFile(caCert)
	require.NoError(t, err)

	caPool := x509.NewCertPool()
	require.True(t, caPool.AppendCertsFromPEM(caCertPEM))

	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				Certificates: []tls.Certificate{cert},
				RootCAs:      caPool,
				MinVersion:   tls.VersionTLS13,
			},
		},
	}
}

// buildMTLSServer starts an httptest.Server with mTLS using the test certificates.
func buildMTLSServer(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()

	tlsCfg, err := buildTLSConfig(testCACert, testServerCert, testServerKey)
	require.NoError(t, err)

	srv := httptest.NewUnstartedServer(handler)
	srv.TLS = tlsCfg
	srv.StartTLS()
	t.Cleanup(srv.Close)

	return srv
}

// ── Router tests (no real TLS) ─────────────────────────────────────────────────

func TestRouter_ListScripts(t *testing.T) {
	s := buildTestServer()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/scripts", nil)
	s.buildRouter().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp api.ListScriptsResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Len(t, resp.Scripts, 1)
}

func TestRouter_UnknownPath(t *testing.T) {
	s := buildTestServer()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/unknown", nil)
	s.buildRouter().ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestRouter_Execute_UnknownScript(t *testing.T) {
	s := buildTestServer()
	w := httptest.NewRecorder()
	body := `{"script":"no-such","params":{}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/execute", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	s.buildRouter().ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ── mTLS integration tests ─────────────────────────────────────────────────────

func TestServer_mTLS_Accepted(t *testing.T) {
	s := buildTestServer()
	srv := buildMTLSServer(t, s.buildRouter())

	client := buildMTLSClient(t, testClientCert, testClientKey, testCACert)

	resp, err := client.Get(srv.URL + "/api/v1/scripts")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestServer_mTLS_Rejected_NoCert(t *testing.T) {
	s := buildTestServer()
	srv := buildMTLSServer(t, s.buildRouter())

	// Client without certificate
	caCertPEM, err := os.ReadFile(testCACert)
	require.NoError(t, err)
	caPool := x509.NewCertPool()
	require.True(t, caPool.AppendCertsFromPEM(caCertPEM))

	noCertClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:    caPool,
				MinVersion: tls.VersionTLS13,
			},
		},
	}

	_, err = noCertClient.Get(srv.URL + "/api/v1/scripts")
	// TLS handshake should fail because client provides no certificate
	assert.Error(t, err)
	// The error should be a network/TLS error
	var netErr net.Error
	assert.ErrorAs(t, err, &netErr)
}
