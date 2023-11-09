package eiffel

import (
	"github.com/org-harmony/harmony/src/core/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestBasicParser_Validate(t *testing.T) {
	v := validation.New()
	parser := basicParser()
	errs := parser.Validate(v)
	require.Len(t, errs, 0)

	parser.Template.Rules = map[string]BasicRule{}
	errs = parser.Validate(v)
	require.Len(t, errs, 2)
	assert.ErrorAs(t, errs[0], &RuleMissingError{})
	assert.ErrorIs(t, errs[1], ErrInvalidTemplate)

	parser.Preprocessors = map[string]BasicPreprocessor{}
	errs = parser.Validate(v)
	require.Len(t, errs, 2) // will return after invalid rules
	assert.ErrorAs(t, errs[0], &PreprocessorMissingError{})
	assert.ErrorIs(t, errs[1], ErrInvalidTemplate)
}

func basicParser() *BasicParser {
	return &BasicParser{
		Template: basicTemplate(),
		Preprocessors: map[string]BasicPreprocessor{
			"lowercase":      lowercasePreprocessor,
			"trimWhitespace": trimWhitespacePreprocessor,
		},
	}
}

func basicTemplate() *BasicTemplate {
	return &BasicTemplate{
		Name:    "Test Template",
		Version: "1.0.0",
		Authors: []string{
			"John Doe",
			"Max Mustermann",
		},
		License: "MIT",
		Preprocessors: []string{
			"lowercase",
			"trimWhitespace",
		},
		Rules: map[string]BasicRule{
			"fooRule": {
				Name:        "Foo Rule",
				Explanation: "\"foo\" must be matched in the input string",
				Value:       "foo",
				Type:        "contains",
			},
		},
		Variants: map[string]BasicVariant{
			"basicVariant": {
				Name:        "Basic Variant (matching \"foo\")",
				Description: "This variant matches \"foo\" in the input string. It is the Basic Variant.",
				Format:      "[some string before] foo [some string after]",
				Example:     "This is a foo example.",
				Rules: []string{
					"fooRule",
				},
			},
		},
	}
}

func lowercasePreprocessor(input string) (string, error) {
	return strings.ToLower(input), nil
}

func trimWhitespacePreprocessor(input string) (string, error) {
	return strings.TrimSpace(input), nil
}
