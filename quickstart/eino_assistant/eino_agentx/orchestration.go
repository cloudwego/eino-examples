/*
 * Copyright 2024 CloudWeGo Authors
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

package eino_agent

import (
	"context"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
)

type AgentGraphBuildConfig struct {
	EinoRetriveKeyOfRetriever       *EinoRetrieverConfig
	PromptTemplateKeyOfChatTemplate *PromptTemplateConfig
	ReactAgentKeyOfLambda           *react.AgentConfig
}

type BuildConfig struct {
	AgentGraph *AgentGraphBuildConfig
}

func BuildAgentGraph(ctx context.Context, config BuildConfig) (r compose.Runnable[*UserMessage, *schema.Message], err error) {
	const (
		EinoRetrive               = "EinoRetrive"
		PromptTemplate            = "PromptTemplate"
		ConvertRetrieverDocuments = "ConvertRetrieverDocuments"
		ConvertRetrieverInput     = "ConvertRetrieverInput"
		ConvertInput              = "ConvertInput"
		ReactAgent                = "ReactAgent"
	)
	g := compose.NewGraph[*UserMessage, *schema.Message]()
	einoRetriveKeyOfRetriever, err := NewEinoRetriever(ctx, config.AgentGraph.EinoRetriveKeyOfRetriever)
	if err != nil {
		return nil, err
	}
	_ = g.AddRetrieverNode(EinoRetrive, einoRetriveKeyOfRetriever)
	promptTemplateKeyOfChatTemplate, err := NewPromptTemplate(ctx, config.AgentGraph.PromptTemplateKeyOfChatTemplate)
	if err != nil {
		return nil, err
	}
	_ = g.AddChatTemplateNode(PromptTemplate, promptTemplateKeyOfChatTemplate)
	_ = g.AddLambdaNode(ConvertRetrieverDocuments, compose.InvokableLambda(NewDocumentsConvert))
	_ = g.AddLambdaNode(ConvertRetrieverInput, compose.InvokableLambda(NewRetrieverInputConvert))
	_ = g.AddLambdaNode(ConvertInput, compose.InvokableLambda(NewInputConvertor))
	reactAgentKeyOfLambda, err := NewEinoAgent(ctx, config.AgentGraph.ReactAgentKeyOfLambda)
	if err != nil {
		return nil, err
	}
	_ = g.AddLambdaNode(ReactAgent, reactAgentKeyOfLambda)
	_ = g.AddEdge(compose.START, ConvertRetrieverInput)
	_ = g.AddEdge(compose.START, ConvertInput)
	_ = g.AddEdge(ReactAgent, compose.END)
	_ = g.AddEdge(ConvertRetrieverInput, EinoRetrive)
	_ = g.AddEdge(EinoRetrive, ConvertRetrieverDocuments)
	_ = g.AddEdge(ConvertInput, PromptTemplate)
	_ = g.AddEdge(ConvertRetrieverDocuments, PromptTemplate)
	_ = g.AddEdge(PromptTemplate, ReactAgent)
	r, err = g.Compile(ctx, compose.WithGraphName("AgentGraph"), compose.WithNodeTriggerMode(compose.AllPredecessor), compose.WithMaxRunSteps(25))
	if err != nil {
		return nil, err
	}
	return r, nil
}
