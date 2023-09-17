package core

import (
	"context"
	"fmt"
	"sync"

	"github.com/org-harmony/harmony/trace"
)

const MODULE_MOD = "sys.core.module"

// Module is an interface representing components that can be managed by the ModuleManager.
// Implementing the Module interface allows for lifecycle management including setup, start and stop operations.
type Module interface {
	// ID returns a unique identifier for the module.
	ID() string
	// Setup prepares the module for operation. It is called before Start.
	Setup(args *ModLifecycleArgs, ctx context.Context) error
	// Start activates the module. It is called after Setup and before Stop.
	Start(args *ModLifecycleArgs, ctx context.Context) error
	// Stop deactivates the module. It is called after Start.
	// Stop should be used for cleanup tasks.
	Stop(args *ModLifecycleArgs) error
}

// ModuleManager is responsible for managing the lifecycle of registered modules.
// It ensures modules are set up, started and stopped in a controlled manner.
type ModuleManager struct {
	mu sync.Mutex
	// Collection of registered modules indexed by their ID
	modules map[string]Module
	// Flag indicating if modules have been set up
	setup bool
}

// ModLifecycleArgs holds arguments that are passed to module lifecycle methods.
type ModLifecycleArgs struct {
	Logger trace.Logger
}

var modules = NewManager()

// Manager provides a singleton instance of ModuleManager.
func Manager() *ModuleManager {
	return modules
}

// NewManager creates and returns a new instance of ModuleManager.
func NewManager() *ModuleManager {
	return &ModuleManager{
		modules: make(map[string]Module),
	}
}

// Register registers one or more modules with the manager.
// It's essential to register modules before setting them up.
// Registering after setup will result in a panic.
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

// Setup initializes all registered modules.
// This method should be called before starting the modules.
// Calling Setup after the modules have been setup will result in a panic.
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

// Start activates all registered modules.
// It's essential to set up the modules before starting otherwise a panic will occur.
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

// Stop deactivates all registered modules.
// Modules should be started before they can be stopped otherwise a panic will occur.
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
			errs = append(errs, fmt.Errorf("failed to stop module %s: %w", module.ID(), err))
		}
	}
	return errs
}
