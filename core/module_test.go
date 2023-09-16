package core

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/org-harmony/harmony/trace"
)

type mockModule struct {
	id string
}

func (m *mockModule) ID() string {
	return m.id
}

func (m *mockModule) Setup(args *ModLifecycleArgs, ctx context.Context) error {
	return nil
}

func (m *mockModule) Start(args *ModLifecycleArgs, ctx context.Context) error {
	return nil
}

func (m *mockModule) Stop(args *ModLifecycleArgs) error {
	return nil
}

func TestModuleManagerLifecycle(t *testing.T) {
	logger := trace.NewStdLogger()

	t.Run("setup", func(t *testing.T) {
		manager := NewManager()
		module := &mockModule{id: "test.module"}

		manager.Register(module)

		errs := manager.Setup(&ModLifecycleArgs{Logger: logger}, context.Background())
		if len(errs) > 0 {
			t.Errorf("Expected no errors during setup, but got: %v", errs)
		}
	})

	t.Run("start", func(t *testing.T) {
		manager := NewManager()
		module := &mockModule{id: "test.module"}

		manager.Register(module)
		manager.Setup(&ModLifecycleArgs{Logger: logger}, context.Background())

		errs := manager.Start(&ModLifecycleArgs{Logger: logger}, context.Background())
		if len(errs) > 0 {
			t.Errorf("Expected no errors during start, but got: %v", errs)
		}
	})

	t.Run("stop", func(t *testing.T) {
		manager := NewManager()
		module := &mockModule{id: "test.module"}

		manager.Register(module)
		manager.Setup(&ModLifecycleArgs{Logger: logger}, context.Background())
		manager.Start(&ModLifecycleArgs{Logger: logger}, context.Background())

		errs := manager.Stop(&ModLifecycleArgs{Logger: logger})
		if len(errs) > 0 {
			t.Errorf("Expected no errors during stop, but got: %v", errs)
		}
	})
}

func TestModuleRegistration(t *testing.T) {
	logger := trace.NewStdLogger()

	t.Run("registration", func(t *testing.T) {
		manager := NewManager()
		module1 := &mockModule{id: "test.module1"}
		module2 := &mockModule{id: "test.module2"}

		manager.Register(module1)
		manager.Register(module2)
	})

	t.Run("registration after setup", func(t *testing.T) {
		manager := NewManager()
		module1 := &mockModule{id: "test.module1"}

		manager.Register(module1)
		manager.Setup(&ModLifecycleArgs{Logger: logger}, context.Background())

		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected a panic when registering after setup but got none")
			}
		}()
		manager.Register(module1)
	})

	t.Run("start without registration", func(t *testing.T) {
		manager := NewManager()

		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected a panic when starting without registration but got none")
			}
		}()
		manager.Start(&ModLifecycleArgs{Logger: logger}, context.Background())
	})

	t.Run("stop without registration", func(t *testing.T) {
		manager := NewManager()

		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected a panic when stopping without registration but got none")
			}
		}()
		manager.Stop(&ModLifecycleArgs{Logger: logger})
	})

	t.Run("duplicate registration", func(t *testing.T) {
		manager := NewManager()
		module1 := &mockModule{id: "test.module"}

		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected a panic during duplicate registration but got none")
			}
		}()

		manager.Register(module1)
		manager.Register(module1)
	})
}

func TestModuleManagerConcurrency(t *testing.T) {
	t.Run("concurrent registration", func(t *testing.T) {
		manager := NewManager()

		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				module := &mockModule{id: fmt.Sprintf("test.module%d", id)}
				manager.Register(module)
			}(i)
		}
		wg.Wait()
	})

	t.Run("concurrent duplicate registration", func(t *testing.T) {
		manager := NewManager()
		module := &mockModule{id: "test.module"}

		// This channel will collect panics from the goroutines
		panicCh := make(chan interface{}, 10)

		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer func() {
					if r := recover(); r != nil {
						panicCh <- r
					}
				}()
				manager.Register(module)
			}()
		}
		wg.Wait()
		close(panicCh)

		// We expect 9 panics due to duplicate registration
		if len(panicCh) != 9 {
			t.Errorf("Expected 9 panics due to duplicate registration, but got %d", len(panicCh))
		}
	})
}

type errorMockModule struct {
	id       string
	setupErr error
	startErr error
	stopErr  error
}

func (m *errorMockModule) ID() string {
	return m.id
}

func (m *errorMockModule) Setup(args *ModLifecycleArgs, ctx context.Context) error {
	return m.setupErr
}

func (m *errorMockModule) Start(args *ModLifecycleArgs, ctx context.Context) error {
	return m.startErr
}

func (m *errorMockModule) Stop(args *ModLifecycleArgs) error {
	return m.stopErr
}

func TestModuleManagerMultipleErrorHandling(t *testing.T) {
	logger := trace.NewStdLogger()

	t.Run("multiple setup errors", func(t *testing.T) {
		manager := NewManager()
		module1 := &errorMockModule{id: "test.module1", setupErr: errors.New("module1 setup error")}
		module2 := &errorMockModule{id: "test.module2", setupErr: errors.New("module2 setup error")}

		manager.Register(module1, module2)
		errs := manager.Setup(&ModLifecycleArgs{Logger: logger}, context.Background())

		if len(errs) != 2 {
			t.Errorf("Expected 2 errors during setup, but got: %v", len(errs))
		}
	})

	t.Run("multiple start errors", func(t *testing.T) {
		manager := NewManager()
		module1 := &errorMockModule{id: "test.module1", startErr: errors.New("module1 start error")}
		module2 := &errorMockModule{id: "test.module2", startErr: errors.New("module2 start error")}

		manager.Register(module1, module2)
		manager.Setup(&ModLifecycleArgs{Logger: logger}, context.Background())

		errs := manager.Start(&ModLifecycleArgs{Logger: logger}, context.Background())
		if len(errs) != 2 {
			t.Errorf("Expected 2 errors during start, but got: %v", len(errs))
		}
	})

	t.Run("multiple stop errors", func(t *testing.T) {
		manager := NewManager()
		module1 := &errorMockModule{id: "test.module1", stopErr: errors.New("module1 stop error")}
		module2 := &errorMockModule{id: "test.module2", stopErr: errors.New("module2 stop error")}

		manager.Register(module1, module2)
		manager.Setup(&ModLifecycleArgs{Logger: logger}, context.Background())
		manager.Start(&ModLifecycleArgs{Logger: logger}, context.Background())

		errs := manager.Stop(&ModLifecycleArgs{Logger: logger})
		if len(errs) != 2 {
			t.Errorf("Expected 2 errors during stop, but got: %v", len(errs))
		}
	})
}
