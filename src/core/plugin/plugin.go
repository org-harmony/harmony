// Package plugin provides HARMONY's plugin system for extending and modifying functionality.
package plugin

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

var (
	// ErrRegistryClosed indicates the registry being closed, adding a new Plugin is illegal.
	ErrRegistryClosed = errors.New("closed registry can not be modified")
	// ErrDuplicateRegistration indicates that a plugin with the exact ID was already registered.
	ErrDuplicateRegistration = errors.New("plugin with same id already registered")
	// ErrNotClosed indicates the Registry from which the Manager should derive is not closed.
	ErrNotClosed = errors.New("registry not closed when it is expected")
	// ErrRegisterEventFailed indicates an error with event.Event registration on event.Bus through Registry.
	ErrRegisterEventFailed = errors.New("could not register event of plugin")
	// ErrAttachListenerFailed indicates an error with event.Listener registration on event.Bus through Registry.
	ErrAttachListenerFailed = errors.New("could not attach listener of plugin")
)

// Reg is HARMONY's default plugin registry. Use it to register your plugin.
//
// Example:
//
//	package mymodule
//
//	import ...
//
//	func init() {
//		plugin.Reg.Register(Plugin{ ... })
//	}
var Reg Registry = &HRegistry{logger: trace.NewLogger()}

// ID is the unique identification for a Plugin in a Registry.
type ID string

// HRegistry is the systems default implementation of a Plugin Registry.
// HRegistry is safe for concurrent use by multiple goroutines.
// See Registry for more information.
type HRegistry struct {
	closed  bool
	logger  trace.Logger
	mu      sync.RWMutex
	plugins map[ID]Plugin
}

// HManager is the systems default implementation of the Manager interface.
// HManager is safe for concurrent use by multiple goroutines.
// See Manager for further information.
type HManager struct {
	plugins map[ID]Plugin
	mu      sync.Mutex
	bus     event.Bus
}

// Plugin
type Plugin struct {
	ID        ID
	Build     func(context.Context, BuildOpts) error
	Teardown  func(context.Context, TeardownOpts) error
	Events    []event.Event
	Listeners []event.Listener
}

// tbd
type BuildOpts struct{}

// tbd
type TeardownOpts struct{}

// tbd
type Registry interface {
	Plugin(ID) (Plugin, bool)
	Register(Plugin) error
	Closed() bool
	Close() error
	Manager() (Manager, error)
}

// tbd
type Manager interface {
	Plugin(ID) (Plugin, bool)
	Build(context.Context, BuildOpts, chan<- error)
	Teardown(context.Context, TeardownOpts, chan<- error)
	Bus() event.Bus
}

// tbd
func (r *HRegistry) Plugin(id ID) (Plugin, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugin, ok := r.plugins[id]

	return plugin, ok
}

// tbd
func (r *HRegistry) Register(plugin Plugin) error {
	if r.Closed() {
		return ErrRegistryClosed
	}

	if _, exists := r.Plugin(plugin.ID); exists {
		return ErrDuplicateRegistration
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.plugins[plugin.ID] = plugin

	return nil
}

// tbd
func (r *HRegistry) Closed() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.closed
}

// tbd
func (r *HRegistry) Close() error {
	if r.Closed() { // use of method to ensure correct locking
		return ErrRegistryClosed
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.closed = true

	return nil
}

// tbd
func (r *HRegistry) Manager() (Manager, error) {
	if !r.Closed() {
		return nil, ErrNotClosed
	}

	manager := &HManager{plugins: r.plugins, bus: &event.HBus{}}

	for _, plugin := range manager.plugins {
		for _, e := range plugin.Events {
			err := manager.bus.Register(e)
			if err == nil {
				continue
			}

			return nil, errors.Join(
				fmt.Errorf("could not register event %s for plugin %s", e.ID, plugin.ID),
				ErrRegisterEventFailed,
				err,
			)
		}

		for _, l := range plugin.Listeners {
			err := manager.bus.Attach(l)
			if err == nil {
				continue
			}

			return nil, errors.Join(
				fmt.Errorf("could not attach listener %s for event %s for plugin %s", l.ID, l.EventID, plugin.ID),
				ErrAttachListenerFailed,
				err,
			)
		}
	}

	return manager, nil
}

// tbd
func (m *HManager) Plugin(id ID) (Plugin, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	plugin, ok := m.plugins[id]

	return plugin, ok
}

// tbd
func (m *HManager) Build(ctx context.Context, opts BuildOpts, errs chan<- error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, plugin := range m.plugins {
		err := plugin.Build(ctx, opts)
		if err == nil {
			continue
		}

		errs <- err
	}
}

// tbd
func (m *HManager) Teardown(ctx context.Context, opts TeardownOpts, errs chan<- error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, plugin := range m.plugins {
		err := plugin.Teardown(ctx, opts)
		if err == nil {
			continue
		}

		errs <- err
	}
}

// tbd
func (m *HManager) Bus() event.Bus {
	return m.bus
}
