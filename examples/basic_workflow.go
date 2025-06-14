// Basic Workflow Example
// This file demonstrates how to create and execute a simple workflow using Warden
package main

import (
	"context"
	"fmt"
	"log"

	warden "github.com/diasYuri/warden/pkg"
)

func main() {
	// Creates a simple order processing workflow
	machine, err := warden.New("order-processing").
		StartState("start").
		TaskState("validate_order", validateOrder).
		TaskState("charge_payment", chargePayment).
		TaskState("ship_order", shipOrder).
		EndState("completed").
		Transition("start", "validate_order", warden.TransitionOnSuccess).
		Transition("validate_order", "charge_payment", warden.TransitionOnSuccess).
		Transition("charge_payment", "ship_order", warden.TransitionOnSuccess).
		Transition("ship_order", "completed", warden.TransitionOnSuccess).
		Build()

	if err != nil {
		log.Fatalf("Failed to build machine: %v", err)
	}

	// Creates an instance with order data
	instance, err := machine.CreateInstance("order-12345", map[string]interface{}{
		"order_id":    "12345",
		"customer_id": "cust-789",
		"product_id":  "prod-456",
		"quantity":    2,
		"price":       99.99,
		"email":       "customer@example.com",
	})

	if err != nil {
		log.Fatalf("Failed to create instance: %v", err)
	}

	// Executes the workflow
	ctx := context.Background()
	fmt.Printf("Starting order processing workflow for instance: %s\n", instance.ID)

	err = instance.Start(ctx)
	if err != nil {
		log.Fatalf("Workflow execution failed: %v", err)
	}

	// Prints the result
	if instance.IsCompleted() {
		fmt.Printf("✅ Order processing completed successfully!\n")
		fmt.Printf("Final state: %s\n", instance.GetCurrentState().ID())

		// Prints final data
		data := instance.GetData()
		fmt.Printf("Order data: %+v\n", data)
	} else if instance.IsFailed() {
		fmt.Printf("❌ Order processing failed: %v\n", instance.Error)
	}
}

// validateOrder validates the order
func validateOrder(ctx warden.StateContext) warden.StateResult {
	fmt.Printf("🔍 Validating order %s...\n", ctx.Data["order_id"])

	// Simulates validation logic
	quantity := ctx.Data["quantity"].(int)
	price := ctx.Data["price"].(float64)

	if quantity <= 0 {
		return warden.StateResult{
			Success: false,
			Error:   fmt.Errorf("invalid quantity: %d", quantity),
			Event:   warden.NewEvent(warden.TransitionOnFailure, nil),
		}
	}

	if price <= 0 {
		return warden.StateResult{
			Success: false,
			Error:   fmt.Errorf("invalid price: %.2f", price),
			Event:   warden.NewEvent(warden.TransitionOnFailure, nil),
		}
	}

	// Calculates total
	total := float64(quantity) * price

	fmt.Printf("✅ Order validated successfully. Total: $%.2f\n", total)

	return warden.StateResult{
		Success: true,
		Data: map[string]interface{}{
			"total":     total,
			"validated": true,
		},
		Event: warden.NewEvent(warden.TransitionOnSuccess, nil),
	}
}

// chargePayment charges the payment
func chargePayment(ctx warden.StateContext) warden.StateResult {
	fmt.Printf("💳 Processing payment for order %s...\n", ctx.Data["order_id"])

	total := ctx.Data["total"].(float64)
	customerID := ctx.Data["customer_id"].(string)

	// Simulates payment processing
	fmt.Printf("Charging $%.2f to customer %s\n", total, customerID)

	// Simulates external payment API call
	// In a real scenario, this would be an actual payment gateway call
	transactionID := fmt.Sprintf("txn_%s_%d", ctx.Data["order_id"], 123456)

	fmt.Printf("✅ Payment processed successfully. Transaction ID: %s\n", transactionID)

	return warden.StateResult{
		Success: true,
		Data: map[string]interface{}{
			"transaction_id": transactionID,
			"payment_status": "charged",
			"charged_amount": total,
		},
		Event: warden.NewEvent(warden.TransitionOnSuccess, nil),
	}
}

// shipOrder ships the order
func shipOrder(ctx warden.StateContext) warden.StateResult {
	fmt.Printf("📦 Shipping order %s...\n", ctx.Data["order_id"])

	productID := ctx.Data["product_id"].(string)
	quantity := ctx.Data["quantity"].(int)
	email := ctx.Data["email"].(string)

	// Simulates shipping process
	fmt.Printf("Preparing %d units of product %s for shipping\n", quantity, productID)

	// Generates tracking number
	trackingNumber := fmt.Sprintf("TRK%s%d", ctx.Data["order_id"], 789)

	// Simulates sending notification email
	fmt.Printf("📧 Sending shipping notification to %s\n", email)
	fmt.Printf("✅ Order shipped successfully. Tracking: %s\n", trackingNumber)

	return warden.StateResult{
		Success: true,
		Data: map[string]interface{}{
			"tracking_number":   trackingNumber,
			"shipping_status":   "shipped",
			"notification_sent": true,
		},
		Event: warden.NewEvent(warden.TransitionOnSuccess, nil),
	}
}
