# Unknown Tools Handler for ReAct Agent

- Demonstrates `UnknownToolsHandler` in `compose.ToolsNodeConfig` when the model emits an unknown tool call.
- Mock ChatModel produces three turns: unknown tool call → correct tool call → final answer.
- Builds on the ReAct agent from the flow package.

## Rationale
- ReAct agents often rely on the model to select tool names from a provided list. In practice, models may hallucinate a tool name not registered with the `ToolsNode`.
- Instead of aborting the agent on such an error, the `UnknownToolsHandler` produces a clear, structured message that is fed back to the ChatModel as the tool result.
- This feedback informs the model that the tool name is invalid and encourages it to pick a valid tool in the next turn, improving robustness and convergence.
- The example shows: first turn emits an unknown tool call; the handler returns guidance; the second turn uses the correct tool; the final turn produces the answer.

## Run
- `cd flow/agent/react/unknown_tool_handler_example`
- `go run main.go`

## Expected
- Prints a handler message for the unknown tool name.
- Executes the `sum` tool on the second turn and returns `{"sum":3}`.
- Outputs the final assistant answer `3`.
