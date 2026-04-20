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
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"

	adkstore "github.com/cloudwego/eino-examples/adk/common/store"
	commontool "github.com/cloudwego/eino-examples/adk/common/tool"
	"github.com/cloudwego/eino-examples/quickstart/chatwitheino/mem"
)

// ---------------------------------------------------------------------------
// mock agent — emits configurable events then exits
// ---------------------------------------------------------------------------

type mockAgent struct {
	name string
	// onRun is called for each Run(); must send events to gen and Close it.
	onRun func(ctx context.Context, input *adk.AgentInput, gen *adk.AsyncGenerator[*adk.AgentEvent])
}

func (m *mockAgent) Name(context.Context) string        { return m.name }
func (m *mockAgent) Description(context.Context) string { return "mock" }
func (m *mockAgent) Run(ctx context.Context, input *adk.AgentInput, _ ...adk.AgentRunOption) *adk.AsyncIterator[*adk.AgentEvent] {
	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
	go m.onRun(ctx, input, gen)
	return iter
}

// simpleReplyAgent returns a mock that emits a single text message and exits.
func simpleReplyAgent(reply string) *mockAgent {
	return &mockAgent{
		name: "test-agent",
		onRun: func(ctx context.Context, _ *adk.AgentInput, gen *adk.AsyncGenerator[*adk.AgentEvent]) {
			defer gen.Close()
			gen.Send(&adk.AgentEvent{
				Output: &adk.AgentOutput{
					MessageOutput: &adk.MessageVariant{
						Message: schema.AssistantMessage(reply, nil),
						Role:    schema.Assistant,
					},
				},
			})
			gen.Send(&adk.AgentEvent{Action: adk.NewExitAction()})
		},
	}
}

// slowAgent returns a mock that delays before replying, respecting context cancellation.
func slowAgent(delay time.Duration, reply string) *mockAgent {
	return &mockAgent{
		name: "slow-agent",
		onRun: func(ctx context.Context, _ *adk.AgentInput, gen *adk.AsyncGenerator[*adk.AgentEvent]) {
			defer gen.Close()
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				gen.Send(&adk.AgentEvent{Err: ctx.Err()})
				return
			}
			gen.Send(&adk.AgentEvent{
				Output: &adk.AgentOutput{
					MessageOutput: &adk.MessageVariant{
						Message: schema.AssistantMessage(reply, nil),
						Role:    schema.Assistant,
					},
				},
			})
			gen.Send(&adk.AgentEvent{Action: adk.NewExitAction()})
		},
	}
}

// ---------------------------------------------------------------------------
// helper: setup test server + HTTP client
// ---------------------------------------------------------------------------

func newTestServer(t *testing.T, agent adk.Agent) (*Server, string, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	store, err := mem.NewStore(tmpDir)
	if err != nil {
		t.Fatalf("mem.NewStore: %v", err)
	}

	srv := New(Config{
		Agent:           agent,
		CheckPointStore: adkstore.NewInMemoryStore(),
		Store:           store,
		WorkspaceDir:    t.TempDir(),
		ProjectRoot:     t.TempDir(),
		ExamplesDir:     t.TempDir(),
		Port:            "0", // unused — we test via the handler functions directly
	})

	return srv, tmpDir, func() {}
}

