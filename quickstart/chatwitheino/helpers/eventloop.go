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

package helpers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/cloudwego/eino/adk"

	"github.com/cloudwego/eino-examples/quickstart/chatwitheino/msgops"
)

// PrintOptions controls how PrintAndCollect renders agent events.
type PrintOptions struct {
	ShowToolCalls    bool
	ShowToolResults  bool
	CaptureInterrupt bool
	ToolResultMaxLen int
	Out              io.Writer
	Err              io.Writer
}

// PrintResult contains the assistant text collected from an event stream.
type PrintResult struct {
	AssistantText string
	InterruptInfo *adk.InterruptInfo
}

// PrintAndCollect prints an agent event stream and returns the assistant text.
func PrintAndCollect[M adk.MessageType](
	events *adk.AsyncIterator[*adk.TypedAgentEvent[M]],
	opts PrintOptions,
) (PrintResult, error) {
	out := opts.Out
	if out == nil {
		out = os.Stdout
	}
	errOut := opts.Err
	if errOut == nil {
		errOut = os.Stderr
	}
	toolResultMaxLen := opts.ToolResultMaxLen
	if toolResultMaxLen == 0 {
		toolResultMaxLen = 200
	}

	var sb strings.Builder
	var interruptInfo *adk.InterruptInfo

	for {
		event, ok := events.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			if LogModelRetry(errOut, event.Err) {
				continue
			}
			return PrintResult{}, event.Err
		}

		if opts.CaptureInterrupt && event.Action != nil && event.Action.Interrupted != nil {
			interruptInfo = event.Action.Interrupted
			continue
		}

		if event.Output == nil || event.Output.MessageOutput == nil {
			continue
		}

		mv := event.Output.MessageOutput
		if msgops.VariantIsToolResult(mv) {
			content, _, _ := msgops.DrainToolResult(mv)
			if opts.ShowToolResults {
				fmt.Fprintf(out, "[tool result] %s\n", Truncate(content, toolResultMaxLen))
			}
			continue
		}

		if mv.IsStreaming {
			mv.MessageStream.SetAutomaticClose()
			var accumulatedToolCalls []msgops.ToolCall
			streamPrefix := sb.String()
			streamWillRetry := false
			for {
				frame, err := mv.MessageStream.Recv()
				if errors.Is(err, io.EOF) {
					break
				}
				if err != nil {
					if LogModelRetry(errOut, err) {
						sb.Reset()
						sb.WriteString(streamPrefix)
						accumulatedToolCalls = nil
						streamWillRetry = true
						break
					}
					return PrintResult{}, err
				}
				if msgops.IsNil(frame) {
					continue
				}
				if text := msgops.AssistantDeltaText(frame); text != "" {
					sb.WriteString(text)
					fmt.Fprint(out, text)
				}
				if opts.ShowToolCalls {
					if calls := msgops.ToolCalls(frame); len(calls) > 0 {
						accumulatedToolCalls = append(accumulatedToolCalls, calls...)
					}
				}
			}
			if streamWillRetry {
				continue
			}
			if opts.ShowToolCalls {
				for _, tc := range accumulatedToolCalls {
					if tc.Name != "" && tc.Args != "" {
						fmt.Fprintf(out, "\n[tool call] %s(%s)\n", tc.Name, tc.Args)
					}
				}
			}
			fmt.Fprintln(out)
			continue
		}

		if msgops.IsNil(mv.Message) {
			continue
		}
		content := msgops.AssistantText(mv.Message)
		sb.WriteString(content)
		fmt.Fprintln(out, content)
		if opts.ShowToolCalls {
			for _, tc := range msgops.ToolCalls(mv.Message) {
				fmt.Fprintf(out, "[tool call] %s(%s)\n", tc.Name, tc.Args)
			}
		}
	}

	return PrintResult{
		AssistantText: sb.String(),
		InterruptInfo: interruptInfo,
	}, nil
}

// Truncate shortens long strings for console output. JSON strings are compacted
// first so tool results remain readable in narrow terminals.
func Truncate(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if compacted := compactJSON(s); compacted != "" {
		s = compacted
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

func compactJSON(s string) string {
	var result bytes.Buffer
	if err := json.Compact(&result, []byte(s)); err != nil {
		return ""
	}
	return result.String()
}
