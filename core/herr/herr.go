// Package herr provides error types and utility functions for HARMONY.
// This package should only be used for the most generic errors and error utilities.
// When necessary or possible, errors should be described domain specific as a part of the domain package.
package herr

import (
	"errors"
	"fmt"
)

var (
	ErrSetEnv = errors.New("failed to write to env")
)

type InvalidConfigError struct {
	Config any
	Prev   error
}

type ParseError struct {
	Parsable any
	Prev     error
}

type ReadFileError struct {
	Path string
	Prev error
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

func NewReadFileError(path string, prev error) *ReadFileError {
	return &ReadFileError{
		Path: path,
		Prev: prev,
	}
}

func (e *ReadFileError) Error() string {
	return fmt.Sprintf("failed to read file %s", e.Path)
}
