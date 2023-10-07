package util

import "fmt"

func Unwrap[T any](v T, e error) T {
	if e != nil {
		panic(fmt.Errorf("panic on error where error is not expected, check misconfigurations, see error: %w", e))
	}

	return v
}

func Ok(e error) {
	if e == nil {
		return
	}

	panic(fmt.Errorf("panic on error where error is not expected, check misconfigurations, see error: %w", e))
}
