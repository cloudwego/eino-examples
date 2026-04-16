# Ralph Loop

This example demonstrates the **Ralph Loop** pattern using the ADK's `TurnLoop` API.

## What is a Ralph Loop?

The Ralph Loop (originated by [Geoffrey Huntley](https://github.com/snarktank/ralph)) is an autonomous agent iteration pattern:

1. A **single task prompt** is fed to an AI agent **repeatedly**
2. Each turn, the agent gets a **fresh context window** but discovers prior work via the filesystem
3. A **stop hook** inspects the agent's output for a **completion promise** (`<COMPLETE/>`)
4. A **verification gate** rejects premature completion — it greps the project files for remaining `BUG:` markers and only accepts the promise when all markers are resolved
5. If rejected or no promise → the task is re-injected for another turn
6. A **max turns** limit provides a safety bound

## How TurnLoop Maps to Ralph

| Ralph Concept | TurnLoop Implementation |
|---|---|
| Task prompt | `*Task` item pushed into the loop |
| Same prompt each turn | `GenInput` always consumes the single item; `OnAgentEvents` re-pushes it if incomplete |
| Stop hook (check for promise) | `OnAgentEvents` inspects output for `<COMPLETE/>` |
| Verification gate | `OnAgentEvents` greps for `BUG:` markers before accepting the completion promise |
| External state as memory | Shared `InMemoryBackend` — file writes persist across turns |
| Fresh context per turn | `PrepareAgent` creates a new `ChatModelAgent` each turn |
| Max turns | Counter in `GenInput`; calls `Stop()` when exceeded |
| Tool error resilience | `ToolCallMiddleware` catches tool errors and returns them as string results |
| Rate limit handling | `ModelRetryConfig` with exponential backoff retries transient API errors |

## Architecture

```
┌──────────────────────────────────────────────────────┐
│                    TurnLoop                          │
│                                                      │
│  ┌──────────┐    ┌──────────────┐    ┌────────────┐  │
│  │ GenInput  │───▶│ PrepareAgent │───▶│   Agent    │  │
│  │(same      │    │(fresh agent, │    │ (runs turn,│  │
│  │ prompt)   │    │ shared fs)   │    │  uses fs   │  │
│  └──────────┘    └──────────────┘    │  tools)    │  │
│       ▲                              └─────┬──────┘  │
│       │          ┌─────────────────┐       │         │
│       │          │ OnAgentEvents   │◀──────┘         │
│       │          │ (stop hook)     │                 │
│       │          └──┬──────────────┘                 │
│       │             │                                │
│       │   <COMPLETE/>?                               │
│       │     YES ─▶ grep BUG: markers                 │
│       │              ├─ found → REJECT, Push(task)   │
│       │              └─ none  → ACCEPT, Stop()       │
│       │     NO  ─▶ Push(task) ───────────────────┐   │
│       │                                          │   │
│       └──────────────────────────────────────────┘   │
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

The verification gate ensures the agent can't declare victory prematurely.

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
