// Package params 管理写在context里面的params
package params

import (
	"context"
	"log"
	"sync"
)

const (
	CustomBizKey = "biz"
)

func InitContextParams(ctx context.Context, key string) context.Context {
	return context.WithValue(ctx, key, &sync.Map{}) // nolint
}

func AppendContextParams(ctx context.Context, key string, values map[string]interface{}) {
	params, ok := ctx.Value(key).(*sync.Map)
	if !ok {
		log.Printf("[params.AppendContextParams] Failed to get params from context")
		return
	}

	for k, v := range values {
		params.Store(k, v)
	}
}

func GetContextParams(ctx context.Context, ctxKey, mapKey string) (interface{}, bool) {
	params, ok := ctx.Value(ctxKey).(*sync.Map)
	if !ok {
		log.Printf("[params.GetContextParams] Failed to get params from context")
		return nil, false
	}

	return params.Load(mapKey)
}

func MustGetContextParams[T any](ctx context.Context, ctxKey, mapKey string) T {
	var empty T
	value, ok := GetContextParams(ctx, ctxKey, mapKey)
	if !ok {
		log.Printf("[params.MustGetPipelineParam] cannot get key: %v", mapKey)
		return empty
	}
	valueT, ok := value.(T)
	if !ok {
		log.Printf("[params.MustGetPipelineParam] value not string, key: %v", mapKey)
		return empty
	}
	return valueT
}

func GetAllContextParams(ctx context.Context, ctxKey string) map[string]interface{} {
	res := make(map[string]interface{})

	params, ok := ctx.Value(ctxKey).(*sync.Map)
	if !ok {
		log.Print("[params.GetAllContextParams] Failed to get params from context")
		return res
	}
	params.Range(func(key, value any) bool {
		keyStr, ok := key.(string)
		if !ok {
			return true
		}
		res[keyStr] = value
		return true
	})
	return res
}
