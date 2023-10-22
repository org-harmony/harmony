package event

import (
	"fmt"
	"github.com/org-harmony/harmony/src/core/trace"
	"sync"
	"sync/atomic"
	"testing"
)

type mockPayload struct {
	data     string
	moreData string
}

type mockEvent struct {
	id string
	p  mockPayload
	d  chan []error
}

func newMockEvent(id string) *mockEvent {
	return &mockEvent{
		id: id,
		p: mockPayload{
			data:     "test",
			moreData: "test",
		},
		d: make(chan []error),
	}
}

func (e *mockEvent) ID() string {
	return e.id
}

func (e *mockEvent) Payload() any {
	return &e.p
}

func (e *mockEvent) DoneChan() chan []error {
	return e.d
}

func TestBasicEventSubscribing(t *testing.T) {
	logger := trace.NewTestLogger(t)

	t.Run("single subscriber", func(t *testing.T) {
		em := NewManager(logger)

		received := false
		subscriberFunc := func(e Event, args *publishArgs) error {
			if e.Payload().(*mockPayload).data != "test" {
				t.Error("Received incorrect event payload data")
			}
			received = true
			return nil
		}

		em.Subscribe("test.event", subscriberFunc, DefaultEventPriority)

		event := newMockEvent("test.event")
		dc := make(chan []error)
		em.Publish(event, dc)

		<-dc

		if !received {
			t.Error("Expected subscriber to receive and process the event, but it didn't")
		}
	})

	t.Run("multiple subscribers same event", func(t *testing.T) {
		em := NewManager(logger)

		var count int
		subscriberFunc := func(e Event, args *publishArgs) error {
			if e.Payload().(*mockPayload).data != "test" {
				t.Error("Received incorrect event payload data")
			}
			count++
			return nil
		}

		em.Subscribe("test.event.multiple", subscriberFunc, DefaultEventPriority)
		em.Subscribe("test.event.multiple", subscriberFunc, DefaultEventPriority)

		event := newMockEvent("test.event.multiple")
		dc := make(chan []error)
		em.Publish(event, dc)

		<-dc

		if count != 2 {
			t.Errorf("Expected both subscribers to process the event, but only %d did", count)
		}
	})

	t.Run("fire and forget", func(t *testing.T) {
		em := NewManager(logger)

		subscriberFunc := func(e Event, args *publishArgs) error {
			return nil
		}

		em.Subscribe("test.event.fire", subscriberFunc, DefaultEventPriority)

		event := newMockEvent("test.event.fire")
		em.Publish(event, nil)
	})

	t.Run("channel closed after use", func(t *testing.T) {
		em := NewManager(logger)

		subscriberFunc := func(e Event, args *publishArgs) error {
			return nil
		}

		em.Subscribe("test.event.channel", subscriberFunc, DefaultEventPriority)

		event := newMockEvent("test.event.channel")
		dc := make(chan []error)
		em.Publish(event, dc)

		<-dc

		_, ok := <-dc
		if ok {
			t.Error("Expected the done channel to be closed after use")
		}
	})
}

