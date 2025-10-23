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

package agents

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
)

func NewCodeAgent(ctx context.Context, tcm model.ToolCallingChatModel) (adk.Agent, error) {
	type RAGInput struct {
		Query   string  `json:"query" jsonschema_description:"query for search"`
		Context *string `json:"context" jsonschema_description:"user input context"`
	}
	type RAGOutput struct {
		Documents []string `json:"documents"`
	}
	knowledgeBaseTool, err := utils.InferTool(
		"knowledge_base",
		"knowledge base which could answer common questions, provide specific reasons for answers, and improve accuracy",
		func(ctx context.Context, input *RAGInput) (output *RAGOutput, err error) {
			// replace it with real knowledge base search
			if input.Query == "" {
				return nil, fmt.Errorf("RAG Input query is required")
			}

			return &RAGOutput{
				[]string{
					"Q: What is the difference between a list and a tuple in Python?\nA: A list is mutable, meaning you can modify its elements after creation, while a tuple is immutable and cannot be changed once created. Lists use square brackets [], tuples use parentheses ().",
					"Q: How do you handle exceptions in Java?\nA: You handle exceptions in Java using try-catch blocks. Code that might throw an exception is placed inside the try block, and the catch block handles the exception. Optionally, a finally block can be used for cleanup.",
					"Q: What is the purpose of the async and await keywords in JavaScript?\nA: async marks a function as asynchronous, allowing it to return a Promise. await pauses the execution of an async function until the Promise resolves, enabling easier asynchronous code writing.",
					"Q: How can you optimize SQL queries for better performance?\nA: Common optimizations include creating indexes on frequently queried columns, avoiding SELECT *, using JOINs efficiently, and analyzing query execution plans to identify bottlenecks.",
					"Q: What is dependency injection and why is it useful?\nA: Dependency injection is a design pattern where an object receives its dependencies from an external source rather than creating them itself. It promotes loose coupling, easier testing, and better code maintainability.",
				},
			}, nil
		})
	if err != nil {
		return nil, err
	}

	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "CodeAgent",
		Description: "The CodeAgent specializes in generating high-quality code by leveraging a knowledge base as a tool. It recalls relevant knowledge and best practices to produce efficient, maintainable, and accurate code solutions tailored to the project requirements.",
		Instruction: `You are the CodeAgent. Your responsibilities include:

- Generating high-quality, efficient, and maintainable code based on the project requirements.
- Utilizing a knowledge base tool to recall relevant coding standards, patterns, and best practices.
- Ensuring the code is clear, well-documented, and meets the specified functionality.
- Reviewing related knowledge to enhance the accuracy and quality of your code.
- Communicating your coding decisions and providing explanations when necessary.
- Responding promptly and professionally to user requests or clarifications.

Tool handling:
When the user's question is vague or exceeds the scope of your answer, please use the knowledge_base tool to recall relevant results from the knowledge base and provide accurate answers based on the results.
`,
		Model: tcm,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{knowledgeBaseTool},
			},
		},
		MaxIterations: 3,
	})
}
