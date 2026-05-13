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
	"fmt"

	"github.com/cloudwego/eino/schema"
)

func agentInstruction(planPath string) string {
	return fmt.Sprintf(`你是一个自主运行的旅行资料整理 Agent。

请为喜欢展览、博物馆、咖啡和轻松散步的用户，生成一份杭州这个周末的两日旅行计划。

请像 agent 一样工作，而不是直接一次性回答：
1. 先调用 lookup_user_profile 和 lookup_travel_constraints，读取用户偏好和行程约束。
2. 使用服务端 web_search 核验天气、开放时间、当前展览、咖啡馆或会影响路线的实时信息。你可以按需要多次搜索，但请保持搜索目标清晰，不要堆砌原始搜索结果。
3. 根据核验结果安排轻松、少折返的路线。
4. 在写文件前，必须调用一次 evaluate_booking_option，选择一个可能超过单项预算上限或涉及付款动作的付费展览、讲解、咖啡或活动候选项，让 travelPolicyMiddleware 检查预算和付款策略。如果 middleware 拒绝，请不要把该候选项写成默认推荐，请给出更低成本、无需付款的替代方案。
5. 使用 filesystem 工具把最终旅行计划写入：%s
6. 文件写入后停止工具调用，给用户一个简短最终回复。

最终回复需要包含：
- 旅行计划已生成的简短说明；
- 最终文件路径；
- 预算和付款策略已由 middleware 做检查；
- 本示例使用到的 Agentic 能力：reasoning/thinking、服务端 web_search、本地工具、filesystem middleware、自定义策略 middleware。

请注意：
- 不要声称已经真实预约、付款或下单。
- 如果 evaluate_booking_option 返回 policy_rejected，请把它当成策略拒绝结果处理，并重新选择可行替代项。
- 不要把 <|FunctionCallBegin|> 这类伪造工具调用标记写进旅行计划；工具调用必须通过 ADK agent loop 发生。
- 行程保持轻松、可步行，旅行计划中应有清晰的“第一天 / 第二天”或 “Day 1 / Day 2”结构。`, planPath)
}

func userRequest(planPath string) *schema.AgenticMessage {
	text := fmt.Sprintf(`请为我生成一份杭州这个周末的两日旅行计划。

偏好：
- 展览和博物馆；
- 好喝的咖啡；
- 轻松散步，不要赶场；
- 需要用 web search 核验实时信息；
- 可以评估付费体验，但不要真实购买、付款或完成预约；超出预算策略的候选项应替换为更轻量的方案。

请把旅行计划保存到：
%s`, planPath)

	return &schema.AgenticMessage{
		Role: schema.AgenticRoleTypeUser,
		ContentBlocks: []*schema.ContentBlock{
			schema.NewContentBlock(&schema.UserInputText{Text: text}),
		},
	}
}
