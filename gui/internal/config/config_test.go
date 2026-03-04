package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig_ValidConfig(t *testing.T) {
	cfg, err := LoadConfig("testdata/gui.yaml")
	require.NoError(t, err)
	assert.Len(t, cfg.Operations, 1)
	assert.Equal(t, "查询 RocketMQ 消息", cfg.Operations[0].Alias)
	assert.Equal(t, "query-rocketmq-msg", cfg.Operations[0].Script)
	assert.Len(t, cfg.Operations[0].Params, 2)
	assert.Equal(t, "https://192.168.0.10:8443", cfg.Executor.Endpoint)
}

func TestLoadConfig_OperationLimit(t *testing.T) {
	// More than 10 operations should return an error
	_, err := LoadConfig("testdata/too-many-ops.yaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "10")
}

func TestLoadConfig_ParamLimit(t *testing.T) {
	// More than 5 params per operation should return an error
	_, err := LoadConfig("testdata/too-many-params.yaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "5")
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := LoadConfig("testdata/nonexistent.yaml")
	assert.Error(t, err)
}

func TestLoadConfig_TLSFields(t *testing.T) {
	cfg, err := LoadConfig("testdata/gui.yaml")
	require.NoError(t, err)
	assert.Equal(t, "certs/ca.crt", cfg.Executor.TLS.CACert)
	assert.Equal(t, "certs/gui.crt", cfg.Executor.TLS.ClientCert)
	assert.Equal(t, "certs/gui.key", cfg.Executor.TLS.ClientKey)
}

func TestLoadConfig_ParamFields(t *testing.T) {
	cfg, err := LoadConfig("testdata/gui.yaml")
	require.NoError(t, err)
	p := cfg.Operations[0].Params[0]
	assert.Equal(t, "Topic 名称", p.Label)
	assert.Equal(t, "topic", p.Name)
	assert.Equal(t, "e.g. test-topic", p.Placeholder)
}
