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

package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/cloudwego/eino-ext/components/model/ollama"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	cbutils "github.com/cloudwego/eino/utils/callbacks"

	"github.com/cloudwego/eino-examples/components/model/abtest"
)

func main() {
	ctx := context.Background()
	handler := cbutils.NewHandlerHelper().ChatModel(&cbutils.ModelCallbackHandler{
		OnStart: func(ctx context.Context, info *callbacks.RunInfo, input *model.CallbackInput) context.Context {
			if info.Component == components.ComponentOfChatModel {
				log.Printf("[abtest choice] %s", info.Name)
			}
			return ctx
		},
	}).Handler()
	ctx = callbacks.InitCallbacks(ctx, &callbacks.RunInfo{Name: "AB-Example", Component: components.ComponentOfChatModel}, handler)
	var t float32 = 0
	oai, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:      os.Getenv("OPENAI_API_KEY"),
		BaseURL:     os.Getenv("OPENAI_BASE_URL"),
		Model:       os.Getenv("OPENAI_MODEL"),
		ByAzure:     os.Getenv("OPENAI_BY_AZURE") == "true",
		Temperature: &t,
	})
	if err != nil {
		log.Fatal(err)
	}
	olm, err := ollama.NewChatModel(ctx, &ollama.ChatModelConfig{
		BaseURL: os.Getenv("OLLAMA_BASE_URL"),
		Model:   os.Getenv("OLLAMA_MODEL_NAME"),
	})
	if err != nil {
		log.Fatal(err)
	}

	router := abtest.NewABRouterChatModel(func(ctx context.Context, in []*schema.Message, _ ...model.Option) (string, model.BaseChatModel, error) {
		if len(in) == 0 {
			return "openai", oai, nil
		}
		m := in[len(in)-1]
		if m.Role == schema.User && len(m.Content)%2 == 0 {
			return "openai", oai, nil
		}
		return "ollama", olm, nil
	})

	msgs := []*schema.Message{
		schema.SystemMessage("You are a helpful assistant."),
		schema.UserMessage("Tell me a joke about gophers."),
	}
	sr, err := router.Stream(ctx, msgs)
	if err != nil {
		log.Fatal(err)
	}
	defer sr.Close()
	for {
		msg, err := sr.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			log.Fatal(err)
		}
		fmt.Print(msg.Content)
	}
}
