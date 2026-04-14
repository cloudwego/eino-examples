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
	"os"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino-ext/adk/backend/local"
	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/middlewares/filesystem"
	"github.com/cloudwego/eino/adk/middlewares/subagent"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/cloudwego/eino-examples/adk/common/store"
	"github.com/cloudwego/eino-examples/adk/common/trace"
)

func main() {
	ctx := context.Background()

	// Setup CozeLoop tracing
	closeFn, startSpanFn := trace.AppendCozeLoopCallbackIfConfigured(ctx)
	defer closeFn(ctx)
	ctx, endSpan := startSpanFn(ctx, "subagent-example", nil)
	defer endSpan(ctx, nil)

	// Create Ark ChatModel
	cm, err := ark.NewChatModel(ctx, &ark.ChatModelConfig{
		APIKey: os.Getenv("ARK_API_KEY"),
		Model:  os.Getenv("ARK_MODEL"),
	})
	if err != nil {
		log.Fatalf("Failed to create Ark ChatModel: %v", err)
	}

	// Create local filesystem backend for tools
	be, err := local.NewBackend(ctx, &local.Config{})
	if err != nil {
		log.Fatalf("Failed to create local backend: %v", err)
	}

	// Create filesystem middleware (provides read_file, write_file, edit_file, glob, grep, execute)
	fsm, err := filesystem.New(ctx, &filesystem.MiddlewareConfig{
		Backend:        be,
		StreamingShell: be,
	})
	if err != nil {
		log.Fatalf("Failed to create filesystem middleware: %v", err)
	}

	// Shared handlers for subagents
	handlers := []adk.ChatModelAgentMiddleware{fsm}

	// Create Explore and Plan subagents
	toolsConfig := adk.ToolsConfig{} // filesystem tools injected via middleware
	exploreAgent, err := newExploreAgent(ctx, cm, toolsConfig, handlers)
	if err != nil {
		log.Fatalf("Failed to create explore agent: %v", err)
	}
	planAgent, err := newPlanAgent(ctx, cm, toolsConfig, handlers)
	if err != nil {
		log.Fatalf("Failed to create plan agent: %v", err)
	}

	// Create TaskMgr for background execution
	taskMgr := subagent.NewTaskMgr(ctx, &subagent.TaskMgrConfig{}) // default: 120s auto-background

	// Create subagent middleware with TaskMgr
	subagentMW, err := subagent.New(ctx, &subagent.Config{
		SubAgents: []adk.Agent{exploreAgent, planAgent},
		TaskMgr:   taskMgr,
	})
	if err != nil {
		log.Fatalf("Failed to create subagent middleware: %v", err)
	}

	// Create main agent with filesystem + subagent middleware
	mainAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "MainAgent",
		Description: "A helpful assistant with background exploration and planning capabilities.",
		Instruction: `You are a helpful assistant with access to filesystem tools and background subagents.

You have two specialized subagents available:
- "explore": Use for codebase exploration, file search, and code analysis. Runs in background.
- "plan": Use for architecture design and implementation planning. Runs in background.

When the user asks you to explore code or plan something, use the agent tool with run_in_background=true
to delegate to the appropriate subagent. Continue working on other tasks while subagents run in background.

Use the task_output tool to check on completed background tasks and incorporate their results.`,
		Model: cm,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{},
		},
		Handlers: []adk.ChatModelAgentMiddleware{fsm, subagentMW},
	})
	if err != nil {
		log.Fatalf("Failed to create main agent: %v", err)
	}

	// Create runner
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		EnableStreaming: true,
		Agent:           mainAgent,
		CheckPointStore: store.NewInMemoryStore(),
	})

	// Create the app model that wraps TUI and handles queries
	app := newAppModel(ctx, runner, taskMgr)

	p := tea.NewProgram(app, tea.WithAltScreen())
	app.setProgram(p)

	// Subscribe to TaskMgr notifications for subagent panel
	notifyCh := taskMgr.Subscribe()
	go func() {
		for notification := range notifyCh {
			task := notification.Task
			p.Send(subAgentNotificationMsg{task: task})

			// If the task is running and has events, drain them in a goroutine
			if task.Status == subagent.StatusRunning && notification.Events != nil {
				go drainSubAgentEvents(task.ID, notification.Events, p)
			}
		}
	}()

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}

	// Cleanup
	_ = taskMgr.Close(ctx)

	time.Sleep(20 * time.Second)
}

// appModel wraps tuiModel and handles query execution with multi-turn message history.
type appModel struct {
	tui     tuiModel
	runner  *adk.Runner
	ctx     context.Context
	taskMgr *subagent.TaskMgr
	cpSeq   int

	// messageHistory accumulates the full conversation across turns.
	// Each turn appends assistant messages/tool calls/tool results from agent events.
	// When subagents complete, their results are appended as a user message to trigger
	// the next turn.
	messageHistory []*schema.Message

	mu      sync.Mutex
	program *tea.Program
}

