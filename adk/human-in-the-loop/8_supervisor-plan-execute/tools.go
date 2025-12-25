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

package main

import (
	"context"
	"fmt"
	"hash/fnv"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"

	tool2 "github.com/cloudwego/eino-examples/adk/common/tool"
)

type SearchRequest struct {
	Query string `json:"query" jsonschema_description:"The search query"`
	Topic string `json:"topic" jsonschema_description:"Topic area (technology, business, market)"`
}

type SearchResponse struct {
	Query   string   `json:"query"`
	Results []string `json:"results"`
	Source  string   `json:"source"`
}

type AnalyzeRequirementsRequest struct {
	ProjectDescription string `json:"project_description" jsonschema_description:"Description of the project to analyze"`
}

type AnalyzeRequirementsResponse struct {
	Requirements   []string `json:"requirements"`
	Complexity     string   `json:"complexity"`
	EstimatedHours int      `json:"estimated_hours"`
}

type CreateDesignRequest struct {
	ProjectName  string   `json:"project_name" jsonschema_description:"Name of the project"`
	Requirements []string `json:"requirements" jsonschema_description:"List of requirements"`
}

type CreateDesignResponse struct {
	DesignID     string   `json:"design_id"`
	ProjectName  string   `json:"project_name"`
	Architecture string   `json:"architecture"`
	Components   []string `json:"components"`
	Status       string   `json:"status"`
}

type AllocateBudgetRequest struct {
	ProjectName string  `json:"project_name" jsonschema_description:"Name of the project"`
	Amount      float64 `json:"amount" jsonschema_description:"Budget amount to allocate"`
	Department  string  `json:"department" jsonschema_description:"Department to allocate budget from"`
}

type AllocateBudgetResponse struct {
	AllocationID    string  `json:"allocation_id"`
	ProjectName     string  `json:"project_name"`
	Amount          float64 `json:"amount"`
	Department      string  `json:"department"`
	RemainingBudget float64 `json:"remaining_budget"`
	Status          string  `json:"status"`
}

type AssignTeamRequest struct {
	ProjectName string   `json:"project_name" jsonschema_description:"Name of the project"`
	TeamMembers []string `json:"team_members" jsonschema_description:"List of team member names to assign"`
	StartDate   string   `json:"start_date" jsonschema_description:"Project start date"`
}

type AssignTeamResponse struct {
	AssignmentID string   `json:"assignment_id"`
	ProjectName  string   `json:"project_name"`
	TeamMembers  []string `json:"team_members"`
	StartDate    string   `json:"start_date"`
	Status       string   `json:"status"`
}

func NewSearchTool(ctx context.Context) (tool.BaseTool, error) {
	return utils.InferTool("search_info", "Search for information on various topics",
		func(ctx context.Context, req *SearchRequest) (*SearchResponse, error) {
			results := map[string][]string{
				"technology": {
					"Latest AI frameworks show 40% improvement in efficiency",
					"Cloud-native architecture adoption increased by 65%",
					"Microservices remain the preferred architecture pattern",
				},
				"business": {
					"Q3 revenue exceeded expectations by 12%",
					"Market expansion opportunities identified in APAC region",
					"Customer satisfaction scores improved to 4.5/5",
				},
				"market": {
					"Industry growth projected at 8.5% annually",
					"Competitor analysis shows market gap in enterprise segment",
					"Emerging markets present significant opportunities",
				},
			}

			topic := req.Topic
			if topic == "" {
				topic = "technology"
			}

			if res, ok := results[topic]; ok {
				return &SearchResponse{
					Query:   req.Query,
					Results: res,
					Source:  fmt.Sprintf("%s Research Database", topic),
				}, nil
			}

			return &SearchResponse{
				Query:   req.Query,
				Results: []string{"General information found for: " + req.Query},
				Source:  "General Database",
			}, nil
		})
}

