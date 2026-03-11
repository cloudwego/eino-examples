---
title: "第一章：ChatModel 与 Message（Console）"
---

本章目标：理解 Eino 的 Component 抽象，用最小代码调用一次 ChatModel（支持流式输出），并掌握 `schema.Message` 的基本用法。

## 代码位置

- 入口代码：[cmd/ch01/main.go](https://github.com/cloudwego/eino/blob/main/examples/quickstart/chatwitheino/cmd/ch01/main.go)

## 为什么需要 Component 接口

Eino 定义了一组 Component 接口（`ChatModel`、`Tool`、`Retriever`、`Loader` 等），每个接口描述一类可替换的能力：

```go
type BaseChatModel interface {
    Generate(ctx context.Context, input []*schema.Message, opts ...Option) (*schema.Message, error)
    Stream(ctx context.Context, input []*schema.Message, opts ...Option) (
        *schema.StreamReader[*schema.Message], error)
}
```

**接口带来的好处：**

1. **实现可替换**：`eino-ext` 提供了 OpenAI、Ark、Claude、Ollama 等多种实现，业务代码只依赖接口，切换模型只需改构造逻辑。
2. **编排可组合**：Agent、Graph、Chain 等编排层只依赖 Component 接口，不关心具体实现。你可以把 OpenAI 换成 Ark，编排代码无需改动。
3. **测试可 Mock**：接口天然支持 mock，单元测试不需要真实调用模型。

本章只涉及 `ChatModel`，后续章节会逐步引入 `Tool`、`Retriever` 等 Component。

## schema.Message：对话的基本单位

`Message` 是 Eino 里对话数据的基本结构：

```go
type Message struct {
    Role      RoleType    // system / user / assistant / tool
    Content   string      // 文本内容
    ToolCalls []ToolCall  // 仅 assistant 消息可能有
    // ...
}
```

常用构造函数：

```go
schema.SystemMessage("You are a helpful assistant.")
schema.UserMessage("What is the weather today?")
schema.AssistantMessage("I don't know.", nil)  // 第二个参数是 ToolCalls
schema.ToolMessage("tool result", "call_id")
```

**角色语义：**
- `system`：系统指令，通常放在 messages 最前面
- `user`：用户输入
- `assistant`：模型回复
- `tool`：工具调用结果（后续章节涉及）

## 前置条件

- Go 版本：与本目录 `go.mod` 一致
- 一个可调用的 ChatModel（默认使用 OpenAI；也支持 Ark）

### 方式 A：OpenAI（默认）

```bash
export OPENAI_API_KEY="..."
export OPENAI_MODEL="gpt-4.1-mini"
# 可选：
# OPENAI_BASE_URL（代理或兼容服务）
# OPENAI_BY_AZURE=true（使用 Azure OpenAI）
```

### 方式 B：Ark

```bash
export MODEL_TYPE="ark"
export ARK_API_KEY="..."
export ARK_MODEL="..."
# 可选：ARK_BASE_URL
```

## 运行

在 `examples/quickstart/chatwitheino` 目录下执行：

```bash
go run ./cmd/ch01 -- "用一句话解释 Eino 的 Component 设计解决了什么问题？"
```

输出示例（流式逐步打印）：

```text
[assistant] Eino 的 Component 设计通过定义统一接口...
```

## 入口代码做了什么

按执行顺序：

1. **创建 ChatModel**：根据 `MODEL_TYPE` 环境变量选择 OpenAI 或 Ark 实现
2. **构造输入 messages**：`SystemMessage(instruction)` + `UserMessage(query)`
3. **调用 Stream**：所有 ChatModel 实现都必须支持 `Stream()`，返回 `StreamReader[*Message]`
4. **打印结果**：迭代 `StreamReader` 逐帧打印 assistant 回复

关键代码片段（**注意：这是简化后的代码片段，不能直接运行，完整代码请参考** [cmd/ch01/main.go](https://github.com/cloudwego/eino/blob/main/examples/quickstart/chatwitheino/cmd/ch01/main.go)）：

```go
// 构造输入
messages := []*schema.Message{
    schema.SystemMessage(instruction),
    schema.UserMessage(query),
}

// 调用 Stream（所有 ChatModel 都必须实现）
stream, err := cm.Stream(ctx, messages)
if err != nil {
    log.Fatal(err)
}
defer stream.Close()

for {
    chunk, err := stream.Recv()
    if errors.Is(err, io.EOF) {
        break
    }
    if err != nil {
        log.Fatal(err)
    }
    fmt.Print(chunk.Content)
}
```

## 本章小结

- **Component 接口**：定义可替换、可组合、可测试的能力边界
- **Message**：对话数据的基本单位，通过角色区分语义
- **ChatModel**：最基础的 Component，提供 `Generate` 和 `Stream` 两个核心方法
- **实现选择**：通过环境变量或配置切换 OpenAI/Ark 等不同实现，业务代码无需改动
