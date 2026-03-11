---
title: "第一章：ChatModel 与 Message（Console）"
---

本章目标：用最小代码调用一次 ChatModel（支持流式输出），并掌握 `schema.Message` 的基本用法。

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

## 关键概念（只讲本章用到的）

- `schema.Message`：对话消息。常见角色为 `system/user/assistant/tool`。本章使用 `system + user` 作为输入。
- `ChatModel`：模型组件，负责基于一组 messages 生成回复。示例中用 `model.NewChatModel()` 创建。

## 运行

在本章里我们用 Console 直接运行，不启动 Web 服务。

```bash
cd /Users/bytedance/github/eino/examples/quickstart/chatwithdoc
go run ./cmd/ch01 -- "用一句话解释 Eino 的 Component 设计解决了什么问题？"
```

你会看到类似输出（流式逐步打印）：

```text
[assistant] ...
```

## 入口代码做了什么

按执行顺序：

1. `NewChatModel()` 创建 ChatModel（从环境变量选择 OpenAI/Ark）
2. 构造输入 messages：`SystemMessage(instruction)` + `UserMessage(query)`
3. 优先尝试 `ChatModel.Stream(...)` 并打印流式输出；若不支持则回退到 `ChatModel.Generate(...)`
