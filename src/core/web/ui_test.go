package web

import (
	"errors"
	"github.com/org-harmony/harmony/src/core/hctx"
	"github.com/org-harmony/harmony/src/core/validation"
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

func TestFormDataWithValidationErrors(t *testing.T) {
	form := struct{}{}
	validationErr := validation.Error{Msg: "Invalid", Struct: "Form", Field: "Name", Path: "Form.Name"}

	formData := NewFormData(form, nil, validationErr)

	assert.Len(t, formData.Violations["Name"], 1)
	assert.Nil(t, formData.WildcardViolations())

	assert.Equal(t, validationErr, formData.ViolationsForField("Name")[0])
	assert.Equal(t, validationErr.Error(), formData.ValidationErrors("Name")[0].Error())
}

func TestNewFormData(t *testing.T) {
	type TestForm struct {
		Name string
	}

	form := TestForm{Name: "Test"}
	successMessages := []string{"Operation successful"}
	validationErr := validation.Error{Msg: "Invalid", Struct: "TestForm", Field: "Name", Path: "TestForm.Name"}

	formData := NewFormData(form, successMessages, validationErr)

	assert.Equal(t, form, formData.Form)
	assert.Equal(t, successMessages, formData.Successes())
	assert.Contains(t, formData.Violations, "Name")
	assert.Len(t, formData.Violations["Name"], 1)
	assert.Equal(t, validationErr, formData.Violations["Name"][0])
}

func TestNewBaseTemplateData(t *testing.T) {
	appCtx := &hctx.AppCtx{}
	webCtx := &Ctx{Navigation: NewNavigation(), Extensions: NewExtensions()}
	io := newMockIO("/")

	webCtx.Navigation.Add("home", NavItem{Name: "Home", URL: "/", Position: 1})
	webCtx.Navigation.Add("about", NavItem{Name: "About", URL: "/about", Position: 2})

	webCtx.Extensions.Add("testExtension", func(io IO, data *BaseTemplateData) error {
		data.Extra["test"] = "extra data"
		return nil
	})

	data := "sample data"
	baseData, err := NewBaseTemplateData(appCtx, webCtx, io, data)

	assert.NoError(t, err)
	assert.Equal(t, data, baseData.Data)
	assert.NotEmpty(t, baseData.Navigation)
	assert.Equal(t, "extra data", baseData.Extra["test"])
}

func TestNewBaseTemplateDataHTMXHeader(t *testing.T) {
	io := newMockIO("/")
	io.Request().Header.Set("HX-Request", "true")

	baseData, err := NewBaseTemplateData(nil, &Ctx{Navigation: NewNavigation(), Extensions: NewExtensions()}, io, nil)

	assert.NoError(t, err)
	assert.True(t, baseData.HTMX)
}

func TestFormDataWithSuccessAndErrors(t *testing.T) {
	form := struct{}{}
	successMessages := []string{"Operation successful"}
	validationErr := validation.Error{Msg: "Invalid", Struct: "TestForm", Field: "Name", Path: "TestForm.Name"}
	genericErr := errors.New("generic error")

	formData := NewFormData(form, successMessages, validationErr, genericErr)

	assert.Equal(t, successMessages, formData.Successes())
	assert.Contains(t, formData.Violations, "Name")
	assert.Contains(t, formData.Violations[WildcardViolation], genericErr)
	assert.Len(t, formData.Violations["Name"], 1)
	assert.Len(t, formData.WildcardViolations(), 1)
	assert.Equal(t, validationErr, formData.ValidationErrors("Name")[0])
	assert.True(t, formData.HasViolations("Name"))
	assert.False(t, formData.HasViolations("NonExistentField"))
}

func TestTemplateDataExtensionsLifecycle(t *testing.T) {
	extensions := NewExtensions()
	assert.Empty(t, extensions.Extensions())

	extensions.Add("testExtension", func(io IO, data *BaseTemplateData) error {
		data.Extra["test"] = "extra data"
		return nil
	})

	exts := extensions.Extensions()
	assert.Len(t, exts, 1)

	baseData := &BaseTemplateData{
		Extra: make(map[string]any),
	}
	io := newMockIO("/")
	err := exts[0](io, baseData)
	assert.NoError(t, err)
	assert.Equal(t, "extra data", baseData.Extra["test"])

	extensions.Add("anotherTestExtension", func(io IO, data *BaseTemplateData) error {
		data.Extra["anotherTest"] = "another extra data"
		return nil
	})
	exts = extensions.Extensions()
	assert.Len(t, exts, 2)
}
