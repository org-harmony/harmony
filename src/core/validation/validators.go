package validation

import (
	"errors"
	"reflect"
	"regexp"
	"strings"
)

// DefaultValidators returns a map of default validators (validation.Func).
// These validators can be used to validate values.
func DefaultValidators() map[string]Func {
	return map[string]Func{
		"required": Required(),
		"notNil":   NotNil(),
		"positive": Positive(),
		"negative": Negative(),
		"email":    Email(),
		"semVer":   SemanticVersion(),
	}
}

// Required validates that the value is not nil, empty or zero.
func Required(msgs ...string) Func {
	msg := "harmony.error.validation.required"
	if len(msgs) > 0 && msgs[0] != "" {
		msg = msgs[0]
	}

	return func(value any) error {
		v := reflect.ValueOf(value)

		if !v.IsValid() {
			return errors.New(msg)
		}

		if str, ok := value.(string); ok {
			if strings.TrimSpace(str) == "" {
				return errors.New(msg)
			}
		}

		if v.Kind() == reflect.Map || v.Kind() == reflect.Slice || v.Kind() == reflect.Array {
			if v.Len() == 0 {
				return errors.New(msg)
			}
		}

		return nil
	}
}

// NotNil validates that the value is not nil. It does not check for empty or zero values. Use Required for that.
func NotNil(msgs ...string) Func {
	msg := "harmony.error.validation.notNil"
	if len(msgs) > 0 && msgs[0] != "" {
		msg = msgs[0]
	}

	return func(value any) error {
		v := reflect.ValueOf(value)

		if value == nil {
			return errors.New(msg)
		}

		switch v.Kind() {
		case reflect.Ptr, reflect.Slice, reflect.Map, reflect.Array, reflect.Chan, reflect.Func, reflect.Interface:
			if v.IsNil() {
				return errors.New(msg)
			}
		}

		return nil
	}
}

// Positive validates that the value is positive. Non-integer values return a validation error.
func Positive(msgs ...string) Func {
	msg := "harmony.error.validation.positive"
	if len(msgs) > 0 && msgs[0] != "" {
		msg = msgs[0]
	}

	return func(value any) error {
		if num, ok := value.(int); ok && num <= 0 {
			return errors.New(msg)
		}

		return nil
	}
}

// Negative validates that the value is negative. Non-integer values return a validation error.
func Negative(msgs ...string) Func {
	msg := "harmony.error.validation.negative"
	if len(msgs) > 0 && msgs[0] != "" {
		msg = msgs[0]
	}

	return func(value any) error {
		if num, ok := value.(int); ok && num >= 0 {
			return errors.New(msg)
		}

		return nil
	}
}

// Email validates that the value is a valid email address. Empty values are ignored, non-string values return a validation error.
func Email(msgs ...string) Func {
	msg := "harmony.error.validation.email"
	if len(msgs) > 0 && msgs[0] != "" {
		msg = msgs[0]
	}

	var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

	return func(value any) error {
		email, ok := value.(string)
		if !ok {
			return errors.New(msg)
		}

		if email == "" {
			return nil
		}

		if !emailRegex.MatchString(email) {
			return errors.New(msg)
		}

		return nil
	}
}

// SemanticVersion validates that the value is a semantic version. Empty values are ignored, non-string values return a validation error.
func SemanticVersion(msgs ...string) Func {
	msg := "harmony.error.validation.semantic-version"
	if len(msgs) > 0 && msgs[0] != "" {
		msg = msgs[0]
	}

	var semanticVersionRegex = regexp.MustCompile(`^(?:(\d+)\.)?(?:(\d+)\.)?(\d+)$`)

	return func(value any) error {
		version, ok := value.(string)
		if !ok {
			return errors.New(msg)
		}

		if version == "" {
			return nil
		}

		if !semanticVersionRegex.MatchString(version) {
			return errors.New(msg)
		}

		return nil
	}
}
