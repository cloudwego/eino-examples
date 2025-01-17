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

	"github.com/cloudwego/eino-ext/components/retriever/volc_vikingdb"
	"github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/flow/retriever/router"

	"github.com/cloudwego/eino-examples/internal/logs"
)

func main() {

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

	// route retriever by custom router
	mqr, err := router.NewRetriever(ctx, &router.Config{
		Retrievers: map[string]retriever.Retriever{
			"1": vk,
			"2": vk,
			"3": vk,
		},
		Router: func(ctx context.Context, query string) ([]string, error) {
			var ret []string
			if strings.Contains(query, "1") {
				ret = append(ret, "1")
			}
			if strings.Contains(query, "2") {
				ret = append(ret, "2")
			}
			if strings.Contains(query, "3") {
				ret = append(ret, "3")
			}
			return ret, nil
		},
		FusionFunc: nil, // use default rrf
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

	logs.Infof("Router Retrieve success, docs=%v", resp)
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
