---
title: "第十一章：TurnLoop — 抢占、中止与多轮生命周期"
---

上一章我们用 `adk.Runner` 实现了完整的 A2UI Web 应用。它能正常工作，但试试这个场景：

> 你问 Agent 一个复杂问题，它开始调用工具、生成长回答……但你忽然意识到问错了，想换一个问题。

在上一章的 Runner 模式下，你只能等它说完，或者刷新页面丢弃一切。

本章引入 `adk.TurnLoop`，让 Agent 支持两个用户侧可感知的新能力：**抢占**和**中止**。

## 前置条件

与第一章一致：需要配置一个可用的 ChatModel（OpenAI 或 Ark），详见第一章的"前置条件"部分。

## 运行 & 体验

在 `quickstart/chatwitheino` 目录下执行：

```bash
go run .
```

打开浏览器访问 `http://localhost:8080`，然后试试以下操作：

### 体验抢占（Preempt）

1. 发送一个会触发长回答的问题，例如"详细解释一下 Eino 的所有组件"
2. **在 Agent 还在回答时**，直接发送一条新消息，例如"算了，就告诉我 ChatModel 是什么"
3. 观察：旧回答立即停止，Agent 开始回答新问题

### 体验中止（Abort）

1. 发送一个问题
2. **在 Agent 回答过程中**，点击右上角的 **Abort 按钮**
3. 观察：Agent 立即停止，不再继续输出

这两个能力在上一章的 Runner 版本中都不存在。以下解释它们是如何实现的。

## 代码位置

