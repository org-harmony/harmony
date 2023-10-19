package util

import "fmt"

// Unwrap panics if the error is not nil. Otherwise, it returns the value.
func Unwrap[T any](v T, e error) T {
	if e != nil {
		panic(fmt.Errorf("panic on error where error is not expected, check misconfigurations, see error: %w", e))
	}

	return v
}

// Ok panics if the error is not nil.
func Ok(e error) {
	if e == nil {
		return
	}

	panic(fmt.Errorf("panic on error where error is not expected, check misconfigurations, see error: %w", e))
}

// Wrap wraps an error with a message.
func Wrap(e error, msg string) error {
	if e == nil {
		return nil
	}

	return fmt.Errorf("%s: %w", msg, e)
}

// ErrErr wraps an error with another error.
func ErrErr(e1, e2 error) error {
	return fmt.Errorf("%w: %w", e1, e2)
}
