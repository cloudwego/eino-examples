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
	"fmt"
	"os"

	localbk "github.com/cloudwego/eino-ext/adk/backend/local"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/deep"
	"github.com/cloudwego/eino/schema"

	"github.com/cloudwego/eino-examples/quickstart/chatwitheino/a2ui"
	"github.com/cloudwego/eino-examples/quickstart/chatwitheino/chatmodel"
	"github.com/cloudwego/eino-examples/quickstart/chatwitheino/helpers"
	"github.com/cloudwego/eino-examples/quickstart/chatwitheino/msgops"
)

func main() {
	query := "what can you do?"
	if len(os.Args) > 1 {
		query = os.Args[1]
	}

	ctx := context.Background()

	switch msgops.KindFromEnv() {
	case msgops.KindAgentic:
		runTyped[*schema.AgenticMessage](ctx, query)
	default:
		runTyped[*schema.Message](ctx, query)
	}
}

func runTyped[M adk.MessageType](ctx context.Context, query string) {
	cm, err := chatmodel.NewModel[M](ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "build model: %v\n", err)
		os.Exit(1)
	}

	backend, err := localbk.NewBackend(ctx, &localbk.Config{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "build backend: %v\n", err)
		os.Exit(1)
	}

	agent, err := deep.NewTyped[M](ctx, &deep.TypedConfig[M]{
		Name:           "ChatWithDocAgent",
		Description:    "An agent that reads and answers questions about documents.",
		ChatModel:      cm,
		Backend:        backend,
		StreamingShell: backend,
		MaxIteration:   10,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "buildAgent: %v\n", err)
		os.Exit(1)
	}

	runner := adk.NewTypedRunner[M](adk.TypedRunnerConfig[M]{
		Agent:           agent,
		EnableStreaming: true,
	})

	messages := []M{msgops.NewUser[M](query)}

	fmt.Printf("→ user: %s\n\n", query)

	iter := runner.Run(ctx, messages)

	lastContent, _, interruptID, _, err := a2ui.StreamToWriter(&jsonlPrinter{}, "smoketest", messages, iter)
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
				fmt.Printf("[surfaceUpdate] %s: Text=%q\n", c.ID, helpers.Truncate(c.Component.Text.Value, 60))
			case c.Component.Column != nil:
				fmt.Printf("[surfaceUpdate] %s: Column children=%v\n", c.ID, c.Component.Column.Children)
			case c.Component.Card != nil:
				fmt.Printf("[surfaceUpdate] %s: Card\n", c.ID)
			}
		}
	case msg.DataModelUpdate != nil:
		for _, dc := range msg.DataModelUpdate.Contents {
			fmt.Printf("[dataModelUpdate] %s = %q\n", dc.Key, helpers.Truncate(dc.ValueString, 80))
		}
	}
}
