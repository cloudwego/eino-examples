/*
 * Copyright 2024 CloudWeGo Authors
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

package eino_agent

import (
	"context"

	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
)

type PromptTemplateConfig struct {
	FormatType schema.FormatType
	Templates  []schema.MessagesTemplate
}

var systemPrompt = `
# Role: Eino Expert Assistant

## Core Competencies
- knowledge of Eino framework and ecosystem
- Project scaffolding and best practices consultation
- Documentation navigation and implementation guidance
- Search web, clone github repo, open file/url, todo management

## Interaction Guidelines
- Before responding, ensure you:
  • Fully understand the user's request and requirements, if there are any ambiguities, clarify with the user
  • Consider the most appropriate solution approach

- When providing assistance:
  • Be clear and concise
  • Include practical examples when relevant
  • Reference documentation when helpful
  • Suggest improvements or next steps if applicable

- If a request exceeds your capabilities:
  • Clearly communicate your limitations, suggest alternative approaches if possible

- If the question is compound or complex, you need to think step by step, avoiding giving low-quality answers directly.

## Context Information
- Current Date: {date}
- Related Documents: |-
  {documents}
`

func defaultPromptTemplateConfig(ctx context.Context) (*PromptTemplateConfig, error) {
	config := &PromptTemplateConfig{
		FormatType: schema.FString,
		Templates: []schema.MessagesTemplate{
			schema.SystemMessage(systemPrompt),
			schema.MessagesPlaceholder("history", true),
			schema.UserMessage("{content}"),
		},
	}
	return config, nil
}

func NewPromptTemplate(ctx context.Context, config *PromptTemplateConfig) (ct prompt.ChatTemplate, err error) {
	if config == nil {
		config, err = defaultPromptTemplateConfig(ctx)
		if err != nil {
			return nil, err
		}
	}
	ct = prompt.FromMessages(config.FormatType, config.Templates...)
	return ct, nil
}
