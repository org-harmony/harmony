package core

import (
	"fmt"
	"sort"
	"sync"

	"github.com/org-harmony/harmony/trace"
)

const EVENT_MOD = "sys.core.event"

// EVENT_BUFFER_SIZE is the size of the buffer for event channels.
// The buffer size is used when creating channels for events.
// If the buffer size is too small publishing an event will block until the event is handled.
// If the buffer size is too large publishing an event will use more memory.
const EVENT_BUFFER_SIZE = 100

// Event is an interface that makes a type publishable through the EventManager.
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

// Subscriber is a struct that holds information about a subscriber.
type Subscriber struct {
	// eventID that the subscriber is subscribed to.
	eventID string
	// publish function that is called when the event is published.
	publish func(Event, *PublishArgs) error
	// priority of the subscriber.
	// The priority is used to determine the order in which subscribers are called.
	//
	// A higher priority means that the subscriber is called earlier.
	priority int
}

// EventManager is a struct that manages events and their subscribers.
// The EventManager is safe to use concurrently and pass to multiple goroutines.
type EventManager struct {
	mu sync.Mutex
	// events is a map of event IDs to channels.
	// The channels are used to publish events to subscribers.
	events map[string]chan pc
	// subscriber is a map of event IDs to subscribers.
	// The subscribers are called when an event is published.
	subscriber map[string][]Subscriber
	logger     trace.Logger
}

// pc (publish container) is a struct that holds information about a published event.
// It captures the event and the subscribers that are subscribed to the event.
type pc struct {
	e Event
	s []Subscriber
}

// PublishArgs is a struct that holds arguments that are passed to subscribers when an event is published.
type PublishArgs struct {
	// StopPropagation is a boolean that can be set to true to stop the propagation of an event.
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

// NewEventManager creates a new event manager. ö
func NewEventManager(l trace.Logger) *EventManager {
	return &EventManager{
		events:     make(map[string]chan pc),
		subscriber: make(map[string][]Subscriber),
		logger:     l,
	}
}

// Subscribe subscribes to an event with the given event ID.
func (em *EventManager) Subscribe(eventID string, publish func(Event, *PublishArgs) error, priority int) {
	em.mu.Lock()
	defer em.mu.Unlock()

	subscriber := Subscriber{
		eventID:  eventID,
		publish:  publish,
		priority: priority,
	}

	em.subscriber[eventID] = append(em.subscriber[eventID], subscriber)

	// sort subscribers by ascending priority
	sort.Slice(em.subscriber[eventID], func(i, j int) bool {
		return em.subscriber[eventID][i].priority < em.subscriber[eventID][j].priority
	})

	em.logger.Debug(EVENT_MOD, "subscribed to event", "eventID", eventID, "priority", priority)
}

// Publish publishes an event to the event's channel.
// Leading to the event being handled by the subscribers of the event within a separate goroutine.
// Therefore, the Publish function is non-blocking.
//
// Callers can use the done channel to wait for the event to be handled.
// Through the done channel the caller may retrieve any errors from execution of the subscribers.
//
// Awaiting the done channel is optional but the only way to be sure all subscribers have handled the
// event before proceeding. Still, it is very viable to not wait for the done channel and continue:
// "fire and forget".
//
// Furthermore, Events are lazily registered.
// Meaning the channel for event publishing is created when the event is published for the first time.
//
// If a nil event is passed to the Publish function, the function will return immediately.
func (em *EventManager) Publish(event Event, doneChan chan []error) {
	if event == nil {
		return
	}

	em.logger.Debug(EVENT_MOD, "publishing event", "eventID", event.ID())

	em.mu.Lock()
	defer em.mu.Unlock()

	if _, exists := em.events[event.ID()]; !exists {
		em.register(event, doneChan)
	}

	em.events[event.ID()] <- pc{
		e: event,
		s: em.subscriber[event.ID()],
	}

	em.logger.Debug(EVENT_MOD, "published event", "eventID", event.ID())
}

// register registers an event with the event manager and creates a channel for the event.
// Also, register boots up a goroutine to handle published events for the event ID.
//
// Register is *NOT* safe to call concurrently. It is expected that the caller locks the event manager beforehand.
func (em *EventManager) register(e Event, doneChan chan []error) {
	if _, exists := em.events[e.ID()]; exists {
		return
	}

	// create a buffered channel to publish events to
	em.events[e.ID()] = make(chan pc, EVENT_BUFFER_SIZE)

	// start a goroutine to handle published events for a given event ID through the channel
	go handle(em.events[e.ID()], em.logger, doneChan)

	em.logger.Debug(EVENT_MOD, "registered event and created channel", "eventID", e.ID())
}

// handle handles events published to the given channel.
// Through the channel the handle function receives a [pc] and published the event to the subscribers.
// If the done channel is not nil, the handle function will signal that the event has been handled through the done channel.
// After the event has been handled, the done channel is closed.
func handle(e chan pc, l trace.Logger, doneChan chan []error) {
	for {
		pc := <-e

		l.Debug(EVENT_MOD, "handling event", "eventID", pc.e.ID())

		var errs []error
		args := &PublishArgs{}

		// publish event to subscribers
		for _, subscriber := range pc.s {
			if args.StopPropagation {
				l.Debug(EVENT_MOD, "stopping propagation of event", "eventID", pc.e.ID())
				break
			}

			err := safePublish(subscriber, pc.e, args)
			if err != nil {
				errs = append(errs, err)
			}
		}

		l.Info(EVENT_MOD, fmt.Sprintf("handled event with %d error(s)", len(errs)), "eventID", pc.e.ID(), "errors", errs)

		if doneChan == nil {
			l.Debug(EVENT_MOD, "no done channel for event", "eventID", pc.e.ID())
			return
		}

		// signal that the event has been handled
		doneChan <- errs
		close(doneChan)
	}
}

// safePublish is a wrapper around the publish function of a subscriber.
// It recovers from panics in the subscriber and returns an error if a panic occurred.
func safePublish(s Subscriber, e Event, args *PublishArgs) error {
	// recover from panics in subscribers
	defer func() error {
		if r := recover(); r != nil {
			return fmt.Errorf("subscriber panicked: %v", r)
		}
		return nil
	}()
	return s.publish(e, args)
}
