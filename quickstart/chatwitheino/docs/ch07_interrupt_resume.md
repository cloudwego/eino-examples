---
title: "第七章：Interrupt/Resume（中断与恢复）"
---

本章目标：理解 Interrupt/Resume 机制，实现 Tool 审批流程，让用户在敏感操作前进行确认。

## 代码位置

- 入口代码：[cmd/ch07/main.go](https://github.com/cloudwego/eino-examples/blob/main/quickstart/chatwitheino/cmd/ch07/main.go)

## 前置条件

与第一章一致：需要配置一个可用的 ChatModel（OpenAI 或 Ark）。同时，需要与第四章一样设置 `PROJECT_ROOT`：

```bash
export PROJECT_ROOT=/path/to/eino  # Eino 核心库根目录（不设置则默认使用当前目录）
```

## 运行

在 `examples/quickstart/chatwitheino` 目录下执行：

```bash
# 设置项目根目录
export PROJECT_ROOT=/path/to/your/project

go run ./cmd/ch07
```

输出示例：

```text
you> 请执行命令 echo hello

⚠️  Approval Required ⚠️
Tool: execute
Arguments: {"command":"echo hello"}

Approve this action? (y/n): y
[tool result] hello

hello
```

## 从自动执行到人工审批：为什么需要 Interrupt

前几章我们实现的 Agent 会自动执行所有 Tool 调用，但在某些场景下这是危险的：

**自动执行的风险：**
- 删除文件：误删重要数据
- 发送邮件：发送错误内容
- 执行命令：执行危险操作
- 修改配置：破坏系统设置

**Interrupt 的定位：**
- **Interrupt 是 Agent 的暂停机制**：在关键操作前暂停，等待用户确认
- **Interrupt 可携带信息**：向用户展示即将执行的操作
- **Interrupt 可恢复**：用户确认后继续执行，拒绝后返回错误

**简单类比：**
- **自动执行** = "自动驾驶"（完全信任系统）
- **Interrupt** = "人工接管"（关键决策由人来做）

## 关键概念

### Interrupt 机制

`Interrupt` 是 Eino 中实现人机协作的核心机制。

**核心思想：在执行关键操作前暂停，等待用户确认后继续。**

一个需要审批的 Tool 的执行被分成**两个阶段**：

1. **第一次调用（触发中断）**：Tool 保存当前参数，然后返回一个中断信号。Runner 暂停执行，向调用侧返回 Interrupt 事件。
2. **用户审批后恢复（Resume）**：Runner 重新调用 Tool，此时 Tool 检测到"已中断过"，直接读取用户的审批结果并执行（或拒绝）。

**简化版伪代码：**

```
func myTool(ctx, args):
    if 第一次调用:
        保存 args
        return 中断信号  // Runner 暂停，展示审批提示
    else:  // Resume 后的第二次调用
        if 用户批准:
            return 执行操作(保存的 args)
        else:
            return "操作被用户拒绝"
```

**完整代码及关键字段说明：**

```go
// 在 Tool 中触发中断
func myTool(ctx context.Context, args string) (string, error) {
    // wasInterrupted: 是否是 Resume 后的第二次调用（第一次为 false，Resume 后为 true）
    // storedArgs: 第一次调用时通过 StatefulInterrupt 保存的参数，Resume 后可取回
    wasInterrupted, _, storedArgs := tool.GetInterruptState[string](ctx)

    if !wasInterrupted {
        // 第一次调用：触发中断，同时保存 args 供 Resume 后使用
        return "", tool.StatefulInterrupt(ctx, &ApprovalInfo{
            ToolName:        "my_tool",
            ArgumentsInJSON: args,
        }, args)  // 第三个参数是要保存的状态（Resume 后通过 storedArgs 取回）
    }

    // Resume 后的第二次调用：读取用户审批结果
    // isTarget: 本次 Resume 是否针对当前 Tool（一次 Resume 只针对一个 Tool）
    // hasData:  Resume 时是否携带了审批结果数据
    // data:     用户传入的审批结果
    isTarget, hasData, data := tool.GetResumeContext[*ApprovalResult](ctx)
    if isTarget && hasData {
        if data.Approved {
            return doSomething(storedArgs)  // 使用保存的参数执行实际操作
        }
        return "Operation rejected by user", nil
    }

    // 其他情况（isTarget=false 意味着本次 Resume 目标不是当前 Tool）：重新中断
    return "", tool.StatefulInterrupt(ctx, &ApprovalInfo{
        ToolName:        "my_tool",
        ArgumentsInJSON: storedArgs,
    }, storedArgs)
}
```

### ApprovalMiddleware

`ApprovalMiddleware` 是一个通用的审批中间件，可以拦截特定 Tool 的调用：

```go
type approvalMiddleware struct {
    *adk.BaseChatModelAgentMiddleware
}

func (m *approvalMiddleware) WrapInvokableToolCall(
    _ context.Context,
    endpoint adk.InvokableToolCallEndpoint,
    tCtx *adk.ToolContext,
) (adk.InvokableToolCallEndpoint, error) {
    // 只拦截需要审批的 Tool
    if tCtx.Name != "execute" {
        return endpoint, nil
    }
    
    return func(ctx context.Context, args string, opts ...tool.Option) (string, error) {
        wasInterrupted, _, storedArgs := tool.GetInterruptState[string](ctx)
        
        if !wasInterrupted {
            return "", tool.StatefulInterrupt(ctx, &commontool.ApprovalInfo{
                ToolName:        tCtx.Name,
                ArgumentsInJSON: args,
            }, args)
        }
        
        isTarget, hasData, data := tool.GetResumeContext[*commontool.ApprovalResult](ctx)
        if isTarget && hasData {
            if data.Approved {
                return endpoint(ctx, storedArgs, opts...)
            }
            if data.DisapproveReason != nil {
                return fmt.Sprintf("tool '%s' disapproved: %s", tCtx.Name, *data.DisapproveReason), nil
            }
            return fmt.Sprintf("tool '%s' disapproved", tCtx.Name), nil
        }
        
        // 重新中断
        return "", tool.StatefulInterrupt(ctx, &commontool.ApprovalInfo{
            ToolName:        tCtx.Name,
            ArgumentsInJSON: storedArgs,
        }, storedArgs)
    }, nil
}

func (m *approvalMiddleware) WrapStreamableToolCall(
    _ context.Context,
    endpoint adk.StreamableToolCallEndpoint,
    tCtx *adk.ToolContext,
) (adk.StreamableToolCallEndpoint, error) {
    // 如果 agent 配置了 StreamingShell，则 execute 会走流式调用，需要实现该方法才能拦截到
    if tCtx.Name != "execute" {
        return endpoint, nil
    }
    return func(ctx context.Context, args string, opts ...tool.Option) (*schema.StreamReader[string], error) {
        wasInterrupted, _, storedArgs := tool.GetInterruptState[string](ctx)
        if !wasInterrupted {
            return nil, tool.StatefulInterrupt(ctx, &commontool.ApprovalInfo{
                ToolName:        tCtx.Name,
                ArgumentsInJSON: args,
            }, args)
        }

        isTarget, hasData, data := tool.GetResumeContext[*commontool.ApprovalResult](ctx)
        if isTarget && hasData {
            if data.Approved {
                return endpoint(ctx, storedArgs, opts...)
            }
            if data.DisapproveReason != nil {
                return singleChunkReader(fmt.Sprintf("tool '%s' disapproved: %s", tCtx.Name, *data.DisapproveReason)), nil
            }
            return singleChunkReader(fmt.Sprintf("tool '%s' disapproved", tCtx.Name)), nil
        }

        isTarget, _, _ = tool.GetResumeContext[any](ctx)
        if !isTarget {
            return nil, tool.StatefulInterrupt(ctx, &commontool.ApprovalInfo{
                ToolName:        tCtx.Name,
                ArgumentsInJSON: storedArgs,
            }, storedArgs)
        }

        return endpoint(ctx, storedArgs, opts...)
    }, nil
}
```

### CheckPointStore

`CheckPointStore` 是实现中断恢复的关键组件：

```go
type CheckPointStore interface {
    // 保存检查点
    Put(ctx context.Context, key string, checkpoint *Checkpoint) error
    
    // 获取检查点
    Get(ctx context.Context, key string) (*Checkpoint, error)
}
```

**为什么需要 CheckPointStore？**
- 中断时保存状态：Tool 参数、执行位置等
- 恢复时加载状态：从中断点继续执行
- 支持跨进程恢复：进程重启后仍可恢复

## Interrupt/Resume 的实现

### 1. 配置 Runner 使用 CheckPointStore

```go
runner := adk.NewRunner(ctx, adk.RunnerConfig{
    Agent:           agent,
    EnableStreaming: true,
    CheckPointStore: adkstore.NewInMemoryStore(),  // 内存存储
})
```

### 2. 配置 Agent 使用 ApprovalMiddleware

```go
agent, err := deep.New(ctx, &deep.Config{
    // ... 其他配置
    Handlers: []adk.ChatModelAgentMiddleware{
        &approvalMiddleware{},  // 添加审批中间件
        &safeToolMiddleware{},  // 将 Tool 错误转换为字符串（中断类错误会继续向上抛出）
    },
})
```

### 3. 处理中断事件

```go
checkPointID := sessionID

events := runner.Run(ctx, history, adk.WithCheckPointID(checkPointID))
content, interruptInfo, err := printAndCollectAssistantFromEvents(events)
if err != nil {
    return err
}

if interruptInfo != nil {
    // 注意：建议使用同一个 stdin reader 同时读取「用户输入」与「审批 y/n」
    // 避免审批输入被当成下一轮 you> 的消息
    content, err = handleInterrupt(ctx, runner, checkPointID, interruptInfo, reader)
    if err != nil {
        return err
    }
}

_ = session.Append(schema.AssistantMessage(content, nil))
```

## Interrupt/Resume 执行流程

```
┌─────────────────────────────────────────┐
│  用户：执行命令 echo hello               │
└─────────────────────────────────────────┘
                   ↓
        ┌──────────────────────┐
        │  Agent 分析意图       │
        │  决定调用 execute     │
        └──────────────────────┘
                   ↓
        ┌──────────────────────┐
        │  ApprovalMiddleware  │
        │  拦截 Tool 调用       │
        └──────────────────────┘
                   ↓
        ┌──────────────────────┐
        │  触发 Interrupt       │
        │  保存状态到 Store     │
        └──────────────────────┘
                   ↓
        ┌──────────────────────┐
        │  返回 Interrupt 事件  │
        │  等待用户审批         │
        └──────────────────────┘
                   ↓
        ┌──────────────────────┐
        │  用户输入 y/n         │
        └──────────────────────┘
                   ↓
        ┌──────────────────────┐
        │  runner.ResumeWith... │
        │  恢复执行             │
        └──────────────────────┘
                   ↓
        ┌──────────────────────┐
        │  执行 execute         │
        │  或返回拒绝信息       │
        └──────────────────────┘
```

## 本章小结

- **Interrupt**：Agent 的暂停机制，在关键操作前暂停等待确认
- **Resume**：恢复执行，用户确认后继续或拒绝后返回错误
- **ApprovalMiddleware**：通用审批中间件，拦截特定 Tool 调用
- **CheckPointStore**：保存中断状态，支持跨进程恢复
- **人机协作**：关键决策由人来确认，提高安全性

## 扩展思考

**其他 Interrupt 场景：**
- 多选项审批：用户选择多个选项之一
- 参数补全：用户提供缺失的参数
- 条件分支：用户决定执行路径

**审批策略：**
- 白名单：只审批敏感操作
- 黑名单：审批所有操作，除了安全的
- 动态规则：根据参数内容决定是否审批
