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

package tool

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"
)

// FollowUpInfo is the information presented to the user during an interrupt.
type FollowUpInfo struct {
	Questions  []string
	UserAnswer string // This field will be populated by the user.
}

func (fi *FollowUpInfo) String() string {
	var sb strings.Builder
	sb.WriteString("We need more information. Please answer the following questions:\n")
	for i, q := range fi.Questions {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, q))
	}
	return sb.String()
}

// FollowUpState is the state saved during the interrupt.
type FollowUpState struct {
	Questions []string
}

// FollowUpToolInput defines the input schema for our tool.
type FollowUpToolInput struct {
	Questions []string `json:"questions"`
}

func init() {
	schema.Register[*FollowUpInfo]()
	schema.Register[*FollowUpState]()
}

func FollowUp(ctx context.Context, input *FollowUpToolInput) (string, error) {
	wasInterrupted, _, storedState := tool.GetInterruptState[*FollowUpState](ctx)

	if !wasInterrupted {
		info := &FollowUpInfo{Questions: input.Questions}
		state := &FollowUpState{Questions: input.Questions}

		return "", tool.StatefulInterrupt(ctx, info, state)
	}

	isResumeTarget, hasData, resumeData := tool.GetResumeContext[*FollowUpInfo](ctx)

	if !isResumeTarget {
		info := &FollowUpInfo{Questions: storedState.Questions}
		return "", tool.StatefulInterrupt(ctx, info, storedState)
	}

	if !hasData || resumeData.UserAnswer == "" {
		return "", fmt.Errorf("tool resumed without a user answer")
	}

	return resumeData.UserAnswer, nil
}

func GetFollowUpTool() tool.InvokableTool {
	t, err := utils.InferTool("FollowUpTool", "Asks the user for more information by providing a list of questions.", FollowUp)
	if err != nil {
		log.Fatal(err)
	}
	return t
}
