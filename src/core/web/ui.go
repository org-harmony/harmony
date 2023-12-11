package web

import (
	"context"
	"errors"
	"fmt"
	"github.com/org-harmony/harmony/src/core/hctx"
	"github.com/org-harmony/harmony/src/core/trans"
	"github.com/org-harmony/harmony/src/core/util"
	"github.com/org-harmony/harmony/src/core/validation"
	"html/template"
	"path/filepath"
	"sync"
)

const (
	// BaseTemplateName is the name of the base template's Templater.
	// The base template is the template most other templates derive from.
	BaseTemplateName = "base"
	// PartialTemplateName is the name of the partial template's Templater.
	// It is used for HTMX partials of the entire page body.
	PartialTemplateName = "partial"
	// EmptyTemplateName is the name of the empty template's Templater.
	// It is used for partials that require no HTML surrounding the content e.g. forms.
	EmptyTemplateName = "empty"
	// WildcardViolation is the key for violations that do not have a field.
	// They are usually global violations on a FormData object and describe some error that is not related to a specific field.
	WildcardViolation = "*"
)

var (
	// ErrTemplaterNotFound is returned when a Templater is not found in the TemplaterStore.
	ErrTemplaterNotFound = fmt.Errorf("templater not found")
	// ErrNoBaseTemplate is returned when the base template of a Templater is not found.
	// This is not to be confused with the general base template.
	// If this error is returned the Templater does not have a template to derive its templates from.
	ErrNoBaseTemplate = fmt.Errorf("no base template")
	// ErrCanNotLoad is returned when a template can not be loaded.
	// This could happen if the template file does not exist or is not readable.
	ErrCanNotLoad = fmt.Errorf("template not loaded")
	// ErrCanNotClone is returned when a template can not be cloned.
	// Each time Templater.Template is called a new template is created from the Templater's base template through cloning `base.New(name)`.
	// If, for whatever reason, the template can not be cloned this error is returned.
	ErrCanNotClone = fmt.Errorf("template not cloned")
)

// UICfg is the config for the UI. It contains the URI to the assets and the TemplatesCfg.
type UICfg struct {
	AssetsUri string        `toml:"assets_uri" hvalidate:"required"`
	Templates *TemplatesCfg `toml:"templates" hvalidate:"required"`
}

// TemplatesCfg is the config for the templates. BaseDir is parsed as a glob. Dir is used to load individual templates.
type TemplatesCfg struct {
	Dir     string `toml:"dir" hvalidate:"required"`
	BaseDir string `toml:"base_dir" hvalidate:"required"`
}

// BaseTemplateData contains the data that is common to all templates and the specific template data.
// It can be viewed like a template context that is passed to the template containing information about what and how to render.
// It contains the template's specific data and extra data that can be used by the template.
type BaseTemplateData struct {
	Data       any
	HTMX       bool
	Navigation []NavItem
	Extra      map[string]any // Extra might be the user session or other data that is not part of the template data.
}

// FormData is the generic template data for forms. It contains any form object, a slice of success messages and a map of violations.
// The key of the violations map is the field name and the value is a slice of errors.
//
// Using FormData.ValidationErrorsForField all validation errors for a field can be retrieved as a slice of validation.Error.
// Alternatively, FormData.ViolationsForField can be used to retrieve the violations for a specific field.
// Also, FormData.FieldHasViolations can be used to check if a field has any violations.
type FormData[T any] struct {
	Form       T
	Violations map[string][]error
	Success    []string // Slice of success messages
}

// TemplateDataExtensions is a collection of template data extensions.
// Extensions are functions that are called during template rendering and can be used to add additional data to the template data.
// Using TemplateDataExtensions.Extensions all extensions can be retrieved as a slice of functions.
// This slice will be cached (until updated in any way) as it is expected that the extensions do not change during runtime.
//
// TemplateDataExtensions is safe for concurrent use by multiple goroutines.
type TemplateDataExtensions struct {
	extensions map[string]func(IO, *BaseTemplateData) error
	mu         sync.RWMutex
	ext        []func(IO, *BaseTemplateData) error
	extMu      sync.Mutex
}