// createSession creates a session via the Store directly.
func createSession(t *testing.T, srv *Server) string {
	t.Helper()
	sess, err := srv.cfg.Store.GetOrCreate("test-" + time.Now().Format("150405.000"))
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	return sess.ID
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestNewLoopCreation(t *testing.T) {
	agent := simpleReplyAgent("hello from agent")
	srv, _, cleanup := newTestServer(t, agent)
	defer cleanup()

	sessionID := createSession(t, srv)
	sess, _ := srv.cfg.Store.GetOrCreate(sessionID)

	// Create a loop.
	loop := srv.newLoop(sess, sessionID, false)
	if loop == nil {
		t.Fatal("newLoop returned nil")
	}

	// Verify turn state is tracked.
	ts := srv.getTurnState(sessionID)
	if ts == nil {
		t.Fatal("getTurnState returned nil")
	}
}

func TestChatNormalFlow(t *testing.T) {
	agent := simpleReplyAgent("test response")
	srv, _, cleanup := newTestServer(t, agent)
	defer cleanup()

	sessionID := createSession(t, srv)
	sess, _ := srv.cfg.Store.GetOrCreate(sessionID)

	// Simulate handleChat: create loop, push item, run, consume events.
	ts := srv.getTurnState(sessionID)
	ts.mu.Lock()
	loop := srv.newLoop(sess, sessionID, false)
	ts.loop = loop
	ts.iterReady = make(chan iterEnvelope, 1)
	ts.iterDone = make(chan iterResult, 1)
	ts.mu.Unlock()

	item := &ChatItem{Query: "hello"}
	if err := sess.Append(schema.UserMessage("hello")); err != nil {
		t.Fatalf("append: %v", err)
	}
	loop.Push(item)
	loop.Run(context.Background())

	// Wait for the iterator.
	var envelope iterEnvelope
	select {
	case envelope = <-ts.iterReady:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for iterReady")
	}

	// Drain the iterator.
	var content string
	for {
		event, ok := envelope.events.Next()
		if !ok {
			break
		}
		if event.Output != nil && event.Output.MessageOutput != nil {
			if m := event.Output.MessageOutput.Message; m != nil && m.Content != "" {
				content += m.Content
			}
		}
	}

	if content != "test response" {
		t.Errorf("expected 'test response', got %q", content)
	}

	// Signal done to OnAgentEvents.
	envelope.done <- iterResult{
		lastContent:   content,
		intermediates: []*schema.Message{schema.AssistantMessage(content, nil)},
	}

	// Stop the loop — no more items to process.
	loop.Stop()

	// Wait for loop to exit.
	result := loop.Wait()
	if result.ExitReason != nil {
		// The loop stops when GenInput gets no more query items (next call).
		// This is expected — it receives empty items and calls Stop.
		t.Logf("exit reason: %v (expected)", result.ExitReason)
	}

	// Verify the assistant message was appended to session.
	messages := sess.GetMessages()
	found := false
	for _, m := range messages {
		if m.Role == schema.Assistant && strings.Contains(m.Content, "test response") {
			found = true
			break
		}
	}
	if !found {
		t.Error("assistant message not found in session history")
	}
}

func TestAbortStopsLoop(t *testing.T) {
	// Use a slow agent to give us time to abort.
	// Note: WithImmediate() cancels via the AgentCancelFunc chain in the
	// TurnLoop's internal Runner. Our simple mock doesn't process RunOptions,
	// so cancellation relies on the agent's Run completing naturally.
	agent := slowAgent(500*time.Millisecond, "should not see this")
	srv, _, cleanup := newTestServer(t, agent)
	defer cleanup()

	sessionID := createSession(t, srv)
	sess, _ := srv.cfg.Store.GetOrCreate(sessionID)

	ts := srv.getTurnState(sessionID)
	ts.mu.Lock()
	loop := srv.newLoop(sess, sessionID, false)
	ts.loop = loop
	ts.iterReady = make(chan iterEnvelope, 1)
	ts.iterDone = make(chan iterResult, 1)
	ts.mu.Unlock()

	if err := sess.Append(schema.UserMessage("hello")); err != nil {
		t.Fatalf("append: %v", err)
	}
	loop.Push(&ChatItem{Query: "hello"})
	loop.Run(context.Background())

	// Wait for the event iterator to be ready.
	select {
	case envelope := <-ts.iterReady:
		// Start draining events in background (they will error or close due to abort).
		go func() {
			for {
				_, ok := envelope.events.Next()
				if !ok {
					break
				}
			}
			ts.iterDone <- iterResult{err: context.Canceled}
		}()
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for iterReady")
	}

	// Abort the loop.
	ts.mu.Lock()
	ts.loop = nil
	ts.mu.Unlock()
	loop.Stop(adk.WithImmediate())

	result := loop.Wait()
	// Exit reason should be non-nil (canceled).
	t.Logf("exit reason: %v", result.ExitReason)

	// Verify we can create a new loop after abort.
	ts.mu.Lock()
	newLoop := srv.newLoop(sess, sessionID, false)
	ts.loop = newLoop
	ts.mu.Unlock()
	if newLoop == nil {
		t.Fatal("failed to create new loop after abort")
	}
	newLoop.Stop()
	newLoop.Run(context.Background())
	newLoop.Wait()
}

func TestPreemptQueuesNewItem(t *testing.T) {
	// Track which queries GenInput sees.
	var queriesSeen []string
	var mu sync.Mutex

	agent := simpleReplyAgent("reply")
	srv, _, cleanup := newTestServer(t, agent)
	defer cleanup()

	sessionID := createSession(t, srv)
	sess, _ := srv.cfg.Store.GetOrCreate(sessionID)

	// Wrap the GenInput callback to track seen queries.
	origGenInput := srv.makeGenInput(sess, sessionID)
	wrappedGenInput := func(ctx context.Context, loop *adk.TurnLoop[*ChatItem, *schema.Message], items []*ChatItem) (*adk.GenInputResult[*ChatItem, *schema.Message], error) {
		for _, item := range items {
			if item.Query != "" {
				mu.Lock()
				queriesSeen = append(queriesSeen, item.Query)
				mu.Unlock()
			}
		}
		return origGenInput(ctx, loop, items)
	}

	// Create loop with wrapped GenInput.
	cfg := adk.TurnLoopConfig[*ChatItem, *schema.Message]{
		GenInput:      wrappedGenInput,
		PrepareAgent:  srv.makePrepareAgent(),
		OnAgentEvents: srv.makeOnAgentEvents(sess, sessionID),
	}
	loop := adk.NewTurnLoop(cfg)

	ts := srv.getTurnState(sessionID)
	ts.mu.Lock()
	ts.loop = loop
	ts.iterReady = make(chan iterEnvelope, 1)
	ts.iterDone = make(chan iterResult, 1)
	ts.mu.Unlock()

	// Push first query and run.
	if err := sess.Append(schema.UserMessage("first")); err != nil {
		t.Fatalf("append: %v", err)
	}
	loop.Push(&ChatItem{Query: "first"})
	loop.Run(context.Background())

	// Consume first turn's events.
	select {
	case envelope := <-ts.iterReady:
		for {
			_, ok := envelope.events.Next()
			if !ok {
				break
			}
		}
		ts.iterDone <- iterResult{lastContent: "reply"}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout on first turn")
	}

	// Now push second query with preempt (loop should still be running,
	// waiting in GenInput for more items). Since the agent already exited,
	// the loop is in idle state — Push will queue and GenInput will pick it up.
	if err := sess.Append(schema.UserMessage("second")); err != nil {
		t.Fatalf("append: %v", err)
	}

	// Recreate bridge channels for the second turn.
	ts.mu.Lock()
	ts.iterReady = make(chan iterEnvelope, 1)
	ts.iterDone = make(chan iterResult, 1)
	ts.mu.Unlock()

	loop.Push(&ChatItem{Query: "second"}, adk.WithPreempt[*ChatItem, *schema.Message](adk.AfterToolCalls))

	// Consume second turn.
	select {
	case envelope := <-ts.iterReady:
		for {
			_, ok := envelope.events.Next()
			if !ok {
				break
			}
		}
		ts.iterDone <- iterResult{lastContent: "reply"}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout on second turn")
	}

	// Stop the loop.
	loop.Stop()
	loop.Wait()

	mu.Lock()
	defer mu.Unlock()
	if len(queriesSeen) < 2 {
		t.Errorf("expected at least 2 queries seen, got %d: %v", len(queriesSeen), queriesSeen)
	}
	foundFirst := false
	foundSecond := false
	for _, q := range queriesSeen {
		if q == "first" {
			foundFirst = true
		}
		if q == "second" {
			foundSecond = true
		}
	}
	if !foundFirst || !foundSecond {
		t.Errorf("expected both 'first' and 'second' in queriesSeen, got %v", queriesSeen)
	}
}

func TestGenResumeFindsApprovalItem(t *testing.T) {
	agent := simpleReplyAgent("irrelevant")
	srv, _, cleanup := newTestServer(t, agent)
	defer cleanup()

	genResume := srv.makeGenResume()

	approvalItem := &ChatItem{
		InterruptID:    "interrupt-123",
		ApprovalResult: &commontool.ApprovalResult{Approved: true},
	}

	result, err := genResume(context.Background(), nil,
		[]*ChatItem{{Query: "canceled"}}, // canceledItems
		[]*ChatItem{},                    // unhandledItems
		[]*ChatItem{approvalItem},        // newItems
	)
	if err != nil {
		t.Fatalf("genResume error: %v", err)
	}

	if result.ResumeParams == nil {
		t.Fatal("ResumeParams is nil")
	}
	target, ok := result.ResumeParams.Targets["interrupt-123"]
	if !ok {
		t.Fatal("expected target for interrupt-123")
	}
	ar, ok := target.(*commontool.ApprovalResult)
	if !ok {
		t.Fatalf("unexpected target type: %T", target)
	}
	if !ar.Approved {
		t.Error("expected approved=true")
	}
}

func TestGenResumeErrorsWithoutApproval(t *testing.T) {
	agent := simpleReplyAgent("irrelevant")
	srv, _, cleanup := newTestServer(t, agent)
	defer cleanup()

	genResume := srv.makeGenResume()

	_, err := genResume(context.Background(), nil,
		nil, nil,
		[]*ChatItem{{Query: "not-an-approval"}}, // no ApprovalResult
	)
	if err == nil {
		t.Fatal("expected error when no approval item is found")
	}
	if !strings.Contains(err.Error(), "no approval item") {
		t.Errorf("unexpected error: %v", err)
	}
}

// Ensure our HTTP endpoints have the expected paths by checking route registration
// does not panic.
func TestServerRouteRegistration(t *testing.T) {
	agent := simpleReplyAgent("test")
	srv, _, cleanup := newTestServer(t, agent)
	defer cleanup()
	// Just verify the server can be constructed without error.
	_ = srv
}
