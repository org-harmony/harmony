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
	// LocaleSessionKey is the key used to store the locale in the client's session cookie.
	LocaleSessionKey = "harmony-app-locale"
)

var (
	// ErrLocaleNotFound is returned when a locale is not found.
	ErrLocaleNotFound = errors.New("locale not found")
	// ErrTranslatorNotFound is returned when a translator is not found.
	ErrTranslatorNotFound = errors.New("translator not found")
)

// Cfg is the trans package's configuration. It is used to load the translations from the translations directory.
// Also, the supported (and default) locales are defined here.
type Cfg struct {
	Locales         map[string]*Locale `toml:"locales" hvalidate:"required"`
	TranslationsDir string             `toml:"translations_dir" hvalidate:"required"` // TranslationsDir is the directory where the translation files are stored. E.g. /translations.
}

// Locale is a locale entity as defined in the configuration.
type Locale struct {
	Path    string `toml:"path" hvalidate:"required"` // Path of the locale. E.g. de/de-DE/en/en-US.
	Name    string `toml:"name" hvalidate:"required"` // Name of the locale. E.g. Deutsch/Deutsch (Deutschland)/English/English (United States).
	Default bool   `toml:"default"`                   // Default declares the locale as default.
}

// HTranslator is a thread-safe translator using templates ({{.argName}}) for user-facing strings.
// Translations are cached with the md5 hash of the string as the key.
// It acts as a lookup table; if a translation is not found, the original string is returned.
// The translations map should not be modified concurrently to maintain thread safety.
type HTranslator struct {
	translations map[string]string
	template     *template.Template
	tMu          sync.RWMutex
	logger       trace.Logger
	locale       *Locale
}

// HTranslatorProvider provides translators for various locales in a thread-safe manner,
// assuming the translators map remains unmodified. If mutable translators are needed,
// a mutex should be added to protect the map.
type HTranslatorProvider struct {
	translators  map[string]Translator
	defaultTrans Translator
}

// HTranslatorOption is a functional option for the HTranslator.
type HTranslatorOption func(*HTranslator)

// Error is an interface for errors that can be translated.
type Error interface {
	Translate(Translator) string
}

// Translator allows translating of strings to other languages.
// It also contains a method to translate strings with arguments.
// Translator is required to be thread-safe for read-only operations after initialization.
type Translator interface {
	T(s string) string // T translates a string.
	// Tf translates a string with arguments. The arguments are passed as key value pairs.
	// Example:
	// 	Tf("Hello {{.name}}", "name", "John") => "Hello John"
	Tf(s string, args ...string) string
	Locale() *Locale // Locale returns the locale the translator translates to.
}

// TranslatorProvider provides translators for various locales.
// It is required to be thread-safe for read-only operations after initialization.
type TranslatorProvider interface {
	Translator(locale string) (Translator, error) // Translator returns a translator for a locale.
	Default() (Translator, error)                 // Default returns the default translator as a fallback.
}

// WithLogger sets the logger for the translator. This should be set to the default application logger.
// The logger could potentially be used to log errors that occur during translation e.g. a missing translation.
func WithLogger(logger trace.Logger) HTranslatorOption {
	return func(t *HTranslator) {
		t.logger = logger
	}
}

// WithTranslations sets the translations for the translator. This should be set for each translator as it is otherwise useless.
func WithTranslations(translations map[string]string) HTranslatorOption {
	return func(t *HTranslator) {
		t.translations = translations
	}
}

// ForLocale sets the locale for the translator. This should be set for each translator.
func ForLocale(locale *Locale) HTranslatorOption {
	return func(t *HTranslator) {
		t.locale = locale
	}
}

// NewTranslator returns a new translator with the given options.
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
// Example:
//
//	Tf("Hello {{.name}}", "name", "John") => "Hello John"
//
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

// ArgsAsMap converts a list of arguments to a map.
// Scheme: key1, value1, key2, value2, ...
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
// The file content will be flattened to a map of strings keeping case-sensitivity, where the key is the path of the translation.
// Example:
//
//	{"a": {"B": "c"}} => {"a.B": "c"}
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

// Default returns the default translator as a fallback.
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

// Locale returns the locale with the given name.
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
