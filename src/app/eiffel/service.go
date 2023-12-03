package eiffel

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/org-harmony/harmony/src/app/template"
	"github.com/org-harmony/harmony/src/app/template/parser"
	"github.com/org-harmony/harmony/src/core/validation"
	"net/http"
	"strings"
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

// SegmentMapFromRequest parses the segments from the request and returns a map of segment names to values.
// The length parameter is used to initialize the map with a given length. If the length is 0, the map will be
// initialized with a length of 0, no error will occur. The length is only used for pre-allocation.
//
// Segments are expected to be in the form of "segment-<name>".
//
// Use SegmentMapToSegments to convert the map into a slice of ParsingSegments.
func SegmentMapFromRequest(request *http.Request, length int) (map[string]string, error) {
	err := request.ParseForm()
	if err != nil {
		return nil, err
	}

	var segments map[string]string
	if length > 0 {
		segments = make(map[string]string, length)
	} else {
		segments = make(map[string]string)
	}

	for name, values := range request.Form {
		if !strings.HasPrefix(name, "segment-") {
			continue
		}

		if len(values) < 1 {
			continue
		}

		segments[strings.TrimPrefix(name, "segment-")] = values[0]
	}

	return segments, nil
}

// SegmentMapToSegments converts a map of segment names to values into a slice of ParsingSegments.
// The order of the segments in the slice is not guaranteed. The map can be generated using SegmentMapFromRequest.
func SegmentMapToSegments(segments map[string]string) []parser.ParsingSegment {
	parsingSegments := make([]parser.ParsingSegment, 0, len(segments))
	for name, value := range segments {
		parsingSegments = append(parsingSegments, parser.ParsingSegment{
			Name:  name,
			Value: value,
		})
	}

	return parsingSegments
}