// HTemplaterStore implements TemplaterStore by storing Templaters in a map and allowing thread-safe access to them.
// HTemplaterStore is safe for concurrent use by multiple goroutines and uses a sync.RWMutex to protect its map.
type HTemplaterStore struct {
	templaters map[string]Templater
	lock       sync.RWMutex
}

// HTemplater implements Templater by storing templates in a map and allowing thread-safe access to them.
// Templates are cached in a map and loaded from the filesystem when not found in the map.
// HTemplater is safe for concurrent use by multiple goroutines.
type HTemplater struct {
	name      string
	dir       string
	templates map[string]*template.Template // TODO replace with one base template and use template.Lookup + template.Clone
	lock      sync.RWMutex
}

// TemplaterStore is a store of Templater. TemplaterStore is safe for concurrent use by multiple goroutines.
type TemplaterStore interface {
	Templater(string) (Templater, error)
	Set(string, Templater)
}

// Templater is a collection of templates. It is used to load and cache templates across different parts of the application.
// The Templater allows loading templates that will then extend the Templater's base template.Template.
// A Templater will clone its base template.Template when loading a template.
//
// Templater is safe for concurrent use by multiple goroutines.
type Templater interface {
	Template(name, path string) (*template.Template, error)
	Name() string
	// Base returns the base template all templates in the Templater derive from.
	Base() (*template.Template, error)
	// JoinedTemplate returns a template that is a combination of the base template and the passed in templates.
	// [Name of Template]: base -> paths[0] -> paths[1] -> ...
	JoinedTemplate(name string, paths ...string) (*template.Template, error)
}

// NewBaseTemplateData returns an instance of BaseTemplateData with the passed in data.
// It will set the HTMX field based on if the request contains an HX-Request header.
// The extra data is initialized by executing the extension functions from web.Ctx.Extensions (TemplateDataExtensions).
// The navigation is built from the passed in web.Ctx.Navigation with web.IO.
func NewBaseTemplateData(appCtx *hctx.AppCtx, webCtx *Ctx, io IO, data any) (*BaseTemplateData, error) {
	baseData := &BaseTemplateData{
		Data:  data,
		HTMX:  io.Request().Header.Get("HX-Request") != "",
		Extra: make(map[string]any),
	}

	navigation, err := webCtx.Navigation.Build(io)
	if err != nil {
		return nil, err
	}
	baseData.Navigation = navigation

	for _, f := range webCtx.Extensions.Extensions() {
		err := f(io, baseData)
		if err != nil {
			return nil, err
		}
	}

	return baseData, nil
}

// NewFormData constructs a FormData with the passed in form and violations.
// The violations are set by calling FormData.ViolationsFromErrors.
// Thus, every error that is not a validation.Error will be added to the FormData's violations as a WildcardViolation.
// If the error is a validation.Error it will be added to the FormData's violations with the field as the key.
func NewFormData[T any](form T, success []string, errs ...error) *FormData[T] {
	formData := &FormData[T]{
		Form:       form,
		Violations: make(map[string][]error),
		Success:    success,
	}

	formData.ViolationsFromErrors(errs...)

	return formData
}

// NewTemplaterStore constructs a TemplaterStore with the passed in Templaters.
// The Templaters are stored in a map with their name as the key.
func NewTemplaterStore(t ...Templater) TemplaterStore {
	templaters := make(map[string]Templater)
	for _, t := range t {
		templaters[t.Name()] = t
	}

	return &HTemplaterStore{
		templaters: templaters,
	}
}

// NewTemplater constructs a Templater with the passed in base template and directory.
// The base template is the template all templates in the Templater derive from.
// If the base template is nil the function will panic.
//
// The directory is used to load templates from the filesystem when they are not found in the map.
func NewTemplater(base *template.Template, dir string) Templater {
	if base == nil {
		panic("base template is nil")
	}

	templates := make(map[string]*template.Template)
	name := base.Name()
	templates[name] = base

	return &HTemplater{
		name:      name,
		dir:       dir,
		templates: templates,
	}
}

