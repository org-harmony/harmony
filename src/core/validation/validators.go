package validation

import (
	"errors"
	"reflect"
	"regexp"
	"strings"
)

func DefaultValidators() map[string]Func {
	return map[string]Func{
		"required": Required(),
		"notNil":   NotNil(),
		"positive": Positive(),
		"negative": Negative(),
		"email":    Email(),
	}
}

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
