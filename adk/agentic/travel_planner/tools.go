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
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

type lookupUserProfileInput struct {
	Destination string `json:"destination,omitempty" jsonschema_description:"目的地城市或区域。"`
}

type lookupTravelConstraintsInput struct {
	TripLength string `json:"trip_length,omitempty" jsonschema_description:"行程长度，例如两天。"`
}

type bookingOptionInput struct {
	Name             string  `json:"name" jsonschema_description:"需要评估的体验、场馆、展览、咖啡馆或活动名称。"`
	Category         string  `json:"category,omitempty" jsonschema_description:"类别，例如展览、博物馆、咖啡、散步、演出。"`
	Date             string  `json:"date,omitempty" jsonschema_description:"计划日期或时间段。"`
	Action           string  `json:"action" jsonschema_description:"Agent 想执行的动作，例如 recommend、reserve、pay。"`
	EstimatedCostCNY float64 `json:"estimated_cost_cny,omitempty" jsonschema_description:"预估单项费用，单位人民币。"`
	Notes            string  `json:"notes,omitempty" jsonschema_description:"简要说明这个候选项为什么值得评估。"`
}

func buildTools(planPath string) ([]tool.BaseTool, error) {
	lookupProfile, err := utils.InferTool(
		"lookup_user_profile",
		"读取本示例中稳定的用户旅行偏好。",
		func(ctx context.Context, input *lookupUserProfileInput) (string, error) {
			if input == nil {
				input = &lookupUserProfileInput{}
			}
			destination := input.Destination
			if destination == "" {
				destination = "杭州"
			}
			return mustJSON(map[string]any{
				"destination":        destination,
				"interests":          []string{"展览", "博物馆", "精品咖啡", "轻松的湖边散步"},
				"pace":               "节奏放松，留足休息时间，不做赶场式清单",
				"walking_preference": "喜欢舒服的步行，但雨天或高温时避免过长步行",
				"budget_cny":         demoBudgetCNY,
				"food_preference":    "偏好轻食和咖啡馆，避免把行程排得过满",
			})
		},
	)
	if err != nil {
		return nil, err
	}

	lookupConstraints, err := utils.InferTool(
		"lookup_travel_constraints",
		"读取本示例中的行程硬约束和策略约束。",
		func(ctx context.Context, input *lookupTravelConstraintsInput) (string, error) {
			if input == nil {
				input = &lookupTravelConstraintsInput{}
			}
			tripLength := input.TripLength
			if tripLength == "" {
				tripLength = "两天"
			}
			return mustJSON(map[string]any{
				"trip_length":            tripLength,
				"trip_dates":             "这个周末",
				"must_verify_with_web":   []string{"天气", "开放状态", "当前展览", "会影响路线的实时信息"},
				"route_expectation":      "基于核验后的地点信息安排紧凑路线，减少不必要的折返。",
				"booking_policy":         fmt.Sprintf("Agent 可以评估候选体验，但不得自动付款；单项自动推荐预算上限为 %.0f 元。", maxAutoBookingOptionCNY),
				"auto_booking_limit_cny": maxAutoBookingOptionCNY,
				"final_file_path":        planPath,
			})
		},
	)
	if err != nil {
		return nil, err
	}

	evaluateBookingOption, err := utils.InferTool(
		"evaluate_booking_option",
		"评估一个旅行候选项是否适合进入计划。middleware 会拒绝超出预算上限或试图自动付款的动作。",
		func(ctx context.Context, input *bookingOptionInput) (string, error) {
			if input == nil {
				input = &bookingOptionInput{}
			}
			return mustJSON(map[string]any{
				"status":             "option_recorded",
				"name":               input.Name,
				"category":           input.Category,
				"date":               input.Date,
				"action":             input.Action,
				"estimated_cost_cny": input.EstimatedCostCNY,
				"message":            "候选项已记录为可推荐选项，没有发生真实预约、付款或下单。",
			})
		},
	)
	if err != nil {
		return nil, err
	}

	return []tool.BaseTool{
		lookupProfile,
		lookupConstraints,
		evaluateBookingOption,
	}, nil
}

func mustJSON(v any) (string, error) {
	bs, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return string(bs), nil
}
