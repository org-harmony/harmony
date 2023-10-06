package web

import (
	"context"
	"fmt"
	"github.com/org-harmony/harmony/core/trans"
	"html/template"
	"path/filepath"
)

const (
	// BaseTemplate is the base template name.
	BaseTemplate = "index"
	// LandingPageTemplate is the landing page template name.
	LandingPageTemplate = "landing-page"
)

// UICfg is the web packages UI configuration.
type UICfg struct {
	AssetsUri string        `toml:"assets_uri" validate:"required"`
	Templates *TemplatesCfg `toml:"templates" validate:"required"`
}

// TemplatesCfg is the web packages UI templates configuration.
type TemplatesCfg struct {
	Dir                 string `toml:"dir" validate:"required"`
	BaseDir             string `toml:"base_dir" validate:"required"`
	LandingPageFilepath string `toml:"landing_page_filepath" validate:"required"`
}

// DeriveTemplater is a base templater that reads the templates from the directory specified in the UICfg struct.
// All templates requested from the DeriveTemplater will derive from the base template.
// The DeriveTemplater will also load the landing page template at the same time the base template is loaded and saved.
type DeriveTemplater struct {
	ui    *UICfg
	trans trans.Translator
	from  *template.Template
}

type DeriveTemplaterOption func(*DeriveTemplater) error

// Templater allows to load a template for a given template path.
type Templater interface {
	Template(templatePath string) (*template.Template, error)
}

// FromBaseTemplate option specifies the base template to derive from.
func FromBaseTemplate() DeriveTemplaterOption {
	return func(t *DeriveTemplater) error {
		base, err := ctrlBaseTmpl(t.trans, t.ui)
		if err != nil {
			return fmt.Errorf("failed to load base template: %w", err)
		}
		t.from = base
		return nil
	}
}

// FromLandingPageTemplate option specifies the landing page template to derive from.
func FromLandingPageTemplate() DeriveTemplaterOption {
	return func(t *DeriveTemplater) error {
		lp, err := ctrlLpTmpl(t.trans, t.ui)
		if err != nil {
			return fmt.Errorf("failed to load landing page template: %w", err)
		}
		t.from = lp
		return nil
	}
}

// FromTemplate allows to provide a template to derive from.
func FromTemplate(tmpl *template.Template) DeriveTemplaterOption {
	return func(t *DeriveTemplater) error {
		t.from = tmpl
		return nil
	}
}

// NewTemplater returns a new DeriveTemplater.
func NewTemplater(ui *UICfg, trans trans.Translator, opts ...DeriveTemplaterOption) (*DeriveTemplater, error) {
	t := &DeriveTemplater{
		ui:    ui,
		trans: trans,
	}

	for _, opt := range opts {
		err := opt(t)
		if err != nil {
			return nil, err
		}
	}

	if t.from == nil {
		return nil, fmt.Errorf("no template to derive from provided")
	}

	return t, nil
}

// Template returns the template for the given template path.
// The DeriveTemplater will return the templates based on the configuration provided in the UICfg struct.
// If the templatePath is the LandingPageTemplate or BaseTemplate, the corresponding template will be returned.
// All other templates will be loaded from the templates directory in the UICfg struct.
func (t *DeriveTemplater) Template(path string) (*template.Template, error) {
	f, err := t.from.Clone()
	if err != nil {
		return nil, fmt.Errorf("failed to clone derived from template: %w", err)
	}

	if f.Name() == path {
		return f, nil
	}

	return f.New(BaseTemplate).ParseFiles(filepath.Join(t.ui.Templates.Dir, path))
}

// ctrlLpTmpl returns the landing page template.
func ctrlLpTmpl(t trans.Translator, ui *UICfg) (*template.Template, error) {
	base, err := ctrlBaseTmpl(t, ui)
	if err != nil {
		return nil, err
	}

	return base.New(LandingPageTemplate).ParseFiles(ui.Templates.LandingPageFilepath)
}

// ctrlBaseTmpl returns the base template for controllers.
func ctrlBaseTmpl(t trans.Translator, ui *UICfg) (*template.Template, error) {
	return template.
		New("index").
		Funcs(ctrlTmplUtilFunc(t, ui)).
		ParseGlob(filepath.Join(ui.Templates.BaseDir, "*.go.html"))
}

// ctrlTmplUtilFunc returns a template.FuncMap for use in templates.
// It contains the functions that are expected to be used in Controller templates.
func ctrlTmplUtilFunc(t trans.Translator, ui *UICfg) template.FuncMap {
	return template.FuncMap{
		"t": func(s string, ctx context.Context) string {
			return t.T(s, ctx)
		},
		"tf": func(s string, ctx context.Context, args ...any) string {
			return t.Tf(s, ctx, args...)
		},
		"html": func(s string) template.HTML {
			return template.HTML(s)
		},
		"asset": func(filename string) string {
			return filepath.Join(ui.AssetsUri, filename)
		},
	}
}
