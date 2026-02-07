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

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"github.com/cloudwego/eino-examples/adk/common/model"
	"github.com/cloudwego/eino-examples/adk/common/prints"
	"github.com/cloudwego/eino-examples/adk/common/store"
	tool2 "github.com/cloudwego/eino-examples/adk/common/tool"
	"github.com/cloudwego/eino-examples/adk/common/tool/graphtool"
)

type TransferInput struct {
	FromAccount string  `json:"from_account" jsonschema_description:"Source account ID"`
	ToAccount   string  `json:"to_account" jsonschema_description:"Destination account ID"`
	Amount      float64 `json:"amount" jsonschema_description:"Amount to transfer"`
}

type TransferOutput struct {
	TransactionID string  `json:"transaction_id"`
	Status        string  `json:"status"`
	Message       string  `json:"message"`
	FromBalance   float64 `json:"from_balance"`
	ToBalance     float64 `json:"to_balance"`
}

type InternalApprovalInfo struct {
	Step    string
	Message string
}

func (ai *InternalApprovalInfo) String() string {
	return fmt.Sprintf("\n[INTERNAL WORKFLOW APPROVAL]\nStep: %s\nMessage: %s\nApprove? (Y/N):", ai.Step, ai.Message)
}

type InternalApprovalResult struct {
	Approved bool
	Comment  string
}

func init() {
	schema.Register[*InternalApprovalInfo]()
	schema.Register[*InternalApprovalResult]()
	schema.Register[*validationResult]()
}

type validationResult struct {
	Valid       bool
	FromAccount string
	ToAccount   string
	Amount      float64
}

func NewTransferToolWithInternalInterrupt(ctx context.Context) (tool.InvokableTool, error) {
	workflow := compose.NewWorkflow[*TransferInput, *TransferOutput]()

	workflow.AddLambdaNode("validate", compose.InvokableLambda(func(ctx context.Context, input *TransferInput) (*validationResult, error) {
		fmt.Println("  [Workflow] Validating transfer...")
		if input.Amount <= 0 {
			return &validationResult{Valid: false}, nil
		}
		return &validationResult{
			Valid:       true,
			FromAccount: input.FromAccount,
			ToAccount:   input.ToAccount,
			Amount:      input.Amount,
		}, nil
	})).AddInput(compose.START)

	workflow.AddLambdaNode("risk_check_and_execute", compose.InvokableLambda(func(ctx context.Context, validation *validationResult) (*TransferOutput, error) {
		wasInterrupted, _, storedValidation := compose.GetInterruptState[*validationResult](ctx)

		if wasInterrupted {
			fmt.Println("  [Workflow] Resuming from interrupt...")
			isTarget, hasData, data := compose.GetResumeContext[*InternalApprovalResult](ctx)

			if !isTarget {
				fmt.Println("  [Workflow] Not resume target, re-interrupting...")
				return nil, compose.StatefulInterrupt(ctx, &InternalApprovalInfo{
					Step:    "risk_check",
					Message: fmt.Sprintf("High-value transfer of $%.2f requires risk team approval", storedValidation.Amount),
				}, storedValidation)
			}

			if !hasData {
				return nil, fmt.Errorf("resumed without approval data")
			}

			if !data.Approved {
				return &TransferOutput{
					Status:  "rejected",
					Message: fmt.Sprintf("Transfer rejected by risk team: %s", data.Comment),
				}, nil
			}

			fmt.Printf("  [Workflow] Risk team approved with comment: %s\n", data.Comment)
			fmt.Println("  [Workflow] Executing transfer...")
			return &TransferOutput{
				TransactionID: "TXN-12345",
				Status:        "completed",
				Message:       fmt.Sprintf("Transfer of $%.2f completed (risk approved)", storedValidation.Amount),
				FromBalance:   10000 - storedValidation.Amount,
				ToBalance:     5000 + storedValidation.Amount,
			}, nil
		}

		if !validation.Valid {
			return &TransferOutput{
				Status:  "rejected",
				Message: "Invalid transfer: validation failed",
			}, nil
		}

		fmt.Println("  [Workflow] Performing risk check...")
		if validation.Amount > 1000 {
			fmt.Println("  [Workflow] High-value transfer detected, triggering INTERNAL interrupt...")
			return nil, compose.StatefulInterrupt(ctx, &InternalApprovalInfo{
				Step:    "risk_check",
				Message: fmt.Sprintf("High-value transfer of $%.2f requires risk team approval", validation.Amount),
			}, validation)
		}

		fmt.Println("  [Workflow] Low-value transfer, executing directly...")
		return &TransferOutput{
			TransactionID: "TXN-12345",
			Status:        "completed",
			Message:       fmt.Sprintf("Transfer of $%.2f completed", validation.Amount),
			FromBalance:   10000 - validation.Amount,
			ToBalance:     5000 + validation.Amount,
		}, nil
	})).AddInput("validate")

	workflow.End().AddInput("risk_check_and_execute")

	return graphtool.NewInvokableGraphTool[*TransferInput, *TransferOutput](
		workflow,
		"transfer_funds",
		"Transfer funds between accounts. High-value transfers (>$1000) require internal risk approval.",
	)
}

