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

	"github.com/cloudwego/eino-examples/adk/common/model"
)

func NewWriterAgent() adk.Agent {
	ctx := context.Background()

	a, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "WriterAgent",
		Description: "An agent that can write poems",
		Instruction: `You are an expert writer that can write poems. 
If feedback is received for the previous version of your poem, you need to modify the poem according to the feedback.
Your response should ALWAYS contain ONLY the poem, and nothing else.`,
		Model:     model.NewChatModel(),
		OutputKey: "content_to_review",
	})
	if err != nil {
		log.Fatal(fmt.Errorf("failed to create chatmodel: %w", err))
	}

	la, err := adk.NewLoopAgent(ctx, &adk.LoopAgentConfig{
		Name:        "Writer MultiAgent",
		Description: "An agent that can write poems",
		SubAgents: []adk.Agent{a,
			&ReviewAgent{AgentName: "ReviewerAgent", AgentDesc: "An agent that can review poems"}},
	})
	if err != nil {
		log.Fatal(fmt.Errorf("failed to create loopagent: %w", err))
	}

	return la
}
