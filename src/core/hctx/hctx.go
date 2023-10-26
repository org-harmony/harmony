package hctx

import (
	"github.com/go-playground/validator/v10"
	"github.com/org-harmony/harmony/src/core/persistence"
	"github.com/org-harmony/harmony/src/core/trace"
)

// AppCtx is the application context.
// It contains parts that are common to all parts of the application.
// It implements the trace.Logger and persistence.RepositoryProvider interfaces.
type AppCtx struct {
	Logger       trace.Logger
	Validator    *validator.Validate
	Repositories persistence.RepositoryProvider
}

func NewAppCtx(l trace.Logger, v *validator.Validate, repos persistence.RepositoryProvider) *AppCtx {
	return &AppCtx{
		Logger:       l,
		Validator:    v,
		Repositories: repos,
	}
}

func (c *AppCtx) Debug(mod, msg string, args ...any) {
	c.Logger.Debug(mod, msg, args...)
}

func (c *AppCtx) Info(mod, msg string, args ...any) {
	c.Logger.Info(mod, msg, args...)
}

func (c *AppCtx) Warn(mod, msg string, args ...any) {
	c.Logger.Warn(mod, msg, args...)
}

func (c *AppCtx) Error(mod, msg string, err error, args ...any) {
	c.Logger.Error(mod, msg, err, args...)
}

func (c *AppCtx) Repository(name string) (persistence.Repository, error) {
	return c.Repositories.Repository(name)
}

func (c *AppCtx) RegisterRepository(init func(db any) (persistence.Repository, error)) error {
	return c.Repositories.RegisterRepository(init)
}
