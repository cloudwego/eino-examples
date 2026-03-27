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

package a2ui

import (
	"errors"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

// RenderHistory writes the beginRendering + history surfaceUpdate messages to w
// without running an agent. Used to populate the chat window when a session is selected.
func RenderHistory(w io.Writer, sessionID string, history []*schema.Message) error {
	surfaceID := "chat-" + sessionID
	rootChildren := make([]string, 0, len(history))
	for i := range history {
		rootChildren = append(rootChildren, fmt.Sprintf("msg-%d-card", i))
	}
	if err := emit(w, Message{
		BeginRendering: &BeginRenderingMsg{SurfaceID: surfaceID, Root: "root-col"},
	}); err != nil {
		return err
	}
	return emitHistory(w, surfaceID, history, rootChildren)
}

// StreamToWriter converts an agent event stream into A2UI JSONL messages written to w.
// It returns the content of the last assistant text response, the interrupt ID if the
// agent was paused awaiting human approval (non-empty), the final A2UI msgIdx, and any error.
func StreamToWriter(w io.Writer, sessionID string, history []*schema.Message, events *adk.AsyncIterator[*adk.AgentEvent]) (string, string, int, error) {
	surfaceID := "chat-" + sessionID

	rootChildren := make([]string, 0, len(history))
	for i := range history {
		rootChildren = append(rootChildren, fmt.Sprintf("msg-%d-card", i))
	}

	if err := emit(w, Message{
		BeginRendering: &BeginRenderingMsg{SurfaceID: surfaceID, Root: "root-col"},
	}); err != nil {
		return "", "", 0, err
	}
	if err := emitHistory(w, surfaceID, history, rootChildren); err != nil {
		return "", "", 0, err
	}

	msgIdx := len(history)
	lastContent, interruptID, err := streamEvents(w, surfaceID, &rootChildren, &msgIdx, events)
	return lastContent, interruptID, msgIdx, err
}

// StreamContinue resumes an interrupted stream without resetting the client UI.
// It continues from startMsgIdx, appending new chips to the existing component tree.
func StreamContinue(w io.Writer, sessionID string, startMsgIdx int, events *adk.AsyncIterator[*adk.AgentEvent]) (string, string, int, error) {
	surfaceID := "chat-" + sessionID

	// Reconstruct rootChildren to match the client's current component tree.
	rootChildren := make([]string, startMsgIdx)
	for i := range rootChildren {
		rootChildren[i] = fmt.Sprintf("msg-%d-card", i)
	}

	msgIdx := startMsgIdx
	lastContent, interruptID, err := streamEvents(w, surfaceID, &rootChildren, &msgIdx, events)
	return lastContent, interruptID, msgIdx, err
}

// streamEvents is the shared event-processing loop used by StreamToWriter and StreamContinue.
func streamEvents(w io.Writer, surfaceID string, rootChildren *[]string, msgIdx *int, events *adk.AsyncIterator[*adk.AgentEvent]) (string, string, error) {
	var lastContent strings.Builder
	var interruptID string

	for {
		event, ok := events.Next()
		if !ok {
			log.Printf("[a2ui] event stream ended (iterator exhausted)")
			break
		}

		if event.Err != nil {
			log.Printf("[a2ui] event error: %v", event.Err)
			_ = emitToolChip(w, surfaceID, rootChildren, msgIdx, "error", event.Err.Error())
			return lastContent.String(), "", event.Err
		}

		// Detect interrupt: the agent is paused awaiting human input.
		if event.Action != nil && event.Action.Interrupted != nil {
			ictxs := event.Action.Interrupted.InterruptContexts
			var desc string
			for _, ic := range ictxs {
				if ic.IsRootCause {
					interruptID = ic.ID
					desc = fmt.Sprintf("%v", ic.Info)
					break
				}
			}
			if interruptID == "" && len(ictxs) > 0 {
				interruptID = ictxs[0].ID
				desc = fmt.Sprintf("%v", ictxs[0].Info)
			}
			log.Printf("[a2ui] interrupt: id=%s desc=%q", interruptID, desc)
			_ = emitToolChip(w, surfaceID, rootChildren, msgIdx, "approval needed", desc)
			_ = emit(w, Message{
				InterruptRequest: &InterruptRequestMsg{
					InterruptID: interruptID,
					Description: desc,
				},
			})
			break
		}

		hasOutput := event.Output != nil && event.Output.MessageOutput != nil
		hasExit := event.Action != nil && event.Action.Exit
		log.Printf("[a2ui] event: hasOutput=%v hasExit=%v", hasOutput, hasExit)

		if !hasOutput {
			if hasExit {
				log.Printf("[a2ui] exit (no output)")
				break
			}
			continue
		}

		mo := event.Output.MessageOutput
		role := mo.Role
		if role == "" && mo.Message != nil {
			role = mo.Message.Role
		}
		log.Printf("[a2ui] message output: role=%q isStreaming=%v hasStream=%v hasMessage=%v",
			role, mo.IsStreaming, mo.MessageStream != nil, mo.Message != nil)

		switch role {
		case schema.Tool:
			// Drain the stream if needed, then show a compact tool-result chip.
			content := drainToolResult(mo)
			log.Printf("[a2ui] tool result (%d chars): %.200s", len(content), content)
			_ = emitToolChip(w, surfaceID, rootChildren, msgIdx, "tool result", content)

		default:
			// Assistant (or unknown role) — may carry text content and/or tool calls.
			if mo.IsStreaming && mo.MessageStream != nil {
				// Stream text tokens to the UI as they arrive.
				// Tool call chunks are accumulated and emitted as chips after the stream ends.

				// Pre-compute IDs for the text card (committed only when content appears).
				textIdx := *msgIdx
				cardID := fmt.Sprintf("msg-%d-card", textIdx)
				colID := fmt.Sprintf("msg-%d-col", textIdx)
				roleID := fmt.Sprintf("msg-%d-role", textIdx)
				contentID := fmt.Sprintf("msg-%d-content", textIdx)
				dataKey := fmt.Sprintf("%s/msg-%d", surfaceID, textIdx)

				nameByIdx := map[int]string{}
				argsByIdx := map[int]*strings.Builder{}
				var tcOrder []int
				seenTCIdx := map[int]bool{}

				var shellEmitted bool
				var accContent strings.Builder

				for {
					chunk, recvErr := mo.MessageStream.Recv()
					if errors.Is(recvErr, io.EOF) {
						break
					}
					if recvErr != nil {
						log.Printf("[a2ui] stream recv error: %v", recvErr)
						break
					}

					// Accumulate tool call argument fragments (keyed by Index).
					for _, tc := range chunk.ToolCalls {
						idx := 0
						if tc.Index != nil {
							idx = *tc.Index
						}
						if !seenTCIdx[idx] {
							seenTCIdx[idx] = true
							tcOrder = append(tcOrder, idx)
						}
						if tc.Function.Name != "" && nameByIdx[idx] == "" {
							nameByIdx[idx] = tc.Function.Name
						}
						if tc.Function.Arguments != "" {
							if argsByIdx[idx] == nil {
								argsByIdx[idx] = &strings.Builder{}
							}
							argsByIdx[idx].WriteString(tc.Function.Arguments)
						}
					}

					// Emit text tokens to the UI immediately.
					if chunk.Content != "" {
						if !shellEmitted {
							// Commit this message slot and send the card scaffold with a data binding.
							*rootChildren = append(*rootChildren, cardID)
							*msgIdx++
							if shellErr := emitMessageShell(w, surfaceID, *rootChildren, cardID, colID, roleID, contentID, dataKey, roleToLabel(role)); shellErr != nil {
								return lastContent.String(), "", shellErr
							}
							shellEmitted = true
						}
						accContent.WriteString(chunk.Content)
						if dataErr := emitDataUpdate(w, surfaceID, dataKey, accContent.String()); dataErr != nil {
							return lastContent.String(), "", dataErr
						}
					}
				}

				// Build final tool-call list and emit chips.
				var toolCalls []toolCallInfo
				for _, i := range tcOrder {
					name := nameByIdx[i]
					if name == "" {
						continue
					}
					args := ""
					if ab := argsByIdx[i]; ab != nil {
						args = ab.String()
					}
					toolCalls = append(toolCalls, toolCallInfo{Name: name, Args: args})
				}
				log.Printf("[a2ui] assistant stream: content=%d chars toolCalls=%d", accContent.Len(), len(toolCalls))

				for _, tc := range toolCalls {
					log.Printf("[a2ui] tool call: %s args=%s", tc.Name, tc.Args)
					_ = emitToolChip(w, surfaceID, rootChildren, msgIdx, "tool call", formatToolCall(tc))
				}
				if shellEmitted {
					lastContent.Reset()
					lastContent.WriteString(accContent.String())
				}

			} else if mo.Message != nil {
				msg := mo.Message
				log.Printf("[a2ui] assistant message: content=%d chars toolCalls=%d", len(msg.Content), len(msg.ToolCalls))

				for _, tc := range msg.ToolCalls {
					log.Printf("[a2ui] tool call: %s args=%s", tc.Function.Name, tc.Function.Arguments)
					_ = emitToolChip(w, surfaceID, rootChildren, msgIdx, "tool call", formatToolCall(toolCallInfo{
						Name: tc.Function.Name,
						Args: tc.Function.Arguments,
					}))
				}
				if msg.Content != "" {
					if err := emitTextCard(w, surfaceID, rootChildren, msgIdx, roleToLabel(role), msg.Content); err != nil {
						return lastContent.String(), "", err
					}
					lastContent.Reset()
					lastContent.WriteString(msg.Content)
				}
			} else {
				log.Printf("[a2ui] assistant event with no stream and no message (skipped)")
			}
		}

		if hasExit {
			log.Printf("[a2ui] exit (after output)")
			break
		}
	}

	return lastContent.String(), interruptID, nil
}

// toolCallInfo holds the accumulated name and arguments for one tool call.
type toolCallInfo struct {
	Name string
	Args string
}

// consumeStream reads all chunks from a MessageStream, accumulating text content
// and tool call info. Used for tool-result messages that may arrive as streams.
func consumeStream(stream *schema.StreamReader[*schema.Message]) (content string, toolCalls []toolCallInfo) {
	nameByIdx := map[int]string{}
	argsByIdx := map[int]*strings.Builder{}
	var order []int
	seenIdx := map[int]bool{}
	var buf strings.Builder

	for {
		chunk, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			log.Printf("[a2ui] stream recv error: %v", err)
			break
		}
		if chunk.Content != "" {
			buf.WriteString(chunk.Content)
		}
		for _, tc := range chunk.ToolCalls {
			idx := 0
			if tc.Index != nil {
				idx = *tc.Index
			}
			if !seenIdx[idx] {
				seenIdx[idx] = true
				order = append(order, idx)
			}
			if tc.Function.Name != "" && nameByIdx[idx] == "" {
				nameByIdx[idx] = tc.Function.Name
			}
			if tc.Function.Arguments != "" {
				if argsByIdx[idx] == nil {
					argsByIdx[idx] = &strings.Builder{}
				}
				argsByIdx[idx].WriteString(tc.Function.Arguments)
			}
		}
	}

	for _, idx := range order {
		name := nameByIdx[idx]
		if name == "" {
			continue
		}
		args := ""
		if ab := argsByIdx[idx]; ab != nil {
			args = ab.String()
		}
		toolCalls = append(toolCalls, toolCallInfo{Name: name, Args: args})
	}
	return buf.String(), toolCalls
}

