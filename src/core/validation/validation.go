package validation

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
)

// StructTag is the default struct tag used for validation.
// Using this struct tag will allow the validator to validate struct fields in V.ValidateStruct.
const StructTag = "hvalidate"

var (
	// ErrUnexpected is returned when an unexpected error occurs, e.g. if the reflection fails for an unknown reason.
	ErrUnexpected = errors.New("unexpected error")
	// ErrNotStruct is returned when a non-struct value is passed to ValidateStruct.
	ErrNotStruct = errors.New("not a struct")
	// ErrUnknownValidator is returned when an unknown validator is used.
	ErrUnknownValidator = errors.New("unknown validator")
)

// Validator is a concurrency-safe structure that holds the validation rules (Func or Validators)
// and configuration for validating structs. It contains a map of validation functions and struct tags used for validation.
//
// TODO: Implement schema caching in ValidateStruct.schemaCache to improve performance by avoiding repetitive processing of the same struct types.
type Validator struct {
	structTags  []string
	funcs       map[string]Func
	fmu         sync.RWMutex
	schemaCache map[reflect.Type][]Func
	scmu        sync.RWMutex
}

// Error struct holds detailed information about a validation error.
type Error struct {
	Msg    string
	Struct string // namespace and name of the struct, e.g. "config/SomeCfg"
	Field  string // name of the field, e.g. "SomeField"
	Path   string // path to the field, e.g. "config/SomeCfg.SomeField(string)"
}

// Func is a function that validates a value and returns an error if the value is invalid.
// A Func should only ever validate one thing, e.g.
// if a Func validates that a string is an email it should not also validate that the string not-empty.
// Instead, it should return early if the string is empty and another Func should be used to validate that the string is not-empty.
type Func func(any) error

// ValidatorOption is a function that configures a Validator. It can be used to override the default validator Func map or struct tags.
type ValidatorOption func(*Validator)

// TransparentError is an error that can be unwrapped to get the underlying validation error.
// Usually, an error returned by a validation Func will be converted to a validation.Error.
// However, if the error implements TransparentError the error will be unwrapped and the underlying validation error will be returned.
// This allows validation Funcs to return custom errors that can be used to seek more information about the error.
type TransparentError interface {
	UnwrapTransparent(Error) error
}

// V is a thread-safe validator that allows validating structs. It can be configured using ValidatorOption functions passed to New.
// If no ValidatorOption functions are passed to New validation.defaultValidator will be used.
// The Func functions on the validator will be used to validate the struct fields. If none are passed to New validation.DefaultValidators will be used.
// The alternative struct tags used for validation can be defined using the WithStructTags Option.
//
// Example usage:
//
//	validator := validation.New(validation.WithValidators(map[string]validation.Func{
//		"notEmptyString": validation.Required(),
//		"isY":            validation.isY, // not part of the default validators (currently)
//	}))
//
//	type Some struct {
//		Description string `hvalidate:"notEmptyString,isY"`
//	}
//
//	some := Some{Description: "Y"}
//	if err := validator.ValidateStruct(some); err != nil {
//		// no error expected
//	}
type V interface {
	// ValidateStruct validates the exported fields of an (un-)exported-struct. It will validate nested structs as well. Not-exported fields will be ignored.
	// The validation is done using the StructTag on the struct fields. The StructTag is a comma-separated list of validator names.
	// It returns a hard error if the passed in struct is not a struct, the reflection fails for an unknown reason or if an unknown validator is used.
	// Otherwise, it returns a slice of validation errors. The slice is empty if no errors occurred.
	ValidateStruct(any, ...string) (error, []error)
	// AddFunc adds a new validation function to the validator.
	AddFunc(string, Func)
	// Lookup returns the validation function for the given name and a bool indicating if the function was found.
	Lookup(string) (Func, bool)
}

// WithValidators allow overriding the validation.DefaultValidators.
// If you don't want to override the default validators but just add some new ones use the AddFunc/Lookup method.
func WithValidators(funcs map[string]Func) ValidatorOption {
	return func(opts *Validator) {
		opts.funcs = funcs
	}
}

// WithStructTags allows overriding the default validation.StructTag used for validation.
func WithStructTags(tags ...string) ValidatorOption {
	return func(opts *Validator) {
		opts.structTags = tags
	}
}

// New returns Validator as configured by defaultValidator and the given ValidatorOption functions.
func New(opts ...ValidatorOption) V {
	v := defaultValidator()

	for _, opt := range opts {
		opt(v)
	}

	return v
}

