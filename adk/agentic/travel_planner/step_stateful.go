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

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

func runStatefulStep(ctx context.Context, cfg *appConfig, s *scenario) error {
	printStepHeader("Step 4: Stateful Response Continuation", "集中展示 Step 1 产生的 session cache anchor，以及 previous_response_id 如何续接上下文。")

	opts := append([]model.Option{sessionCacheOption()}, deepThinkingOptions()...)
	if s.statefulBaseResponseID != "" {
		fmt.Println("[stateful] session cache anchor was produced by Step 1's local-tool planning turn.")
		fmt.Printf("[stateful] previous_response_id=%s\n", s.statefulBaseResponseID)
		fmt.Println("[stateful] web_search/image_process outputs are passed as compact new summaries because Ark built-in tools do not support caching.")
		opts = append([]model.Option{previousResponseCacheOption(s.statefulBaseResponseID)}, deepThinkingOptions()...)
	} else {
		fmt.Println("[stateful] no previous response id found; running with compact fallback summaries.")
	}

	mdl, err := newAgenticArkModel(ctx, cfg, cfg.model, opts...)
	if err != nil {
		return err
	}

	const instruction = `你是一个会维护长上下文的旅行 agent。

如果请求通过 previous_response_id 续接了上一步，请直接利用已有上下文，不要要求用户重发计划和搜索结果。
用户现在会补充新的现场约束，你需要给出一版调整后的最终行动方案。
最终答案用中文输出，并明确说明：哪些部分继承自之前的上下文、哪些部分是根据新约束调整的。`

	runner, err := newTravelRunner(ctx, "travel_stateful_agent", "Continue a cached Ark response session.", instruction, mdl, nil)
	if err != nil {
		return err
	}

	userText := statefulUserText(s)
	result, err := runAgenticAgent(ctx, runner, []*schema.AgenticMessage{
		schema.UserAgenticMessage(userText),
	})
	if err != nil {
		return err
	}

	rememberLastResponse(s, result)
	return nil
}

func statefulUserText(s *scenario) string {
	changeRequest := `用户现场反馈：
- 下午开始下雨，不想长时间户外步行。
- 预算从 1800 元收紧到 1500 元。
- 更想把“咖啡休息”放在每天下午，而不是晚上。
- 如果热门展览需要排长队，请优先选择附近的室内备选。

请在不重做全部规划的前提下，给出最终版两天路线、预算调整和临场备选。`

	if s.statefulBaseResponseID != "" {
		return fmt.Sprintf("实时搜索摘要：\n%s\n\n现场视觉摘要：\n%s\n\n%s", s.searchSummary, s.visualSummary, changeRequest)
	}

	return fmt.Sprintf("初版计划：\n%s\n\n实时搜索摘要：\n%s\n\n现场视觉摘要：\n%s\n\n%s", s.planSummary, s.searchSummary, s.visualSummary, changeRequest)
}
