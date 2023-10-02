// Package trans provides generic translation utilities.
// Trans allows to translate user facing strings to other languages.
package trans

import (
	"context"
	"fmt"
)

type Translator interface {
	T(s string, ctx context.Context) string
	Tf(s string, ctx context.Context, args ...any) string
}

type StdTranslator struct {
	translations map[string]string
}

func NewTranslator() *StdTranslator {
	return &StdTranslator{
		translations: make(map[string]string),
	}
}

func (t *StdTranslator) T(s string, ctx context.Context) string {
	return s
}

func (t *StdTranslator) Tf(s string, ctx context.Context, args ...any) string {
	s = t.T(s, ctx)

	return fmt.Sprintf(s, args...)
}
