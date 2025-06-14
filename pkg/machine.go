package warden

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// StateMachine represents the state machine
type StateMachine struct {
	id                string
	name              string
	states            map[string]State
	transitions       map[string]map[TransitionType]string
	startStateID      string
	endStateIDs       []string
	instancePersister InstancePersister
	eventBus          *EventBus
	built             bool
	mutex             sync.RWMutex

	// Configurations
	config *StateMachineConfig
}

// StateMachineConfig state machine configurations
type StateMachineConfig struct {
	EnablePersistence      bool
	EnableEventBus         bool
	MaxConcurrentInstances int
	TransitionTimeout      time.Duration
	RetryAttempts          int
	RetryDelay             time.Duration
}

// DefaultStateMachineConfig returns default configurations
func DefaultStateMachineConfig() *StateMachineConfig {
	return &StateMachineConfig{
		EnablePersistence:      true,
		EnableEventBus:         true,
		MaxConcurrentInstances: 1000,
		TransitionTimeout:      30 * time.Second,
		RetryAttempts:          3,
		RetryDelay:             1 * time.Second,
	}
}

// StateMachineBuilder builder to create state machines
type StateMachineBuilder struct {
	machine *StateMachine
}

// New creates a new builder for state machine
func New(name string) *StateMachineBuilder {
	return &StateMachineBuilder{
		machine: &StateMachine{
			id:          generateID(),
			name:        name,
			states:      make(map[string]State),
			transitions: make(map[string]map[TransitionType]string),
			endStateIDs: make([]string, 0),
			config:      DefaultStateMachineConfig(),
		},
	}
}

// WithConfig sets the machine configuration
func (b *StateMachineBuilder) WithConfig(config *StateMachineConfig) *StateMachineBuilder {
	b.machine.config = config
	return b
}

// WithPersister sets the persister for the machine
func (b *StateMachineBuilder) WithPersister(persister InstancePersister) *StateMachineBuilder {
	b.machine.instancePersister = persister
	return b
}

// WithEventBus sets the event bus for the machine
func (b *StateMachineBuilder) WithEventBus(eventBus *EventBus) *StateMachineBuilder {
	b.machine.eventBus = eventBus
	return b
}

// State adds a state to the machine
func (b *StateMachineBuilder) State(stateID string, state State) *StateMachineBuilder {
	if state == nil {
		// Creates a basic state if none was provided
		switch {
		case stateID == "start":
			state = NewStartState(stateID)
		case stateID == "end":
			state = NewEndState(stateID)
		default:
			state = NewTaskState(stateID, nil)
		}
	}

	b.machine.states[stateID] = state
	b.machine.transitions[stateID] = make(map[TransitionType]string)

	// Sets initial state if it's a Start type
	if state.Type() == StateTypeStart {
		b.machine.startStateID = stateID
	}

	// Adds to end states if it's an End type
	if state.Type() == StateTypeEnd {
		b.machine.endStateIDs = append(b.machine.endStateIDs, stateID)
	}

	return b
}

// StartState defines the initial state
func (b *StateMachineBuilder) StartState(stateID string) *StateMachineBuilder {
	startState := NewStartState(stateID)
	b.machine.states[stateID] = startState
	b.machine.transitions[stateID] = make(map[TransitionType]string)
	b.machine.startStateID = stateID
	return b
}

// TaskState adds a task state
func (b *StateMachineBuilder) TaskState(stateID string, taskFunc func(StateContext) StateResult) *StateMachineBuilder {
	taskState := NewTaskState(stateID, taskFunc)
	b.machine.states[stateID] = taskState
	b.machine.transitions[stateID] = make(map[TransitionType]string)
	return b
}

// DecisionState adds a decision state
func (b *StateMachineBuilder) DecisionState(stateID string, conditionFunc func(StateContext) (TransitionType, error)) *StateMachineBuilder {
	decisionState := NewDecisionState(stateID, conditionFunc)
	b.machine.states[stateID] = decisionState
	b.machine.transitions[stateID] = make(map[TransitionType]string)
	return b
}

// ParallelState adds a parallel state
func (b *StateMachineBuilder) ParallelState(stateID string, tasks []func(StateContext) StateResult) *StateMachineBuilder {
	parallelState := NewParallelState(stateID, tasks)
	b.machine.states[stateID] = parallelState
	b.machine.transitions[stateID] = make(map[TransitionType]string)
	return b
}

