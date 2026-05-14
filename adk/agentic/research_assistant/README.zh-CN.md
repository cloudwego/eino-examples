# Agentic Research Assistant

[English README](./README.md)

## AgenticMessage 的优势 ✨

在一次模型调用中，模型可能会返回多个有序的结构化事件，例如先 reasoning，然后调用 server tool，再 reasoning，接着再调用 function tool 等。`AgenticMessage` 会用 `ContentBlock` 保存这些有序的结构化事件。

## 项目简介 ✨

这个示例展示如何用 `schema.AgenticMessage` 构建并运行一个精简的 Eino ADK Agent。

Agent 会为工程团队生成一份证据型研究报告。运行过程中，它会把 `AgenticModel`、服务端 `web_search`、本地函数工具、Eino 原生 filesystem middleware 和 streaming ADK events 串成一个完整的可运行流程。

终端输出直接使用 Eino 官方 `AgenticMessage.String()`，让你看到一次 Agent 运行被表达为结构化 blocks。

## 运行 🚀

```bash
export ARK_API_KEY="your-ark-api-key"
export ARK_MODEL_ID="your-ark-model-id"

go run ./adk/agentic/research_assistant
```

可选：

```bash
export ARK_BASE_URL="your-ark-base-url"
```

## 输出 👀

终端会打印 materialized 后的 `AgenticMessage`。实际输出会随模型和搜索结果变化，但有序的 `content_blocks` 可以展示 Agent 的执行顺序，例如先 reasoning，再服务端搜索，再继续 reasoning，最后发起本地函数调用：

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

生成的报告会写入：

```text
adk/agentic/research_assistant/workspace/research_report.md
```
