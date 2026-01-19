/*
 * Copyright 2025 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"

	"github.com/cloudwego/eino-examples/adk/common/model"
	"github.com/cloudwego/eino-examples/adk/common/prints"
	"github.com/cloudwego/eino-examples/adk/common/store"
	tool2 "github.com/cloudwego/eino-examples/adk/common/tool"
	"github.com/cloudwego/eino-examples/adk/common/tool/graphtool"
)

type OrderInput struct {
	CustomerID string `json:"customer_id" jsonschema_description:"Customer identifier"`
	ProductID  string `json:"product_id" jsonschema_description:"Product identifier to order"`
	Quantity   int    `json:"quantity" jsonschema_description:"Number of items to order"`
}

type OrderOutput struct {
	OrderID      string  `json:"order_id"`
	Status       string  `json:"status"`
	TotalPrice   float64 `json:"total_price"`
	CustomerName string  `json:"customer_name"`
	ProductName  string  `json:"product_name"`
	Quantity     int     `json:"quantity"`
	Message      string  `json:"message"`
}

type validationResult struct {
	Valid       bool
	ProductName string
	UnitPrice   float64
	CustomerID  string
	Quantity    int
}

type pricingResult struct {
	TotalPrice  float64
	ProductName string
	Quantity    int
}

type customerInfo struct {
	CustomerName string
	Email        string
	Address      string
}

type orderContext struct {
	Pricing  *pricingResult
	Customer *customerInfo
}

func mockValidateOrder(ctx context.Context, input *OrderInput) (*validationResult, error) {
	time.Sleep(50 * time.Millisecond)

	products := map[string]struct {
		name  string
		price float64
	}{
		"P100": {"Laptop Pro", 999.99},
		"P101": {"Wireless Mouse", 29.99},
		"P102": {"Mechanical Keyboard", 149.99},
		"P103": {"4K Monitor", 499.99},
	}

	product, exists := products[input.ProductID]
	if !exists {
		return &validationResult{Valid: false}, nil
	}

	if input.Quantity <= 0 || input.Quantity > 100 {
		return &validationResult{Valid: false}, nil
	}

	return &validationResult{
		Valid:       true,
		ProductName: product.name,
		UnitPrice:   product.price,
		CustomerID:  input.CustomerID,
		Quantity:    input.Quantity,
	}, nil
}

func mockCalculatePrice(ctx context.Context, validation *validationResult) (*pricingResult, error) {
	time.Sleep(30 * time.Millisecond)

	total := validation.UnitPrice * float64(validation.Quantity)

	if validation.Quantity >= 10 {
		total *= 0.9
	}

	return &pricingResult{
		TotalPrice:  total,
		ProductName: validation.ProductName,
		Quantity:    validation.Quantity,
	}, nil
}

func mockLookupCustomer(ctx context.Context, validation *validationResult) (*customerInfo, error) {
	time.Sleep(40 * time.Millisecond)

	customers := map[string]*customerInfo{
		"C001": {"Alice Johnson", "alice@example.com", "123 Main St, New York"},
		"C002": {"Bob Smith", "bob@example.com", "456 Oak Ave, Los Angeles"},
		"C003": {"Carol White", "carol@example.com", "789 Pine Rd, Chicago"},
	}

	customer, exists := customers[validation.CustomerID]
	if !exists {
		return &customerInfo{
			CustomerName: "Unknown Customer",
			Email:        "unknown@example.com",
			Address:      "Unknown Address",
		}, nil
	}

	return customer, nil
}

func NewOrderProcessingTool(ctx context.Context) (tool.InvokableTool, error) {
	workflow := compose.NewWorkflow[*OrderInput, *OrderOutput]()

	workflow.AddLambdaNode("validate", compose.InvokableLambda(func(ctx context.Context, input *OrderInput) (*validationResult, error) {
		return mockValidateOrder(ctx, input)
	})).AddInput(compose.START)

	workflow.AddLambdaNode("calculate_price", compose.InvokableLambda(func(ctx context.Context, validation *validationResult) (*pricingResult, error) {
		if !validation.Valid {
			return nil, fmt.Errorf("invalid order: product not found or invalid quantity")
		}
		return mockCalculatePrice(ctx, validation)
	})).AddInput("validate")

	workflow.AddLambdaNode("lookup_customer", compose.InvokableLambda(func(ctx context.Context, validation *validationResult) (*customerInfo, error) {
		return mockLookupCustomer(ctx, validation)
	})).AddInput("validate")

	workflow.AddLambdaNode("generate_confirmation", compose.InvokableLambda(func(ctx context.Context, input *orderContext) (*OrderOutput, error) {
		orderID := fmt.Sprintf("ORD-%d", time.Now().UnixNano()%1000000)

		return &OrderOutput{
			OrderID:      orderID,
			Status:       "confirmed",
			TotalPrice:   input.Pricing.TotalPrice,
			CustomerName: input.Customer.CustomerName,
			ProductName:  input.Pricing.ProductName,
			Quantity:     input.Pricing.Quantity,
			Message:      fmt.Sprintf("Order %s confirmed! %d x %s for %s. Total: $%.2f", orderID, input.Pricing.Quantity, input.Pricing.ProductName, input.Customer.CustomerName, input.Pricing.TotalPrice),
		}, nil
	})).AddInput("calculate_price", compose.ToField("Pricing")).
		AddInput("lookup_customer", compose.ToField("Customer"))

	workflow.End().AddInput("generate_confirmation")

	return graphtool.NewInvokableGraphTool[*OrderInput, *OrderOutput](
		workflow,
		"process_order",
		"Process a customer order. Validates the order, calculates pricing, looks up customer info, and generates a confirmation.",
	)
}

func main() {
	ctx := context.Background()

	innerTool, err := NewOrderProcessingTool(ctx)
	if err != nil {
		log.Fatalf("failed to create order tool: %v", err)
	}

	orderTool := tool2.InvokableApprovableTool{InvokableTool: innerTool}

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "OrderAssistant",
		Description: "An assistant that can process customer orders",
		Instruction: `You are a helpful order processing assistant.
When the user wants to place an order, use the process_order tool with the customer_id, product_id, and quantity.
Available products: P100 (Laptop Pro $999.99), P101 (Wireless Mouse $29.99), P102 (Mechanical Keyboard $149.99), P103 (4K Monitor $499.99).
Available customers: C001 (Alice), C002 (Bob), C003 (Carol).
All orders require human approval before processing.`,
		Model: model.NewChatModel(),
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{orderTool},
			},
		},
	})
	if err != nil {
		log.Fatalf("failed to create agent: %v", err)
	}

	checkpointStore := store.NewInMemoryStore()
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		EnableStreaming: true,
		Agent:           agent,
		CheckPointStore: checkpointStore,
	})

	query := "Place an order for customer C001, product P100 (Laptop Pro), quantity 3"

	fmt.Println("=== Order Processing with Interrupt/Resume Example ===")
	fmt.Println()
	fmt.Println("This example demonstrates using InvokableGraphTool (with compose.Workflow)")
	fmt.Println("wrapped with InvokableApprovableTool for human-in-the-loop approval.")
	fmt.Println()
	fmt.Printf("User Query: %s\n\n", query)

	checkpointID := "order-session-1"
	iter := runner.Query(ctx, query, adk.WithCheckPointID(checkpointID))

	var lastEvent *adk.AgentEvent
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			log.Fatalf("error: %v", event.Err)
		}
		prints.Event(event)
		lastEvent = event
	}

	if lastEvent == nil {
		log.Fatal("no events received")
	}

	if lastEvent.Action != nil && lastEvent.Action.Interrupted != nil {
		fmt.Println("\n--- Order requires approval ---")

		interruptID := lastEvent.Action.Interrupted.InterruptContexts[0].ID

		var approvalResult *tool2.ApprovalResult
		for {
			scanner := bufio.NewScanner(os.Stdin)
			fmt.Print("\nYour decision (Y/N): ")
			scanner.Scan()
			input := strings.TrimSpace(scanner.Text())

			if strings.ToUpper(input) == "Y" {
				approvalResult = &tool2.ApprovalResult{Approved: true}
				break
			} else if strings.ToUpper(input) == "N" {
				fmt.Print("Reason for rejection: ")
				scanner.Scan()
				reason := scanner.Text()
				approvalResult = &tool2.ApprovalResult{Approved: false, DisapproveReason: &reason}
				break
			}
			fmt.Println("Invalid input. Please enter Y or N.")
		}

		fmt.Println("\n--- Resuming order processing ---")

		iter, err = runner.ResumeWithParams(ctx, checkpointID, &adk.ResumeParams{
			Targets: map[string]any{
				interruptID: approvalResult,
			},
		})
		if err != nil {
			log.Fatalf("failed to resume: %v", err)
		}

		for {
			event, ok := iter.Next()
			if !ok {
				break
			}
			if event.Err != nil {
				log.Fatalf("error after resume: %v", event.Err)
			}
			prints.Event(event)
		}
	}

	fmt.Println("\n=== Order Processing Complete ===")
}
