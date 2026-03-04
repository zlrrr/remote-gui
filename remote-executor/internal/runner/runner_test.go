package runner

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun_Success(t *testing.T) {
	r := NewRunner()
	result, err := r.Run(context.Background(), RunRequest{
		ScriptPath: "testdata/echo-params.sh",
		Params:     map[string]string{"PARAM_TOPIC": "test-topic"},
		TimeoutSec: 10,
	})
	require.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
	assert.Contains(t, result.Stdout, "test-topic")
	assert.GreaterOrEqual(t, result.DurationMs, int64(0))
}

func TestRun_Timeout(t *testing.T) {
	r := NewRunner()
	_, err := r.Run(context.Background(), RunRequest{
		ScriptPath: "testdata/sleep.sh",
		TimeoutSec: 1,
	})
	assert.ErrorIs(t, err, ErrTimeout)
}

func TestRun_ScriptNotFound(t *testing.T) {
	r := NewRunner()
	_, err := r.Run(context.Background(), RunRequest{
		ScriptPath: "testdata/not-exist.sh",
		TimeoutSec: 10,
	})
	assert.Error(t, err)
	assert.NotErrorIs(t, err, ErrTimeout)
}

func TestRun_NonZeroExitCode(t *testing.T) {
	r := NewRunner()
	result, err := r.Run(context.Background(), RunRequest{
		ScriptPath: "testdata/exit-fail.sh",
		TimeoutSec: 10,
	})
	// Non-zero exit is not a Go error — it is reported in result
	require.NoError(t, err)
	assert.Equal(t, 1, result.ExitCode)
	assert.Contains(t, result.Stderr, "stderr output")
}

func TestRun_ContextCancel(t *testing.T) {
	r := NewRunner()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := r.Run(ctx, RunRequest{
		ScriptPath: "testdata/sleep.sh",
		TimeoutSec: 30,
	})
	assert.Error(t, err)
}

func TestRun_ParamsInjectedAsEnvVars(t *testing.T) {
	r := NewRunner()
	result, err := r.Run(context.Background(), RunRequest{
		ScriptPath: "testdata/echo-params.sh",
		Params: map[string]string{
			"PARAM_TOPIC":      "my-topic",
			"PARAM_MESSAGE_ID": "ABCDEF1234",
		},
		TimeoutSec: 10,
	})
	require.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
	assert.Contains(t, result.Stdout, "TOPIC=my-topic")
	assert.Contains(t, result.Stdout, "MSG_ID=ABCDEF1234")
}

func TestRun_ErrTimeout_IsError(t *testing.T) {
	// Ensure ErrTimeout satisfies errors.Is chain
	wrapped := errors.New("wrapped: " + ErrTimeout.Error())
	assert.NotErrorIs(t, wrapped, ErrTimeout) // wrapping without %w doesn't match
	assert.ErrorIs(t, ErrTimeout, ErrTimeout)
}
