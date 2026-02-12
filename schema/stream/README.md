# Eino Streaming Programming Examples

This directory contains examples demonstrating Eino's streaming programming capabilities.

## Streaming Paradigms

Eino supports four streaming paradigms based on input/output stream types:

| Lambda Type | Input | Output | Mode | Use Case |
|-------------|-------|--------|------|----------|
| `InvokableLambda` | Non-stream | Non-stream | Ping-Pong | Simple request-response |
| `StreamableLambda` | Non-stream | Stream | Server-Streaming | LLM token generation |
| `CollectableLambda` | Stream | Non-stream | Client-Streaming | Aggregating stream data |
| `TransformableLambda` | Stream | Stream | Bidirectional-Streaming | Real-time processing |

## Examples

| Directory | Description |
|-----------|-------------|
| [1_invoke](./1_invoke) | Basic Invoke pattern - Text summarizer |
| [2_stream](./2_stream) | Stream output - Word-by-word text generator |
| [3_collect](./3_collect) | Collect stream input - Log aggregator |
| [4_transform](./4_transform) | Transform stream - Real-time text processor |
| [5_auto_streaming](./5_auto_streaming) | Auto streaming/concat between nodes |
| [6_stream_reader_utils](./6_stream_reader_utils) | StreamReader utilities (Pipe, Convert, etc.) |
| [7_merge_stream_readers](./7_merge_stream_readers) | Merge multiple streams into one |

## Running Examples

```bash
cd schema/stream/<example_dir>
go run .
```

## Key Concepts

### StreamReader and StreamWriter

```go
sr, sw := schema.Pipe[string](bufferSize)

go func() {
    defer sw.Close()
    sw.Send("chunk1", nil)
    sw.Send("chunk2", nil)
}()

for {
    chunk, err := sr.Recv()
    if err == io.EOF {
        break
    }
    // process chunk
}
```

### Auto Streaming/Concat

When orchestrating nodes with mismatched stream types, Eino automatically:
- **Streaming**: Converts `T` to single-chunk `StreamReader[T]`
- **Concat**: Merges `StreamReader[T]` into complete `T`

## Documentation

- [Eino Streaming Programming Essentials](https://www.cloudwego.io/zh/docs/eino/core_modules/chain_and_graph_orchestration/stream_programming_essentials/)
