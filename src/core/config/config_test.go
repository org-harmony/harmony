package config

import (
	"github.com/go-playground/validator/v10"
	"github.com/org-harmony/harmony/src/core/herr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
)

type MockConfig struct {
	Name string `env:"NAME" validate:"required"`
}

type MockEnvConfig struct {
	A string `env:"A"`
	B string `env:"B"`
	C struct {
		D string `env:"D"`
		E string `env:"E"`
	}
	F *struct {
		G string `env:"G"`
	}
}

func TestC(t *testing.T) {
	v := validator.New()

	t.Run("file not readable", func(t *testing.T) {
		config := &MockConfig{}
		err := C(config, From("not_readable"))
		assert.ErrorIs(t, err, herr.ErrReadFile)
	})

	t.Run("valid config file", func(t *testing.T) {
		tempDir := t.TempDir()
		path := filepath.Join(tempDir, "config.toml")
		err := os.WriteFile(path, []byte(`Name = "ValidName"`), 0644)
		require.NoError(t, err)

		config := &MockConfig{}
		err = C(config, FromDir(tempDir), Validate(v), DisableEnvOverwrite())
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
		assert.ErrorIs(t, err, ErrParse)
	})

	t.Run("invalid config", func(t *testing.T) {
		tempDir := t.TempDir()
		path := filepath.Join(tempDir, "invalid.toml")
		err := os.WriteFile(path, []byte(`Name = ""`), 0644)
		require.NoError(t, err)

		config := &MockConfig{}
		err = C(config, From("invalid"), FromDir(tempDir), Validate(v), DisableEnvOverwrite())
		assert.ErrorIs(t, err, ErrInvalidConfig)
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
		err = C(config, FromDir(tempDir), Validate(v), DisableEnvOverwrite())
		assert.NoError(t, err)
		assert.Equal(t, "LocalName", config.Name)
	})

	t.Run("overwrite with env", func(t *testing.T) {
		t.Cleanup(func() { os.Unsetenv("NAME") }) // ignore error on cleanup - test is over anyway

		tempDir := t.TempDir()
		path := filepath.Join(tempDir, "config.toml")
		err := os.WriteFile(path, []byte(`Name = "ValidName"`), 0644)
		require.NoError(t, err)

		err = os.Setenv("NAME", "EnvName")
		require.NoError(t, err)

		config := &MockConfig{}
		err = C(config, FromDir(tempDir), Validate(v))
		assert.NoError(t, err)
		assert.Equal(t, "EnvName", config.Name)
	})
}

func TestToEnv(t *testing.T) {
	t.Run("file not readable", func(t *testing.T) {
		err := ToEnv(From("not_readable"))
		assert.ErrorIs(t, err, herr.ErrReadFile)
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
		assert.ErrorIs(t, err, ErrParse)
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

func TestOverwriteWithEnv(t *testing.T) {
	t.Run("overwrite string fields", func(t *testing.T) {
		t.Cleanup(func() {
			os.Unsetenv("A")
			os.Unsetenv("B")
		}) // ignore error on cleanup - test is over anyway

		config := &MockEnvConfig{}
		err := os.Setenv("A", "valueA")
		require.NoError(t, err)
		err = os.Setenv("B", "valueB")
		require.NoError(t, err)

		err = overwriteWithEnv(config)
		assert.NoError(t, err)
		assert.Equal(t, "valueA", config.A)
		assert.Equal(t, "valueB", config.B)
	})

	t.Run("overwrite nested struct fields", func(t *testing.T) {
		t.Cleanup(func() {
			os.Unsetenv("D")
			os.Unsetenv("E")
		}) // ignore error on cleanup - test is over anyway

		config := &MockEnvConfig{}
		err := os.Setenv("D", "valueD")
		require.NoError(t, err)
		err = os.Setenv("E", "valueE")
		require.NoError(t, err)

		err = overwriteWithEnv(config)
		assert.NoError(t, err)
		assert.Equal(t, "valueD", config.C.D)
		assert.Equal(t, "valueE", config.C.E)
	})

	t.Run("overwrite pointer to struct fields", func(t *testing.T) {
		t.Cleanup(func() { os.Unsetenv("G") }) // ignore error on cleanup - test is over anyway

		config := &MockEnvConfig{F: &struct {
			G string `env:"G"`
		}{}}
		err := os.Setenv("G", "valueG")
		require.NoError(t, err)

		err = overwriteWithEnv(config)
		assert.NoError(t, err)
		assert.Equal(t, "valueG", config.F.G)
	})

	t.Run("ignore unsettable fields", func(t *testing.T) {
		config := MockEnvConfig{} // Non-pointer struct is not settable

		err := overwriteWithEnv(config)
		assert.NoError(t, err)
	})

	t.Run("ignore non-string fields", func(t *testing.T) {
		t.Cleanup(func() { os.Unsetenv("A") }) // ignore error on cleanup - test is over anyway

		config := &struct {
			A int `env:"A"`
		}{}
		err := os.Setenv("A", "123")
		require.NoError(t, err)

		err = overwriteWithEnv(config)
		assert.NoError(t, err)
	})

	t.Run("unexpected error", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				assert.Fail(t, "Function panicked")
			}
		}()

		err := overwriteWithEnv(nil) // Passing nil can be used to simulate unexpected error
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrUnexpectedEnvOverwrite)
	})
}
