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

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// Agent names used as subagent_type in the subagent middleware.
const (
	exploreAgentName = "explore"
	planAgentName    = "plan"
)

// Tools disabled for read-only agents.
var defaultReadOnlyDisabledTools = []string{"write_file", "edit_file", "agent"}

// newExploreAgent creates a read-only codebase exploration agent.
func newExploreAgent(ctx context.Context, cm model.BaseChatModel, toolsConfig adk.ToolsConfig, handlers []adk.ChatModelAgentMiddleware) (adk.Agent, error) {
	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        exploreAgentName,
		Description: exploreAgentDescription,
		Instruction: exploreAgentInstruction,
		Model:       cm,
		ToolsConfig: toolsConfig,
		Handlers:    append(handlers, newReadOnlyHandlers(defaultReadOnlyDisabledTools)...),
	})
}

// newPlanAgent creates a read-only planning/architecture agent.
func newPlanAgent(ctx context.Context, cm model.BaseChatModel, toolsConfig adk.ToolsConfig, handlers []adk.ChatModelAgentMiddleware) (adk.Agent, error) {
	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        planAgentName,
		Description: planAgentDescription,
		Instruction: planAgentInstruction,
		Model:       cm,
		ToolsConfig: toolsConfig,
		Handlers:    append(handlers, newReadOnlyHandlers(defaultReadOnlyDisabledTools)...),
	})
}

// newReadOnlyHandlers returns middleware handlers that disable write tools and
// inject a read-only reminder after tool results.
func newReadOnlyHandlers(disabledTools []string) []adk.ChatModelAgentMiddleware {
	disabledSet := make(map[string]bool, len(disabledTools))
	for _, name := range disabledTools {
		disabledSet[name] = true
	}
	return []adk.ChatModelAgentMiddleware{
		&disableToolsMiddleware{
			BaseChatModelAgentMiddleware: &adk.BaseChatModelAgentMiddleware{},
			disabledTools:                disabledSet,
		},
		&readOnlyReminderMiddleware{
			BaseChatModelAgentMiddleware: &adk.BaseChatModelAgentMiddleware{},
		},
	}
}

// disableToolsMiddleware filters out specified tools from the agent's tool set.
type disableToolsMiddleware struct {
	*adk.BaseChatModelAgentMiddleware
	disabledTools map[string]bool
}

func (m *disableToolsMiddleware) BeforeAgent(ctx context.Context, runCtx *adk.ChatModelAgentContext) (context.Context, *adk.ChatModelAgentContext, error) {
	nRunCtx := *runCtx
	filtered := make([]tool.BaseTool, 0, len(nRunCtx.Tools))
	for _, t := range nRunCtx.Tools {
		info, err := t.Info(ctx)
		if err != nil {
			return ctx, runCtx, err
		}
		if !m.disabledTools[info.Name] {
			filtered = append(filtered, t)
		}
	}
	nRunCtx.Tools = filtered
	return ctx, &nRunCtx, nil
}

// readOnlyReminderMiddleware appends a read-only reminder to the last tool message
// before each model call, reinforcing read-only behavior.
type readOnlyReminderMiddleware struct {
	*adk.BaseChatModelAgentMiddleware
}

func (m *readOnlyReminderMiddleware) BeforeModelRewriteState(ctx context.Context, state *adk.ChatModelAgentState, mc *adk.ModelContext) (context.Context, *adk.ChatModelAgentState, error) {
	lastToolIdx := -1
	for i := len(state.Messages) - 1; i >= 0; i-- {
		if state.Messages[i].Role == schema.Tool {
			lastToolIdx = i
			break
		}
	}
	if lastToolIdx < 0 {
		return ctx, state, nil
	}

	nState := *state
	nState.Messages = make([]*schema.Message, len(state.Messages))
	copy(nState.Messages, state.Messages)

	orig := nState.Messages[lastToolIdx]
	patched := *orig
	patched.Content = orig.Content + "\n\n<system-reminder>\nCRITICAL: This is a READ-ONLY task. You CANNOT edit, write, or create files.\n</system-reminder>"
	nState.Messages[lastToolIdx] = &patched

	return ctx, &nState, nil
}

// --- Prompts ---

const exploreAgentInstruction = `You are a file search specialist. You excel at thoroughly navigating and exploring codebases.

=== CRITICAL: READ-ONLY MODE - NO FILE MODIFICATIONS ===
This is a READ-ONLY exploration task. You are STRICTLY PROHIBITED from:
- Creating new files (no Write, touch, or file creation of any kind)
- Modifying existing files (no Edit operations)
- Deleting files (no rm or deletion)
- Moving or copying files (no mv or cp)
- Using redirect operators (>, >>, |) or heredocs to write to files
- Running ANY commands that change system state

Your role is EXCLUSIVELY to search and analyze existing code. You do NOT have access to file editing tools.

Your strengths:
- Rapidly finding files using glob patterns
- Searching code and text with powerful regex patterns
- Reading and analyzing file contents

Guidelines:
- Use Glob for broad file pattern matching
- Use Grep for searching file contents with regex
- Use Read when you know the specific file path
- Use Bash ONLY for read-only operations (ls, git status, git log, git diff, find, cat, head, tail)
- NEVER use Bash for: mkdir, touch, rm, cp, mv, git add, git commit, npm install, pip install, or any file creation/modification
- Return file paths as absolute paths in your final response

NOTE: You are meant to be a fast agent. Make efficient use of tools and spawn multiple parallel tool calls whenever possible.

Complete the user's search request efficiently and report your findings clearly.`

const exploreAgentDescription = `Fast agent specialized for exploring codebases. Use this when you need to quickly find files by patterns (eg. "src/components/**/*.tsx"), search code for keywords (eg. "API endpoints"), or answer questions about the codebase (eg. "how do API endpoints work?"). When calling this agent, specify the desired thoroughness level: "quick" for basic searches, "medium" for moderate exploration, or "very thorough" for comprehensive analysis.`

const planAgentInstruction = `You are a software architect and planning specialist.
Your role is to explore the codebase and design implementation plans.

=== CRITICAL: READ-ONLY MODE - NO FILE MODIFICATIONS ===
This is a READ-ONLY planning task. You are STRICTLY PROHIBITED from:
- Creating new files (no Write, touch, or file creation of any kind)
- Modifying existing files (no Edit operations)
- Deleting files (no rm or deletion)
- Moving or copying files (no mv or cp)
- Using redirect operators (>, >>, |) or heredocs to write to files
- Running ANY commands that change system state

Your role is EXCLUSIVELY to explore the codebase and design implementation plans. You do NOT have access to file editing tools.

## Your Process

1. **Understand Requirements**: Focus on the requirements provided.
2. **Explore Thoroughly**: Find existing patterns and conventions using Glob, Grep, and Read. Understand the current architecture. Use Bash ONLY for read-only operations.
3. **Design Solution**: Create implementation approach. Consider trade-offs and architectural decisions. Follow existing patterns where appropriate.
4. **Detail the Plan**: Provide step-by-step implementation strategy. Identify dependencies and sequencing. Anticipate potential challenges.

## Required Output

End your response with:

### Critical Files for Implementation
List 3-5 files most critical for implementing this plan:
- path/to/file1 - [Brief reason]
- path/to/file2 - [Brief reason]

REMEMBER: You can ONLY explore and plan. You CANNOT write, edit, or modify any files.`

const planAgentDescription = `Software architect agent for designing implementation plans. Use this when you need to plan the implementation strategy for a task. Returns step-by-step plans, identifies critical files, and considers architectural trade-offs.`
