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
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

type researchBriefInput struct {
	Topic string `json:"topic,omitempty" jsonschema_description:"Research topic to refine."`
}

type scoreEvidenceInput struct {
	Claim     string `json:"claim" jsonschema_description:"The concrete claim or signal being scored."`
	SourceURL string `json:"source_url" jsonschema_description:"Source URL for the claim."`
	Notes     string `json:"notes,omitempty" jsonschema_description:"Short notes about why this evidence matters."`
}

func buildTools(reportPath string) ([]tool.BaseTool, error) {
	loadBrief, err := utils.InferTool(
		"load_research_brief",
		"Load the audience, report structure, and evidence scoring criteria for this example.",
		func(ctx context.Context, input *researchBriefInput) (string, error) {
			topic := "AI agent application patterns"
			if input != nil && input.Topic != "" {
				topic = input.Topic
			}

			return mustJSON(map[string]any{
				"topic":    topic,
				"audience": "engineering team evaluating agentic application architecture",
				"required_sections": []string{
					"Summary",
					"Evidence",
					"Risks",
					"Recommendation",
					"Next Steps",
					"Sources",
				},
				"evidence_scoring": map[string]any{
					"confidence": "How strongly the source supports the claim.",
					"freshness":  "Whether the information is recent enough for architecture decisions.",
					"relevance":  "How directly the evidence applies to production agent applications.",
					"risk":       "What could go wrong if the team over-relies on this signal.",
				},
				"save_path": reportPath,
			})
		},
	)
	if err != nil {
		return nil, err
	}

	scoreEvidence, err := utils.InferTool(
		"score_evidence",
		"Score one concrete source-backed evidence item for the research report.",
		func(ctx context.Context, input *scoreEvidenceInput) (string, error) {
			if input == nil {
				return "", fmt.Errorf("score_evidence input is required")
			}

			return mustJSON(map[string]any{
				"claim":      input.Claim,
				"source_url": input.SourceURL,
				"scores": map[string]any{
					"confidence": 0.82,
					"freshness":  0.78,
					"relevance":  0.86,
				},
				"risk":  "Treat the source as one signal, not a complete architecture decision.",
				"notes": input.Notes,
			})
		},
	)
	if err != nil {
		return nil, err
	}

	return []tool.BaseTool{
		loadBrief,
		scoreEvidence,
	}, nil
}

func mustJSON(v any) (string, error) {
	bs, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return string(bs), nil
}