// drainToolResult reads content from a tool-result MessageVariant.
func drainToolResult(mo *adk.MessageVariant) string {
	if mo.IsStreaming && mo.MessageStream != nil {
		content, _ := consumeStream(mo.MessageStream)
		return content
	}
	if mo.Message != nil {
		return mo.Message.Content
	}
	return ""
}

// formatToolCall formats a toolCallInfo for display in a chip.
// Output: "🔧 functionName\n<args>" (args truncated to 400 runes).
func formatToolCall(tc toolCallInfo) string {
	text := "🔧 " + tc.Name
	if tc.Args != "" {
		args := tc.Args
		if len([]rune(args)) > 400 {
			args = string([]rune(args)[:400]) + "…"
		}
		text += "\n" + args
	}
	return text
}

// emitTextCard emits a text card with full content (non-streaming path).
func emitTextCard(w io.Writer, surfaceID string, rootChildren *[]string, msgIdx *int, roleLabel, content string) error {
	idx := *msgIdx
	cardID := fmt.Sprintf("msg-%d-card", idx)
	colID := fmt.Sprintf("msg-%d-col", idx)
	roleID := fmt.Sprintf("msg-%d-role", idx)
	contentID := fmt.Sprintf("msg-%d-content", idx)
	dataKey := fmt.Sprintf("%s/msg-%d", surfaceID, idx)

	*rootChildren = append(*rootChildren, cardID)
	*msgIdx++

	if err := emitMessageShell(w, surfaceID, *rootChildren, cardID, colID, roleID, contentID, dataKey, roleLabel); err != nil {
		return err
	}
	return emitDataUpdate(w, surfaceID, dataKey, content)
}

