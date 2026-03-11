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
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/cloudwego/eino/schema"

	examplemodel "github.com/cloudwego/eino-examples/adk/common/model"
)

func main() {
	var instruction string
	flag.StringVar(&instruction, "instruction", "You are a helpful assistant.", "")
	flag.Parse()

	query := strings.TrimSpace(strings.Join(flag.Args(), " "))
	if query == "" {
		_, _ = fmt.Fprintln(os.Stderr, "usage: go run ./cmd/ch01 -- \"your question\"")
		os.Exit(2)
	}

	ctx := context.Background()
	cm := examplemodel.NewChatModel()

	msgs := make([]*schema.Message, 0, 2)
	if strings.TrimSpace(instruction) != "" {
		msgs = append(msgs, schema.SystemMessage(instruction))
	}
	msgs = append(msgs, schema.UserMessage(query))

	_, _ = fmt.Fprint(os.Stdout, "[assistant] ")
	stream, err := cm.Stream(ctx, msgs)
	if err == nil {
		stream.SetAutomaticClose()
		if err := printChatModelStream(stream); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		_, _ = fmt.Fprintln(os.Stdout)
		return
	}

	out, err := cm.Generate(ctx, msgs)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if out != nil {
		_, _ = fmt.Fprintln(os.Stdout, out.Content)
	} else {
		_, _ = fmt.Fprintln(os.Stdout)
	}
}

func printChatModelStream(stream *schema.StreamReader[*schema.Message]) error {
	for {
		frame, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		if frame != nil {
			_, _ = fmt.Fprint(os.Stdout, frame.Content)
		}
	}
}
