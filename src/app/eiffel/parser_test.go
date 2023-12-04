package eiffel

import (
	"context"
	"github.com/org-harmony/harmony/src/app/template"
	"github.com/org-harmony/harmony/src/app/template/parser"
	"github.com/org-harmony/harmony/src/core/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestBasicParser_Validate(t *testing.T) {
	v := validation.New()
	bt := basicTemplate()
	rp := ruleParsers()
	errs := bt.Validate(v, rp)
	require.Len(t, errs, 0)

	bt.Rules = map[string]BasicRule{}
	errs = bt.Validate(v, rp)
	require.Len(t, errs, 2)
	assert.ErrorAs(t, errs[0], &RuleMissingError{})
	assert.ErrorIs(t, errs[1], template.ErrInvalidTemplate)
}

func TestBasicParser_Parse(t *testing.T) {
	bt := basicTemplate()
	rp := ruleParsers()

	errs := bt.Validate(validation.New(), rp)
	require.Len(t, errs, 0)

	parsingResult, err := bt.Parse(
		context.Background(),
		rp,
		"basicVariant",
		parser.ParsingSegment{Name: "pre", Value: "This "},
		parser.ParsingSegment{Name: "stateVerbRule", Value: "is"},
		parser.ParsingSegment{Name: "mid", Value: "a"},
		parser.ParsingSegment{Name: "fooRule", Value: " foo "},
		parser.ParsingSegment{Name: "fooPostfixRule", Value: "example"},
		parser.ParsingSegment{Name: "end", Value: "."},
		parser.ParsingSegment{Name: "optionalErrorTestRule", Value: "bar"},
	)

	require.NoError(t, err)
	require.Len(t, parsingResult.Notices, 1)
	assert.True(t, parsingResult.Notices[0].Downgrade)
	assert.Equal(t, "test-template", parsingResult.TemplateID)
	assert.True(t, parsingResult.Ok(), "parsing result should be ok but parsing errors occurred")
	assert.True(t, parsingResult.Flawless(), "parsing result should be flawless")
	assert.Equal(t, parsingResult.Notices[0].Segment.Name, "optionalErrorTestRule")
	assert.True(t, parsingResult.Notices[0].Downgrade, "notice should be downgraded for optional rule")
}

func basicTemplate() *BasicTemplate {
	return &BasicTemplate{
		ID:      "test-template",
		Name:    "Test Template",
		Version: "1.0.0",
		Authors: []string{
			"John Doe",
			"Max Mustermann",
		},
		License: "MIT",
		Rules: map[string]BasicRule{
			"stateVerbRule": {
				Name:        "State Verb Rule",
				Type:        "equalsAny",
				Explanation: "One of the state verbs must be matched in the input string",
				Value: []any{ // this should not be []string{}, but []any{} because the parsed json is []interface{}
					"was",
					"will",
					"is",
				},
			},
			"fooRule": {
				Name:        "Foo Rule",
				Type:        "equals",
				Explanation: "\"foo\" must be matched in the input string",
				Value:       "foo",
			},
			"fooPostfixRule": {
				Name:     "Foo Postfix Rule",
				Type:     "placeholder",
				Optional: true,
			},
			"optionalMissingTestRule": {
				Name:     "Optional Missing Test Rule",
				Type:     "placeholder",
				Optional: true,
			},
			"optionalErrorTestRule": {
				Name:     "Optional Empty Error Test Rule",
				Type:     "equals",
				Value:    "foo",
				Optional: true,
			},
		},
		Variants: map[string]BasicVariant{
			"basicVariant": {
				Name:        "Basic VariantName (matching \"foo\")",
				Description: "This variant matches \"foo\" in the input string. It is the Basic VariantName.",
				Format:      "[some string before] foo [some string after]",
				Example:     "This is a foo example.",
				Rules: []string{
					"stateVerbRule",
					"fooRule",
					"fooPostfixRule",
					"optionalMissingTestRule",
					"optionalErrorTestRule",
				},
			},
		},
	}
}

func ruleParsers() *RuleParserProvider {
	return &RuleParserProvider{
		parsers: map[string]RuleParser{
			"equals":      EqualsRuleParser{},
			"equalsAny":   EqualsAnyRuleParser{},
			"placeholder": PlaceholderRuleParser{},
		},
	}
}
