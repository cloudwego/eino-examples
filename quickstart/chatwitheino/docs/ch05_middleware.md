---
title: "第五章：Middleware（中间件模式）"
---

本章目标：理解 Middleware 模式，实现 Tool 错误处理和 ChatModel 重试机制。

## 为什么需要 Middleware

第四章我们为 Agent 添加了 Tool 能力，让 Agent 能够访问文件系统。但在实际应用场景中，**Tool 报错或 ChatModel 报错是常见的现象**，例如：

- **Tool 报错**：文件不存在、参数错误、权限不足等
- **ChatModel 报错**：API 限流（429）、网络超时、服务不可用等

### 问题一：Tool 错误会中断整个流程

当 Tool 执行失败时，错误会直接传播到 Agent，导致整个对话中断：

```text
[tool call] read_file(file_path: "nonexistent.txt")
Error: open nonexistent.txt: no such file or directory
// 对话中断，用户需要重新开始
```

### 问题二：模型调用可能因限流失败

当模型 API 返回 429（Too Many Requests）错误时，整个对话也会中断：

```text
Error: rate limit exceeded (429)
// 对话中断
```

### 期望的行为

这些报错信息往往**不希望直接终止 Agent 流程**，而是希望把报错信息给到模型，由模型自动纠错进行下一轮。例如：

```text
[tool call] read_file(file_path: "nonexistent.txt")
[tool result] [tool error] open nonexistent.txt: no such file or directory
[assistant] 抱歉，文件不存在。让我先列出当前目录的文件...
[tool call] glob(pattern: "*")
```

### Middleware 的定位

**Middleware 模式**可以扩展 Tool 和 ChatModel 的行为，非常适合解决这个问题：

- **Middleware 是 Agent 的拦截器**：在调用前后插入自定义逻辑
- **Middleware 可处理错误**：将错误转换为模型可理解的格式
- **Middleware 可实现重试**：自动重试失败的操作
- **Middleware 可组合**：多个 Middleware 可以串联使用

**简单类比：**
- **Agent** = "业务逻辑"
- **Middleware** = "AOP 切面"（日志、重试、错误处理等横切关注点）

## 代码位置

