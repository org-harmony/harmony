package web

import (
	"fmt"
	"github.com/org-harmony/harmony/core/trans"
	"html/template"
	"path/filepath"
	"sync"
)

const (
	// BaseTemplateName is the base template name.
	BaseTemplateName = "base"
	// LandingPageTemplateName is the landing page template name.
	LandingPageTemplateName = "landing-page"
)

var (
	ErrTemplaterNotFound = fmt.Errorf("templater not found")
	ErrNoBaseTemplate    = fmt.Errorf("no base template")
	ErrCanNotLoad        = fmt.Errorf("template not loaded")
	ErrCanNotClone       = fmt.Errorf("template not cloned")
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
func SetupTemplaterStore(ui *UICfg, t trans.Translator) (TemplaterStore, error) {
	base, err := BaseTemplate(ui, t)
	if err != nil {
		return nil, err
	}

	landingPage, err := LandingPageTemplate(ui, t)
	if err != nil {
		return nil, err
	}

	return NewTemplaterStore(NewTemplater(base, ui.Templates.Dir), NewTemplater(landingPage, ui.Templates.Dir)), nil
}

// LandingPageTemplate returns the landing page template.
func LandingPageTemplate(ui *UICfg, t trans.Translator) (*template.Template, error) {
	base, err := BaseTemplate(ui, t)
	if err != nil {
		return nil, err
	}

	return base.New(LandingPageTemplateName).ParseFiles(filepath.Join(ui.Templates.Dir, "landing-page.go.html"))
}

// BaseTemplate returns the base template.
func BaseTemplate(ui *UICfg, t trans.Translator) (*template.Template, error) {
	return template.
		New(BaseTemplateName).
		Funcs(templateFuncs(ui, t)).
		ParseGlob(filepath.Join(ui.Templates.BaseDir, "*.go.html"))
}

// templateFuncs returns a template.FuncMap for use in templates.
// It contains the functions that are expected to be used in "base" templates.
func templateFuncs(ui *UICfg, t trans.Translator) template.FuncMap {
	return template.FuncMap{
		"t": func(s string) string {
			return t.T(s)
		},
		"tf": func(s string, args ...string) string {
			return t.Tf(s, args...)
		},
		"asset": func(filename string) string {
			return filepath.Join(ui.AssetsUri, filename)
		},
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
	}
}
