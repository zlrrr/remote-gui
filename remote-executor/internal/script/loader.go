package script

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ParamSpec describes a single parameter specification from spec.yaml.
type ParamSpec struct {
	Name        string    `yaml:"name"`
	Description string    `yaml:"description"`
	Type        string    `yaml:"type"`
	Required    bool      `yaml:"required"`
	Rules       ParamRule `yaml:"rules"`
}

// ScriptSpec is the parsed content of a script's spec.yaml.
type ScriptSpec struct {
	Name           string      `yaml:"name"`
	Description    string      `yaml:"description"`
	TimeoutSeconds int         `yaml:"timeout_seconds"`
	Params         []ParamSpec `yaml:"params"`
	ScriptPath     string      // absolute path to run.sh, set at load time
}

// Registry maps script names to their specs.
type Registry map[string]*ScriptSpec

// LoadScripts scans the given directory for script subdirectories,
// parses each spec.yaml, and returns a Registry.
// Each subdirectory must contain both spec.yaml and run.sh.
func LoadScripts(dir string) (Registry, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("cannot read scripts directory %q: %w", dir, err)
	}

	registry := make(Registry)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		scriptDir := filepath.Join(dir, entry.Name())
		spec, err := loadScript(scriptDir)
		if err != nil {
			return nil, fmt.Errorf("failed to load script %q: %w", entry.Name(), err)
		}

		registry[spec.Name] = spec
	}

	if len(registry) == 0 {
		// No scripts found is acceptable — return empty registry
		return registry, nil
	}

	return registry, nil
}

func loadScript(scriptDir string) (*ScriptSpec, error) {
	specPath := filepath.Join(scriptDir, "spec.yaml")
	runPath := filepath.Join(scriptDir, "run.sh")

	// Check spec.yaml exists
	specData, err := os.ReadFile(specPath)
	if err != nil {
		return nil, fmt.Errorf("spec.yaml not found in %q: %w", scriptDir, err)
	}

	// Parse spec.yaml
	var spec ScriptSpec
	if err := yaml.Unmarshal(specData, &spec); err != nil {
		return nil, fmt.Errorf("invalid spec.yaml in %q: %w", scriptDir, err)
	}

	if spec.Name == "" {
		return nil, fmt.Errorf("spec.yaml in %q is missing required field 'name'", scriptDir)
	}

	// Check run.sh exists
	if _, err := os.Stat(runPath); err != nil {
		return nil, fmt.Errorf("run.sh not found in %q: %w", scriptDir, err)
	}

	absRunPath, err := filepath.Abs(runPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve run.sh path in %q: %w", scriptDir, err)
	}
	spec.ScriptPath = absRunPath

	return &spec, nil
}
