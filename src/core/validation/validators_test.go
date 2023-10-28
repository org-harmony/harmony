package validation_test

import (
	"testing"

	"github.com/org-harmony/harmony/src/core/validation"
	"github.com/stretchr/testify/assert"
)

func TestRequired(t *testing.T) {
	validator := validation.Required()

	tests := []struct {
		value any
		valid bool
	}{
		{"hello", true},
		{"", false},
		{"    ", false},
		{nil, false},
		{[]string{"hello"}, true},
		{[]string{}, false},
		{map[string]string{"key": "value"}, true},
		{map[string]string{}, false},
		{[5]int{0, 1, 2, 3, 4}, true},
		{[0]int{}, false},
	}

	for _, test := range tests {
		err := validator(test.value)
		if test.valid {
			assert.NoError(t, err)
		} else {
			assert.Error(t, err)
		}
	}
}

func TestNotNil(t *testing.T) {
	validator := validation.NotNil()

	var nilPtr *int

	tests := []struct {
		value any
		valid bool
	}{
		{"hello", true},
		{[]string{"hello"}, true},
		{[]string{}, true},
		{nil, false},
		{nilPtr, false},
		{(func())(nil), false},
		{(chan int)(nil), false},
		{(map[string]string)(nil), false},
		{(*int)(nil), false},
	}

	for _, test := range tests {
		err := validator(test.value)
		if test.valid {
			assert.NoError(t, err)
		} else {
			assert.Error(t, err)
		}
	}
}

func TestPositive(t *testing.T) {
	validator := validation.Positive()

	tests := []struct {
		value any
		valid bool
	}{
		{5, true},
		{0, false},
		{-1, false},
	}

	for _, test := range tests {
		err := validator(test.value)
		if test.valid {
			assert.NoError(t, err)
		} else {
			assert.Error(t, err)
		}
	}
}

func TestNegative(t *testing.T) {
	validator := validation.Negative()

	tests := []struct {
		value any
		valid bool
	}{
		{-5, true},
		{0, false},
		{1, false},
	}

	for _, test := range tests {
		err := validator(test.value)
		if test.valid {
			assert.NoError(t, err)
		} else {
			assert.Error(t, err)
		}
	}
}

func TestEmail(t *testing.T) {
	validator := validation.Email()

	tests := []struct {
		value any
		valid bool
	}{
		{"test@example.com", true},
		{"k@x.de", true},
		{"", true}, // empty string is considered valid in this implementation
		{"notAnEmail", false},
	}

	for _, test := range tests {
		err := validator(test.value)
		if test.valid {
			assert.NoError(t, err)
		} else {
			assert.Error(t, err)
		}
	}
}
