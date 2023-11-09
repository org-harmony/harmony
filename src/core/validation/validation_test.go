package validation_test

import (
	"errors"
	"fmt"
	"github.com/org-harmony/harmony/src/core/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"reflect"
	"strings"
	"testing"
)

var ErrFooInvalid = errors.New("what the foo")

type InnerStruct struct {
	Description string `hvalidate:"notEmptyString, isY"`
}

type TestStruct struct {
	Name     string `hvalidate:"notEmptyString"`
	Age      int    `hvalidate:"positive"`
	Inner    InnerStruct
	InnerRef *InnerStruct `hvalidate:"notNil"`
}

type TestStruct2 struct {
	Name    string   `hvalidate:"notEmptyString"`
	Consent string   `hvalidate:"isY"`
	Rules   []string `hvalidate:"notNil,notEmptyString"`
}

type TestUnexportedStruct struct {
	name string `hvalidate:"notEmptyString"`
	ref  *unexportedStruct
}

type TestUnexportedStruct2 struct {
	Ref *unexportedStruct
}

type unexportedStruct struct {
	Cond string `hvalidate:"isY"`
}

type testStruct3 struct {
	Name  string `hvalidate:"notEmptyString"`
	Email string `xxx:"notEmptyString"`
	Road  string `foo:"notEmptyString"`
}

type TransparentlyInvalidStruct struct {
	Foo string `hvalidate:"returnsTransparentError,notEmptyString"`
}

type TransparentError struct{}

func TestValidator_ValidateStruct(t *testing.T) {
	v := mockValidator()

	tests := []struct {
		input    any
		expected int // expected number of validation errors
	}{
		{TestStruct{Name: "John", Age: 30, Inner: InnerStruct{Description: "Y"}, InnerRef: &InnerStruct{Description: "Y"}}, 0},
		{TestStruct{Name: "", Age: 30, Inner: InnerStruct{Description: "Y"}, InnerRef: &InnerStruct{Description: "Y"}}, 1},
		{TestStruct{Name: "John", Age: 0, Inner: InnerStruct{Description: "Y"}, InnerRef: &InnerStruct{Description: "Y"}}, 1},
		{TestStruct{Name: "John", Age: 30, Inner: InnerStruct{Description: "N"}, InnerRef: &InnerStruct{Description: "Y"}}, 1},
		{TestStruct{Name: "John", Age: 30, Inner: InnerStruct{Description: "Y"}, InnerRef: nil}, 1},
		{TestStruct{Name: "John", Age: 30, Inner: InnerStruct{Description: ""}, InnerRef: &InnerStruct{Description: ""}}, 2},
		{TestStruct2{Name: "John", Consent: "Y", Rules: []string{"a", "b"}}, 0},
		{TestStruct2{Name: "John", Consent: "N", Rules: []string{"a", "b"}}, 1},
		{&TestStruct2{Name: "John", Consent: "Y", Rules: []string{}}, 0}, // because of notEmptyString only working on string not slices
		{&TestStruct2{Name: "John", Consent: "Y", Rules: nil}, 1},
		{TestUnexportedStruct{name: "John"}, 0},
		{TestUnexportedStruct{name: ""}, 0},
		{TestUnexportedStruct{ref: &unexportedStruct{Cond: "N"}}, 0},
		{unexportedStruct{Cond: "N"}, 1},
		{TestUnexportedStruct2{Ref: &unexportedStruct{Cond: "N"}}, 1},
	}

	for k, test := range tests {
		herr, errs := v.ValidateStruct(test.input)
		assert.NoError(t, herr)
		assert.Equal(t, test.expected, len(errs), "Unexpected number of validation errors on %d", k+1)
	}
}

func TestValidator_ValidateStruct_TransparentError(t *testing.T) {
	v := mockValidator()

	err, errs := v.ValidateStruct(TransparentlyInvalidStruct{Foo: ""})
	require.NoError(t, err)
	require.Len(t, errs, 2)
	assert.ErrorIs(t, errs[0], ErrFooInvalid)
	assert.ErrorAs(t, errs[1], &validation.Error{})
}

func TestValidator_AddFunc(t *testing.T) {
	v := validation.New(validation.WithValidators(map[string]validation.Func{}))
	v.AddFunc("notEmptyString", notEmptyString)

	err, _ := v.ValidateStruct("not a struct")
	assert.ErrorIs(t, err, validation.ErrNotStruct)

	testStruct := &TestStruct{Name: "John", Age: 30, Inner: InnerStruct{Description: "Y"}, InnerRef: &InnerStruct{Description: "Y"}}
	err, _ = v.ValidateStruct(testStruct)
	assert.ErrorIs(t, err, validation.ErrUnknownValidator)

	_, ok := v.Lookup("notEmptyString")
	assert.True(t, ok)

	v.AddFunc("positive", positive)
	v.AddFunc("notNil", notNil)
	v.AddFunc("isY", isY)

	err, validationErrs := v.ValidateStruct(testStruct)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(validationErrs))
}

