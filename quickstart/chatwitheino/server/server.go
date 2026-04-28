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
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	hserver "github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/google/uuid"
	"github.com/hertz-contrib/sse"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"

	commontool "github.com/cloudwego/eino-examples/adk/common/tool"
	"github.com/cloudwego/eino-examples/quickstart/chatwitheino/a2ui"
	"github.com/cloudwego/eino-examples/quickstart/chatwitheino/mem"
)

func init() {
	schema.RegisterName[ChatItem]("chatwitheino_chat_item")
	schema.RegisterName[commontool.ApprovalResult]("chatwitheino_approval_result")
}

// ChatItem is the item type for TurnLoop. Each user query or approval decision
// is pushed as a ChatItem.
type ChatItem struct {
	Query          string                     // user message text (empty for approval items)
	ApprovalResult *commontool.ApprovalResult // non-nil when this item carries an approval decision
	InterruptID    string                     // which interrupt this approval resolves
}

// errInterrupted is returned by OnAgentEvents when the agent is interrupted
// for approval. The TurnLoop exits with this as ExitReason.
var errInterrupted = errors.New("agent interrupted for approval")

// Config holds all dependencies for the HTTP server.
type Config struct {
	Agent           adk.Agent
	CheckPointStore adk.CheckPointStore
	Store           *mem.Store
	WorkspaceDir    string
	ProjectRoot     string // root of the codebase the agent can explore
	ExamplesDir     string // root of the eino-examples repo (for example searches)
	Port            string
}

// Server wraps a Hertz HTTP server with the chat-with-doc routes.
type Server struct {
	cfg        Config
	turnStates sync.Map // sessionID → *sessionTurnState
}

// New creates a Server from the given config.
func New(cfg Config) *Server {
	return &Server{cfg: cfg}
}

// iterEnvelope carries the event iterator from OnAgentEvents to the HTTP handler.
// The done channel is included so the handler always sends results back to the
// correct OnAgentEvents invocation, even if a preempt replaces the session channels.
type iterEnvelope struct {
	events  *adk.AsyncIterator[*adk.AgentEvent]
	history []*schema.Message
	done    chan iterResult
}

// iterResult carries the outcome from the HTTP handler back to OnAgentEvents.
type iterResult struct {
	lastContent   string
	intermediates []*schema.Message // tool call + tool result messages to persist
	interruptID   string
	msgIdx        int
	err           error
}

// sessionTurnState holds the TurnLoop and event bridge channels for a session.
type sessionTurnState struct {
	mu          sync.Mutex
	loop        *adk.TurnLoop[*ChatItem, *schema.Message]
	iterReady   chan iterEnvelope // OnAgentEvents → HTTP handler
	iterDone    chan iterResult   // HTTP handler → OnAgentEvents
	handlerDone chan struct{}     // closed to tell a prev handler to bail on preempt
}

func (s *Server) getTurnState(sessionID string) *sessionTurnState {
	val, _ := s.turnStates.LoadOrStore(sessionID, &sessionTurnState{})
	return val.(*sessionTurnState)
}