// JoinState adds a join state
func (b *StateMachineBuilder) JoinState(stateID string, requiredInputs []string) *StateMachineBuilder {
	joinState := NewJoinState(stateID, requiredInputs)
	b.machine.states[stateID] = joinState
	b.machine.transitions[stateID] = make(map[TransitionType]string)
	return b
}

// EndState adds a final state
func (b *StateMachineBuilder) EndState(stateID string) *StateMachineBuilder {
	endState := NewEndState(stateID)
	b.machine.states[stateID] = endState
	b.machine.transitions[stateID] = make(map[TransitionType]string)
	b.machine.endStateIDs = append(b.machine.endStateIDs, stateID)
	return b
}

// Transition adds a transition between states
func (b *StateMachineBuilder) Transition(fromStateID string, toStateID string, transitionType TransitionType) *StateMachineBuilder {
	if _, exists := b.machine.transitions[fromStateID]; !exists {
		b.machine.transitions[fromStateID] = make(map[TransitionType]string)
	}
	b.machine.transitions[fromStateID][transitionType] = toStateID

	// Adds the transition to the state as well
	if state, exists := b.machine.states[fromStateID]; exists {
		if baseState, ok := state.(*BaseState); ok {
			baseState.AddTransition(transitionType, toStateID)
		}
	}

	return b
}

// Build constructs the state machine
func (b *StateMachineBuilder) Build() (*StateMachine, error) {
	if b.machine.built {
		return nil, ErrMachineAlreadyBuilt
	}

	// Validations
	if err := b.validate(); err != nil {
		return nil, err
	}

	// Sets up default components if not provided
	if b.machine.instancePersister == nil && b.machine.config.EnablePersistence {
		b.machine.instancePersister = NewInMemoryInstancePersister()
	}

	if b.machine.eventBus == nil && b.machine.config.EnableEventBus {
		b.machine.eventBus = NewEventBus()
	}

	b.machine.built = true
	return b.machine, nil
}

// validate validates the machine configuration
func (b *StateMachineBuilder) validate() error {
	if b.machine.name == "" {
		return fmt.Errorf("machine name cannot be empty")
	}

	if len(b.machine.states) == 0 {
		return fmt.Errorf("machine must have at least one state")
	}

	if b.machine.startStateID == "" {
		return fmt.Errorf("machine must have a start state")
	}

	if len(b.machine.endStateIDs) == 0 {
		return fmt.Errorf("machine must have at least one end state")
	}

	// Checks if the initial state exists
	if _, exists := b.machine.states[b.machine.startStateID]; !exists {
		return fmt.Errorf("start state '%s' not found", b.machine.startStateID)
	}

	// Checks if all end states exist
	for _, endStateID := range b.machine.endStateIDs {
		if _, exists := b.machine.states[endStateID]; !exists {
			return fmt.Errorf("end state '%s' not found", endStateID)
		}
	}

	// Checks if all transitions point to existing states
	for fromStateID, transitions := range b.machine.transitions {
		for transitionType, toStateID := range transitions {
			if _, exists := b.machine.states[toStateID]; !exists {
				return fmt.Errorf("transition from '%s' via '%s' points to non-existent state '%s'",
					fromStateID, transitionType, toStateID)
			}
		}
	}

	return nil
}

// StateMachineInstance represents a running instance of the machine
type StateMachineInstance struct {
	ID              string
	ExecutionID     string
	Machine         *StateMachine
	CurrentStateID  string
	Status          InstanceStatus
	Data            map[string]interface{}
	Metadata        map[string]interface{}
	CreatedAt       time.Time
	UpdatedAt       time.Time
	CompletedAt     *time.Time
	Error           error
	mutex           sync.RWMutex
	idempotencyKeys map[string]bool
}