// ValidateStruct implements the ValidateStruct method of the V interface.
// It performs validation on struct fields based on defined validation functions.
// Refer to the documentation of the V interface for more detailed information.
//
// The returned validation error slice will contain errors of type Error as well as unwrapped TransparentError`s.
// It can not be assumed that the returned error slice only contains Error`s.
// However, it can be assumed that the returned error slice contains errors that occurred during validation and imply that the data is invalid.
//
// TODO break this function up into smaller functions
func (v *Validator) ValidateStruct(s any, path ...string) (hardErr error, validationErrs []error) {
	defer func() {
		if r := recover(); r != nil {
			errorPath := "<unknown>"
			if len(path) > 0 {
				errorPath = path[0]
			}

			hardErr = fmt.Errorf("%w: %v on %s", ErrUnexpected, r, errorPath)
		}
	}()

	typeOfS := reflect.TypeOf(s)
	valueOfS := reflect.ValueOf(s)

	currentPath := ""
	if len(path) < 1 {
		currentPath = typeOfS.String() // initialize with root struct
	}

	if len(path) >= 1 {
		currentPath = path[0] // only first should be set, ignore the rest
	}

	if typeOfS.Kind() == reflect.Pointer {
		typeOfS = typeOfS.Elem()
		valueOfS = valueOfS.Elem()
	}

	if typeOfS.Kind() != reflect.Struct {
		return fmt.Errorf("%w on %s", ErrNotStruct, currentPath), nil
	}

	var errs []error
	for i := 0; i < typeOfS.NumField(); i++ {
		typeOfField := typeOfS.Field(i)
		valueOfField := valueOfS.Field(i)

		fieldPath := fmt.Sprintf("%s.%s(%s)", currentPath, typeOfField.Name, typeOfField.Type.String()) // construct path for field (e.g. "config/SomeCfg.SomeField(string)")

		if typeOfField.PkgPath != "" { // skip unexported fields
			continue
		}

		// handle nested structs
		kind := typeOfField.Type.Kind()
		isStruct := kind == reflect.Struct
		isPtr := kind == reflect.Ptr
		if (isStruct || (isPtr && !valueOfField.IsNil())) && valueOfField.CanInterface() { // is a struct, non-nil pointer and can be interfaced
			h, v := v.ValidateStruct(valueOfField.Interface(), fieldPath)
			if h != nil {
				return fmt.Errorf("%w on %s", h, fieldPath), nil
			}

			errs = append(errs, v...)
		}

		validatorNames := make([]string, 0)
		for _, tag := range v.structTags {
			validatorName := typeOfField.Tag.Get(tag)
			if validatorName == "" {
				continue
			}

			validatorNames = append(validatorNames, strings.Split(validatorName, ",")...)
		}

		for _, validatorName := range validatorNames {
			validatorName = strings.TrimSpace(validatorName)
			if validatorName == "" {
				continue
			}

			v.fmu.RLock()
			validatorFunc, ok := v.funcs[validatorName]
			v.fmu.RUnlock()
			if !ok {
				return fmt.Errorf("%w: %s on %s", ErrUnknownValidator, validatorName, fieldPath), nil
			}

			if !valueOfField.CanInterface() {
				continue
			}

			err := validatorFunc(valueOfField.Interface())
			if err == nil {
				continue
			}

			var validationErr error
			validationErr = Error{Msg: err.Error(), Struct: typeOfS.Name(), Field: typeOfField.Name, Path: fieldPath}

			if terr, ok := err.(TransparentError); ok {
				validationErr = terr.UnwrapTransparent(validationErr.(Error))
			}

			errs = append(errs, validationErr)
		}
	}

	return nil, errs
}

// AddFunc implements the AddFunc method of the V interface. It locks the mutex before adding the function to the map.
func (v *Validator) AddFunc(name string, f Func) {
	v.fmu.Lock()
	v.funcs[name] = f
	v.fmu.Unlock()
}

// Lookup implements the Lookup method of the V interface. It locks the mutex before looking up the function in the map.
func (v *Validator) Lookup(name string) (Func, bool) {
	v.fmu.RLock()
	f, ok := v.funcs[name]
	v.fmu.RUnlock()

	return f, ok
}

// Validate validates a single value using the given validation funcs.
// The returned error slice will contain all validation Error`s as well as any other errors that occurred during validation.
//
// For validation Error`s the fields Field, Struct and Path will be set through the passed in values for fieldName and structName.
// Passing in fieldName and structName is recommended for proper error message construction. However, depending on the use-case this is not necessary.
//
// If a Func returns a TransparentError the returned error will be unwrapped.
// Due to unwrapping transparent errors the returned error slice may contain any error type.
//
// This function is especially useful for validating single values, e.g. validating a string is an email.
func Validate(fieldName string, structName string, value any, funcs ...Func) []error {
	var errs []error
	for _, f := range funcs {
		if err := f(value); err != nil {
			var validationErr error
			validationErr = Error{
				Msg:    err.Error(),
				Field:  fieldName,
				Struct: structName,
				Path:   fmt.Sprintf("%s.%s(%s)", structName, fieldName, reflect.TypeOf(value).String()),
			}

			if terr, ok := err.(TransparentError); ok {
				validationErr = terr.UnwrapTransparent(validationErr.(Error))
			}

			errs = append(errs, validationErr)
		}
	}

	return errs
}

// GenericErrorKey returns a generic error key for the validation error.
// The key is constructed as follows: "<msg>.generic". This can be used for i18n.
func (e Error) GenericErrorKey() string {
	return fmt.Sprintf("%s.generic", e.Msg)
}

// FieldErrorKey returns a field error key for the validation error.
// The key is constructed as follows: "<msg>.field.<field>". This can be used for i18n.
func (e Error) FieldErrorKey() string {
	if e.Field != "" {
		return fmt.Sprintf("%s.field.%s", e.Msg, e.Field)
	}

	return e.Msg
}

// Error implements the Error method of the error interface by returning a string representation of the validation error using the following format:
// "<path to field>: <msg> (on struct: <struct>, field: <field>)"
func (e Error) Error() string {
	return fmt.Sprintf("%s: %s (on struct: %s, field: %s)", e.Path, e.Msg, e.Struct, e.Field)
}

// defaultValidator returns a new Validator with the default validation funcs and struct tag.
func defaultValidator() *Validator {
	return &Validator{
		funcs:       DefaultValidators(),
		structTags:  []string{StructTag},
		schemaCache: make(map[reflect.Type][]Func),
	}
}
