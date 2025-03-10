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

package main

import (
	"context"
	"os"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino-ext/components/retriever/volc_vikingdb"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/flow/retriever/multiquery"

	"github.com/cloudwego/eino-examples/internal/logs"
)

func main() {

	openAIAPIKey := os.Getenv("OPENAI_API_KEY")
	openAIBaseURL := os.Getenv("OPENAI_BASE_URL")
	openAIModelName := os.Getenv("OPENAI_MODEL_NAME")

	vikingDBHost := os.Getenv("VIKING_DB_HOST")
	vikingDBRegion := os.Getenv("VIKING_DB_REGION")
	vikingDBAK := os.Getenv("VIKING_DB_AK")
	vikingDBSK := os.Getenv("VIKING_DB_SK")

	ctx := context.Background()
	vk, err := newVikingDBRetriever(ctx, vikingDBHost, vikingDBRegion, vikingDBAK, vikingDBSK)
	if err != nil {
		logs.Errorf("newVikingDBRetriever failed, err=%v", err)
		return
	}

	llm, err := newChatModel(ctx, openAIBaseURL, openAIAPIKey, openAIModelName)
	if err != nil {
		logs.Errorf("newChatModel failed, err=%v", err)
		return
	}

	// rewrite query by llm
	mqr, err := multiquery.NewRetriever(ctx, &multiquery.Config{
		RewriteLLM:      llm,
		RewriteTemplate: nil, // use default
		QueryVar:        "",  // use default
		LLMOutputParser: nil, // use default
		MaxQueriesNum:   3,
		OrigRetriever:   vk,
		FusionFunc:      nil, // use default fusion, just deduplicate by doc id
	})
	if err != nil {
		logs.Errorf("NewMultiQueryRetriever failed, err=%v", err)
		return
	}

	resp, err := mqr.Retrieve(ctx, "tourist attraction")
	if err != nil {
		logs.Errorf("Multi-Query Retrieve failed, err=%v", err)
		return
	}

	logs.Infof("Multi-Query Retrieve success, docs=%v", resp)

	// rewrite query by custom method
	mqr, err = multiquery.NewRetriever(ctx, &multiquery.Config{
		RewriteHandler: func(ctx context.Context, query string) ([]string, error) {
			return strings.Split(query, "\n"), nil
		},
		MaxQueriesNum: 3,
		OrigRetriever: vk,
		FusionFunc:    nil, // use default fusion, just deduplicate by doc id
	})
	if err != nil {
		logs.Errorf("NewMultiQueryRetriever failed, err=%v", err)
		return
	}

	resp, err = mqr.Retrieve(ctx, "tourist attraction")
	if err != nil {
		logs.Errorf("Multi-Query Retrieve failed, err=%v", err)
		return
	}

	logs.Infof("Multi-Query Retrieve success, docs=%v", resp)
}

func newChatModel(ctx context.Context, baseURL, apiKey, modelName string) (model.ChatModel, error) {

	return openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Model:   modelName,
	})
}

func newVikingDBRetriever(ctx context.Context, host, region, ak, sk string) (retriever.Retriever, error) {

	baseTopK := 5
	return volc_vikingdb.NewRetriever(ctx, &volc_vikingdb.RetrieverConfig{
		Host:   host,
		Region: region,
		AK:     ak,
		SK:     sk,
		EmbeddingConfig: volc_vikingdb.EmbeddingConfig{
			UseBuiltin: true,
		},
		Index: "3", // index version, replace if needed
		TopK:  &baseTopK,
	})
}
