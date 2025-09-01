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

package tools

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

type AskForClarificationInput struct {
	Question string `json:"question" jsonschema:"description=The specific question you want to ask the user to get the missing information"`
}

func NewAskForClarificationTool() tool.InvokableTool {
	t, err := utils.InferOptionableTool(
		"ask_for_clarification",
		"Call this tool when the user's request is ambiguous or lacks the necessary information to proceed. Use it to ask a follow-up question to get the details you need, such as the book's genre, before you can use other tools effectively.",
		func(ctx context.Context, input *AskForClarificationInput, opts ...tool.Option) (output string, err error) {
			fmt.Printf("\nQuestion: %s\n", input.Question)
			scanner := bufio.NewScanner(os.Stdin)
			fmt.Print("\nyour input here: ")
			scanner.Scan()
			fmt.Println()
			nInput := scanner.Text()
			return nInput, nil
		})
	if err != nil {
		log.Fatal(err)
	}
	return t
}
