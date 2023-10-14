package trans

import (
	"github.com/org-harmony/harmony/core/trace"
	"github.com/stretchr/testify/assert"
	"testing"
	"text/template"
)

func TestHTranslator_Tf(t *testing.T) {
	translator := mockTranslator(t)
	args := map[string]string{
		"foo":  "Bär",
		"crux": "Fuchs",
	}

	t.Run("correct translation", func(t *testing.T) {
		result := translator.Tf("foo", args)
		assert.Equal(t, "füü", result)
	})

	t.Run("correct with args", func(t *testing.T) {
		result := translator.Tf("{{.foo}} is like a bar", args)
		assert.Equal(t, "Bär ist wie ein bar", result)

		result = translator.Tf("{{.foo}} is like a bar", args)
		assert.Equal(t, "Bär ist wie ein bar", result)

		result = translator.Tf("qux is like a {{.foo}} with a {{.crux}}", args)
		assert.Equal(t, "qux ist wie ein Bär mit einem Fuchs", result)
	})

	t.Run("without translations", func(t *testing.T) {
		result := translator.Tf("{{.foo}} wie {{.crux}}", args)
		assert.Equal(t, "Bär wie Fuchs", result)

		result = translator.Tf("foo ist ein qux", args)
		assert.Equal(t, "foo ist ein qux", result)
	})

	t.Run("missing args", func(t *testing.T) {
		result := translator.Tf("{{.foo}} is like a bar", map[string]string{})
		assert.Equal(t, "<no value> ist wie ein bar", result)

		result = translator.Tf("{{or .foo \"\"}} is like a bar", map[string]string{})
		assert.Equal(t, " is like a bar", result)
	})
}

func TestHTranslator_T(t *testing.T) {
	translator := mockTranslator(t)

	t.Run("correct translations", func(t *testing.T) {
		result := translator.T("foo")
		assert.Equal(t, "füü", result)

		result = translator.T("qux is a fux")
		assert.Equal(t, "qux ist ein fuchs", result)
	})

	t.Run("correct with args in t", func(t *testing.T) {
		result := translator.T("{{.foo}} is like a bar")
		assert.Equal(t, "{{.foo}} ist wie ein bar", result)
	})

	t.Run("without translation", func(t *testing.T) {
		result := translator.T("fux ist ein qux")
		assert.Equal(t, "fux ist ein qux", result)
	})
}

func TestParams(t *testing.T) {
	t.Run("correct params", func(t *testing.T) {
		params := Params("foo", "bar")
		assert.Equal(t, map[string]string{"foo": "bar"}, params)
	})

	t.Run("correct params with multiple", func(t *testing.T) {
		params := Params("foo", "bar", "qux", "fux")
		assert.Equal(t, map[string]string{"foo": "bar", "qux": "fux"}, params)
	})

	t.Run("correct params with multiple and odd", func(t *testing.T) {
		params := Params("foo", "bar", "qux", "fux", "baz")
		assert.Equal(t, map[string]string{"foo": "bar", "qux": "fux"}, params)
	})

	t.Run("correct params with multiple and odd", func(t *testing.T) {
		params := Params("foo", "bar", "qux", "fux", "baz", "quux")
		assert.Equal(t, map[string]string{"foo": "bar", "qux": "fux", "baz": "quux"}, params)
	})

	t.Run("no params", func(t *testing.T) {
		params := Params()
		assert.Equal(t, map[string]string{}, params)
	})
}

func mockTranslator(t *testing.T) Translator {
	return &HTranslator{
		translations: map[string]string{
			"foo":                    "füü",
			"{{.foo}} is like a bar": "{{.foo}} ist wie ein bar",
			"qux is like a {{.foo}} with a {{.crux}}": "qux ist wie ein {{.foo}} mit einem {{.crux}}",
			"qux is a fux": "qux ist ein fuchs",
		},
		template: template.New("translator-base"),
		logger:   trace.NewTestLogger(t),
	}
}
