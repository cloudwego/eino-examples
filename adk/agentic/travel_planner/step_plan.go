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

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

func runPlanStep(ctx context.Context, cfg *appConfig, s *scenario) error {
	printStepHeader("Step 1: ADK + AgenticMessage + Local Tools", "开启 Ark 服务端 deep thinking，让模型先思考、再调用本地工具，并产出可追踪的 AgenticMessage。")

	opts := append([]model.Option{sessionCacheOption()}, deepThinkingOptions()...)
	mdl, err := newAgenticArkModel(ctx, cfg, cfg.model, opts...)
	if err != nil {
		return err
	}

	tools, err := travelTools()
	if err != nil {
		return err
	}

	const instruction = `你是一个谨慎的周末出行规划 agent。

请严格按这个流程工作：
1. 先调用 lookup_user_profile 获取用户偏好。
2. 再调用 lookup_travel_policy 获取预算和约束。
3. 形成一个两天杭州周末方案后，调用 estimate_trip_cost 估算总成本。
4. 调用 score_itinerary 给方案打分。
5. 最终用中文输出精简但完整的方案。

最终答案需要包含：两天路线、为什么适合用户、预算估算、风险和备选方案。
不要编造实时开放时间或天气，这些会在下一步由 Ark Web Search 确认。`

	runner, err := newTravelRunner(ctx, "travel_plan_agent", "Plan a weekend trip with local ADK tools.", instruction, mdl, tools)
	if err != nil {
		return err
	}

	result, err := runAgenticAgent(ctx, runner, []*schema.AgenticMessage{
		schema.UserAgenticMessage("我这个周末想去杭州放松两天，喜欢展览、博物馆、咖啡和轻松散步。请先根据本地工具里的偏好和预算做一个初版计划。"),
	})
	if err != nil {
		return err
	}

	s.planSummary = fallbackIfEmpty(result.text, fallbackPlanSummary)
	rememberLastResponse(s, result)
	rememberStatefulBase(s, result)
	return nil
}
