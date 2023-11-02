package web

import (
	"github.com/stretchr/testify/assert"
	"html/template"
	"testing"
)

func TestTemplaterStoreOperations(t *testing.T) {
	ts := NewTemplaterStore()
	assert.NotNil(t, ts)

	mockTemplater := NewTemplater(template.New("mock"), "/mock/path")

	ts.Set("mock", mockTemplater)

	retrievedTemplater, err := ts.Templater("mock")
	assert.NoError(t, err)
	assert.Equal(t, mockTemplater, retrievedTemplater)
}

func TestTemplaterTemplateRetrieval(t *testing.T) {
	_, ts := setupMock(t)

	templater, err := ts.Templater(BaseTemplateName)
	assert.NoError(t, err)

	tmpl, err := templater.Template("partial", "partial.go.html")
	assert.NoError(t, err)
	assert.NotNil(t, tmpl)

	tmpl, err = templater.Template("not-found", "not-found.go.html")
	assert.ErrorIs(t, err, ErrCanNotLoad)

	_, err = ts.Templater("invalid")
	assert.ErrorIs(t, err, ErrTemplaterNotFound)
}
