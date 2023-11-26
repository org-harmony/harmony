package template

import (
	"errors"
	"github.com/org-harmony/harmony/src/core/event"
	"github.com/org-harmony/harmony/src/core/trace"
	"github.com/org-harmony/harmony/src/core/validation"
)

var (
	// ErrValidateConfigEvent is returned when the validation of a template config failed during an event.
	// This should not happen and is therefore an internal error.
	ErrValidateConfigEvent = errors.New("validating template config failed during event")
	// ErrDidNotValidate is returned when a module did not validate a template config during an event.
	// This means that the template type is most likely not supported by any module.
	// Validation of templates before creation is required and therefore a validation error.
	ErrDidNotValidate = validation.Error{Msg: "template.new.did-not-validate"}
)

// ValidateTemplateConfigEvent is published to validate a template config. It allows for other modules to validate
// specific parts or entire templates based on their own rules. This is helpful if a module defines a template.ParsableTemplate
// and the template should be validated against the rules of the parser.
type ValidateTemplateConfigEvent struct {
	Config         string
	TemplateType   string
	validationErrs []error
	DidValidate    bool
}

// ValidateTemplateToCreate validates the template to create against the template set's rules and publishes an event
// to validate the template config. The event allows for other modules to validate specific parts or entire templates
// based on their own rules. This is helpful if a module defines a template.ParsableTemplate and the template should be
// validated against the rules of the parser.
func ValidateTemplateToCreate(toCreate *ToCreate, validator validation.V, em event.Manager, logger trace.Logger) ([]error, error) {
	err, validationErrs := validator.ValidateStruct(toCreate)
	if err != nil {
		return nil, err
	}

	configValidationErrs, err := publishValidationEvent(&ValidateTemplateConfigEvent{
		Config:       toCreate.Config,
		TemplateType: toCreate.Type,
	}, em, logger)
	if err != nil {
		return nil, err
	}

	validationErrs = append(validationErrs, configValidationErrs...)

	return validationErrs, nil
}

// ValidateTemplateToUpdate validates the template to update against the template set's rules and publishes an event
// to validate the template config. The event allows for other modules to validate specific parts or entire templates
// based on their own rules. This is helpful if a module defines a template.ParsableTemplate and the template should be
// validated against the rules of the parser.
func ValidateTemplateToUpdate(toUpdate *ToUpdate, validator validation.V, em event.Manager, logger trace.Logger) ([]error, error) {
	err, validationErrs := validator.ValidateStruct(toUpdate)
	if err != nil {
		return nil, err
	}

	configValidationErrs, err := publishValidationEvent(&ValidateTemplateConfigEvent{
		Config:       toUpdate.Config,
		TemplateType: toUpdate.Type,
	}, em, logger)
	if err != nil {
		return nil, err
	}

	validationErrs = append(validationErrs, configValidationErrs...)

	return validationErrs, nil
}

// ID returns the event id.
func (e *ValidateTemplateConfigEvent) ID() string {
	return "template.config.validate"
}

// Payload returns the event payload. It is the event itself as a pointer, the content should not be modified.
// Only errors should be added to the event.
func (e *ValidateTemplateConfigEvent) Payload() any {
	return e
}

// AddErrors adds an error to the event. They will be returned in the slice of validation errors.
// Errors added here should be safe to show to the user.
func (e *ValidateTemplateConfigEvent) AddErrors(errs ...error) {
	e.validationErrs = append(e.validationErrs, errs...)
}

// publishValidationEvent validates the config using an event published to other modules that may define their own parsers.
// It returns an error if the event execution failed. Otherwise, a slice of validation errors is returned.
func publishValidationEvent(validationEvent *ValidateTemplateConfigEvent, em event.Manager, logger trace.Logger) ([]error, error) {
	// TODO add tests for this
	dc := make(chan []error)
	em.Publish(validationEvent, dc)

	errs := <-dc
	if errs != nil {
		logger.Error(Pkg, "validating template config failed during event", nil, "errors", errs, "event", validationEvent.ID())
		return nil, ErrValidateConfigEvent
	}

	if !validationEvent.DidValidate {
		errs = append(errs, ErrDidNotValidate)
	}

	errs = append(errs, validationEvent.validationErrs...)

	return errs, nil
}
