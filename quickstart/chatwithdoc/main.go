/*
 * Copyright 2026 CloudWeGo Authors
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
	"log"
	"os"
	"path/filepath"
	"time"

	clc "github.com/cloudwego/eino-ext/callbacks/cozeloop"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/callbacks"
	"github.com/coze-dev/cozeloop-go"

	adkstore "github.com/cloudwego/eino-examples/adk/common/store"
	"github.com/cloudwego/eino-examples/quickstart/chatwithdoc/mem"
	"github.com/cloudwego/eino-examples/quickstart/chatwithdoc/server"
)

func main() {
	ctx := context.Background()

	// setup cozeloop tracing (optional)
	// COZELOOP_WORKSPACE_ID=your workspace id
	// COZELOOP_API_TOKEN=your token
	cozeloopApiToken := os.Getenv("COZELOOP_API_TOKEN")
	cozeloopWorkspaceID := os.Getenv("COZELOOP_WORKSPACE_ID")
	if cozeloopApiToken != "" && cozeloopWorkspaceID != "" {
		client, err := cozeloop.NewClient(
			cozeloop.WithAPIToken(cozeloopApiToken),
			cozeloop.WithWorkspaceID(cozeloopWorkspaceID),
		)
		if err != nil {
			log.Fatalf("cozeloop.NewClient failed: %v", err)
		}
		defer func() {
			time.Sleep(5 * time.Second)
			client.Close(ctx)
		}()
		callbacks.AppendGlobalHandlers(clc.NewLoopHandler(client))
	}

	agent, err := buildAgent(ctx)
	if err != nil {
		log.Fatalf("failed to build agent: %v", err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
		CheckPointStore: adkstore.NewInMemoryStore(),
	})

	sessionDir := os.Getenv("SESSION_DIR")
	if sessionDir == "" {
		sessionDir = "./data/sessions"
	}

	workspaceDir := os.Getenv("WORKSPACE_DIR")
	if workspaceDir == "" {
		workspaceDir = "./data/workspace"
	}

	store, err := mem.NewStore(sessionDir)
	if err != nil {
		log.Fatalf("failed to create session store: %v", err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	projectRoot := os.Getenv("PROJECT_ROOT")
	if projectRoot == "" {
		// Default: the directory from which the binary is run.
		// Override with PROJECT_ROOT=/path/to/repo to give the agent full codebase access.
		if cwd, err := os.Getwd(); err == nil {
			projectRoot = cwd
		}
	}
	if abs, err := filepath.Abs(projectRoot); err == nil {
		projectRoot = abs
	}
	log.Printf("project root: %s", projectRoot)

	// EXAMPLES_DIR points to the root of the eino-examples repository.
	// Defaults to PROJECT_ROOT/examples if that directory exists, otherwise PROJECT_ROOT.
	examplesDir := os.Getenv("EXAMPLES_DIR")
	if examplesDir == "" {
		candidate := filepath.Join(projectRoot, "examples")
		if fi, err := os.Stat(candidate); err == nil && fi.IsDir() {
			examplesDir = candidate
		} else {
			examplesDir = projectRoot
		}
	}
	if abs, err := filepath.Abs(examplesDir); err == nil {
		examplesDir = abs
	}
	log.Printf("examples dir: %s", examplesDir)

	srv := server.New(server.Config{
		Runner:       runner,
		Store:        store,
		WorkspaceDir: workspaceDir,
		ProjectRoot:  projectRoot,
		ExamplesDir:  examplesDir,
		Port:         port,
	})

	log.Printf("starting server on http://localhost:%s", port)
	srv.Spin()
}
