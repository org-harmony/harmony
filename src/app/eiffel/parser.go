package eiffel

import (
	"context"
	"errors"
	t "github.com/org-harmony/harmony/src/app/template"
	"github.com/org-harmony/harmony/src/app/template/parser"
	"github.com/org-harmony/harmony/src/core/trans"
	"github.com/org-harmony/harmony/src/core/validation"
	"strings"
	"sync"
)

// BasicTemplateType is the type name of the basic EIFFEL template used to identify the corresponding parser for a template.
const BasicTemplateType = "ebt"

var (
	// ErrInvalidVariant is an error that is returned when trying to parse a requirement for an invalid variant.
	ErrInvalidVariant = errors.New("eiffel.parser.error.invalid-variant")
)

// BasicTemplate is the basic EIFFEL template. It is parsable by implementing the template.ParsableTemplate interface.
//
// A basic template is a template that defines a set of rules and a set of variants.
// Each variant contains rules that are to be applied to a requirement in the parsing step to validate the requirement.
// If each rule is valid, the requirement is valid. Otherwise, the requirement is invalid.
//
// Side note: Preprocessors were removed to simplify the code and remove unnecessary complexity.
// E.g. They could be used to remove whitespaces from the input string, but this should be done on each segment by default.
// Also, they should convert the input string to lower-case, but as of now, this would be the only real use case
// and the goal of simplifying the code is more important than having this little artifact optimized for potential future flexibility.
type BasicTemplate struct {
	// ID is the technical specifier of the template. E.g. a technical name.
	ID string `json:"id" hvalidate:"required"`
	// Name is the display name of the template.
	Name string `json:"name" hvalidate:"required"`
	// Version is the version of the template. E.g. 1.0.0
	Version string `json:"version" hvalidate:"required,semVer"`
	// Authors are the authors of the template. It is an optional list of names.
	Authors []string `json:"authors"`
	// License is the license under which the template is published. Specifying a license is optional.
	License string `json:"license"`
	// Description is the description of the template. It is optional.
	Description string `json:"description"`
	// Format can be used to optionally describe the format of the requirement specified by the template.
	Format string `json:"format"`
	// Example can be used to optionally provide an example of a requirement specified by the template.
	Example string `json:"example"`
	// Rules are the rules that can be used in variants to validate requirements.
	Rules map[string]BasicRule `json:"rules"`
	// Variants are the variants that can be used to validate requirements.
	Variants map[string]BasicVariant `json:"variants" hvalidate:"required"`
}

// BasicRule is a rule to reference in a variant.
// Rules are used to validate different parts of a requirement.
// More precisely, they are used to validate the segments of a requirement.
type BasicRule struct {
	// Name is the display name of the rule.
	Name string `json:"name" hvalidate:"required"`
	// Type is the type of the rule. It is used to determine the rules value and how to validate it.
	// Check further documentation for all valid types that are supported by the EIFFEL basic template (EBT).
	Type string `json:"type" hvalidate:"required"`
	// Hint is an optional, short description to hint the user of the template into correct usage of the template according to the rule(s).
	Hint string `json:"hint"`
	// Explanation can optionally be defined to further explain the use/parsing of the rule to the user of the template.
	Explanation string `json:"explanation"`
	// Value is the value of the rule used during rule parsing. Each rule type may expect different values or even no value at all.
	// Value might be something like an exact string or a slice of strings.
	// Check further documentation for all valid types that are supported by the EIFFEL basic template (EBT).
	Value any `json:"value"`
	// Optional means a rule is not required to be parsed without template.ParsingLogLevelError (parsing error).
	// By that parsing an invalid requirement for an optional rule will not result in a parsing error.
	Optional bool `json:"optional"`
	// Size is the expected size of the rule's value. It is optional. Possible values are:
	//  - "small" (default): The value is expected to be a short string. 1/4 of the input field width.
	//  - "medium": The value is expected to be a medium-sized string. 2/4 of the input field width.
	//  - "large": The value is expected to be a large string. 3/4 of the input field width.
	//  - "full": The value is expected to be a very large string. 4/4 of the input field (this will be a textarea) width.
	// For now this should not be overcomplicated.
	// TODO add adaptive sizing => make more convenient for the user
	Size string `json:"size"`
	// Extra is an optional map of additional data that can be used by the rule parser.
	Extra map[string]any `json:"extra"`
}