// Templater returns a Templater by name from the TemplaterStore.
// The method will return ErrTemplaterNotFound if the Templater is not found.
func (s *HTemplaterStore) Templater(name string) (Templater, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	t, ok := s.templaters[name]
	if !ok {
		return nil, ErrTemplaterNotFound
	}

	return t, nil
}

// Set sets a Templater in the TemplaterStore. The Templater.Name() will be ignored. Instead the passed in name will be used.
// Example:
//
//	templaterStore.Set(templater.Name(), templater)
//	templaterStore.Templater("TEMPLATER_NAME_HERE", templater) // this is the same as previous
//	templaterStore.Templater(templater.Name()) // returns templater
func (s *HTemplaterStore) Set(name string, t Templater) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.templaters[name] = t
}

// Template returns a template by template name and path in the filesystem.
// The path is relative to the HTemplater's directory. Usually this should be templates/
//
// Template on the HTemplater first looks in its map for the template. If it is not found it will load the template from the filesystem.
// If the template does not exist in the cache the Template method will parse it from a file deriving from the HTemplater's base template.
// If no base template exists on the Templater ErrNoBaseTemplate is returned. If the template can not be loaded ErrCanNotLoad is returned.
// After that the template is cached in the HTemplater's map. Then, the template is cloned and returned.
// If the template can not be cloned ErrCanNotClone is returned.
//
// Cloning the template upon each request to Template prevents the state of the initially loaded template.Template from changing.
func (t *HTemplater) Template(name string, path string) (*template.Template, error) {
	t.lock.RLock()
	tmpl, ok := t.templates[path]
	t.lock.RUnlock()
	if !ok {
		base, err := t.Base()
		if err != nil {
			return nil, ErrNoBaseTemplate
		}

		tmpl, err = base.New(name).ParseFiles(filepath.Join(t.dir, path))
		if err != nil {
			return nil, errors.Join(ErrCanNotLoad, err)
		}

		t.lock.Lock()
		t.templates[path] = tmpl
		t.lock.Unlock()
	}

	tmpl, err := tmpl.Clone()
	if err != nil {
		return nil, errors.Join(ErrCanNotClone, err)
	}

	return tmpl, nil
}

// Name returns the name of the HTemplater.
func (t *HTemplater) Name() string {
	return t.name
}

// Base returns the base template that all templates within the Templater derive from.
// If the base template is not found ErrNoBaseTemplate is returned.
// Base will clone the base template and return the clone.
// If the clone fails ErrCanNotClone is returned.
func (t *HTemplater) Base() (*template.Template, error) {
	t.lock.RLock()
	defer t.lock.RUnlock()

	tmpl, ok := t.templates[t.name]
	if !ok {
		return nil, ErrNoBaseTemplate
	}

	b, err := tmpl.Clone()
	if err != nil {
		return nil, errors.Join(ErrCanNotClone, err)
	}

	return b, nil
}

// JoinedTemplate implements Templater.JoinedTemplate on HTemplater by joining the base template and the passed in templates.
// The templates are joined in the order they are passed in. The last template is the template that is returned.
// The joined templates are cached in the HTemplater's map and returned as cloned templates to prevent the state of the initially loaded template.Template from changing.
// If the base template is not found ErrNoBaseTemplate is returned. If the template can not be loaded ErrCanNotLoad is returned.
// If the template can not be cloned ErrCanNotClone is returned.
func (t *HTemplater) JoinedTemplate(name string, paths ...string) (*template.Template, error) {
	if len(paths) < 1 {
		return nil, fmt.Errorf("at least one template path must be passed in")
	}

	t.lock.RLock()
	tmpl, ok := t.templates[name]
	t.lock.RUnlock()

	if ok {
		clone, err := tmpl.Clone()
		if err != nil {
			return nil, errors.Join(ErrCanNotClone, err)
		}

		return clone, nil
	}

	base, err := t.Base()
	if err != nil {
		return nil, err
	}

	tmpl, err = base.New(name).ParseFiles(filepath.Join(t.dir, paths[0]))
	if err != nil {
		return nil, errors.Join(ErrCanNotLoad, err)
	}

	for _, path := range paths[1:] {
		tmpl, err = tmpl.ParseFiles(filepath.Join(t.dir, path))
		if err != nil {
			return nil, errors.Join(ErrCanNotLoad, err)
		}
	}

	t.lock.Lock()
	t.templates[name] = tmpl
	t.lock.Unlock()

	clone, err := tmpl.Clone()
	if err != nil {
		return nil, errors.Join(ErrCanNotClone, err)
	}

	return clone, nil
}

