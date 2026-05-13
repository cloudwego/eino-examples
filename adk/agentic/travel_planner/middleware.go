/*
 * Copyright 2026 CloudWeGo Authors
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
	"encoding/json"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

type travelPolicyMiddleware struct {
	*adk.TypedBaseChatModelAgentMiddleware[*schema.AgenticMessage]
	maxAutoBookingCostCNY float64
}

func newTravelPolicyMiddleware(maxAutoBookingCostCNY float64) adk.TypedChatModelAgentMiddleware[*schema.AgenticMessage] {
	return &travelPolicyMiddleware{
		TypedBaseChatModelAgentMiddleware: &adk.TypedBaseChatModelAgentMiddleware[*schema.AgenticMessage]{},
		maxAutoBookingCostCNY:             maxAutoBookingCostCNY,
	}
}

func (m *travelPolicyMiddleware) WrapInvokableToolCall(
	_ context.Context,
	endpoint adk.InvokableToolCallEndpoint,
	tCtx *adk.ToolContext,
) (adk.InvokableToolCallEndpoint, error) {
	if tCtx == nil || tCtx.Name != "evaluate_booking_option" {
		return endpoint, nil
	}

	return func(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
		var input bookingOptionInput
		if err := json.Unmarshal([]byte(argumentsInJSON), &input); err != nil {
			return "", err
		}

		if decision, blocked := m.review(input); blocked {
			return mustJSON(decision)
		}

		return endpoint(ctx, argumentsInJSON, opts...)
	}, nil
}

func (m *travelPolicyMiddleware) review(input bookingOptionInput) (map[string]any, bool) {
	var reasons []string

	action := strings.ToLower(input.Action)
	if strings.Contains(action, "pay") ||
		strings.Contains(action, "purchase") {
		reasons = append(reasons, "示例策略禁止 Agent 自动付款或购买")
	}
	if input.EstimatedCostCNY > m.maxAutoBookingCostCNY {
		reasons = append(reasons, "预估费用超过示例允许自动推荐的单项预算上限")
	}

	if len(reasons) == 0 {
		return nil, false
	}

	return map[string]any{
		"status":             "policy_rejected",
		"allowed":            false,
		"intercepted_by":     "travelPolicyMiddleware",
		"name":               input.Name,
		"category":           input.Category,
		"action":             input.Action,
		"estimated_cost_cny": input.EstimatedCostCNY,
		"limit_cny":          m.maxAutoBookingCostCNY,
		"reasons":            reasons,
		"next_step":          "不要把该候选项写成默认推荐或已执行动作。请选择更低成本、无需付款的替代方案。",
	}, true
}
