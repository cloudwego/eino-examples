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
	"net/http"
	"os"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/schema"

	"github.com/cloudwego/eino-examples/components/model/httptransport"
)

func main() {
	ctx := context.Background()

	client := &http.Client{Transport: httptransport.NewCurlRT(
		http.DefaultTransport,
		httptransport.WithLogger(log.Default()),
		httptransport.WithCtxLogger(httptransport.IDCtxLogger{L: log.Default()}),
		httptransport.WithPrintAuth(false),
		httptransport.WithMaskHeaders([]string{"X-API-KEY", "API-KEY"}),
		httptransport.WithStreamLogging(true),
		httptransport.WithMaxStreamLogBytes(8192),
	)}

	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL:    os.Getenv("OPENAI_BASE_URL"),
		APIKey:     os.Getenv("OPENAI_API_KEY"),
		Model:      os.Getenv("OPENAI_MODEL"),
		ByAzure:    os.Getenv("OPENAI_BY_AZURE") == "true",
		HTTPClient: client,
	})
	if err != nil {
		log.Fatal(err)
	}

	input := []*schema.Message{
		schema.SystemMessage("You are a helpful assistant."),
		schema.UserMessage("Stream a single-sentence greeting."),
	}
	ctx = context.WithValue(ctx, "log_id", "stream-req-001")
	sr, err := chatModel.Stream(ctx, input)
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
