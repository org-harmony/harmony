package core

import (
	"fmt"
	"testing"

	"github.com/org-harmony/harmony/trace"
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

func NewMockEvent(id string) *mockEvent {
	return &mockEvent{
		id: id,
		p: mockPayload{
			data:     "test",
			moreData: "test",
		},
		d: make(chan []error),
	}
}

func (me *mockEvent) ID() string {
	return me.id
}

func (me *mockEvent) Payload() any {
	return &me.p
}

func (me *mockEvent) DoneChan() chan []error {
	return me.d
}

func TestBasicEventSubscribing(t *testing.T) {
	logger := trace.NewTestLogger(t)

	t.Run("single subscriber", func(t *testing.T) {
		em := NewEventManager(logger)

		received := false
		subscriberFunc := func(e Event, args *PublishArgs) error {
			if e.Payload().(*mockPayload).data != "test" {
				t.Error("Received incorrect event payload data")
			}
			received = true
			return nil
		}

		em.Subscribe("test.event", subscriberFunc, DEFAULT_EVENT_PRIORITY)

		event := NewMockEvent("test.event")
		dc := make(chan []error)
		em.Publish(event, dc)

		<-dc

		if !received {
			t.Error("Expected subscriber to receive and process the event, but it didn't")
		}
	})

	t.Run("multiple subscribers same event", func(t *testing.T) {
		em := NewEventManager(logger)

		var count int
		subscriberFunc := func(e Event, args *PublishArgs) error {
			if e.Payload().(*mockPayload).data != "test" {
				t.Error("Received incorrect event payload data")
			}
			count++
			return nil
		}

		em.Subscribe("test.event.multiple", subscriberFunc, DEFAULT_EVENT_PRIORITY)
		em.Subscribe("test.event.multiple", subscriberFunc, DEFAULT_EVENT_PRIORITY)

		event := NewMockEvent("test.event.multiple")
		dc := make(chan []error)
		em.Publish(event, dc)

		<-dc

		if count != 2 {
			t.Errorf("Expected both subscribers to process the event, but only %d did", count)
		}
	})

	t.Run("fire and forget", func(t *testing.T) {
		em := NewEventManager(logger)

		subscriberFunc := func(e Event, args *PublishArgs) error {
			return nil
		}

		em.Subscribe("test.event.fire", subscriberFunc, DEFAULT_EVENT_PRIORITY)

		event := NewMockEvent("test.event.fire")
		em.Publish(event, nil)
	})

	t.Run("channel closed after use", func(t *testing.T) {
		em := NewEventManager(logger)

		subscriberFunc := func(e Event, args *PublishArgs) error {
			return nil
		}

		em.Subscribe("test.event.channel", subscriberFunc, DEFAULT_EVENT_PRIORITY)

		event := NewMockEvent("test.event.channel")
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
		em := NewEventManager(logger)

		order := []int{}

		// Subscribers that just append their priority to the order slice
		for i := 1; i <= 5; i++ {
			priority := i // capture range variable
			em.Subscribe("test.event.priority", func(e Event, args *PublishArgs) error {
				order = append(order, priority)
				return nil
			}, priority)
		}

		event := NewMockEvent("test.event.priority")
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
		em := NewEventManager(logger)

		received := []int{}

		// First subscriber stops propagation
		em.Subscribe("test.event.stop", func(e Event, args *PublishArgs) error {
			received = append(received, 1)
			args.StopPropagation = true
			return nil
		}, 1)

		// This should not be invoked because of the stop
		em.Subscribe("test.event.stop", func(e Event, args *PublishArgs) error {
			received = append(received, 2)
			return nil
		}, 2)

		event := NewMockEvent("test.event.stop")
		dc := make(chan []error)
		em.Publish(event, dc)

		<-dc

		if len(received) != 1 || received[0] != 1 {
			t.Errorf("Expected only the first subscriber to be invoked but got %+v", received)
		}
	})

	t.Run("payload modification", func(t *testing.T) {
		em := NewEventManager(logger)

		// Subscriber that modifies the payload
		em.Subscribe("test.event.payload", func(e Event, args *PublishArgs) error {
			e.Payload().(*mockPayload).data = "modified"
			return nil
		}, DEFAULT_EVENT_PRIORITY)

		event := NewMockEvent("test.event.payload")
		dc := make(chan []error)
		em.Publish(event, dc)

		<-dc

		if event.Payload().(*mockPayload).data != "modified" {
			t.Error("Expected the payload to be modified by the subscriber")
		}
	})

	t.Run("multiple subscribers same priority", func(t *testing.T) {
		em := NewEventManager(logger)

		received := []int{}

		em.Subscribe("test.event.multiple", func(e Event, args *PublishArgs) error {
			received = append(received, 1)
			return nil
		}, DEFAULT_EVENT_PRIORITY)

		em.Subscribe("test.event.multiple", func(e Event, args *PublishArgs) error {
			received = append(received, 2)
			return nil
		}, DEFAULT_EVENT_PRIORITY)

		event := NewMockEvent("test.event.multiple")
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
		em := NewEventManager(logger)

		em.Subscribe("test.event.error", func(e Event, args *PublishArgs) error {
			return fmt.Errorf("test error")
		}, DEFAULT_EVENT_PRIORITY)

		event := NewMockEvent("test.event.error")
		dc := make(chan []error)
		em.Publish(event, dc)

		errs := <-dc

		if len(errs) != 1 {
			t.Errorf("Expected 1 error but got %d", len(errs))
		}
	})

	t.Run("multiple subscribers", func(t *testing.T) {
		em := NewEventManager(logger)

		em.Subscribe("test.event.error", func(e Event, args *PublishArgs) error {
			return fmt.Errorf("test error")
		}, DEFAULT_EVENT_PRIORITY)

		em.Subscribe("test.event.error", func(e Event, args *PublishArgs) error {
			return fmt.Errorf("test error")
		}, DEFAULT_EVENT_PRIORITY)

		event := NewMockEvent("test.event.error")
		dc := make(chan []error)
		em.Publish(event, dc)

		errs := <-dc

		if len(errs) != 2 {
			t.Errorf("Expected 2 errors but got %d", len(errs))
		}
	})

	t.Run("panic", func(t *testing.T) {
		em := NewEventManager(logger)

		em.Subscribe("test.event.panic", func(e Event, args *PublishArgs) error {
			panic("test panic")
		}, DEFAULT_EVENT_PRIORITY)

		event := NewMockEvent("test.event.panic")
		dc := make(chan []error)
		em.Publish(event, dc)

		errs := <-dc

		if len(errs) != 1 {
			t.Errorf("Expected 1 error but got %d", len(errs))
		}
	})

	t.Run("panic and other error", func(t *testing.T) {
		em := NewEventManager(logger)

		em.Subscribe("test.event.panic", func(e Event, args *PublishArgs) error {
			panic("test panic")
		}, DEFAULT_EVENT_PRIORITY)

		em.Subscribe("test.event.panic", func(e Event, args *PublishArgs) error {
			return fmt.Errorf("test error")
		}, DEFAULT_EVENT_PRIORITY)

		em.Subscribe("test.event.panic", func(e Event, args *PublishArgs) error {
			return nil
		}, DEFAULT_EVENT_PRIORITY)

		event := NewMockEvent("test.event.panic")
		dc := make(chan []error)
		em.Publish(event, dc)

		errs := <-dc

		if len(errs) != 2 {
			t.Errorf("Expected 2 errors but got %d", len(errs))
		}
	})

	t.Run("panic and further processing", func(t *testing.T) {
		em := NewEventManager(logger)

		var received bool

		em.Subscribe("test.event.panic", func(e Event, args *PublishArgs) error {
			panic("test panic")
		}, DEFAULT_EVENT_PRIORITY)

		em.Subscribe("test.event.panic", func(e Event, args *PublishArgs) error {
			received = true
			return nil
		}, DEFAULT_EVENT_PRIORITY)

		event := NewMockEvent("test.event.panic")
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

// TODO: Test to ensure multiple events published concurrently are all processed.
// TODO: Test to ensure multiple subscribers registered concurrently are all invoked when an event is published.
// TODO: Test mixed operations of concurrent publishing and subscribing.