func TestValidate(t *testing.T) {
	tests := []struct {
		input    any
		expected int // expected number of validation errors
	}{
		{"John", 0},
		{"", 1},
		{0, 0},
	}

	for k, test := range tests {
		err := validation.Validate("test", "", test.input, notEmptyString)
		assert.Equal(t, test.expected, len(err), "Unexpected number of validation errors on %d", k+1)
	}
}

func TestValidate_TransparentError(t *testing.T) {
	err := validation.Validate("test", "", "", returnsTransparentError, notEmptyString)
	require.Len(t, err, 2)
	assert.ErrorIs(t, err[0], ErrFooInvalid)
	assert.ErrorAs(t, err[1], &validation.Error{})
}

func TestMultipleStructTags(t *testing.T) {
	v := validation.New(validation.WithStructTags(validation.StructTag, "xxx"), validation.WithValidators(map[string]validation.Func{
		"notEmptyString": notEmptyString,
	}))

	err, errs := v.ValidateStruct(testStruct3{Name: "", Email: "", Road: ""})
	assert.NoError(t, err)
	assert.Len(t, errs, 2)

	err, errs = v.ValidateStruct(testStruct3{Name: "John", Email: "", Road: ""})
	assert.NoError(t, err)
	assert.Len(t, errs, 1)

	err, errs = v.ValidateStruct(testStruct3{Name: "John", Email: "john@foo.bar", Road: ""})
	assert.NoError(t, err)
	assert.Len(t, errs, 0)

	v = validation.New(validation.WithStructTags("foo"), validation.WithValidators(map[string]validation.Func{
		"notEmptyString": notEmptyString,
	}))

	err, errs = v.ValidateStruct(testStruct3{Name: "", Email: "", Road: ""})
	assert.NoError(t, err)
	assert.Len(t, errs, 1)

	err, errs = v.ValidateStruct(testStruct3{Name: "", Email: "", Road: "John's Road"})
	assert.NoError(t, err)
	assert.Len(t, errs, 0)
}

func TestNestedStructErrorPath(t *testing.T) {
	v := mockValidator()

	err, validationErrs := v.ValidateStruct(TestStruct{Name: "John", Age: 30, Inner: InnerStruct{Description: ""}, InnerRef: &InnerStruct{Description: ""}})
	assert.NoError(t, err)
	assert.Equal(t, 2, len(validationErrs))

	for _, verr := range validationErrs {
		var vError validation.Error
		ok := errors.As(verr, &vError)
		assert.True(t, ok)

		assert.NotEmpty(t, vError.Path) // asserting equal would be too strict as it depends on exact implementation
		assert.Equal(t, "should be non-empty", vError.Msg)
		assert.Equal(t, "InnerStruct", vError.Struct)
		assert.Equal(t, "Description", vError.Field)
	}
}

func TestErrorFormatting(t *testing.T) {
	v := mockValidator()

	err, validationErrs := v.ValidateStruct(TestStruct{Name: "John", Age: 30, Inner: InnerStruct{Description: ""}, InnerRef: &InnerStruct{Description: ""}})
	assert.NoError(t, err)
	assert.Equal(t, 2, len(validationErrs))

	for _, verr := range validationErrs {
		var vError validation.Error
		ok := errors.As(verr, &vError)
		assert.True(t, ok)

		assert.Equal(t, "should be non-empty.generic", vError.GenericErrorKey())
		assert.Equal(t, "should be non-empty.field.Description", vError.FieldErrorKey())
	}
}

func (e *TransparentError) Error() string {
	return ErrFooInvalid.Error()
}

func (e *TransparentError) UnwrapTransparent(err validation.Error) error {
	return ErrFooInvalid
}

func mockValidator() validation.V {
	validatorFuncs := map[string]validation.Func{
		"returnsTransparentError": returnsTransparentError,
		"notEmptyString":          notEmptyString,
		"positive":                positive,
		"notNil":                  notNil,
		"isY":                     isY,
	}

	return validation.New(validation.WithValidators(validatorFuncs))
}

func returnsTransparentError(value any) error {
	return &TransparentError{}
}

func notEmptyString(value any) error {
	if str, ok := value.(string); ok && str == "" {
		return fmt.Errorf("should be non-empty")
	}

	return nil
}

func positive(value any) error {
	if num, ok := value.(int); ok && num <= 0 {
		return fmt.Errorf("should be positive")
	}

	return nil
}

func notNil(value any) error {
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Ptr, reflect.Slice, reflect.Map, reflect.Array, reflect.Chan, reflect.Func, reflect.Interface:
		if v.IsNil() {
			return fmt.Errorf("should not be nil")
		}
	}

	return nil
}

func isY(value any) error {
	str, ok := value.(string)
	if !ok {
		return nil
	}

	if str == "" { // let notEmptyString handle this
		return nil
	}

	if strings.ToLower(str) != "y" {
		return fmt.Errorf("should be Y")
	}

	return nil
}
