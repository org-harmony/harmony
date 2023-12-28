// Package event provides an event manager that allows for events to be published and subscribed to.
package event

import (
	"fmt"
	"github.com/org-harmony/harmony/src/core/trace"
	"sort"
	"sync"
)

const Pkg = "sys.event"

// DefaultPriority can be used as a general default priority for an event subscriber.
// The priority is used to determine the order in which subscribers are called.
// A higher priority means that the subscriber is called earlier.
// If you do not care about the order in which subscribers are called, use this constant.
const DefaultPriority = 0

// BufferSize is the size of the buffer for event channels.
// The buffer size is used when creating channels for events.
// If the buffer size is too small publishing an event will block until the event is handled.
// If the buffer size is too large publishing an event will use more memory.
const BufferSize = 100

// Event makes a type publishable through the HManager.
type Event interface {
	// ID returns the unique ID of the event.
	// This ID is used to identify the event and route it to the correct subscribers.
	// The ID should be unique across all events.
	//
	// The ID should be built using the [BuildEventID] function.
	ID() string
	// Payload returns the payload of the event. The payload can be any type.
	// The payload is, as part of the event, passed to the subscribers of an event.
	// The payload can be used to pass data to subscribers.
	//
	// The payload can also be pointer and allow subscribers to modify the data.
	// While this is absolutely viable in many situations it can be a potential risk to be aware of.
	// Deep copying or not passing a reference as the payload is a way to mitigate this risk.
	Payload() any
}

// Manager manages events and their subscribers.
type Manager interface {
	// Subscribe subscribes to an event with the given event ID.
	// The publish function is called when the event is published.
	// The priority is used to determine the order in which subscribers are called.
	Subscribe(eventID string, publish func(Event, *PublishArgs) error, priority int)
	// Publish publishes an event and allows for errors to be returned through the done channel.
	Publish(event Event, doneChan chan []error)
}

// subscriber is a struct that holds information about a subscriber.
type subscriber struct {
	// eventID that the subscriber is subscribed to.
	eventID string
	// publish function that is called when the event is published.
	publish func(Event, *PublishArgs) error
	//priority is used to determine the order in which subscribers are called.
	//
	// A higher priority means that the subscriber is called earlier.
	priority int
}

// pc (publish container) holds information about a published event.
// It captures the event, subscribers that are subscribed to the event and the done channel.
type pc struct {
	e  Event
	s  []subscriber
	dc chan []error
}

// PublishArgs holds arguments that are passed to subscribers when an event is published.
type PublishArgs struct {
	// StopPropagation can be set to true to stop the propagation of an event.
	// When set to true, the event manager will stop calling subscribers for the event.
	// Stopping propagation will be logged.
	StopPropagation bool
}

// BuildEventID builds an event ID from the given module, namespace and action.
// The event ID is built using the following format: <module>.<namespace>.<action>
//
// Example: "sys.auth.user.repository.created" or "sys.mailer.send.std-service.sent"
func BuildEventID(module, namespace, action string) string {
	return fmt.Sprintf("%s.%s.%s", module, namespace, action)
}

// HManager is the standard implementation of the Manager interface.
// The HManager is safe to use concurrently and pass to multiple goroutines.
type HManager struct {
	mu sync.Mutex
	// events is a map of event IDs to channels.
	// The channels are used to publish events to subscribers.
	events map[string]chan pc
	// subscriber is a map of event IDs to subscribers.
	// The subscribers are called when an event is published.
	subscriber map[string][]subscriber
	logger     trace.Logger
}

// NewManager creates a new event manager.
func NewManager(l trace.Logger) *HManager {
	return &HManager{
		events:     make(map[string]chan pc),
		subscriber: make(map[string][]subscriber),
		logger:     l,
	}
}

