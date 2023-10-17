package ctx

import (
	"github.com/go-playground/validator/v10"
	"github.com/org-harmony/harmony/core/trace"
)

type AppCtx struct {
	logger    trace.Logger
	validator *validator.Validate
}

type AppContext interface {
	Logger() trace.Logger
	Validator() *validator.Validate
}

func NewAppContext(l trace.Logger, v *validator.Validate) AppContext {
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