// BasicVariant is a concrete variation of a template to parse requirements. Each variant contains a set of rules.
// The rules are applied in the order they are defined in the variant. Each template can define multiple variants.
type BasicVariant struct {
	// Name is the display name of the variant.
	Name string `json:"name" hvalidate:"required"`
	// Description is the description of the variant. It is optional.
	Description string `json:"description"`
	// Format can be used to optionally describe the format of the requirement specified by the variant.
	// E.g. "As a <role>, I want <feature> so that <benefit>."
	Format string `json:"format"`
	// Example can be used to optionally provide an example of a requirement following the format specified by the variant.
	Example string `json:"example"`
	// Rules contains rule names, rule objects should be contained in the template
	Rules []string `json:"rules"`
}

// RuleMissingError is an error that is returned when a rule is referenced in a variant but not defined in the template.
// It is returned by the RuleReferencesValidator.
type RuleMissingError struct {
	Rule     string
	Template string
	Variant  string
}

// RuleInvalidValueError is an error that is returned when a rule value is of the wrong type or otherwise invalid.
// It returns the ErrInvalidRuleValue error on Error(). It may occur during template validation.
type RuleInvalidValueError struct {
	Rule *BasicRule
}

// MissingRuleParserError is an error that is returned when a rule type is not registered in the RuleParserProvider.
// This usually means that the provided template is invalid because the rule type is not defined in the template.
type MissingRuleParserError struct {
	RuleType string
}

// RuleParserProvider provides rules for different rule types. It is independent of templates and variants.
// Its sole purpose is to manage RuleParser instances and allow access to them during validation
// (for validating each rule in a template) and parsing. RuleParserProvider is safe for concurrent use by multiple goroutines.
type RuleParserProvider struct {
	// parsers is a map of rule parsers per rule type.
	parsers map[string]RuleParser
	mu      sync.RWMutex
}

// RuleParser provides the capabilities to validate a rule (after defining it in a template), ideally during the template validation,
// and to parse a rule during the parsing step of a requirement using a template.
//
// Per rule type, a rule parser implementation is required. This is because each rule type implements its own parsing and validation logic
// as well as its own data type for the rule value.
//
// RuleParser is expected to be stateless and therefore safe for concurrent use by multiple goroutines.
type RuleParser interface {
	// Parse parses a rule using a segment of a requirement and a parsing result.
	Parse(ctx context.Context, rule BasicRule, segment parser.ParsingSegment) ([]parser.ParsingLog, error)
	// Validate validates that a rule, defined in a template, is valid. This could for example validate that the rule's value is of the correct data type.
	Validate(v validation.V, rule BasicRule) []error
	// DisplayType returns the display type of the rule. This is used to determine which input field or other UI element to render for the rule.
	DisplayType(rule BasicRule) TemplateDisplayType
}

// EqualsRuleParser is a rule parser for the rule type 'equals'.
// It is case-insensitive and will therefore convert the segment's value and the rule's value to lowercase before comparing them.
type EqualsRuleParser struct{}

// EqualsAnyRuleParser is a rule parser for the rule type 'equalsAny'.
// It is case-insensitive and will therefore convert the segment's value and the rule's values to lowercase before comparing them.
// It expects the rule's value to be a slice of strings. Any of the strings in the slice must match the segment's value.
type EqualsAnyRuleParser struct{}

// PlaceholderRuleParser is a rule parser for the rule type 'placeholder'. Placeholders can be used to parse segments that contain some arbitrary string content.
// Placeholders may be used to generate input fields for the user of the template without knowing the exact content of the segment.
// If it wasn't for the input field the placeholder is used for, it would be useless.
// Therefore, the value of a placeholder is optional.
type PlaceholderRuleParser struct{}

// RuleParsers constructs a new RuleParserProvider with the default rule parsers registered.
func RuleParsers() *RuleParserProvider {
	return &RuleParserProvider{
		parsers: map[string]RuleParser{
			"equals":      EqualsRuleParser{},
			"equalsAny":   EqualsAnyRuleParser{},
			"placeholder": PlaceholderRuleParser{},
		},
	}
}

