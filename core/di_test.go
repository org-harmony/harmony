package core

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestDIInitialization(t *testing.T) {
	di := NewDI()

	if di.isInitialized {
		t.Error("DI should not be initialized by default")
	}

	di.Init()

	if !di.isInitialized {
		t.Error("DI should be initialized after calling Init")
	}
}

func TestServiceRegistration(t *testing.T) {
	di := NewDI()

	t.Run("FirstTimeRegistration", func(t *testing.T) {
		err := di.Register("test.service", func() any { return "test" }, false)
		if err != nil {
			t.Errorf("Failed to register service: %v on first try.", err)
		}
	})

	t.Run("DuplicateRegistration", func(t *testing.T) {
		err := di.Register("test.service", func() any { return "test" }, false)
		if err == nil {
			t.Error("Should not be able to register duplicate service")
		}
	})

	t.Run("RegistrationAfterInitialization", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Should have panicked when trying to register a service after DI has been initialized")
			}
		}()

		di.Init()
		di.Register("another.service", func() any { return "another" }, false)
	})
}

func TestServiceRetrieval(t *testing.T) {
	t.Run("GetFromUninitializedDI", func(t *testing.T) {
		di := NewDI()
		defer func() {
			if r := recover(); r == nil {
				t.Error("Should have panicked when trying to get a service from an uninitialized DI")
			}
		}()
		_, _ = di.Get("test.service", nil)
	})

	di := NewDI()
	err := di.Register("test.service", func() any { return "test" }, false)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}
	di.Init()

	t.Run("GetNonExistentService", func(t *testing.T) {
		_, err := di.Get("nonexistent.service", nil)
		if err == nil {
			t.Error("Expected an error when getting a non-existent service, but got none")
		}
	})

	t.Run("GetRegisteredService", func(t *testing.T) {
		instance, err := di.Get("test.service", (*string)(nil))
		if err != nil {
			t.Errorf("Failed to get registered service: %v", err)
		}
		if instance.Instance().(string) != "test" {
			t.Errorf("Expected service value to be 'test', got: %v", instance.Instance())
		}
	})

	t.Run("GetWithIncorrectType", func(t *testing.T) {
		_, err := di.Get("test.service", (*int)(nil))
		if err == nil {
			t.Error("Expected an error when getting a service with incorrect type, but got none")
		}
	})
}

func TestConcurrentServiceAccess(t *testing.T) {
	const concurrentAccesses = 100

	// Helper function to concurrently access services
	runConcurrentAccess := func(di *DI, serviceName string, serviceType any) {
		var wg sync.WaitGroup
		wg.Add(concurrentAccesses)

		for i := 0; i < concurrentAccesses; i++ {
			go func() {
				defer wg.Done()
				instance, err := di.Get(serviceName, serviceType)
				if err != nil {
					t.Errorf("Failed to get service: %v", err)
				}
				// Simulating some work with the service
				time.Sleep(time.Millisecond)
				instance.Release()
			}()
		}

		wg.Wait()
	}

	di := NewDI()

	err := di.Register("singleton.service", func() any { return "singleton" }, true)
	if err != nil {
		t.Fatalf("Setup failed for singleton service: %v", err)
	}

	err = di.Register("non.singleton.service", func() any { return "non-singleton" }, false)
	if err != nil {
		t.Fatalf("Setup failed for non-singleton service: %v", err)
	}

	di.Init()

	// Test concurrent access to singleton service
	t.Run("SingletonServiceConcurrency", func(t *testing.T) {
		runConcurrentAccess(&di, "singleton.service", (*string)(nil))
	})

	// Test concurrent access to non-singleton service
	t.Run("NonSingletonServiceConcurrency", func(t *testing.T) {
		runConcurrentAccess(&di, "non.singleton.service", (*string)(nil))
	})
}

type FailsOnInit struct{}

func (s *FailsOnInit) Init() error {
	return fmt.Errorf("failed to initialize")
}

func (s *FailsOnInit) Shutdown() error {
	return nil
}

type FailsOnShutdown struct{}

func (s *FailsOnShutdown) Init() error {
	return nil
}

func (s *FailsOnShutdown) Shutdown() error {
	return fmt.Errorf("failed to shutdown")
}

func TestServiceInitializationError(t *testing.T) {
	di := NewDI()

	err := di.Register("error.service", func() any { return &FailsOnInit{} }, false)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	di.Init()

	instance, err := di.Get("error.service", (*Serviceable)(nil))
	if err == nil {
		t.Error("Expected an error when getting a service with initialization error, but got none")
	}
	if instance != nil {
		t.Error("Expected no instance when service initialization fails")
	}

	// Ensure the error persists across multiple gets
	_, err2 := di.Get("error.service", (*Serviceable)(nil))
	if err2 == nil || err2.Error() != err.Error() {
		t.Error("Expected the same initialization error on subsequent gets")
	}
}

type SingletonService struct {
	isShutdown *BoolReference
}

func (s *SingletonService) Init() error {
	return nil
}

func (s *SingletonService) Shutdown() error {
	s.isShutdown.b = true
	return nil
}

type BoolReference struct {
	b bool
}

func TestSingletonServiceShutdown(t *testing.T) {
	di := NewDI()
	isShutdown := BoolReference{b: false}

	err := di.Register("singleton.service", func() any {
		return &SingletonService{
			isShutdown: &isShutdown,
		}
	}, true)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	di.Init()
	di.Shutdown()

	if !isShutdown.b {
		t.Error("Expected Shutdown() to be called on the singleton service")
	}
}

func TestNonSingletonServicePooling(t *testing.T) {
	di := NewDI()

	err := di.Register("non.singleton.service", func() any { return &struct{}{} }, false)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	di.Init()

	instance1, _ := di.Get("non.singleton.service", nil)
	instance2, _ := di.Get("non.singleton.service", nil)

	if instance1 == instance2 {
		t.Error("Expected different instances for non-singleton services")
	}
}

func TestServiceTypeSafety(t *testing.T) {
	di := NewDI()

	err := di.Register("string.service", func() any { return "test" }, false)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	di.Init()

	_, err = di.Get("string.service", (*int)(nil))
	if err == nil {
		t.Error("Expected a type mismatch error, but got none")
	}
}

func TestConcurrentRegistration(t *testing.T) {
	di := NewDI()

	const goroutineCount = 100

	failedCount := int32(0) // Using int32 to leverage atomic operations

	var wg sync.WaitGroup
	wg.Add(goroutineCount)

	for i := 0; i < goroutineCount; i++ {
		go func(id int) {
			defer wg.Done()

			err := di.Register("concurrent.service", func() any { return fmt.Sprintf("from goroutine %d", id) }, false)
			if err != nil {
				atomic.AddInt32(&failedCount, 1)
			}
		}(i)
	}

	wg.Wait()

	di.Init()
	instance, _ := di.Get("concurrent.service", nil)

	if instance.Instance() == nil || !strings.HasPrefix(instance.Instance().(string), "from goroutine ") {
		t.Error("Concurrent registration failed to register the service properly")
	}

	if failedCount != goroutineCount-1 {
		t.Errorf("Expected %d goroutines to fail, but %d did", goroutineCount-1, failedCount)
	}
}
