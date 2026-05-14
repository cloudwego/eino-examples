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
	"encoding/json"
	"regexp"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

var sourceURLPattern = regexp.MustCompile(`https?://[^\s)>"]+`)

type evidenceGateMiddleware struct {
	*adk.TypedBaseChatModelAgentMiddleware[*schema.AgenticMessage]
}

func newEvidenceGateMiddleware() adk.TypedChatModelAgentMiddleware[*schema.AgenticMessage] {
	return &evidenceGateMiddleware{
		TypedBaseChatModelAgentMiddleware: &adk.TypedBaseChatModelAgentMiddleware[*schema.AgenticMessage]{},
	}
}

func (m *evidenceGateMiddleware) WrapInvokableToolCall(
	_ context.Context,
	endpoint adk.InvokableToolCallEndpoint,
	tCtx *adk.ToolContext,
) (adk.InvokableToolCallEndpoint, error) {
	if tCtx == nil || tCtx.Name != "save_research_report" {
		return endpoint, nil
	}

	return func(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
		var input saveResearchReportInput
		if err := json.Unmarshal([]byte(argumentsInJSON), &input); err != nil {
			return "", err
		}

		missing := missingReportRequirements(input.Markdown)
		if len(missing) > 0 {
			return mustJSON(map[string]any{
				"status":      "blocked_by_evidence_gate",
				"missing":     missing,
				"instruction": "Revise the report, add the missing evidence structure, then call save_research_report again.",
			})
		}

		return endpoint(ctx, argumentsInJSON, opts...)
	}, nil
}

func missingReportRequirements(markdown string) []string {
	var missing []string
	for _, section := range []string{
		"## Summary",
		"## Evidence",
		"## Risks",
		"## Recommendation",
		"## Next Steps",
		"## Sources",
	} {
		if !strings.Contains(markdown, section) {
			missing = append(missing, section)
		}
	}

	if len(sourceURLPattern.FindAllString(markdown, -1)) < 2 {
		missing = append(missing, "at least two source URLs")
	}

	return missing
}
