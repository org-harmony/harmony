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
	BaseTemplateName        = "base"
	LandingPageTemplateName = "landing-page"
	ErrorTemplateName       = "error"
)

var (
	ErrTemplaterNotFound = fmt.Errorf("templater not found")
	ErrNoBaseTemplate    = fmt.Errorf("no base template")
	ErrCanNotLoad        = fmt.Errorf("template not loaded")
	ErrCanNotClone       = fmt.Errorf("template not cloned")
)

type UICfg struct {
	AssetsUri string        `toml:"assets_uri" validate:"required"`
	Templates *TemplatesCfg `toml:"templates" validate:"required"`
}

// TemplatesCfg is the config for the templates. BaseDir is parsed as a glob. Dir is used to load individual templates.
type TemplatesCfg struct {
	Dir     string `toml:"dir" validate:"required"`
	BaseDir string `toml:"base_dir" validate:"required"`
}

// BaseTemplateData is the base template data.
// It is a generic struct containing certain data and soon maybe some extra data that is common to all templates.
// Maybe this data structure will be removed in the future.
//
// Deprecated: This struct is deprecated and will be removed in the future.
type BaseTemplateData[T any] struct {
	Data T
}

// HTemplaterStore is a thread-safe store of Templater.
type HTemplaterStore struct {
	templaters map[string]Templater
	lock       sync.RWMutex
}

// HTemplater is an implementation of Templater. It contains and loads templates derived from a base template.
// Templates are cached in a map and loaded from the filesystem when not found in the map.
// HTemplater is safe for concurrent use by multiple goroutines.
type HTemplater struct {
	name      string
	dir       string
	templates map[string]*template.Template
	lock      sync.RWMutex
}

// TemplaterStore is a store of Templater. TemplaterStore is expected to be thread-safe.
type TemplaterStore interface {
	Templater(string) (Templater, error)
	Set(string, Templater)
}

// Templater allows retrieving templates by name and path extending from a base template.
// Templater is expected to be thread-safe.
type Templater interface {
	Template(name, path string) (*template.Template, error)
	Name() string
	Base() (*template.Template, error) // Base returns the base template all templates derive from.
}

// Deprecated: This function is deprecated and will be removed in the future.
func NewTemplateData[T any](data T) *BaseTemplateData[T] {
	return &BaseTemplateData[T]{
		Data: data,
	}
}

func NewTemplaterStore(t ...Templater) TemplaterStore {
	templaters := make(map[string]Templater)
	for _, t := range t {
		templaters[t.Name()] = t
	}

	return &HTemplaterStore{
		templaters: templaters,
	}
}

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

func (s *HTemplaterStore) Templater(name string) (Templater, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	t, ok := s.templaters[name]
	if !ok {
		return nil, ErrTemplaterNotFound
	}

	return t, nil
}

func (s *HTemplaterStore) Set(name string, t Templater) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.templaters[name] = t
}

// Template returns a template by name and path. The template file is loaded from the filesystem when not found in the map.
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

// SetupTemplaterStore sets up a TemplaterStore with the base, landing page and error templates.
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

func ErrorTemplate(ui *UICfg) (*template.Template, error) {
	landingPage, err := LandingPageTemplate(ui)
	if err != nil {
		return nil, err
	}

	return landingPage.New(ErrorTemplateName).ParseFiles(filepath.Join(ui.Templates.Dir, "error.go.html"))
}

func LandingPageTemplate(ui *UICfg) (*template.Template, error) {
	base, err := BaseTemplate(ui)
	if err != nil {
		return nil, err
	}

	return base.New(LandingPageTemplateName).ParseFiles(filepath.Join(ui.Templates.Dir, "landing-page.go.html"))
}

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

// templateFuncs returns a template.FuncMap containing basic template functions.
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
