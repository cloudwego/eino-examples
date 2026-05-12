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
	"github.com/cloudwego/eino/schema"
	"github.com/volcengine/volcengine-go-sdk/service/arkruntime/model/responses"
)

func runVisualStep(ctx context.Context, cfg *appConfig, s *scenario) error {
	printStepHeader("Step 3: Ark Image Process", "把景点或展馆平面图交给 Ark image_process，让模型按需点选、框选、放大或旋转，再生成现场游览路线。")

	mdl, err := newAgenticArkModel(ctx, cfg, cfg.model,
		agenticark.WithServerTools(imageProcessServerTools()),
		agenticark.WithCustomHeaders(map[string]string{
			"ark-beta-image-process": "true",
		}),
	)
	if err != nil {
		return err
	}

	const instruction = `你是一个展馆现场助手。

用户已经有一份经过实时搜索更新的杭州周末计划。现在用户到了景点或展馆，发来一张平面图、导览图或游客中心地图照片。
请根据图片内容做现场路线决策：如果文字小、方向不清或区域难以确认，你可以使用 Ark image_process 的 point、grounding、zoom、rotate 能力辅助识别。
最终用中文输出：你从图里确认了哪些空间信息、建议用户按什么顺序游览、如何把现场路线合并回旅行计划。
注意：不要使用 web_search；这里展示的是纯视觉处理能力。`

	runner, err := newTravelRunner(ctx, "travel_visual_agent", "Read a venue sign with Ark image process.", instruction, mdl, nil)
	if err != nil {
		return err
	}

	userMessage := &schema.AgenticMessage{
		Role: schema.AgenticRoleTypeUser,
		ContentBlocks: []*schema.ContentBlock{
			schema.NewContentBlock(&schema.UserInputImage{
				URL:    cfg.imageURL,
				Detail: schema.ImageURLDetailHigh,
			}),
			schema.NewContentBlock(&schema.UserInputText{
				Text: fmt.Sprintf("这是景点或展馆平面图：%s\n\n上一步实时信息摘要：\n%s\n\n请识别入口、重点展区、咖啡休息点、出口或其他影响路线的空间信息，并给出少走路、少排队的现场游览路线。", cfg.imageURL, s.searchSummary),
			}),
		},
	}

	result, err := runAgenticAgent(ctx, runner, []*schema.AgenticMessage{userMessage})
	if err != nil {
		return err
	}

	s.visualSummary = fallbackIfEmpty(result.text, fallbackVisualSummary)
	return nil
}

func imageProcessServerTools() []*agenticark.ServerToolConfig {
	enabled := "enabled"
	return []*agenticark.ServerToolConfig{
		{
			ImageProcess: &responses.ToolImageProcess{
				Type:      responses.ToolType_image_process,
				Point:     &responses.ImageProcessPointOptions{Type: &enabled},
				Grounding: &responses.ImageProcessGroundingOptions{Type: &enabled},
				Zoom:      &responses.ImageProcessZoomOptions{Type: &enabled},
				Rotate:    &responses.ImageProcessRotateOptions{Type: &enabled},
			},
		},
	}
}
