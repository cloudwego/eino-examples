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

// ch10 demonstrates the full chat-with-eino web application using adk.Runner.
// This is the "before TurnLoop" version — it has the complete web UI with sessions,
// streaming, approval/interrupt, and file upload, but uses adk.Runner instead of
// TurnLoop, so there is no preempt or abort support.
//
// Routes:
//
//	GET  /                        → serves static/index.html
//	POST /sessions                → create a new session
//	GET  /sessions                → list all sessions
//	DELETE /sessions/:id          → delete a session
//	POST /sessions/:id/chat       → stream agent response as SSE (A2UI JSONL)
//	GET  /sessions/:id/render     → render session history as A2UI messages
//	POST /sessions/:id/approve    → resume an interrupted agent with approval decision
//	POST /sessions/:id/docs       → upload a document to the session workspace
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	hserver "github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/google/uuid"
	"github.com/hertz-contrib/sse"

	localbk "github.com/cloudwego/eino-ext/adk/backend/local"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/deep"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	examplemodel "github.com/cloudwego/eino-examples/adk/common/model"
	adkstore "github.com/cloudwego/eino-examples/adk/common/store"
	commontool "github.com/cloudwego/eino-examples/adk/common/tool"
	"github.com/cloudwego/eino-examples/quickstart/chatwitheino/a2ui"
	"github.com/cloudwego/eino-examples/quickstart/chatwitheino/mem"
	"github.com/cloudwego/eino-examples/quickstart/chatwitheino/rag"
)

