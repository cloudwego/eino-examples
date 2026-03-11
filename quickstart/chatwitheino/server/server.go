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

package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	hserver "github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/google/uuid"
	"github.com/hertz-contrib/sse"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"

	"github.com/cloudwego/eino-examples/adk/common/tool"
	"github.com/cloudwego/eino-examples/quickstart/chatwitheino/a2ui"
	"github.com/cloudwego/eino-examples/quickstart/chatwitheino/mem"
)

// Config holds all dependencies for the HTTP server.
type Config struct {
	Runner       *adk.Runner
	Store        *mem.Store
	WorkspaceDir string
	ProjectRoot  string // root of the codebase the agent can explore
	ExamplesDir  string // root of the eino-examples repo (for example searches)
	Port         string
}

// Server wraps a Hertz HTTP server with the chat-with-doc routes.
type Server struct {
	cfg Config
}

// New creates a Server from the given config.
func New(cfg Config) *Server {
	return &Server{cfg: cfg}
}

// Spin starts the HTTP server (blocking).
func (s *Server) Spin() {
	h := hserver.Default(hserver.WithHostPorts(":" + s.cfg.Port))

	h.GET("/", func(ctx context.Context, c *app.RequestContext) {
		data, err := os.ReadFile("static/index.html")
		if err != nil {
			c.JSON(consts.StatusNotFound, map[string]string{"error": "index.html not found"})
			return
		}
		c.Data(consts.StatusOK, "text/html; charset=utf-8", data)
	})

	h.POST("/sessions", func(ctx context.Context, c *app.RequestContext) {
		id := uuid.New().String()
		if _, err := s.cfg.Store.GetOrCreate(id); err != nil {
			c.JSON(consts.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		c.JSON(consts.StatusOK, map[string]string{"id": id})
	})

	h.GET("/sessions", func(ctx context.Context, c *app.RequestContext) {
		metas, err := s.cfg.Store.List()
		if err != nil {
			c.JSON(consts.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if metas == nil {
			metas = []mem.SessionMeta{}
		}
		c.JSON(consts.StatusOK, metas)
	})

	h.DELETE("/sessions/:id", func(ctx context.Context, c *app.RequestContext) {
		id := c.Param("id")
		if err := s.cfg.Store.Delete(id); err != nil {
			c.JSON(consts.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		c.Status(consts.StatusNoContent)
	})

	h.POST("/sessions/:id/chat", func(ctx context.Context, c *app.RequestContext) {
		s.handleChat(ctx, c)
	})

	h.GET("/sessions/:id/render", func(ctx context.Context, c *app.RequestContext) {
		s.handleRender(ctx, c)
	})

	h.POST("/sessions/:id/approve", func(ctx context.Context, c *app.RequestContext) {
		s.handleApprove(ctx, c)
	})

	h.POST("/sessions/:id/docs", func(ctx context.Context, c *app.RequestContext) {
		s.handleUpload(ctx, c)
	})

	h.Spin()
}

type chatRequest struct {
	Message string `json:"message"`
}

type approveRequest struct {
	Approved bool   `json:"approved"`
	Reason   string `json:"reason,omitempty"`
}

func (s *Server) handleRender(_ context.Context, c *app.RequestContext) {
	id := c.Param("id")
	sess, err := s.cfg.Store.GetOrCreate(id)
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

func (s *Server) handleChat(ctx context.Context, c *app.RequestContext) {
	id := c.Param("id")

	body, _ := c.Body()
	var req chatRequest
	if err := json.Unmarshal(body, &req); err != nil || req.Message == "" {
		c.JSON(consts.StatusBadRequest, map[string]string{"error": "message is required"})
		return
	}

	log.Printf("[chat] session=%s msg=%q", id, req.Message)

	sess, err := s.cfg.Store.GetOrCreate(id)
	if err != nil {
		c.JSON(consts.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	userMsg := schema.UserMessage(req.Message)
	if appendErr := sess.Append(userMsg); appendErr != nil {
		log.Printf("warn: failed to persist user message: %v", appendErr)
	}

	// history is rendered in the UI; runMessages adds workspace context for the agent.
	history := sess.GetMessages()
	runMessages := s.buildRunMessages(id, history)

	log.Printf("[chat] session=%s running agent with %d messages (%d history + %d context)",
		id, len(runMessages), len(history), len(runMessages)-len(history))

	iter := s.cfg.Runner.Run(ctx, runMessages, adk.WithCheckPointID(id))

	stream := sse.NewStream(c)
	defer func() { _ = c.Flush() }()

	// Send a keep-alive ping every 5 s so the SSE connection isn't dropped
	// by Hertz or browser timeouts while the agent is processing tool results.
	kaStop := make(chan struct{})
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-kaStop:
				return
			case <-ticker.C:
				_ = stream.Publish(&sse.Event{Data: []byte{}})
				log.Printf("[chat] session=%s keep-alive ping sent", id)
			}
		}
	}()

	lastContent, interruptID, finalMsgIdx, streamErr := a2ui.StreamToWriter(&sseLineWriter{stream: stream}, id, history, iter)
	close(kaStop)
	if streamErr != nil {
		log.Printf("[chat] session=%s stream error: %v", id, streamErr)
	} else if interruptID != "" {
		log.Printf("[chat] session=%s interrupted: id=%s", id, interruptID)
		sess.SetPendingInterruptID(interruptID)
		sess.SetMsgIdx(finalMsgIdx)
	} else {
		log.Printf("[chat] session=%s done, response=%d chars", id, len(lastContent))
	}

	if lastContent != "" {
		assistantMsg := schema.AssistantMessage(lastContent, nil)
		if appendErr := sess.Append(assistantMsg); appendErr != nil {
			log.Printf("warn: failed to persist assistant message: %v", appendErr)
		}
	}
}

// buildRunMessages prepends a context message so the agent knows about the
// project root and the session workspace. This message is never stored in history.
func (s *Server) buildRunMessages(sessionID string, history []*schema.Message) []*schema.Message {
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

	if s.cfg.ProjectRoot != "" {
		lines = append(lines,
			fmt.Sprintf("Project root: %s", s.cfg.ProjectRoot),
			"  IMPORTANT: Always pass the project root as the path argument when using filesystem tools.",
			fmt.Sprintf("  - grep(pattern=\"...\", path=\"%s\")", s.cfg.ProjectRoot),
			fmt.Sprintf("  - glob(pattern=\"%s/**/*.go\")", s.cfg.ProjectRoot),
			fmt.Sprintf("  - read_file(file_path=\"%s/some/file.go\")", s.cfg.ProjectRoot),
			"  grep and glob recurse into ALL subdirectories under the given path.",
			"  Top-level subdirectories of the project root:",
		)
		if entries, err := os.ReadDir(s.cfg.ProjectRoot); err == nil {
			for _, e := range entries {
				if e.IsDir() {
					lines = append(lines, "    - "+filepath.Join(s.cfg.ProjectRoot, e.Name())+"/")
				}
			}
		}
		lines = append(lines, "  Use these tools to read actual source code before answering questions about the codebase.")
	}

	if s.cfg.ExamplesDir != "" && s.cfg.ExamplesDir != s.cfg.ProjectRoot {
		lines = append(lines,
			fmt.Sprintf("eino-examples directory: %s", s.cfg.ExamplesDir),
			"  When the user asks about examples or sample code, search here specifically:",
			fmt.Sprintf("  - grep(pattern=\"...\", path=\"%s\")", s.cfg.ExamplesDir),
			fmt.Sprintf("  - glob(pattern=\"%s/**/*.go\")", s.cfg.ExamplesDir),
		)
	}

	absWorkDir, err := filepath.Abs(filepath.Join(s.cfg.WorkspaceDir, sessionID))
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

func (s *Server) handleUpload(ctx context.Context, c *app.RequestContext) {
	id := c.Param("id")

	absWorkDir, err := filepath.Abs(filepath.Join(s.cfg.WorkspaceDir, id))
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

// handleApprove resumes an interrupted agent run with the user's approval decision.
// The agent must have been interrupted earlier in this session (via the approval
// middleware). The session ID is used as the checkpoint ID.
func (s *Server) handleApprove(ctx context.Context, c *app.RequestContext) {
	id := c.Param("id")

	sess, err := s.cfg.Store.GetOrCreate(id)
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
	result := &tool.ApprovalResult{Approved: req.Approved, DisapproveReason: reason}

	iter, err := s.cfg.Runner.ResumeWithParams(ctx, id, &adk.ResumeParams{
		Targets: map[string]any{interruptID: result},
	})
	if err != nil {
		c.JSON(consts.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Clear the pending interrupt immediately so a double-approve returns 400.
	sess.SetPendingInterruptID("")

	log.Printf("[approve] session=%s interruptID=%s approved=%v", id, interruptID, req.Approved)

	stream := sse.NewStream(c)
	defer func() { _ = c.Flush() }()

	kaStop := make(chan struct{})
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-kaStop:
				return
			case <-ticker.C:
				_ = stream.Publish(&sse.Event{Data: []byte{}})
			}
		}
	}()

	lastContent, newInterruptID, finalMsgIdx, streamErr := a2ui.StreamContinue(&sseLineWriter{stream: stream}, id, sess.GetMsgIdx(), iter)
	close(kaStop)
	if streamErr != nil {
		log.Printf("[approve] session=%s stream error: %v", id, streamErr)
	} else if newInterruptID != "" {
		log.Printf("[approve] session=%s re-interrupted: id=%s", id, newInterruptID)
		sess.SetPendingInterruptID(newInterruptID)
		sess.SetMsgIdx(finalMsgIdx)
	} else {
		log.Printf("[approve] session=%s done, response=%d chars", id, len(lastContent))
	}

	if lastContent != "" {
		assistantMsg := schema.AssistantMessage(lastContent, nil)
		if appendErr := sess.Append(assistantMsg); appendErr != nil {
			log.Printf("warn: failed to persist assistant message: %v", appendErr)
		}
	}
}

// sseLineWriter implements io.Writer, buffering until a newline is found,
// then publishing each complete line as an SSE event (without the trailing newline).
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
