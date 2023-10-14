// Package trans provides generic translation utilities.
// Trans allows to translate user facing strings to other languages.
package trans

import (
	"crypto/md5"
	"github.com/org-harmony/harmony/core/trace"
	"strings"
	"text/template"
)

const Pkg = "sys.trans"

type HTranslator struct {
	translations map[string]string
	template     *template.Template
	logger       trace.Logger
}

type Translator interface {
	T(s string) string
	Tf(s string, args map[string]string) string
}

func NewTranslator() *HTranslator {
	return &HTranslator{
		translations: make(map[string]string),
	}
}

func (t *HTranslator) T(s string) string {
	transS, ok := t.translations[s]
	if !ok {
		return s
	}

	return transS
}

func (t *HTranslator) Tf(s string, args map[string]string) string {
	var err error
	s = t.T(s)
	hash := md5.New()
	hash.Write([]byte(s))
	sh := string(hash.Sum(nil))

	transTemplate := t.template.Lookup(sh)
	if transTemplate == nil {
		t.logger.Debug(Pkg, "template not found, parsing", "hash", sh, "template", s)
		transTemplate, err = t.template.New(sh).Parse(s)
		if err != nil {
			t.logger.Error(Pkg, "error parsing template", err, "template", s)
			return s
		}
	}

	wr := &strings.Builder{}
	err = transTemplate.Execute(wr, args)
	if err != nil {
		t.logger.Error(Pkg, "error executing template", err, "template", s)
		return s
	}

	return wr.String()
}
