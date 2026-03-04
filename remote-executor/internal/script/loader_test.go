package script

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadScripts_Success(t *testing.T) {
	registry, err := LoadScripts("testdata/scripts")
	require.NoError(t, err)
	assert.Contains(t, registry, "query-rocketmq-msg")

	spec := registry["query-rocketmq-msg"]
	assert.Equal(t, "query-rocketmq-msg", spec.Name)
	assert.Equal(t, 60, spec.TimeoutSeconds)
	assert.Len(t, spec.Params, 2)
	assert.NotEmpty(t, spec.ScriptPath)

	// Verify param names
	assert.Equal(t, "topic", spec.Params[0].Name)
	assert.Equal(t, "message_id", spec.Params[1].Name)
}

func TestLoadScripts_MissingSpec(t *testing.T) {
	// spec.yaml is missing — should return error
	_, err := LoadScripts("testdata/missing-spec")
	assert.Error(t, err)
}

func TestLoadScripts_InvalidSpec(t *testing.T) {
	// spec.yaml has invalid YAML — should return error
	_, err := LoadScripts("testdata/invalid-spec")
	assert.Error(t, err)
}

func TestLoadScripts_MissingRunSh(t *testing.T) {
	// run.sh is missing — should return error
	_, err := LoadScripts("testdata/no-run-sh")
	assert.Error(t, err)
}

func TestLoadScripts_NonexistentDir(t *testing.T) {
	// directory does not exist
	_, err := LoadScripts("testdata/does-not-exist")
	assert.Error(t, err)
}

func TestLoadScripts_ParamRulesLoaded(t *testing.T) {
	registry, err := LoadScripts("testdata/scripts")
	require.NoError(t, err)

	spec := registry["query-rocketmq-msg"]
	topicParam := spec.Params[0]
	assert.True(t, topicParam.Required)
	assert.Equal(t, `^[a-zA-Z0-9_\-]{1,64}$`, topicParam.Rules.Pattern)
	assert.Equal(t, 1, topicParam.Rules.MinLength)
	assert.Equal(t, 64, topicParam.Rules.MaxLength)

	msgParam := spec.Params[1]
	assert.True(t, msgParam.Required)
	assert.Equal(t, `^[A-F0-9]{32,40}$`, msgParam.Rules.Pattern)
}
