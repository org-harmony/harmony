package core

import (
	"context"
	"fmt"
	"sync"

	"github.com/org-harmony/harmony/trace"
)

const MOD = "sys.core.module"

type Module interface {
	ID() string
	Setup(args *ModLifecycleArgs, ctx context.Context) error
	Start(args *ModLifecycleArgs, ctx context.Context) error
	Stop(args *ModLifecycleArgs) error
}

type ModuleManager struct {
	mu      sync.Mutex
	modules map[string]Module
	setup   bool
}

type ModLifecycleArgs struct {
	Logger trace.Logger
}

var modules = NewManager()

func Manager() *ModuleManager {
	return modules
}

func NewManager() *ModuleManager {
	return &ModuleManager{
		modules: make(map[string]Module),
	}
}

func (m *ModuleManager) Register(modules ...Module) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.setup {
		panic("can't register new modules after they've been setup")
	}

	for _, module := range modules {
		if _, exists := m.modules[module.ID()]; exists {
			panic(fmt.Sprintf("module with ID %s already registered", module.ID()))
		}
		m.modules[module.ID()] = module
	}
}

func (m *ModuleManager) Setup(args *ModLifecycleArgs, ctx context.Context) []error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.setup {
		panic("modules have already been setup")
	}

	m.setup = true

	var errs []error
	for _, module := range m.modules {
		if err := module.Setup(args, ctx); err != nil {
			errs = append(errs, fmt.Errorf("failed to setup module %s: %w", module.ID(), err))
		}
	}

	return errs
}

func (m *ModuleManager) Start(args *ModLifecycleArgs, ctx context.Context) []error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.setup {
		panic("modules must be setup before they can be started")
	}

	var errs []error
	for _, module := range m.modules {
		if err := module.Start(args, ctx); err != nil {
			errs = append(errs, fmt.Errorf("failed to start module %s: %w", module.ID(), err))
		}
	}

	return errs
}

func (m *ModuleManager) Stop(args *ModLifecycleArgs) []error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.setup {
		panic("modules must be setup before they can be stopped")
	}

	var errs []error
	for _, module := range m.modules {
		err := module.Stop(args)
		if err != nil {
			args.Logger.Error(MOD, "failed to stop module:", module.ID(), "Error:", err)
			errs = append(errs, err)
		} else {
			args.Logger.Info(MOD, "successfully stopped module:", module.ID())
		}
	}
	return errs
}
