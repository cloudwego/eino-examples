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

import "github.com/cloudwego/eino/schema"

type scenario struct {
	planSummary   string
	searchSummary string
	visualSummary string

	lastResponse           *schema.AgenticMessage
	lastResponseID         string
	statefulBaseResponseID string
}

func newScenario() *scenario {
	return &scenario{}
}

func seedScenarioForSearch(s *scenario) {
	s.planSummary = fallbackPlanSummary
}

func seedScenarioForVisual(s *scenario) {
	seedScenarioForSearch(s)
	s.searchSummary = fallbackSearchSummary
}

func seedScenarioAfterPlanForStateful(s *scenario) {
	if s.searchSummary == "" {
		s.searchSummary = fallbackSearchSummary
	}
	if s.visualSummary == "" {
		s.visualSummary = fallbackVisualSummary
	}
}

func rememberLastResponse(s *scenario, result *runResult) {
	if s == nil || result == nil {
		return
	}
	if result.finalMessage != nil {
		s.lastResponse = result.finalMessage
	}
	if result.responseID != "" {
		s.lastResponseID = result.responseID
	}
}

func rememberStatefulBase(s *scenario, result *runResult) {
	if s == nil || result == nil || result.responseID == "" {
		return
	}
	s.statefulBaseResponseID = result.responseID
}

func fallbackIfEmpty(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

const fallbackPlanSummary = `初版计划：周六中午抵达杭州，下午选择一个展览或博物馆，傍晚安排咖啡和轻量散步；周日上午自然醒，安排一个低强度景点，午后返程。预算控制在 1800 元内，偏好轻松、不早起、展览、咖啡和少排队。`

const fallbackSearchSummary = `实时信息摘要：需要优先确认展馆开放时间、预约要求、临时展览、周末天气和排队风险；若遇到热门展览，应准备备选咖啡店和室内路线。`

const fallbackVisualSummary = `现场视觉摘要：用户到达景点或展馆后，需要根据平面图识别入口、重点展区、咖啡区、出口和排队区域，并把路线调整为少走路、少排队。`
