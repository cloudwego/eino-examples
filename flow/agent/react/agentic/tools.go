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
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/eino-contrib/jsonschema"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

const (
	summarizeNewsToolName   = "summarize_news"
	getUserLocationToolName = "get_user_location"
)

type SummarizeNewsTool struct{}

type NewsSummary struct {
	Date    string `json:"date"`
	Title   string `json:"title"`
	Summary string `json:"summary"`
}

type SummarizeNewsToolInput struct {
	News []*NewsSummary `json:"news"`
}

func (s *SummarizeNewsTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	sc := &jsonschema.Schema{
		Type: "object",
		Properties: orderedmap.New[string, *jsonschema.Schema](
			orderedmap.WithInitialData[string, *jsonschema.Schema](
				orderedmap.Pair[string, *jsonschema.Schema]{
					Key: "news",
					Value: &jsonschema.Schema{
						Type: "array",
						Items: &jsonschema.Schema{
							Type: "object",
							Properties: orderedmap.New[string, *jsonschema.Schema](
								orderedmap.WithInitialData[string, *jsonschema.Schema](
									orderedmap.Pair[string, *jsonschema.Schema]{
										Key: "date",
										Value: &jsonschema.Schema{
											Type:        "string",
											Description: "The date of the news. e.g. 2026-01-13",
											Pattern:     `^\d{4}-\d{2}-\d{2}$`,
										},
									},
									orderedmap.Pair[string, *jsonschema.Schema]{
										Key: "title",
										Value: &jsonschema.Schema{
											Type:        "string",
											Description: "The title of the news.",
										},
									},
									orderedmap.Pair[string, *jsonschema.Schema]{
										Key: "summary",
										Value: &jsonschema.Schema{
											Type:        "string",
											Description: "The summary of the news. Limit to 100 words.",
										},
									},
								),
							),
							Required: []string{"time", "title", "summary"},
						},
					},
				},
			),
		),
		Required: []string{"news"},
	}

	return &schema.ToolInfo{
		Name:        summarizeNewsToolName,
		Desc:        "Summarize the news.",
		ParamsOneOf: schema.NewParamsOneOfByJSONSchema(sc),
	}, nil
}

func (s *SummarizeNewsTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	return argumentsInJSON, nil
}

type GetUserLocationTool struct{}

func (g *GetUserLocationTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: getUserLocationToolName,
		Desc: "Get the user's country",
		ParamsOneOf: schema.NewParamsOneOfByJSONSchema(&jsonschema.Schema{
			Type: "object",
			Properties: orderedmap.New[string, *jsonschema.Schema](
				orderedmap.WithInitialData[string, *jsonschema.Schema](
					orderedmap.Pair[string, *jsonschema.Schema]{
						Key: "language",
						Value: &jsonschema.Schema{
							Type:        "string",
							Description: "The language of the user using",
							Enum:        []any{"en", "zh"},
							Default:     "zh",
						},
					},
				),
			),
			Required: []string{"language"},
		}),
	}, nil
}

type GetUserLocationToolInput struct {
	Language string `json:"language"`
}

func (g *GetUserLocationTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var input GetUserLocationToolInput
	if err := sonic.UnmarshalString(argumentsInJSON, &input); err != nil {
		return "", err
	}
	switch input.Language {
	case "en":
		return "United States", nil
	case "zh":
		return "China", nil
	default:
		return "", fmt.Errorf("unsupported language: %s", input.Language)
	}
}
