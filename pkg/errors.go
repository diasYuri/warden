package warden

import "errors"

// State-related errors
var (
	ErrStateNotFound      = errors.New("state not found")
	ErrStateAlreadyExists = errors.New("state already exists")
	ErrInvalidStateType   = errors.New("invalid state type")
	ErrStateNotActive     = errors.New("state is not active")
)

// Transition-related errors
var (
	ErrTransitionNotFound   = errors.New("transition not found")
	ErrTransitionNotAllowed = errors.New("transition not allowed")
	ErrInvalidTransition    = errors.New("invalid transition")
	ErrCircularTransition   = errors.New("circular transition detected")
)

// Instance-related errors
var (
	ErrInstanceNotFound       = errors.New("instance not found")
	ErrInstanceNotRunning     = errors.New("instance is not running")
	ErrInstanceAlreadyExists  = errors.New("instance already exists")
	ErrInstanceAlreadyStarted = errors.New("instance already started")
	ErrInstanceCompleted      = errors.New("instance already completed")
	ErrInstanceFailed         = errors.New("instance has failed")
	ErrInvalidInstanceID      = errors.New("invalid instance ID")
)

// Machine-related errors
var (
	ErrMachineNotBuilt     = errors.New("machine not built")
	ErrMachineAlreadyBuilt = errors.New("machine already built")
	ErrInvalidMachine      = errors.New("invalid machine configuration")
	ErrMachineNotValid     = errors.New("machine validation failed")
)

// Persistence-related errors
var (
	ErrPersistenceNotSet    = errors.New("persistence not configured")
	ErrSnapshotNotFound     = errors.New("snapshot not found")
	ErrInvalidSnapshot      = errors.New("invalid snapshot")
	ErrEventPersisterNotSet = errors.New("event persister not set")
)

// Validation errors
var (
	ErrInvalidInput     = errors.New("invalid input")
	ErrValidationFailed = errors.New("validation failed")
	ErrMissingRequired  = errors.New("missing required field")
	ErrInvalidFormat    = errors.New("invalid format")
)

// Concurrency errors
var (
	ErrConcurrentModification = errors.New("concurrent modification detected")
	ErrDeadlock               = errors.New("deadlock detected")
	ErrTimeout                = errors.New("operation timeout")
	ErrCancelled              = errors.New("operation cancelled")
)

// Event-related errors
var (
	ErrEventNotFound    = errors.New("event not found")
	ErrInvalidEvent     = errors.New("invalid event")
	ErrEventProcessing  = errors.New("error processing event")
	ErrListenerNotFound = errors.New("listener not found")
	ErrDuplicateEvent   = errors.New("duplicate event")
)

// Idempotency errors
var (
	ErrIdempotencyKeyExists  = errors.New("idempotency key already exists")
	ErrInvalidIdempotencyKey = errors.New("invalid idempotency key")
	ErrIdempotencyViolation  = errors.New("idempotency violation")
)

// Configuration errors
var (
	ErrInvalidConfiguration = errors.New("invalid configuration")
	ErrConfigurationMissing = errors.New("configuration missing")
	ErrUnsupportedFeature   = errors.New("unsupported feature")
)
