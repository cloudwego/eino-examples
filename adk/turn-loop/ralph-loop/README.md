# Ralph Loop

This example demonstrates the **Ralph Loop** pattern using the ADK's `TurnLoop` API, wrapped in a reusable `RalphLoop` abstraction.

## What is a Ralph Loop?

The Ralph Loop (originated by [Geoffrey Huntley](https://github.com/snarktank/ralph)) is an autonomous agent iteration pattern:

1. A **single task prompt** is fed to an AI agent **repeatedly**
2. Each turn, the agent gets a **fresh context window** but discovers prior work via the filesystem
3. The agent's output is inspected for a **completion promise** (`<COMPLETE/>`)
4. A **verification gate** rejects premature completion — the caller decides what "done" means
5. If rejected or no promise → the prompt is re-injected for another turn
6. A **max turns** limit provides a safety bound

## The `RalphLoop` Abstraction

The example provides a `RalphLoop` type that encapsulates the pattern, hiding the raw `TurnLoop` mechanics:

```go
agent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
    Name:        "ralph",
    Instruction: "You are an autonomous AI developer...",
    Model:       chatModel,
    Handlers:    []adk.ChatModelAgentMiddleware{fsMw},
})

rl := NewRalphLoop(RalphLoopConfig{
    Agent:             agent,
    Prompt:            taskPrompt,
    MaxTurns:          10,
    CompletionPromise: "<COMPLETE/>",
    VerifyCompletion: func(ctx context.Context) error {
        // Return nil to accept, error to reject and continue.
        bugs, _ := backend.GrepRaw(ctx, &filesystem.GrepRequest{
            Path: "/project", Pattern: "BUG:",
        })
        if len(bugs) > 0 {
            return fmt.Errorf("%d BUG markers remaining", len(bugs))
        }
        return nil
    },
    OnEvent: func(event *adk.AgentEvent) {
        // Observability hook — print events, log metrics, etc.
        prints.Event(event)
    },
})

result := rl.Run(ctx) // blocking
fmt.Printf("Complete: %v, Turns: %d/%d\n", result.Complete, result.Turns, result.MaxTurns)
```

### `RalphLoopConfig`

| Field | Description |
|---|---|
| `Agent` | The `adk.Agent` to run each turn. Built by the caller with desired tools, middleware, retry config. |
| `Prompt` | Task description re-injected every turn. |
| `MaxTurns` | Safety bound — forced stop when reached. |
| `CompletionPromise` | String the agent must output to signal completion. Defaults to `<COMPLETE/>`. |
| `VerifyCompletion` | Called when the promise is detected. Return `nil` to accept, `error` to reject and continue. Optional. |
| `OnEvent` | Called for each agent event during a turn (observability). Optional. |
| `IdleTimeout` | Safety net — auto-stop if the loop is idle for this long. Defaults to 30s. |

### `RalphLoopResult`

| Field | Description |
|---|---|
| `Complete` | `true` if the agent's completion promise was accepted. |
| `Turns` | Number of turns executed. |
| `MaxTurns` | Configured maximum. |
| `StopCause` | Reason the loop stopped (e.g. `"completion promise accepted"`, `"max turns reached"`). |
| `Err` | Non-nil if the loop exited due to an error. |

## How It Maps to TurnLoop

| Ralph Concept | TurnLoop Implementation (inside `RalphLoop`) |
|---|---|
| Task prompt | Internal item pushed into the loop |
| Same prompt each turn | `GenInput` always re-injects the prompt; `OnAgentEvents` re-pushes the item if incomplete |
| Completion detection | `OnAgentEvents` inspects output for `CompletionPromise` |
| Verification gate | `VerifyCompletion` callback — caller decides what "done" means |
| External state as memory | Shared `InMemoryBackend` — file writes persist across turns |
| Fresh context per turn | `PrepareAgent` returns the same agent; TurnLoop feeds fresh input each turn |
| Max turns | Counter in `GenInput`; calls `Stop()` when exceeded |

## Architecture

```
┌──────────────────────────────────────────────────────┐
│                   RalphLoop.Run()                    │
│                                                      │
│  ┌──────────┐    ┌──────────────┐    ┌────────────┐  │
│  │ GenInput  │───▶│PrepareAgent  │───▶│   Agent    │  │
│  │(same      │    │(returns the  │    │ (runs turn,│  │
│  │ prompt)   │    │ shared agent)│    │  uses fs   │  │
│  └──────────┘    └──────────────┘    │  tools)    │  │
│       ▲                              └─────┬──────┘  │
│       │          ┌─────────────────┐       │         │
│       │          │ OnAgentEvents   │◀──────┘         │
│       │          └──┬──────────────┘                 │
│       │             │                                │
│       │   CompletionPromise detected?                │
│       │     YES ─▶ VerifyCompletion()                │
│       │              ├─ error → REJECT, re-push      │
│       │              └─ nil   → ACCEPT, Stop()       │
│       │     NO  ─▶ re-push ───────────────────┐      │
│       │                                       │      │
│       └───────────────────────────────────────┘      │
│                                                      │
│  ┌────────────────────────────────────────────────┐  │
│  │          InMemoryBackend (shared)              │  │
│  │    Files persist across all turns              │  │
│  │    Pre-seeded with buggy starter project       │  │
│  └────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────┘
```

## The Task

The agent is given a buggy URL shortener project pre-seeded in the in-memory filesystem.
The project has **15 intentional bugs** across 5 files (marked with `// BUG:` comments).
The agent must iteratively:

1. Read files and find remaining `BUG:` markers
2. Fix 2-3 bugs per turn using `edit_file`
3. Remove the corresponding `BUG:` comments
4. Verify fixes by re-reading the edited files
5. Only declare `<COMPLETE/>` when all `BUG:` markers are gone

The `VerifyCompletion` gate ensures the agent can't declare victory prematurely.

## Environment Variables

Configure your LLM provider:

**OpenAI (default):**
```bash
export OPENAI_API_KEY="sk-..."
export OPENAI_MODEL="gpt-4o"
export OPENAI_BASE_URL="https://api.openai.com/v1"
```

**Ark (Volcengine):**
```bash
export MODEL_TYPE="ark"
export ARK_API_KEY="..."
export ARK_MODEL="..."
export ARK_BASE_URL="..."
```

## Run

```bash
cd examples
go run ./adk/turn-loop/ralph-loop/
```

## Expected Behavior

1. Turn 1: Agent reads all files, finds 15 `BUG:` markers, fixes 2-3 in store.go
2. Turn 2-4: Agent continues fixing handler.go, handler_test.go, main.go, README.md
3. When agent outputs `<COMPLETE/>`, the verification gate greps for remaining BUG markers
4. If markers remain → promise rejected, task re-pushed for another turn
5. When all markers are resolved → promise accepted, loop exits
6. Final summary shows stop cause, turn count, and all files in the filesystem
