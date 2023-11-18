package eiffel

import (
	"errors"
	"github.com/org-harmony/harmony/src/core/trans"
	"github.com/org-harmony/harmony/src/core/validation"
)

const BasicTemplateName = "ebt"

var (
	ErrInvalidTemplate = errors.New("eiffel.parser.error.invalid-template")
)

type ParsingResult struct {
	Template string
	Variant  string
	Errors   []error
	Warnings []error
	Notices  []error
}

type BasicParser struct {
	Template      *BasicTemplate `hvalidate:"required,ruleReferences"`
	Preprocessors map[string]BasicPreprocessor
}

type BasicTemplate struct {
	Name          string                  `json:"name" hvalidate:"required"`
	Version       string                  `json:"version" hvalidate:"required"`
	Authors       []string                `json:"authors"`
	License       string                  `json:"license"`
	Description   string                  `json:"description"`
	Format        string                  `json:"format"`
	Example       string                  `json:"example"`
	Preprocessors []string                `json:"preprocessors"`
	Rules         map[string]BasicRule    `json:"rules"`
	Variants      map[string]BasicVariant `json:"variants" hvalidate:"required"`
}

type BasicRule struct {
	Name        string `json:"name" hvalidate:"required"`
	Type        string `json:"type" hvalidate:"required"`
	Hint        string `json:"hint"`
	Explanation string `json:"explanation"`
	Value       any    `json:"value"`
	Optional    bool   `json:"optional"`
}

type BasicVariant struct {
	Name        string `json:"name" hvalidate:"required"`
	Description string `json:"description"`
	Format      string `json:"format"`
	Example     string `json:"example"`
	// Rules contains rule names, rule objects should be contained in the template
	Rules []string `json:"rules"`
}

type RuleMissingError struct {
	Rule     string
	Template string
}

type PreprocessorMissingError struct {
	Preprocessor string
	Template     string
}

type BasicPreprocessor func(string) (string, error)

func (p *BasicParser) Parse(input string, variant string) (string, error) {
	/*v, ok := p.Template.Variants[variant]
	if !ok {
		return "", errors.New("eiffel.parser.error.variant-not-found")
	}*/
	return "", nil
}

func (p *BasicParser) Validate(v validation.V) []error {
	v.AddFunc("ruleReferences", RuleReferencesValidator)

	errs := validation.Validate("", "", p, PreprocessorsValidator)
	if len(errs) > 0 {
		return append(errs, ErrInvalidTemplate)
	}

	err, validationErrs := v.ValidateStruct(p)
	if err != nil {
		return []error{err}
	}

	if len(validationErrs) > 0 {
		return append(validationErrs, ErrInvalidTemplate)
	}

	return nil
}

func RuleReferencesValidator(template any) error {
	t, ok := template.(*BasicTemplate)
	if !ok {
		return nil
	}

	for _, variant := range t.Variants {
		for _, rule := range variant.Rules {
			if _, ok := t.Rules[rule]; ok {
				continue
			}

			return RuleMissingError{
				Rule:     rule,
				Template: t.Name,
			}
		}
	}

	return nil
}

func PreprocessorsValidator(template any) error {
	t, ok := template.(*BasicParser)
	if !ok {
		return nil
	}

	for _, preprocessor := range t.Template.Preprocessors {
		if _, ok := t.Preprocessors[preprocessor]; ok {
			continue
		}

		return PreprocessorMissingError{
			Preprocessor: preprocessor,
			Template:     t.Template.Name,
		}
	}

	return nil
}

func (e RuleMissingError) Error() string {
	return "eiffel.parser.error.missing-rule"
}

func (e RuleMissingError) UnwrapTransparent(err validation.Error) error {
	return e
}

func (e RuleMissingError) Translate(t trans.Translator) string {
	return t.Tf(e.Error(), "template", e.Template, "rule", e.Rule)
}

func (e PreprocessorMissingError) Error() string {
	return "eiffel.parser.error.missing-preprocessor"
}

func (e PreprocessorMissingError) UnwrapTransparent(err validation.Error) error {
	return e
}

func (e PreprocessorMissingError) Translate(t trans.Translator) string {
	return t.Tf(e.Error(), "template", e.Template, "preprocessor", e.Preprocessor)
}
