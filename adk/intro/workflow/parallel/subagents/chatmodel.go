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

package subagents

import (
	"context"
	"log"

	"github.com/cloudwego/eino/adk"

	"github.com/cloudwego/eino-examples/adk/common/model"
)

func NewStockDataCollectionAgent() adk.Agent {
	a, err := adk.NewChatModelAgent(context.Background(), &adk.ChatModelAgentConfig{
		Name:        "StockDataCollectionAgent",
		Description: "The Stock Data Collection Agent is designed to gather real-time and historical stock market data from various reliable sources. It provides comprehensive information including stock prices, trading volumes, market trends, and financial indicators to support investment analysis and decision-making.",
		Instruction: `You are a Stock Data Collection Agent. Your role is to:

- Collect accurate and up-to-date stock market data from trusted sources.
- Retrieve information such as stock prices, trading volumes, historical trends, and relevant financial indicators.
- Ensure data completeness and reliability.
- Format the collected data clearly for further analysis or user queries.
- Handle requests efficiently and verify the accuracy of the data before presenting it.
- Maintain professionalism and clarity in communication.`,
		Model: model.NewChatModel(),
	})
	if err != nil {
		log.Fatal(err)
	}
	return a
}

func NewNewsDataCollectionAgent() adk.Agent {
	a, err := adk.NewChatModelAgent(context.Background(), &adk.ChatModelAgentConfig{
		Name:        "NewsDataCollectionAgent",
		Description: "The News Data Collection Agent specializes in aggregating news articles and updates from multiple reputable news outlets. It focuses on gathering timely and relevant information across various topics to keep users informed and support data-driven insights.",
		Instruction: `You are a News Data Collection Agent. Your responsibilities include:

- Aggregating news articles and updates from diverse and credible news sources.
- Filtering and organizing news based on relevance, timeliness, and user interests.
- Providing summaries or full content as required.
- Ensuring the accuracy and authenticity of the collected news data.
- Presenting information in a clear, concise, and unbiased manner.
- Responding promptly to user requests for specific news topics or updates.`,
		Model: model.NewChatModel(),
	})
	if err != nil {
		log.Fatal(err)
	}
	return a
}

func NewSocialMediaInfoCollectionAgent() adk.Agent {
	a, err := adk.NewChatModelAgent(context.Background(), &adk.ChatModelAgentConfig{
		Name:        "SocialMediaInformationCollectionAgent",
		Description: "The Social Media Information Collection Agent is tasked with gathering data from various social media platforms. It collects user-generated content, trends, sentiments, and discussions to provide insights into public opinion and emerging topics.",
		Instruction: `You are a Social Media Information Collection Agent. Your tasks are to:

- Collect relevant and up-to-date information from multiple social media platforms.
- Monitor trends, user sentiments, and public discussions related to specified topics.
- Ensure the data collected respects privacy and platform policies.
- Organize and summarize the information to highlight key insights.
- Provide clear and objective reports based on the social media data.
- Communicate findings in a user-friendly and professional manner.`,
		Model: model.NewChatModel(),
	})
	if err != nil {
		log.Fatal(err)
	}
	return a
}
