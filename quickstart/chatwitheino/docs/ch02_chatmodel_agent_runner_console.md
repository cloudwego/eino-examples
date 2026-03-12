---
title: "第二章：ChatModelAgent、Runner、AgentEvent（Console 多轮）"
---

本章目标：引入 ADK 的执行抽象（Agent + Runner），并用一个 Console 程序实现多轮对话。

## 代码位置

- 入口代码：[cmd/ch02/main.go](https://github.com/cloudwego/eino-examples/blob/main/quickstart/chatwitheino/cmd/ch02/main.go)

## 前置条件

与第一章一致：需要配置一个可用的 ChatModel（OpenAI 或 Ark）。

## 运行

在 `examples/quickstart/chatwitheino` 目录下执行：

```bash
go run ./cmd/ch02
```

看到提示后输入问题（空行退出）：

```text
you> 你好，解释一下 Eino 里的 Agent 是什么？
...
you> 再用一句话总结一下
...
```

## 关键概念

### 从 Component 到 Agent

第一章我们学习了 **Component**（组件），它是 Eino 中可替换、可组合的能力单元：

- `ChatModel`：调用大语言模型
- `Tool`：执行特定任务
- `Retriever`：检索信息
- `Loader`：加载数据

**Component 和 Agent 的关系：**

- **Component 不构成完整的 AI 应用**：它只是能力单元，需要被组织、编排、执行
- **Agent 是完整的 AI 应用**：它封装了完整的业务逻辑，可以直接运行
- **Agent 内部使用 Component**：最核心的是 `ChatModel`（对话能力）和 `Tool`（执行能力）

**为什么需要 Agent？**

如果只有 Component，你需要自己：
- 管理对话历史
- 编排调用流程（何时调用模型、何时调用工具）
- 处理流式输出
- 实现中断恢复
- ...

**Agent 提供了什么？**

- **完整的运行时框架**：通过 `Runner` 统一管理执行过程
- **标准的事件流输出**：`Run() -> AsyncIterator[*AgentEvent]`，支持流式、中断、恢复
- **可扩展能力**：可以添加 tools、middleware、interrupt 等
- **开箱即用**：创建 Agent 后直接运行，无需关心内部细节

**本章示例：**

`ChatModelAgent` 是最简单的 Agent，它内部只使用了 `ChatModel`，但已经具备了 Agent 的完整能力框架。后续章节会展示如何添加 `Tool` 等更多能力。

### Agent 接口

`Agent` 是 ADK 中的核心接口，定义了智能体的基本行为：

```go
type Agent interface {
    Name(ctx context.Context) string
    Description(ctx context.Context) string
    
    // Run 执行 Agent，返回事件流
    Run(ctx context.Context, input *AgentInput, options ...AgentRunOption) *AsyncIterator[*AgentEvent]
}
```

**接口职责：**
- `Name()` / `Description()`：标识 Agent 的名称和描述
- `Run()`：执行 Agent 的核心方法，接收输入消息，返回事件流

**设计理念：**
- **统一抽象**：所有 Agent（ChatModelAgent、WorkflowAgent、SupervisorAgent 等）都实现这个接口
- **事件驱动**：通过事件流（`AsyncIterator[*AgentEvent]`）输出执行过程，支持流式响应
- **可扩展性**：后续加入 tools、middleware、interrupt 等能力时，接口保持不变

### ChatModelAgent

`ChatModelAgent` 是 Agent 接口的一个实现，基于 ChatModel 构建：

```go
agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
    Name:        "Ch02ChatModelAgent",
    Description: "A minimal ChatModelAgent with in-memory multi-turn history.",
    Instruction: instruction,
    Model:       cm,
})
```

**ChatModel vs ChatModelAgent：本质区别**

| 维度 | ChatModel | ChatModelAgent |
|------|-----------|----------------|
| **定位** | Component（组件） | Agent（智能体） |
| **接口** | `Generate() / Stream()` | `Run() -> AsyncIterator[*AgentEvent]` |
| **输出** | 直接返回消息内容 | 返回事件流（包含消息、控制动作等） |
| **能力** | 单纯的模型调用 | 可扩展 tools、middleware、interrupt 等 |
| **适用场景** | 简单的对话场景 | 复杂的智能体应用 |

**为什么需要 ChatModelAgent？**

1. **统一抽象**：ChatModel 只是 Component 的一种，而 Agent 是更高层的抽象，可以组合多种 Component
2. **事件驱动**：Agent 输出事件流，支持流式响应、中断恢复、状态转移等复杂场景
3. **可扩展性**：ChatModelAgent 可以添加 tools、middleware、interrupt 等能力，而 ChatModel 只能调用模型
4. **编排友好**：Agent 可以被 Runner 统一管理，支持 checkpoint、恢复等运行时能力

**简单来说：**
- **ChatModel** = "负责与大语言模型通信的组件，屏蔽不同模型提供商的差异（OpenAI、Ark、Claude 等）"
- **ChatModelAgent** = "基于模型构建的智能体，可以调用模型，但还能做更多事"

**类比理解：**
- **ChatModel** 就像"数据库驱动"：负责与数据库通信，屏蔽 MySQL/PostgreSQL 的差异
- **ChatModelAgent** 就像"业务逻辑层"：基于数据库驱动构建，但还包含业务规则、事务管理等

**特点：**
- 封装了 ChatModel 的调用逻辑
- 提供统一的 `Run() -> AgentEvent` 输出形态
- 后续可以添加 tools、middleware 等能力

### Runner

`Runner` 是执行 Agent 的入口点，负责管理 Agent 的生命周期：

```go
type Runner struct {
    a Agent  // 要执行的 Agent
    enableStreaming bool
    store CheckPointStore  // 用于中断恢复的状态存储
}
```

**为什么需要 Runner？**

虽然 Agent 提供了 `Run()` 方法，但直接调用会缺少很多运行时能力：

1. **生命周期管理**：Runner 管理 Agent 的启动、恢复、中断等状态
2. **Checkpoint 支持**：配合 `CheckPointStore` 实现中断恢复（后续章节涉及）
3. **统一入口**：提供 `Run()` 和 `Query()` 等便捷方法
4. **事件流封装**：将 Agent 的事件流转换为可消费的 `AsyncIterator[*AgentEvent]`

**使用方式：**

```go
runner := adk.NewRunner(ctx, adk.RunnerConfig{
    Agent:           agent,
    EnableStreaming: true,
})

// 方式 1：传入消息列表
events := runner.Run(ctx, history)

// 方式 2：便捷方法，传入单个查询字符串
events := runner.Query(ctx, "你好")
```

### AgentEvent

`AgentEvent` 是 Runner 返回的事件单元：

```go
type AgentEvent struct {
    AgentName string
    RunPath   []RunStep

    Output *AgentOutput  // 输出内容
    Action *AgentAction  // 控制动作
    Err    error         // 执行错误
}
```

**主要字段：**
- `event.Err`：执行错误
- `event.Output.MessageOutput`：message 或 message stream（流式）
- `event.Action`：中断/转移/退出等控制动作（后续章节用到）

### AsyncIterator：事件流的消费方式

`Runner.Run()` 返回的是 `*AsyncIterator[*AgentEvent]`，这是一个非阻塞的流式迭代器。

**为什么用 AsyncIterator 而不是直接返回结果？**

因为 Agent 的执行是**流式**的：模型逐 token 生成回复，Tool 调用穿插其中。如果等全部完成再返回，用户需要等待更长时间。`AsyncIterator` 让你可以实时消费每一个事件。

**消费方式：**

```go
// events 是 *AsyncIterator[*AgentEvent]，由 runner.Run() 返回
events := runner.Run(ctx, history)

for {
    event, ok := events.Next()  // 获取下一个事件，阻塞直到有事件或结束
    if !ok {
        break  // 迭代器关闭，全部事件已消费
    }
    if event.Err != nil {
        // 处理错误
    }
    if event.Output != nil && event.Output.MessageOutput != nil {
        // 处理消息输出（可能是流式）
    }
}
```

**注意：**每次 `runner.Run()` 创建新的迭代器，消费一次后不可重复使用。

## 多轮对话的实现

本章实现的是简单的多轮对话：用户输入 → 模型回复 → 用户继续输入 → ...

**实现方式：**

没有 tools 时，`ChatModelAgent` 在一次 `Run()` 里只会完成一轮模型调用。多轮对话是通过调用侧维护 history 实现的：

1. 用 `history []*schema.Message` 保存累计对话
2. 每次用户输入：把 `UserMessage` 追加到 history
3. 调用 `runner.Run(ctx, history)` 得到事件流，消费得到 assistant 文本
4. 把本轮 assistant 文本追加回 history，进入下一轮

**关键代码片段（**注意：这是简化后的代码片段，不能直接运行，完整代码请参考** [cmd/ch02/main.go](https://github.com/cloudwego/eino-examples/blob/main/quickstart/chatwitheino/cmd/ch02/main.go)）：

```go
history := make([]*schema.Message, 0, 16)

for {
    // 1. 读取用户输入
    line := readUserInput()
    if line == "" {
        break
    }
    
    // 2. 追加用户消息到 history
    history = append(history, schema.UserMessage(line))
    
    // 3. 调用 Runner 执行 Agent
    events := runner.Run(ctx, history)
    
    // 4. 消费事件流，收集 assistant 回复
    content := collectAssistantFromEvents(events)
    
    // 5. 追加 assistant 消息到 history
    history = append(history, schema.AssistantMessage(content, nil))
}
```

**流程图：**

```
┌─────────────────────────────────────────┐
│  初始化 history = []                     │
└─────────────────────────────────────────┘
                   ↓
        ┌──────────────────────┐
        │  用户输入 UserMessage  │
        └──────────────────────┘
                   ↓
        ┌──────────────────────┐
        │  追加到 history       │
        └──────────────────────┘
                   ↓
        ┌──────────────────────┐
        │  runner.Run(history) │
        └──────────────────────┘
                   ↓
        ┌──────────────────────┐
        │  消费事件流           │
        └──────────────────────┘
                   ↓
        ┌──────────────────────┐
        │  追加 AssistantMessage│
        └──────────────────────┘
                   ↓
              (循环继续)
```

## 本章小结

- **Agent 接口**：定义智能体的基本行为，核心是 `Run() -> AsyncIterator[*AgentEvent]`
- **ChatModelAgent**：基于 ChatModel 实现的 Agent，提供统一的执行抽象
- **Runner**：Agent 的执行入口，管理生命周期、checkpoint、事件流等运行时能力
- **AgentEvent**：事件驱动的输出单元，支持流式响应和控制动作
- **多轮对话**：通过调用侧维护 history 实现，每次 `Run()` 完成一轮对话
