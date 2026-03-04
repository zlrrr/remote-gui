package runner

import (
	"context"
	"errors"
)

// ErrTimeout is returned when a script execution exceeds its timeout.
var ErrTimeout = errors.New("script execution timed out")

// RunRequest describes a script execution request.
type RunRequest struct {
	ScriptPath string
	Params     map[string]string
	TimeoutSec int
}

// RunResult contains the outcome of a script execution.
type RunResult struct {
	ExitCode    int
	Stdout      string
	Stderr      string
	DurationMs  int64
}

// Runner executes scripts safely.
type Runner interface {
	Run(ctx context.Context, req RunRequest) (*RunResult, error)
}

// NewRunner creates a new default Runner.
func NewRunner() Runner {
	return &defaultRunner{}
}

type defaultRunner struct{}

func (r *defaultRunner) Run(ctx context.Context, req RunRequest) (*RunResult, error) {
	// TODO: implement in Phase 2.1
	return nil, nil
}
