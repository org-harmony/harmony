// Package trans provides generic translation utilities.
// Trans allows to translate user facing strings to other languages.
package trans

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/org-harmony/harmony/src/core/herr"
	"github.com/org-harmony/harmony/src/core/trace"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"text/template"
)

const (
	// Pkg is the package name used for logging.
	Pkg = "sys.trans"
	// TranslatorContextKey is the key used to store the translator in the request context.
	TranslatorContextKey = "translator"
	// LocaleSessionKey is the key used to store the locale in the request cookie.
	LocaleSessionKey = "harmony-app-locale"
)

var (
	// ErrLocaleNotFound is returned if a locale is not found.
	ErrLocaleNotFound = errors.New("locale not found")
	// ErrTranslatorNotFound is returned if a translator is not found.
	ErrTranslatorNotFound = errors.New("translator not found")
)

// Cfg is the translation configuration.
type Cfg struct {
	// Locales contains a list of locales.
	Locales map[string]*Locale `toml:"locales" validate:"required"`
	// TranslationsDir is the directory where the translation files are stored.
	TranslationsDir string `toml:"translations_dir" validate:"required"`
}

// Locale is a locale for a language
type Locale struct {
	// Path defines the path of the locale. E.g. de/de-DE/en/en-US.
	Path string `toml:"path" validate:"required"`
	// Name is the name of the locale. E.g. Deutsch, English.
	Name string `toml:"name" validate:"required"`
	// Default defines if this is the default locale.
	Default bool `toml:"default"`
}

// HTranslator is a translator for user facing strings.
// It uses templates to translate strings with arguments. Scheme: {{.argName}}.
// The template is stored in a map with the md5 hash of the string as key. This allows to cache the template per string.
// The HTranslator contains a map of translations.
// If a translation is not found in the map, the original string is returned.
// Therefore, this translator can be instantiated with any translation map, it works like a lookup table.
// The HTranslator is safe for concurrent use by multiple goroutines as long as the translations map is not changed.
// If it would ever be changed that the HTranslator needs to have changeable translations, the map needs to be protected by a mutex.
type HTranslator struct {
	translations map[string]string
	template     *template.Template
	tMu          sync.RWMutex
	logger       trace.Logger
	locale       *Locale
}

// HTranslatorProvider provides translators for various locales.
// HTranslatorProvider map is thread-safe as long as the map is not changed.
// If it would ever be changed that the HTranslatorProvider needs to have changeable translators, the map needs to be protected by a mutex.
type HTranslatorProvider struct {
	translators  map[string]Translator
	defaultTrans Translator
}

// HTranslatorOption is a function that sets an option on the HTranslator.
type HTranslatorOption func(*HTranslator)

// Translator allows translating of strings to other languages.
// It also contains a method to translate strings with arguments.
// Translator is required to be thread-safe for read-only operations after initialization.
type Translator interface {
	T(s string) string                  // T translates a string.
	Tf(s string, args ...string) string // Tf translates a string with arguments.
	Locale() *Locale                    // Locale returns the locale the translator translates to.
}

// TranslatorProvider provides translators for various locales.
// TranslatorProvider is required to be thread-safe for read-only operations after initialization.
type TranslatorProvider interface {
	Translator(locale string) (Translator, error) // Translator returns a translator for a locale.
	Default() (Translator, error)                 // Default returns the default translator.
}

// WithLogger sets the logger for the translator.
func WithLogger(logger trace.Logger) HTranslatorOption {
	return func(t *HTranslator) {
		t.logger = logger
	}
}

// WithTranslations sets the translations for the translator.
func WithTranslations(translations map[string]string) HTranslatorOption {
	return func(t *HTranslator) {
		t.translations = translations
	}
}

// ForLocale sets the locale for the translator.
func ForLocale(locale *Locale) HTranslatorOption {
	return func(t *HTranslator) {
		t.locale = locale
	}
}

// NewTranslator returns a new HTranslator covered by the Translator interface.
// A logger, translations and a locale can and should be passed in.
func NewTranslator(opts ...HTranslatorOption) Translator {
	translator := &HTranslator{
		translations: make(map[string]string),
	}

	for _, opt := range opts {
		opt(translator)
	}

	if translator.logger == nil {
		translator.logger = trace.NewLogger()
	}

	if translator.template == nil {
		translator.template = template.New("translator-base")
	}

	return translator
}

// T translates a string.
func (t *HTranslator) T(s string) string {
	transS, ok := t.translations[s]
	if !ok {
		return s
	}

	return transS
}

