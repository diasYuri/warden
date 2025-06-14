package warden

import "errors"

// Erros comuns da biblioteca Warden
var (
	// Erros de estado
	ErrStateNotFound      = errors.New("state not found")
	ErrInvalidState       = errors.New("invalid state")
	ErrStateAlreadyExists = errors.New("state already exists")

	// Erros de transição
	ErrTransitionNotFound   = errors.New("transition not found")
	ErrInvalidTransition    = errors.New("invalid transition")
	ErrTransitionNotAllowed = errors.New("transition not allowed")

	// Erros de instância
	ErrInstanceNotFound      = errors.New("instance not found")
	ErrInstanceAlreadyExists = errors.New("instance already exists")
	ErrInstanceNotRunning    = errors.New("instance not running")
	ErrInstanceCompleted     = errors.New("instance already completed")
	ErrInstanceFailed        = errors.New("instance failed")

	// Erros de snapshot
	ErrSnapshotNotFound = errors.New("snapshot not found")
	ErrInvalidSnapshot  = errors.New("invalid snapshot")

	// Erros de persistência
	ErrEventPersisterNotSet = errors.New("event persister not set")
	ErrPersistenceFailed    = errors.New("persistence operation failed")

	// Erros de listener
	ErrListenerNotFound      = errors.New("listener not found")
	ErrListenerAlreadyExists = errors.New("listener already exists")

	// Erros de máquina de estado
	ErrMachineNotBuilt     = errors.New("state machine not built")
	ErrMachineAlreadyBuilt = errors.New("state machine already built")
	ErrInvalidMachine      = errors.New("invalid state machine")
	ErrMachineNotStarted   = errors.New("state machine not started")

	// Erros de validação
	ErrEmptyStateID     = errors.New("state ID cannot be empty")
	ErrEmptyInstanceID  = errors.New("instance ID cannot be empty")
	ErrEmptyExecutionID = errors.New("execution ID cannot be empty")
	ErrInvalidEventType = errors.New("invalid event type")

	// Erros de concorrência
	ErrInstanceLocked      = errors.New("instance is locked")
	ErrConcurrentExecution = errors.New("concurrent execution detected")

	// Erros de idempotência
	ErrDuplicateEvent       = errors.New("duplicate event detected")
	ErrIdempotencyKeyExists = errors.New("idempotency key already exists")
)
