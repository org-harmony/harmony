package hctx

import (
	"github.com/go-playground/validator/v10"
	"github.com/org-harmony/harmony/src/core/persistence"
	"github.com/org-harmony/harmony/src/core/trace"
	"github.com/org-harmony/harmony/src/core/util"
)

// AppCtx is the application context.
// It is an implementation of the AppContext interface.
// It contains parts that are common to all parts of the application.
type AppCtx struct {
	logger    trace.Logger
	repos     persistence.RepositoryProvider
	Validator *validator.Validate
}

// NewAppCtx creates a new application context.
func NewAppCtx(l trace.Logger, v *validator.Validate, repos persistence.RepositoryProvider) *AppCtx {
	return &AppCtx{
		logger:    l,
		Validator: v,
		repos:     repos,
	}
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
func SessionStore[V any](app *AppCtx, name string) persistence.SessionRepository[V] {
	return util.UnwrapType[persistence.SessionRepository[V]](app.Repository(name))
}