// CreateInstance creates a new machine instance
func (sm *StateMachine) CreateInstance(instanceID string, initialData map[string]interface{}) (*StateMachineInstance, error) {
	if !sm.built {
		return nil, ErrMachineNotBuilt
	}

	if err := validateInstanceID(instanceID); err != nil {
		return nil, err
	}

	executionID := generateExecutionID(instanceID)

	instance := &StateMachineInstance{
		ID:              instanceID,
		ExecutionID:     executionID,
		Machine:         sm,
		CurrentStateID:  sm.startStateID,
		Status:          InstanceStatusCreated,
		Data:            copyMap(initialData),
		Metadata:        make(map[string]interface{}),
		CreatedAt:       getCurrentTimestamp(),
		UpdatedAt:       getCurrentTimestamp(),
		idempotencyKeys: make(map[string]bool),
	}

	// Persists the instance if configured
	if sm.instancePersister != nil {
		snapshot := instance.createSnapshot()
		if err := sm.instancePersister.SaveSnapshot(snapshot); err != nil {
			return nil, fmt.Errorf("failed to persist instance: %w", err)
		}
	}

	// Publishes creation event
	if sm.eventBus != nil {
		event := NewStateEvent(EventExecutionStarted, instanceID, executionID)
		event.StateID = sm.startStateID
		event.Data = copyMap(initialData)
		sm.eventBus.PublishAsync(event)
	}

	return instance, nil
}

// LoadInstance loads an existing instance
func (sm *StateMachine) LoadInstance(instanceID string) (*StateMachineInstance, error) {
	if !sm.built {
		return nil, ErrMachineNotBuilt
	}

	if sm.instancePersister == nil {
		return nil, ErrEventPersisterNotSet
	}

	snapshot, err := sm.instancePersister.LoadSnapshot(instanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to load instance: %w", err)
	}

	instance := &StateMachineInstance{
		ID:              snapshot.InstanceID,
		ExecutionID:     snapshot.ExecutionID,
		Machine:         sm,
		CurrentStateID:  snapshot.CurrentStateID,
		Status:          snapshot.Status,
		Data:            copyMap(snapshot.Data),
		Metadata:        copyMap(snapshot.Metadata),
		CreatedAt:       snapshot.CreatedAt,
		UpdatedAt:       snapshot.UpdatedAt,
		CompletedAt:     snapshot.CompletedAt,
		idempotencyKeys: make(map[string]bool),
	}

	// Copies idempotency keys
	for k, v := range snapshot.IdempotencyKeys {
		instance.idempotencyKeys[k] = v
	}

	if snapshot.Error != nil {
		instance.Error = fmt.Errorf("%s", *snapshot.Error)
	}

	return instance, nil
}

// Start initiates the instance execution
func (instance *StateMachineInstance) Start(ctx context.Context) error {
	instance.mutex.Lock()
	defer instance.mutex.Unlock()

	if instance.Status != InstanceStatusCreated {
		return ErrInstanceNotRunning
	}

	instance.Status = InstanceStatusRunning
	instance.UpdatedAt = getCurrentTimestamp()

	// Persists status change
	if instance.Machine.instancePersister != nil {
		snapshot := instance.createSnapshot()
		if err := instance.Machine.instancePersister.SaveSnapshot(snapshot); err != nil {
			return fmt.Errorf("failed to persist instance: %w", err)
		}
	}

	return instance.executeCurrentState(ctx)
}

// Next advances to the next state based on an event
func (instance *StateMachineInstance) Next(ctx context.Context, event Event) error {
	instance.mutex.Lock()
	defer instance.mutex.Unlock()

	if instance.Status != InstanceStatusRunning {
		return ErrInstanceNotRunning
	}

	// Checks idempotency
	if instance.idempotencyKeys[event.ID] {
		return ErrDuplicateEvent
	}
	instance.idempotencyKeys[event.ID] = true

	// Searches for transition for the event
	transitions, exists := instance.Machine.transitions[instance.CurrentStateID]
	if !exists {
		return ErrTransitionNotFound
	}

	nextStateID, exists := transitions[event.Type]
	if !exists {
		return ErrTransitionNotAllowed
	}

	return instance.transitionTo(ctx, nextStateID, event)
}

// Goto forces a transition to a specific state
func (instance *StateMachineInstance) Goto(ctx context.Context, stateID string) error {
	instance.mutex.Lock()
	defer instance.mutex.Unlock()

	if instance.Status != InstanceStatusRunning {
		return ErrInstanceNotRunning
	}

	// Checks if the state exists
	if _, exists := instance.Machine.states[stateID]; !exists {
		return ErrStateNotFound
	}

	event := NewEvent(TransitionOnCustom, map[string]interface{}{"forced": true})
	return instance.transitionTo(ctx, stateID, event)
}

