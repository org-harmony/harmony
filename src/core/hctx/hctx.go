package hctx

import (
	"github.com/org-harmony/harmony/src/core/event"
	"github.com/org-harmony/harmony/src/core/persistence"
	"github.com/org-harmony/harmony/src/core/trace"
	"github.com/org-harmony/harmony/src/core/validation"
)

// AppCtx is the application context.
// It contains parts that are common to all parts of the application.
// It implements the trace.Logger and persistence.RepositoryProvider interfaces.
type AppCtx struct {
	Logger       trace.Logger
	Validator    validation.V
	Repositories persistence.RepositoryProvider
	EventManager event.Manager
}

// NewAppCtx constructs a new application context.
func NewAppCtx(l trace.Logger, v validation.V, repos persistence.RepositoryProvider, em event.Manager) *AppCtx {
	return &AppCtx{
		Logger:       l,
		Validator:    v,
		Repositories: repos,
		EventManager: em,
	}
}

// Debug implements the trace.Logger interface for the application context by forwarding the call to the logger.
func (c *AppCtx) Debug(mod, msg string, args ...any) {
	c.Logger.Debug(mod, msg, args...)
}

// Info implements the trace.Logger interface for the application context by forwarding the call to the logger.
func (c *AppCtx) Info(mod, msg string, args ...any) {
	c.Logger.Info(mod, msg, args...)
}

// Warn implements the trace.Logger interface for the application context by forwarding the call to the logger.
func (c *AppCtx) Warn(mod, msg string, args ...any) {
	c.Logger.Warn(mod, msg, args...)
}

// Error implements the trace.Logger interface for the application context by forwarding the call to the logger.
func (c *AppCtx) Error(mod, msg string, err error, args ...any) {
	c.Logger.Error(mod, msg, err, args...)
}

// Repository implements the persistence.RepositoryProvider interface for the application context by forwarding the call to the repository provider.
func (c *AppCtx) Repository(name string) (persistence.Repository, error) {
	return c.Repositories.Repository(name)
}

// RegisterRepository implements the persistence.RepositoryProvider interface for the application context by forwarding the call to the repository provider.
func (c *AppCtx) RegisterRepository(init func(db any) (persistence.Repository, error)) error {
	return c.Repositories.RegisterRepository(init)
}
