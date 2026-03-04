package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config holds the executor server configuration.
type Config struct {
	Server  ServerConfig  `yaml:"server"`
	Scripts ScriptsConfig `yaml:"scripts"`
	Records RecordsConfig `yaml:"records"`
	TLS     TLSConfig     `yaml:"tls"`
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// ScriptsConfig holds scripts directory settings.
type ScriptsConfig struct {
	Dir string `yaml:"dir"`
}

// RecordsConfig holds records storage settings.
type RecordsConfig struct {
	Dir string `yaml:"dir"`
}

// TLSConfig holds TLS certificate paths.
type TLSConfig struct {
	CACert     string `yaml:"ca_cert"`
	ServerCert string `yaml:"server_cert"`
	ServerKey  string `yaml:"server_key"`
}

// Load reads and parses the configuration file at the given path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read config file %q: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid YAML in config file %q: %w", path, err)
	}

	return &cfg, nil
}