func TestBasicPublishing(t *testing.T) {
	logger := trace.NewTestLogger(t)

	t.Run("priority order", func(t *testing.T) {
		em := NewManager(logger)

		var order []int

		// Subscribers that just append their priority to the order slice
		for i := 1; i <= 5; i++ {
			priority := i // capture range variable
			em.Subscribe("test.event.priority", func(e Event, args *publishArgs) error {
				order = append(order, priority)
				return nil
			}, priority)
		}

		event := newMockEvent("test.event.priority")
		dc := make(chan []error)
		em.Publish(event, dc)

		<-dc

		// Ensure the order is correct
		for i, priority := range order {
			if priority != i+1 {
				t.Errorf("Expected priority %d but got %d", i+1, priority)
			}
		}
	})

	t.Run("stop propagation", func(t *testing.T) {
		em := NewManager(logger)

		var received []int

		// First subscriber stops propagation
		em.Subscribe("test.event.stop", func(e Event, args *publishArgs) error {
			received = append(received, 1)
			args.StopPropagation = true
			return nil
		}, 1)

		// This should not be invoked because of the stop
		em.Subscribe("test.event.stop", func(e Event, args *publishArgs) error {
			received = append(received, 2)
			return nil
		}, 2)

		event := newMockEvent("test.event.stop")
		dc := make(chan []error)
		em.Publish(event, dc)

		<-dc

		if len(received) != 1 || received[0] != 1 {
			t.Errorf("Expected only the first subscriber to be invoked but got %+v", received)
		}
	})

	t.Run("payload modification", func(t *testing.T) {
		em := NewManager(logger)

		// subscriber that modifies the payload
		em.Subscribe("test.event.payload", func(e Event, args *publishArgs) error {
			e.Payload().(*mockPayload).data = "modified"
			return nil
		}, DefaultEventPriority)

		event := newMockEvent("test.event.payload")
		dc := make(chan []error)
		em.Publish(event, dc)

		<-dc

		if event.Payload().(*mockPayload).data != "modified" {
			t.Error("Expected the payload to be modified by the subscriber")
		}
	})

	t.Run("multiple subscribers same priority", func(t *testing.T) {
		em := NewManager(logger)

		var received []int

		em.Subscribe("test.event.multiple", func(e Event, args *publishArgs) error {
			received = append(received, 1)
			return nil
		}, DefaultEventPriority)

		em.Subscribe("test.event.multiple", func(e Event, args *publishArgs) error {
			received = append(received, 2)
			return nil
		}, DefaultEventPriority)

		event := newMockEvent("test.event.multiple")
		dc := make(chan []error)
		em.Publish(event, dc)

		<-dc

		if len(received) != 2 || received[0] != 1 || received[1] != 2 {
			t.Errorf("Expected both subscribers to be invoked but got %+v", received)
		}
	})
}

func TestErrorHandling(t *testing.T) {
	logger := trace.NewTestLogger(t)

	t.Run("single subscriber", func(t *testing.T) {
		em := NewManager(logger)

		em.Subscribe("test.event.error", func(e Event, args *publishArgs) error {
			return fmt.Errorf("test error")
		}, DefaultEventPriority)

		event := newMockEvent("test.event.error")
		dc := make(chan []error)
		em.Publish(event, dc)

		errs := <-dc

		if len(errs) != 1 {
			t.Errorf("Expected 1 error but got %d", len(errs))
		}
	})

	t.Run("multiple subscribers", func(t *testing.T) {
		em := NewManager(logger)

		em.Subscribe("test.event.error", func(e Event, args *publishArgs) error {
			return fmt.Errorf("test error")
		}, DefaultEventPriority)

		em.Subscribe("test.event.error", func(e Event, args *publishArgs) error {
			return fmt.Errorf("test error")
		}, DefaultEventPriority)

		event := newMockEvent("test.event.error")
		dc := make(chan []error)
		em.Publish(event, dc)

		errs := <-dc

		if len(errs) != 2 {
			t.Errorf("Expected 2 errors but got %d", len(errs))
		}
	})

	t.Run("panic", func(t *testing.T) {
		em := NewManager(logger)

		em.Subscribe("test.event.panic", func(e Event, args *publishArgs) error {
			panic("test panic")
		}, DefaultEventPriority)

		event := newMockEvent("test.event.panic")
		dc := make(chan []error)
		em.Publish(event, dc)

		errs := <-dc

		if len(errs) != 1 {
			t.Errorf("Expected 1 error but got %d", len(errs))
		}
	})

	t.Run("panic and other error", func(t *testing.T) {
		em := NewManager(logger)

		em.Subscribe("test.event.panic", func(e Event, args *publishArgs) error {
			panic("test panic")
		}, DefaultEventPriority)

		em.Subscribe("test.event.panic", func(e Event, args *publishArgs) error {
			return fmt.Errorf("test error")
		}, DefaultEventPriority)

		em.Subscribe("test.event.panic", func(e Event, args *publishArgs) error {
			return nil
		}, DefaultEventPriority)

		event := newMockEvent("test.event.panic")
		dc := make(chan []error)
		em.Publish(event, dc)

		errs := <-dc

		if len(errs) != 2 {
			t.Errorf("Expected 2 errors but got %d", len(errs))
		}
	})

	t.Run("panic and further processing", func(t *testing.T) {
		em := NewManager(logger)

		var received bool

		em.Subscribe("test.event.panic", func(e Event, args *publishArgs) error {
			panic("test panic")
		}, DefaultEventPriority)

		em.Subscribe("test.event.panic", func(e Event, args *publishArgs) error {
			received = true
			return nil
		}, DefaultEventPriority)

		event := newMockEvent("test.event.panic")
		dc := make(chan []error)
		em.Publish(event, dc)

		errs := <-dc

		if len(errs) != 1 {
			t.Errorf("Expected 1 error but got %d", len(errs))
		}

		if !received {
			t.Error("Expected the second subscriber to be invoked after the panic")
		}
	})
}

