# Agentic Research Assistant

[中文说明](./README.zh-CN.md)

## Why AgenticMessage ✨

As model providers add native reasoning, built-in tools, MCP-style tool use, and richer response metadata, model output is no longer just "one text answer". A single turn can become an ordered agentic timeline: think, call a server-side tool, continue reasoning, call a local function, read the result, and then answer.

`schema.Message` is still a good fit for classic chat: role, text content, multimodal parts, and local function tool calls. `schema.AgenticMessage` is better when you need to preserve that richer timeline. Its `ContentBlocks` keep reasoning, user input, assistant generated content, function tool calls/results, server tool calls/results, and MCP-related items as dedicated block types. `AgenticResponseMeta` carries token usage and provider extensions, while block `Extra` and extension fields can preserve provider-specific details such as item IDs or statuses.

In Eino, `AgenticMessage` is the shared message shape behind `AgenticModel`, `AgenticChatTemplate`, and `AgenticToolsNode`. This example uses the ADK path: a typed agent runs an `AgenticModel`, uses server-side `web_search`, executes local Eino tools through the agent loop, applies typed middleware, and streams structured events. With `AgenticMessage`, the output shows what happened during the agent run directly and structurally, which is better for UI display, logs, tracing, and debugging.

## Project ✨

This example shows how to build and run a compact Eino ADK agent with `schema.AgenticMessage`.

The agent prepares an evidence-backed research report for an engineering team. It combines an `AgenticModel`, server-side `web_search`, local function tools, typed middleware, and streaming ADK events in one runnable flow.

The console output prints Eino's built-in `AgenticMessage.String()` representation, so you can see an agent turn as structured blocks instead of plain text.

## Run 🚀

```bash
export ARK_API_KEY="your-ark-api-key"
export ARK_MODEL_ID="your-ark-model-id"

go run ./adk/agentic/research_assistant
```

Optional:

```bash
export ARK_BASE_URL="your-ark-base-url"
```

## Output 👀

The terminal prints each materialized `AgenticMessage`. The exact output depends on the model and search results, but the ordered `content_blocks` can show an agentic sequence such as reasoning, server-side search, more reasoning, and then a local function call:

```text
--- AgenticMessage #2 ---
role: assistant
content_blocks:
  [0] type: reasoning
      text: ...
  [1] type: server_tool_call
      name: web_search
      arguments: ...
  [2] type: reasoning
      text: ...
  [3] type: function_tool_call
      call_id: ...
      name: score_evidence
      arguments: ...
response_meta:
  token_usage: prompt=..., completion=..., total=...

--- AgenticMessage #3 ---
role: user
content_blocks:
  [0] type: function_tool_result
      call_id: ...
      name: score_evidence
      content: (1 blocks)
        [0] text: ...
```

When `save_research_report` is invoked, the middleware also prints:

```text
[middleware] evidence_gate running before save_research_report
```

The generated report is written to:

```text
adk/agentic/research_assistant/workspace/research_report.md
```
