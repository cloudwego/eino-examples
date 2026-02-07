# Example 2: Multi-Source Research with compose.Graph + Streaming

This example demonstrates using `StreamableGraphTool` with `compose.Graph` to create a research tool that queries multiple sources in parallel and streams the synthesized results.

## What This Example Shows

- Using `compose.Graph` with edge-based connections
- Wrapping a graph as a `StreamableTool`
- Parallel execution within a graph node
- Streaming output with `ReturnDirectly`
- ChatModel integration for result synthesis

## Architecture

```
Research Query
      │
      ▼
┌─────────────────────────────────────┐
│         parallel_search             │
│  ┌─────────┬─────────┬──────────┐   │
│  │   Web   │   KB    │  Local   │   │  ← Parallel goroutines
│  │ Search  │ Search  │  Search  │   │
│  └────┬────┴────┬────┴────┬─────┘   │
│       └─────────┼─────────┘         │
└─────────────────┼───────────────────┘
                  │
                  ▼
┌─────────────────────────────────────┐
│       prepare_prompt_input          │  ← Format for template
└─────────────────┬───────────────────┘
                  │
                  ▼
┌─────────────────────────────────────┐
│          prepare_prompt             │  ← ChatTemplate node
└─────────────────┬───────────────────┘
                  │
                  ▼
┌─────────────────────────────────────┐
│           synthesize                │  ← ChatModel (streams output)
└─────────────────┬───────────────────┘
                  │
                  ▼
        Streaming Response
```

## Key Components

### Parallel Search Implementation

```go
graph.AddLambdaNode("parallel_search", compose.InvokableLambda(func(ctx context.Context, input *ResearchInput) (*searchResults, error) {
    resultCh := make(chan result, 3)
    
    // Launch parallel searches
    go func() { /* web search */ }()
    go func() { /* KB search */ }()
    go func() { /* local search */ }()
    
    // Collect results
    for i := 0; i < 3; i++ {
        r := <-resultCh
        // aggregate results
    }
    return results, nil
}))
```

### Graph Edge Connections

```go
graph.AddEdge(compose.START, "parallel_search")
graph.AddEdge("parallel_search", "prepare_prompt_input")
graph.AddEdge("prepare_prompt_input", "prepare_prompt")
graph.AddEdge("prepare_prompt", "synthesize")
graph.AddEdge("synthesize", compose.END)
```

### Streaming Tool Creation

```go
tool, err := graphtool.NewStreamableGraphTool[*ResearchInput, *schema.Message](
    graph,
    "research_topic",
    "Research a topic by querying multiple sources...",
)
```

### ReturnDirectly Configuration

```go
agent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
    ToolsConfig: adk.ToolsConfig{
        ToolsNodeConfig: compose.ToolsNodeConfig{
            Tools: []tool.BaseTool{researchTool},
        },
        ReturnDirectly: map[string]bool{
            "research_topic": true,  // Stream directly to user
        },
    },
})
```

## Running the Example

```bash
# Set your OpenAI API key
export OPENAI_API_KEY=your-api-key

# Run the example
go run main.go
```

## Expected Output

```
=== Multi-Source Research Example (using compose.Graph + StreamableGraphTool) ===

This example demonstrates:
1. StreamableGraphTool with compose.Graph
2. Parallel search execution within a graph node
3. Streaming output from ChatModel via ReturnDirectly

  [Graph] Starting parallel searches...
  [Graph] Local file search completed
  [Graph] Knowledge base search completed
  [Graph] Web search completed
  [Graph] All searches completed, preparing synthesis...

{"role":"assistant","content":"Based"...}
{"role":"assistant","content":" on"...}
{"role":"assistant","content":" the"...}
... (streaming chunks)
```

## Key Takeaways

1. **Graph for Complex Flows**: `compose.Graph` allows flexible node connections via edges
2. **Parallel Execution**: Use goroutines within a node for concurrent operations
3. **Streaming Output**: `StreamableGraphTool` + `ReturnDirectly` enables real-time streaming to users
4. **Message Output**: Output `*schema.Message` directly for proper streaming chunk handling