// Subscribe subscribes to an event with the given event ID.
func (em *HManager) Subscribe(eventID string, publish func(Event, *PublishArgs) error, priority int) {
	em.mu.Lock()
	defer em.mu.Unlock()

	subscriber := subscriber{
		eventID:  eventID,
		publish:  publish,
		priority: priority,
	}

	em.subscriber[eventID] = append(em.subscriber[eventID], subscriber)

	// sort subscribers by ascending priority
	sort.Slice(em.subscriber[eventID], func(i, j int) bool {
		return em.subscriber[eventID][i].priority < em.subscriber[eventID][j].priority
	})

	em.logger.Debug(Pkg, "subscribed to event", "eventID", eventID, "priority", priority)
}

// Publish publishes an event to the event's channel.
// Leading to the event being handled by the subscribers of the event within a separate goroutine.
// Therefore, the Publish function is non-blocking.
//
// Callers can use the done channel to wait for the event to be handled.
// Through the done channel the caller may retrieve any errors from execution of the subscribers.
//
// Awaiting the done channel is optional but the only way to be sure all subscribers have handled the
// event before proceeding. Still, it is very viable not to wait for the done channel and continue:
// "fire and forget".
//
// Furthermore, Events are lazily registered.
// Meaning the channel for event publishing is created when the event is published for the first time.
//
// If a nil event is passed to the Publish function, the function will return immediately.
func (em *HManager) Publish(event Event, doneChan chan []error) {
	if event == nil {
		return
	}

	em.logger.Debug(Pkg, "publishing event", "eventID", event.ID())

	em.mu.Lock()
	defer em.mu.Unlock()

	if _, exists := em.events[event.ID()]; !exists {
		em.register(event)
	}

	em.events[event.ID()] <- pc{
		e:  event,
		s:  em.subscriber[event.ID()],
		dc: doneChan,
	}

	em.logger.Debug(Pkg, "published event", "eventID", event.ID())
}

// register registers an event with the event manager and creates a channel for the event.
// Also, register boots up a goroutine to handle published events for the event ID.
//
// Register is *NOT* safe to call concurrently. It is expected that the caller locks the event manager beforehand.
func (em *HManager) register(e Event) {
	if _, exists := em.events[e.ID()]; exists {
		return
	}

	// create a buffered channel to publish events to
	em.events[e.ID()] = make(chan pc, BufferSize)

	// start a goroutine to handle published events for a given event ID through the channel
	go handle(em.events[e.ID()], em.logger)

	em.logger.Debug(Pkg, "registered event and created channel", "eventID", e.ID())
}

// handle handles events published to the given channel.
// Through the channel the handle function receives a [pc] and publishes the event to the subscribers.
// If the done channel is not nil, the handle function will signal that the event has been handled through the done channel.
// After the event has been handled, the done channel is closed.
func handle(e chan pc, l trace.Logger) {
	for {
		pc := <-e

		l.Debug(Pkg, "handling event", "eventID", pc.e.ID())

		var errs []error
		args := &PublishArgs{}

		// publish event to subscribers
		for _, subscriber := range pc.s {
			if args.StopPropagation {
				l.Debug(Pkg, "stopping propagation of event", "eventID", pc.e.ID())
				break
			}

			err := safePublish(subscriber, pc.e, args)
			if err != nil {
				errs = append(errs, err)
			}
		}

		if len(errs) > 0 {
			l.Info(Pkg, fmt.Sprintf("handled event with %d error(s)", len(errs)), "eventID", pc.e.ID(), "errors", errs)
		} else {
			l.Debug(Pkg, "handled event without errors", "eventID", pc.e.ID())
		}

		dc := pc.dc
		if dc == nil {
			l.Debug(Pkg, "no done channel for event", "eventID", pc.e.ID())
			return
		}

		// signal that the event has been handled
		dc <- errs
		close(dc)
	}
}

// safePublish is a wrapper around the publish function of a subscriber.
// It recovers from panics in the subscriber and returns an error if a panic occurred.
func safePublish(s subscriber, e Event, args *PublishArgs) (err error) {
	// recover from panics in subscribers
	// the named return value err is necessary to return the error from the deferred function,
	// as the return value from the deferred function is discarded
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("subscriber panicked: %v", r)
		}
	}()
	return s.publish(e, args)
}