- 入口代码：[main.go](https://github.com/cloudwego/eino-examples/blob/main/quickstart/chatwitheino/main.go)
- Agent 构建：[agent.go](https://github.com/cloudwego/eino-examples/blob/main/quickstart/chatwitheino/agent.go)
- TurnLoop 服务端：[server/server.go](https://github.com/cloudwego/eino-examples/blob/main/quickstart/chatwitheino/server/server.go)

## 为什么 Runner 做不到

上一章的 `cmd/ch10` 中，每个 `/sessions/:id/chat` 请求调用一次 `runner.Run(ctx, messages)`。Runner 是**单轮（single-turn）**模型——调用一次、执行一次、结束。如果用户在 Agent 执行过程中又发了一条消息，Runner 没有"正在运行的循环"可以接收它。

TurnLoop 则是一个**持久运行的多轮（multi-turn）执行循环**。它在轮次之间保持 idle 等待，随时可以通过 `Push()` 接收新输入并立即响应。正是因为有一个持续运行的循环，抢占和中止才成为可能——你可以打断一个正在进行的轮次，或者直接停止整个循环。

| 能力 | Ch10（Runner，单轮） | Ch11（TurnLoop，多轮） |
|------|---------------------|----------------------|
| 流式输出 | ✅ | ✅ |
| 审批/中断 | ✅ | ✅ |
| 跨轮次持久运行、实时响应新输入 | ❌ 每次 `Run()` 独立 | ✅ `Push()` 随时送入 |
| 抢占正在进行的回答 | ❌ | ✅ `Push(item, WithPreempt(...))` |
| 中止 Agent | ❌ | ✅ `loop.Stop(WithImmediate())` |
| 灵活的 per-turn 输入构建 | ❌ 业务层手动拼装 | ✅ `GenInput` 回调 |

## TurnLoop 的核心模型

TurnLoop 是一个**基于推送的事件循环，以轮次（turn）为单位管理 Agent 的执行**。与 Runner 的"调用一次、执行一次"不同，TurnLoop 持续运行：轮次结束后进入 idle 等待，新 item 到来时立即启动下一轮。

```
Push(item) → [队列] → GenInput(items) → Agent.Run() → OnAgentEvents(events)
                                ↑                              │
                                └──── idle 等待 / 下一轮 ←──────┘
```

关键概念：

- **Item**：用户输入的载体。本示例定义为 `ChatItem`，可以携带用户消息或审批决定
- **GenInput**：从队列中的 items 构建 Agent 输入（选择哪些 items 消费、哪些保留给下一轮）
- **OnAgentEvents**：接收 Agent 输出的事件流，负责渲染和持久化
- **Push**：向队列推入新 item，可附带抢占选项

## 一个 Session 对应一个 TurnLoop

在本示例的 Web 场景中，每个聊天 session 对应一个 TurnLoop 实例。当用户发送第一条消息时，服务端为该 session 创建一个 TurnLoop 并调用 `Run()` 启动它；后续消息通过 `Push()` 送入同一个循环。这个循环在轮次之间保持 idle 等待，直到 session 被删除或用户 abort。

这是 TurnLoop 最典型的使用模式：**循环的生命周期与用户会话绑定**。一个长期运行的 TurnLoop 让抢占和中止成为自然的操作——因为"正在运行的循环"始终存在，新输入随时可以送入。

## 常规流程：idle → 新消息 → 回答 → idle

最简单的场景是用户依次提问、等回答、再提下一个问题：

```go
// 用户发送第一条消息时，创建并启动 TurnLoop
loop := adk.NewTurnLoop(cfg)
loop.Push(&ChatItem{Query: "hello"})
loop.Run(ctx)
// → GenInput 构建输入 → Agent 执行 → OnAgentEvents 流式输出
// → 轮次结束，TurnLoop 进入 idle 等待

// 用户发送第二条消息（此时 loop 处于 idle）
loop.Push(&ChatItem{Query: "explain Eino's architecture"})
// → TurnLoop 唤醒，开始新一轮：GenInput → Agent → OnAgentEvents → idle
```

这个流程与上一章的 Runner 在用户体验上没有区别——区别在于 TurnLoop 的循环**持续存在**，不需要每次都重新创建。而一旦用户在 Agent 还在回答时发来新消息，就进入了下面的"抢占"场景。

## 抢占是怎么实现的

当用户在 Agent 回答过程中发送新消息时，业务层只需一行代码触发抢占：

```go
loop.Push(item, adk.WithPreempt[*ChatItem, *schema.Message](adk.AfterToolCalls))
```

TurnLoop 收到这个指令后：

1. 等待当前 tool call 完成（`AfterToolCalls` 表示不打断正在执行的工具，避免不一致状态）
2. 取消当前轮次——OnAgentEvents 的 context 被取消，旧轮次退出
3. 从队列取出新 item，通过 GenInput 构建输入，启动新一轮

抢占模式可以根据业务需要选择不同的安全点：

| 模式 | 行为 |
|------|------|
| `AfterToolCalls` | 等当前 tool call 完成后取消（推荐，避免工具执行到一半） |
| `AfterChatModel` | 等当前 model 调用完成后取消 |
| `AnySafePoint` | 任一安全点即取消 |

> 本示例中 TurnLoop 运行在独立 goroutine 中，而 HTTP handler 需要把事件流写入 SSE 响应。两者之间通过 channel 协调（见 [server/server.go](https://github.com/cloudwego/eino-examples/blob/main/quickstart/chatwitheino/server/server.go) 中的 `iterEnvelope`/`iterResult` 以及 `handlerDone` 信号机制）。这些是 HTTP 适配层的细节，不属于 TurnLoop API 本身。

## 中止是怎么实现的

中止更简单——直接停止整个 TurnLoop：

```go
loop.Stop(adk.WithImmediate())  // 立即取消，不等待当前轮次
loop.Wait()                     // 等待完全退出
```

### Stop 的三种模式

| 模式 | 行为 |
|------|------|
| `loop.Stop()` | 轮次边界退出：等待当前轮次完成后退出 |
| `loop.Stop(WithImmediate())` | 立即退出：取消当前轮次的 context |
| `loop.Stop(WithGraceful())` | 安全点退出：在下一个安全点（如 tool call 之间）退出 |

## TurnLoop 的配置

创建 TurnLoop 时，通过 `TurnLoopConfig` 指定回调和选项：

```go
cfg := adk.TurnLoopConfig[*ChatItem, *schema.Message]{
    // GenInput：每轮开始时调用，决定"这一轮 Agent 看到什么"
    // 从队列中选择 items 构建 Agent 输入，返回 Consumed（本轮处理）和 Remaining（留到后续轮次）
    GenInput: func(ctx context.Context, loop *adk.TurnLoop[*ChatItem, *schema.Message], items []*ChatItem) (*adk.GenInputResult[*ChatItem, *schema.Message], error) {
        // ...构建 AgentInput，持久化用户消息...
    },

    // PrepareAgent：每轮调用一次，返回本轮使用的 Agent
    // 本示例直接返回同一个 Agent，但你可以根据 items 动态选择不同 Agent
    PrepareAgent: func(ctx context.Context, loop *adk.TurnLoop[*ChatItem, *schema.Message], consumed []*ChatItem) (adk.Agent, error) {
        return agent, nil
    },

    // OnAgentEvents：接收 Agent 的事件流，负责渲染输出和持久化中间消息
    // 本示例通过 channel 把事件流转交给 HTTP handler 做 SSE 输出
    OnAgentEvents: func(ctx context.Context, tc *adk.TurnContext[*ChatItem, *schema.Message], events *adk.AsyncIterator[*adk.AgentEvent]) error {
        // ...把 events 交给 HTTP handler，等待消费完成...
    },

    // 以下三个字段用于声明式 checkpoint（审批恢复），下一节详细介绍
    GenResume:    makeGenResume(),
    Store:        checkpointStore,
    CheckpointID: sessionID,
}

loop := adk.NewTurnLoop(cfg)
```

| 回调 | 调用时机 | 职责 |
|------|---------|------|
| `GenInput` | 队列中有 items 时 | 选择消费哪些 items，构建 Agent 输入 |
| `PrepareAgent` | GenInput 之后 | 返回本轮使用的 Agent 实例 |
| `OnAgentEvents` | Agent 产出事件流时 | 消费事件、渲染输出、持久化 |
| `GenResume` | 从 checkpoint 恢复时 | 用审批结果构建 ResumeParams（见下一节） |
| `Store` + `CheckpointID` | — | 启用声明式 checkpoint（见下一节） |

> 完整的回调实现请参考 [server/server.go](https://github.com/cloudwego/eino-examples/blob/main/quickstart/chatwitheino/server/server.go)。

## 声明式 Checkpoint：审批恢复的自动化

在第七章（Runner 模式）中，审批恢复需要业务层手动调用 `runner.ResumeWithParams()`，自己判断"这次是正常执行还是恢复执行"。TurnLoop 提供了更简洁的方式——在配置中声明 `Store` 和 `CheckpointID`（见上一节），TurnLoop 会自动处理保存与恢复：

1. Agent 执行到审批 interrupt 时，TurnLoop 自动将执行状态保存到 `Store`（以 `CheckpointID` 为 key）
2. 用户做出审批决定后，业务层创建一个新的 TurnLoop（使用**相同的** `CheckpointID`），并 Push 审批 item
3. 新 TurnLoop `Run()` 时，检测到 checkpoint 存在，**自动调用 `GenResume`**（而非 `GenInput`）获取恢复参数
4. Agent 从 interrupt 点继续执行

`GenResume` 的职责就是从新 Push 进来的 items 中提取审批结果，构建 `ResumeParams`：

```go
GenResume: func(ctx context.Context, loop *adk.TurnLoop[*ChatItem, *schema.Message],
    canceledItems, unhandledItems, newItems []*ChatItem,
) (*adk.GenResumeResult[*ChatItem, *schema.Message], error) {
    // newItems 包含审批恢复时 Push 的 item
    item := newItems[0]
    return &adk.GenResumeResult[*ChatItem, *schema.Message]{
        ResumeParams: &adk.ResumeParams{
            InterruptID: item.InterruptID,
            ApprovalResult: item.ApprovalResult,
        },
    }, nil
}
```

相比 Runner 的 `ResumeWithParams()`，声明式 checkpoint 让业务层不需要管理"正常执行 vs 恢复执行"的分支——TurnLoop 根据 checkpoint 是否存在自动选择走 `GenInput` 还是 `GenResume`。

## 本章小结

- **TurnLoop** 是一个持久运行的多轮执行循环，生命周期与用户会话绑定
- **常规流程**：`Push(item)` → GenInput → Agent → OnAgentEvents → idle → 等待下一个 Push
- **抢占**：`Push(item, WithPreempt(AfterToolCalls))` 一行代码取消当前轮次并开始新一轮
- **中止**：`loop.Stop(WithImmediate())` 一行代码终止整个循环
- **声明式 checkpoint**：配置 `Store` + `CheckpointID`，TurnLoop 自动处理 interrupt 的保存与恢复
- 回调的具体实现请参考 [server/server.go](https://github.com/cloudwego/eino-examples/blob/main/quickstart/chatwitheino/server/server.go)

## 系列收尾：完整 Agent 应用骨架

到本章为止，我们用一个可以实际运行的 Agent 串起了 Eino 的核心能力：

- **运行时**：Runner / TurnLoop 驱动执行，支持流式输出、抢占与中止
- **工具层**：Filesystem / Shell 等 Tool 能力接入，工具错误可被安全处理
- **中间件**：可插拔的 middleware/handler，用于错误处理、重试、审批等横切能力
- **可观测**：callbacks/trace 能力把关键链路打通，便于调试与线上观测
- **人机协作**：interrupt/resume + checkpoint 支持审批、补参、分支选择等交互式流程
- **确定性编排**：compose（graph/chain/workflow）把复杂业务流程组织为可维护、可复用的执行图
- **业务交付**：A2UI 协议把 Agent 能力以流式 UI 的形式呈现给用户
- **执行控制**：TurnLoop 提供抢占、中止、多轮生命周期管理，适配真实业务场景的复杂交互需求

你可以在这个骨架上逐步替换/扩展任意环节：模型、工具、存储、工作流、前端渲染协议，而不需要推倒重来。
