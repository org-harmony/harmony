package herr

import (
	"errors"
	"fmt"
)

var (
	ErrSetEnv = errors.New("failed to write to env")
)

type InvalidOptions struct {
	Options any
	Prev    error
}

func NewInvalidOptions(options any, prev error) *InvalidOptions {
	return &InvalidOptions{
		Options: options,
		Prev:    prev,
	}
}

func (e *InvalidOptions) Error() string {
	return fmt.Sprintf("invalid options %v", e.Options)
}

type InvalidConfig struct {
	Config any
	Prev   error
}

func NewInvalidConfig(config any, prev error) *InvalidConfig {
	return &InvalidConfig{
		Config: config,
		Prev:   prev,
	}
}

func (e *InvalidConfig) Error() string {
	return fmt.Sprintf("invalid config %v", e.Config)
}

type Parse struct {
	Parsable any
	Prev     error
}

func NewParse(parsable any, prev error) *Parse {
	return &Parse{
		Parsable: parsable,
		Prev:     prev,
	}
}

func (e *Parse) Error() string {
	return fmt.Sprintf("failed to parse %v", e.Parsable)
}

type ReadFile struct {
	Path string
	Prev error
}

func NewReadFile(path string, prev error) *ReadFile {
	return &ReadFile{
		Path: path,
		Prev: prev,
	}
}

func (e *ReadFile) Error() string {
	return fmt.Sprintf("failed to read file %s", e.Path)
}
