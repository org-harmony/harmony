package ctx

import (
	"github.com/go-playground/validator/v10"
	"github.com/org-harmony/harmony/core/trace"
)

type AppCtx struct {
	logger    trace.Logger
	validator *validator.Validate
}

type App interface {
	Logger() trace.Logger
	Validator() *validator.Validate
}

func NewApp(l trace.Logger, v *validator.Validate) App {
	return &AppCtx{
		logger:    l,
		validator: v,
	}
}

func (c *AppCtx) Logger() trace.Logger {
	return c.logger
}

func (c *AppCtx) Validator() *validator.Validate {
	return c.validator
}
