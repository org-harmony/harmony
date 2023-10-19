// Package herr provides error types and utility functions for HARMONY.
// This package should only be used for the most generic errors and error utilities.
// When necessary or possible, errors should be described domain specific as a part of the domain package.
package herr

import (
	"errors"
)

var (
	SetEnvError         = errors.New("failed to write to env")
	ReadFileError       = errors.New("failed to read file")
	NotImplementedError = errors.New("not implemented")
)