// emitToolChip emits a compact single-line chip for tool calls or tool results.
func emitToolChip(w io.Writer, surfaceID string, rootChildren *[]string, msgIdx *int, kind, text string) error {
	idx := *msgIdx
	cardID := fmt.Sprintf("msg-%d-card", idx)
	colID := fmt.Sprintf("msg-%d-col", idx)
	labelID := fmt.Sprintf("msg-%d-label", idx)
	textID := fmt.Sprintf("msg-%d-text", idx)

	*rootChildren = append(*rootChildren, cardID)
	*msgIdx++

	// Truncate long tool output to keep the UI tidy.
	// Approval cards are never truncated — users need the full context to decide.
	display := text
	if kind != "approval needed" && len([]rune(display)) > 300 {
		display = string([]rune(display)[:300]) + "…"
	}

	return emit(w, Message{
		SurfaceUpdate: &SurfaceUpdateMsg{
			SurfaceID: surfaceID,
			Components: []Component{
				{ID: "root-col", Component: ComponentValue{Column: &ColumnComp{Children: append([]string{}, *rootChildren...)}}},
				{ID: cardID, Component: ComponentValue{Card: &CardComp{Children: []string{colID}}}},
				{ID: colID, Component: ComponentValue{Column: &ColumnComp{Children: []string{labelID, textID}}}},
				{ID: labelID, Component: ComponentValue{Text: &TextComp{Value: kind, UsageHint: "caption"}}},
				{ID: textID, Component: ComponentValue{Text: &TextComp{Value: display, UsageHint: "body"}}},
			},
		},
	})
}

