/*
 * Copyright 2026 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://www.apache.org/licenses/LICENSE-2.0
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

	"github.com/cloudwego/eino-ext/components/model/agenticark"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/volcengine/volcengine-go-sdk/service/arkruntime/model/responses"
)

func runSearchStep(ctx context.Context, cfg *appConfig, s *scenario) error {
	printStepHeader("Step 2: Ark Built-in Web Search", "把 Step 1 的本地计划交给 Ark 服务端 web_search，确认实时信息，并保留 citation。")

	mdl, err := newAgenticArkModel(ctx, cfg, cfg.model,
		agenticark.WithServerTools(webSearchServerTools()),
		agenticark.WithMaxToolCalls(4),
		model.WithAgenticToolChoice(&schema.AgenticToolChoice{
			Type: schema.ToolChoiceForced,
			Forced: &schema.AgenticForcedToolChoice{
				Tools: []*schema.AllowedTool{
					{
						ServerTool: &schema.AllowedServerTool{
							Name: string(agenticark.ServerToolNameWebSearch),
						},
					},
				},
			},
		}),
	)
	if err != nil {
		return err
	}

	const instruction = `你是一个会核验实时信息的出行 agent。

你必须使用 Ark 内置 web_search 来确认信息，不要只依赖常识。
请重点检索：杭州周末天气、热门展馆开放/预约/闭馆风险、适合雨天或排队时替换的室内咖啡/展览选择。
最终答案用中文输出，保留可核查的信息来源，并明确哪些内容来自实时搜索。`

	runner, err := newTravelRunner(ctx, "travel_search_agent", "Refresh the plan with Ark web search.", instruction, mdl, nil)
	if err != nil {
		return err
	}

	result, err := runAgenticAgent(ctx, runner, []*schema.AgenticMessage{
		schema.UserAgenticMessage(fmt.Sprintf("这是上一步的初版计划：\n\n%s\n\n请使用实时搜索校验它，并给出更新后的建议。", s.planSummary)),
	})
	if err != nil {
		return err
	}

	s.searchSummary = fallbackIfEmpty(result.text, fallbackSearchSummary)
	rememberLastResponse(s, result)
	return nil
}

func webSearchServerTools() []*agenticark.ServerToolConfig {
	return []*agenticark.ServerToolConfig{
		{
			WebSearch: &responses.ToolWebSearch{
				Type:  responses.ToolType_web_search,
				Limit: ptrOf[int64](5),
			},
		},
	}
}
