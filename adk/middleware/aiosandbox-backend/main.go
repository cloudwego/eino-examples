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

// This example demonstrates how to implement a custom filesystem.Backend
// and use it with filesystem.NewMiddleware to provide file system tools
// to an agent running in a remote AIO Sandbox environment.
//
// It supports two modes:
//  1. Direct mode: Set AIO_SANDBOX_BASE_URL with faasInstanceName to use an existing sandbox.
//  2. Managed mode: Set VOLC_ACCESSKEY, VOLC_SECRETKEY, and VEFAAS_FUNCTION_ID
//     to automatically create and cleanup a sandbox via veFaaS API.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/middlewares/filesystem"
	"github.com/cloudwego/eino/adk/prebuilt/deep"
	"github.com/cloudwego/eino/schema"

	"github.com/cloudwego/eino-examples/adk/common/model"
	"github.com/cloudwego/eino-examples/adk/common/prints"
)

func main() {
	ctx := context.Background()

	baseURL, cleanup, err := resolveBaseURL()
	if err != nil {
		log.Fatal(err)
	}
	if cleanup != nil {
		defer cleanup()
	}

	token := os.Getenv("AIO_SANDBOX_TOKEN") // optional

	// Create custom filesystem backend using AIO Sandbox
	backend, err := NewAIOSandboxBackend(ctx, &AIOSandboxBackendConfig{
		BaseURL: baseURL,
		Token:   token,
		WorkDir: "/tmp",
	})
	if err != nil {
		log.Fatal(err)
	}

	// Create filesystem middleware with custom backend
	// This automatically provides: ls, read_file, write_file, edit_file, glob, grep, execute tools
	fsMW, err := filesystem.NewMiddleware(ctx, &filesystem.Config{
		Backend: backend,
		Shell:   backend,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Create chat model
	cm := model.NewChatModel()

	// Create agent with the filesystem middleware
	agent, err := deep.New(ctx, &deep.Config{
		Name:        "FileAgent",
		Description: "An agent that can work with files in a remote sandbox",
		ChatModel:   cm,
		Middlewares: []adk.AgentMiddleware{fsMW},
	})
	if err != nil {
		log.Fatal(err)
	}

	// Run the agent
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
	})

	// Example: Test all filesystem tools (ls, read_file, write_file, edit_file, glob, grep, execute)
	query := schema.UserMessage(`Please test all filesystem tools:
1. execute: run "echo Hello"
2. write_file: create /tmp/test.txt with "Hello World"
3. read_file: read /tmp/test.txt
4. edit_file: replace "Hello" with "Hi" in /tmp/test.txt
5. ls: list /tmp
6. glob: find *.txt in /tmp
7. grep: search "World" in /tmp/*.txt (use glob parameter "*.txt" to limit search scope)
8. execute: run "cat /tmp/test.txt"`)

	fmt.Println("Query:", query.Content)
	fmt.Println()

	iter := runner.Run(ctx, []*schema.Message{query})
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			log.Fatal(event.Err)
		}
		prints.Event(event)
	}
}

// resolveBaseURL determines the sandbox base URL and optional cleanup function.
// In managed mode, it creates a sandbox via veFaaS API and returns a cleanup function
// that kills the sandbox when done.
func resolveBaseURL() (baseURL string, cleanup func(), err error) {
	// Direct mode: use provided base URL (with faasInstanceName query param)
	if u := os.Getenv("AIO_SANDBOX_BASE_URL"); u != "" {
		return u, nil, nil
	}

	// Managed mode: create sandbox via veFaaS control plane
	ak := os.Getenv("VOLC_ACCESSKEY")
	sk := os.Getenv("VOLC_SECRETKEY")
	functionID := os.Getenv("VEFAAS_FUNCTION_ID")
	gatewayURL := os.Getenv("VEFAAS_GATEWAY_URL")

	if ak == "" || sk == "" || functionID == "" || gatewayURL == "" {
		return "", nil, fmt.Errorf("either AIO_SANDBOX_BASE_URL, or VOLC_ACCESSKEY + VOLC_SECRETKEY + VEFAAS_FUNCTION_ID + VEFAAS_GATEWAY_URL must be set")
	}

	mgr, err := NewSandboxManager(&SandboxManagerConfig{
		AccessKey:  ak,
		SecretKey:  sk,
		FunctionID: functionID,
	})
	if err != nil {
		return "", nil, fmt.Errorf("failed to create sandbox manager: %w", err)
	}

	sandboxID, err := mgr.CreateSandbox()
	if err != nil {
		return "", nil, fmt.Errorf("failed to create sandbox: %w", err)
	}
	fmt.Printf("Created sandbox: %s\n", sandboxID)

	baseURL = mgr.DataPlaneBaseURL(gatewayURL, sandboxID)
	cleanup = func() {
		fmt.Printf("Killing sandbox: %s\n", sandboxID)
		if err := mgr.KillSandbox(sandboxID); err != nil {
			log.Printf("Warning: failed to kill sandbox: %v", err)
		}
	}

	return baseURL, cleanup, nil
}