// startLoopCleanup spawns a goroutine that waits for the loop to exit
// (e.g. due to an error or all items consumed) and nils out ts.loop so
// the next handleChat creates a fresh loop instead of trying to preempt
// a dead one.
func (s *Server) startLoopCleanup(ts *sessionTurnState, loop *adk.TurnLoop[*ChatItem, *schema.Message], sessionID string) {
	go func() {
		result := loop.Wait()
		ts.mu.Lock()
		if ts.loop == loop {
			ts.loop = nil
		}
		ts.mu.Unlock()
		if result.ExitReason != nil {
			log.Printf("[loop] session=%s exited with error: %v", sessionID, result.ExitReason)
		} else {
			log.Printf("[loop] session=%s exited cleanly", sessionID)
		}
	}()
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
		// Stop any running loop for this session.
		ts := s.getTurnState(id)
		ts.mu.Lock()
		if ts.loop != nil {
			ts.loop.Stop(adk.WithImmediate())
			ts.loop = nil
		}
		ts.mu.Unlock()

		if err := s.cfg.Store.Delete(id); err != nil {
			c.JSON(consts.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		s.turnStates.Delete(id)
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

	h.POST("/sessions/:id/abort", func(ctx context.Context, c *app.RequestContext) {
		s.handleAbort(ctx, c)
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

// handleChat handles a new chat message. It creates or reuses a TurnLoop for the session.
// If a loop is already running (busy), it pushes with preempt to cancel the current turn.
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

	item := &ChatItem{Query: req.Message}

	ts := s.getTurnState(id)

	// Each handler gets its own local iterReady channel reference and a
	// handlerDone channel. This avoids races when multiple preempts replace
	// the channels on ts concurrently.
	var localIterReady chan iterEnvelope
	var localHandlerDone chan struct{}

	ts.mu.Lock()
	if ts.loop != nil {
		// Loop exists — try to push with preempt (AfterToolCalls).
		loop := ts.loop
		log.Printf("[chat] session=%s preempting current turn", id)
		// Signal any previous handler waiting on iterReady to bail.
		if ts.handlerDone != nil {
			close(ts.handlerDone)
		}
		ts.iterReady = make(chan iterEnvelope, 1)
		ts.iterDone = make(chan iterResult, 1)
		ts.handlerDone = make(chan struct{})
		localIterReady = ts.iterReady
		localHandlerDone = ts.handlerDone
		ts.mu.Unlock()
		ok, _ := loop.Push(item, adk.WithPreempt[*ChatItem, *schema.Message](adk.AfterToolCalls))
		if !ok {
			// Loop already stopped (e.g. error on previous turn) — create new one.
			log.Printf("[chat] session=%s loop was dead, creating new loop", id)
			ts.mu.Lock()
			loop = s.newLoop(sess, id, false)
			ts.loop = loop
			ts.iterReady = make(chan iterEnvelope, 1)
			ts.iterDone = make(chan iterResult, 1)
			ts.handlerDone = make(chan struct{})
			localIterReady = ts.iterReady
			localHandlerDone = ts.handlerDone
			ts.mu.Unlock()
			loop.Push(item)
			loop.Run(context.Background())
			s.startLoopCleanup(ts, loop, id)
		}
	} else {
		// No loop — create a new one.
		loop := s.newLoop(sess, id, false)
		ts.loop = loop
		ts.iterReady = make(chan iterEnvelope, 1)
		ts.iterDone = make(chan iterResult, 1)
		ts.handlerDone = make(chan struct{})
		localIterReady = ts.iterReady
		localHandlerDone = ts.handlerDone
		ts.mu.Unlock()
		loop.Push(item)
		loop.Run(context.Background())
		s.startLoopCleanup(ts, loop, id)
	}

	// User message is persisted in GenInput (not here) to guarantee correct
	// session history ordering: the preempted turn's intermediates are persisted
	// by OnAgentEvents before GenInput fires for the new turn.

	// Open SSE stream and start keepalives BEFORE waiting for the iterator.
	// During a preempt the old turn may take tens of seconds to drain; if we
	// don't write anything the browser/TCP stack may consider the connection
	// dead, causing all subsequent writes to fail silently.
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

	// Wait for OnAgentEvents to send us the iterator. Use local channel
	// references so a concurrent preempt replacing ts.iterReady doesn't
	// orphan us on a stale channel.
	var envelope iterEnvelope
	select {
	case envelope = <-localIterReady:
	case <-localHandlerDone:
		// Another preempt took over — our turn was superseded.
		close(kaStop)
		log.Printf("[chat] session=%s handler superseded by newer preempt", id)
		_ = stream.Publish(&sse.Event{Data: []byte(`{"event":"preempted"}`)})
		return
	case <-time.After(60 * time.Second):
		close(kaStop)
		// Stream is already open; send an error event instead of JSON.
		_ = stream.Publish(&sse.Event{Data: []byte(`{"error":"agent did not start in time"}`)})
		return
	}

	lastContent, intermediates, interruptID, finalMsgIdx, streamErr := a2ui.StreamToWriter(
		&sseLineWriter{stream: stream}, id, envelope.history, envelope.events,
	)
	close(kaStop)

	// Send result back to the SAME OnAgentEvents that sent us this envelope.
	envelope.done <- iterResult{
		lastContent:   lastContent,
		intermediates: intermediates,
		interruptID:   interruptID,
		msgIdx:        finalMsgIdx,
		err:           streamErr,
	}

	if streamErr != nil {
		log.Printf("[chat] session=%s stream error: %v", id, streamErr)
	} else if interruptID != "" {
		log.Printf("[chat] session=%s interrupted: id=%s", id, interruptID)
	} else {
		log.Printf("[chat] session=%s done, response=%d chars", id, len(lastContent))
	}
}

// handleApprove resumes an interrupted agent run with the user's approval decision.
// Creates a new TurnLoop with checkpoint/resume to continue from the interrupt.
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
	result := &commontool.ApprovalResult{Approved: req.Approved, DisapproveReason: reason}

	// Clear the pending interrupt so a double-approve returns 400.
	sess.SetPendingInterruptID("")

	log.Printf("[approve] session=%s interruptID=%s approved=%v", id, interruptID, req.Approved)

	// Create a new loop with checkpoint resume.
	ts := s.getTurnState(id)
	ts.mu.Lock()
	// Clear any old loop.
	if ts.loop != nil {
		ts.loop.Stop(adk.WithImmediate())
	}
	// Signal any previous handler to bail.
	if ts.handlerDone != nil {
		close(ts.handlerDone)
	}
	loop := s.newLoop(sess, id, true)
	ts.loop = loop
	ts.iterReady = make(chan iterEnvelope, 1)
	ts.iterDone = make(chan iterResult, 1)
	ts.handlerDone = make(chan struct{})
	localIterReady := ts.iterReady
	localHandlerDone := ts.handlerDone
	ts.mu.Unlock()

	// Push the approval item before starting.
	loop.Push(&ChatItem{
		ApprovalResult: result,
		InterruptID:    interruptID,
	})
	loop.Run(context.Background())
	s.startLoopCleanup(ts, loop, id)

	// Open SSE stream and start keepalives before waiting.
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

	// Wait for OnAgentEvents to send us the iterator.
	var envelope iterEnvelope
	select {
	case envelope = <-localIterReady:
	case <-localHandlerDone:
		close(kaStop)
		log.Printf("[approve] session=%s handler superseded by newer request", id)
		_ = stream.Publish(&sse.Event{Data: []byte(`{"event":"preempted"}`)})
		return
	case <-time.After(60 * time.Second):
		close(kaStop)
		_ = stream.Publish(&sse.Event{Data: []byte(`{"error":"agent did not start in time"}`)})
		return
	}
	_ = envelope.history // not used for StreamContinue

	lastContent, newInterruptID, finalMsgIdx, streamErr := a2ui.StreamContinue(
		&sseLineWriter{stream: stream}, id, sess.GetMsgIdx(), envelope.events,
	)
	close(kaStop)

	// Send result back to the SAME OnAgentEvents that sent us this envelope.
	envelope.done <- iterResult{
		lastContent: lastContent,
		interruptID: newInterruptID,
		msgIdx:      finalMsgIdx,
		err:         streamErr,
	}

	if streamErr != nil {
		log.Printf("[approve] session=%s stream error: %v", id, streamErr)
	} else if newInterruptID != "" {
		log.Printf("[approve] session=%s re-interrupted: id=%s", id, newInterruptID)
	} else {
		log.Printf("[approve] session=%s done, response=%d chars", id, len(lastContent))
	}
}

// handleAbort immediately stops the current TurnLoop for a session.
func (s *Server) handleAbort(_ context.Context, c *app.RequestContext) {
	id := c.Param("id")

	ts := s.getTurnState(id)
	ts.mu.Lock()
	loop := ts.loop
	ts.loop = nil
	ts.mu.Unlock()

	if loop == nil {
		c.JSON(consts.StatusOK, map[string]string{"status": "no active loop"})
		return
	}

	log.Printf("[abort] session=%s stopping loop immediately", id)
	loop.Stop(adk.WithImmediate())
	loop.Wait()
	log.Printf("[abort] session=%s loop stopped", id)

	c.JSON(consts.StatusOK, map[string]string{"status": "aborted"})
}

// newLoop creates a new TurnLoop for the session. If withResume is true,
// the loop is configured with a checkpoint store and GenResume for interrupt resume.
func (s *Server) newLoop(sess *mem.Session, sessionID string, withResume bool) *adk.TurnLoop[*ChatItem, *schema.Message] {
	cfg := adk.TurnLoopConfig[*ChatItem, *schema.Message]{
		GenInput:      s.makeGenInput(sess, sessionID),
		PrepareAgent:  s.makePrepareAgent(),
		OnAgentEvents: s.makeOnAgentEvents(sess, sessionID),
	}
	if withResume {
		cfg.Store = s.cfg.CheckPointStore
		cfg.CheckpointID = sessionID
		cfg.GenResume = s.makeGenResume()
	}
	return adk.NewTurnLoop(cfg)
}

// makeGenInput returns the GenInput callback. It builds agent messages from
// session history + workspace context.
func (s *Server) makeGenInput(sess *mem.Session, sessionID string) func(ctx context.Context, loop *adk.TurnLoop[*ChatItem, *schema.Message], items []*ChatItem) (*adk.GenInputResult[*ChatItem, *schema.Message], error) {
	return func(ctx context.Context, loop *adk.TurnLoop[*ChatItem, *schema.Message], items []*ChatItem) (*adk.GenInputResult[*ChatItem, *schema.Message], error) {
		// Find the first item with a query.
		var consumed []*ChatItem
		var remaining []*ChatItem
		var queryItem *ChatItem
		for _, item := range items {
			if queryItem == nil && item.Query != "" {
				queryItem = item
				consumed = append(consumed, item)
			} else {
				remaining = append(remaining, item)
			}
		}
		if queryItem == nil {
			// No query items — stop the loop.
			loop.Stop(adk.WithStopCause("no query items"))
			return &adk.GenInputResult[*ChatItem, *schema.Message]{
				Input:     &adk.AgentInput{Messages: []adk.Message{schema.UserMessage("done")}},
				Remaining: items,
			}, nil
		}

		// Persist the user message NOW — GenInput fires only after any previous
		// turn's OnAgentEvents has finished persisting its intermediates, so the
		// session history order is guaranteed correct.
		userMsg := schema.UserMessage(queryItem.Query)
		if appendErr := sess.Append(userMsg); appendErr != nil {
			log.Printf("warn: failed to persist user message: %v", appendErr)
		}

		history := sess.GetMessages()
		runMessages := s.buildRunMessages(sessionID, history)

		log.Printf("[genInput] session=%s query=%q messages=%d", sessionID, queryItem.Query, len(runMessages))

		return &adk.GenInputResult[*ChatItem, *schema.Message]{
			Input: &adk.AgentInput{
				Messages:        runMessages,
				EnableStreaming: true,
			},
			Consumed:  consumed,
			Remaining: remaining,
		}, nil
	}
}

// makePrepareAgent returns the PrepareAgent callback — returns the same agent.
func (s *Server) makePrepareAgent() func(ctx context.Context, loop *adk.TurnLoop[*ChatItem, *schema.Message], consumed []*ChatItem) (adk.Agent, error) {
	return func(ctx context.Context, loop *adk.TurnLoop[*ChatItem, *schema.Message], consumed []*ChatItem) (adk.Agent, error) {
		return s.cfg.Agent, nil
	}
}

// makeOnAgentEvents returns the OnAgentEvents callback — the bridge between
// the TurnLoop and the HTTP handler.
func (s *Server) makeOnAgentEvents(sess *mem.Session, sessionID string) func(ctx context.Context, tc *adk.TurnContext[*ChatItem, *schema.Message], events *adk.AsyncIterator[*adk.AgentEvent]) error {
	return func(ctx context.Context, tc *adk.TurnContext[*ChatItem, *schema.Message], events *adk.AsyncIterator[*adk.AgentEvent]) error {
		ts := s.getTurnState(sessionID)

		history := sess.GetMessages()

		// Snapshot bridge channels under lock to avoid races with handleChat
		// which may recreate them for a preempt.
		ts.mu.Lock()
		ready := ts.iterReady
		done := ts.iterDone
		ts.mu.Unlock()

		// Send the iterator to the HTTP handler. Include the done channel
		// so the handler replies to THIS invocation, not a future one.
		select {
		case ready <- iterEnvelope{events: events, history: history, done: done}:
		case <-ctx.Done():
			return ctx.Err()
		}

		// Wait for the HTTP handler to finish draining. Also select on ctx.Done
		// to avoid hanging when a preempt supersedes the handler — in that case
		// the old handler bails via handlerDone and nobody sends to our done channel.
		var result iterResult
		select {
		case result = <-done:
		case <-ctx.Done():
			return ctx.Err()
		}

		// Persist all intermediate messages (assistant text+tool calls, tool results).
		// The intermediates already include the final assistant text message if any,
		// so we don't need to persist lastContent separately.
		for _, msg := range result.intermediates {
			if appendErr := sess.Append(msg); appendErr != nil {
				log.Printf("warn: failed to persist intermediate message: %v", appendErr)
			}
		}
		if result.interruptID != "" {
			sess.SetPendingInterruptID(result.interruptID)
			sess.SetMsgIdx(result.msgIdx)
			return errInterrupted
		}
		return result.err
	}
}

// makeGenResume returns the GenResume callback for interrupt/resume.
func (s *Server) makeGenResume() func(ctx context.Context, loop *adk.TurnLoop[*ChatItem, *schema.Message], canceledItems, unhandledItems, newItems []*ChatItem) (*adk.GenResumeResult[*ChatItem, *schema.Message], error) {
	return func(ctx context.Context, loop *adk.TurnLoop[*ChatItem, *schema.Message], canceledItems, unhandledItems, newItems []*ChatItem) (*adk.GenResumeResult[*ChatItem, *schema.Message], error) {
		// Find the approval item in newItems.
		var approvalItem *ChatItem
		for _, item := range newItems {
			if item.ApprovalResult != nil {
				approvalItem = item
				break
			}
		}
		if approvalItem == nil {
			return nil, errors.New("no approval item found for resume")
		}

		return &adk.GenResumeResult[*ChatItem, *schema.Message]{
			ResumeParams: &adk.ResumeParams{
				Targets: map[string]any{approvalItem.InterruptID: approvalItem.ApprovalResult},
			},
			Consumed:  canceledItems,
			Remaining: unhandledItems,
		}, nil
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
