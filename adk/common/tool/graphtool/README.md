# GraphTool Package

This package provides utilities for wrapping Eino's composition types (`compose.Graph`, `compose.Chain`, `compose.Workflow`) as agent tools. It enables you to expose complex multi-step processing pipelines as single tools that can be used by `ChatModelAgent`.

## Overview

The package provides two main tool types:

| Tool Type | Interface | Use Case |
|-----------|-----------|----------|
| `InvokableGraphTool` | `tool.InvokableTool` | Standard request-response tools |
| `StreamableGraphTool` | `tool.StreamableTool` | Tools that stream output incrementally |

Both tools support:
- Any `Compilable` type (`compose.Graph`, `compose.Chain`, `compose.Workflow`)
- Interrupt/Resume for human-in-the-loop workflows
- Checkpoint-based state persistence

## Installation

```go
import "github.com/cloudwego/eino-examples/adk/common/tool/graphtool"
```

## Quick Start

### InvokableGraphTool

Wrap a composition as a standard invokable tool:

```go
// Define input/output types
type MyInput struct {
    Query string `json:"query" jsonschema_description:"The query to process"`
}

type MyOutput struct {
    Result string `json:"result"`
}

// Create a chain/graph/workflow
chain := compose.NewChain[*MyInput, *MyOutput]()
chain.AppendLambda(compose.InvokableLambda(func(ctx context.Context, input *MyInput) (*MyOutput, error) {
    return &MyOutput{Result: "Processed: " + input.Query}, nil
}))

// Wrap as tool
tool, err := graphtool.NewInvokableGraphTool[*MyInput, *MyOutput](
    chain,
    "my_tool",
    "Description of what this tool does",
)
```

### StreamableGraphTool

Wrap a composition as a streaming tool (useful when the final node streams output):

```go
// Graph that outputs streaming messages
graph := compose.NewGraph[*MyInput, *schema.Message]()
graph.AddChatModelNode("llm", chatModel)
// ... add edges ...

// Wrap as streaming tool
tool, err := graphtool.NewStreamableGraphTool[*MyInput, *schema.Message](
    graph,
    "streaming_tool",
    "A tool that streams its response",
)

// Use with ReturnDirectly for direct streaming to user
agent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
    // ...
    ToolsConfig: adk.ToolsConfig{
        ToolsNodeConfig: compose.ToolsNodeConfig{
            Tools: []tool.BaseTool{tool},
        },
        ReturnDirectly: map[string]bool{
            "streaming_tool": true,
        },
    },
})
```

## Compilable Interface

Both tool types accept any type implementing the `Compilable` interface:

```go
type Compilable[I, O any] interface {
    Compile(ctx context.Context, opts ...compose.GraphCompileOption) (compose.Runnable[I, O], error)
}
```

This includes:
- `compose.Graph[I, O]`
- `compose.Chain[I, O]`
- `compose.Workflow[I, O]`

## Interrupt/Resume Support

GraphTools fully support Eino's interrupt/resume mechanism for human-in-the-loop workflows:

```go
// Inside a workflow node
if needsApproval {
    return nil, compose.StatefulInterrupt(ctx, &ApprovalInfo{
        Message: "Approval required",
    }, currentState)
}
```

The tool automatically:
1. Captures checkpoint state when interrupted
2. Wraps the interrupt with `CompositeInterrupt` for proper propagation
3. Restores state and resumes execution when `runner.ResumeWithParams` is called

### Composable Tool Wrappers

GraphTools implement standard `tool.InvokableTool` or `tool.StreamableTool` interfaces, making them compatible with any tool wrapper in the ecosystem. Examples of wrappers you can use:

- **`InvokableApprovableTool`**: Adds human approval before tool execution
- **`InvokableReviewEditTool`**: Allows users to review and edit tool arguments
- **`FollowUpTool`**: Asks users follow-up questions during execution
- Custom wrappers you create

### Nested Interrupts

When a GraphTool with internal interrupts is wrapped by another interrupt-based wrapper (e.g., `InvokableApprovableTool`), both interrupt layers work independently:

1. **Outer interrupt**: Wrapper-level interrupt (e.g., approval via `InvokableApprovableTool`)
2. **Inner interrupt**: Workflow-level interrupt (via `StatefulInterrupt` inside graph nodes)

This works because each layer uses distinct interrupt state types, preventing conflicts.

## Tool Options

Pass compose options to the underlying runnable:

```go
result, err := tool.InvokableRun(ctx, argsJSON, 
    graphtool.WithGraphToolOption(
        compose.WithCallbacks(myCallback),
    ),
)
```

## Examples

See the [examples](./examples) directory for complete working examples:

| Example | Description |
|---------|-------------|
| [1_chain_summarize](./examples/1_chain_summarize) | Document summarization using `compose.Chain` |
| [2_graph_research](./examples/2_graph_research) | Multi-source research with `compose.Graph` + streaming |
| [3_workflow_order](./examples/3_workflow_order) | Order processing with `compose.Workflow` + approval |
| [4_nested_interrupt](./examples/4_nested_interrupt) | Nested interrupts (outer approval + inner risk check) |

## API Reference

### NewInvokableGraphTool

```go
func NewInvokableGraphTool[I, O any](
    compilable Compilable[I, O],
    name, desc string,
    opts ...compose.GraphCompileOption,
) (*InvokableGraphTool[I, O], error)
```

Creates a new invokable tool from a compilable composition.

### NewStreamableGraphTool

```go
func NewStreamableGraphTool[I, O any](
    compilable Compilable[I, O],
    name, desc string,
    opts ...compose.GraphCompileOption,
) (*StreamableGraphTool[I, O], error)
```

Creates a new streaming tool from a compilable composition.

### WithGraphToolOption

```go
func WithGraphToolOption(opts ...compose.Option) tool.Option
```

Wraps compose options as tool options for passing to `InvokableRun` or `StreamableRun`.