func main() {
	ctx := context.Background()

	agent, err := buildAgent(ctx)
	if err != nil {
		log.Fatalf("build agent: %v", err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming:  true,
		CheckPointStore: adkstore.NewInMemoryStore(),
	})

	sessionDir := os.Getenv("SESSION_DIR")
	if sessionDir == "" {
		sessionDir = "./data/sessions"
	}
	store, err := mem.NewStore(sessionDir)
	if err != nil {
		log.Fatalf("create store: %v", err)
	}

	workspaceDir := os.Getenv("WORKSPACE_DIR")
	if workspaceDir == "" {
		workspaceDir = "./data/workspace"
	}

	projectRoot := os.Getenv("PROJECT_ROOT")
	if projectRoot == "" {
		if cwd, err := os.Getwd(); err == nil {
			projectRoot = cwd
		}
	}
	if abs, err := filepath.Abs(projectRoot); err == nil {
		projectRoot = abs
	}
	log.Printf("project root: %s", projectRoot)

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

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &server{
		runner:       runner,
		store:        store,
		workspaceDir: workspaceDir,
		projectRoot:  projectRoot,
		examplesDir:  examplesDir,
	}

	h := hserver.Default(hserver.WithHostPorts(":" + port))

	h.GET("/", func(_ context.Context, c *app.RequestContext) {
		data, err := os.ReadFile("static/index.html")
		if err != nil {
			c.JSON(consts.StatusNotFound, map[string]string{"error": "index.html not found"})
			return
		}
		c.Data(consts.StatusOK, "text/html; charset=utf-8", data)
	})

	h.POST("/sessions", func(_ context.Context, c *app.RequestContext) {
		id := uuid.New().String()
		if _, err := store.GetOrCreate(id); err != nil {
			c.JSON(consts.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		c.JSON(consts.StatusOK, map[string]string{"id": id})
	})

	h.GET("/sessions", func(_ context.Context, c *app.RequestContext) {
		metas, err := store.List()
		if err != nil {
			c.JSON(consts.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if metas == nil {
			metas = []mem.SessionMeta{}
		}
		c.JSON(consts.StatusOK, metas)
	})

	h.DELETE("/sessions/:id", func(_ context.Context, c *app.RequestContext) {
		id := c.Param("id")
		if err := store.Delete(id); err != nil {
			c.JSON(consts.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		c.Status(consts.StatusNoContent)
	})

	h.POST("/sessions/:id/chat", func(ctx context.Context, c *app.RequestContext) {
		srv.handleChat(ctx, c)
	})

	h.GET("/sessions/:id/render", func(_ context.Context, c *app.RequestContext) {
		srv.handleRender(c)
	})

	h.POST("/sessions/:id/approve", func(ctx context.Context, c *app.RequestContext) {
		srv.handleApprove(ctx, c)
	})

	h.POST("/sessions/:id/abort", func(_ context.Context, c *app.RequestContext) {
		// No-op: abort requires TurnLoop (introduced in the next chapter).
		c.JSON(consts.StatusOK, map[string]string{"status": "not supported without TurnLoop"})
	})

	h.POST("/sessions/:id/docs", func(_ context.Context, c *app.RequestContext) {
		srv.handleUpload(c)
	})

	log.Printf("starting server on http://localhost:%s", port)
	h.Spin()
}

// server holds shared state for all handlers.
type server struct {
	runner       *adk.Runner
	store        *mem.Store
	workspaceDir string
	projectRoot  string
	examplesDir  string
}

// --- HTTP handlers ---

type chatRequest struct {
	Message string `json:"message"`
}

type approveRequest struct {
	Approved bool   `json:"approved"`
	Reason   string `json:"reason,omitempty"`
}

func (s *server) handleChat(ctx context.Context, c *app.RequestContext) {
	id := c.Param("id")

	body, _ := c.Body()
	var req chatRequest
	if err := json.Unmarshal(body, &req); err != nil || req.Message == "" {
		c.JSON(consts.StatusBadRequest, map[string]string{"error": "message is required"})
		return
	}

	log.Printf("[chat] session=%s msg=%q", id, req.Message)

	sess, err := s.store.GetOrCreate(id)
	if err != nil {
		c.JSON(consts.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	userMsg := schema.UserMessage(req.Message)
	if err := sess.Append(userMsg); err != nil {
		log.Printf("warn: failed to persist user message: %v", err)
	}

	history := sess.GetMessages()
	runMessages := s.buildRunMessages(id, history)
	events := s.runner.Run(ctx, runMessages, adk.WithCheckPointID(id))

	stream := sse.NewStream(c)
	defer func() { _ = c.Flush() }()

	lastContent, intermediates, interruptID, _, streamErr := a2ui.StreamToWriter(
		&sseLineWriter{stream: stream}, id, history, events,
	)

	for _, msg := range intermediates {
		if appendErr := sess.Append(msg); appendErr != nil {
			log.Printf("warn: failed to persist intermediate: %v", appendErr)
		}
	}

	if interruptID != "" {
		sess.SetPendingInterruptID(interruptID)
		log.Printf("[chat] session=%s interrupted: id=%s", id, interruptID)
	} else if streamErr != nil {
		log.Printf("[chat] session=%s error: %v", id, streamErr)
	} else {
		log.Printf("[chat] session=%s done, response=%d chars", id, len(lastContent))
	}
}

func (s *server) handleRender(c *app.RequestContext) {
	id := c.Param("id")
	sess, err := s.store.GetOrCreate(id)
	if err != nil {
		c.JSON(consts.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	var buf bytes.Buffer
	if err := a2ui.RenderHistory(&buf, id, sess.GetMessages()); err != nil {
		c.JSON(consts.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	c.Data(consts.StatusOK, "application/x-ndjson", buf.Bytes())
}

func (s *server) handleApprove(ctx context.Context, c *app.RequestContext) {
	id := c.Param("id")

	sess, err := s.store.GetOrCreate(id)
	if err != nil {
		c.JSON(consts.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	interruptID := sess.GetPendingInterruptID()
	if interruptID == "" {
		c.JSON(consts.StatusBadRequest, map[string]string{"error": "no pending interrupt for this session"})
		return
	}

	body, _ := c.Body()
	var req approveRequest
	if err := json.Unmarshal(body, &req); err != nil {
		c.JSON(consts.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	var reason *string
	if req.Reason != "" {
		reason = &req.Reason
	}
	result := &commontool.ApprovalResult{Approved: req.Approved, DisapproveReason: reason}

	sess.SetPendingInterruptID("")

	log.Printf("[approve] session=%s interruptID=%s approved=%v", id, interruptID, req.Approved)

	// Resume via Runner with the approval decision.
	events, err2 := s.runner.ResumeWithParams(ctx, id, &adk.ResumeParams{
		Targets: map[string]any{interruptID: result},
	})
	if err2 != nil {
		c.JSON(consts.StatusInternalServerError, map[string]string{"error": err2.Error()})
		return
	}

	stream := sse.NewStream(c)
	defer func() { _ = c.Flush() }()

	msgIdx := sess.GetMsgIdx()
	lastContent, newInterruptID, finalMsgIdx, streamErr := a2ui.StreamContinue(
		&sseLineWriter{stream: stream}, id, msgIdx, events,
	)
	_ = finalMsgIdx

	if newInterruptID != "" {
		sess.SetPendingInterruptID(newInterruptID)
		sess.SetMsgIdx(finalMsgIdx)
		log.Printf("[approve] session=%s re-interrupted: id=%s", id, newInterruptID)
	} else if streamErr != nil {
		log.Printf("[approve] session=%s stream error: %v", id, streamErr)
	} else {
		log.Printf("[approve] session=%s done, response=%d chars", id, len(lastContent))
	}
}

func (s *server) handleUpload(c *app.RequestContext) {
	id := c.Param("id")

	absWorkDir, err := filepath.Abs(filepath.Join(s.workspaceDir, id))
	if err != nil {
		c.JSON(consts.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if err := os.MkdirAll(absWorkDir, 0o755); err != nil {
		c.JSON(consts.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(consts.StatusBadRequest, map[string]string{"error": "file field is required"})
		return
	}

	dst := filepath.Join(absWorkDir, filepath.Base(fileHeader.Filename))
	if err := c.SaveUploadedFile(fileHeader, dst); err != nil {
		c.JSON(consts.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	c.JSON(consts.StatusOK, map[string]string{
		"name": fileHeader.Filename,
		"path": dst,
	})
}

// buildRunMessages prepends a context message so the agent knows about the
// project root and the session workspace. This message is never stored in history.
func (s *server) buildRunMessages(sessionID string, history []*schema.Message) []*schema.Message {
	var lines []string
	lines = append(lines, "[Context]")
	lines = append(lines,
		"IMPORTANT RULES:",
		"  1. Always use filesystem tools to look up real code before answering. Do not guess or make up information.",
		"  2. After using tools (even if they return no results), you MUST write a text response to the user summarizing what you found.",
		"  3. Never end your turn without a text response — tool calls alone are not sufficient.",
		"  4. When asked to build or test code, use the execute tool to run the command.",
		"     Each Go example has its own go.mod. To build an example, run:",
		"       cd <example-dir> && go build ./...",
		"     NEVER assume a build succeeded without actually running it.",
		"  5. When writing or editing a file and then claiming it compiles, you MUST run the build tool to verify.",
	)

	if s.projectRoot != "" {
		lines = append(lines,
			fmt.Sprintf("Project root: %s", s.projectRoot),
			"  IMPORTANT: Always pass the project root as the path argument when using filesystem tools.",
			fmt.Sprintf("  - grep(pattern=\"...\", path=\"%s\")", s.projectRoot),
			fmt.Sprintf("  - glob(pattern=\"%s/**/*.go\")", s.projectRoot),
			fmt.Sprintf("  - read_file(file_path=\"%s/some/file.go\")", s.projectRoot),
			"  grep and glob recurse into ALL subdirectories under the given path.",
			"  Top-level subdirectories of the project root:",
		)
		if entries, err := os.ReadDir(s.projectRoot); err == nil {
			for _, e := range entries {
				if e.IsDir() {
					lines = append(lines, "    - "+filepath.Join(s.projectRoot, e.Name())+"/")
				}
			}
		}
		lines = append(lines, "  Use these tools to read actual source code before answering questions about the codebase.")
	}

	if s.examplesDir != "" && s.examplesDir != s.projectRoot {
		lines = append(lines,
			fmt.Sprintf("eino-examples directory: %s", s.examplesDir),
			"  When the user asks about examples or sample code, search here specifically:",
			fmt.Sprintf("  - grep(pattern=\"...\", path=\"%s\")", s.examplesDir),
			fmt.Sprintf("  - glob(pattern=\"%s/**/*.go\")", s.examplesDir),
		)
	}

	absWorkDir, err := filepath.Abs(filepath.Join(s.workspaceDir, sessionID))
	if err == nil {
		entries, _ := os.ReadDir(absWorkDir)
		var uploadedFiles []string
		for _, e := range entries {
			if !e.IsDir() {
				uploadedFiles = append(uploadedFiles, filepath.Join(absWorkDir, e.Name()))
			}
		}
		if len(uploadedFiles) > 0 {
			lines = append(lines,
				fmt.Sprintf("Session workspace: %s", absWorkDir),
				"  Uploaded files:",
			)
			for _, f := range uploadedFiles {
				lines = append(lines, "    - "+f)
			}
		}
	}

	ctx := strings.Join(lines, "\n")
	runMessages := make([]*schema.Message, 0, len(history)+1)
	runMessages = append(runMessages, schema.UserMessage(ctx))
	runMessages = append(runMessages, history...)
	return runMessages
}

// --- sseLineWriter ---

type sseLineWriter struct {
	stream *sse.Stream
	buf    []byte
}

func (w *sseLineWriter) Write(p []byte) (int, error) {
	w.buf = append(w.buf, p...)
	for {
		idx := -1
		for i, b := range w.buf {
			if b == '\n' {
				idx = i
				break
			}
		}
		if idx < 0 {
			break
		}
		line := w.buf[:idx]
		w.buf = w.buf[idx+1:]
		if len(line) == 0 {
			continue
		}
		if err := w.stream.Publish(&sse.Event{Data: line}); err != nil {
			return 0, err
		}
	}
	return len(p), nil
}

// --- Agent construction ---

func buildAgent(ctx context.Context) (adk.Agent, error) {
	cm := examplemodel.NewChatModel()

	backend, err := localbk.NewBackend(ctx, &localbk.Config{})
	if err != nil {
		return nil, err
	}

	ragTool, err := rag.BuildTool(ctx, cm)
	if err != nil {
		return nil, fmt.Errorf("build rag tool: %w", err)
	}

	handlers := []adk.ChatModelAgentMiddleware{&approvalMiddleware{}, &safeToolMiddleware{}}

	return deep.New(ctx, &deep.Config{
		Name:           "ChatWithEinoAgent",
		Description:    "An agent that reads and answers questions about documents.",
		ChatModel:      cm,
		Backend:        backend,
		StreamingShell: backend,
		MaxIteration:   50,
		Handlers:       handlers,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{ragTool},
			},
		},
		ModelRetryConfig: &adk.ModelRetryConfig{
			MaxRetries: 5,
			IsRetryAble: func(_ context.Context, err error) bool {
				return strings.Contains(err.Error(), "429") ||
					strings.Contains(err.Error(), "Too Many Requests") ||
					strings.Contains(err.Error(), "qpm limit")
			},
		},
	})
}

// --- Middlewares ---

type approvalMiddleware struct {
	*adk.BaseChatModelAgentMiddleware
}

func (m *approvalMiddleware) WrapInvokableToolCall(
	_ context.Context,
	endpoint adk.InvokableToolCallEndpoint,
	tCtx *adk.ToolContext,
) (adk.InvokableToolCallEndpoint, error) {
	if tCtx.Name != "answer_from_document" {
		return endpoint, nil
	}
	return func(ctx context.Context, args string, opts ...tool.Option) (string, error) {
		wasInterrupted, _, storedArgs := tool.GetInterruptState[string](ctx)
		if !wasInterrupted {
			return "", tool.StatefulInterrupt(ctx, &commontool.ApprovalInfo{
				ToolName:        tCtx.Name,
				ArgumentsInJSON: args,
			}, args)
		}

		isTarget, hasData, data := tool.GetResumeContext[*commontool.ApprovalResult](ctx)
		if isTarget && hasData {
			if data.Approved {
				return endpoint(ctx, storedArgs, opts...)
			}
			if data.DisapproveReason != nil {
				return fmt.Sprintf("tool '%s' disapproved: %s", tCtx.Name, *data.DisapproveReason), nil
			}
			return fmt.Sprintf("tool '%s' disapproved", tCtx.Name), nil
		}

		isTarget, _, _ = tool.GetResumeContext[any](ctx)
		if !isTarget {
			return "", tool.StatefulInterrupt(ctx, &commontool.ApprovalInfo{
				ToolName:        tCtx.Name,
				ArgumentsInJSON: storedArgs,
			}, storedArgs)
		}

		return endpoint(ctx, storedArgs, opts...)
	}, nil
}

type safeToolMiddleware struct {
	*adk.BaseChatModelAgentMiddleware
}

func (m *safeToolMiddleware) WrapInvokableToolCall(
	_ context.Context,
	endpoint adk.InvokableToolCallEndpoint,
	_ *adk.ToolContext,
) (adk.InvokableToolCallEndpoint, error) {
	return func(ctx context.Context, args string, opts ...tool.Option) (string, error) {
		result, err := endpoint(ctx, args, opts...)
		if err != nil {
			if _, ok := compose.IsInterruptRerunError(err); ok {
				return "", err
			}
			return fmt.Sprintf("[tool error] %v", err), nil
		}
		return result, nil
	}, nil
}

func (m *safeToolMiddleware) WrapStreamableToolCall(
	_ context.Context,
	endpoint adk.StreamableToolCallEndpoint,
	_ *adk.ToolContext,
) (adk.StreamableToolCallEndpoint, error) {
	return func(ctx context.Context, args string, opts ...tool.Option) (*schema.StreamReader[string], error) {
		sr, err := endpoint(ctx, args, opts...)
		if err != nil {
			if _, ok := compose.IsInterruptRerunError(err); ok {
				return nil, err
			}
			return singleChunkReader(fmt.Sprintf("[tool error] %v", err)), nil
		}
		return safeWrapReader(sr), nil
	}, nil
}

// --- Stream helpers ---

func singleChunkReader(msg string) *schema.StreamReader[string] {
	r, w := schema.Pipe[string](1)
	_ = w.Send(msg, nil)
	w.Close()
	return r
}

func safeWrapReader(sr *schema.StreamReader[string]) *schema.StreamReader[string] {
	r, w := schema.Pipe[string](64)
	go func() {
		defer w.Close()
		for {
			chunk, err := sr.Recv()
			if errors.Is(err, io.EOF) {
				return
			}
			if err != nil {
				_ = w.Send(fmt.Sprintf("\n[tool error] %v", err), nil)
				return
			}
			_ = w.Send(chunk, nil)
		}
	}()
	return r
}
