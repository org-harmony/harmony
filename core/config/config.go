// Package config provides configuration loading and optionally validation.
// Config files are expected to be in TOML format.
package config

import (
	"fmt"
	"github.com/go-playground/validator/v10"
	"github.com/org-harmony/harmony/core/herr"
	"github.com/pelletier/go-toml/v2"
	"os"
	"path"
	"strings"
)

const Pkg = "sys.config"

// Dir is the default directory for config files.
const Dir = "config"

type Options struct {
	dir       string
	filename  string
	fileExt   string
	validator *validator.Validate
}

type Option func(*Options)

// From sets the filename to read the config from.
// This Option is required.
func From(filename string) Option {
	return func(o *Options) {
		o.filename = filename
	}
}

// Validate sets a validator.Validate to validate the config struct.
// Without passing a non-nil validator.Validate the config will not be validated.
func Validate(v *validator.Validate) Option {
	return func(o *Options) {
		o.validator = v
	}
}

// FromDir sets the directory to read the config file from.
func FromDir(dir string) Option {
	return func(o *Options) {
		o.dir = dir
	}
}

// WithFileExt sets the file extension of the config file. (default is "toml")
func WithFileExt(ext string) Option {
	return func(o *Options) {
		o.fileExt = ext
	}
}

// defaultOptions returns a new instance of Options with default values.
func defaultOptions() *Options {
	return &Options{
		dir:       Dir,
		filename:  "config",
		fileExt:   "toml",
		validator: nil,
	}
}

// C reads a config file of type TOML and unmarshalls it into the given config struct.
// C will override the config struct with a local config file if it exists.
// The C function expects parameters through Option functions, default values are provided.
//
// If the Validate() Option is passed a validator.Validate the config struct will be validated.
//
// Using default options the config file is expected to be located in the config/ directory.
// Default example: config/config.toml
//
// The config file will be overwritten by a local config file if it exists.
// For the local config a "local" will be inserted between filename and file extension.
// Default example: config/config.local.toml
//
// Errors are returned if they occur on validating the options/config, reading or unmarshalling the config file.
func C(c any, opts ...Option) error {
	o := defaultOptions()
	for _, opt := range opts {
		opt(o)
	}

	fPath := path.Join(o.dir, fmt.Sprintf("%s.%s", o.filename, o.fileExt))
	b, err := os.ReadFile(fPath)
	if err != nil {
		return herr.NewReadFile(fPath, err)
	}

	flPath := path.Join(o.dir, fmt.Sprintf("%s.local.%s", o.filename, o.fileExt))
	bl, _ := os.ReadFile(flPath) // ignore error

	if err := parseConfig(c, b, bl); err != nil {
		return herr.NewParse(c, err)
	}

	if o.validator == nil {
		return nil
	}

	if err := o.validator.Struct(c); err != nil {
		return herr.NewInvalidConfig(c, err)
	}

	return nil
}

// parseConfig unmarshalls byte slices into the given config struct.
func parseConfig(config any, b ...[]byte) error {
	for _, v := range b {
		err := toml.Unmarshal(v, config)
		if err != nil {
			return fmt.Errorf("failed to unmarshal config from file: %w", err)
		}
	}

	return nil
}

// ToEnv reads a TOML config file and loads it into the environment.
// As with C, the options are passed through Option functions.
//
// The config file will be loaded recursively, meaning that nested maps will be flattened and joined with underscores.
// The values will be converted to strings and may be accessed through os.Getenv(<CONFIG_NAME>_<KEY>).
//
// The Validate() has no effect on this function.
// As of right now there is no validation implemented for env variables loaded from config files.
func ToEnv(opts ...Option) error {
	o := defaultOptions()
	for _, opt := range opts {
		opt(o)
	}

	fPath := path.Join(o.dir, fmt.Sprintf("%s.%s", o.filename, o.fileExt))
	b, err := os.ReadFile(fPath)
	if err != nil {
		return herr.NewReadFile(fPath, err)
	}

	m := make(map[string]any)
	err = toml.Unmarshal(b, &m)
	if err != nil {
		return herr.NewParse("config to env", err)
	}

	fm := makeEnvMap(m)
	if err := mapToEnv(fm); err != nil {
		return herr.ErrSetEnv
	}

	return nil
}

// mapToEnv loads the given map into the environment.
func mapToEnv(m map[string]string) error {
	for k, v := range m {
		if err := os.Setenv(k, v); err != nil {
			return fmt.Errorf("failed to set env variable: %w", err)
		}
	}

	return nil
}

// makeEnvMap recursively flattens a map of any type to a string map and changes the naming scheme to be compatible with
// environment variables. The capitalized keys of the map will be joined with underscores and the values will be
// converted to strings and flattened to a single level.
//
// Example:
//
//		{
//		  "foo": {
//		    "bar": "baz"
//	        "qux": {
//	          "kelvin": 273.15
//	        }
//		  }
//		}
//
// will be flattened and converted to:
//
//	{
//	  "FOO_BAR": "baz"
//	  "FOO_QUX_KELVIN": "273.15"
//	}
func makeEnvMap(m map[string]any) map[string]string {
	fm := make(map[string]string)

	for k, v := range m {
		vm, ok := v.(map[string]any)
		if ok {
			for fk, fv := range makeEnvMap(vm) {
				fm[fmt.Sprintf("%s_%s", strings.ToUpper(k), strings.ToUpper(fk))] = fv
			}
		} else {
			fm[strings.ToUpper(k)] = fmt.Sprint(v)
		}
	}

	return fm
}
