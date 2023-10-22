package hctx

import (
	"github.com/go-playground/validator/v10"
	"github.com/org-harmony/harmony/core/persistence"
	"github.com/org-harmony/harmony/core/trace"
	"github.com/org-harmony/harmony/core/util"
)

// AppCtx is the application context.
// It is an implementation of the AppContext interface.
// It contains parts that are common to all parts of the application.
type AppCtx struct {
	logger    trace.Logger
	repos     persistence.RepositoryProvider
	validator *validator.Validate
}

// AppContext is the interface for the application context.
// It contains parts that are common to all parts of the application.
// AppContext is itself a trace.Logger and a persistence.RepositoryProvider.
type AppContext interface {
	trace.Logger
	persistence.RepositoryProvider

	Validator() *validator.Validate
}

// NewAppContext creates a new application context.
func NewAppContext(l trace.Logger, v *validator.Validate, repos persistence.RepositoryProvider) AppContext {
	return &AppCtx{
		logger:    l,
		validator: v,
		repos:     repos,
	}
}

// Validator returns the validator.
func (c *AppCtx) Validator() *validator.Validate {
	return c.validator
}

// Debug logs a debug message with the trace.Logger of the application context.
func (c *AppCtx) Debug(mod, msg string, args ...any) {
	c.logger.Debug(mod, msg, args...)
}

// Info logs an info message with the trace.Logger of the application context.
func (c *AppCtx) Info(mod, msg string, args ...any) {
	c.logger.Info(mod, msg, args...)
}

// Warn logs a warning message with the trace.Logger of the application context.
func (c *AppCtx) Warn(mod, msg string, args ...any) {
	c.logger.Warn(mod, msg, args...)
}

// Error logs an error message with the trace.Logger of the application context.
func (c *AppCtx) Error(mod, msg string, err error, args ...any) {
	c.logger.Error(mod, msg, err, args...)
}

// Repository returns a repository by name.
func (c *AppCtx) Repository(name string) (persistence.Repository, error) {
	return c.repos.Repository(name)
}

// RegisterRepository registers a repository.
func (c *AppCtx) RegisterRepository(init func(db any) (persistence.Repository, error)) error {
	return c.repos.RegisterRepository(init)
}

// SessionStore returns a session store by name and type.
// It uses util.UnwrapType which panics if the session store is not found or the type is wrong.
func SessionStore[V any](app AppContext, name string) persistence.SessionRepository[V] {
	return util.UnwrapType[persistence.SessionRepository[V]](app.Repository(name))
}
