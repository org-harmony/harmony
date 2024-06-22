// Package event provides an event manager that allows others to publish and listen to events.
// Events are the primary means of (decoupled) communication between components in HARMONY.
package event

// TODO events for informing, services for returning stuff

import (
	"context"
	"errors"
	"sort"
	"sync"
)

var (
	// ErrDuplicateRegistration might indicate that the Bus has a duplicate registration for an Event or Listener.
	ErrDuplicateRegistration = errors.New("attempt for duplicate registration")
	// ErrNotFound might indicate that a Listener or Event is not found in the Bus.
	ErrNotFound = errors.New("item not found")
)

// ID is a unique identification for Event and Listener.
type ID string

// Event is something that happens.
// Anyone can publish an event using the Bus. Anyone can listen to events using the Bus.
//
// A listener can use the event's payload to:
//  1. get information about something that happened. E.g. the username of someone who just logged in.
//  2. modify referenced data that the publisher of the event passed in as a payload.
//  3. do nothing, in that case the publisher might pass a nil-payload.
//
// Event contains meta information for better debugging and DX of the Bus.
// Theoretically, the Bus could function without registering events and listeners first.
type Event struct {
	ID   ID
	Name string
	Desc string
}

// Listener reacts to an Event after the Bus informs them. The Bus uses the listener's ID to identify them.
// The Bus uses the listener's event ID to inform them when some component publishes the corresponding event.
//
// Bus executes listener's Func and passes the payload on. If the payload is a reference, the listener can modify
// its content. For that reason async listeners (not yet added!) will/should not gain access to payload data via
// references to prevent unexpected behaviour because sequential execution of listeners will not be guaranteed.
//
// Bus uses Prio to prioritize listeners when calling their Func.
//
// Listener can stop further propagation of the event by setting StopProp to true. This can be helpful
// when a Listener should disable all following listeners e.g. to overwrite functionality.
// However, handle StopProp with great care. With great power comes great responsibility!
type Listener struct {
	ID       ID
	EventID  ID
	Func     func(context.Context, any) error
	Prio     int
	StopProp bool
	// TODO add support for async listeners => events is fired, listener is sent to separate queue and processed
	//  asynchronously, errors are logged and further ignored for async listeners.
	//  Important: No references! Sequential processing can not be guaranteed.
	// Async    bool
}

// HBus is HARMONY's standard implementation of the Bus interface.
//
// HBus saves events in a map by event ID. HBus saves listeners in a map of lists of listeners by event ID.
// It also maps the listener ID to the event ID for finding a Listeners in the corresponding list by event ID
// through lstToEvt.
//
// HBus is safe for concurrent use by multiple goroutines.
//
// See Bus for more details.
type HBus struct {
	events   map[ID]Event
	lstByEvt map[ID][]Listener
	lstToEvt map[ID]ID
	mu       sync.RWMutex
}

// Bus allows anyone with access to register events and attach listeners for them.
// Components can publish an event with a context, the event ID, any payload and an error channel.
//
// If you pass a nil error channel to publish it proceeds in a fire and forget manner.
// Otherwise, you can wait for event processing to finish by idling until the Bus closes the error channel.
type Bus interface {
	Listener(ID) (Listener, bool)
	Event(ID) (Event, bool)
	ListenersForEvent(ID) ([]Listener, bool)
	Attach(Listener) error
	Register(Event) error
	Publish(context.Context, ID, any, chan<- error) error
}

// Listener looks up the event listener by its ID in the Bus.
func (b *HBus) Listener(id ID) (Listener, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	evtID, ok := b.lstToEvt[id]
	if !ok {
		return Listener{}, false
	}

	lsts, ok := b.lstByEvt[evtID]
	if !ok {
		return Listener{}, false
	}

	for _, lst := range lsts {
		if lst.ID != id {
			continue
		}

		return lst, true
	}

	return Listener{}, false
}

// Event looks up the event by its ID in the Bus.
func (b *HBus) Event(id ID) (Event, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	evt, ok := b.events[id]

	return evt, ok
}

// ListenersForEvent looks up all event listeners for a certain event ID in the Bus.
// Function returns false when the Bus does not contain the event ID.
// Listeners are copied and not passed by reference.
func (b *HBus) ListenersForEvent(id ID) ([]Listener, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	lsts, ok := b.lstByEvt[id]
	if !ok {
		return nil, false
	}

	listeners := make([]Listener, len(lsts))
	copy(listeners, lsts)

	return listeners, true
}

// Attach registers an event listener with the Bus. If the Bus already contains a listener with the same ID,
// the function returns ErrDuplicateRegistration. Function saves listeners for an event in a list of listeners.
// Function sorts listeners in this list descending by their Listener.Prio (higher prio, earlier call).
//
// Attach indexes a lookup array of listener IDs to event IDs for easier lookup, as it saves listeners in a list
// by their corresponding event ID.
func (b *HBus) Attach(listener Listener) error {
	if _, exists := b.Listener(listener.ID); exists {
		return ErrDuplicateRegistration
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	listeners, ok := b.lstByEvt[listener.EventID]
	if !ok {
		listeners = make([]Listener, 1)
	}

	listeners = append(listeners, listener)
	sort.Slice(listeners, func(i, j int) bool { return listeners[i].Prio > listeners[j].Prio })

	b.lstByEvt[listener.EventID] = listeners
	b.lstToEvt[listener.ID] = listener.EventID

	return nil
}

// Register registers an Event with the Bus and returns ErrDuplicateRegistration if someone already registered the same Event.
func (b *HBus) Register(event Event) error {
	if _, exists := b.Event(event.ID); exists {
		return ErrDuplicateRegistration
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.events[event.ID] = event

	return nil
}

// Publish informs all Listener`s of an Event in their sequential order by priority that something happened.
// Function does not recalculate the order, Bus does this when it Attach`es a Listener.
//
// Returns ErrNotFound if Bus does not know an Event with the ID you passed in.
//
// Publish passes the ctx and payload to Listener.Func. If func returns an error, publish propagates it through
// the error channel. Publish closes the error channel when it is done processing the Event.
// TODO maybe add some sort of statistics for publishing?
func (b *HBus) Publish(ctx context.Context, id ID, payload any, errs chan<- error) error {
	if _, exists := b.Event(id); !exists {
		return ErrNotFound
	}

	lsts, exist := b.ListenersForEvent(id)
	if !exist {
		if errs != nil {
			close(errs)
		}

		return nil
	}

	go func() {
		for _, lst := range lsts {
			err := lst.Func(ctx, payload)

			if err != nil {
				errs <- err
			}

			if lst.StopProp {
				break
			}
		}

		close(errs)
	}()

	return nil
}