// Tf translates a string with arguments. The arguments are passed as key value pairs.
// The key is the name of the argument in the template, the value is the value of the argument.
// This parsing of args is done by the ArgsAsMap function.
func (t *HTranslator) Tf(s string, args ...string) string {
	var err error
	s = t.T(s)
	hash := md5.New()
	hash.Write([]byte(s))
	sh := string(hash.Sum(nil))

	t.tMu.RLock()
	transTemplate := t.template.Lookup(sh) // Lookup is thread safe
	t.tMu.RUnlock()
	if transTemplate == nil {
		t.logger.Debug(Pkg, "template not found, parsing", "hash", sh, "template", s)

		t.tMu.Lock()
		transTemplate, err = t.template.New(sh).Parse(s) // New is not thread safe, so we need to lock
		t.tMu.Unlock()

		if err != nil {
			t.logger.Error(Pkg, "error parsing template", err, "template", s)
			return s
		}
	}

	wr := &strings.Builder{}
	err = transTemplate.Execute(wr, ArgsAsMap(args...))
	if err != nil {
		t.logger.Error(Pkg, "error executing template", err, "template", s)
		return s
	}

	return wr.String()
}

// Locale returns the locale the translator translates to.
func (t *HTranslator) Locale() *Locale {
	return t.locale
}

// ArgsAsMap returns a map of arguments for HTranslator.Tf.
func ArgsAsMap(args ...string) map[string]string {
	params := make(map[string]string)
	for i := 0; i+1 < len(args); i += 2 {
		params[args[i]] = args[i+1]
	}

	return params
}

// FromCfg returns a translator provider from a configuration.
// It loads the translations from the translations directory.
func FromCfg(cfg *Cfg, logger trace.Logger) (TranslatorProvider, error) {
	lt := make([]Translator, 0, len(cfg.Locales))
	for _, locale := range cfg.Locales {
		translator, err := FromLocale(locale, cfg.TranslationsDir, logger)
		if err != nil {
			return nil, err
		}

		lt = append(lt, translator)
	}

	return NewTranslatorProvider(lt...), nil
}

// FromLocale returns a translator for a locale.
// It loads the translations from the translations directory.
func FromLocale(locale *Locale, translationsDir string, logger trace.Logger) (Translator, error) {
	translations, err := LoadTranslations(translationsDir, locale.Path)
	if err != nil {
		return nil, err
	}

	return NewTranslator(WithTranslations(translations), ForLocale(locale), WithLogger(logger)), nil
}

// LoadTranslations loads the translations from a file.
// The file content will be flattened to a map of strings, where the key is the path of the translation.
// Example:
//
//	{"a": {"B": "c"}} => {"a.B": "c"} => keep case-sensitivity
func LoadTranslations(translationsDir string, locale string) (map[string]string, error) {
	filePath := filepath.Join(translationsDir, fmt.Sprintf("%s.json", locale))
	bytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var translations map[string]any
	err = json.Unmarshal(bytes, &translations)
	if err != nil {
		return nil, errors.Join(herr.ErrReadFile, err)
	}

	return flattenTranslations(translations), nil
}

// NewTranslatorProvider returns a new translator provider using a list of translators.
// The first translator's locale is used as default locale as long as no other translator's locale is marked as default.
// It ignores translators with a nil locale.
func NewTranslatorProvider(lt ...Translator) TranslatorProvider {
	p := &HTranslatorProvider{
		translators: make(map[string]Translator),
	}

	for _, t := range lt {
		locale := t.Locale()
		if locale == nil {
			continue
		}

		p.translators[locale.Path] = t

		if locale.Default || p.defaultTrans == nil { // set the first translator as default => fallback
			p.defaultTrans = t
		}
	}

	return p
}

// Translator returns a translator for a locale.
func (t *HTranslatorProvider) Translator(locale string) (Translator, error) {
	if translator, ok := t.translators[locale]; ok {
		return translator, nil
	}

	return nil, ErrLocaleNotFound
}

// Default returns the default translator.
func (t *HTranslatorProvider) Default() (Translator, error) {
	if t.defaultTrans == nil {
		return nil, ErrLocaleNotFound
	}

	return t.defaultTrans, nil
}

// DefaultLocale returns the default locale.
func (cfg *Cfg) DefaultLocale() (*Locale, error) {
	for _, locale := range cfg.Locales {
		if locale.Default {
			return locale, nil
		}
	}

	return nil, ErrLocaleNotFound
}

// Locale returns a locale by name.
func (cfg *Cfg) Locale(name string) (*Locale, error) {
	for _, locale := range cfg.Locales {
		if locale.Name == name {
			return locale, nil
		}
	}

	return nil, ErrLocaleNotFound
}

// flattenTranslations flattens a map of translations to a map of strings.
// See usage and schema in LoadTranslations.
func flattenTranslations(m map[string]any) map[string]string {
	fm := make(map[string]string)

	for k, v := range m {
		vm, ok := v.(map[string]any)
		if ok {
			for fk, fv := range flattenTranslations(vm) {
				fm[fmt.Sprintf("%s.%s", k, fk)] = fv
			}
		} else {
			fm[k] = fmt.Sprint(v)
		}
	}

	return fm
}
