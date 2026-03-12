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
	"errors"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

type ReviewAgent struct {
	AgentName string
	AgentDesc string
}

func (r ReviewAgent) Name(ctx context.Context) string {
	return r.AgentName
}

func (r ReviewAgent) Description(ctx context.Context) string {
	return r.AgentDesc
}

type FeedbackInfo struct {
	OriginalContent string
	Feedback        *string
	NoNeedToEdit    bool
}

func (fi *FeedbackInfo) String() string {
	return fmt.Sprintf("Original content to review: \n`\n%s\n`. \n\nIf you think the content is good as it is, please reply with \"No need to edit\". \nOtherwise, please provide your feedback.", fi.OriginalContent)
}

func (r ReviewAgent) Run(ctx context.Context, input *adk.AgentInput,
	options ...adk.AgentRunOption) *adk.AsyncIterator[*adk.AgentEvent] {
	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()

	go func() {
		defer gen.Close()

		contentToReview, ok := adk.GetSessionValue(ctx, "content_to_review")
		if !ok {
			event := &adk.AgentEvent{
				Err: errors.New("content_to_review not found in session"),
			}
			gen.Send(event)
			return
		}

		feInfo := &FeedbackInfo{
			OriginalContent: contentToReview.(string),
		}

		event := adk.StatefulInterrupt(ctx, feInfo, contentToReview.(string))
		gen.Send(event)
	}()

	return iter
}

func (r ReviewAgent) Resume(ctx context.Context, info *adk.ResumeInfo,
	opts ...adk.AgentRunOption) *adk.AsyncIterator[*adk.AgentEvent] {
	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()

	go func() {
		defer gen.Close()
		if !info.IsResumeTarget { // not explicitly resumed, interrupt with the same review content again
			event := adk.Interrupt(ctx, &FeedbackInfo{
				OriginalContent: info.InterruptState.(string),
			})
			gen.Send(event)
			return
		}

		if info.ResumeData == nil {
			event := &adk.AgentEvent{
				Err: errors.New("review agent receives nil resume data"),
			}
			gen.Send(event)
			return
		}

		feInfo, ok := info.ResumeData.(*FeedbackInfo)
		if !ok {
			event := &adk.AgentEvent{
				Err: errors.New("review agent receives invalid resume data"),
			}
			gen.Send(event)
			return
		}

		if feInfo.NoNeedToEdit {
			event := &adk.AgentEvent{
				Action: adk.NewExitAction(),
			}
			gen.Send(event)
			return
		}

		if feInfo.Feedback == nil {
			event := &adk.AgentEvent{
				Err: errors.New("feedback is nil"),
			}
			gen.Send(event)
			return
		}

		event := &adk.AgentEvent{
			Output: &adk.AgentOutput{
				MessageOutput: &adk.MessageVariant{
					IsStreaming: false,
					Message: &schema.Message{
						Role:    schema.Assistant,
						Content: *feInfo.Feedback,
					},
				},
			},
		}
		gen.Send(event)
	}()

	return iter
}
