package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config holds the GUI configuration.
type Config struct {
	Executor   ExecutorConfig    `yaml:"executor"`
	Operations []OperationConfig `yaml:"operations"`
}

// ExecutorConfig holds connection settings for the remote-executor.
type ExecutorConfig struct {
	Endpoint string    `yaml:"endpoint"`
	TLS      TLSConfig `yaml:"tls"`
}

// TLSConfig holds TLS certificate paths.
type TLSConfig struct {
	CACert     string `yaml:"ca_cert"`
	ClientCert string `yaml:"client_cert"`
	ClientKey  string `yaml:"client_key"`
}

// OperationConfig defines a single GUI operation (alias for a script).
type OperationConfig struct {
	Alias  string        `yaml:"alias"`
	Script string        `yaml:"script"`
	Params []ParamConfig `yaml:"params"`
}

// ParamConfig defines a parameter input field in the GUI.
type ParamConfig struct {
	Label       string `yaml:"label"`
	Name        string `yaml:"name"`
	Placeholder string `yaml:"placeholder"`
}

const (
	maxOperations = 10
	maxParams     = 5
)

// LoadConfig reads and parses the GUI config file at the given path.
// Returns an error if the file cannot be read, the YAML is invalid,
// or the limits (10 operations, 5 params per operation) are exceeded.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read config file %q: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid YAML in config file %q: %w", path, err)
	}

	if len(cfg.Operations) > maxOperations {
		return nil, fmt.Errorf("too many operations: %d (max %d)", len(cfg.Operations), maxOperations)
	}

	for _, op := range cfg.Operations {
		if len(op.Params) > maxParams {
			return nil, fmt.Errorf("operation %q has too many params: %d (max %d)", op.Alias, len(op.Params), maxParams)
		}
	}

	return &cfg, nil
}
