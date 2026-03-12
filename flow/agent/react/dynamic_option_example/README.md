# Dynamic Option Modification for ReAct Agent

This example demonstrates how to dynamically modify `model.Option` during ReAct agent execution. The key challenge is that options passed to `agent.Generate()` or `agent.Stream()` are fixed at call time, but we may want to change options based on the current iteration, previous tool calls, or other runtime conditions.

## Problem

When calling a ReAct agent, the option list is passed once and applied to all ChatModel calls during the loop:

```go
agent.Invoke(ctx, messages, opts...)  // opts are fixed for all iterations
```

However, you may want to:
- Enable/disable extended thinking based on iteration
- Change `tool_choice` to force a final answer after N iterations
- Modify tool bindings dynamically

## Solution

We solve this by:
1. **Wrapping the ChatModel** with a `dynamic.ChatModel` that intercepts `Generate()`/`Stream()` calls
2. **Using Graph State** via `compose.ProcessState` to persist iteration count across the ReAct loop
3. **Wrapping the ReAct Agent** in a parent Graph that provides the state
4. **Using MessageFuture** to observe all intermediate results (reasoning, tool calls, tool results)

### Architecture

```
┌─────────────────────────────────────────────────────────┐
│  Parent Graph (with dynamic.State)                      │
│  ┌───────────────────────────────────────────────────┐  │
│  │  ReAct Agent (as sub-graph node)                  │  │
│  │  ┌─────────────────────────────────────────────┐  │  │
│  │  │  dynamic.ChatModel                          │  │  │
│  │  │  ├─ Reads iteration from state              │  │  │
│  │  │  ├─ Calls GetOptionFunc(ctx, input, state)  │  │  │
│  │  │  ├─ Increments iteration in state           │  │  │
│  │  │  └─ Calls inner ChatModel with merged opts  │  │  │
│  │  └─────────────────────────────────────────────┘  │  │
│  └───────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
```

## Components

### State (`dynamic/state.go`)

Holds the iteration counter and optional data for decision making:

```go
type State struct {
    Iteration     int
    LastToolCalls []*schema.ToolCall
    CustomData    map[string]any
}

type OptionFunc func(ctx context.Context, input []*schema.Message, state *State) []model.Option
```

### ChatModel (`dynamic/model.go`)

Wraps any `model.BaseChatModel` and uses `compose.ProcessState` to access the graph state:

```go
type ChatModel struct {
    Model         model.BaseChatModel
    GetOptionFunc OptionFunc
}
```

## Usage

```go
// Wrap the ChatModel with dynamic option support
dynamicModel := &dynamic.ChatModel{
    Model:         arkChatModel,
    GetOptionFunc: getDynamicOptions,
}

// Create ReAct agent with the dynamic model
rAgent, _ := react.NewAgent(ctx, &react.AgentConfig{
    ToolCallingModel: dynamicModel,
    ToolsConfig:      toolsConfig,
})

// Create parent graph with local state
parentGraph := compose.NewGraph[[]*schema.Message, *schema.Message](
    compose.WithGenLocalState(func(ctx context.Context) *dynamic.State {
        return dynamic.NewState()
    }),
)

// Export and add ReAct agent as sub-graph
agentGraph, agentOpts := rAgent.ExportGraph()
_ = parentGraph.AddGraphNode("react_agent", agentGraph, agentOpts...)
_ = parentGraph.AddEdge(compose.START, "react_agent")
_ = parentGraph.AddEdge("react_agent", compose.END)

runnable, _ := parentGraph.Compile(ctx)

// Use MessageFuture to observe intermediate results
msgFutureOpt, msgFuture := react.WithMessageFuture()

go func() {
    // Process intermediate messages in a goroutine
    iter := msgFuture.GetMessageStreams()
    for {
        sr, ok, _ := iter.Next()
        if !ok {
            break
        }
        // Read and print messages...
    }
}()

// Use Invoke with DesignateNode to pass options to the sub-graph
runnable.Invoke(ctx, messages, agent.GetComposeOptions(msgFutureOpt)[0].DesignateNode("react_agent"))
```

## Example: Dynamic Option Function

```go
func getDynamicOptions(ctx context.Context, input []*schema.Message, state *dynamic.State) []model.Option {
    var opts []model.Option

    // Control thinking mode based on iteration
    if state.Iteration >= 1 {
        opts = append(opts, ark.WithThinking(&arkModel.Thinking{
            Type: arkModel.ThinkingTypeDisabled,
        }))
    }

    // Force final answer after first iteration
    if state.Iteration >= 1 {
        opts = append(opts, model.WithToolChoice(schema.ToolChoiceForbidden))
        opts = append(opts, model.WithTools([]*schema.ToolInfo{}))
    } else {
        opts = append(opts, model.WithToolChoice(schema.ToolChoiceAllowed))
        // Bind tools for first iteration
        opts = append(opts, model.WithTools(toolInfos))
    }

    return opts
}
```

## Observing Intermediate Results with MessageFuture

The `react.WithMessageFuture()` function returns an option and a `MessageFuture` interface that allows you to observe all intermediate messages during agent execution:

- **Reasoning Content**: The model's thinking process (`msg.ReasoningContent`)
- **Tool Calls**: Function calls made by the model (`msg.ToolCalls`)
- **Tool Results**: Results returned from tool execution (`msg.Role == schema.Tool`)
- **Assistant Messages**: Text responses from the model

**Note**: When using `Invoke` instead of `Stream`, you must use `DesignateNode` to pass the MessageFuture option to the correct sub-graph node:

```go
runnable.Invoke(ctx, messages, agent.GetComposeOptions(msgFutureOpt)[0].DesignateNode("react_agent"))
```

## Quick Start

Environment variables:
- `ARK_API_KEY`
- `ARK_MODEL_NAME`

Build and run:

```bash
cd flow/agent/react/dynamic_option_example
go build -o dynamic_option_example .
./dynamic_option_example
```

## Key Design Decisions

1. **Graph State over Context**: Context values don't propagate back from `Generate()`, so we use `compose.ProcessState` to persist state across iterations.

2. **Wrapper Pattern**: Following the decorator pattern used elsewhere in the codebase.

3. **Simple Function Type**: Using `OptionFunc` instead of an interface keeps the API simple and easy to understand.

4. **Parent Graph**: The ReAct agent is wrapped as a sub-graph node in a parent graph that provides the state.

5. **MessageFuture for Observability**: Using `react.WithMessageFuture()` to capture and display all intermediate results including reasoning, tool calls, and tool results.

6. **Invoke with DesignateNode**: When using `Invoke` instead of `Stream`, use `DesignateNode` to ensure options are passed to the correct sub-graph node.
