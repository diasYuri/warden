// Advanced Workflow Example
// This file demonstrates advanced features of Warden including:
// - Decision states
// - Parallel execution
// - Event handling
// - Persistence
// - Metrics collection
package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	warden "github.com/diasYuri/warden/pkg"
)

// Exemplo avançado demonstrando decisões, paralelismo e listeners
func main() {
	fmt.Println("=== Warden State Machine - Advanced Workflow Example ===")

	// Configura persistência e event bus
	persister := warden.NewInMemoryInstancePersister()
	eventBus := warden.NewEventBus()

	// Adiciona listeners
	metricsListener := warden.NewMetricsListener()
	loggingListener := warden.NewLoggingListener("info")

	eventBus.Subscribe(metricsListener)
	eventBus.Subscribe(loggingListener)

	// Cria uma máquina de estados mais complexa
	machine, err := warden.New("complex-order-processing").
		WithPersister(persister).
		WithEventBus(eventBus).
		StartState("start").
		DecisionState("check_order_type", checkOrderTypeDecision).
		TaskState("process_standard", processStandardOrder).
		ParallelState("process_premium", []func(warden.StateContext) warden.StateResult{
			processPremiumValidation,
			processPremiumShipping,
			processPremiumNotification,
		}).
		JoinState("finalize", []string{"standard_done", "premium_done"}).
		EndState("completed").
		EndState("failed").
		// Transições do start
		Transition("start", "check_order_type", warden.TransitionOnSuccess).
		// Transições da decisão
		Transition("check_order_type", "process_standard", warden.TransitionOnSuccess).
		Transition("check_order_type", "process_premium", warden.TransitionOnFailure).
		Transition("check_order_type", "failed", warden.TransitionOnTimeout).
		// Transições dos processamentos
		Transition("process_standard", "finalize", warden.TransitionOnSuccess).
		Transition("process_standard", "failed", warden.TransitionOnFailure).
		Transition("process_premium", "finalize", warden.TransitionOnSuccess).
		Transition("process_premium", "failed", warden.TransitionOnFailure).
		// Transições finais
		Transition("finalize", "completed", warden.TransitionOnSuccess).
		Transition("finalize", "failed", warden.TransitionOnFailure).
		Build()

	if err != nil {
		log.Fatalf("Failed to build state machine: %v", err)
	}

	// Testa com diferentes tipos de pedidos
	testCases := []map[string]interface{}{
		{
			"order_id":   "STD-001",
			"order_type": "standard",
			"amount":     50.0,
			"priority":   "normal",
		},
		{
			"order_id":   "PRM-001",
			"order_type": "premium",
			"amount":     250.0,
			"priority":   "high",
		},
		{
			"order_id":   "INV-001",
			"order_type": "invalid",
			"amount":     0.0,
			"priority":   "unknown",
		},
	}

	ctx := context.Background()

	for i, testData := range testCases {
		fmt.Printf("\n=== Test Case %d: %s ===\n", i+1, testData["order_id"])

		instanceID := fmt.Sprintf("test-instance-%d", i+1)

		// Cria instância
		instance, err := machine.CreateInstance(instanceID, testData)
		if err != nil {
			log.Printf("Failed to create instance: %v", err)
			continue
		}

		fmt.Printf("Created instance: %s\n", instance.ID)
		fmt.Printf("Order type: %s\n", testData["order_type"])

		// Executa workflow
		err = instance.Start(ctx)
		if err != nil {
			log.Printf("Workflow execution failed: %v", err)
		}

		// Mostra resultado
		fmt.Printf("Final state: %s\n", instance.CurrentStateID)
		fmt.Printf("Status: %s\n", instance.Status)

		if instance.IsFailed() && instance.Error != nil {
			fmt.Printf("Error: %v\n", instance.Error)
		}

		// Demonstra persistência - recarrega instância
		if instance.IsCompleted() {
			fmt.Println("\nDemonstrating persistence...")
			reloadedInstance, err := machine.LoadInstance(instanceID)
			if err != nil {
				log.Printf("Failed to reload instance: %v", err)
			} else {
				fmt.Printf("Reloaded instance state: %s\n", reloadedInstance.CurrentStateID)
				fmt.Printf("Reloaded instance status: %s\n", reloadedInstance.Status)
			}
		}

		time.Sleep(100 * time.Millisecond) // Pausa entre testes
	}

	// Mostra métricas coletadas
	fmt.Println("\n=== Collected Metrics ===")
	metrics := metricsListener.GetMetrics()
	for metric, count := range metrics {
		fmt.Printf("%s: %d\n", metric, count)
	}

	// Demonstra listagem de instâncias
	fmt.Println("\n=== Instance Listing ===")
	instances, err := persister.ListInstances("", 10, 0)
	if err != nil {
		log.Printf("Failed to list instances: %v", err)
	} else {
		for _, snapshot := range instances {
			fmt.Printf("Instance %s: %s (status: %s)\n",
				snapshot.InstanceID, snapshot.CurrentStateID, snapshot.Status)
		}
	}

	// Cleanup
	eventBus.Close()
}

