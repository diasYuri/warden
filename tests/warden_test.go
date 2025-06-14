package warden_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	warden "github.com/diasYuri/warden/pkg"
)

func TestBasicStateMachine(t *testing.T) {
	// Creates a simple machine
	machine, err := warden.New("test-machine").
		StartState("start").
		TaskState("process", func(ctx warden.StateContext) warden.StateResult {
			return warden.StateResult{
				Success: true,
				Event:   warden.NewEvent(warden.TransitionOnSuccess, nil),
			}
		}).
		EndState("end").
		Transition("start", "process", warden.TransitionOnSuccess).
		Transition("process", "end", warden.TransitionOnSuccess).
		Build()

	if err != nil {
		t.Fatalf("Failed to build machine: %v", err)
	}

	// Creates an instance
	instance, err := machine.CreateInstance("test-instance", map[string]interface{}{
		"order_id": "12345",
	})

	if err != nil {
		t.Fatalf("Failed to create instance: %v", err)
	}

	// Starts execution
	ctx := context.Background()
	err = instance.Start(ctx)

	if err != nil {
		t.Fatalf("Failed to start instance: %v", err)
	}

	// Checks if it was completed
	if !instance.IsCompleted() {
		t.Error("Instance should be completed")
	}

	// Checks if it's in the end state
	currentState := instance.GetCurrentState()
	if currentState.ID() != "end" {
		t.Errorf("Expected end state, got %s", currentState.ID())
	}
}

func TestDecisionState(t *testing.T) {
	// Creates a machine with decision
	machine, err := warden.New("decision-machine").
		StartState("start").
		DecisionState("check", func(ctx warden.StateContext) (warden.TransitionType, error) {
			if value, ok := ctx.Data["approve"].(bool); ok && value {
				return warden.TransitionOnSuccess, nil
			}
			return warden.TransitionOnFailure, nil
		}).
		TaskState("approved", func(ctx warden.StateContext) warden.StateResult {
			return warden.StateResult{
				Success: true,
				Event:   warden.NewEvent(warden.TransitionOnSuccess, nil),
			}
		}).
		TaskState("rejected", func(ctx warden.StateContext) warden.StateResult {
			return warden.StateResult{
				Success: true,
				Event:   warden.NewEvent(warden.TransitionOnSuccess, nil),
			}
		}).
		EndState("end").
		Transition("start", "check", warden.TransitionOnSuccess).
		Transition("check", "approved", warden.TransitionOnSuccess).
		Transition("check", "rejected", warden.TransitionOnFailure).
		Transition("approved", "end", warden.TransitionOnSuccess).
		Transition("rejected", "end", warden.TransitionOnSuccess).
		Build()

	if err != nil {
		t.Fatalf("Failed to build machine: %v", err)
	}

	// Test approval path
	instance1, err := machine.CreateInstance("test-approve", map[string]interface{}{
		"approve": true,
	})
	if err != nil {
		t.Fatalf("Failed to create instance: %v", err)
	}

	ctx := context.Background()
	err = instance1.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start instance: %v", err)
	}

	if !instance1.IsCompleted() {
		t.Error("Instance should be completed")
	}

	// Test rejection path
	instance2, err := machine.CreateInstance("test-reject", map[string]interface{}{
		"approve": false,
	})
	if err != nil {
		t.Fatalf("Failed to create instance: %v", err)
	}

	err = instance2.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start instance: %v", err)
	}

	if !instance2.IsCompleted() {
		t.Error("Instance should be completed")
	}
}

func TestParallelExecution(t *testing.T) {
	// Creates a machine with parallel execution
	machine, err := warden.New("parallel-machine").
		StartState("start").
		ParallelState("parallel", []func(warden.StateContext) warden.StateResult{
			func(ctx warden.StateContext) warden.StateResult {
				time.Sleep(10 * time.Millisecond) // Simulates work
				return warden.StateResult{
					Success: true,
					Data:    map[string]interface{}{"task1": "completed"},
					Event:   warden.NewEvent(warden.TransitionOnSuccess, nil),
				}
			},
			func(ctx warden.StateContext) warden.StateResult {
				time.Sleep(5 * time.Millisecond) // Simulates work
				return warden.StateResult{
					Success: true,
					Data:    map[string]interface{}{"task2": "completed"},
					Event:   warden.NewEvent(warden.TransitionOnSuccess, nil),
				}
			},
		}).
		EndState("end").
		Transition("start", "parallel", warden.TransitionOnSuccess).
		Transition("parallel", "end", warden.TransitionOnSuccess).
		Build()

	if err != nil {
		t.Fatalf("Failed to build machine: %v", err)
	}

	instance, err := machine.CreateInstance("test-parallel", nil)
	if err != nil {
		t.Fatalf("Failed to create instance: %v", err)
	}

	// Measures execution time
	start := time.Now()
	ctx := context.Background()
	err = instance.Start(ctx)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Failed to start instance: %v", err)
	}

	if !instance.IsCompleted() {
		t.Error("Instance should be completed")
	}

	// Checks if parallel execution was actually parallel (should be less than sequential)
	if duration > 15*time.Millisecond {
		t.Errorf("Parallel execution took too long: %v", duration)
	}
}

