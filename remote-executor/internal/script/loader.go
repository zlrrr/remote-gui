package script

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
func LoadScripts(dir string) (Registry, error) {
	// TODO: implement in Phase 1.2
	return Registry{}, nil
}
