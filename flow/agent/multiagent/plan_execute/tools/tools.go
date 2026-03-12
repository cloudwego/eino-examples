/*
 * Copyright 2025 CloudWeGo Authors
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

package tools

import (
	"context"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"
)

func GetTools(ctx context.Context) (tools []tool.BaseTool, err error) {
	queryTimeTool, err := SafeInferTool("query_theme_park_opening_hour", "查询乐园 A 的整体营业时间", GetParkHour)
	if err != nil {
		return nil, err
	}

	tools = append(tools, queryTimeTool)

	queryTicketPriceTool, err := SafeInferTool("query_park_ticket_price", "查询乐园 A 的门票价格", GetParkTicketPrice)
	if err != nil {
		return nil, err
	}

	tools = append(tools, queryTicketPriceTool)

	listLocationsTool, err := SafeInferTool("list_locations", "列出乐园 A 中的所有区域，每个游乐设施都归属于一个区域", ListLocations)
	if err != nil {
		return nil, err
	}

	tools = append(tools, listLocationsTool)

	queryEntranceTool, err := SafeInferTool("query_entrance_location", "查询乐园的哪个区域是入口区域", QueryEntrance)
	if err != nil {
		return nil, err
	}
	tools = append(tools, queryEntranceTool)

	adjacencyTool, err := SafeInferTool("query_location_adjacency_info", "查询乐园 A 中的一个区域到其他相邻区域的步行时间，以分钟为单位", GetAdjacentLocation)
	if err != nil {
		return nil, err
	}

	tools = append(tools, adjacencyTool)

	queueTimeTool, err := SafeInferTool("query_attraction_queue_time", "query the queue time of one or more attractions, in minutes", GetQueueTime)
	if err != nil {
		return nil, err
	}

	tools = append(tools, queueTimeTool)

	queryAttractionTool, err := SafeInferTool("query_attraction_info", "query the detailed information of one or more attractions", GetAttractionInfo)
	if err != nil {
		return nil, err
	}

	tools = append(tools, queryAttractionTool)

	queryPerformanceTool, err := SafeInferTool("query_performance_info", "query the detailed information of one or more performances", GetPerformanceInfo)
	if err != nil {
		return nil, err
	}
	tools = append(tools, queryPerformanceTool)

	queryRestaurantTool, err := SafeInferTool("query_restaurant_info", "query the detailed information of one or more restaurants", GetRestaurantInfo)
	if err != nil {
		return nil, err
	}
	tools = append(tools, queryRestaurantTool)

	validatePerformanceTimeTableTool, err := SafeInferTool("validate_performance_time_table", "validate whether the chosen start time of a performance matches the performance's time table", ValidatePerformanceTimeTable)
	if err != nil {
		return nil, err
	}
	tools = append(tools, validatePerformanceTimeTableTool)

	arrangePerformancesTool, err := SafeInferTool("arrange_performances", "arrange the chosen performances into time slots of the day, according to the performances' time tables and duration", ArrangePerformances)
	if err != nil {
		return nil, err
	}
	tools = append(tools, arrangePerformancesTool)

	ValidatePlanItemsTool, err := SafeInferTool("validate_plan_items", "validate whether the plan items for a full day's plan is valid", ValidatePlanItems)
	if err != nil {
		return nil, err
	}
	tools = append(tools, ValidatePlanItemsTool)

	return tools, nil
}

type safeTool struct {
	tool.InvokableTool
}

func (s safeTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return s.InvokableTool.Info(ctx)
}

func (s safeTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	out, e := s.InvokableTool.InvokableRun(ctx, argumentsInJSON, opts...)
	if e != nil {
		return e.Error(), nil
	}

	return out, nil
}

func SafeInferTool[T, D any](toolName, toolDesc string, i utils.InvokeFunc[T, D]) (tool.InvokableTool, error) {
	t, err := utils.InferTool(toolName, toolDesc, i)
	if err != nil {
		return nil, err
	}

	return safeTool{
		InvokableTool: t,
	}, nil
}
