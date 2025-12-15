# Short-Term Memory for ReAct Agent

This example demonstrates a minimal short-term memory for a `flow/react` agent:

1. Run the agent with a new input message list, get the assistant output.
2. Serialize and persist the original input messages plus the assistant output.
3. On the next run, restore the stored messages, append the new input, and continue the conversation.
4. Do not persist the system message; inject it at runtime via `MessageModifier`.
5. Storage options include an in-memory map and Redis (with optional in-memory `miniredis`).

## Where to Look

- `main.go` — minimal demo: two turns that share memory and a system prompt injected at runtime.
- `memory/store.go` — `MemoryStore` interface and Gob encode/decode helpers.
- `memory/inmem.go` — in-memory store.
- `memory/redis.go` — Redis-backed store and `NewMiniRedisClient()` for an embedded Redis server.

## System Prompt Handling

- Do not persist system messages.
- Use the agent hook `react.AgentConfig.MessageModifier` to prepend the system prompt at execution time:

```go
agent, _ := react.NewAgent(ctx, &react.AgentConfig{
  Model: model,
  MessageModifier: func(_ context.Context, input []*schema.Message) []*schema.Message {
    return append([]*schema.Message{schema.SystemMessage(sys)}, input...)
  },
})
```

## Serialization

- Messages are serialized using `encoding/gob`.
- Eino registers the necessary types, so no manual `gob.Register` is required here.

## Quick Start (OpenAI)

Environment variables:

- `OPENAI_API_KEY`
- `OPENAI_MODEL` (e.g., `gpt-4o-mini`)
- `OPENAI_BASE_URL` (optional for proxy endpoints)
- `OPENAI_BY_AZURE`

Build and run:

```bash
cd flow/agent/react/memory_example
go build -o memory_example main.go
./memory_example
```

Expected output:

- First run: prints assistant response; memory stores the turn.
- Second run: restores prior messages, appends the new input, and maintains context.

## Switch Storage Implementations

Use the in-memory store (default):

```go
store := memory.NewInMemoryStore()
```

Use Redis with in-memory `miniredis`:

```go
cli, closer, _ := memory.NewMiniRedisClient()
defer closer()
store := memory.NewRedisStore(cli)
```

Use Redis with a real server:

```go
cli := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
store := memory.NewRedisStore(cli)
```

## Minimal Flow

```go
sessionID := "session:demo"
prev, _ := store.Read(ctx, sessionID)
effective := append(prev, schema.UserMessage(userInput))
resp, _ := agent.Generate(ctx, effective)
_ = store.Write(ctx, sessionID, append(effective, resp))

hits, _ := store.Query(ctx, sessionID, "CloudWeGo", 3)
```

## Notes

- The example uses `Generate`. You can use `Stream` similarly and persist on `io.EOF`.
- Keep the memory window small to cap serialization size; this can be enforced by your store implementation.
