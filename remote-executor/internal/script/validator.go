package script

import (
	"fmt"
	"regexp"
)

// ParamRule defines validation rules for a single parameter.
type ParamRule struct {
	Type      string `yaml:"type"`
	Required  bool   `yaml:"required"`
	Pattern   string `yaml:"pattern"`
	MinLength int    `yaml:"min_length"`
	MaxLength int    `yaml:"max_length"`
}

// ValidateParam validates a parameter value against the given rule.
// Returns a non-nil error describing the first violation found.
func ValidateParam(value string, rule ParamRule) error {
	if rule.Required && value == "" {
		return fmt.Errorf("parameter is required but empty")
	}

	if value == "" {
		// Not required and empty — skip remaining checks
		return nil
	}

	if rule.MinLength > 0 && len(value) < rule.MinLength {
		return fmt.Errorf("value length %d is less than minimum %d", len(value), rule.MinLength)
	}

	if rule.MaxLength > 0 && len(value) > rule.MaxLength {
		return fmt.Errorf("value length %d exceeds maximum %d", len(value), rule.MaxLength)
	}

	if rule.Pattern != "" {
		re, err := regexp.Compile(rule.Pattern)
		if err != nil {
			return fmt.Errorf("invalid pattern %q: %w", rule.Pattern, err)
		}
		if !re.MatchString(value) {
			return fmt.Errorf("value %q does not match pattern %q", value, rule.Pattern)
		}
	}

	return nil
}
