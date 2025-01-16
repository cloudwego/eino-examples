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

package index_graph

import (
	"context"

	"github.com/cloudwego/eino-examples/agent/redis"
	"github.com/cloudwego/eino/components/indexer"
)

type RedisVectorStoreConfigImpl struct {
	redis.RedisVectorStoreConfigImpl
}

type RedisVectorStoreConfig struct {
	redis.RedisVectorStoreConfig
}

func defaultRedisVectorStoreConfig(ctx context.Context) (*RedisVectorStoreConfig, error) {
	config := &RedisVectorStoreConfig{}
	return config, nil
}

func NewRedisVectorStore(ctx context.Context, config *RedisVectorStoreConfig) (idx indexer.Indexer, err error) {
	if config == nil {
		config, err = defaultRedisVectorStoreConfig(ctx)
		if err != nil {
			return nil, err
		}
	}

	return redis.NewRedisIndexer(ctx, &config.RedisVectorStoreConfig)
}