// Parse implements the template.ParsableTemplate interface for the BasicTemplate. It is used to parse requirements in the form of segments.
// Each segment is a part of the requirement that is to be parsed. For the EIFFEL basic template (EBT), each segment is an input from auto-generated
// input field based on the rules defined in the template. Therefore, each segment will be validated by the corresponding rule.
//
// This function might panic if the template is not valid. Therefore, it is recommended to validate the template before parsing requirements.
//
// The parsing process is as follows:
//  1. Prepare segments by trimming whitespaces from the input string and indexing them.
//  2. Find the variant to parse the requirement for.
//  3. Validate each rule of the variant with the corresponding segment.
//     - superfluous segments are ignored
//     - missing segments are reported as parsing errors
//     - logs (errors, warning, notices) during rule parsing are reported
//  4. Return the parsing result.
//
// Consequences of **optional** rule parsing: If a rule is optional (BasicRule.Optional flag) and the segment is missing, the rule is ignored.
// If a rule is optional and the segment is present, Parse will parse the segment and report any warnings and notices.
// After parsing an optional rule the errors will be downgraded to notices. The template.ParsingLog Downgrade flag is set to true.
// Therefore, the parsing result will be ok and flawless. However, the parsing result will contain notices.
// Also, it is possible that a parsed rule contains warnings those are not downgraded and the parsing result will not be flawless.
func (bt *BasicTemplate) Parse(ctx context.Context, ruleParsers *RuleParserProvider, variantName string, segments ...parser.ParsingSegment) (parser.ParsingResult, error) {
	result := parser.ParsingResult{
		TemplateID:   bt.ID,
		TemplateType: BasicTemplateType,
		TemplateName: bt.Name,
		VariantName:  variantName,
	}

	indexedSegments := prepareSegments(segments)
	variant, ok := bt.Variants[variantName]
	if !ok {
		return result, ErrInvalidVariant
	}

	for _, ruleName := range variant.Rules {
		rule, ok := bt.Rules[ruleName]
		if !ok {
			return result, RuleMissingError{Rule: ruleName, Template: bt.Name, Variant: variant.Name}
		}

		segment, ok := indexedSegments[ruleName]
		if !ok && !rule.Optional {
			result.Errors = append(result.Errors, parser.ParsingLog{
				Segment: nil,
				Level:   parser.ParsingLogLevelError,
				Message: "eiffel.parser.error.missing-segment",
				TranslationArgs: []string{
					"name",
					rule.Name,
					"technicalName",
					ruleName,
				},
			})
			continue
		}

		if !ok {
			continue // rule is optional and segment is missing -> ignore
		}

		ruleParser, err := ruleParsers.Parser(rule.Type)
		if err != nil {
			return result, err // should also never happen because the template is expected to be valid
		}

		parsingLogs, err := ruleParser.Parse(ctx, rule, segment)
		if err != nil {
			return result, err
		}

		for _, log := range parsingLogs {
			switch log.Level {
			case parser.ParsingLogLevelError:
				if rule.Optional {
					log.Level = parser.ParsingLogLevelNotice
					log.Downgrade = true
					result.Notices = append(result.Notices, log)
					break
				}
				result.Errors = append(result.Errors, log)
			case parser.ParsingLogLevelWarning:
				result.Warnings = append(result.Warnings, log)
			case parser.ParsingLogLevelNotice:
				result.Notices = append(result.Notices, log)
			}
		}
	}

	return result, nil
}

func (bt *BasicTemplate) Validate(v validation.V, ruleParsers *RuleParserProvider) []error {
	errs := validation.Validate("", "", bt, RuleReferencesValidator)
	if len(errs) > 0 {
		return append(errs, t.ErrInvalidTemplate)
	}

	err, validationErrs := v.ValidateStruct(bt)
	if err != nil {
		return []error{err}
	}

	// Unfortunately, validation is currently not capable of validating slices/maps of structs.
	// Therefore, we have to do it manually. TODO support validation slices/maps of structs?
	for _, rule := range bt.Rules {
		err, ruleValidationErrs := v.ValidateStruct(rule) // this is validating the generic rule struct
		if err != nil {
			return []error{err}
		}

		validationErrs = append(validationErrs, ruleValidationErrs...)

		ruleParser, err := ruleParsers.Parser(rule.Type)
		if err != nil {
			return []error{err}
		}

		// This is requesting the rule parser to validate the rule value.
		// A parser might for example validate that the rule value is of the correct data type.
		ruleValidationErrs = ruleParser.Validate(v, rule)
		validationErrs = append(validationErrs, ruleValidationErrs...)
	}

	for _, variant := range bt.Variants {
		err, variantValidationErrs := v.ValidateStruct(variant)
		if err != nil {
			return []error{err}
		}

		validationErrs = append(validationErrs, variantValidationErrs...)
	}

	if len(validationErrs) > 0 {
		return append(validationErrs, t.ErrInvalidTemplate)
	}

	return nil
}

