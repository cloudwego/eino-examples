---
title: "第六章：Callback 与 Trace（可观测性）"
---

本章目标：理解 Callback 机制，集成 CozeLoop 实现链路追踪和可观测性。

## 代码位置

- 入口代码：[cmd/ch06/main.go](https://github.com/cloudwego/eino-examples/blob/main/quickstart/chatwitheino/cmd/ch06/main.go)

## 前置条件

与第一章一致：需要配置一个可用的 ChatModel（OpenAI 或 Ark）。同时，需要与第四章一样设置 `PROJECT_ROOT`：

```bash
export PROJECT_ROOT=/path/to/eino  # Eino 核心库根目录（不设置则默认使用当前目录）
```

可选：配置 CozeLoop 实现链路追踪：

```bash
export COZELOOP_WORKSPACE_ID=your_workspace_id
export COZELOOP_API_TOKEN=your_token
```

## 运行

在 `examples/quickstart/chatwitheino` 目录下执行：

```bash
# 设置项目根目录
export PROJECT_ROOT=/path/to/your/project

# 可选：配置 CozeLoop
export COZELOOP_WORKSPACE_ID=your_workspace_id
export COZELOOP_API_TOKEN=your_token

go run ./cmd/ch06
```

输出示例：

```text
[trace] starting session: 083d16da-6b13-4fe6-afb0-c45d8f490ce1
you> 你好
[trace] chat_model_generate: model=gpt-4.1-mini tokens=150
[trace] tool_call: name=list_files duration=23ms
[assistant] 你好！有什么我可以帮助你的吗？
```

## 从黑盒到白盒：为什么需要 Callback

前几章我们实现的 Agent 是一个"黑盒"：输入问题，输出答案，但中间发生了什么我们并不清楚。

**黑盒的问题：**
- 不知道模型调用了多少次
- 不知道 Tool 执行了多长时间
- 不知道 Token 消耗了多少
- 出问题时难以定位原因

**Callback 的定位：**
- **Callback 是 Eino 的旁路机制**：从 component 到 compose（下文详谈）到 adk，一以贯之
- **Callback 在固定点位触发**：组件生命周期的 5 个关键时机
- **Callback 可抽取实时信息**：输入、输出、错误、流式数据等
- **Callback 用途广泛**：观测、日志、指标、追踪、调试、审计等

**简单类比：**
- **Agent** = "业务逻辑"（主路）
- **Callback** = "旁路钩子"（在固定点位抽取信息）

## 关键概念

### Handler 接口

`Handler` 是 Eino 中定义回调处理器的核心接口：

```go
type Handler interface {
    // 非流式输入（组件开始处理前）
    OnStart(ctx context.Context, info *RunInfo, input CallbackInput) context.Context
    
    // 非流式输出（组件成功返回后）
    OnEnd(ctx context.Context, info *RunInfo, output CallbackOutput) context.Context
    
    // 错误（组件返回错误时）
    OnError(ctx context.Context, info *RunInfo, err error) context.Context
    
    // 流式输入（组件接收流式输入时）
    OnStartWithStreamInput(ctx context.Context, info *RunInfo, 
        input *schema.StreamReader[CallbackInput]) context.Context
    
    // 流式输出（组件返回流式输出时）
    OnEndWithStreamOutput(ctx context.Context, info *RunInfo, 
        output *schema.StreamReader[CallbackOutput]) context.Context
}
```

**设计理念：**
- **旁路机制**：不干扰主流程，在固定点位抽取信息
- **全流程覆盖**：从 component 到 compose 到 adk，所有组件都支持
- **状态传递**：同一 Handler 的 OnStart→OnEnd 可通过 context 传递状态
- **性能优化**：实现 `TimingChecker` 接口可跳过不需要的时机

**RunInfo 结构：**
```go
type RunInfo struct {
    Name      string        // 业务名称（节点名或用户指定）
    Type      string        // 实现类型（如 "OpenAI"）
    Component string        // 组件类型（如 "ChatModel"）
}
```

**重要提示：**
- 流式回调必须关闭 StreamReader，否则会导致 goroutine 泄漏
- 不要修改 Input/Output，它们被所有下游共享
- RunInfo 可能为 nil，使用前需要检查

### CozeLoop

CozeLoop 是字节跳动开源的 AI 应用可观测性平台，提供了：

- **链路追踪**：完整的调用链路可视化
- **指标监控**：延迟、Token 消耗、错误率等
- **日志聚合**：集中管理所有日志
- **调试支持**：在线查看和调试

**集成方式：**

```go
import (
    clc "github.com/cloudwego/eino-ext/callbacks/cozeloop"
    "github.com/cloudwego/eino/callbacks"
    "github.com/coze-dev/cozeloop-go"
)

// 创建 CozeLoop 客户端
client, err := cozeloop.NewClient(
    cozeloop.WithAPIToken(apiToken),
    cozeloop.WithWorkspaceID(workspaceID),
)

// 注册为全局 Callback
callbacks.AppendGlobalHandlers(clc.NewLoopHandler(client))
```

### Callback 的触发时机

Callback 在组件生命周期的 5 个关键时机触发。下表中 `Timing*` 是 Eino 内部常量名（用于 `TimingChecker` 接口），对应的 Handler 接口方法是右侧所示：

| 时机常量 | 对应 Handler 方法 | 触发点 | 输入/输出 |
|------|--------|--------|-----------|
| `TimingOnStart` | `OnStart` | 组件开始处理前 | CallbackInput |
| `TimingOnEnd` | `OnEnd` | 组件成功返回后 | CallbackOutput |
| `TimingOnError` | `OnError` | 组件返回错误时 | error |
| `TimingOnStartWithStreamInput` | `OnStartWithStreamInput` | 组件接收流式输入时 | StreamReader[CallbackInput] |
| `TimingOnEndWithStreamOutput` | `OnEndWithStreamOutput` | 组件返回流式输出时 | StreamReader[CallbackOutput] |

**示例：ChatModel 调用流程**

```
┌─────────────────────────────────────────┐
│  ChatModel.Generate(ctx, messages)      │
└─────────────────────────────────────────┘
                   ↓
        ┌──────────────────────┐
        │  OnStart             │  ← 输入: CallbackInput (messages)
        └──────────────────────┘
                   ↓
        ┌──────────────────────┐
        │  模型处理             │
        └──────────────────────┘
                   ↓
        ┌──────────────────────┐
        │  OnEnd               │  ← 输出: CallbackOutput (response)
        └──────────────────────┘
```

**示例：流式输出流程**

```
┌─────────────────────────────────────────┐
│  ChatModel.Stream(ctx, messages)        │
└─────────────────────────────────────────┘
                   ↓
        ┌──────────────────────┐
        │  OnStart             │  ← 输入: CallbackInput (messages)
        └──────────────────────┘
                   ↓
        ┌──────────────────────┐
        │  模型处理（流式）     │
        └──────────────────────┘
                   ↓
        ┌──────────────────────┐
        │  OnEndWithStreamOutput │  ← 输出: StreamReader[CallbackOutput]
        └──────────────────────┘
                   ↓
        ┌──────────────────────┐
        │  逐个 chunk 返回      │
        └──────────────────────┘
```

**注意：**
- 流式错误（stream 中途出错）不会触发 OnError，而是在 StreamReader 中返回
- 同一 Handler 的 OnStart→OnEnd 可通过 context 传递状态
- 不同 Handler 之间没有执行顺序保证

## Callback 的实现

### 1. 实现自定义 Callback Handler

完整实现 `Handler` 接口需要实现所有 5 个方法，较为繁琐。Eino 提供了 `callbacks.HandlerHelper` 帮助类来简化实现：

```go
import "github.com/cloudwego/eino/callbacks"

// 使用 NewHandlerHelper 注册感兴趣的回调
handler := callbacks.NewHandlerHelper().
    OnStart(func(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
        log.Printf("[trace] %s/%s start", info.Component, info.Name)
        return ctx
    }).
    OnEnd(func(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
        log.Printf("[trace] %s/%s end", info.Component, info.Name)
        return ctx
    }).
    OnError(func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
        log.Printf("[trace] %s/%s error: %v", info.Component, info.Name, err)
        return ctx
    }).
    Handler()

// 注册为全局 Callback
callbacks.AppendGlobalHandlers(handler)
```

**注意**：`RunInfo` 可能为 `nil`（如顶层调用没有 RunInfo），使用前请检查。

### 2. 集成 CozeLoop

```go
// Setup CozeLoop tracing (optional)
// Set COZELOOP_API_TOKEN and COZELOOP_WORKSPACE_ID to enable
cozeloopApiToken := os.Getenv("COZELOOP_API_TOKEN")
cozeloopWorkspaceID := os.Getenv("COZELOOP_WORKSPACE_ID")
if cozeloopApiToken != "" && cozeloopWorkspaceID != "" {
    client, err := cozeloop.NewClient(
        cozeloop.WithAPIToken(cozeloopApiToken),
        cozeloop.WithWorkspaceID(cozeloopWorkspaceID),
    )
    if err != nil {
        log.Fatalf("cozeloop.NewClient failed: %v", err)
    }
    defer func() {
        time.Sleep(5 * time.Second)
        client.Close(ctx)
    }()
    callbacks.AppendGlobalHandlers(clc.NewLoopHandler(client))
    log.Println("CozeLoop tracing enabled")
} else {
    log.Println("CozeLoop tracing disabled (set COZELOOP_API_TOKEN and COZELOOP_WORKSPACE_ID to enable)")
}
```

**关键代码片段（**注意：这是简化后的代码片段，不能直接运行，完整代码请参考** [cmd/ch06/main.go](https://github.com/cloudwego/eino-examples/blob/main/quickstart/chatwitheino/cmd/ch06/main.go)）：

```go
// 设置 CozeLoop 追踪
cozeloopApiToken := os.Getenv("COZELOOP_API_TOKEN")
cozeloopWorkspaceID := os.Getenv("COZELOOP_WORKSPACE_ID")
if cozeloopApiToken != "" && cozeloopWorkspaceID != "" {
    client, err := cozeloop.NewClient(
        cozeloop.WithAPIToken(cozeloopApiToken),
        cozeloop.WithWorkspaceID(cozeloopWorkspaceID),
    )
    if err != nil {
        log.Fatalf("cozeloop.NewClient failed: %v", err)
    }
    defer func() {
        time.Sleep(5 * time.Second)
        client.Close(ctx)
    }()
    callbacks.AppendGlobalHandlers(clc.NewLoopHandler(client))
}
```

## 可观测性的价值

### 1. 性能分析

通过 Callback 收集的数据，可以分析：
- 模型调用延迟分布
- Tool 执行时间排行
- Token 消耗趋势

### 2. 错误追踪

当 Agent 出现问题时：
- 查看完整的调用链路
- 定位是哪个环节出错
- 分析错误原因

### 3. 成本优化

通过 Token 消耗数据：
- 识别高消耗的对话
- 优化 Prompt 减少 Token
- 选择更经济的模型

## 本章小结

- **Callback**：Eino 的观测钩子，在关键节点触发回调
- **CozeLoop**：字节跳动的 AI 应用可观测性平台
- **全局注册**：通过 `callbacks.AppendGlobalHandlers` 注册全局 Callback
- **非侵入式**：业务代码不需要修改，Callback 自动触发
- **可观测性价值**：性能分析、错误追踪、成本优化

## 扩展思考

**其他 Callback 实现：**
- OpenTelemetry Callback：对接标准可观测性协议
- 自定义日志 Callback：记录到本地文件
- 指标 Callback：对接 Prometheus 等监控系统

**高级用法：**
- 在 Callback 中实现采样（只记录部分请求）
- 在 Callback 中实现限流（根据 Token 消耗）
- 在 Callback 中实现告警（错误率过高时通知）
