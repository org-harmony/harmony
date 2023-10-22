// Package trans provides generic translation utilities.
// Trans allows to translate user facing strings to other languages.
package trans

import (
	"crypto/md5"
	"github.com/org-harmony/harmony/src/core/trace"
	"strings"
	"sync"
	"text/template"
)

const Pkg = "sys.trans"

// HTranslator is a translator for user facing strings.
// It uses templates to translate strings with arguments. Scheme: {{.argName}}.
// The template is stored in a map with the md5 hash of the string as key. This allows to cache the template per string.
// The HTranslator contains a map of translations.
// If a translation is not found in the map, the original string is returned.
// Therefore, this translator can be instantiated with any translation map, it works like a lookup table.
// The HTranslator is safe for concurrent use by multiple goroutines.
type HTranslator struct {
	translations map[string]string
	template     *template.Template
	tMu          sync.RWMutex
	logger       trace.Logger
}

// TranslatorOption is a function that sets an option on the HTranslator.
type TranslatorOption func(*HTranslator)

// Translator allows translating of strings to other languages.
// It also contains a method to translate strings with arguments.
type Translator interface {
	T(s string) string                  // T translates a string.
	Tf(s string, args ...string) string // Tf translates a string with arguments.
}

// WithLogger sets the logger for the translator.
func WithLogger(logger trace.Logger) TranslatorOption {
	return func(t *HTranslator) {
		t.logger = logger
	}
}

// NewTranslator returns a new HTranslator covered by the Translator interface.
func NewTranslator(opts ...TranslatorOption) Translator {
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
// It is safe for concurrent use by multiple goroutines.
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

// ArgsAsMap returns a map of arguments for HTranslator.Tf.
func ArgsAsMap(args ...string) map[string]string {
	params := make(map[string]string)
	for i := 0; i+1 < len(args); i += 2 {
		params[args[i]] = args[i+1]
	}

	return params
}
