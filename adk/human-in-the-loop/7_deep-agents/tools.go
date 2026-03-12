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
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

type SearchRequest struct {
	Query    string `json:"query" jsonschema_description:"The search query to find information"`
	Category string `json:"category" jsonschema_description:"Category of information (market, technology, finance, general)"`
}

type SearchResponse struct {
	Query   string         `json:"query"`
	Results []SearchResult `json:"results"`
}

type SearchResult struct {
	Title   string `json:"title"`
	Summary string `json:"summary"`
	Source  string `json:"source"`
}

type AnalyzeRequest struct {
	Data         string `json:"data" jsonschema_description:"The data to analyze"`
	AnalysisType string `json:"analysis_type" jsonschema_description:"Type of analysis (trend, comparison, summary, statistical)"`
}

type AnalyzeResponse struct {
	AnalysisType string   `json:"analysis_type"`
	Findings     []string `json:"findings"`
	Conclusion   string   `json:"conclusion"`
}

func NewSearchTool(ctx context.Context) (tool.BaseTool, error) {
	return utils.InferTool("search", "Search for information on various topics",
		func(ctx context.Context, req *SearchRequest) (*SearchResponse, error) {
			marketData := map[string][]SearchResult{
				"market": {
					{Title: "Q3 2025 Market Overview", Summary: "Global markets showed mixed performance with tech sector leading gains", Source: "MarketWatch"},
					{Title: "Emerging Markets Analysis", Summary: "Asian markets outperformed expectations with 12% YoY growth", Source: "Bloomberg"},
					{Title: "Commodity Prices Update", Summary: "Oil prices stabilized around $75/barrel amid supply concerns", Source: "Reuters"},
				},
				"technology": {
					{Title: "AI Industry Report 2025", Summary: "AI adoption in enterprises reached 67%, up from 45% in 2024", Source: "Gartner"},
					{Title: "Cloud Computing Trends", Summary: "Multi-cloud strategies dominate with 78% of enterprises using 2+ providers", Source: "IDC"},
					{Title: "Semiconductor Outlook", Summary: "Chip shortage easing with new fab capacity coming online in Q4", Source: "TechCrunch"},
				},
				"finance": {
					{Title: "Interest Rate Forecast", Summary: "Fed expected to maintain rates through Q4 2025", Source: "WSJ"},
					{Title: "Banking Sector Health", Summary: "Major banks report strong Q3 earnings with 15% profit growth", Source: "Financial Times"},
					{Title: "Cryptocurrency Update", Summary: "Bitcoin stabilizes around $45K with institutional adoption increasing", Source: "CoinDesk"},
				},
			}

			category := strings.ToLower(req.Category)
			if category == "" {
				category = "general"
			}

			if results, ok := marketData[category]; ok {
				return &SearchResponse{
					Query:   req.Query,
					Results: results,
				}, nil
			}

			hashInput := req.Query + req.Category
			return &SearchResponse{
				Query: req.Query,
				Results: []SearchResult{
					{
						Title:   fmt.Sprintf("Research on: %s", req.Query),
						Summary: fmt.Sprintf("Comprehensive analysis of %s shows positive trends", req.Query),
						Source:  "Research Database",
					},
					{
						Title:   fmt.Sprintf("Latest Updates: %s", req.Query),
						Summary: fmt.Sprintf("Recent developments in %s indicate growth potential", req.Query),
						Source:  fmt.Sprintf("Source-%d", consistentHashing(hashInput, 1, 100)),
					},
				},
			}, nil
		})
}

func NewAnalyzeTool(ctx context.Context) (tool.BaseTool, error) {
	return utils.InferTool("analyze", "Analyze data and generate insights",
		func(ctx context.Context, req *AnalyzeRequest) (*AnalyzeResponse, error) {
			analysisResults := map[string]struct {
				findings   []string
				conclusion string
			}{
				"trend": {
					findings: []string{
						"Upward trend observed over the past 6 months",
						"Growth rate accelerating in recent quarters",
						"Seasonal patterns detected with Q4 peaks",
					},
					conclusion: "Overall positive trajectory with strong momentum",
				},
				"comparison": {
					findings: []string{
						"Performance exceeds industry average by 15%",
						"Competitive positioning improved year-over-year",
						"Market share gains observed in key segments",
					},
					conclusion: "Favorable comparison against benchmarks",
				},
				"summary": {
					findings: []string{
						"Key metrics show healthy performance",
						"Major milestones achieved on schedule",
						"Strategic initiatives progressing well",
					},
					conclusion: "Overall status is positive with continued growth expected",
				},
				"statistical": {
					findings: []string{
						"Mean value: 45.2, Median: 43.8",
						"Standard deviation: 12.3",
						"95% confidence interval: [40.1, 50.3]",
						"Correlation coefficient: 0.82 (strong positive)",
					},
					conclusion: "Statistical analysis indicates significant patterns with high confidence",
				},
			}

			analysisType := strings.ToLower(req.AnalysisType)
			if analysisType == "" {
				analysisType = "summary"
			}

			if result, ok := analysisResults[analysisType]; ok {
				return &AnalyzeResponse{
					AnalysisType: req.AnalysisType,
					Findings:     result.findings,
					Conclusion:   result.conclusion,
				}, nil
			}

			return &AnalyzeResponse{
				AnalysisType: req.AnalysisType,
				Findings: []string{
					"Analysis completed successfully",
					"Data patterns identified",
					"Insights generated based on input",
				},
				Conclusion: "Analysis complete with actionable insights",
			}, nil
		})
}

func consistentHashing(s string, min, max int) int {
	h := fnv.New32a()
	h.Write([]byte(s))
	hash := h.Sum32()
	return min + int(hash)%(max-min+1)
}
