# Agentic Research Assistant

[中文说明](./README.zh-CN.md)

## Why AgenticMessage ✨

In a single model call, the model may return multiple ordered structured events, such as reasoning, calling a server tool, continuing reasoning, and then calling a function tool. `AgenticMessage` uses `ContentBlock`s to store these ordered structured events.

## Project ✨

This example shows how to build and run a compact Eino ADK agent with `schema.AgenticMessage`.

The agent prepares an evidence-backed research report for an engineering team. It combines an `AgenticModel`, server-side `web_search`, local function tools, Eino's native filesystem middleware, and streaming ADK events in one runnable flow.

The console output prints Eino's built-in `AgenticMessage.String()` representation.

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

The terminal prints each materialized `AgenticMessage`. The exact output depends on the model and search results, but the ordered `content_blocks` can show an agentic sequence such as reasoning, server-side search, more reasoning, and then local function calls:

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

The generated report is written to:

```text
adk/agentic/research_assistant/workspace/research_report.md
```