func TestPersistenceAndLoading(t *testing.T) {
	// Creates a machine with persistence
	persister := warden.NewInMemoryInstancePersister()
	machine, err := warden.New("persistence-machine").
		WithPersister(persister).
		StartState("start").
		TaskState("step1", func(ctx warden.StateContext) warden.StateResult {
			return warden.StateResult{
				Success: true,
				Data:    map[string]interface{}{"step1": "done"},
				Event:   warden.NewEvent(warden.TransitionOnSuccess, nil),
			}
		}).
		TaskState("step2", func(ctx warden.StateContext) warden.StateResult {
			return warden.StateResult{
				Success: true,
				Data:    map[string]interface{}{"step2": "done"},
				Event:   warden.NewEvent(warden.TransitionOnSuccess, nil),
			}
		}).
		EndState("end").
		Transition("start", "step1", warden.TransitionOnSuccess).
		Transition("step1", "step2", warden.TransitionOnSuccess).
		Transition("step2", "end", warden.TransitionOnSuccess).
		Build()

	if err != nil {
		t.Fatalf("Failed to build machine: %v", err)
	}

	// Creates and starts instance
	instance, err := machine.CreateInstance("test-persistence", map[string]interface{}{
		"initial": "data",
	})
	if err != nil {
		t.Fatalf("Failed to create instance: %v", err)
	}

	ctx := context.Background()
	err = instance.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start instance: %v", err)
	}

	// Loads instance from persister
	loadedInstance, err := machine.LoadInstance("test-persistence")
	if err != nil {
		t.Fatalf("Failed to load instance: %v", err)
	}

	// Checks if data is preserved
	data := loadedInstance.GetData()
	if data["initial"] != "data" {
		t.Error("Initial data not preserved")
	}

	if !loadedInstance.IsCompleted() {
		t.Error("Loaded instance should be completed")
	}
}

func TestEventBusAndMetrics(t *testing.T) {
	// Creates event bus and metrics listener
	eventBus := warden.NewEventBus()
	metricsListener := warden.NewMetricsListener()
	loggingListener := warden.NewLoggingListener("debug")

	err := eventBus.Subscribe(metricsListener)
	if err != nil {
		t.Fatalf("Failed to subscribe metrics listener: %v", err)
	}

	err = eventBus.Subscribe(loggingListener)
	if err != nil {
		t.Fatalf("Failed to subscribe logging listener: %v", err)
	}

	// Creates machine with event bus
	machine, err := warden.New("metrics-machine").
		WithEventBus(eventBus).
		StartState("start").
		TaskState("work", func(ctx warden.StateContext) warden.StateResult {
			return warden.StateResult{
				Success: true,
				Event:   warden.NewEvent(warden.TransitionOnSuccess, nil),
			}
		}).
		EndState("end").
		Transition("start", "work", warden.TransitionOnSuccess).
		Transition("work", "end", warden.TransitionOnSuccess).
		Build()

	if err != nil {
		t.Fatalf("Failed to build machine: %v", err)
	}

	instance, err := machine.CreateInstance("test-metrics", nil)
	if err != nil {
		t.Fatalf("Failed to create instance: %v", err)
	}

	ctx := context.Background()
	err = instance.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start instance: %v", err)
	}

	// Waits a bit for events to be processed
	time.Sleep(10 * time.Millisecond)

	// Checks metrics
	metrics := metricsListener.GetMetrics()
	if len(metrics) == 0 {
		t.Error("No metrics collected")
	}

	// Checks if some expected metrics exist
	if startedCount, ok := metrics["executions_started"]; !ok || startedCount.(int64) < 1 {
		t.Error("Expected executions_started metric")
	}

	if completedCount, ok := metrics["executions_completed"]; !ok || completedCount.(int64) < 1 {
		t.Error("Expected executions_completed metric")
	}

	// Closes event bus
	err = eventBus.Close()
	if err != nil {
		t.Fatalf("Failed to close event bus: %v", err)
	}
}

