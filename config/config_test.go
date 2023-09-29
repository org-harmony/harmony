package config

import (
	"github.com/go-playground/validator/v10"
	"github.com/org-harmony/harmony/herr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
)

type MockConfig struct {
	Name string `validate:"required"`
}

func TestC(t *testing.T) {
	v := validator.New()

	t.Run("file not readable", func(t *testing.T) {
		config := &MockConfig{}
		err := C(config, From("not_readable"))
		assert.IsType(t, &herr.ReadFile{}, err)
	})

	t.Run("valid config file", func(t *testing.T) {
		tempDir := t.TempDir()
		path := filepath.Join(tempDir, "config.toml")
		err := os.WriteFile(path, []byte(`Name = "ValidName"`), 0644)
		require.NoError(t, err)

		config := &MockConfig{}
		err = C(config, FromDir(tempDir), Validate(v))
		assert.NoError(t, err)
		assert.Equal(t, "ValidName", config.Name)
	})

	t.Run("unparsable config file", func(t *testing.T) {
		tempDir := t.TempDir()
		path := filepath.Join(tempDir, "unparsable.toml")
		err := os.WriteFile(path, []byte(`InvalidToml`), 0644)
		require.NoError(t, err)

		config := &MockConfig{}
		err = C(config, From("unparsable"), FromDir(tempDir), Validate(v))
		t.Logf("error: %v", err)
		assert.IsType(t, &herr.Parse{}, err)
	})

	t.Run("invalid config", func(t *testing.T) {
		tempDir := t.TempDir()
		path := filepath.Join(tempDir, "invalid.toml")
		err := os.WriteFile(path, []byte(`Name = ""`), 0644)
		require.NoError(t, err)

		config := &MockConfig{}
		err = C(config, From("invalid"), FromDir(tempDir), Validate(v))
		assert.IsType(t, err, &herr.InvalidConfig{})
	})

	t.Run("no validate", func(t *testing.T) {
		tempDir := t.TempDir()
		path := filepath.Join(tempDir, "invalid.toml")
		err := os.WriteFile(path, []byte(`Name = ""`), 0644)
		require.NoError(t, err)

		config := &MockConfig{}
		err = C(config, From("invalid"), FromDir(tempDir))
		assert.NoError(t, err)
	})

	t.Run("local config override", func(t *testing.T) {
		tempDir := t.TempDir()
		path := filepath.Join(tempDir, "config.toml")
		localPath := filepath.Join(tempDir, "config.local.toml")
		err := os.WriteFile(path, []byte(`Name = "OriginalName"`), 0644)
		require.NoError(t, err)

		err = os.WriteFile(localPath, []byte(`Name = "LocalName"`), 0644)
		require.NoError(t, err)

		config := &MockConfig{}
		err = C(config, FromDir(tempDir), Validate(v))
		assert.NoError(t, err)
		assert.Equal(t, "LocalName", config.Name)
	})
}

func TestToEnv(t *testing.T) {
	t.Run("file not readable", func(t *testing.T) {
		err := ToEnv(From("not_readable"))
		assert.IsType(t, &herr.ReadFile{}, err)
	})

	t.Run("valid config file", func(t *testing.T) {
		t.Cleanup(func() { os.Unsetenv("NAME") }) // ignore error on cleanup - test is over anyway

		tempDir := t.TempDir()
		path := filepath.Join(tempDir, "config.toml")
		err := os.WriteFile(path, []byte(`Name = "ValidName"`), 0644)
		require.NoError(t, err)

		err = ToEnv(FromDir(tempDir))
		assert.NoError(t, err)
		assert.Equal(t, "ValidName", os.Getenv("NAME"))
	})

	t.Run("unparsable config file", func(t *testing.T) {
		tempDir := t.TempDir()
		path := filepath.Join(tempDir, "unparsable.toml")
		err := os.WriteFile(path, []byte(`InvalidToml`), 0644)
		require.NoError(t, err)

		err = ToEnv(From("unparsable"), FromDir(tempDir))
		assert.IsType(t, &herr.Parse{}, err)
	})

	t.Run("env config naming scheme", func(t *testing.T) {
		t.Cleanup(func() { os.Unsetenv("APP_TEST_FOO_BAR") }) // ignore error on cleanup - test is over anyway

		tempDir := t.TempDir()
		path := filepath.Join(tempDir, "config.toml")
		err := os.WriteFile(path, []byte(`
[app.test.foo]
bar = "baz"
`), 0644)
		require.NoError(t, err)

		err = ToEnv(FromDir(tempDir))
		assert.NoError(t, err)
		assert.Equal(t, "baz", os.Getenv("APP_TEST_FOO_BAR"))
	})
}