// ── history / shell / data helpers ───────────────────────────────────────────

func emitHistory(w io.Writer, surfaceID string, history []*schema.Message, rootChildren []string) error {
	comps := []Component{
		{ID: "root-col", Component: ComponentValue{Column: &ColumnComp{Children: append([]string{}, rootChildren...)}}},
	}
	for i, msg := range history {
		cardID := fmt.Sprintf("msg-%d-card", i)
		colID := fmt.Sprintf("msg-%d-col", i)
		roleID := fmt.Sprintf("msg-%d-role", i)
		contentID := fmt.Sprintf("msg-%d-content", i)
		comps = append(comps,
			Component{ID: cardID, Component: ComponentValue{Card: &CardComp{Children: []string{colID}}}},
			Component{ID: colID, Component: ComponentValue{Column: &ColumnComp{Children: []string{roleID, contentID}}}},
			Component{ID: roleID, Component: ComponentValue{Text: &TextComp{Value: roleToLabel(msg.Role), UsageHint: "caption"}}},
			Component{ID: contentID, Component: ComponentValue{Text: &TextComp{Value: msg.Content, UsageHint: "body"}}},
		)
	}
	return emit(w, Message{SurfaceUpdate: &SurfaceUpdateMsg{SurfaceID: surfaceID, Components: comps}})
}

func emitMessageShell(w io.Writer, surfaceID string, rootChildren []string, cardID, colID, roleID, contentID, dataKey, roleLabel string) error {
	return emit(w, Message{
		SurfaceUpdate: &SurfaceUpdateMsg{
			SurfaceID: surfaceID,
			Components: []Component{
				{ID: "root-col", Component: ComponentValue{Column: &ColumnComp{Children: append([]string{}, rootChildren...)}}},
				{ID: cardID, Component: ComponentValue{Card: &CardComp{Children: []string{colID}}}},
				{ID: colID, Component: ComponentValue{Column: &ColumnComp{Children: []string{roleID, contentID}}}},
				{ID: roleID, Component: ComponentValue{Text: &TextComp{Value: roleLabel, UsageHint: "caption"}}},
				{ID: contentID, Component: ComponentValue{Text: &TextComp{DataKey: dataKey, UsageHint: "body"}}},
			},
		},
	})
}

func emitDataUpdate(w io.Writer, surfaceID, dataKey, content string) error {
	return emit(w, Message{
		DataModelUpdate: &DataModelUpdateMsg{
			SurfaceID: surfaceID,
			Contents:  []DataContent{{Key: dataKey, ValueString: content}},
		},
	})
}

func emit(w io.Writer, msg Message) error {
	data, err := Encode(msg)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func roleToLabel(role schema.RoleType) string {
	switch role {
	case schema.User:
		return "You"
	case schema.Assistant:
		return "Agent"
	case schema.Tool:
		return "Tool"
	case schema.System:
		return "System"
	default:
		if role != "" {
			return string(role)
		}
		return "Agent"
	}
}