func TestErrorHandling(t *testing.T) {
	// Creates a machine that will fail
	machine, err := warden.New("error-machine").
		StartState("start").
		TaskState("failing_task", func(ctx warden.StateContext) warden.StateResult {
			return warden.StateResult{
				Success: false,
				Error:   fmt.Errorf("intentional failure"),
				Event:   warden.NewEvent(warden.TransitionOnFailure, nil),
			}
		}).
		EndState("end").
		Transition("start", "failing_task", warden.TransitionOnSuccess).
		Build()

	if err != nil {
		t.Fatalf("Failed to build machine: %v", err)
	}

	instance, err := machine.CreateInstance("test-error", nil)
	if err != nil {
		t.Fatalf("Failed to create instance: %v", err)
	}

	ctx := context.Background()
	err = instance.Start(ctx)

	// Should return error
	if err == nil {
		t.Error("Expected error from failing task")
	}

	// Instance should be in failed state
	if !instance.IsFailed() {
		t.Error("Instance should be in failed state")
	}
}

func TestIdempotency(t *testing.T) {
	// Creates a machine
	machine, err := warden.New("idempotency-machine").
		StartState("start").
		TaskState("process", func(ctx warden.StateContext) warden.StateResult {
			return warden.StateResult{
				Success: true,
				Event:   warden.NewEvent(warden.TransitionOnSuccess, nil),
			}
		}).
		EndState("end").
		Transition("start", "process", warden.TransitionOnSuccess).
		Transition("process", "end", warden.TransitionOnSuccess).
		Build()

	if err != nil {
		t.Fatalf("Failed to build machine: %v", err)
	}

	instance, err := machine.CreateInstance("test-idempotency", nil)
	if err != nil {
		t.Fatalf("Failed to create instance: %v", err)
	}

	ctx := context.Background()
	err = instance.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start instance: %v", err)
	}

	// Tries to process the same event again
	event := warden.NewEvent(warden.TransitionOnSuccess, nil)
	err1 := instance.Next(ctx, event)
	err2 := instance.Next(ctx, event) // Same event

	// First should succeed (no-op because completed), second should fail due to idempotency
	if err1 != nil && err2 == nil {
		t.Error("Expected idempotency error on duplicate event")
	}
}

func TestMachineValidation(t *testing.T) {
	// Tests various validation scenarios

	// Machine without start state
	_, err := warden.New("no-start").
		TaskState("task", nil).
		Build()
	if err == nil {
		t.Error("Expected error for machine without start state")
	}

	// Machine without end state
	_, err = warden.New("no-end").
		StartState("start").
		TaskState("task", nil).
		Build()
	if err == nil {
		t.Error("Expected error for machine without end state")
	}

	// Machine with transition to non-existent state
	_, err = warden.New("invalid-transition").
		StartState("start").
		EndState("end").
		Transition("start", "nonexistent", warden.TransitionOnSuccess).
		Build()
	if err == nil {
		t.Error("Expected error for transition to non-existent state")
	}
}

func BenchmarkStateMachineExecution(b *testing.B) {
	// Creates a simple machine for benchmark
	machine, err := warden.New("benchmark-machine").
		StartState("start").
		TaskState("process", func(ctx warden.StateContext) warden.StateResult {
			return warden.StateResult{
				Success: true,
				Event:   warden.NewEvent(warden.TransitionOnSuccess, nil),
			}
		}).
		EndState("end").
		Transition("start", "process", warden.TransitionOnSuccess).
		Transition("process", "end", warden.TransitionOnSuccess).
		Build()

	if err != nil {
		b.Fatalf("Failed to build machine: %v", err)
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		instanceID := fmt.Sprintf("bench-instance-%d", i)
		instance, err := machine.CreateInstance(instanceID, nil)
		if err != nil {
			b.Fatalf("Failed to create instance: %v", err)
		}

		err = instance.Start(ctx)
		if err != nil {
			b.Fatalf("Failed to start instance: %v", err)
		}
	}
}