- 入口代码：[cmd/ch05/main.go](https://github.com/cloudwego/eino-examples/blob/main/quickstart/chatwitheino/cmd/ch05/main.go)

## 前置条件

与第一章一致：需要配置一个可用的 ChatModel（OpenAI 或 Ark）。同时，需要与第四章一样设置 `PROJECT_ROOT`：

```bash
export PROJECT_ROOT=/path/to/eino  # Eino 核心库根目录
```

## 运行

在 `examples/quickstart/chatwitheino` 目录下执行：

```bash
# 设置项目根目录
export PROJECT_ROOT=/path/to/your/project

go run ./cmd/ch05
```

输出示例：

```text
you> 列出当前目录的文件
[assistant] 我来帮你列出文件...
[tool call] list_files(directory: ".")

you> 读取一个不存在的文件
[assistant] 尝试读取文件...
[tool call] read_file(file_path: "nonexistent.txt")
[tool result] [tool error] open nonexistent.txt: no such file or directory
[assistant] 抱歉，文件不存在...
```

## 关键概念

### Middleware 接口

`ChatModelAgentMiddleware` 是 Agent 的中间件接口：

```go
type ChatModelAgentMiddleware interface {
    // BeforeAgent is called before each agent run, allowing modification of
    // the agent's instruction and tools configuration.
    BeforeAgent(ctx context.Context, runCtx *ChatModelAgentContext) (context.Context, *ChatModelAgentContext, error)

    // BeforeModelRewriteState is called before each model invocation.
    // The returned state is persisted to the agent's internal state and passed to the model.
    BeforeModelRewriteState(ctx context.Context, state *ChatModelAgentState, mc *ModelContext) (context.Context, *ChatModelAgentState, error)

    // AfterModelRewriteState is called after each model invocation.
    // The input state includes the model's response as the last message.
    AfterModelRewriteState(ctx context.Context, state *ChatModelAgentState, mc *ModelContext) (context.Context, *ChatModelAgentState, error)

    // WrapInvokableToolCall wraps a tool's synchronous execution with custom behavior.
    // This method is only called for tools that implement InvokableTool.
    WrapInvokableToolCall(ctx context.Context, endpoint InvokableToolCallEndpoint, tCtx *ToolContext) (InvokableToolCallEndpoint, error)

    // WrapStreamableToolCall wraps a tool's streaming execution with custom behavior.
    // This method is only called for tools that implement StreamableTool.
    WrapStreamableToolCall(ctx context.Context, endpoint StreamableToolCallEndpoint, tCtx *ToolContext) (StreamableToolCallEndpoint, error)

    // WrapEnhancedInvokableToolCall wraps an enhanced tool's synchronous execution.
    // This method is only called for tools that implement EnhancedInvokableTool.
    WrapEnhancedInvokableToolCall(ctx context.Context, endpoint EnhancedInvokableToolCallEndpoint, tCtx *ToolContext) (EnhancedInvokableToolCallEndpoint, error)

    // WrapEnhancedStreamableToolCall wraps an enhanced tool's streaming execution.
    // This method is only called for tools that implement EnhancedStreamableTool.
    WrapEnhancedStreamableToolCall(ctx context.Context, endpoint EnhancedStreamableToolCallEndpoint, tCtx *ToolContext) (EnhancedStreamableToolCallEndpoint, error)

    // WrapModel wraps a chat model with custom behavior.
    // This method is called at request time when the model is about to be invoked.
    WrapModel(ctx context.Context, m model.BaseChatModel, mc *ModelContext) (model.BaseChatModel, error)
}
```

**设计理念：**
- **装饰器模式**：每个 Middleware 包装原始调用，可以修改输入、输出或错误
- **洋葱模型**：请求从外向内穿过 Middleware，响应从内向外返回
- **可组合**：多个 Middleware 按顺序执行

### Middleware 执行顺序

`Handlers`（即 Middlewares）按**数组正序**包装，形成洋葱模型：

```go
Handlers: []adk.ChatModelAgentMiddleware{
    &middlewareA{},  // 最外层：最先 Wrap，最先拦截请求，但 WrapModel 最后生效
    &middlewareB{},  // 中间层
    &middlewareC{},  // 最内层：最后 Wrap
}
```

**对于 Tool 调用的执行顺序：**

```
请求 → A.Wrap → B.Wrap → C.Wrap → 实际 Tool 执行 → C返回 → B返回 → A返回 → 响应
```

**实用建议：** 将 `safeToolMiddleware`（错误捕获）放在最内层（数组末尾），确保其他 Middleware 抛出的中断错误能正确向外传播。

### SafeToolMiddleware

`SafeToolMiddleware` 将 Tool 错误转换为字符串，让模型能够理解并处理：

```go
type safeToolMiddleware struct {
    *adk.BaseChatModelAgentMiddleware
}

func (m *safeToolMiddleware) WrapInvokableToolCall(
    _ context.Context,
    endpoint adk.InvokableToolCallEndpoint,
    _ *adk.ToolContext,
) (adk.InvokableToolCallEndpoint, error) {
    return func(ctx context.Context, args string, opts ...tool.Option) (string, error) {
        result, err := endpoint(ctx, args, opts...)
        if err != nil {
            if _, ok := compose.IsInterruptRerunError(err); ok {
                return "", err
            }
            // 将错误转换为字符串，而不是返回错误
            return fmt.Sprintf("[tool error] %v", err), nil
        }
        return result, nil
    }, nil
}
```

**效果：**

```text
[tool call] read_file(file_path: "nonexistent.txt")
[tool result] [tool error] open nonexistent.txt: no such file or directory
[assistant] 抱歉，文件不存在，请检查文件路径...
// 对话继续，模型可以根据错误信息调整策略
```

### ModelRetryConfig

`ModelRetryConfig` 配置 ChatModel 的自动重试：

```go
type ModelRetryConfig struct {
    MaxRetries int                          // 最大重试次数
    IsRetryAble func(ctx context.Context, err error) bool  // 判断是否可重试
}
```

**使用方式（以 DeepAgent 为例）：**

```go
agent, err := deep.New(ctx, &deep.Config{
    // ...
    ModelRetryConfig: &adk.ModelRetryConfig{
        MaxRetries: 5,
        IsRetryAble: func(_ context.Context, err error) bool {
            // 429 限流错误可重试
            return strings.Contains(err.Error(), "429") ||
                strings.Contains(err.Error(), "Too Many Requests") ||
                strings.Contains(err.Error(), "qpm limit")
        },
    },
})
```

**重试策略：**
- 指数退避：每次重试间隔递增
- 可配置条件：通过 `IsRetryAble` 判断哪些错误可重试
- 自动恢复：无需用户干预

## Middleware 的实现

### 1. 实现 SafeToolMiddleware

```go
type safeToolMiddleware struct {
    *adk.BaseChatModelAgentMiddleware
}

func (m *safeToolMiddleware) WrapInvokableToolCall(
    _ context.Context,
    endpoint adk.InvokableToolCallEndpoint,
    _ *adk.ToolContext,
) (adk.InvokableToolCallEndpoint, error) {
    return func(ctx context.Context, args string, opts ...tool.Option) (string, error) {
        result, err := endpoint(ctx, args, opts...)
        if err != nil {
            // 中断错误不转换，需要继续传播
            if _, ok := compose.IsInterruptRerunError(err); ok {
                return "", err
            }
            // 其他错误转换为字符串
            return fmt.Sprintf("[tool error] %v", err), nil
        }
        return result, nil
    }, nil
}
```

### 2. 实现流式 Tool 错误处理

```go
func (m *safeToolMiddleware) WrapStreamableToolCall(
    _ context.Context,
    endpoint adk.StreamableToolCallEndpoint,
    _ *adk.ToolContext,
) (adk.StreamableToolCallEndpoint, error) {
    return func(ctx context.Context, args string, opts ...tool.Option) (*schema.StreamReader[string], error) {
        sr, err := endpoint(ctx, args, opts...)
        if err != nil {
            if _, ok := compose.IsInterruptRerunError(err); ok {
                return nil, err
            }
            // 返回包含错误信息的单帧流
            return singleChunkReader(fmt.Sprintf("[tool error] %v", err)), nil
        }
        // 包装流，捕获流中的错误
        return safeWrapReader(sr), nil
    }, nil
}
```

### 3. 配置 Agent 使用 Middleware

本章继续使用第四章引入的 `DeepAgent`，在其 `Handlers` 字段中注册 Middleware：

```go
agent, err := deep.New(ctx, &deep.Config{
    Name:           "Ch05MiddlewareAgent",
    Description:    "ChatWithDoc agent with safe tool middleware and retry.",
    ChatModel:      cm,
    Instruction:    agentInstruction,
    Backend:        backend,
    StreamingShell: backend,
    MaxIteration:   50,
    Handlers: []adk.ChatModelAgentMiddleware{
        &safeToolMiddleware{},  // 将 Tool 错误转换为字符串
    },
    ModelRetryConfig: &adk.ModelRetryConfig{
        MaxRetries: 5,
        IsRetryAble: func(_ context.Context, err error) bool {
            return strings.Contains(err.Error(), "429") ||
                strings.Contains(err.Error(), "Too Many Requests") ||
                strings.Contains(err.Error(), "qpm limit")
        },
    },
})
```

**注意**：`Handlers` 字段（在配置中）和 "Middleware"（在文档中讨论的概念）是同一回事——`Handlers` 是配置字段名，而 `ChatModelAgentMiddleware` 是接口名。
```

**关键代码片段（**注意：这是简化后的代码片段，不能直接运行，完整代码请参考** [cmd/ch05/main.go](https://github.com/cloudwego/eino-examples/blob/main/quickstart/chatwitheino/cmd/ch05/main.go)）：

```go
// SafeToolMiddleware 捕获 Tool 错误并转换为字符串
type safeToolMiddleware struct {
    *adk.BaseChatModelAgentMiddleware
}

func (m *safeToolMiddleware) WrapInvokableToolCall(
    _ context.Context,
    endpoint adk.InvokableToolCallEndpoint,
    _ *adk.ToolContext,
) (adk.InvokableToolCallEndpoint, error) {
    return func(ctx context.Context, args string, opts ...tool.Option) (string, error) {
        result, err := endpoint(ctx, args, opts...)
        if err != nil {
            if _, ok := compose.IsInterruptRerunError(err); ok {
                return "", err
            }
            return fmt.Sprintf("[tool error] %v", err), nil
        }
        return result, nil
    }, nil
}

// 配置 DeepAgent（与第四章一样，新增 Handlers 和 ModelRetryConfig）
agent, _ := deep.New(ctx, &deep.Config{
    ChatModel:      cm,
    Backend:        backend,
    StreamingShell: backend,
    MaxIteration:   50,
    Handlers: []adk.ChatModelAgentMiddleware{
        &safeToolMiddleware{},
    },
    ModelRetryConfig: &adk.ModelRetryConfig{
        MaxRetries: 5,
        IsRetryAble: func(_ context.Context, err error) bool {
            return strings.Contains(err.Error(), "429")
        },
    },
})
```

## Middleware 执行流程

```
┌─────────────────────────────────────────┐
│  用户：读取不存在的文件                   │
└─────────────────────────────────────────┘
                   ↓
        ┌──────────────────────┐
        │  Agent 分析意图       │
        │  决定调用 read_file   │
        └──────────────────────┘
                   ↓
        ┌──────────────────────┐
        │  SafeToolMiddleware  │
        │  拦截 Tool 调用       │
        └──────────────────────┘
                   ↓
        ┌──────────────────────┐
        │  执行 read_file       │
        │  返回错误             │
        └──────────────────────┘
                   ↓
        ┌──────────────────────┐
        │  SafeToolMiddleware  │
        │  将错误转换为字符串    │
        └──────────────────────┘
                   ↓
        ┌──────────────────────┐
        │  返回 Tool Result     │
        │  "[tool error] ..."   │
        └──────────────────────┘
                   ↓
        ┌──────────────────────┐
        │  Agent 生成回复       │
        │  "抱歉，文件不存在..." │
        └──────────────────────┘
```

## 本章小结

- **Middleware**：Agent 的拦截器，可以在调用前后插入自定义逻辑
- **SafeToolMiddleware**：将 Tool 错误转换为字符串，让模型能够理解并处理
- **ModelRetryConfig**：配置 ChatModel 的自动重试，处理限流等临时错误
- **装饰器模式**：Middleware 包装原始调用，可以修改输入、输出或错误
- **洋葱模型**：请求从外向内穿过 Middleware，响应从内向外返回

## 扩展思考

**Eino 内置 Middleware：**

| Middleware | 功能说明 |
|------------|----------|
| **reduction** | 工具输出缩减，当工具返回内容过长时自动截断并卸载到文件系统，防止上下文溢出 |
| **summarization** | 对话历史自动摘要，当 token 数量超过阈值时自动生成摘要压缩历史 |
| **skill** | 技能加载中间件，让 Agent 能够动态加载和执行预定义的技能 |

**Middleware 链示例：**

```go
import (
    "github.com/cloudwego/eino/adk/middlewares/reduction"
    "github.com/cloudwego/eino/adk/middlewares/summarization"
    "github.com/cloudwego/eino/adk/middlewares/skill"
)

// 创建 reduction middleware：管理工具输出长度
reductionMW, _ := reduction.New(ctx, &reduction.Config{
    Backend:           filesystemBackend,     // 存储后端
    MaxLengthForTrunc: 50000,                  // 单次工具输出最大长度
    MaxTokensForClear: 30000,                  // 触发清理的 token 阈值
})

// 创建 summarization middleware：自动压缩对话历史
summarizationMW, _ := summarization.New(ctx, &summarization.Config{
    Model: chatModel,                          // 用于生成摘要的模型
    Trigger: &summarization.TriggerCondition{
        ContextTokens: 190000,                 // 触发摘要的 token 阈值
    },
})

// 组合多个 middleware（概念示例，使用 DeepAgent 时将 adk.NewChatModelAgent 替换为 deep.New）
agent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
    Handlers: []adk.ChatModelAgentMiddleware{  // 注意：配置字段名为 Handlers，概念上与 Middlewares 等价
        summarizationMW,   // 最外层：对话历史摘要
        reductionMW,       // 中间层：工具输出缩减
    },
})
```
