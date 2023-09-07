package core

import (
	"fmt"
	"reflect"
	"sync"
)

// TODO: logging
// TODO: allow for accessing di on init and shutdown
// TODO: allow for shutting down non-singleton services
// TODO: allow for transparently overriding services
// TODO: implement multiple instance container that allows to get multiple different services at once and Release() them all together

type ServiceID string

type ConstructorFunc func() any

// Dependency Injection container.
// The DI is thread-safe. It currently DI does not allow for cyclical dependency checking.
// A service does not have to implement the Serviceable interface. If it does, the DI will call Init() on the service when it is first used.
// The DI will call Shutdown() on all singleton services when the DI is shutdown.
// It also does not allow for cleaning up non-singleton services on Shutdown().
type DI struct {
	mu            sync.RWMutex
	services      map[ServiceID]ServiceContainer
	isInitialized bool
	initOnce      sync.Once
}

// Container for a service.
type ServiceContainer struct {
	di          *DI
	name        string
	reflection  reflect.Type
	constructor ConstructorFunc
	service     Service
}

// The service instance is lazily initialized through Service.InitializedInstance().
type Service struct {
	sc          *ServiceContainer
	isSingleton bool
	instances   *sync.Pool
	singleton   chan *Instance
}

// The service instance. Service.InitializedInstance() will initialize the service if it is not initialized yet.
type Instance struct {
	initErr  error
	onceInit sync.Once
	i        any
	parent   *Service
}

type Serviceable interface {
	// Init is called on all services implementing the Serviceable interface when they are first used.
	// Services are lazily initialized on a DI.Get().
	Init() error

	// Shutdown is called on Singleton services which implement the Serviceable interface when the DI is shutdown.
	// Currently it is not possible to shutdown non-singleton services.
	Shutdown() error
}

func NewDI() DI {
	return DI{
		services:      make(map[ServiceID]ServiceContainer),
		isInitialized: false,
	}
}

// Initializes the DI. After initialization, no more services can be registered.
// The initialization is idempotent.
func (di *DI) Init() []error {
	var err []error

	di.initOnce.Do(func() {
		di.isInitialized = true
	})

	return err
}

// Shuts down all singleton services in the DI.
func (di *DI) Shutdown() []error {
	di.mu.RLock()
	defer di.mu.RUnlock()

	var errs []error

	for _, container := range di.services {
		if !container.service.isSingleton {
			continue
		}

		i, err := container.service.InitializedInstance()
		if err != nil {
			errs = append(errs, err)
		}

		if service, ok := i.i.(Serviceable); ok {
			if err := service.Shutdown(); err != nil {
				errs = append(errs, err)
			}
		}

		i.Release()
	}

	return errs
}

// Returns a string as ServiceID.
func GenerateServiceID(s string) ServiceID {
	return ServiceID(s)
}

// Get a service from the DI. Get() will panic if the DI is not initialized.
func (di *DI) Get(name string, t any) (*Instance, error) {
	if !di.isInitialized {
		panic("DI is not initialized! DI should be initialized before using it.")
	}

	di.mu.RLock()
	defer di.mu.RUnlock()

	id := GenerateServiceID(name)
	container, exists := di.services[id]
	if !exists {
		return nil, fmt.Errorf("service %s from %v is not registered", id, t)
	}

	rt := reflect.TypeOf(t)
	if t != nil && !container.reflection.AssignableTo(rt.Elem()) {
		return nil, fmt.Errorf("requested service %s (%s) does not meet required type: %v, got: %v", name, id, container.reflection, rt)
	}

	instance, err := container.service.InitializedInstance()

	return instance, err
}

// Get an initialized from a service struct. The service is initialized upon first use.
// This is achieved by using a sync.Once, meaning there is no retry mechanism.
// If the service fails to initialize once, it will always fail to initialize and return the same error.
func (s *Service) InitializedInstance() (*Instance, error) {
	var i *Instance
	var ok bool

	if s.isSingleton {
		i = <-s.singleton
	} else {
		i, ok = s.instances.Get().(*Instance)
		if !ok {
			return nil, fmt.Errorf("failed to get instance from pool, wrong type")
		}
	}

	i.onceInit.Do(func() {
		if service, ok := i.i.(Serviceable); ok {
			i.initErr = service.Init()
		}
	})

	if i.initErr != nil {
		return nil, fmt.Errorf("failed to initialize service: %w", i.initErr)
	}

	return i, nil
}

// Register a service with the DI.
// A service currently cannot be registered after the DI is initialized.
// Registering a service with the same name twice will return an error.
func (di *DI) Register(name string, constructor ConstructorFunc, isSingleton bool) error {
	if di.isInitialized {
		panic("DI is already initialized! Registering services is only allowed during initialization.")
	}

	di.mu.Lock()
	defer di.mu.Unlock()

	id := GenerateServiceID(name)

	_, exists := di.services[id]
	if exists {
		return fmt.Errorf("service %s is already registered", id)
	}

	i := constructor()
	rt := reflect.TypeOf(i)

	sc := ServiceContainer{
		di:          di,
		name:        name,
		reflection:  rt,
		constructor: constructor,
	}

	sc.service = newService(constructor, isSingleton, &sc)

	di.services[id] = sc

	return nil
}

// Create a new service for a service container.
func newService(constructor ConstructorFunc, isSingleton bool, sc *ServiceContainer) Service {
	s := Service{
		sc:          sc,
		isSingleton: isSingleton,
	}

	if isSingleton {
		s.singleton = make(chan *Instance, 1)
		s.singleton <- &Instance{
			i:      constructor(),
			parent: &s,
		}

		return s
	}

	s.instances = &sync.Pool{
		New: func() any {
			return &Instance{
				i:      constructor(),
				parent: &s,
			}
		},
	}

	return s
}

func (i *Instance) Instance() any {
	return i.i
}

// Release an instance back to the DI.
// Non-singleton services are released back to the sync.Pool.
// Singleton services are released back to the singleton channel.
func (i *Instance) Release() {
	if i.parent.isSingleton {
		i.parent.singleton <- i
		return
	}
	i.parent.instances.Put(i)
}