func NewAnalyzeRequirementsTool(ctx context.Context) (tool.BaseTool, error) {
	return utils.InferTool("analyze_requirements", "Analyze project requirements and estimate complexity",
		func(ctx context.Context, req *AnalyzeRequirementsRequest) (*AnalyzeRequirementsResponse, error) {
			hashInput := req.ProjectDescription
			complexity := []string{"Low", "Medium", "High"}
			complexityIdx := consistentHashing(hashInput+"complexity", 0, 2)

			return &AnalyzeRequirementsResponse{
				Requirements: []string{
					"User authentication and authorization",
					"Data storage and retrieval system",
					"API integration layer",
					"User interface components",
					"Testing and quality assurance",
				},
				Complexity:     complexity[complexityIdx],
				EstimatedHours: consistentHashing(hashInput+"hours", 80, 320),
			}, nil
		})
}

func NewCreateDesignTool(ctx context.Context) (tool.BaseTool, error) {
	return utils.InferTool("create_design", "Create a technical design document for the project",
		func(ctx context.Context, req *CreateDesignRequest) (*CreateDesignResponse, error) {
			hashInput := req.ProjectName

			return &CreateDesignResponse{
				DesignID:     fmt.Sprintf("DESIGN-%d", consistentHashing(hashInput+"id", 1000, 9999)),
				ProjectName:  req.ProjectName,
				Architecture: "Microservices with Event-Driven Architecture",
				Components: []string{
					"API Gateway",
					"Authentication Service",
					"Core Business Logic Service",
					"Database Layer",
					"Message Queue",
					"Monitoring Dashboard",
				},
				Status: "draft",
			}, nil
		})
}

func NewAllocateBudgetTool(ctx context.Context) (tool.BaseTool, error) {
	baseTool, err := utils.InferTool("allocate_budget", "Allocate budget for a project. This is a sensitive financial operation that requires approval.",
		func(ctx context.Context, req *AllocateBudgetRequest) (*AllocateBudgetResponse, error) {
			departmentBudgets := map[string]float64{
				"engineering": 500000.00,
				"marketing":   200000.00,
				"operations":  300000.00,
			}

			remaining := departmentBudgets[req.Department] - req.Amount
			if remaining < 0 {
				remaining = 0
			}

			return &AllocateBudgetResponse{
				AllocationID:    fmt.Sprintf("BUDGET-%s-%d", req.Department[:3], consistentHashing(req.ProjectName+"budget", 1000, 9999)),
				ProjectName:     req.ProjectName,
				Amount:          req.Amount,
				Department:      req.Department,
				RemainingBudget: remaining,
				Status:          "approved",
			}, nil
		})
	if err != nil {
		return nil, err
	}

	return &tool2.InvokableApprovableTool{InvokableTool: baseTool}, nil
}

func NewAssignTeamTool(ctx context.Context) (tool.BaseTool, error) {
	return utils.InferTool("assign_team", "Assign team members to a project",
		func(ctx context.Context, req *AssignTeamRequest) (*AssignTeamResponse, error) {
			return &AssignTeamResponse{
				AssignmentID: fmt.Sprintf("TEAM-%d", consistentHashing(req.ProjectName+"team", 1000, 9999)),
				ProjectName:  req.ProjectName,
				TeamMembers:  req.TeamMembers,
				StartDate:    req.StartDate,
				Status:       "confirmed",
			}, nil
		})
}

func GetResearchTools(ctx context.Context) ([]tool.BaseTool, error) {
	searchTool, err := NewSearchTool(ctx)
	if err != nil {
		return nil, err
	}
	return []tool.BaseTool{searchTool}, nil
}

func GetProjectExecutionTools(ctx context.Context) ([]tool.BaseTool, error) {
	analyzeTool, err := NewAnalyzeRequirementsTool(ctx)
	if err != nil {
		return nil, err
	}

	designTool, err := NewCreateDesignTool(ctx)
	if err != nil {
		return nil, err
	}

	budgetTool, err := NewAllocateBudgetTool(ctx)
	if err != nil {
		return nil, err
	}

	teamTool, err := NewAssignTeamTool(ctx)
	if err != nil {
		return nil, err
	}

	return []tool.BaseTool{analyzeTool, designTool, budgetTool, teamTool}, nil
}

func consistentHashing(s string, min, max int) int {
	h := fnv.New32a()
	h.Write([]byte(s))
	hash := h.Sum32()
	return min + int(hash)%(max-min+1)
}
