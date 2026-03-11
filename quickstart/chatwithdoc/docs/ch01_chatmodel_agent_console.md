---
title: "第一章：用 ChatModel 跑通第一个 ChatModelAgent（Console）"
---

本章目标：从 0 跑通一个最小可用的 Agent，理解 ADK 的基本执行模型：`ChatModel` → `ChatModelAgent` → `Runner` → `AgentEvent`（流式输出）。

本章是 self-contained 的：你只需要一个可用的模型配置，就能直接运行并看到 agent 的流式回复。

## 代码位置

- 入口代码：[cmd/ch01/main.go](file:///Users/bytedance/github/eino/examples/quickstart/chatwithdoc/cmd/ch01/main.go)

## 前置条件

- Go 版本：与本目录 `go.mod` 一致
- 一个可调用的 ChatModel（默认使用 OpenAI；也支持 Ark）

本 quickstart 复用 `github.com/cloudwego/eino-examples/adk/common/model.NewChatModel()` 来创建模型，它会读取环境变量选择不同 provider：

### 方式 A：OpenAI（默认）

需要配置（最小集合）：

- `OPENAI_API_KEY`
- `OPENAI_MODEL`

可选：

- `OPENAI_BASE_URL`（例如走代理或兼容 OpenAI 协议的服务）
- `OPENAI_BY_AZURE=true`（如使用 Azure OpenAI）

示例：

```bash
export OPENAI_API_KEY="..."
export OPENAI_MODEL="gpt-4.1-mini"
```

### 方式 B：Ark

需要配置：

- `MODEL_TYPE=ark`
- `ARK_API_KEY`
- `ARK_MODEL`

可选：

- `ARK_BASE_URL`

示例：

```bash
export MODEL_TYPE="ark"
export ARK_API_KEY="..."
export ARK_MODEL="..."
```

## 运行

在本章里我们用 Console 直接运行，不启动 Web 服务。

```bash
cd /Users/bytedance/github/eino/examples/quickstart/chatwithdoc
go run ./cmd/ch01 -- "用一句话解释 Eino ADK 是什么？"
```

你会看到类似输出（流式逐步打印）：

```text
[assistant] ...
```

## 本章学到什么

- ChatModel 是什么：负责“把一串 messages 变成一条 assistant reply”，并可选择支持 tool calling / streaming。
- ChatModelAgent 是什么：一个围绕 ChatModel 的“对话循环器”，负责组织 prompt（instruction + history），并把模型输出封装成 `AgentEvent`。
- Runner 是什么：执行 Agent 的统一入口，提供 streaming / checkpoint（本章不启用 checkpoint）。
- AgentEvent 是什么：Runner 的输出流。你可以把它当作“把 agent 执行过程翻译成事件”，用于 Console/Web UI/日志等。

## 本章刻意不做的事（下一章再引入）

- 不做多轮对话记忆（memory）：本章只跑通单轮，先建立最小闭环。
- 不引入 tools：工具会改变 agent 的控制流（tool call / tool result），更适合在跑通闭环之后再加。
