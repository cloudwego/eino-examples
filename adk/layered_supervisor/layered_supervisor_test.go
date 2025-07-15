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
	"fmt"
	"log"
	"testing"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"

	adk2 "github.com/cloudwego/eino-examples/adk"
)

func TestLayeredSupervisor(t *testing.T) {
	sv, err := buildSupervisor(context.Background())
	if err != nil {
		log.Fatalf("build layered supervisor failed: %v", err)
	}

	query := "find US and New York state GDP in 2024. what % of US GDP was New York state? " +
		"Then multiply that percentage by 1.589."

	iter := sv.Run(context.Background(), &adk.AgentInput{
		Messages: []adk.Message{
			schema.UserMessage(query),
		},
		EnableStreaming: true,
	})

	fmt.Println("\nuser query: ", query)

	for {
		event, hasEvent := iter.Next()
		if !hasEvent {
			break
		}

		adk2.PrintEvent(event)
	}
}
