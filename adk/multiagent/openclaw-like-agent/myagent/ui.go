package myagent

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

type turnRecorder struct {
	assistantParts []string
	messages       []*schema.Message
	toolNames      map[string]string
	printedCalls   map[string]struct{}
	printedResults map[string]struct{}
}

func newTurnRecorder() *turnRecorder {
	return &turnRecorder{
		toolNames:      make(map[string]string),
		printedCalls:   make(map[string]struct{}),
		printedResults: make(map[string]struct{}),
	}
}

func (r *turnRecorder) AddMessage(message *schema.Message) {
	if message == nil {
		return
	}
	cp := *message
	if len(message.ToolCalls) > 0 {
		cp.ToolCalls = append([]schema.ToolCall(nil), message.ToolCalls...)
	}
	r.messages = append(r.messages, &cp)
	if cp.Role == schema.Assistant && cp.Content != "" {
		r.assistantParts = append(r.assistantParts, cp.Content)
	}
	for _, tc := range cp.ToolCalls {
		if tc.ID != "" && tc.Function.Name != "" {
			r.toolNames[tc.ID] = tc.Function.Name
		}
	}
	if cp.Role == schema.Tool && cp.ToolCallID != "" && cp.ToolName != "" {
		r.toolNames[cp.ToolCallID] = cp.ToolName
	}
}

func (r *turnRecorder) Messages() []*schema.Message {
	return cloneMessages(r.messages)
}

func (r *turnRecorder) AssistantReply() string {
	return strings.TrimSpace(strings.Join(r.assistantParts, ""))
}

func printWelcome(out io.Writer, rt *runtime) {
	fmt.Fprintf(out, "MyAgent TUI\n")
	fmt.Fprintf(out, "workspace: %s\n", rt.workspace.root)
	fmt.Fprintf(out, "session:   %s\n", rt.sessionID)
	fmt.Fprintf(out, "commands:  /help /session /history /clear /exit /skills\n")
}

func printRunHeader(out io.Writer, rt *runtime, query string) {
	fmt.Fprintf(out, "\n[%s] user> %s\n", time.Now().Format("15:04:05"), query)
	fmt.Fprintf(out, "[run.started] session=%s workspace=%s\n", rt.sessionID, rt.workspace.root)
}

func renderAgentEvent(out io.Writer, event *adk.AgentEvent, recorder *turnRecorder) error {
	if event == nil {
		return nil
	}

	if event.Output != nil && event.Output.MessageOutput != nil {
		if msg := event.Output.MessageOutput.Message; msg != nil && shouldRenderMessage(msg) {
			renderMessage(out, msg, recorder)
			recorder.AddMessage(msg)
		}
		if stream := event.Output.MessageOutput.MessageStream; stream != nil {
			message, err := collectStreamMessage(out, stream, recorder)
			if err != nil {
				return err
			}
			if message != nil {
				recorder.AddMessage(message)
			}
		}
	}
	if event.Action != nil && event.Action.Exit {
		fmt.Fprintln(out, "\n[status] agent exited current loop")
	}
	return nil
}

func collectStreamMessage(out io.Writer, stream *schema.StreamReader[*schema.Message], recorder *turnRecorder) (*schema.Message, error) {
	var (
		chunks          []*schema.Message
		assistantOpened bool
	)
	for {
		chunk, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("读取消息流失败: %w", err)
		}
		if chunk == nil {
			continue
		}
		cp := *chunk
		if len(chunk.ToolCalls) > 0 {
			cp.ToolCalls = append([]schema.ToolCall(nil), chunk.ToolCalls...)
			cacheToolNames(recorder, chunk.ToolCalls)
		}
		if cp.Content != "" {
			if cp.Role != schema.Tool {
				if !assistantOpened {
					fmt.Fprint(out, "[assistant] ")
					assistantOpened = true
				}
				fmt.Fprint(out, cp.Content)
			}
		}
		chunks = append(chunks, &cp)
	}
	if assistantOpened {
		fmt.Fprintln(out)
	}
	if len(chunks) == 0 {
		return nil, nil
	}
	message, err := schema.ConcatMessages(chunks)
	if err != nil {
		return nil, fmt.Errorf("合并消息流失败: %w", err)
	}
	renderMessage(out, message, recorder)
	return message, nil
}

