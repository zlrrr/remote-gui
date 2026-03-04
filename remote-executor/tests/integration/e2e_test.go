package integration

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/remote-gui/remote-executor/internal/record"
	"github.com/remote-gui/remote-executor/internal/runner"
	"github.com/remote-gui/remote-executor/internal/script"
	"github.com/remote-gui/remote-executor/internal/server"
)

const (
	testCACert     = "../../internal/server/testdata/certs/ca.crt"
	testServerCert = "../../internal/server/testdata/certs/executor.crt"
	testServerKey  = "../../internal/server/testdata/certs/executor.key"
	testClientCert = "../../internal/server/testdata/certs/client.crt"
	testClientKey  = "../../internal/server/testdata/certs/client.key"
	testScriptsDir = "testdata/scripts"
)

// buildMTLSClient creates an http.Client with the given client cert and CA.
func buildMTLSClient(t *testing.T) *http.Client {
	t.Helper()

	cert, err := tls.LoadX509KeyPair(testClientCert, testClientKey)
	require.NoError(t, err)

	caCertPEM, err := os.ReadFile(testCACert)
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

// startTestExecutor launches an mTLS server on a random port and returns the URL + shutdown func.
func startTestExecutor(t *testing.T, registry script.Registry, recordsDir string) string {
	t.Helper()

	// Find a free port
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := ln.Addr().String()
	ln.Close()

	srv := server.New(server.Config{
		Addr:       addr,
		CACert:     testCACert,
		ServerCert: testServerCert,
		ServerKey:  testServerKey,
		Registry:   registry,
		Runner:     runner.NewRunner(),
		Store:      record.NewFileStore(recordsDir),
	})

	go func() {
		_ = srv.Start()
	}()

	// Wait for server to be ready
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := tls.Dial("tcp", addr, &tls.Config{InsecureSkipVerify: true})
		if err == nil {
			conn.Close()
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	return "https://" + addr
}

// TestE2E_ListScripts verifies the full chain: client → executor → script registry.
func TestE2E_ListScripts(t *testing.T) {
	registry, err := script.LoadScripts(testScriptsDir)
	require.NoError(t, err)

	recordsDir := t.TempDir()
	url := startTestExecutor(t, registry, recordsDir)
	client := buildMTLSClient(t)

	resp, err := client.Get(url + "/api/v1/scripts")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body struct {
		Scripts []struct {
			Name string `json:"name"`
		} `json:"scripts"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Len(t, body.Scripts, 1)
	assert.Equal(t, "echo-test", body.Scripts[0].Name)
}

// TestE2E_ExecuteScript verifies a full execute cycle and that a record is written.
func TestE2E_ExecuteScript(t *testing.T) {
	registry, err := script.LoadScripts(testScriptsDir)
	require.NoError(t, err)

	recordsDir := t.TempDir()
	url := startTestExecutor(t, registry, recordsDir)
	client := buildMTLSClient(t)

	// Execute the script
	reqBody, _ := json.Marshal(map[string]interface{}{
		"script": "echo-test",
		"params": map[string]string{
			"message": "hello-world",
		},
	})

	resp, err := client.Post(
		url+"/api/v1/execute",
		"application/json",
		bytes.NewReader(reqBody),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result struct {
		RecordID string `json:"record_id"`
		Status   string `json:"status"`
		ExitCode int    `json:"exit_code"`
		Stdout   string `json:"stdout"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result.Status)
	assert.Equal(t, 0, result.ExitCode)
	assert.Contains(t, result.Stdout, "hello-world")
	assert.NotEmpty(t, result.RecordID)

	// Verify a record was written to disk
	files, err := os.ReadDir(recordsDir)
	require.NoError(t, err)
	assert.NotEmpty(t, files, "expected at least one record file in %s", recordsDir)
}

// TestE2E_ExecuteScript_ValidationFail verifies 422 on invalid params.
func TestE2E_ExecuteScript_ValidationFail(t *testing.T) {
	registry, err := script.LoadScripts(testScriptsDir)
	require.NoError(t, err)

	recordsDir := t.TempDir()
	url := startTestExecutor(t, registry, recordsDir)
	client := buildMTLSClient(t)

	// message param fails pattern (contains special char)
	reqBody, _ := json.Marshal(map[string]interface{}{
		"script": "echo-test",
		"params": map[string]string{
			"message": "invalid; rm -rf /",
		},
	})

	resp, err := client.Post(
		url+"/api/v1/execute",
		"application/json",
		bytes.NewReader(reqBody),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)

	var errResp struct {
		Error string `json:"error"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
	assert.Equal(t, "validation_failed", errResp.Error)
}

// TestE2E_mTLS_Rejected verifies that requests without client cert are rejected.
func TestE2E_mTLS_Rejected(t *testing.T) {
	registry, err := script.LoadScripts(testScriptsDir)
	require.NoError(t, err)

	recordsDir := t.TempDir()
	url := startTestExecutor(t, registry, recordsDir)

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

	_, err = noCertClient.Get(url + "/api/v1/scripts")
	assert.Error(t, err, "expected TLS handshake failure without client cert")
}

// TestE2E_RecordRetrieval verifies GET /records/{id} returns the stored record.
func TestE2E_RecordRetrieval(t *testing.T) {
	registry, err := script.LoadScripts(testScriptsDir)
	require.NoError(t, err)

	recordsDir := t.TempDir()
	url := startTestExecutor(t, registry, recordsDir)
	client := buildMTLSClient(t)

	// Execute first
	reqBody, _ := json.Marshal(map[string]interface{}{
		"script": "echo-test",
		"params": map[string]string{"message": "retrieve-test"},
	})
	resp, err := client.Post(url+"/api/v1/execute", "application/json", bytes.NewReader(reqBody))
	require.NoError(t, err)
	var execResult struct {
		RecordID string `json:"record_id"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&execResult))
	resp.Body.Close()
	require.NotEmpty(t, execResult.RecordID)

	// Retrieve by ID
	getResp, err := client.Get(fmt.Sprintf("%s/api/v1/records/%s", url, execResult.RecordID))
	require.NoError(t, err)
	defer getResp.Body.Close()
	assert.Equal(t, http.StatusOK, getResp.StatusCode)

	var rec struct {
		Script string `json:"script"`
		Status string `json:"status"`
	}
	require.NoError(t, json.NewDecoder(getResp.Body).Decode(&rec))
	assert.Equal(t, "echo-test", rec.Script)
	assert.Equal(t, "success", rec.Status)
}

// helper: ensure testdata exists
func init() {
	_ = os.MkdirAll(filepath.Join("testdata", "scripts", "echo-test"), 0o755)
}
