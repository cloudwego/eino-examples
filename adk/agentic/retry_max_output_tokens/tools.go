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

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

type travelRouteInput struct {
	Destination string `json:"destination" jsonschema_description:"Travel destination."`
	Days        int    `json:"days" jsonschema_description:"Number of travel days."`
	Preference  string `json:"preference" jsonschema_description:"Traveler preference, such as relaxed, culture, food, couple, family."`
}

func buildTools() ([]tool.BaseTool, error) {
	route, err := utils.InferTool(
		travelRouteToolName,
		"Return one complete travel route. Call at most once per user request.",
		func(_ context.Context, input *travelRouteInput) (string, error) {
			destination := "杭州"
			days := 3
			preference := "情侣、第一次到访、轻松文化美食路线"
			if input != nil {
				if input.Destination != "" {
					destination = input.Destination
				}
				if input.Days > 0 {
					days = input.Days
				}
				if input.Preference != "" {
					preference = input.Preference
				}
			}

			return marshalJSON(map[string]any{
				"status":      "complete",
				"destination": destination,
				"days":        days,
				"preference":  preference,
				"next_action": "write_final_answer_without_calling_tools_again",
				"route": map[string]any{
					"title": "杭州 3 天 2 晚轻松情侣路线",
					"summary": []string{
						"节奏以西湖、茶园、运河和美食为主，避免每天跨城奔波。",
						"住宿建议选湖滨、武林广场或凤起路一带，去西湖和地铁都方便。",
					},
					"daily_plan": []map[string]any{
						{
							"day":   1,
							"theme": "西湖初印象与湖滨夜景",
							"morning": []string{
								"抵达杭州后入住湖滨或凤起路附近酒店。",
								"从一公园或湖滨步行进入西湖，慢慢适应节奏。",
							},
							"afternoon": []string{
								"游览断桥、白堤、平湖秋月，可在孤山附近喝茶休息。",
								"如果体力充足，继续到曲院风荷或岳王庙。",
							},
							"evening": []string{
								"晚餐选湖滨银泰、龙翔桥或武林路周边。",
								"饭后看西湖夜景，轻松散步到音乐喷泉附近。",
							},
						},
						{
							"day":   2,
							"theme": "灵隐、茶园与城市烟火",
							"morning": []string{
								"早起前往灵隐寺和飞来峰，建议预留 3 小时。",
								"避开中午高峰，出发前准备好预约和身份证件。",
							},
							"afternoon": []string{
								"去龙井村或梅家坞喝茶、看茶园，午餐可选农家菜。",
								"下午回市区休息，避免行程过满。",
							},
							"evening": []string{
								"去大马弄、河坊街或中山中路一带吃小吃。",
								"想安静一点可改去小河直街，适合拍照和散步。",
							},
						},
						{
							"day":   3,
							"theme": "运河街区与返程",
							"morning": []string{
								"前往京杭大运河沿线，逛桥西历史文化街区。",
								"可选中国伞博物馆、扇博物馆或刀剪剑博物馆。",
							},
							"afternoon": []string{
								"午餐后去拱宸桥附近喝咖啡或买伴手礼。",
								"根据返程时间回酒店取行李，前往杭州东站或机场。",
							},
							"evening": []string{
								"若晚班返程，可补一个武林夜市或湖滨商圈。",
							},
						},
					},
					"transportation": []string{
						"市区优先地铁加步行，西湖周边少打车，拥堵时步行更稳。",
						"灵隐和龙井方向建议上午早出发，返程可用公交或网约车。",
						"返程去杭州东站通常地铁最稳定，去萧山机场预留 1.5 到 2 小时。",
					},
					"food": []string{
						"杭帮菜可选龙井虾仁、东坡肉、西湖醋鱼、片儿川。",
						"小吃可以安排葱包烩、定胜糕、酥油饼、藕粉。",
						"热门餐厅建议提前排号，周末湖滨和灵隐附近人会很多。",
					},
					"budget": []string{
						"经济型：每人 1200 到 1800 元，不含大交通。",
						"舒适型：每人 2000 到 3200 元，不含大交通。",
						"预算主要受酒店位置、餐厅选择和是否打车影响。",
					},
					"notes": []string{
						"西湖景区范围大，不建议一天把所有景点走完。",
						"灵隐寺、飞来峰等景点出行前确认预约和开放时间。",
						"雨天可把龙井茶园换成浙江省博物馆、德寿宫或运河博物馆。",
					},
				},
			})
		},
	)
	if err != nil {
		return nil, err
	}

	return []tool.BaseTool{route}, nil
}

func marshalJSON(v any) (string, error) {
	bs, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(bs), nil
}
