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
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/supervisor"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	commonModel "github.com/cloudwego/eino-examples/adk/common/model"
	tool2 "github.com/cloudwego/eino-examples/adk/common/tool"
)

type rateLimitedModel struct {
	m     model.ToolCallingChatModel
	delay time.Duration
}

func (r *rateLimitedModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	newM, err := r.m.WithTools(tools)
	if err != nil {
		return nil, err
	}
	return &rateLimitedModel{newM, r.delay}, nil
}

func (r *rateLimitedModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	time.Sleep(r.delay)
	return r.m.Generate(ctx, input, opts...)
}

func (r *rateLimitedModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	time.Sleep(r.delay)
	return r.m.Stream(ctx, input, opts...)
}

func getRateLimitDelay() time.Duration {
	delayMs := os.Getenv("RATE_LIMIT_DELAY_MS")
	if delayMs == "" {
		return 0
	}
	ms, err := strconv.Atoi(delayMs)
	if err != nil {
		return 0
	}
	return time.Duration(ms) * time.Millisecond
}

func newRateLimitedModel() model.ToolCallingChatModel {
	delay := getRateLimitDelay()
	if delay == 0 {
		return commonModel.NewChatModel()
	}
	return &rateLimitedModel{
		m:     commonModel.NewChatModel(),
		delay: delay,
	}
}

func buildAccountAgent(ctx context.Context) (adk.Agent, error) {
	m := newRateLimitedModel()

	type balanceReq struct {
		AccountID string `json:"account_id" jsonschema_description:"The account ID to check balance for"`
	}

	type balanceResp struct {
		AccountID string  `json:"account_id"`
		Balance   float64 `json:"balance"`
		Currency  string  `json:"currency"`
	}

	checkBalance := func(ctx context.Context, req *balanceReq) (*balanceResp, error) {
		balances := map[string]float64{
			"checking": 5000.00,
			"savings":  15000.00,
			"main":     5000.00,
		}
		balance, ok := balances[req.AccountID]
		if !ok {
			balance = 1000.00
		}
		return &balanceResp{
			AccountID: req.AccountID,
			Balance:   balance,
			Currency:  "USD",
		}, nil
	}

	balanceTool, err := utils.InferTool("check_balance", "Check the balance of a specific account", checkBalance)
	if err != nil {
		return nil, err
	}

	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "account_agent",
		Description: "the agent responsible for checking account information and balances",
		Instruction: `You are an account information agent.

INSTRUCTIONS:
- Assist ONLY with account-related queries like checking balances
- Use the check_balance tool to get account information
- After you're done with your tasks, respond to the supervisor directly
- Respond ONLY with the results of your work, do NOT include ANY other text.`,
		Model: m,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{balanceTool},
			},
		},
	})
}

func buildTransactionAgent(ctx context.Context) (adk.Agent, error) {
	m := newRateLimitedModel()

	type transferReq struct {
		FromAccount string  `json:"from_account" jsonschema_description:"Source account ID"`
		ToAccount   string  `json:"to_account" jsonschema_description:"Destination account ID"`
		Amount      float64 `json:"amount" jsonschema_description:"Amount to transfer"`
		Currency    string  `json:"currency" jsonschema_description:"Currency code (e.g., USD)"`
	}

	type transferResp struct {
		TransactionID string  `json:"transaction_id"`
		Status        string  `json:"status"`
		FromAccount   string  `json:"from_account"`
		ToAccount     string  `json:"to_account"`
		Amount        float64 `json:"amount"`
		Currency      string  `json:"currency"`
		Message       string  `json:"message"`
	}

	transfer := func(ctx context.Context, req *transferReq) (*transferResp, error) {
		return &transferResp{
			TransactionID: "TXN-2025-001234",
			Status:        "completed",
			FromAccount:   req.FromAccount,
			ToAccount:     req.ToAccount,
			Amount:        req.Amount,
			Currency:      req.Currency,
			Message:       fmt.Sprintf("Successfully transferred %.2f %s from %s to %s", req.Amount, req.Currency, req.FromAccount, req.ToAccount),
		}, nil
	}

	transferTool, err := utils.InferTool("transfer_funds", "Transfer funds between accounts. This is a sensitive operation that requires user approval.", transfer)
	if err != nil {
		return nil, err
	}

	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "transaction_agent",
		Description: "the agent responsible for executing financial transactions like fund transfers",
		Instruction: `You are a transaction processing agent.

INSTRUCTIONS:
- Assist ONLY with transaction-related tasks like fund transfers
- Use the transfer_funds tool to execute transfers
- The transfer_funds tool requires user approval before execution
- After you're done with your tasks, respond to the supervisor directly
- Respond ONLY with the results of your work, do NOT include ANY other text.`,
		Model: m,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{
					&tool2.InvokableApprovableTool{InvokableTool: transferTool},
				},
			},
		},
	})
}

func buildFinancialSupervisor(ctx context.Context) (adk.Agent, error) {
	m := newRateLimitedModel()

	sv, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "financial_supervisor",
		Description: "the supervisor agent responsible for coordinating financial tasks",
		Instruction: `You are a financial advisor supervisor managing two agents:

- an account_agent: Assign account-related tasks to this agent (checking balances, account info)
- a transaction_agent: Assign transaction-related tasks to this agent (fund transfers, payments)

INSTRUCTIONS:
- Analyze the user's request and delegate to the appropriate agent
- For requests involving both checking balances AND making transfers, first delegate to account_agent, then to transaction_agent
- Assign work to one agent at a time, do not call agents in parallel
- Do not do any work yourself - always delegate to the appropriate agent
- After all tasks are complete, summarize the results for the user`,
		Model: m,
		Exit:  &adk.ExitTool{},
	})
	if err != nil {
		return nil, err
	}

	accountAgent, err := buildAccountAgent(ctx)
	if err != nil {
		return nil, err
	}
	transactionAgent, err := buildTransactionAgent(ctx)
	if err != nil {
		return nil, err
	}

	return supervisor.New(ctx, &supervisor.Config{
		Supervisor: sv,
		SubAgents:  []adk.Agent{accountAgent, transactionAgent},
	})
}
