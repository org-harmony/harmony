package core

import (
	"context"

	"github.com/org-harmony/harmony/trace"
)

type Module interface {
	ID() string
	Setup(args *ModLifecycleArgs, ctx context.Context) (context.Context, error)
	Start(args *ModLifecycleArgs, ctx context.Context) (context.Context, error)
	Stop(args *ModLifecycleArgs, ctx context.Context) error
}

type ModuleLoader struct {
	modules []Module
}

type ModLifecycleArgs struct {
	Logger trace.Logger
}

func NewManager() *ModuleLoader {
	return &ModuleLoader{}
}

func (m *ModuleLoader) Register(modules ...Module) {
	m.modules = append(m.modules, modules...)
}

func (m *ModuleLoader) Setup(args *ModLifecycleArgs, ctx context.Context) (context.Context, error) {
	for _, module := range m.modules {
		var err error
		ctx, err = module.Setup(args, ctx)
		if err != nil {
			return ctx, err
		}
	}
	return ctx, nil
}

func (m *ModuleLoader) Start(args *ModLifecycleArgs, ctx context.Context) (context.Context, error) {
	for _, module := range m.modules {
		var err error
		ctx, err = module.Start(args, ctx)
		if err != nil {
			return ctx, err
		}
	}
	return ctx, nil
}

func (m *ModuleLoader) Stop(args *ModLifecycleArgs, ctx context.Context) []error {
	var errs []error
	for _, module := range m.modules {
		err := module.Stop(args, ctx)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

var Modules = NewManager()
