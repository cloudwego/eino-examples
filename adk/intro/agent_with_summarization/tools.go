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
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

type RepeatSectionsInput struct {
	Title       string   `json:"title" jsonschema_description:"Section title"`
	Paragraphs  []string `json:"paragraphs" jsonschema_description:"Paragraphs to be repeated"`
	RepeatCount int      `json:"repeat_count" jsonschema_description:"Times to repeat paragraphs, default 2"`
}

func NewRepeatSectionsTool() tool.InvokableTool {
	t, err := utils.InferTool(
		"repeat_sections",
		"Repeat given paragraphs to quickly accumulate context",
		func(ctx context.Context, in *RepeatSectionsInput) (string, error) {
			if in.RepeatCount <= 0 {
				in.RepeatCount = 2
			}
			var b strings.Builder
			b.WriteString("## ")
			b.WriteString(in.Title)
			b.WriteString("\n")
			idx := 0
			for i := 0; i < in.RepeatCount; i++ {
				for _, sec := range in.Paragraphs {
					b.WriteString(strconv.Itoa(idx + 1))
					b.WriteString(". ")
					b.WriteString(sec)
					b.WriteString("\n")
					idx++
				}
			}
			callCount := 0
			times, ok := adk.GetSessionValue(ctx, "_tool_call_count")
			if !ok {
				times = 0
			} else {
				callCount = times.(int)
			}
			callCount++
			adk.AddSessionValue(ctx, "_tool_call_count", callCount)

			b.WriteString(fmt.Sprintf("Tool calls so far: %d", callCount))
			return b.String(), nil
		},
	)
	if err != nil {
		log.Fatalf("failed to create repeat_sections tool: %v", err)
	}
	return t
}
