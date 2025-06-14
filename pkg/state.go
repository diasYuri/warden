package warden

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// StateType represents the supported state types
type StateType string

const (
	StateTypeStart    StateType = "start"
	StateTypeTask     StateType = "task"
	StateTypeDecision StateType = "decision"
	StateTypeParallel StateType = "parallel"
	StateTypeJoin     StateType = "join"
	StateTypeEnd      StateType = "end"
)

// TransitionType represents the transition types
type TransitionType string

const (
	TransitionOnSuccess TransitionType = "on_success"
	TransitionOnFailure TransitionType = "on_failure"
	TransitionOnTimeout TransitionType = "on_timeout"
	TransitionOnCustom  TransitionType = "on_custom"
)

// Event represents an event that can trigger a transition
type Event struct {
	ID        string                 `json:"id"`
	Type      TransitionType         `json:"type"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// NewEvent creates a new event
func NewEvent(eventType TransitionType, data map[string]interface{}) Event {
	return Event{
		ID:        uuid.New().String(),
		Type:      eventType,
		Data:      data,
		Timestamp: time.Now(),
	}
}

// StateContext contains information about the state execution context
type StateContext struct {
	Context     context.Context
	InstanceID  string
	Data        map[string]interface{}
	Metadata    map[string]interface{}
	ExecutionID string
}

// StateResult represents the result of state execution
type StateResult struct {
	Success bool
	Data    map[string]interface{}
	Error   error
	Event   Event
}

// State interface that all state types must implement
type State interface {
	// ID returns the unique identifier of the state
	ID() string

	// Type returns the type of the state
	Type() StateType

	// Enter is called when the state is activated
	Enter(ctx StateContext) error

	// Execute executes the main logic of the state
	Execute(ctx StateContext) StateResult

	// Exit is called when the state is deactivated
	Exit(ctx StateContext) error

	// GetTransitions returns the available transitions from this state
	GetTransitions() map[TransitionType]string

	// Clone creates a copy of the state
	Clone() State
}

// BaseState provides base implementation for all states
type BaseState struct {
	id          string
	stateType   StateType
	transitions map[TransitionType]string
	metadata    map[string]interface{}
}

// NewBaseState creates a new base state
func NewBaseState(id string, stateType StateType) *BaseState {
	return &BaseState{
		id:          id,
		stateType:   stateType,
		transitions: make(map[TransitionType]string),
		metadata:    make(map[string]interface{}),
	}
}

func (s *BaseState) ID() string {
	return s.id
}

func (s *BaseState) Type() StateType {
	return s.stateType
}

func (s *BaseState) Enter(ctx StateContext) error {
	// Base implementation - can be overridden
	return nil
}

func (s *BaseState) Execute(ctx StateContext) StateResult {
	// Base implementation - returns simple success
	return StateResult{
		Success: true,
		Event:   NewEvent(TransitionOnSuccess, nil),
	}
}

func (s *BaseState) Exit(ctx StateContext) error {
	// Base implementation - can be overridden
	return nil
}

func (s *BaseState) GetTransitions() map[TransitionType]string {
	result := make(map[TransitionType]string)
	for k, v := range s.transitions {
		result[k] = v
	}
	return result
}

func (s *BaseState) AddTransition(transitionType TransitionType, targetStateID string) {
	s.transitions[transitionType] = targetStateID
}

func (s *BaseState) SetMetadata(key string, value interface{}) {
	s.metadata[key] = value
}

func (s *BaseState) GetMetadata(key string) (interface{}, bool) {
	value, exists := s.metadata[key]
	return value, exists
}

func (s *BaseState) Clone() State {
	clone := &BaseState{
		id:          s.id,
		stateType:   s.stateType,
		transitions: make(map[TransitionType]string),
		metadata:    make(map[string]interface{}),
	}

	// Deep copy of maps
	for k, v := range s.transitions {
		clone.transitions[k] = v
	}

	for k, v := range s.metadata {
		clone.metadata[k] = v
	}

	return clone
}

// StartState represents the initial state
type StartState struct {
	*BaseState
}

func NewStartState(id string) *StartState {
	return &StartState{
		BaseState: NewBaseState(id, StateTypeStart),
	}
}

func (s *StartState) Execute(ctx StateContext) StateResult {
	return StateResult{
		Success: true,
		Event:   NewEvent(TransitionOnSuccess, nil),
	}
}

func (s *StartState) Clone() State {
	clone := NewStartState(s.id)
	clone.transitions = make(map[TransitionType]string)
	for k, v := range s.transitions {
		clone.transitions[k] = v
	}
	return clone
}

// TaskState represents a task state
type TaskState struct {
	*BaseState
	TaskFunc func(StateContext) StateResult
}

func NewTaskState(id string, taskFunc func(StateContext) StateResult) *TaskState {
	return &TaskState{
		BaseState: NewBaseState(id, StateTypeTask),
		TaskFunc:  taskFunc,
	}
}

func (s *TaskState) Execute(ctx StateContext) StateResult {
	if s.TaskFunc != nil {
		return s.TaskFunc(ctx)
	}

	return StateResult{
		Success: true,
		Event:   NewEvent(TransitionOnSuccess, nil),
	}
}

func (s *TaskState) Clone() State {
	clone := NewTaskState(s.id, s.TaskFunc)
	clone.transitions = make(map[TransitionType]string)
	for k, v := range s.transitions {
		clone.transitions[k] = v
	}
	return clone
}

// DecisionState represents a decision state
type DecisionState struct {
	*BaseState
	ConditionFunc func(StateContext) (TransitionType, error)
}

func NewDecisionState(id string, conditionFunc func(StateContext) (TransitionType, error)) *DecisionState {
	return &DecisionState{
		BaseState:     NewBaseState(id, StateTypeDecision),
		ConditionFunc: conditionFunc,
	}
}

func (s *DecisionState) Execute(ctx StateContext) StateResult {
	if s.ConditionFunc != nil {
		transitionType, err := s.ConditionFunc(ctx)
		if err != nil {
			return StateResult{
				Success: false,
				Error:   err,
				Event:   NewEvent(TransitionOnFailure, map[string]interface{}{"error": err.Error()}),
			}
		}

		return StateResult{
			Success: true,
			Event:   NewEvent(transitionType, nil),
		}
	}

	return StateResult{
		Success: true,
		Event:   NewEvent(TransitionOnSuccess, nil),
	}
}

func (s *DecisionState) Clone() State {
	clone := NewDecisionState(s.id, s.ConditionFunc)
	clone.transitions = make(map[TransitionType]string)
	for k, v := range s.transitions {
		clone.transitions[k] = v
	}
	return clone
}

// ParallelState represents a state that executes multiple tasks in parallel
type ParallelState struct {
	*BaseState
	ParallelTasks []func(StateContext) StateResult
}

func NewParallelState(id string, tasks []func(StateContext) StateResult) *ParallelState {
	return &ParallelState{
		BaseState:     NewBaseState(id, StateTypeParallel),
		ParallelTasks: tasks,
	}
}

func (s *ParallelState) Execute(ctx StateContext) StateResult {
	if len(s.ParallelTasks) == 0 {
		return StateResult{
			Success: true,
			Event:   NewEvent(TransitionOnSuccess, nil),
		}
	}

	results := make(chan StateResult, len(s.ParallelTasks))

	for _, task := range s.ParallelTasks {
		go func(t func(StateContext) StateResult) {
			results <- t(ctx)
		}(task)
	}

	allResults := make([]StateResult, 0, len(s.ParallelTasks))
	for i := 0; i < len(s.ParallelTasks); i++ {
		result := <-results
		allResults = append(allResults, result)
	}

	// Check if all tasks were successful
	allSuccess := true
	for _, result := range allResults {
		if !result.Success {
			allSuccess = false
			break
		}
	}

	eventType := TransitionOnSuccess
	if !allSuccess {
		eventType = TransitionOnFailure
	}

	return StateResult{
		Success: allSuccess,
		Data:    map[string]interface{}{"parallel_results": allResults},
		Event:   NewEvent(eventType, nil),
	}
}

func (s *ParallelState) Clone() State {
	clone := NewParallelState(s.id, s.ParallelTasks)
	clone.transitions = make(map[TransitionType]string)
	for k, v := range s.transitions {
		clone.transitions[k] = v
	}
	return clone
}

// JoinState represents a state that waits for multiple transitions
type JoinState struct {
	*BaseState
	RequiredInputs []string
	receivedInputs map[string]bool
}

func NewJoinState(id string, requiredInputs []string) *JoinState {
	return &JoinState{
		BaseState:      NewBaseState(id, StateTypeJoin),
		RequiredInputs: requiredInputs,
		receivedInputs: make(map[string]bool),
	}
}

func (s *JoinState) Execute(ctx StateContext) StateResult {
	// Mark this input as received
	if inputID, ok := ctx.Data["input_id"].(string); ok {
		s.receivedInputs[inputID] = true
	}

	// Check if all inputs have been received
	allReceived := true
	for _, inputID := range s.RequiredInputs {
		if !s.receivedInputs[inputID] {
			allReceived = false
			break
		}
	}

	if allReceived {
		// Reset for next execution
		s.receivedInputs = make(map[string]bool)
		return StateResult{
			Success: true,
			Event:   NewEvent(TransitionOnSuccess, nil),
		}
	}

	// Still waiting for inputs
	return StateResult{
		Success: false,
		Data:    map[string]interface{}{"waiting_for_inputs": true},
		Event:   NewEvent(TransitionOnCustom, map[string]interface{}{"status": "waiting"}),
	}
}

func (s *JoinState) Clone() State {
	clone := NewJoinState(s.id, s.RequiredInputs)
	clone.transitions = make(map[TransitionType]string)
	for k, v := range s.transitions {
		clone.transitions[k] = v
	}
	return clone
}

// EndState represents the final state
type EndState struct {
	*BaseState
}

func NewEndState(id string) *EndState {
	return &EndState{
		BaseState: NewBaseState(id, StateTypeEnd),
	}
}

func (s *EndState) Execute(ctx StateContext) StateResult {
	return StateResult{
		Success: true,
		Data:    map[string]interface{}{"completed": true},
		Event:   NewEvent(TransitionOnSuccess, nil),
	}
}

func (s *EndState) Clone() State {
	clone := NewEndState(s.id)
	clone.transitions = make(map[TransitionType]string)
	for k, v := range s.transitions {
		clone.transitions[k] = v
	}
	return clone
}