func main() {
	ctx := context.Background()

	innerTool, err := NewTransferToolWithInternalInterrupt(ctx)
	if err != nil {
		log.Fatalf("failed to create transfer tool: %v", err)
	}

	transferTool := tool2.InvokableApprovableTool{InvokableTool: innerTool}

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "TransferAssistant",
		Description: "An assistant that can transfer funds between accounts",
		Instruction: `You are a helpful banking assistant.
When the user wants to transfer funds, IMMEDIATELY use the transfer_funds tool without asking for confirmation.
All transfers require initial approval. High-value transfers (>$1000) also require internal risk team approval.`,
		Model: model.NewChatModel(),
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{transferTool},
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

	query := "Transfer $1500 from account A001 to account B002"

	fmt.Println("=== Nested Interrupt Test ===")
	fmt.Println()
	fmt.Println("This example tests:")
	fmt.Println("1. InvokableApprovableTool wraps InvokableGraphTool")
	fmt.Println("2. The inner workflow has its own interrupt (risk check)")
	fmt.Println("3. Both interrupts should work independently")
	fmt.Println()
	fmt.Printf("User Query: %s\n\n", query)

	checkpointID := "nested-interrupt-test"
	iter := runner.Query(ctx, query, adk.WithCheckPointID(checkpointID))

	interruptCount := 0
	for {
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
			break
		}

		if lastEvent.Action != nil && lastEvent.Action.Interrupted != nil {
			interruptCount++
			fmt.Printf("\n--- Interrupt #%d detected ---\n", interruptCount)

			interruptID := lastEvent.Action.Interrupted.InterruptContexts[0].ID
			fmt.Printf("Interrupt ID: %s\n, Address: %v\n, Info: %v\n", interruptID,
				lastEvent.Action.Interrupted.InterruptContexts[0].Address,
				lastEvent.Action.Interrupted.InterruptContexts[0].Info)

			scanner := bufio.NewScanner(os.Stdin)
			fmt.Print("\nYour decision (Y/N): ")
			scanner.Scan()
			input := strings.TrimSpace(scanner.Text())

			var resumeData any
			if strings.ToUpper(input) == "Y" {
				if interruptCount == 1 {
					resumeData = &tool2.ApprovalResult{Approved: true}
				} else {
					resumeData = &InternalApprovalResult{Approved: true, Comment: "Risk approved by manager"}
				}
			} else {
				if interruptCount == 1 {
					reason := "User rejected"
					resumeData = &tool2.ApprovalResult{Approved: false, DisapproveReason: &reason}
				} else {
					resumeData = &InternalApprovalResult{Approved: false, Comment: "Risk team rejected"}
				}
			}

			fmt.Printf("\n--- Resuming (interrupt #%d) ---\n\n", interruptCount)

			iter, err = runner.ResumeWithParams(ctx, checkpointID, &adk.ResumeParams{
				Targets: map[string]any{
					interruptID: resumeData,
				},
			})
			if err != nil {
				log.Fatalf("failed to resume: %v", err)
			}
			continue
		}

		break
	}

	fmt.Printf("\n=== Test Complete (Total interrupts: %d) ===\n", interruptCount)
}
