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
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	localbk "github.com/cloudwego/eino-ext/adk/backend/local"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/middlewares/skill"
	"github.com/cloudwego/eino/adk/prebuilt/deep"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	commontool "github.com/cloudwego/eino-examples/adk/common/tool"
	"github.com/cloudwego/eino-examples/quickstart/chatwitheino/chatmodel"
	"github.com/cloudwego/eino-examples/quickstart/chatwitheino/msgops"
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

	var handlers []adk.TypedChatModelAgentMiddleware[M]
	if skillsDir, ok := resolveSkillsDir(); ok {
		skillBackend, sbErr := skill.NewBackendFromFilesystem(ctx, &skill.BackendFromFilesystemConfig{
			Backend: backend,
			BaseDir: skillsDir,
		})
		if sbErr != nil {
			return nil, sbErr
		}
		skillMiddleware, smErr := skill.NewTyped[M](ctx, &skill.TypedConfig[M]{
			Backend: skillBackend,
		})
		if smErr != nil {
			return nil, smErr
		}
		handlers = append(handlers, skillMiddleware)
	}
	handlers = append(handlers, newApprovalMiddleware[M](), newSafeToolMiddleware[M]())

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
				Tools: []tool.BaseTool{ragTool},
			},
		},
	}
	if msgops.KindOf[M]() == msgops.KindMessage {
		cfg.ModelRetryConfig = &adk.TypedModelRetryConfig[M]{
			MaxRetries: 5,
			IsRetryAble: func(_ context.Context, err error) bool {
				return strings.Contains(err.Error(), "429") ||
					strings.Contains(err.Error(), "Too Many Requests") ||
					strings.Contains(err.Error(), "qpm limit")
			},
		}
	}
	return deep.NewTyped[M](ctx, cfg)
}

func resolveSkillsDir() (string, bool) {
	skillsDir := strings.TrimSpace(os.Getenv("EINO_EXT_SKILLS_DIR"))
	if skillsDir == "" {
		return "", false
	}
	if absSkillsDir, absErr := filepath.Abs(skillsDir); absErr == nil {
		skillsDir = absSkillsDir
	}
	fi, err := os.Stat(skillsDir)
	if err != nil || !fi.IsDir() {
		return "", false
	}
	return skillsDir, true
}

// safeToolMiddleware converts streaming tool errors into error-message strings
// so that a non-zero exit code or mid-stream failure is returned to the model
// as a readable tool result instead of aborting the agent pipeline.
type safeToolMiddleware[M adk.MessageType] struct {
	*adk.TypedBaseChatModelAgentMiddleware[M]
}

// approvalMiddleware intercepts calls to the answer_from_document tool and
// pauses the agent with a human-approval interrupt before executing the RAG
// workflow. The runner's CheckPointStore must be configured for this to work.
type approvalMiddleware[M adk.MessageType] struct {
	*adk.TypedBaseChatModelAgentMiddleware[M]
}

func newSafeToolMiddleware[M adk.MessageType]() adk.TypedChatModelAgentMiddleware[M] {
	return &safeToolMiddleware[M]{
		TypedBaseChatModelAgentMiddleware: &adk.TypedBaseChatModelAgentMiddleware[M]{},
	}
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

func (m *safeToolMiddleware[M]) WrapInvokableToolCall(
	_ context.Context,
	endpoint adk.InvokableToolCallEndpoint,
	_ *adk.ToolContext,
) (adk.InvokableToolCallEndpoint, error) {
	return func(ctx context.Context, args string, opts ...tool.Option) (string, error) {
		result, err := endpoint(ctx, args, opts...)
		if err != nil {
			if _, ok := compose.IsInterruptRerunError(err); ok {
				return "", err
			}
			return fmt.Sprintf("[tool error] %v", err), nil
		}
		return result, nil
	}, nil
}

func (m *safeToolMiddleware[M]) WrapStreamableToolCall(
	_ context.Context,
	endpoint adk.StreamableToolCallEndpoint,
	_ *adk.ToolContext,
) (adk.StreamableToolCallEndpoint, error) {
	return func(ctx context.Context, args string, opts ...tool.Option) (*schema.StreamReader[string], error) {
		sr, err := endpoint(ctx, args, opts...)
		if err != nil {
			if _, ok := compose.IsInterruptRerunError(err); ok {
				return nil, err
			}
			return singleChunkReader(fmt.Sprintf("[tool error] %v", err)), nil
		}
		return safeWrapReader(sr), nil
	}, nil
}

// singleChunkReader returns a StreamReader that emits one string then EOF.
func singleChunkReader(msg string) *schema.StreamReader[string] {
	r, w := schema.Pipe[string](1)
	_ = w.Send(msg, nil)
	w.Close()
	return r
}

// safeWrapReader proxies chunks from sr; on a stream error it emits the error
// as a final chunk instead of propagating it, so the model sees a complete
// (if error-annotated) tool result rather than a pipeline failure.
func safeWrapReader(sr *schema.StreamReader[string]) *schema.StreamReader[string] {
	r, w := schema.Pipe[string](64)
	go func() {
		defer w.Close()
		for {
			chunk, err := sr.Recv()
			if errors.Is(err, io.EOF) {
				return
			}
			if err != nil {
				_ = w.Send(fmt.Sprintf("\n[tool error] %v", err), nil)
				return
			}
			_ = w.Send(chunk, nil)
		}
	}()
	return r
}
