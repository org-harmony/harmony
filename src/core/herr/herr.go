// Package herr provides error types and utility functions for HARMONY.
// This package should only be used for the most generic errors and error utilities.
// When necessary or possible, errors should be described domain specific as a part of the domain package.
package herr

import (
	"errors"
)

var (
	// ErrSetEnv is returned when an environment variable could not be set.
	ErrSetEnv = errors.New("failed to write to env")
	// ErrReadFile is returned when a file could not be read. This error may wrap underlying errors.
	ErrReadFile = errors.New("failed to read file")
)
