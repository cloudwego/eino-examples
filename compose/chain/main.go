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
	"fmt"
	"log"
	"math/rand"
	"os"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"github.com/cloudwego/eino-examples/internal/gptr"
	"github.com/cloudwego/eino-examples/internal/logs"
)

func main() {
	openAPIBaseURL := os.Getenv("OPENAI_BASE_URL")
	openAPIAK := os.Getenv("OPENAI_API_KEY")
	modelName := os.Getenv("OPENAI_MODEL_NAME")

	ctx := context.Background()
	// build branch func
	const randLimit = 2
	branchCond := func(ctx context.Context, input map[string]any) (string, error) { // nolint: byted_all_nil_return
		if rand.Intn(randLimit) == 1 {
			return "b1", nil
		}

		return "b2", nil
	}

	b1 := compose.InvokableLambda(func(ctx context.Context, kvs map[string]any) (map[string]any, error) {
		logs.Infof("hello in branch lambda 01")
		if kvs == nil {
			return nil, fmt.Errorf("nil map")
		}

		kvs["role"] = "cat"
		return kvs, nil
	})

	b2 := compose.InvokableLambda(func(ctx context.Context, kvs map[string]any) (map[string]any, error) {
		logs.Infof("hello in branch lambda 02")
		if kvs == nil {
			return nil, fmt.Errorf("nil map")
		}

		kvs["role"] = "dog"
		return kvs, nil
	})

	// build parallel node
	parallel := compose.NewParallel()
	parallel.
		AddLambda("role", compose.InvokableLambda(func(ctx context.Context, kvs map[string]any) (string, error) {
			// may be change role to others by input kvs, for example (dentist/doctor...)
			role, ok := kvs["role"].(string)
			if !ok || role == "" {
				role = "bird"
			}

			return role, nil
		})).
		AddLambda("input", compose.InvokableLambda(func(ctx context.Context, kvs map[string]any) (string, error) {
			return "你的叫声是怎样的？", nil
		}))

	// create chat model node
	cm, err := openai.NewChatModel(context.Background(), &openai.ChatModelConfig{
		BaseURL:     openAPIBaseURL,
		APIKey:      openAPIAK,
		Model:       modelName,
		Temperature: gptr.Of(float32(0.7)),
	})
	if err != nil {
		log.Panic(err)
		return
	}

	rolePlayerChain := compose.NewChain[map[string]any, *schema.Message]()
	rolePlayerChain.
		AppendChatTemplate(prompt.FromMessages(schema.FString, schema.SystemMessage(`You are a {role}.`), schema.UserMessage(`{input}`))).
		AppendChatModel(cm)

	// =========== build chain ===========
	chain := compose.NewChain[map[string]any, string]()
	chain.
		AppendLambda(compose.InvokableLambda(func(ctx context.Context, kvs map[string]any) (map[string]any, error) {
			// do some logic to prepare kv as input val for next node
			// just pass through
			logs.Infof("in view lambda: %v", kvs)
			return kvs, nil
		})).
		AppendBranch(compose.NewChainBranch(branchCond).AddLambda("b1", b1).AddLambda("b2", b2)). // nolint: byted_use_receiver_without_nilcheck
		AppendPassthrough().
		AppendParallel(parallel).
		AppendGraph(rolePlayerChain).
		AppendLambda(compose.InvokableLambda(func(ctx context.Context, m *schema.Message) (string, error) {
			// do some logic to check the output or something
			logs.Infof("in view of messages: %v", m.Content)
			return m.Content, nil
		}))

	// compile
	r, err := chain.Compile(ctx)
	if err != nil {
		log.Panic(err)
		return
	}

	output, err := r.Invoke(context.Background(), map[string]any{})
	if err != nil {
		log.Panic(err)
		return
	}

	logs.Infof("output is : %v", output)
}
