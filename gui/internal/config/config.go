package config

// Config holds the GUI configuration.
type Config struct {
	Executor   ExecutorConfig  `yaml:"executor"`
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

// LoadConfig reads and parses the GUI config file at the given path.
func LoadConfig(path string) (*Config, error) {
	// TODO: implement in Phase 4.2
	return &Config{}, nil
}
