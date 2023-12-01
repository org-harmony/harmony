package eiffel

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/org-harmony/harmony/src/app/template"
	"github.com/org-harmony/harmony/src/core/validation"
)

// TemplateDisplayTypes returns a map of rule names to display types. The rule names are the keys of the BasicTemplate.Rules map.
// This can be used in the eiffel.TemplateFormData`.DisplayTypes field.
func TemplateDisplayTypes(bt *BasicTemplate, ruleParsers *RuleParserProvider) map[string]TemplateDisplayType {
	displayTypes := map[string]TemplateDisplayType{}

	for ruleName, rule := range bt.Rules {
		ruleParser, err := ruleParsers.Parser(rule.Type)
		if err != nil {
			continue
		}

		displayType := ruleParser.DisplayType(rule)
		if displayType == "" {
			continue
		}

		displayTypes[ruleName] = displayType
	}

	return displayTypes
}

// TemplateIntoBasicTemplate parses a templates config into a BasicTemplate and validates it.
// If unmarshalling the config into the BasicTemplate fails or validation fails, an error is returned.
func TemplateIntoBasicTemplate(t *template.Template, validator validation.V, ruleParsers *RuleParserProvider) (*BasicTemplate, error) {
	ebt := &BasicTemplate{}
	err := json.Unmarshal([]byte(t.Config), ebt)
	if err != nil {
		return nil, err
	}

	errs := ebt.Validate(validator, ruleParsers)
	if len(errs) > 0 {
		return nil, template.ErrInvalidTemplate
	}

	return ebt, nil
}

// TemplateFormFromRequest parses the template and variant from the passed in templateID and variantKey and returns a
// TemplateFormData struct. If the template or variant could not be found, an error is returned.
// However, using the defaultFirstVariant flag, the first variant will be used if no variant was specified and no
// error will be returned. TemplateFormFromRequest will also parse and validate the template.
//
// Returned errors from TemplateFormFromRequest are safe to display to the user.
func TemplateFormFromRequest(
	ctx context.Context,
	templateID string,
	variantKey string,
	templateRepository template.Repository,
	ruleParsers *RuleParserProvider,
	validator validation.V,
	defaultFirstVariant bool,
) (TemplateFormData, error) {
	templateUUID, err := uuid.Parse(templateID)
	if err != nil {
		return TemplateFormData{}, ErrTemplateNotFound
	}

	tmpl, err := templateRepository.FindByID(ctx, templateUUID)
	if err != nil {
		return TemplateFormData{}, ErrTemplateNotFound
	}

	bt, err := TemplateIntoBasicTemplate(tmpl, validator, ruleParsers)
	if err != nil {
		return TemplateFormData{}, err
	}

	variant, ok := bt.Variants[variantKey]
	if !ok && !defaultFirstVariant {
		return TemplateFormData{}, ErrTemplateVariantNotFound
	}

	if !ok {
		for n, v := range bt.Variants {
			variant = v
			variantKey = n
			break
		}
	}

	displayTypes := TemplateDisplayTypes(bt, RuleParsers())

	return TemplateFormData{
		Template:     bt,
		Variant:      &variant,
		VariantKey:   variantKey,
		DisplayTypes: displayTypes,
		TemplateID:   templateUUID,
	}, nil
}
