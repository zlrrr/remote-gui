package script

// ParamRule defines validation rules for a single parameter.
type ParamRule struct {
	Type      string `yaml:"type"`
	Required  bool   `yaml:"required"`
	Pattern   string `yaml:"pattern"`
	MinLength int    `yaml:"min_length"`
	MaxLength int    `yaml:"max_length"`
}

// ValidateParam validates a parameter value against the given rule.
func ValidateParam(value string, rule ParamRule) error {
	// TODO: implement in Phase 1.1
	return nil
}
