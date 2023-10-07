// Package trans provides generic translation utilities.
// Trans allows to translate user facing strings to other languages.
package trans

import (
	"context"
	"fmt"
)

type HTranslator struct {
	translations map[string]string
}

type Translator interface {
	T(s string, ctx context.Context) string
	Tf(s string, ctx context.Context, args ...any) string
}

func NewTranslator() *HTranslator {
	return &HTranslator{
		translations: make(map[string]string),
	}
}

func (t *HTranslator) T(s string, ctx context.Context) string {
	return s
}

func (t *HTranslator) Tf(s string, ctx context.Context, args ...any) string {
	s = t.T(s, ctx)

	return fmt.Sprintf(s, args...)
}