// NewExtensions constructs a TemplateDataExtensions collection with an empty map of extensions.
func NewExtensions() *TemplateDataExtensions {
	return &TemplateDataExtensions{
		extensions: make(map[string]func(IO, *BaseTemplateData) error),
	}
}

// Add adds a new extension to the TemplateDataExtensions collection by name.
// The extension slice cache that will be used for TemplateDataExtensions.Extensions is invalidated.
func (e *TemplateDataExtensions) Add(name string, f func(IO, *BaseTemplateData) error) {
	e.mu.Lock()
	e.extensions[name] = f
	e.mu.Unlock()

	e.extMu.Lock()
	e.ext = nil
	e.extMu.Unlock()
}

// Extensions returns a slice of all extensions in the TemplateDataExtensions collection.
// The slice is cached and will not be recalculated until the TemplateDataExtensions collection is modified.
func (e *TemplateDataExtensions) Extensions() []func(IO, *BaseTemplateData) error {
	e.extMu.Lock()
	defer e.extMu.Unlock()

	if e.ext != nil {
		return e.ext
	}

	e.ext = make([]func(IO, *BaseTemplateData) error, 0, len(e.extensions))

	e.mu.RLock()
	extensions := e.extensions
	e.mu.RUnlock()
	for _, f := range extensions {
		e.ext = append(e.ext, f)
	}

	return e.ext
}

// Successes returns the success messages of the FormData. They are usually displayed after a successful form submission.
// TODO implement toast via events and HTMX
func (d *FormData[T]) Successes() []string {
	return d.Success
}

// ViolationsFromErrors adds the passed in non-nil errors to the FormData's violations. Nil errors are ignored.
// If the error is not a validation.Error it will be added to the FormData's violations with the WildcardViolation as the key.
// If the error is a validation.Error it will be added to the FormData's violations with the field as the key.
//
// Thus, only validation.Error will be treated field specific.
func (d *FormData[T]) ViolationsFromErrors(errs ...error) {
	for _, err := range errs {
		if err == nil {
			continue
		}

		var v validation.Error
		if !errors.As(err, &v) {
			d.Violations[WildcardViolation] = append(d.Violations[WildcardViolation], err)
			continue
		}

		d.Violations[v.Field] = append(d.Violations[v.Field], err)
	}
}

// WildcardViolations returns all violations that do not have a field.
// They are usually global violations on a FormData object and describe some error that is not related to a specific field.
func (d *FormData[T]) WildcardViolations() []error {
	return d.Violations[WildcardViolation]
}

// ViolationsForField returns all violations for the passed in field. This could be any error but is most likely a validation.Error.
// Use FormData.ValidationErrorsForField to retrieve all validation errors for a field.
func (d *FormData[T]) ViolationsForField(field string) []error {
	return d.Violations[field]
}

// ValidationErrorsForField returns all validation errors for the passed in field.
// If the error is not a validation.Error it will be ignored.
func (d *FormData[T]) ValidationErrorsForField(field string) []validation.Error {
	var errs []validation.Error
	for _, err := range d.Violations[field] {
		var v validation.Error
		if errors.As(err, &v) {
			errs = append(errs, v)
		}
	}

	return errs
}

// AllValidationErrors returns all validation errors for all fields.
// If you want to display all errors to the user use AllViolations instead.
func (d *FormData[T]) AllValidationErrors() []validation.Error {
	var errs []validation.Error
	for _, fieldErrs := range d.Violations {
		for _, err := range fieldErrs {
			var v validation.Error
			if errors.As(err, &v) {
				errs = append(errs, v)
			}
		}
	}

	return errs
}