// RuleReferencesValidator validates that each rule referenced in a variant is defined in the template's 'rules' section.
func RuleReferencesValidator(basicTemplate any) error {
	bt, ok := basicTemplate.(*BasicTemplate)
	if !ok {
		return nil
	}

	for _, variant := range bt.Variants {
		for _, rule := range variant.Rules {
			if _, ok := bt.Rules[rule]; ok {
				continue
			}

			return RuleMissingError{
				Rule:     rule,
				Template: bt.Name,
				Variant:  variant.Name,
			}
		}
	}

	return nil
}

// Parser returns the rule parser for the given rule type. If no rule parser is registered for the given rule type, an error is returned.
func (p *RuleParserProvider) Parser(ruleType string) (RuleParser, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	ruleParser, ok := p.parsers[ruleType]
	if !ok {
		return nil, MissingRuleParserError{RuleType: ruleType}
	}

	return ruleParser, nil
}

// Lookup returns true if a rule parser is registered for the given rule type.
func (p *RuleParserProvider) Lookup(ruleType string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	_, ok := p.parsers[ruleType]
	return ok
}

// Register registers a rule parser for the given rule type.
func (p *RuleParserProvider) Register(ruleType string, parser RuleParser) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.parsers[ruleType] = parser
}

// Error on RuleMissingError returns the error code of the error.
func (e RuleMissingError) Error() string {
	return "eiffel.parser.error.missing-rule"
}

// UnwrapTransparent on RuleMissingError returns the error itself, implementing the validation.TransparentError interface.
// Being a transparent error means that the error is not wrapped by a validation error after being returned by a validator.
func (e RuleMissingError) UnwrapTransparent(err validation.Error) error {
	return e
}

// Translate on RuleMissingError translates the error using the given translator.
func (e RuleMissingError) Translate(t trans.Translator) string {
	return t.Tf(e.Error(), "template", e.Template, "rule", e.Rule, "variant", e.Variant)
}

// Error on RuleInvalidValueError returns the error code of the error.
func (e RuleInvalidValueError) Error() string {
	return "eiffel.parser.error.invalid-rule-value"
}

// Translate on RuleInvalidValueError translates the error using the given translator.
// Rule name and type are passed in.
func (e RuleInvalidValueError) Translate(t trans.Translator) string {
	return t.Tf(e.Error(), "rule", e.Rule.Name, "type", e.Rule.Type)
}

// Error on MissingRuleParserError returns the error code of the error.
func (e MissingRuleParserError) Error() string {
	return "eiffel.parser.error.missing-rule-parser"
}

// Translate on MissingRuleParserError translates the error using the given translator.
// Rule type is passed in.
func (e MissingRuleParserError) Translate(t trans.Translator) string {
	return t.Tf(e.Error(), "type", e.RuleType)
}

// Parse implements the RuleParser interface for the EqualsRuleParser. It is used to parse rules of the type 'equals'.
// The equals rule expects a string value, converts it to lowercase and compares it to the lowercase segment's value.
// If the values are not equal, a parsing error is reported.
func (p EqualsRuleParser) Parse(ctx context.Context, rule BasicRule, segment parser.ParsingSegment) ([]parser.ParsingLog, error) {
	rv, ok := rule.Value.(string)
	if !ok {
		return nil, RuleInvalidValueError{Rule: &rule}
	}

	segmentValue := strings.ToLower(segment.Value)
	ruleValue := strings.ToLower(rv)

	if segmentValue == ruleValue {
		return nil, nil
	}

	return []parser.ParsingLog{{
		Segment:         &segment,
		Level:           parser.ParsingLogLevelError,
		Message:         "eiffel.parser.equals.error",
		TranslationArgs: []string{"expected", rv, "actual", segment.Value}, // use the original values here
	}}, nil
}

