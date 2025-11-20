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
	"log"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"

	"github.com/cloudwego/eino-examples/adk/common/model"
	tool2 "github.com/cloudwego/eino-examples/adk/common/tool"
)

func NewTicketBookingAgent() adk.Agent {
	ctx := context.Background()

	type bookInput struct {
		Location             string `json:"location"`
		PassengerName        string `json:"passenger_name"`
		PassengerPhoneNumber string `json:"passenger_phone_number"`
	}

	getWeather, err := utils.InferTool(
		"BookTicket",
		"this tool can book ticket of the specific location",
		func(ctx context.Context, input bookInput) (output string, err error) {
			return "success", nil
		})
	if err != nil {
		log.Fatal(err)
	}

	a, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "TicketBooker",
		Description: "An agent that can book tickets",
		Instruction: `You are an expert ticket booker.
Based on the user's request, use the "BookTicket" tool to book tickets.`,
		Model: model.NewChatModel(),
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{
					&tool2.InvokableApprovableTool{InvokableTool: getWeather},
				},
			},
		},
	})
	if err != nil {
		log.Fatal(fmt.Errorf("failed to create chatmodel: %w", err))
	}

	return a
}
