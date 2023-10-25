package util

import (
	"context"
	"fmt"
)

// Unwrap panics if the error is not nil. Otherwise, it returns the value.
func Unwrap[T any](v T, e error) T {
	if e != nil {
		panic(fmt.Errorf("unwrap failed - error is not expected, check misconfigurations, see error: %w", e))
	}

	return v
}

// UnwrapType panics if the error is not nil. Otherwise, it returns the value if it is of type T.
// If v is not of type T, it panics.
func UnwrapType[T any](v any, e error) T {
	if e != nil {
		panic(fmt.Errorf("unwrap failed - error is not expected, check misconfigurations, see error: %w", e))
	}

	vT, ok := v.(T)
	if !ok {
		panic(fmt.Errorf("unwrap type failed - expected type %T, got %T", v, vT))
	}

	return vT
}

// Ok panics if the error is not nil.
func Ok(e error) {
	if e == nil {
		return
	}

	panic(fmt.Errorf("ok assumption failed - error is not expected, check misconfigurations, see error: %w", e))
}

// Wrap wraps an error with a message.
func Wrap(e error, msg string) error {
	if e == nil {
		return nil
	}

	return fmt.Errorf("%s: %w", msg, e)
}

// TODO remove and use errors.Join instead

// ErrErr wraps an error with another error.
func ErrErr(e1, e2 error) error {
	return fmt.Errorf("%w: %w", e1, e2)
}

// CtxPtr returns a pointer to the value of the context key.
// If the value is not of type *T, it returns false.
func CtxPtr[T any](ctx context.Context, key any) (*T, bool) {
	v := ctx.Value(key)
	if v == nil {
		return nil, false
	}

	vT, ok := v.(*T)
	if !ok {
		return nil, false
	}

	return vT, true
}

// CtxValue returns the value of the context key.
// If the value is not of type T, it returns false and the empty value.
// The empty value is required because the zero value of T could not be inferred.
func CtxValue[T any](ctx context.Context, key any, empty T) (T, bool) {
	v := ctx.Value(key)
	if v == nil {
		return empty, false
	}

	vT, ok := v.(T)
	if !ok {
		return empty, false
	}

	return vT, true
}
