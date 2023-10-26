package hctx

import (
	"github.com/go-playground/validator/v10"
	"github.com/org-harmony/harmony/src/core/persistence"
	"github.com/org-harmony/harmony/src/core/trace"
)

// AppCtx is the application context.
// It contains parts that are common to all parts of the application.
// While it contains the logger, validator and repositories it also implements the trace.Logger
// interface and persistence.RepositoryProvider interface itself.
type AppCtx struct {
	Logger       trace.Logger
	Validator    *validator.Validate
	Repositories persistence.RepositoryProvider
}

// NewAppCtx creates a new application context.
func NewAppCtx(l trace.Logger, v *validator.Validate, repos persistence.RepositoryProvider) *AppCtx {
	return &AppCtx{
		Logger:       l,
		Validator:    v,
		Repositories: repos,
	}
}

// Debug logs a debug message with the trace.Logger of the application context.
func (c *AppCtx) Debug(mod, msg string, args ...any) {
	c.Logger.Debug(mod, msg, args...)
}

// Info logs an info message with the trace.Logger of the application context.
func (c *AppCtx) Info(mod, msg string, args ...any) {
	c.Logger.Info(mod, msg, args...)
}

// Warn logs a warning message with the trace.Logger of the application context.
func (c *AppCtx) Warn(mod, msg string, args ...any) {
	c.Logger.Warn(mod, msg, args...)
}

// Error logs an error message with the trace.Logger of the application context.
func (c *AppCtx) Error(mod, msg string, err error, args ...any) {
	c.Logger.Error(mod, msg, err, args...)
}

// Repository returns a repository by name.
func (c *AppCtx) Repository(name string) (persistence.Repository, error) {
	return c.Repositories.Repository(name)
}

// RegisterRepository registers a repository.
func (c *AppCtx) RegisterRepository(init func(db any) (persistence.Repository, error)) error {
	return c.Repositories.RegisterRepository(init)
}
