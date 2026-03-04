package client

// ExecutorClientConfig holds connection settings.
type ExecutorClientConfig struct {
	Endpoint   string
	CACert     string
	ClientCert string
	ClientKey  string
}

// ScriptInfo describes a registered script on the executor.
type ScriptInfo struct {
	Name        string
	Description string
}

// ExecuteResult holds the result of a script execution.
type ExecuteResult struct {
	RecordID   string
	Script     string
	Status     string
	ExitCode   int
	Stdout     string
	Stderr     string
	DurationMs int64
}

// ValidationError is returned when the executor responds with 422.
type ValidationError struct {
	Details []ValidationDetail
}

func (e *ValidationError) Error() string {
	return "validation failed"
}

// ValidationDetail describes a single validation failure.
type ValidationDetail struct {
	Param  string
	Reason string
}

// ExecutorClient communicates with the remote-executor service.
type ExecutorClient interface {
	ListScripts() ([]ScriptInfo, error)
	Execute(script string, params map[string]string) (*ExecuteResult, error)
}

// NewExecutorClient creates a new ExecutorClient with mTLS.
func NewExecutorClient(cfg ExecutorClientConfig) ExecutorClient {
	// TODO: implement in Phase 4.1
	return &httpExecutorClient{cfg: cfg}
}

type httpExecutorClient struct {
	cfg ExecutorClientConfig
}

func (c *httpExecutorClient) ListScripts() ([]ScriptInfo, error) {
	// TODO: implement in Phase 4.1
	return nil, nil
}

func (c *httpExecutorClient) Execute(script string, params map[string]string) (*ExecuteResult, error) {
	// TODO: implement in Phase 4.1
	return nil, nil
}