// AllViolations returns all errors for all fields. They can then be used to display all errors to the user.
//
// Important: AllViolations does *not* return any validation errors. Use AllValidationErrors for that.
// ValidationErrors are filtered out because they are usually displayed to the user in a different way than other errors.
func (d *FormData[T]) AllViolations() []error {
	var errs []error
	for _, fieldErrs := range d.Violations {
		for _, err := range fieldErrs {
			if errors.Is(err, &validation.Error{}) {
				continue
			}

			errs = append(errs, err)
		}
	}

	return errs
}

// FieldHasViolations returns true if the passed in field has any violations.
// Be careful when using FieldHasViolations but only displaying the validation errors.
// There might be other errors that are not validation errors.
func (d *FormData[T]) FieldHasViolations(field string) bool {
	return len(d.Violations[field]) > 0
}

// Valid returns true if the FormData has no violations.
func (d *FormData[T]) Valid() bool {
	return len(d.Violations) < 1
}

// SetupTemplaterStore sets up a TemplaterStore with the base, partial and empty templates.
func SetupTemplaterStore(ui *UICfg) (TemplaterStore, error) {
	base, err := BaseTemplate(ui)
	if err != nil {
		return nil, err
	}

	partialPage, err := PartialTemplate(ui)
	if err != nil {
		return nil, err
	}

	emptyPage, err := EmptyTemplate(ui)
	if err != nil {
		return nil, err
	}

	return NewTemplaterStore(
		NewTemplater(base, ui.Templates.Dir),
		NewTemplater(partialPage, ui.Templates.Dir),
		NewTemplater(emptyPage, ui.Templates.Dir),
	), nil
}

// BaseTemplate returns the base template from the passed in UICfg.
func BaseTemplate(ui *UICfg) (*template.Template, error) {
	return template.
		New(BaseTemplateName).
		Funcs(templateFuncs(ui)).
		ParseGlob(filepath.Join(ui.Templates.BaseDir, "*.go.html"))
}

// PartialTemplate returns the partial template from the passed in UICfg.
// It extends the base template and makes it partial to be used with HTMX.
func PartialTemplate(ui *UICfg) (*template.Template, error) {
	base, err := BaseTemplate(ui)
	if err != nil {
		return nil, err
	}

	return base.New(PartialTemplateName).
		Funcs(templateFuncs(ui)).
		ParseFiles(filepath.Join(ui.Templates.Dir, "partial.go.html"))
}

// EmptyTemplate returns the empty template from the passed in UICfg.
// It contains only the most essential template blocks to be used as an empty template without any surrounding HTML.
func EmptyTemplate(ui *UICfg) (*template.Template, error) {
	return template.New(EmptyTemplateName).
		Funcs(templateFuncs(ui)).
		ParseFiles(filepath.Join(ui.Templates.Dir, "empty.go.html"))
}

// makeTemplateTranslatable overrides the translation functions t/tf on the template using the translator from the context.
// This function is intended to be used with the trans.Middleware.
func makeTemplateTranslatable(ctx context.Context, t *template.Template) error {
	translator, ok := util.CtxValue[trans.Translator](ctx, trans.TranslatorContextKey)
	if !ok {
		return trans.ErrTranslatorNotFound
	}

	t.Funcs(template.FuncMap{
		"t": func(s string) string {
			return translator.T(s)
		},
		"tf": func(s string, args ...string) string {
			return translator.Tf(s, args...)
		},
		"tryTranslate": func(t any) string {
			if s, ok := t.(string); ok {
				return translator.T(s)
			}

			if t, ok := t.(trans.Translatable); ok {
				return t.Translate(translator)
			}

			if e, ok := t.(error); ok {
				return translator.T(e.Error())
			}

			return translator.T(fmt.Sprintf("%s", t))
		},
	})

	return nil
}

// templateFuncs returns a template.FuncMap containing basic template functions.
func templateFuncs(ui *UICfg) template.FuncMap {
	return template.FuncMap{
		"add": func(a, b int) int {
			return a + b
		},
		"asset": func(filename string) string {
			return filepath.Join(ui.AssetsUri, filename)
		},
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
		"t": func(s string) string {
			return s
		},
		"tf": func(s string, args ...string) string {
			return s
		},
		"tryTranslate": func(t any) string {
			if s, ok := t.(string); ok {
				return s
			}

			return fmt.Sprintf("%s", t)
		},
	}
}
