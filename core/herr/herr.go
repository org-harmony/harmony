// Package herr provides error types and utility functions for HARMONY.
// This package should only be used for the most generic errors and error utilities.
// When necessary or possible, errors should be described domain specific as a part of the domain package.
package herr

import (
	"errors"
)

var (
	ErrSetEnv   = errors.New("failed to write to env")
	ErrReadFile = errors.New("failed to read file")
)
