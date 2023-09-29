package harmony

import (
	"fmt"
	"github.com/pelletier/go-toml/v2"
	"os"
	"strings"
)

const ConfigPkg = "sys.config"

const configDir = "config"

// TODO write tests

// TODO add docs
func Config(filename string, config any) error {
	b, err := os.ReadFile(fmt.Sprintf("%s/%s.toml", configDir, filename))
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	err = toml.Unmarshal(b, &config)
	if err != nil {
		return fmt.Errorf("failed to unmarshal config file: %w", err)
	}

	// read contents from local config overwrite file
	b, err = os.ReadFile(fmt.Sprintf("%s/%s.local.toml", configDir, filename))
	if err != nil {
		return nil // ignore error
	}

	// overwrite config with local config
	err = toml.Unmarshal(b, &config)
	if err != nil {
		return fmt.Errorf("failed to unmarshal local config file: %w", err)
	}

	return nil
}

// TODO add docs
func LoadConfigToEnv(filename string) error {
	b, err := os.ReadFile(fmt.Sprintf("%s/%s.toml", configDir, filename))

	m := make(map[string]any)
	err = toml.Unmarshal(b, &m)
	if err != nil {
		return fmt.Errorf("failed to unmarshal config file: %w", err)
	}

	fm := makeEnvMap(m)
	if err := loadMapToEnv(fm); err != nil {
		return fmt.Errorf("failed to load config to env: %w", err)
	}

	return nil
}

// TODO add docs
func loadMapToEnv(m map[string]string) error {
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
