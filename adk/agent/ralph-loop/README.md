# Ralph Loop

This example demonstrates the **Ralph Loop** pattern using the ADK's `Runner` API, wrapped in a reusable `RalphLoop` abstraction.

## What is a Ralph Loop?

The Ralph Loop (originated by [Geoffrey Huntley](https://github.com/snarktank/ralph)) is an autonomous agent iteration pattern:

1. A **single task prompt** is fed to an AI agent **repeatedly**
2. Each turn, the agent gets a **fresh context window** but discovers prior work via the filesystem
3. The agent's output is inspected for a **completion promise** (`<COMPLETE/>`)
4. A **verification gate** rejects premature completion — the caller decides what "done" means
5. If rejected or no promise → the prompt is re-injected for another turn
6. A **max turns** limit provides a safety bound

## The `RalphLoop` Abstraction

The example provides a `RalphLoop` type that encapsulates the pattern. The turn loop is driven externally via a simple `for` loop — each turn uses `Runner` to execute the agent once:

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

### `RalphLoopResult`

| Field | Description |
|---|---|
| `Complete` | `true` if the agent's completion promise was accepted. |
| `Turns` | Number of turns executed. |
| `MaxTurns` | Configured maximum. |
| `StopCause` | Reason the loop stopped (e.g. `"completion promise accepted"`, `"max turns reached"`). |
| `Err` | Non-nil if the loop exited due to an error. |

## Architecture

```
┌──────────────────────────────────────────────────────┐
│                   RalphLoop.Run()                     │
│                                                      │
│  for turn := 1; turn <= MaxTurns; turn++ {           │
│                                                      │
│    ┌──────────────────────────────────────────────┐  │
│    │  Runner.Run(prompt)                          │  │
│    │    → Agent executes one turn using fs tools  │  │
│    │    → Collects text output                    │  │
│    └──────────────────────┬───────────────────────┘  │
│                           │                          │
│    CompletionPromise in output?                       │
│      YES → VerifyCompletion()                        │
│              ├─ error → REJECT, continue loop        │
│              └─ nil   → ACCEPT, return result        │
│      NO  → continue loop                            │
│                                                      │
│  }                                                   │
│  → max turns reached, return result                  │
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
go run ./adk/agent/ralph-loop/
```

## Expected Behavior

1. Turn 1: Agent reads all files, finds 15 `BUG:` markers, fixes 2-3 in store.go
2. Turn 2-4: Agent continues fixing handler.go, handler_test.go, main.go, README.md
3. When agent outputs `<COMPLETE/>`, the verification gate greps for remaining BUG markers
4. If markers remain → promise rejected, loop continues with another turn
5. When all markers are resolved → promise accepted, loop exits
6. Final summary shows stop cause, turn count, and all files in the filesystem