func newAppModel(ctx context.Context, runner *adk.Runner, taskMgr *subagent.TaskMgr) *appModel {
	return &appModel{
		tui:     newTUIModel(),
		runner:  runner,
		ctx:     ctx,
		taskMgr: taskMgr,
	}
}

func (a *appModel) setProgram(p *tea.Program) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.program = p
}

func (a *appModel) getProgram() *tea.Program {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.program
}

func (a *appModel) Init() tea.Cmd {
	return a.tui.Init()
}

func (a *appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case userQueryMsg:
		// Append user message to history
		a.messageHistory = append(a.messageHistory, schema.UserMessage(msg.query))

		// Launch the multi-turn loop in a background goroutine
		return a, func() tea.Msg {
			a.runMultiTurnLoop()
			return turnCompleteMsg{}
		}
	}

	// Delegate to inner TUI model
	newModel, cmd := a.tui.Update(msg)
	if tm, ok := newModel.(tuiModel); ok {
		a.tui = tm
	}
	return a, cmd
}

func (a *appModel) View() string {
	return a.tui.View()
}

// runMultiTurnLoop runs the main agent, waits for subagents, and triggers new turns
// until there are no more pending subagent results.
func (a *appModel) runMultiTurnLoop() {
	p := a.getProgram()

	for {
		// Run one turn of the main agent with the full message history
		a.cpSeq++
		cpID := fmt.Sprintf("cp-%d", a.cpSeq)

		iter := a.runner.Run(a.ctx, a.messageHistory, adk.WithCheckPointID(cpID))

		var lastErr error
		for {
			event, ok := iter.Next()
			if !ok {
				break
			}
			if event.Err != nil {
				lastErr = event.Err
				continue
			}
			// processAgentEvent returns *schema.Message for history accumulation
			if m := processAgentEvent(event, p); m != nil {
				a.messageHistory = append(a.messageHistory, m)
			}
		}

		// Notify TUI that this turn is done
		p.Send(queryDoneMsg{err: lastErr})

		// Check if there are running subagent tasks
		if !a.taskMgr.HasRunning() {
			// No subagents running, we're done
			return
		}

		// Wait for all subagent tasks to complete
		p.Send(mainEventMsg{entry: logEntry{
			Type:      "info",
			Content:   "Waiting for background subagents to complete...",
			Timestamp: time.Now(),
		}})

		if err := a.taskMgr.WaitAllDone(a.ctx); err != nil {
			p.Send(errMsg{err: fmt.Errorf("error waiting for subagents: %w", err)})
			return
		}

		// Collect completed subagent results
		summary := a.collectSubAgentResults()
		if summary == "" {
			// No meaningful results to feed back
			return
		}

		// Notify TUI
		p.Send(subAgentsDoneMsg{summary: summary})

		// Append subagent results as a user message to trigger next turn
		feedbackMsg := schema.UserMessage(fmt.Sprintf(
			"[System] Background subagent tasks have completed. Here are their results:\n\n%s\n\nPlease incorporate these results into your response to the user.",
			summary,
		))
		a.messageHistory = append(a.messageHistory, feedbackMsg)

		// Loop will continue to run another turn with the updated history
	}
}

// collectSubAgentResults gathers results from all completed tasks and formats them.
func (a *appModel) collectSubAgentResults() string {
	tasks := a.taskMgr.List()
	var parts []string
	for _, t := range tasks {
		if t.Status == subagent.StatusCompleted && t.Result != "" && !t.ResultQueried {
			parts = append(parts, fmt.Sprintf("### Task [%s]: %s\nResult:\n%s", t.ID, t.Description, t.Result))
			a.taskMgr.MarkQueried(t.ID)
		}
		if t.Status == subagent.StatusFailed && t.Error != "" && !t.ResultQueried {
			parts = append(parts, fmt.Sprintf("### Task [%s]: %s\nFailed: %s", t.ID, t.Description, t.Error))
			a.taskMgr.MarkQueried(t.ID)
		}
	}
	return strings.Join(parts, "\n\n")
}

// drainSubAgentEvents reads all events from a subagent's event iterator
// and sends them to the TUI as subAgentEventMsg.
func drainSubAgentEvents(taskID string, events *adk.AsyncIterator[*adk.AgentEvent], p *tea.Program) {
	for {
		event, ok := events.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			p.Send(subAgentEventMsg{taskID: taskID, entry: logEntry{
				Type:      "error",
				Content:   event.Err.Error(),
				Timestamp: time.Now(),
			}})
			continue
		}
		processSubAgentEvent(event, taskID, p)
	}
}
