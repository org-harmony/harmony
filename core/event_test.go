package core

import (
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

func TestEventRegistrationAndPublishing(t *testing.T) {
	logger := trace.NewTestLogger(t)

	t.Run("event registration and publishing", func(t *testing.T) {
		em := NewEventManager(logger)

		received := false
		subscriberFunc := func(e Event, args *PublishArgs) error {
			if e.Payload().(*mockPayload).data != "test" {
				t.Error("Received incorrect event payload data")
			}
			received = true
			return nil
		}

		em.Subscribe("test.event", subscriberFunc, 1)

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

		em.Subscribe("test.event.multiple", subscriberFunc, 1)
		em.Subscribe("test.event.multiple", subscriberFunc, 2)

		event := NewMockEvent("test.event.multiple")
		dc := make(chan []error)
		em.Publish(event, dc)

		<-dc

		if count != 2 {
			t.Errorf("Expected both subscribers to process the event, but only %d did", count)
		}
	})
}

// TODO: Test to ensure subscribers are invoked based on their priority.
// TODO: Test to ensure propagation is stopped when a subscriber sets StopPropagation.
// TODO: Test to ensure errors returned by subscribers are propagated correctly.
// TODO: Test to ensure that panics in subscribers are recovered and don't crash the event manager.
// TODO: Test to ensure multiple events published concurrently are all processed.
// TODO: Test to ensure multiple subscribers registered concurrently are all invoked when an event is published.
// TODO: Test mixed operations of concurrent publishing and subscribing.
// TODO: Test a subscriber's ability to modify the payload of an event and ensure the changes are reflected.
// TODO: Test publishing an event with a nil done channel.
// TODO: Test to ensure the done channel is closed after usage.
