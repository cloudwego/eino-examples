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
	"fmt"

	localbk "github.com/cloudwego/eino-ext/adk/backend/local"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/deep"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"

	commontool "github.com/cloudwego/eino-examples/adk/common/tool"
	"github.com/cloudwego/eino-examples/quickstart/chatwitheino/chatmodel"
	"github.com/cloudwego/eino-examples/quickstart/chatwitheino/helpers"
	"github.com/cloudwego/eino-examples/quickstart/chatwitheino/rag"
)

func buildAgentTyped[M adk.MessageType](ctx context.Context) (adk.TypedResumableAgent[M], error) {
	cm, err := chatmodel.NewModel[M](ctx)
	if err != nil {
		return nil, err
	}

	backend, err := localbk.NewBackend(ctx, &localbk.Config{})
	if err != nil {
		return nil, err
	}

	ragTool, err := rag.BuildTool[M](ctx, cm)
	if err != nil {
		return nil, fmt.Errorf("build rag tool: %w", err)
	}

	middlewareStack, err := buildMiddlewareStack(ctx, cm, backend, ragTool)
	if err != nil {
		return nil, err
	}
	handlers := middlewareStack.Handlers
	handlers = append(handlers, newApprovalMiddleware[M](), helpers.NewSafeToolMiddleware[M]())

	cfg := &deep.TypedConfig[M]{
		Name:           "ChatWithEinoAgent",
		Description:    "An agent that reads and answers questions about documents.",
		ChatModel:      cm,
		Backend:        backend,
		StreamingShell: backend,
		MaxIteration:   50,
		Handlers:       handlers,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: middlewareStack.Tools,
			},
		},
	}
	helpers.ApplyMessageModelRetry(cfg)
	return deep.NewTyped[M](ctx, cfg)
}

// approvalMiddleware intercepts calls to the answer_from_document tool and
// pauses the agent with a human-approval interrupt before executing the RAG
// workflow. The runner's CheckPointStore must be configured for this to work.
type approvalMiddleware[M adk.MessageType] struct {
	*adk.TypedBaseChatModelAgentMiddleware[M]
}

func newApprovalMiddleware[M adk.MessageType]() adk.TypedChatModelAgentMiddleware[M] {
	return &approvalMiddleware[M]{
		TypedBaseChatModelAgentMiddleware: &adk.TypedBaseChatModelAgentMiddleware[M]{},
	}
}

// WrapInvokableToolCall inserts an approval gate around the answer_from_document
// tool. All other tools pass through unchanged.
func (m *approvalMiddleware[M]) WrapInvokableToolCall(
	_ context.Context,
	endpoint adk.InvokableToolCallEndpoint,
	tCtx *adk.ToolContext,
) (adk.InvokableToolCallEndpoint, error) {
	if tCtx.Name != "answer_from_document" {
		return endpoint, nil
	}
	return func(ctx context.Context, args string, opts ...tool.Option) (string, error) {
		wasInterrupted, _, storedArgs := tool.GetInterruptState[string](ctx)
		if !wasInterrupted {
			return "", tool.StatefulInterrupt(ctx, &commontool.ApprovalInfo{
				ToolName:        tCtx.Name,
				ArgumentsInJSON: args,
			}, args)
		}

		isTarget, hasData, data := tool.GetResumeContext[*commontool.ApprovalResult](ctx)
		if isTarget && hasData {
			if data.Approved {
				return endpoint(ctx, storedArgs, opts...)
			}
			if data.DisapproveReason != nil {
				return fmt.Sprintf("tool '%s' disapproved: %s", tCtx.Name, *data.DisapproveReason), nil
			}
			return fmt.Sprintf("tool '%s' disapproved", tCtx.Name), nil
		}

		// Re-interrupt if this is not the resume target (another tool was resumed instead).
		isTarget, _, _ = tool.GetResumeContext[any](ctx)
		if !isTarget {
			return "", tool.StatefulInterrupt(ctx, &commontool.ApprovalInfo{
				ToolName:        tCtx.Name,
				ArgumentsInJSON: storedArgs,
			}, storedArgs)
		}

		return endpoint(ctx, storedArgs, opts...)
	}, nil
}