// transitionTo executes the transition to a state
func (instance *StateMachineInstance) transitionTo(ctx context.Context, nextStateID string, event Event) error {
	currentState := instance.Machine.states[instance.CurrentStateID]
	nextState := instance.Machine.states[nextStateID]

	// Creates state context
	stateContext := StateContext{
		Context:     ctx,
		InstanceID:  instance.ID,
		Data:        copyMap(instance.Data),
		Metadata:    copyMap(instance.Metadata),
		ExecutionID: instance.ExecutionID,
	}

	// Exits current state
	if err := currentState.Exit(stateContext); err != nil {
		return fmt.Errorf("failed to exit state %s: %w", instance.CurrentStateID, err)
	}

	// Publishes exit event
	if instance.Machine.eventBus != nil {
		exitEvent := NewStateEvent(EventStateExited, instance.ID, instance.ExecutionID)
		exitEvent.StateID = instance.CurrentStateID
		exitEvent.FromStateID = instance.CurrentStateID
		exitEvent.ToStateID = nextStateID
		exitEvent.Transition = event.Type
		exitEvent.Data = event.Data
		instance.Machine.eventBus.PublishAsync(exitEvent)
	}

	// Updates current state
	previousStateID := instance.CurrentStateID
	instance.CurrentStateID = nextStateID
	instance.UpdatedAt = getCurrentTimestamp()

	// Merges event data
	if event.Data != nil {
		instance.Data = mergeMap(instance.Data, event.Data)
	}

	// Enters new state
	if err := nextState.Enter(stateContext); err != nil {
		return fmt.Errorf("failed to enter state %s: %w", nextStateID, err)
	}

	// Publishes entry event
	if instance.Machine.eventBus != nil {
		enterEvent := NewStateEvent(EventStateEntered, instance.ID, instance.ExecutionID)
		enterEvent.StateID = nextStateID
		enterEvent.FromStateID = previousStateID
		enterEvent.ToStateID = nextStateID
		enterEvent.Transition = event.Type
		enterEvent.Data = event.Data
		instance.Machine.eventBus.PublishAsync(enterEvent)
	}

	// Checks if it's a final state
	if containsString(instance.Machine.endStateIDs, nextStateID) {
		instance.Status = InstanceStatusCompleted
		now := getCurrentTimestamp()
		instance.CompletedAt = &now

		// Publishes completion event
		if instance.Machine.eventBus != nil {
			completeEvent := NewStateEvent(EventExecutionFinished, instance.ID, instance.ExecutionID)
			completeEvent.StateID = nextStateID
			completeEvent.Data = copyMap(instance.Data)
			instance.Machine.eventBus.PublishAsync(completeEvent)
		}
	}

	// Persists changes
	if instance.Machine.instancePersister != nil {
		snapshot := instance.createSnapshot()
		if err := instance.Machine.instancePersister.SaveSnapshot(snapshot); err != nil {
			return fmt.Errorf("failed to persist instance: %w", err)
		}
	}

	// Executes the new state if not final
	if instance.Status == InstanceStatusRunning {
		return instance.executeCurrentState(ctx)
	}

	return nil
}

// executeCurrentState executes the current state
func (instance *StateMachineInstance) executeCurrentState(ctx context.Context) error {
	currentState := instance.Machine.states[instance.CurrentStateID]

	stateContext := StateContext{
		Context:     ctx,
		InstanceID:  instance.ID,
		Data:        copyMap(instance.Data),
		Metadata:    copyMap(instance.Metadata),
		ExecutionID: instance.ExecutionID,
	}

	// Executes the state
	result := currentState.Execute(stateContext)

	// Updates instance data
	if result.Data != nil {
		instance.Data = mergeMap(instance.Data, result.Data)
	}

	// Handles error
	if !result.Success && result.Error != nil {
		instance.Error = result.Error
		instance.Status = InstanceStatusFailed
		instance.UpdatedAt = getCurrentTimestamp()

		// Publishes failure event
		if instance.Machine.eventBus != nil {
			failEvent := NewStateEvent(EventExecutionFailed, instance.ID, instance.ExecutionID)
			failEvent.StateID = instance.CurrentStateID
			failEvent.Error = result.Error
			failEvent.Data = result.Data
			instance.Machine.eventBus.PublishAsync(failEvent)
		}

		// Persists error
		if instance.Machine.instancePersister != nil {
			snapshot := instance.createSnapshot()
			if err := instance.Machine.instancePersister.SaveSnapshot(snapshot); err != nil {
				return fmt.Errorf("failed to persist instance: %w", err)
			}
		}

		return result.Error
	}

	// Publishes transition event
	if instance.Machine.eventBus != nil {
		transitionEvent := NewStateEvent(EventTransitionFired, instance.ID, instance.ExecutionID)
		transitionEvent.StateID = instance.CurrentStateID
		transitionEvent.Transition = result.Event.Type
		transitionEvent.Data = result.Event.Data
		instance.Machine.eventBus.PublishAsync(transitionEvent)
	}

	// Automatically advances to next state if there's a transition
	if transitions, exists := instance.Machine.transitions[instance.CurrentStateID]; exists {
		if nextStateID, exists := transitions[result.Event.Type]; exists {
			return instance.transitionTo(ctx, nextStateID, result.Event)
		}
	}

	return nil
}

