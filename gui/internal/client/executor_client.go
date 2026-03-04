package client

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

// ExecutorClientConfig holds connection settings.
type ExecutorClientConfig struct {
	Endpoint   string
	CACert     string
	ClientCert string
	ClientKey  string
}

// ScriptInfo describes a registered script on the executor.
type ScriptInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// ExecuteResult holds the result of a script execution.
type ExecuteResult struct {
	RecordID   string `json:"record_id"`
	Script     string `json:"script"`
	Status     string `json:"status"`
	ExitCode   int    `json:"exit_code"`
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	DurationMs int64  `json:"duration_ms"`
}

// ValidationError is returned when the executor responds with 422.
type ValidationError struct {
	Details []ValidationDetail `json:"details"`
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed (%d errors)", len(e.Details))
}

// ValidationDetail describes a single validation failure.
type ValidationDetail struct {
	Param  string `json:"param"`
	Reason string `json:"reason"`
}

// ExecutorClient communicates with the remote-executor service.
type ExecutorClient interface {
	ListScripts() ([]ScriptInfo, error)
	Execute(script string, params map[string]string) (*ExecuteResult, error)
}

// NewExecutorClient creates a new ExecutorClient with mTLS when cert paths are provided.
func NewExecutorClient(cfg ExecutorClientConfig) (ExecutorClient, error) {
	httpClient := &http.Client{}

	if cfg.CACert != "" || cfg.ClientCert != "" {
		tlsCfg, err := buildTLSConfig(cfg.CACert, cfg.ClientCert, cfg.ClientKey)
		if err != nil {
			return nil, fmt.Errorf("failed to build TLS config: %w", err)
		}
		httpClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: tlsCfg,
			},
		}
	}

	return &httpExecutorClient{cfg: cfg, httpClient: httpClient}, nil
}

func buildTLSConfig(caCertPath, clientCertPath, clientKeyPath string) (*tls.Config, error) {
	tlsCfg := &tls.Config{MinVersion: tls.VersionTLS13}

	if caCertPath != "" {
		caCert, err := os.ReadFile(caCertPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA cert %q: %w", caCertPath, err)
		}
		caPool := x509.NewCertPool()
		if !caPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA cert")
		}
		tlsCfg.RootCAs = caPool
	}

	if clientCertPath != "" && clientKeyPath != "" {
		cert, err := tls.LoadX509KeyPair(clientCertPath, clientKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load client cert/key: %w", err)
		}
		tlsCfg.Certificates = []tls.Certificate{cert}
	}

	return tlsCfg, nil
}

type httpExecutorClient struct {
	cfg        ExecutorClientConfig
	httpClient *http.Client
}

type listScriptsResponse struct {
	Scripts []ScriptInfo `json:"scripts"`
}

type executeRequest struct {
	Script string            `json:"script"`
	Params map[string]string `json:"params"`
}

type errorResponse struct {
	Error   string          `json:"error"`
	Details json.RawMessage `json:"details,omitempty"`
}

func (c *httpExecutorClient) ListScripts() ([]ScriptInfo, error) {
	resp, err := c.httpClient.Get(c.cfg.Endpoint + "/api/v1/scripts")
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned %d", resp.StatusCode)
	}

	var result listScriptsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Scripts, nil
}

func (c *httpExecutorClient) Execute(scriptName string, params map[string]string) (*ExecuteResult, error) {
	body, err := json.Marshal(executeRequest{Script: scriptName, Params: params})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.httpClient.Post(
		c.cfg.Endpoint+"/api/v1/execute",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var result ExecuteResult
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		return &result, nil

	case http.StatusUnprocessableEntity:
		var errResp struct {
			Error   string             `json:"error"`
			Details []ValidationDetail `json:"details"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			return nil, fmt.Errorf("validation error (failed to parse details): %w", err)
		}
		return nil, &ValidationError{Details: errResp.Details}

	default:
		var errResp errorResponse
		_ = json.NewDecoder(resp.Body).Decode(&errResp)
		errCode := errResp.Error
		if errCode == "" {
			errCode = fmt.Sprintf("http_%d", resp.StatusCode)
		}
		return nil, fmt.Errorf("%s", errCode)
	}
}
