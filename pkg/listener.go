package warden

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// EventType represents the types of events
type EventType string

const (
	EventExecutionStarted  EventType = "execution_started"
	EventExecutionFinished EventType = "execution_finished"
	EventExecutionFailed   EventType = "execution_failed"
	EventStateEntered      EventType = "state_entered"
	EventStateExited       EventType = "state_exited"
	EventTransitionFired   EventType = "transition_fired"
)

// StateEvent represents an event in the state machine
type StateEvent struct {
	ID          string                 `json:"id"`
	Type        EventType              `json:"type"`
	InstanceID  string                 `json:"instance_id"`
	ExecutionID string                 `json:"execution_id"`
	StateID     string                 `json:"state_id,omitempty"`
	FromStateID string                 `json:"from_state_id,omitempty"`
	ToStateID   string                 `json:"to_state_id,omitempty"`
	Transition  TransitionType         `json:"transition,omitempty"`
	Data        map[string]interface{} `json:"data,omitempty"`
	Error       error                  `json:"error,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
}

// NewStateEvent creates a new state event
func NewStateEvent(eventType EventType, instanceID, executionID string) StateEvent {
	return StateEvent{
		ID:          generateID(),
		Type:        eventType,
		InstanceID:  instanceID,
		ExecutionID: executionID,
		Data:        make(map[string]interface{}),
		Timestamp:   time.Now(),
	}
}

// EventListener interface for event listeners
type EventListener interface {
	// HandleEvent processes an event
	HandleEvent(event StateEvent) error

	// GetID returns the listener ID
	GetID() string

	// IsActive returns whether the listener is active
	IsActive() bool

	// Close closes the listener
	Close() error
}

// EventBus manages event listeners and distributes events
type EventBus struct {
	listeners map[string]EventListener
	channels  map[string]chan StateEvent
	mutex     sync.RWMutex
	closed    bool
}

// NewEventBus creates a new event bus
func NewEventBus() *EventBus {
	return &EventBus{
		listeners: make(map[string]EventListener),
		channels:  make(map[string]chan StateEvent),
	}
}

// Subscribe adds a listener to the event bus
func (eb *EventBus) Subscribe(listener EventListener) error {
	if listener == nil {
		return fmt.Errorf("listener cannot be nil")
	}

	eb.mutex.Lock()
	defer eb.mutex.Unlock()

	if eb.closed {
		return fmt.Errorf("event bus is closed")
	}

	listenerID := listener.GetID()
	if _, exists := eb.listeners[listenerID]; exists {
		return fmt.Errorf("listener with ID %s already exists", listenerID)
	}

	eb.listeners[listenerID] = listener
	eb.channels[listenerID] = make(chan StateEvent, 100) // Buffer of 100 events

	// Starts goroutine to process events
	go eb.processEvents(listenerID)

	return nil
}

// Unsubscribe removes a listener from the event bus
func (eb *EventBus) Unsubscribe(listenerID string) error {
	eb.mutex.Lock()
	defer eb.mutex.Unlock()

	listener, exists := eb.listeners[listenerID]
	if !exists {
		return fmt.Errorf("listener with ID %s not found", listenerID)
	}

	// Closes the listener
	if err := listener.Close(); err != nil {
		log.Printf("Error closing listener %s: %v", listenerID, err)
	}

	// Closes the channel
	if channel, exists := eb.channels[listenerID]; exists {
		close(channel)
	}

	delete(eb.listeners, listenerID)
	return nil
}

// Publish publishes an event synchronously
func (eb *EventBus) Publish(event StateEvent) error {
	eb.mutex.RLock()
	defer eb.mutex.RUnlock()

	if eb.closed {
		return fmt.Errorf("event bus is closed")
	}

	var errors []error

	for listenerID, listener := range eb.listeners {
		if !listener.IsActive() {
			continue
		}

		if err := listener.HandleEvent(event); err != nil {
			errors = append(errors, fmt.Errorf("listener %s error: %w", listenerID, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("multiple listener errors: %v", errors)
	}

	return nil
}

// PublishAsync publishes an event asynchronously
func (eb *EventBus) PublishAsync(event StateEvent) {
	eb.mutex.RLock()
	defer eb.mutex.RUnlock()

	if eb.closed {
		return
	}

	for listenerID, listener := range eb.listeners {
		if !listener.IsActive() {
			continue
		}

		if channel, exists := eb.channels[listenerID]; exists {
			select {
			case channel <- event:
				// Event sent successfully
			default:
				// Channel full, logs warning
				log.Printf("Warning: Channel for listener %s is full, dropping event %s", listenerID, event.ID)
			}
		}
	}
}

// processEvents processes events for a listener
func (eb *EventBus) processEvents(listenerID string) {
	eb.mutex.RLock()
	listener := eb.listeners[listenerID]
	channel := eb.channels[listenerID]
	eb.mutex.RUnlock()

	if listener == nil || channel == nil {
		return
	}

	for event := range channel {
		if !listener.IsActive() {
			continue
		}

		if err := listener.HandleEvent(event); err != nil {
			log.Printf("Error processing event %s in listener %s: %v", event.ID, listenerID, err)
		}
	}
}

// GetListeners returns all active listeners
func (eb *EventBus) GetListeners() []EventListener {
	eb.mutex.RLock()
	defer eb.mutex.RUnlock()

	var result []EventListener
	for _, listener := range eb.listeners {
		if listener.IsActive() {
			result = append(result, listener)
		}
	}

	return result
}

// Close closes the event bus and all listeners
func (eb *EventBus) Close() error {
	eb.mutex.Lock()
	defer eb.mutex.Unlock()

	if eb.closed {
		return nil
	}

	eb.closed = true

	var errors []error

	// Closes all listeners
	for listenerID, listener := range eb.listeners {
		if err := listener.Close(); err != nil {
			errors = append(errors, fmt.Errorf("error closing listener %s: %w", listenerID, err))
		}

		// Closes the channel
		if channel, exists := eb.channels[listenerID]; exists {
			close(channel)
		}
	}

	// Clears maps
	eb.listeners = make(map[string]EventListener)
	eb.channels = make(map[string]chan StateEvent)

	if len(errors) > 0 {
		return fmt.Errorf("multiple errors closing event bus: %v", errors)
	}

	return nil
}

// BaseEventListener provides base implementation for listeners
type BaseEventListener struct {
	id     string
	active bool
	mutex  sync.RWMutex
}

// NewBaseEventListener creates a new base listener
func NewBaseEventListener(id string) *BaseEventListener {
	return &BaseEventListener{
		id:     id,
		active: true,
	}
}

func (l *BaseEventListener) GetID() string {
	return l.id
}

func (l *BaseEventListener) IsActive() bool {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	return l.active
}

func (l *BaseEventListener) Close() error {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.active = false
	return nil
}

func (l *BaseEventListener) HandleEvent(event StateEvent) error {
	// Base implementation - does nothing
	return nil
}

// MetricsListener listener for collecting metrics
type MetricsListener struct {
	*BaseEventListener
	metrics map[string]interface{}
	mutex   sync.RWMutex
}

// NewMetricsListener creates a new metrics listener
func NewMetricsListener() *MetricsListener {
	return &MetricsListener{
		BaseEventListener: NewBaseEventListener("metrics-listener"),
		metrics:           make(map[string]interface{}),
	}
}

func (l *MetricsListener) HandleEvent(event StateEvent) error {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if !l.IsActive() {
		return nil
	}

	// Collects basic metrics
	switch event.Type {
	case EventExecutionStarted:
		l.incrementCounter("executions_started")

	case EventExecutionFinished:
		l.incrementCounter("executions_completed")

	case EventExecutionFailed:
		l.incrementCounter("executions_failed")

	case EventStateEntered:
		l.incrementCounter(fmt.Sprintf("state_entered_%s", event.StateID))

	case EventTransitionFired:
		l.incrementCounter(fmt.Sprintf("transition_%s", event.Transition))
	}

	return nil
}

// incrementCounter increments a counter
func (l *MetricsListener) incrementCounter(key string) {
	if current, exists := l.metrics[key]; exists {
		if count, ok := current.(int64); ok {
			l.metrics[key] = count + 1
		}
	} else {
		l.metrics[key] = int64(1)
	}
}

// GetMetrics returns the collected metrics
func (l *MetricsListener) GetMetrics() map[string]interface{} {
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	result := make(map[string]interface{})
	for k, v := range l.metrics {
		result[k] = v
	}

	return result
}

// Reset resets all metrics
func (l *MetricsListener) Reset() {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	l.metrics = make(map[string]interface{})
}

// LoggingListener listener for logging events
type LoggingListener struct {
	*BaseEventListener
	logLevel string
}

// NewLoggingListener creates a new logging listener
func NewLoggingListener(logLevel string) *LoggingListener {
	return &LoggingListener{
		BaseEventListener: NewBaseEventListener("logging-listener"),
		logLevel:          logLevel,
	}
}

func (l *LoggingListener) HandleEvent(event StateEvent) error {
	if !l.IsActive() {
		return nil
	}

	// Logs the event
	switch event.Type {
	case EventExecutionStarted:
		log.Printf("[INFO] Execution started: instance=%s, execution=%s",
			event.InstanceID, event.ExecutionID)

	case EventExecutionFinished:
		log.Printf("[INFO] Execution finished: instance=%s, execution=%s",
			event.InstanceID, event.ExecutionID)

	case EventExecutionFailed:
		log.Printf("[ERROR] Execution failed: instance=%s, execution=%s, error=%v",
			event.InstanceID, event.ExecutionID, event.Error)

	case EventStateEntered:
		log.Printf("[DEBUG] State entered: instance=%s, state=%s",
			event.InstanceID, event.StateID)

	case EventStateExited:
		log.Printf("[DEBUG] State exited: instance=%s, state=%s",
			event.InstanceID, event.StateID)

	case EventTransitionFired:
		log.Printf("[DEBUG] Transition fired: instance=%s, state=%s, transition=%s",
			event.InstanceID, event.StateID, event.Transition)
	}

	return nil
}

// PersistenceListener listener for persisting events
type PersistenceListener struct {
	*BaseEventListener
	eventPersister EventPersister
}

// NewPersistenceListener creates a new persistence listener
func NewPersistenceListener(eventPersister EventPersister) *PersistenceListener {
	return &PersistenceListener{
		BaseEventListener: NewBaseEventListener("persistence-listener"),
		eventPersister:    eventPersister,
	}
}

func (l *PersistenceListener) HandleEvent(event StateEvent) error {
	if !l.IsActive() {
		return nil
	}

	if l.eventPersister == nil {
		return fmt.Errorf("event persister not set")
	}

	return l.eventPersister.SaveEvent(event)
}

// EventPersister interface for persisting events
type EventPersister interface {
	// SaveEvent saves an event
	SaveEvent(event StateEvent) error

	// GetEvents returns all events for an instance
	GetEvents(instanceID string) ([]StateEvent, error)

	// GetEventsByType returns events of a specific type
	GetEventsByType(instanceID string, eventType EventType) ([]StateEvent, error)

	// DeleteEvents removes all events for an instance
	DeleteEvents(instanceID string) error
}

// InMemoryEventPersister in-memory implementation of event persister
type InMemoryEventPersister struct {
	events map[string][]StateEvent // instanceID -> events
	mutex  sync.RWMutex
}

// NewInMemoryEventPersister creates a new in-memory event persister
func NewInMemoryEventPersister() *InMemoryEventPersister {
	return &InMemoryEventPersister{
		events: make(map[string][]StateEvent),
	}
}

func (p *InMemoryEventPersister) SaveEvent(event StateEvent) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if _, exists := p.events[event.InstanceID]; !exists {
		p.events[event.InstanceID] = make([]StateEvent, 0)
	}

	p.events[event.InstanceID] = append(p.events[event.InstanceID], event)
	return nil
}

func (p *InMemoryEventPersister) GetEvents(instanceID string) ([]StateEvent, error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	events, exists := p.events[instanceID]
	if !exists {
		return []StateEvent{}, nil
	}

	// Returns a copy to avoid external modifications
	result := make([]StateEvent, len(events))
	copy(result, events)

	return result, nil
}

func (p *InMemoryEventPersister) GetEventsByType(instanceID string, eventType EventType) ([]StateEvent, error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	events, exists := p.events[instanceID]
	if !exists {
		return []StateEvent{}, nil
	}

	var result []StateEvent
	for _, event := range events {
		if event.Type == eventType {
			result = append(result, event)
		}
	}

	return result, nil
}

func (p *InMemoryEventPersister) DeleteEvents(instanceID string) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	delete(p.events, instanceID)
	return nil
}