// checkOrderTypeDecision decide como processar o pedido baseado no tipo
func checkOrderTypeDecision(ctx warden.StateContext) (warden.TransitionType, error) {
	fmt.Println("Checking order type...")

	orderType, exists := ctx.Data["order_type"]
	if !exists {
		return warden.TransitionOnTimeout, fmt.Errorf("order_type not found")
	}

	switch orderType {
	case "standard":
		fmt.Println("  → Standard order detected")
		return warden.TransitionOnSuccess, nil
	case "premium":
		fmt.Println("  → Premium order detected")
		return warden.TransitionOnFailure, nil // Usamos failure para ir para premium
	default:
		fmt.Printf("  → Invalid order type: %v\n", orderType)
		return warden.TransitionOnTimeout, fmt.Errorf("invalid order type: %v", orderType)
	}
}

// processStandardOrder processa pedidos padrão
func processStandardOrder(ctx warden.StateContext) warden.StateResult {
	fmt.Println("Processing standard order...")
	time.Sleep(200 * time.Millisecond)

	orderID := ctx.Data["order_id"]
	amount := ctx.Data["amount"]

	fmt.Printf("  Standard processing for order %v (amount: %v)\n", orderID, amount)

	// Simula processamento padrão
	processingID := fmt.Sprintf("STD-PROC-%d", time.Now().Unix())

	fmt.Printf("  ✓ Standard order processed (ID: %s)\n", processingID)

	return warden.StateResult{
		Success: true,
		Data: map[string]interface{}{
			"processing_id":   processingID,
			"processing_type": "standard",
			"processed_at":    time.Now(),
			"standard_done":   true,
		},
		Event: warden.NewEvent(warden.TransitionOnSuccess, nil),
	}
}

// processPremiumValidation valida pedidos premium (executa em paralelo)
func processPremiumValidation(ctx warden.StateContext) warden.StateResult {
	fmt.Println("  [PARALLEL] Premium validation...")
	time.Sleep(time.Duration(rand.Intn(300)) * time.Millisecond)

	amount := ctx.Data["amount"]

	// Valida se amount é suficiente para premium
	if amountFloat, ok := amount.(float64); !ok || amountFloat < 100.0 {
		return warden.StateResult{
			Success: false,
			Error:   fmt.Errorf("insufficient amount for premium: %v", amount),
			Event:   warden.NewEvent(warden.TransitionOnFailure, nil),
		}
	}

	fmt.Println("    ✓ Premium validation passed")

	return warden.StateResult{
		Success: true,
		Data: map[string]interface{}{
			"premium_validated": true,
			"validation_score":  rand.Intn(100) + 80, // Score entre 80-179
		},
		Event: warden.NewEvent(warden.TransitionOnSuccess, nil),
	}
}

// processPremiumShipping processa envio premium (executa em paralelo)
func processPremiumShipping(ctx warden.StateContext) warden.StateResult {
	fmt.Println("  [PARALLEL] Premium shipping...")
	time.Sleep(time.Duration(rand.Intn(250)) * time.Millisecond)

	orderID := warden.SafeString(ctx.Data["order_id"])

	// Simula processamento de envio premium
	trackingNumber := fmt.Sprintf("PRM-TRACK-%d", time.Now().Unix())

	fmt.Printf("    ✓ Premium shipping processed for order %s (Tracking: %s)\n", orderID, trackingNumber)

	return warden.StateResult{
		Success: true,
		Data: map[string]interface{}{
			"premium_tracking":   trackingNumber,
			"shipping_method":    "express",
			"estimated_delivery": time.Now().Add(24 * time.Hour),
		},
		Event: warden.NewEvent(warden.TransitionOnSuccess, nil),
	}
}

// processPremiumNotification envia notificações premium (executa em paralelo)
func processPremiumNotification(ctx warden.StateContext) warden.StateResult {
	fmt.Println("  [PARALLEL] Premium notification...")
	time.Sleep(time.Duration(rand.Intn(150)) * time.Millisecond)

	orderID := warden.SafeString(ctx.Data["order_id"])

	// Simula envio de notificações premium
	notificationID := fmt.Sprintf("PRM-NOTIFY-%d", time.Now().Unix())

	fmt.Printf("    ✓ Premium notifications sent for order %s (ID: %s)\n", orderID, notificationID)

	return warden.StateResult{
		Success: true,
		Data: map[string]interface{}{
			"notification_id":    notificationID,
			"notifications_sent": []string{"email", "sms", "push"},
			"premium_done":       true,
		},
		Event: warden.NewEvent(warden.TransitionOnSuccess, nil),
	}
}
