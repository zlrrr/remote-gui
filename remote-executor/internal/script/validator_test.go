package script

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateParam_String_PatternOK(t *testing.T) {
	rule := ParamRule{Type: "string", Pattern: `^[a-zA-Z0-9_\-]{1,64}$`}
	err := ValidateParam("test-topic", rule)
	assert.NoError(t, err)
}

func TestValidateParam_String_PatternFail(t *testing.T) {
	rule := ParamRule{Type: "string", Pattern: `^[a-zA-Z0-9_\-]{1,64}$`}
	err := ValidateParam("test; rm -rf /", rule)
	assert.Error(t, err)
}

func TestValidateParam_Required_Empty(t *testing.T) {
	rule := ParamRule{Required: true}
	err := ValidateParam("", rule)
	assert.Error(t, err)
}

func TestValidateParam_Required_NotEmpty(t *testing.T) {
	rule := ParamRule{Required: true}
	err := ValidateParam("value", rule)
	assert.NoError(t, err)
}

func TestValidateParam_MaxLength(t *testing.T) {
	rule := ParamRule{Type: "string", MaxLength: 5}
	err := ValidateParam("toolong", rule)
	assert.Error(t, err)
}

func TestValidateParam_MaxLength_OK(t *testing.T) {
	rule := ParamRule{Type: "string", MaxLength: 10}
	err := ValidateParam("short", rule)
	assert.NoError(t, err)
}

func TestValidateParam_MinLength_Fail(t *testing.T) {
	rule := ParamRule{Type: "string", MinLength: 5}
	err := ValidateParam("ab", rule)
	assert.Error(t, err)
}

func TestValidateParam_MinLength_OK(t *testing.T) {
	rule := ParamRule{Type: "string", MinLength: 2}
	err := ValidateParam("abc", rule)
	assert.NoError(t, err)
}

func TestValidateParam_MessageID_PatternOK(t *testing.T) {
	rule := ParamRule{Type: "string", Pattern: `^[A-F0-9]{32,40}$`, MinLength: 32, MaxLength: 40}
	err := ValidateParam("A1B2C3D4E5F6A1B2C3D4E5F6A1B2C3D4", rule)
	assert.NoError(t, err)
}

func TestValidateParam_MessageID_PatternFail(t *testing.T) {
	rule := ParamRule{Type: "string", Pattern: `^[A-F0-9]{32,40}$`}
	err := ValidateParam("invalid!!", rule)
	assert.Error(t, err)
}

func TestValidateParam_MessageID_LowercaseFail(t *testing.T) {
	// message_id must be uppercase hex
	rule := ParamRule{Type: "string", Pattern: `^[A-F0-9]{32,40}$`}
	err := ValidateParam("a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4", rule)
	assert.Error(t, err)
}

func TestValidateParam_InvalidPattern(t *testing.T) {
	// Invalid regex should return an error
	rule := ParamRule{Type: "string", Pattern: `[invalid`}
	err := ValidateParam("value", rule)
	assert.Error(t, err)
}

func TestValidateParam_OptionalEmpty_NoRules(t *testing.T) {
	// Not required, no rules — empty value should be fine
	rule := ParamRule{Required: false}
	err := ValidateParam("", rule)
	assert.NoError(t, err)
}
