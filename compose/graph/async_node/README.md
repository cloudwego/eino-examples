# Async Lambda Node in Eino Graph

This example demonstrates an "async node" implemented as a normal lambda in an Eino graph.
It covers two realistic business scenarios:

- Report Generation (invokable): a long-running background job that produces a document URL.
- Live Transcription (streamable): a stream of tokens produced over time and converted via `StreamReaderWithConvert`.

## Files
- `service.go`: mocked services (`generateReport`, `transcribeLive`).
- `main.go`: graph wiring, lambda nodes, and run flows.

## How It Works
- The invokable lambda starts `generateReport` in a goroutine and blocks on a channel until completion or cancellation.
- The streamable lambda obtains a live `StreamReader[string]` and wraps it with `StreamReaderWithConvert` to transform tokens.

## Run
```bash
cd compose/graph/async_node
go run .
```

You will see:
- A report URL logged from the invokable path.
- A stream of uppercase tokens from the transcription path until `EOF`.

## Notes
- The services inject errors when inputs contain the word `error`, to showcase error propagation.
- Cancellation is respected via `context.Context`.
