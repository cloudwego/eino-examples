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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/hertz-contrib/sse"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"

	"github.com/cloudwego/eino-examples/adk/common/model"
)

type SSEEvent struct {
	Type       string            `json:"type"`
	AgentName  string            `json:"agent_name,omitempty"`
	RunPath    string            `json:"run_path,omitempty"`
	Content    string            `json:"content,omitempty"`
	ToolCalls  []schema.ToolCall `json:"tool_calls,omitempty"`
	ActionType string            `json:"action_type,omitempty"`
	Error      string            `json:"error,omitempty"`
}

func main() {
	ctx := context.Background()

	agent, err := createAgent(ctx)
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		EnableStreaming: true,
		Agent:           agent,
	})

	h := server.Default(server.WithHostPorts(":8080"))

	h.GET("/chat", func(ctx context.Context, c *app.RequestContext) {
		handleChat(ctx, c, runner)
	})

	log.Println("Server starting on http://localhost:8080")
	log.Println("Try: curl -N 'http://localhost:8080/chat?query=tell me a short story'")
	h.Spin()
}

func createAgent(ctx context.Context) (adk.Agent, error) {
	// add sub-agents if you want to.
	// for demonstration purpose we use a simple ChatModelAgent
	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "SSEAgent",
		Description: "An agent that responds via Server-Sent Events",
		Instruction: `You are a helpful assistant. Provide clear and concise responses to user queries.`,
		Model:       model.NewChatModel(),
		// add tools if you want to
	})
}

func formatRunPath(runPath []adk.RunStep) string {
	return fmt.Sprintf("%v", runPath)
}

func handleChat(ctx context.Context, c *app.RequestContext, runner *adk.Runner) {
	query := c.Query("query")
	if query == "" {
		c.JSON(consts.StatusBadRequest, map[string]string{
			"error": "query parameter is required",
		})
		return
	}

	log.Printf("Received query: %s", query)

	iter := runner.Query(ctx, query)

	s := sse.NewStream(c)
	defer func(c *app.RequestContext) {
		_ = c.Flush()
	}(c)

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if err := processAgentEvent(ctx, s, event); err != nil {
			log.Printf("Error processing event: %v", err)
			break
		}
	}
}

func processAgentEvent(ctx context.Context, s *sse.Stream, event *adk.AgentEvent) error {
	if event.Err != nil {
		return sendSSEEvent(s, SSEEvent{
			Type:      "error",
			AgentName: event.AgentName,
			RunPath:   formatRunPath(event.RunPath),
			Error:     event.Err.Error(),
		})
	}

	if event.Output != nil && event.Output.MessageOutput != nil {
		if err := handleMessageOutput(ctx, s, event); err != nil {
			return err
		}
	}

	if event.Action != nil {
		if err := handleAction(s, event); err != nil {
			return err
		}
	}

	return nil
}

func handleMessageOutput(ctx context.Context, s *sse.Stream, event *adk.AgentEvent) error {
	msgOutput := event.Output.MessageOutput

	if msg := msgOutput.Message; msg != nil {
		return handleRegularMessage(s, event, msg)
	}

	if stream := msgOutput.MessageStream; stream != nil {
		return handleStreamingMessage(ctx, s, event, stream)
	}

	return nil
}

func handleRegularMessage(s *sse.Stream, event *adk.AgentEvent, msg *schema.Message) error {
	eventType := "message"
	if msg.Role == schema.Tool {
		eventType = "tool_result"
	}

	sseEvent := SSEEvent{
		Type:      eventType,
		AgentName: event.AgentName,
		RunPath:   formatRunPath(event.RunPath),
		Content:   msg.Content,
	}

	if len(msg.ToolCalls) > 0 {
		sseEvent.ToolCalls = msg.ToolCalls
	}

	return sendSSEEvent(s, sseEvent)
}

func handleStreamingMessage(ctx context.Context, s *sse.Stream, event *adk.AgentEvent, stream *schema.StreamReader[*schema.Message]) error {
	toolCallsMap := make(map[int][]*schema.Message)

	for {
		chunk, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return sendSSEEvent(s, SSEEvent{
				Type:      "error",
				AgentName: event.AgentName,
				RunPath:   formatRunPath(event.RunPath),
				Error:     fmt.Sprintf("stream error: %v", err),
			})
		}

		if chunk.Content != "" {
			eventType := "stream_chunk"
			if chunk.Role == schema.Tool {
				eventType = "tool_result_chunk"
			}

			if err := sendSSEEvent(s, SSEEvent{
				Type:      eventType,
				AgentName: event.AgentName,
				RunPath:   formatRunPath(event.RunPath),
				Content:   chunk.Content,
			}); err != nil {
				return err
			}
		}

		if len(chunk.ToolCalls) > 0 {
			for _, tc := range chunk.ToolCalls {
				if tc.Index != nil {
					toolCallsMap[*tc.Index] = append(toolCallsMap[*tc.Index], &schema.Message{
						Role: chunk.Role,
						ToolCalls: []schema.ToolCall{
							{
								ID:    tc.ID,
								Type:  tc.Type,
								Index: tc.Index,
								Function: schema.FunctionCall{
									Name:      tc.Function.Name,
									Arguments: tc.Function.Arguments,
								},
							},
						},
					})
				}
			}
		}
	}

	for _, msgs := range toolCallsMap {
		concatenatedMsg, err := schema.ConcatMessages(msgs)
		if err != nil {
			return err
		}

		if err := sendSSEEvent(s, SSEEvent{
			Type:      "tool_calls",
			AgentName: event.AgentName,
			RunPath:   formatRunPath(event.RunPath),
			ToolCalls: concatenatedMsg.ToolCalls,
		}); err != nil {
			return err
		}
	}

	return nil
}

func handleAction(s *sse.Stream, event *adk.AgentEvent) error {
	action := event.Action

	if action.TransferToAgent != nil {
		return sendSSEEvent(s, SSEEvent{
			Type:       "action",
			AgentName:  event.AgentName,
			RunPath:    formatRunPath(event.RunPath),
			ActionType: "transfer",
			Content:    fmt.Sprintf("Transfer to agent: %s", action.TransferToAgent.DestAgentName),
		})
	}

	if action.Interrupted != nil {
		for _, ic := range action.Interrupted.InterruptContexts {
			content := fmt.Sprintf("%v", ic.Info)
			if stringer, ok := ic.Info.(fmt.Stringer); ok {
				content = stringer.String()
			}

			if err := sendSSEEvent(s, SSEEvent{
				Type:       "action",
				AgentName:  event.AgentName,
				RunPath:    formatRunPath(event.RunPath),
				ActionType: "interrupted",
				Content:    content,
			}); err != nil {
				return err
			}
		}
	}

	if action.Exit {
		return sendSSEEvent(s, SSEEvent{
			Type:       "action",
			AgentName:  event.AgentName,
			RunPath:    formatRunPath(event.RunPath),
			ActionType: "exit",
			Content:    "Agent execution completed",
		})
	}

	return nil
}

func sendSSEEvent(s *sse.Stream, event SSEEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal SSE event: %w", err)
	}

	return s.Publish(&sse.Event{
		Data: data,
	})
}
