package ctx

import (
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/org-harmony/harmony/core/trace"
)

type AppCtx struct {
	logger    trace.Logger
	validator *validator.Validate
	db        *pgxpool.Pool
}

type AppContext interface {
	Logger() trace.Logger
	Validator() *validator.Validate
	DB() *pgxpool.Pool
}

func NewAppContext(l trace.Logger, v *validator.Validate, db *pgxpool.Pool) AppContext {
	return &AppCtx{
		logger:    l,
		validator: v,
		db:        db,
	}
}

func (c *AppCtx) Logger() trace.Logger {
	return c.logger
}

func (c *AppCtx) Validator() *validator.Validate {
	return c.validator
}

func (c *AppCtx) DB() *pgxpool.Pool {
	return c.db
}
