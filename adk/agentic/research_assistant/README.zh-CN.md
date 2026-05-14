# Agentic Research Assistant

[English README](./README.md)

## AgenticMessage 的优势 ✨

随着模型厂商原生支持 reasoning、内置工具、MCP 风格工具调用和更丰富的响应元信息，模型输出已经不再只是“一段文本答案”。一次模型调用可能是一条有序的 agentic 时间线：先思考，再调用服务端工具，再继续 reasoning，然后调用本地函数工具，读取结果，最后生成回答。

`schema.Message` 仍然很适合经典聊天场景：role、文本内容、多模态 part，以及本地 function tool call。`schema.AgenticMessage` 则更适合需要保留复杂 agentic 时间线的场景。它的 `ContentBlocks` 会把 reasoning、用户输入、助手生成内容、function tool call/result、server tool call/result、MCP 相关内容都保留为专门的 block 类型。`AgenticResponseMeta` 承载 token usage 和 provider extension，block `Extra` 和 extension 字段也可以保留 provider-specific 的 item ID、状态等信息。

在 Eino 里，`AgenticMessage` 是 `AgenticModel`、`AgenticChatTemplate` 和 `AgenticToolsNode` 共同围绕的消息形态。这个示例走的是 ADK 路径：typed agent 运行 `AgenticModel`，使用服务端 `web_search`，在 agent loop 中执行本地 Eino 工具，接入 typed middleware，并输出 streaming 事件。使用 `AgenticMessage` 后，输出能直接、结构化地展示 Agent 运行中发生了什么，更适合 UI 展示、日志、tracing 和调试。

## 项目简介 ✨

这个示例展示如何用 `schema.AgenticMessage` 构建并运行一个精简的 Eino ADK Agent。

Agent 会为工程团队生成一份证据型研究报告。运行过程中，它会把 `AgenticModel`、服务端 `web_search`、本地函数工具、typed middleware 和 streaming ADK events 串成一个完整的可运行流程。

终端输出直接使用 Eino 官方 `AgenticMessage.String()`，让你看到一次 Agent 运行被表达为结构化 blocks，而不是一段普通文本。

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
