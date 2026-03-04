package runner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// maxOutputBytes is the maximum number of bytes captured from stdout/stderr.
const maxOutputBytes = 4 * 1024 // 4 KB

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
	ExitCode   int
	Stdout     string
	Stderr     string
	DurationMs int64
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
	// Verify script exists before starting
	if _, err := os.Stat(req.ScriptPath); err != nil {
		slog.Error("script not found", "path", req.ScriptPath, "error", err)
		return nil, fmt.Errorf("script not found %q: %w", req.ScriptPath, err)
	}

	slog.Debug("running script", "path", req.ScriptPath, "timeout_sec", req.TimeoutSec, "env_keys", envKeyNames(req.Params))

	// Apply timeout if requested
	runCtx := ctx
	var cancel context.CancelFunc
	if req.TimeoutSec > 0 {
		runCtx, cancel = context.WithTimeout(ctx, time.Duration(req.TimeoutSec)*time.Second)
		defer cancel()
	}

	cmd := exec.Command("bash", req.ScriptPath)

	// Inject parameters as environment variables only — no command-line string concatenation.
	env := make([]string, 0, len(req.Params)+2)
	for k, v := range req.Params {
		env = append(env, k+"="+v)
	}
	// Preserve PATH and HOME so bash scripts work
	if path := os.Getenv("PATH"); path != "" {
		env = append(env, "PATH="+path)
	}
	if home := os.Getenv("HOME"); home != "" {
		env = append(env, "HOME="+home)
	}
	cmd.Env = env

	// Place child in its own process group so we can kill the whole group on timeout.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &limitWriter{w: &stdoutBuf, remaining: maxOutputBytes}
	cmd.Stderr = &limitWriter{w: &stderrBuf, remaining: maxOutputBytes}

	start := time.Now()
	if err := cmd.Start(); err != nil {
		slog.Error("failed to start script", "path", req.ScriptPath, "error", err)
		return nil, fmt.Errorf("failed to start script: %w", err)
	}
	slog.Debug("script started", "pid", cmd.Process.Pid, "path", req.ScriptPath)

	// Watch for context cancellation/timeout and kill the process group.
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-runCtx.Done():
			// Kill the entire process group to clean up child processes.
			if cmd.Process != nil {
				_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
			}
		case <-done:
		}
	}()

	waitErr := cmd.Wait()
	durationMs := time.Since(start).Milliseconds()

	// Check if the timeout fired
	if runCtx.Err() == context.DeadlineExceeded {
		slog.Warn("script timed out", "path", req.ScriptPath, "timeout_sec", req.TimeoutSec)
		return nil, ErrTimeout
	}

	// Check if the parent context was cancelled
	if ctx.Err() != nil {
		slog.Warn("script context cancelled", "path", req.ScriptPath, "error", ctx.Err())
		return nil, ctx.Err()
	}

	// A non-zero exit code is not a Go error — captured in result.ExitCode.
	exitCode := 0
	if waitErr != nil {
		var exitErr *exec.ExitError
		if errors.As(waitErr, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			slog.Error("script run error", "path", req.ScriptPath, "error", waitErr)
			return nil, fmt.Errorf("failed to run script: %w", waitErr)
		}
	}

	slog.Debug("script finished", "path", req.ScriptPath, "exit_code", exitCode, "duration_ms", durationMs,
		"stdout_len", len(stdoutBuf.String()), "stderr_len", len(stderrBuf.String()))

	return &RunResult{
		ExitCode:   exitCode,
		Stdout:     stdoutBuf.String(),
		Stderr:     stderrBuf.String(),
		DurationMs: durationMs,
	}, nil
}

// envKeyNames returns only the key names from an env params map (safe to log).
func envKeyNames(params map[string]string) []string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	return keys
}

// limitWriter truncates writes after remaining bytes are exhausted.
type limitWriter struct {
	w         io.Writer
	remaining int
}

func (lw *limitWriter) Write(p []byte) (int, error) {
	if lw.remaining <= 0 {
		return len(p), nil // discard silently
	}
	if len(p) > lw.remaining {
		p = p[:lw.remaining]
	}
	n, err := lw.w.Write(p)
	lw.remaining -= n
	return n, err
}
