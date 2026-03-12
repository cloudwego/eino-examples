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

// smoketest is a standalone CLI that exercises the full pipeline without the browser:
//
//	go run ./smoketest "what can you do?"
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	localbk "github.com/cloudwego/eino-ext/adk/backend/local"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/deep"
	"github.com/cloudwego/eino/schema"

	"github.com/cloudwego/eino-examples/adk/common/model"
	"github.com/cloudwego/eino-examples/quickstart/chatwitheino/a2ui"
)

func main() {
	query := "what can you do?"
	if len(os.Args) > 1 {
		query = os.Args[1]
	}

	ctx := context.Background()

	agent, err := func() (adk.Agent, error) {
		backend, err := localbk.NewBackend(ctx, &localbk.Config{})
		if err != nil {
			return nil, err
		}
		return deep.New(ctx, &deep.Config{
			Name:           "ChatWithDocAgent",
			Description:    "An agent that reads and answers questions about documents.",
			ChatModel:      model.NewChatModel(),
			Backend:        backend,
			StreamingShell: backend,
			MaxIteration:   10,
		})
	}()
	if err != nil {
		fmt.Fprintf(os.Stderr, "buildAgent: %v\n", err)
		os.Exit(1)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
	})

	messages := []*schema.Message{schema.UserMessage(query)}

	fmt.Printf("→ user: %s\n\n", query)

	iter := runner.Run(ctx, messages)

	lastContent, interruptID, _, err := a2ui.StreamToWriter(&jsonlPrinter{}, "smoketest", messages, iter)
	fmt.Println()
	if err != nil {
		fmt.Fprintf(os.Stderr, "stream error: %v\n", err)
		os.Exit(1)
	}
	if interruptID != "" {
		fmt.Printf("\n⏸ agent interrupted (id=%s); re-run with approval to continue\n", interruptID)
		return
	}
	fmt.Printf("\n← final content (%d chars):\n%s\n", len(lastContent), lastContent)
}

// jsonlPrinter writes each A2UI JSONL line to stdout with a human-readable summary.
type jsonlPrinter struct {
	buf []byte
}

func (p *jsonlPrinter) Write(b []byte) (int, error) {
	p.buf = append(p.buf, b...)
	for {
		idx := -1
		for i, c := range p.buf {
			if c == '\n' {
				idx = i
				break
			}
		}
		if idx < 0 {
			break
		}
		line := p.buf[:idx]
		p.buf = p.buf[idx+1:]
		if len(line) == 0 {
			continue
		}
		p.printLine(line)
	}
	return len(b), nil
}

func (p *jsonlPrinter) printLine(line []byte) {
	var msg a2ui.Message
	if err := json.Unmarshal(line, &msg); err != nil {
		fmt.Printf("[raw] %s\n", line)
		return
	}
	switch {
	case msg.BeginRendering != nil:
		fmt.Printf("[beginRendering] surface=%s root=%s\n",
			msg.BeginRendering.SurfaceID, msg.BeginRendering.Root)
	case msg.SurfaceUpdate != nil:
		for _, c := range msg.SurfaceUpdate.Components {
			switch {
			case c.Component.Text != nil && c.Component.Text.Value != "":
				fmt.Printf("[surfaceUpdate] %s: Text=%q\n", c.ID, truncate(c.Component.Text.Value, 60))
			case c.Component.Column != nil:
				fmt.Printf("[surfaceUpdate] %s: Column children=%v\n", c.ID, c.Component.Column.Children)
			case c.Component.Card != nil:
				fmt.Printf("[surfaceUpdate] %s: Card\n", c.ID)
			}
		}
	case msg.DataModelUpdate != nil:
		for _, dc := range msg.DataModelUpdate.Contents {
			fmt.Printf("[dataModelUpdate] %s = %q\n", dc.Key, truncate(dc.ValueString, 80))
		}
	}
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

// Ensure io.EOF is imported (used by a2ui internally).
var _ = errors.Is(io.EOF, io.EOF)
