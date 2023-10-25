package web

import (
	"context"
	"fmt"
	"github.com/org-harmony/harmony/src/core/trans"
	"github.com/org-harmony/harmony/src/core/util"
	"html/template"
	"path/filepath"
	"sync"
)

const (
	// BaseTemplateName is the base template name.
	BaseTemplateName = "base"
	// LandingPageTemplateName is the landing page template name.
	LandingPageTemplateName = "landing-page"
	// ErrorTemplateName is the error template name.
	ErrorTemplateName = "error"
)

var (
	// ErrTemplaterNotFound is returned when a Templater is not found.
	ErrTemplaterNotFound = fmt.Errorf("templater not found")
	// ErrNoBaseTemplate is returned when a base template is not found.
	ErrNoBaseTemplate = fmt.Errorf("no base template")
	// ErrCanNotLoad is returned when a template can not be loaded.
	ErrCanNotLoad = fmt.Errorf("template not loaded")
	// ErrCanNotClone is returned when a template can not be cloned.
	ErrCanNotClone = fmt.Errorf("template not cloned")
	// ErrTemplateNotTranslatable is returned when a template is not translatable.
	ErrTemplateNotTranslatable = fmt.Errorf("template not translatable")
)

// UICfg is the web packages UI configuration.
type UICfg struct {
	AssetsUri string        `toml:"assets_uri" validate:"required"`
	Templates *TemplatesCfg `toml:"templates" validate:"required"`
}

// TemplatesCfg is the web packages UI templates configuration.
type TemplatesCfg struct {
	Dir     string `toml:"dir" validate:"required"`
	BaseDir string `toml:"base_dir" validate:"required"`
}

// BaseTemplateData is the base template data.
// It is a generic struct containing certain data and soon maybe some extra data that is common to all templates.
// Maybe this data structure will be removed in the future.
type BaseTemplateData[T any] struct {
	Data T
}

// HTemplaterStore is a store of Templater. Templaters can each derive from a template.
// Each Templater is stored in a map and can be retrieved by name.
// E.g. a "base" Templater containing all templates deriving the base template.
// HTemplaterStore is safe for concurrent use by multiple goroutines.
type HTemplaterStore struct {
	templaters map[string]Templater
	lock       sync.RWMutex
}

// HTemplater is an implementation of Templater. It contains and loads templates derived from a base template.
// Templates are cached in a map and loaded from the filesystem when not found in the map.
// HTemplater is safe for concurrent use by multiple goroutines.
type HTemplater struct {
	name      string                        // name of the templater (usually the name of the base template)
	dir       string                        // dir the templates are loaded from when not found in the map
	templates map[string]*template.Template // map of templates cached
	lock      sync.RWMutex                  // lock for the map
}

// TemplaterStore is a store of Templater.
type TemplaterStore interface {
	Templater(string) (Templater, error) // Templater returns a Templater by name.
	Set(string, Templater)               // Set sets a Templater by name.
}

// Templater retrieves templates by name and path.
type Templater interface {
	Template(name string, path string) (*template.Template, error) // Template returns a template by name and path.
	Name() string                                                  // Name returns the name of the Templater.
	Base() (*template.Template, error)                             // Base returns the base template.
}

// NewTemplateData returns a new BaseTemplateData.
func NewTemplateData[T any](data T) *BaseTemplateData[T] {
	return &BaseTemplateData[T]{
		Data: data,
	}
}

// NewTemplaterStore returns a new TemplaterStore.
func NewTemplaterStore(t ...Templater) TemplaterStore {
	templaters := make(map[string]Templater)
	for _, t := range t {
		templaters[t.Name()] = t
	}

	return &HTemplaterStore{
		templaters: templaters,
	}
}

// NewTemplater returns a new Templater.
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

// Templater returns a Templater by name.
func (s *HTemplaterStore) Templater(name string) (Templater, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	t, ok := s.templaters[name]
	if !ok {
		return nil, ErrTemplaterNotFound
	}

	return t, nil
}

// Set sets a Templater by name.
func (s *HTemplaterStore) Set(name string, t Templater) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.templaters[name] = t
}

// Template returns a template by name and path.
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
			return nil, fmt.Errorf("%w: %w", ErrCanNotLoad, err)
		}

		t.lock.Lock()
		t.templates[path] = tmpl
		t.lock.Unlock()
	}

	tmpl, err := tmpl.Clone()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCanNotClone, err)
	}

	return tmpl, nil
}

// Name returns the name of the Templater.
func (t *HTemplater) Name() string {
	return t.name
}

// Base returns the base template that all templates within the Templater derive from.
func (t *HTemplater) Base() (*template.Template, error) {
	t.lock.RLock()
	defer t.lock.RUnlock()

	tmpl, ok := t.templates[t.name]
	if !ok {
		return nil, ErrNoBaseTemplate
	}

	b, err := tmpl.Clone()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCanNotClone, err)
	}

	return b, nil
}

// SetupTemplaterStore returns a new TemplaterStore.
func SetupTemplaterStore(ui *UICfg) (TemplaterStore, error) {
	base, err := BaseTemplate(ui)
	if err != nil {
		return nil, err
	}

	landingPage, err := LandingPageTemplate(ui)
	if err != nil {
		return nil, err
	}

	errorPage, err := ErrorTemplate(ui)
	if err != nil {
		return nil, err
	}

	return NewTemplaterStore(
		NewTemplater(base, ui.Templates.Dir),
		NewTemplater(landingPage, ui.Templates.Dir),
		NewTemplater(errorPage, ui.Templates.Dir),
	), nil
}

// ErrorTemplate returns the error template.
func ErrorTemplate(ui *UICfg) (*template.Template, error) {
	landingPage, err := LandingPageTemplate(ui)
	if err != nil {
		return nil, err
	}

	return landingPage.New(ErrorTemplateName).ParseFiles(filepath.Join(ui.Templates.Dir, "error.go.html"))
}

// LandingPageTemplate returns the landing page template.
func LandingPageTemplate(ui *UICfg) (*template.Template, error) {
	base, err := BaseTemplate(ui)
	if err != nil {
		return nil, err
	}

	return base.New(LandingPageTemplateName).ParseFiles(filepath.Join(ui.Templates.Dir, "landing-page.go.html"))
}

// BaseTemplate returns the base template.
func BaseTemplate(ui *UICfg) (*template.Template, error) {
	return template.
		New(BaseTemplateName).
		Funcs(templateFuncs(ui)).
		ParseGlob(filepath.Join(ui.Templates.BaseDir, "*.go.html"))
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
	})

	return nil
}

// templateFuncs returns a template.FuncMap for use in templates.
// It contains the functions that are expected to be used in "base" templates.
func templateFuncs(ui *UICfg) template.FuncMap {
	return template.FuncMap{
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
	}
}
