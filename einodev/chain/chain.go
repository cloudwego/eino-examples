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

package chain

import (
	"context"

	"github.com/cloudwego/eino/compose"

	"github.com/cloudwego/eino-examples/internal/logs"
)

func RegisterSimpleChain(ctx context.Context) {

	chain := compose.NewChain[string, string]()

	c1 := compose.InvokableLambda(func(ctx context.Context, input string) (output string, err error) {
		return input + " process by node_1,", nil
	})

	c2 := compose.InvokableLambda(func(ctx context.Context, input string) (output string, err error) {
		return input + " process by node_2,", nil
	})

	chain.AppendLambda(c1, compose.WithNodeName("c1")).
		AppendLambda(c2, compose.WithNodeName("c2"))

	r, err := chain.Compile(ctx)
	if err != nil {
		logs.Infof("compile chain failed, err=%v", err)
		return
	}

	message, err := r.Invoke(ctx, "eino chain test")
	if err != nil {
		logs.Infof("invoke chain failed, err=%v", err)
		return
	}

	logs.Infof("eino simple chain output is: %v", message)
}
