// Package herr provides error types and utility functions for HARMONY.
// This package should only be used for the most generic errors and error utilities.
// When necessary or possible, errors should be described domain specific as a part of the domain package.
package herr

import (
	"errors"
	"fmt"
)

var (
	SetEnvError         = errors.New("failed to write to env")
	NotImplementedError = errors.New("not implemented")
	ReadFileError       = errors.New("failed to read file")
)

// TODO move to config as var
type InvalidConfigError struct {
	Config any
	Prev   error
}

// TODO move to config as var
type ParseError struct {
	Parsable any
	Prev     error
}

func NewInvalidConfigError(config any, prev error) *InvalidConfigError {
	return &InvalidConfigError{
		Config: config,
		Prev:   prev,
	}
}

func (e *InvalidConfigError) Error() string {
	return fmt.Sprintf("invalid config %s: %s", e.Config, e.Prev.Error())
}

func NewParseError(parsable any, prev error) *ParseError {
	return &ParseError{
		Parsable: parsable,
		Prev:     prev,
	}
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("failed to parse %s, with: %s", e.Parsable, e.Prev.Error())
}
