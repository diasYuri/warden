package warden

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// InstanceStatus represents the status of an instance
type InstanceStatus string

const (
	InstanceStatusCreated   InstanceStatus = "created"
	InstanceStatusRunning   InstanceStatus = "running"
	InstanceStatusCompleted InstanceStatus = "completed"
	InstanceStatusFailed    InstanceStatus = "failed"
	InstanceStatusSuspended InstanceStatus = "suspended"
)

// InstanceSnapshot represents a snapshot of an instance
type InstanceSnapshot struct {
	ID              string                 `json:"id"`
	InstanceID      string                 `json:"instance_id"`
	ExecutionID     string                 `json:"execution_id"`
	CurrentStateID  string                 `json:"current_state_id"`
	Status          InstanceStatus         `json:"status"`
	Data            map[string]interface{} `json:"data"`
	Metadata        map[string]interface{} `json:"metadata"`
	Version         int64                  `json:"version"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	CompletedAt     *time.Time             `json:"completed_at,omitempty"`
	LastEventID     string                 `json:"last_event_id,omitempty"`
	IdempotencyKeys map[string]bool        `json:"idempotency_keys,omitempty"`
	Error           *string                `json:"error,omitempty"`
}

// NewInstanceSnapshot creates a new snapshot
func NewInstanceSnapshot(instanceID, executionID, currentStateID string) *InstanceSnapshot {
	now := time.Now()
	return &InstanceSnapshot{
		ID:              generateID(),
		InstanceID:      instanceID,
		ExecutionID:     executionID,
		CurrentStateID:  currentStateID,
		Status:          InstanceStatusCreated,
		Data:            make(map[string]interface{}),
		Metadata:        make(map[string]interface{}),
		Version:         1,
		CreatedAt:       now,
		UpdatedAt:       now,
		IdempotencyKeys: make(map[string]bool),
	}
}

// Clone creates a copy of the snapshot
func (s *InstanceSnapshot) Clone() *InstanceSnapshot {
	clone := &InstanceSnapshot{
		ID:             generateID(),
		InstanceID:     s.InstanceID,
		ExecutionID:    s.ExecutionID,
		CurrentStateID: s.CurrentStateID,
		Status:         s.Status,
		Version:        s.Version + 1,
		CreatedAt:      s.CreatedAt,
		UpdatedAt:      time.Now(),
		CompletedAt:    s.CompletedAt,
		LastEventID:    s.LastEventID,
	}

	// Deep copy of maps
	clone.Data = make(map[string]interface{})
	for k, v := range s.Data {
		clone.Data[k] = v
	}

	clone.Metadata = make(map[string]interface{})
	for k, v := range s.Metadata {
		clone.Metadata[k] = v
	}

	clone.IdempotencyKeys = make(map[string]bool)
	for k, v := range s.IdempotencyKeys {
		clone.IdempotencyKeys[k] = v
	}

	if s.Error != nil {
		errorCopy := *s.Error
		clone.Error = &errorCopy
	}

	return clone
}

// SetError sets an error in the snapshot
func (s *InstanceSnapshot) SetError(err error) {
	if err != nil {
		errStr := err.Error()
		s.Error = &errStr
		s.Status = InstanceStatusFailed
	}
}

// ClearError clears the error from the snapshot
func (s *InstanceSnapshot) ClearError() {
	s.Error = nil
	if s.Status == InstanceStatusFailed {
		s.Status = InstanceStatusRunning
	}
}

// AddIdempotencyKey adds an idempotency key
func (s *InstanceSnapshot) AddIdempotencyKey(key string) bool {
	if s.IdempotencyKeys[key] {
		return false // Already exists
	}
	s.IdempotencyKeys[key] = true
	return true
}

// HasIdempotencyKey checks if an idempotency key exists
func (s *InstanceSnapshot) HasIdempotencyKey(key string) bool {
	return s.IdempotencyKeys[key]
}

// InstancePersister interface for persisting instances
type InstancePersister interface {
	// SaveSnapshot saves an instance snapshot
	SaveSnapshot(snapshot *InstanceSnapshot) error

	// LoadSnapshot loads an instance snapshot
	LoadSnapshot(instanceID string) (*InstanceSnapshot, error)

	// LoadSnapshotByVersion loads a specific snapshot by version
	LoadSnapshotByVersion(instanceID string, version int64) (*InstanceSnapshot, error)

	// GetSnapshots returns all snapshots of an instance
	GetSnapshots(instanceID string) ([]*InstanceSnapshot, error)

	// DeleteInstance removes an instance and all its snapshots
	DeleteInstance(instanceID string) error

	// ListInstances lists all instances
	ListInstances(status InstanceStatus, limit, offset int) ([]*InstanceSnapshot, error)

	// GetInstanceCount returns the total number of instances
	GetInstanceCount(status InstanceStatus) (int64, error)
}

// InMemoryInstancePersister in-memory implementation
type InMemoryInstancePersister struct {
	snapshots map[string][]*InstanceSnapshot // instanceID -> list of snapshots
	mutex     sync.RWMutex
}

// NewInMemoryInstancePersister creates a new in-memory persister
func NewInMemoryInstancePersister() *InMemoryInstancePersister {
	return &InMemoryInstancePersister{
		snapshots: make(map[string][]*InstanceSnapshot),
	}
}

func (p *InMemoryInstancePersister) SaveSnapshot(snapshot *InstanceSnapshot) error {
	if snapshot == nil {
		return ErrInvalidSnapshot
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	if _, exists := p.snapshots[snapshot.InstanceID]; !exists {
		p.snapshots[snapshot.InstanceID] = make([]*InstanceSnapshot, 0)
	}

	// Clones the snapshot to avoid external modifications
	clone := snapshot.Clone()
	clone.ID = snapshot.ID           // Keeps original ID
	clone.Version = snapshot.Version // Keeps original version
	clone.UpdatedAt = time.Now()

	p.snapshots[snapshot.InstanceID] = append(p.snapshots[snapshot.InstanceID], clone)

	return nil
}

func (p *InMemoryInstancePersister) LoadSnapshot(instanceID string) (*InstanceSnapshot, error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	snapshots, exists := p.snapshots[instanceID]
	if !exists || len(snapshots) == 0 {
		return nil, ErrInstanceNotFound
	}

	// Returns the most recent snapshot
	latest := snapshots[len(snapshots)-1]
	return latest.Clone(), nil
}

func (p *InMemoryInstancePersister) LoadSnapshotByVersion(instanceID string, version int64) (*InstanceSnapshot, error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	snapshots, exists := p.snapshots[instanceID]
	if !exists {
		return nil, ErrInstanceNotFound
	}

	for _, snapshot := range snapshots {
		if snapshot.Version == version {
			return snapshot.Clone(), nil
		}
	}

	return nil, ErrSnapshotNotFound
}

func (p *InMemoryInstancePersister) GetSnapshots(instanceID string) ([]*InstanceSnapshot, error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	snapshots, exists := p.snapshots[instanceID]
	if !exists {
		return []*InstanceSnapshot{}, nil
	}

	result := make([]*InstanceSnapshot, len(snapshots))
	for i, snapshot := range snapshots {
		result[i] = snapshot.Clone()
	}

	return result, nil
}

func (p *InMemoryInstancePersister) DeleteInstance(instanceID string) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	delete(p.snapshots, instanceID)
	return nil
}

func (p *InMemoryInstancePersister) ListInstances(status InstanceStatus, limit, offset int) ([]*InstanceSnapshot, error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	var result []*InstanceSnapshot
	count := 0

	for _, snapshots := range p.snapshots {
		if len(snapshots) == 0 {
			continue
		}

		// Gets the most recent snapshot
		latest := snapshots[len(snapshots)-1]

		// Filters by status if specified
		if status != "" && latest.Status != status {
			continue
		}

		// Implements pagination
		if count < offset {
			count++
			continue
		}

		if len(result) >= limit && limit > 0 {
			break
		}

		result = append(result, latest.Clone())
		count++
	}

	return result, nil
}

func (p *InMemoryInstancePersister) GetInstanceCount(status InstanceStatus) (int64, error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	count := int64(0)

	for _, snapshots := range p.snapshots {
		if len(snapshots) == 0 {
			continue
		}

		// Gets the most recent snapshot
		latest := snapshots[len(snapshots)-1]

		// Filters by status if specified
		if status != "" && latest.Status != status {
			continue
		}

		count++
	}

	return count, nil
}

// JSONInstancePersister implementation that serializes snapshots to JSON
type JSONInstancePersister struct {
	*InMemoryInstancePersister
}

// NewJSONInstancePersister creates a new JSON persister
func NewJSONInstancePersister() *JSONInstancePersister {
	return &JSONInstancePersister{
		InMemoryInstancePersister: NewInMemoryInstancePersister(),
	}
}

// SerializeSnapshot serializes a snapshot to JSON
func (p *JSONInstancePersister) SerializeSnapshot(snapshot *InstanceSnapshot) ([]byte, error) {
	return json.Marshal(snapshot)
}

// DeserializeSnapshot deserializes a snapshot from JSON
func (p *JSONInstancePersister) DeserializeSnapshot(data []byte) (*InstanceSnapshot, error) {
	var snapshot InstanceSnapshot
	err := json.Unmarshal(data, &snapshot)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize snapshot: %w", err)
	}
	return &snapshot, nil
}

// EventSourcePersister implementation that uses event sourcing
type EventSourcePersister struct {
	eventPersister EventPersister
	snapshots      map[string]*InstanceSnapshot
	mutex          sync.RWMutex
}

// NewEventSourcePersister creates a new event sourcing based persister
func NewEventSourcePersister(eventPersister EventPersister) *EventSourcePersister {
	return &EventSourcePersister{
		eventPersister: eventPersister,
		snapshots:      make(map[string]*InstanceSnapshot),
	}
}

func (p *EventSourcePersister) SaveSnapshot(snapshot *InstanceSnapshot) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.snapshots[snapshot.InstanceID] = snapshot.Clone()
	return nil
}

func (p *EventSourcePersister) LoadSnapshot(instanceID string) (*InstanceSnapshot, error) {
	p.mutex.RLock()
	cachedSnapshot, exists := p.snapshots[instanceID]
	p.mutex.RUnlock()

	if exists {
		return cachedSnapshot.Clone(), nil
	}

	// Rebuilds the snapshot from events
	return p.rebuildFromEvents(instanceID)
}

func (p *EventSourcePersister) LoadSnapshotByVersion(instanceID string, version int64) (*InstanceSnapshot, error) {
	return p.rebuildFromEventsToVersion(instanceID, version)
}

func (p *EventSourcePersister) GetSnapshots(instanceID string) ([]*InstanceSnapshot, error) {
	// For event sourcing, returns only the current snapshot
	snapshot, err := p.LoadSnapshot(instanceID)
	if err != nil {
		return nil, err
	}
	return []*InstanceSnapshot{snapshot}, nil
}

func (p *EventSourcePersister) DeleteInstance(instanceID string) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	delete(p.snapshots, instanceID)
	return nil
}

func (p *EventSourcePersister) ListInstances(status InstanceStatus, limit, offset int) ([]*InstanceSnapshot, error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	var result []*InstanceSnapshot
	count := 0

	for _, snapshot := range p.snapshots {
		// Filters by status if specified
		if status != "" && snapshot.Status != status {
			continue
		}

		// Implements pagination
		if count < offset {
			count++
			continue
		}

		if len(result) >= limit && limit > 0 {
			break
		}

		result = append(result, snapshot.Clone())
		count++
	}

	return result, nil
}

func (p *EventSourcePersister) GetInstanceCount(status InstanceStatus) (int64, error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	count := int64(0)

	for _, snapshot := range p.snapshots {
		// Filters by status if specified
		if status != "" && snapshot.Status != status {
			continue
		}
		count++
	}

	return count, nil
}

// rebuildFromEvents rebuilds a snapshot from events
func (p *EventSourcePersister) rebuildFromEvents(instanceID string) (*InstanceSnapshot, error) {
	if p.eventPersister == nil {
		return nil, ErrEventPersisterNotSet
	}

	events, err := p.eventPersister.GetEvents(instanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get events: %w", err)
	}

	if len(events) == 0 {
		return nil, ErrInstanceNotFound
	}

	// Creates an initial snapshot
	snapshot := &InstanceSnapshot{
		ID:              generateID(),
		InstanceID:      instanceID,
		Data:            make(map[string]interface{}),
		Metadata:        make(map[string]interface{}),
		IdempotencyKeys: make(map[string]bool),
		Version:         1,
	}

	// Applies events in chronological order
	for _, event := range events {
		if err := p.applyEventToSnapshot(snapshot, event); err != nil {
			return nil, fmt.Errorf("failed to apply event %s: %w", event.ID, err)
		}
	}

	return snapshot, nil
}

// rebuildFromEventsToVersion rebuilds a snapshot up to a specific version
func (p *EventSourcePersister) rebuildFromEventsToVersion(instanceID string, version int64) (*InstanceSnapshot, error) {
	if p.eventPersister == nil {
		return nil, ErrEventPersisterNotSet
	}

	events, err := p.eventPersister.GetEvents(instanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get events: %w", err)
	}

	if len(events) == 0 {
		return nil, ErrInstanceNotFound
	}

	// Creates an initial snapshot
	snapshot := &InstanceSnapshot{
		ID:              generateID(),
		InstanceID:      instanceID,
		Data:            make(map[string]interface{}),
		Metadata:        make(map[string]interface{}),
		IdempotencyKeys: make(map[string]bool),
		Version:         1,
	}

	// Applies events up to the desired version
	eventCount := int64(0)
	for _, event := range events {
		if eventCount >= version {
			break
		}

		if err := p.applyEventToSnapshot(snapshot, event); err != nil {
			return nil, fmt.Errorf("failed to apply event %s: %w", event.ID, err)
		}
		eventCount++
	}

	snapshot.Version = version
	return snapshot, nil
}

// applyEventToSnapshot applies an event to a snapshot
func (p *EventSourcePersister) applyEventToSnapshot(snapshot *InstanceSnapshot, event StateEvent) error {
	switch event.Type {
	case EventExecutionStarted:
		snapshot.ExecutionID = event.ExecutionID
		snapshot.Status = InstanceStatusRunning
		snapshot.CreatedAt = event.Timestamp

	case EventStateEntered:
		snapshot.CurrentStateID = event.StateID
		snapshot.UpdatedAt = event.Timestamp

	case EventExecutionFinished:
		snapshot.Status = InstanceStatusCompleted
		if snapshot.CompletedAt == nil {
			snapshot.CompletedAt = &event.Timestamp
		}
		snapshot.UpdatedAt = event.Timestamp

	case EventExecutionFailed:
		snapshot.Status = InstanceStatusFailed
		if event.Error != nil {
			errStr := event.Error.Error()
			snapshot.Error = &errStr
		}
		snapshot.UpdatedAt = event.Timestamp
	}

	// Applies event data to snapshot
	if event.Data != nil {
		for k, v := range event.Data {
			snapshot.Data[k] = v
		}
	}

	snapshot.LastEventID = event.ID
	return nil
}