func renderMessage(out io.Writer, msg *schema.Message, recorder *turnRecorder) {
	if msg == nil {
		return
	}
	cacheToolNames(recorder, msg.ToolCalls)
	printToolCallsWithRecorder(out, recorder, msg.ToolCalls)
	switch msg.Role {
	case schema.Tool:
		key := toolResultKey(msg)
		if recorder != nil {
			if _, ok := recorder.printedResults[key]; ok {
				return
			}
			recorder.printedResults[key] = struct{}{}
		}
		fmt.Fprintf(out, "[tool.call] name=%s output=%s\n", resolveToolName(recorder, msg), trimForDisplay(msg.Content, 240))
	case schema.Assistant:
		if len(msg.ToolCalls) == 0 && msg.Content != "" {
			fmt.Fprintf(out, "[assistant] %s\n", msg.Content)
		}
	}
}

func cacheToolNames(recorder *turnRecorder, calls []schema.ToolCall) {
	if recorder == nil {
		return
	}
	for _, tc := range calls {
		if tc.ID != "" && tc.Function.Name != "" {
			recorder.toolNames[tc.ID] = tc.Function.Name
		}
	}
}

func printToolCalls(out io.Writer, calls []schema.ToolCall) {
	printToolCallsWithRecorder(out, nil, calls)
}

func printToolCallsWithRecorder(out io.Writer, recorder *turnRecorder, calls []schema.ToolCall) {
	for _, tc := range calls {
		if !isCompleteToolCall(tc) {
			continue
		}
		key := toolCallKey(tc)
		if recorder != nil {
			if _, ok := recorder.printedCalls[key]; ok {
				continue
			}
			recorder.printedCalls[key] = struct{}{}
		}
		fmt.Fprintf(out, "[tool.call.start] name=%s args=%s\n", tc.Function.Name, trimForDisplay(tc.Function.Arguments, 160))
	}
}

func resolveToolName(recorder *turnRecorder, msg *schema.Message) string {
	if msg == nil {
		return "unknown"
	}
	if msg.ToolName != "" {
		return msg.ToolName
	}
	if recorder != nil && msg.ToolCallID != "" {
		if name := recorder.toolNames[msg.ToolCallID]; name != "" {
			return name
		}
	}
	return "unknown"
}

func shouldRenderMessage(msg *schema.Message) bool {
	if msg == nil {
		return false
	}
	if msg.Role == schema.Tool {
		return strings.TrimSpace(msg.Content) != ""
	}
	if len(msg.ToolCalls) > 0 {
		for _, tc := range msg.ToolCalls {
			if isCompleteToolCall(tc) {
				return true
			}
		}
	}
	if msg.Role == schema.Assistant && strings.TrimSpace(msg.Content) != "" {
		return true
	}
	return false
}

func isCompleteToolCall(tc schema.ToolCall) bool {
	if strings.TrimSpace(tc.Function.Name) == "" {
		return false
	}
	args := strings.TrimSpace(tc.Function.Arguments)
	if args == "" {
		return false
	}
	return json.Valid([]byte(args))
}

func toolCallKey(tc schema.ToolCall) string {
	if tc.ID != "" {
		return tc.ID
	}
	return tc.Function.Name + "|" + tc.Function.Arguments
}

func toolResultKey(msg *schema.Message) string {
	if msg == nil {
		return ""
	}
	if msg.ToolCallID != "" {
		return msg.ToolCallID + "|" + msg.Content
	}
	return msg.ToolName + "|" + msg.Content
}

func trimForDisplay(s string, max int) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\n", " "))
	if max <= 0 || len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}

func trimForPrompt(s string, max int) string {
	s = strings.TrimSpace(s)
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "\n...(truncated)"
}

func firstNonEmptyLine(s string) string {
	for _, line := range strings.Split(s, "\n") {
		if strings.TrimSpace(line) != "" {
			return strings.TrimSpace(line)
		}
	}
	return ""
}

func nowRFC3339() string {
	return time.Now().Format(time.RFC3339)
}