// Validate implements the RuleParser interface for the EqualsRuleParser. It is used to validate rules of the type 'equals'.
// The equals rule expects a string value.
func (p EqualsRuleParser) Validate(v validation.V, rule BasicRule) []error {
	_, ok := rule.Value.(string)
	if ok {
		return nil
	}

	return []error{RuleInvalidValueError{Rule: &rule}}
}

// DisplayType implements the RuleParser interface for the EqualsRuleParser. Equals rules are arbitrary strings displayed as text.
func (p EqualsRuleParser) DisplayType(rule BasicRule) TemplateDisplayType {
	return TemplateDisplayString
}

// Parse implements the RuleParser interface for the EqualsAnyRuleParser. It is used to parse rules of the type 'equalsAny'.
// The equalsAny rule expects a slice of strings as value, converts each string to lowercase and compares it to the lowercase segment's value.
// If any of the values are equal, no parsing error is reported.
func (p EqualsAnyRuleParser) Parse(ctx context.Context, rule BasicRule, segment parser.ParsingSegment) ([]parser.ParsingLog, error) {
	rv, ok := rule.Value.([]string)
	if !ok {
		return nil, RuleInvalidValueError{Rule: &rule}
	}

	segmentValue := strings.ToLower(segment.Value)
	for _, v := range rv {
		ruleValue := strings.ToLower(v)

		if segmentValue == ruleValue {
			return nil, nil
		}
	}

	return []parser.ParsingLog{{
		Segment:         &segment,
		Level:           parser.ParsingLogLevelError,
		Message:         "eiffel.parser.equals-any.error",
		TranslationArgs: []string{"expected", strings.Join(rv, ", "), "actual", segment.Value}, // use the original values here
	}}, nil
}

// Validate implements the RuleParser interface for the EqualsAnyRuleParser. It is used to validate rules of the type 'equalsAny'.
// The equalsAny rule expects a slice of strings as value.
func (p EqualsAnyRuleParser) Validate(v validation.V, rule BasicRule) []error {
	_, ok := rule.Value.([]string)
	if ok {
		return nil
	}

	return []error{RuleInvalidValueError{Rule: &rule}}
}

// DisplayType implements the RuleParser interface for the EqualsAnyRuleParser. EqualsAny rules are input fields with a single select datalist.
func (p EqualsAnyRuleParser) DisplayType(rule BasicRule) TemplateDisplayType {
	return TemplateDisplayInputTypeSingleSelect
}

// Parse implements the RuleParser interface for the PlaceholderRuleParser. It is used to parse rules of the type 'placeholder'.
func (p PlaceholderRuleParser) Parse(ctx context.Context, rule BasicRule, segment parser.ParsingSegment) ([]parser.ParsingLog, error) {
	return nil, nil
}

// Validate implements the RuleParser interface for the PlaceholderRuleParser. It is used to validate rules of the type 'placeholder'.
func (p PlaceholderRuleParser) Validate(v validation.V, rule BasicRule) []error {
	return nil
}

// DisplayType implements the RuleParser interface for the PlaceholderRuleParser. Placeholder rules are input fields with a text type.
// The size of the input field is determined by the rule's size property. Large and full size will be rendered as a textarea.
func (p PlaceholderRuleParser) DisplayType(rule BasicRule) TemplateDisplayType {
	switch rule.Size {
	case "small":
		return TemplateDisplayInputTypeText
	case "medium":
		return TemplateDisplayInputTypeText
	case "large":
		return TemplateDisplayInputTypeTextarea
	case "full":
		return TemplateDisplayInputTypeTextarea
	default:
		return TemplateDisplayInputTypeText
	}
}

// prepareSegments prepares segments by trimming whitespaces from the input string and indexing them.
func prepareSegments(segments []parser.ParsingSegment) map[string]parser.ParsingSegment {
	indexedSegments := make(map[string]parser.ParsingSegment, len(segments))
	for _, segment := range segments {
		segment.Value = strings.TrimSpace(segment.Value)
		indexedSegments[segment.Name] = segment
	}

	return indexedSegments
}