// createSnapshot creates an instance snapshot
func (instance *StateMachineInstance) createSnapshot() *InstanceSnapshot {
	snapshot := &InstanceSnapshot{
		ID:              generateID(),
		InstanceID:      instance.ID,
		ExecutionID:     instance.ExecutionID,
		CurrentStateID:  instance.CurrentStateID,
		Status:          instance.Status,
		Data:            copyMap(instance.Data),
		Metadata:        copyMap(instance.Metadata),
		Version:         1, // Will be incremented by persister
		CreatedAt:       instance.CreatedAt,
		UpdatedAt:       instance.UpdatedAt,
		CompletedAt:     instance.CompletedAt,
		IdempotencyKeys: make(map[string]bool),
	}

	// Copies idempotency keys
	for k, v := range instance.idempotencyKeys {
		snapshot.IdempotencyKeys[k] = v
	}

	if instance.Error != nil {
		errStr := instance.Error.Error()
		snapshot.Error = &errStr
	}

	return snapshot
}

// GetCurrentState returns the current state of the instance
func (instance *StateMachineInstance) GetCurrentState() State {
	instance.mutex.RLock()
	defer instance.mutex.RUnlock()

	return instance.Machine.states[instance.CurrentStateID]
}

// GetAvailableTransitions returns the available transitions from the current state
func (instance *StateMachineInstance) GetAvailableTransitions() map[TransitionType]string {
	instance.mutex.RLock()
	defer instance.mutex.RUnlock()

	if transitions, exists := instance.Machine.transitions[instance.CurrentStateID]; exists {
		result := make(map[TransitionType]string)
		for k, v := range transitions {
			result[k] = v
		}
		return result
	}

	return make(map[TransitionType]string)
}

// IsCompleted checks if the instance was completed
func (instance *StateMachineInstance) IsCompleted() bool {
	instance.mutex.RLock()
	defer instance.mutex.RUnlock()

	return instance.Status == InstanceStatusCompleted
}

// IsFailed checks if the instance failed
func (instance *StateMachineInstance) IsFailed() bool {
	instance.mutex.RLock()
	defer instance.mutex.RUnlock()

	return instance.Status == InstanceStatusFailed
}

// IsRunning checks if the instance is running
func (instance *StateMachineInstance) IsRunning() bool {
	instance.mutex.RLock()
	defer instance.mutex.RUnlock()

	return instance.Status == InstanceStatusRunning
}

// GetData returns the instance data
func (instance *StateMachineInstance) GetData() map[string]interface{} {
	instance.mutex.RLock()
	defer instance.mutex.RUnlock()

	return copyMap(instance.Data)
}

// SetData sets data in the instance
func (instance *StateMachineInstance) SetData(key string, value interface{}) {
	instance.mutex.Lock()
	defer instance.mutex.Unlock()

	if instance.Data == nil {
		instance.Data = make(map[string]interface{})
	}
	instance.Data[key] = value
	instance.UpdatedAt = getCurrentTimestamp()
}

// GetMetadata returns the instance metadata
func (instance *StateMachineInstance) GetMetadata() map[string]interface{} {
	instance.mutex.RLock()
	defer instance.mutex.RUnlock()

	return copyMap(instance.Metadata)
}

// SetMetadata sets metadata in the instance
func (instance *StateMachineInstance) SetMetadata(key string, value interface{}) {
	instance.mutex.Lock()
	defer instance.mutex.Unlock()

	if instance.Metadata == nil {
		instance.Metadata = make(map[string]interface{})
	}
	instance.Metadata[key] = value
	instance.UpdatedAt = getCurrentTimestamp()
}
