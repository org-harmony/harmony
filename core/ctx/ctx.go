package ctx

import (
	"github.com/go-playground/validator/v10"
	"github.com/org-harmony/harmony/core/trace"
	"github.com/org-harmony/harmony/core/trans"
)

type AppCtx struct {
	logger     trace.Logger
	translator trans.Translator
	validator  *validator.Validate
}

type App interface {
	Logger() trace.Logger
	Translator() trans.Translator
	Validator() *validator.Validate
}

func NewApp(l trace.Logger, t trans.Translator, v *validator.Validate) App {
	return &AppCtx{
		logger:     l,
		translator: t,
		validator:  v,
	}
}

func (c *AppCtx) Logger() trace.Logger {
	return c.logger
}

func (c *AppCtx) Translator() trans.Translator {
	return c.translator
}

func (c *AppCtx) Validator() *validator.Validate {
	return c.validator
}