func TestConcurrentOperations(t *testing.T) {
	logger := trace.NewTestLogger(t)

	t.Run("concurrent event publishing", func(t *testing.T) {
		em := NewManager(logger)

		var count int32

		em.Subscribe("test.event.concurrent.publish", func(e Event, args *publishArgs) error {
			atomic.AddInt32(&count, 1)
			return nil
		}, DefaultEventPriority)

		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)

			go func() {
				defer wg.Done()

				event := newMockEvent("test.event.concurrent.publish")
				dc := make(chan []error)
				em.Publish(event, dc)

				<-dc
			}()
		}
		wg.Wait()

		if atomic.LoadInt32(&count) != 100 {
			t.Errorf("Expected 100 processed events but got %d", count)
		}
	})

	t.Run("concurrent subscriber registration", func(t *testing.T) {
		em := NewManager(logger)

		var count int32

		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)

			go func(num int) {
				defer wg.Done()

				em.Subscribe(fmt.Sprintf("test.event.concurrent.subscribe.%d", num), func(e Event, args *publishArgs) error {
					atomic.AddInt32(&count, 1)
					return nil
				}, DefaultEventPriority)
			}(i)
		}
		wg.Wait()

		for i := 0; i < 100; i++ {
			event := newMockEvent(fmt.Sprintf("test.event.concurrent.subscribe.%d", i))
			dc := make(chan []error)
			em.Publish(event, dc)

			<-dc
		}

		if atomic.LoadInt32(&count) != 100 {
			t.Errorf("Expected 100 subscribers to be invoked but got %d", count)
		}
	})

	t.Run("mixed operations of concurrent publishing and subscribing", func(t *testing.T) {
		em := NewManager(logger)

		var pubCount int32
		var subCount int32

		var wg sync.WaitGroup

		for i := 0; i < 100; i++ {
			wg.Add(1)

			c := i

			go func() {
				defer wg.Done()

				em.Subscribe(fmt.Sprintf("test.event.concurrent.mixed.%d", c), func(e Event, args *publishArgs) error {
					atomic.AddInt32(&subCount, 1)
					return nil
				}, DefaultEventPriority)

				event := newMockEvent(fmt.Sprintf("test.event.concurrent.mixed.%d", c))
				dc := make(chan []error)
				em.Publish(event, dc)

				<-dc

				atomic.AddInt32(&pubCount, 1)
			}()
		}

		wg.Wait()

		if atomic.LoadInt32(&pubCount) != 100 {
			t.Errorf("Expected 100 published events but got %d", pubCount)
		}

		if atomic.LoadInt32(&subCount) != 100 {
			t.Errorf("Expected 100 processed events by subscribers but got %d", subCount)
		}
	})
}

// TODO test for nil function as subscriber func
// TODO maybe stress testing concurrent execution with a lot of subscribers and events (e.g. 1.000.000.000)?
// TODO test order of subscribers in highly concurrent scenarios
