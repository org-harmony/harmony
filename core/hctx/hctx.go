package hctx

import (
	"github.com/go-playground/validator/v10"
	"github.com/org-harmony/harmony/core/persistence"
	"github.com/org-harmony/harmony/core/trace"
)

type AppCtx struct {
	logger    trace.Logger
	repos     persistence.RepositoryProvider
	validator *validator.Validate
}

type AppContext interface {
	trace.Logger
	persistence.RepositoryProvider

	Validator() *validator.Validate
}

func NewAppContext(l trace.Logger, v *validator.Validate, repos persistence.RepositoryProvider) AppContext {
	return &AppCtx{
		logger:    l,
		validator: v,
		repos:     repos,
	}
}

func (c *AppCtx) Logger() trace.Logger {
	return c.logger
}

func (c *AppCtx) Validator() *validator.Validate {
	return c.validator
}

func (c *AppCtx) Debug(mod, msg string, args ...any) {
	c.logger.Debug(mod, msg, args...)
}

func (c *AppCtx) Info(mod, msg string, args ...any) {
	c.logger.Info(mod, msg, args...)
}

func (c *AppCtx) Warn(mod, msg string, args ...any) {
	c.logger.Warn(mod, msg, args...)
}

func (c *AppCtx) Error(mod, msg string, err error, args ...any) {
	c.logger.Error(mod, msg, err, args...)
}

func (c *AppCtx) Repository(name string) (persistence.Repository, error) {
	return c.repos.Repository(name)
}

func (c *AppCtx) RegisterRepository(init func(db any) (persistence.Repository, error)) error {
	return c.repos.RegisterRepository(init)
}
