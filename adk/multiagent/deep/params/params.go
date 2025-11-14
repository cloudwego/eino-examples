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

package params

import (
	"context"
	"log"
	"sync"
)

const customBizKey = "biz"

func InitContextParams(ctx context.Context) context.Context {
	return context.WithValue(ctx, customBizKey, &sync.Map{}) // nolint
}

func AppendContextParams(ctx context.Context, values map[string]interface{}) {
	params, ok := ctx.Value(customBizKey).(*sync.Map)
	if !ok {
		log.Printf("[params.AppendContextParams] Failed to get params from context")
		return
	}

	for k, v := range values {
		params.Store(k, v)
	}
}

func GetTypedContextParams[T any](ctx context.Context, mapKey string) (T, bool) {
	var empty T
	value, ok := getContextParams(ctx, mapKey)
	if !ok {
		return empty, false
	}
	valueT, ok := value.(T)
	if !ok {
		return empty, false
	}
	return valueT, true
}

func MustGetContextParams[T any](ctx context.Context, mapKey string) T {
	var empty T
	value, ok := getContextParams(ctx, mapKey)
	if !ok {
		log.Printf("[params.MustGetContextParams] cannot get key: %v", mapKey)
		return empty
	}
	valueT, ok := value.(T)
	if !ok {
		log.Printf("[params.MustGetContextParams] value not string, key: %v", mapKey)
		return empty
	}
	return valueT
}

func getContextParams(ctx context.Context, mapKey string) (interface{}, bool) {
	params, ok := ctx.Value(customBizKey).(*sync.Map)
	if !ok {
		log.Printf("[params.GetContextParams] Failed to get params from context")
		return nil, false
	}

	return params.Load(mapKey)
}
