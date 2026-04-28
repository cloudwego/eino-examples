---
title: "第三章：Memory 与 Session（持久化对话）"
---

本章目标：实现对话历史的持久化存储，支持跨进程恢复会话。

> **⚠️ 重要说明：业务层概念 vs 框架概念**
>
> 本章介绍的 **Memory、Session、Store 是业务层概念**，**不是 Eino 框架的核心组件**。
>
> - **Eino 框架层面**：只提供 `adk.Runner` 和 `schema.Message` 等基础抽象，框架本身不关心对话历史的存储方式
> - **业务层层面**：Memory/Session/Store 是本示例项目为了实现持久化对话而设计的业务逻辑，通过组装给 `adk.Runner` 的输入来与 Eino 框架交互
>
> 换句话说，Eino 框架只负责"如何处理消息"，而"如何存储消息"完全由业务层决定。本章提供的实现只是一个简单的参考示例，你可以根据自己的业务需求选择完全不同的存储方案（数据库、Redis、云存储等）。

## 代码位置

- 入口代码：[cmd/ch03/main.go](https://github.com/cloudwego/eino-examples/blob/main/quickstart/chatwitheino/cmd/ch03/main.go)
- Memory 实现：[mem/store.go](https://github.com/cloudwego/eino-examples/blob/main/quickstart/chatwitheino/mem/store.go)

## 前置条件

与第一章一致：需要配置一个可用的 ChatModel（OpenAI 或 Ark）。

## 运行

在 `examples/quickstart/chatwitheino` 目录下执行：

```bash
# 创建新会话
go run ./cmd/ch03

# 恢复已有会话
go run ./cmd/ch03 --session <session-id>
```

输出示例：

```text
Created new session: 083d16da-6b13-4fe6-afb0-c45d8f490ce1
Session title: New Session
Enter your message (empty line to exit):
you> 你好，我是张三
[assistant] 你好张三！很高兴认识你...
you> 我叫什么名字？
[assistant] 你叫张三...

Session saved: 083d16da-6b13-4fe6-afb0-c45d8f490ce1
Resume with: go run ./cmd/ch03 --session 083d16da-6b13-4fe6-afb0-c45d8f490ce1
```

## 从内存到持久化：为什么需要 Memory

第二章我们实现了多轮对话，但有一个问题：**对话历史只存在于内存中**。

**内存存储的局限：**
- 进程退出后，对话历史丢失
- 无法跨设备、跨进程恢复会话
- 无法实现会话管理（列表、删除、搜索等）

**Memory 的定位：**
- **Memory 是对话历史的持久化存储**：将对话保存到磁盘或数据库
- **Memory 支持 Session 管理**：每个 Session 代表一次完整的对话
- **Memory 与 Agent 解耦**：Agent 不关心存储细节，只关心消息列表

**简单类比：**
- **内存存储** = "草稿纸"（进程退出就没了）
- **Memory** = "笔记本"（永久保存，随时翻阅）

## 关键概念

> **再次强调**：以下 Session、Store 等概念都是**业务层实现**，用于管理对话历史的存储。Eino 框架本身不提供这些组件，而是由业务层负责管理消息列表，然后将消息传递给 `adk.Runner` 进行处理。

### Session（业务层概念）

`Session` 代表一次完整的对话会话：

```go
type Session struct {
    ID        string
    CreatedAt time.Time

    messages []*schema.Message  // 对话历史
    // ...
}
```

**核心方法：**
- `Append(msg)`：追加消息到会话，并持久化
- `GetMessages()`：获取所有消息
- `Title()`：从第一条用户消息生成会话标题

### Store（业务层概念）

`Store` 管理多个 Session 的持久化存储：

```go
type Store struct {
    dir   string              // 存储目录
    cache map[string]*Session // 内存缓存
}
```

**核心方法：**
- `GetOrCreate(id)`：获取或创建 Session
- `List()`：列出所有 Session
- `Delete(id)`：删除 Session

### JSONL 文件格式

每个 Session 存储为一个 `.jsonl` 文件：

```jsonl
{"type":"session","id":"083d16da-...","created_at":"2026-03-11T10:00:00Z"}
{"role":"user","content":"你好，我是谁？"}
{"role":"assistant","content":"你好！我暂时不知道你是谁..."}
{"role":"user","content":"我叫张三"}
{"role":"assistant","content":"好的，张三，很高兴认识你！"}
```

**为什么用 JSONL？**
- **简单**：每行一个 JSON 对象，易于读写
- **可扩展**：可以追加新消息，无需重写整个文件
- **可读性好**：可以用文本编辑器直接查看
- **容错性强**：单行损坏不影响其他行

## Memory 的实现（业务层示例）

以下是一个简单的业务层实现示例，使用 JSONL 文件存储对话历史。这只是众多可能实现中的一种，你可以根据实际需求选择数据库、Redis 等其他存储方案。

### 1. 创建 Store

```go
sessionDir := "./data/sessions"
store, err := mem.NewStore(sessionDir)
if err != nil {
    log.Fatal(err)
}
```

### 2. 获取或创建 Session

```go
sessionID := "083d16da-6b13-4fe6-afb0-c45d8f490ce1"
session, err := store.GetOrCreate(sessionID)
if err != nil {
    log.Fatal(err)
}
```

### 3. 追加用户消息

```go
userMsg := schema.UserMessage("你好")
if err := session.Append(userMsg); err != nil {
    log.Fatal(err)
}
```

### 4. 获取历史并调用 Agent

```go
history := session.GetMessages()
events := runner.Run(ctx, history)
content := printAndCollectAssistantFromEvents(events)
```

### 5. 追加助手消息

```go
assistantMsg := schema.AssistantMessage(content, nil)
if err := session.Append(assistantMsg); err != nil {
    log.Fatal(err)
}
```

**关键代码片段（**注意：这是简化后的代码片段，不能直接运行，完整代码请参考** [cmd/ch03/main.go](https://github.com/cloudwego/eino-examples/blob/main/quickstart/chatwitheino/cmd/ch03/main.go)）：

```go
// 创建或恢复 Session
session, err := store.GetOrCreate(sessionID)
if err != nil {
    log.Fatal(err)
}

// 用户输入
userMsg := schema.UserMessage(line)
if err := session.Append(userMsg); err != nil {
    log.Fatal(err)
}

// 调用 Agent
history := session.GetMessages()
events := runner.Run(ctx, history)
content := printAndCollectAssistantFromEvents(events)

// 保存助手回复
assistantMsg := schema.AssistantMessage(content, nil)
if err := session.Append(assistantMsg); err != nil {
    log.Fatal(err)
}
```

## Session 与 Agent 的关系：业务层与框架层的协作

**关键理解：**
- **Session 是业务层概念**：由业务代码实现和管理，负责存储和加载对话历史
- **Agent（Runner）是框架层概念**：由 Eino 框架提供，负责处理消息并生成回复
- **两者的交互点**：业务层通过 `session.GetMessages()` 获取消息列表，传递给 `runner.Run(ctx, history)` 进行处理

**架构分层：**

```
┌─────────────────────────────────────────────────────────────┐
│                     业务层（你的代码）                          │
│  ┌─────────────┐    ┌──────────────┐    ┌───────────────┐  │
│  │   Session   │───→│ GetMessages() │───→│ runner.Run()  │  │
│  │   (存储)    │    │  (消息列表)    │    │  (框架调用)    │  │
│  └─────────────┘    └──────────────┘    └───────────────┘  │
│         ↑                                      │            │
│         │                                      ↓            │
│  ┌─────────────┐                      ┌───────────────┐    │
│  │   Append()  │←─────────────────────│  助手回复      │    │
│  │  (保存消息)  │                      └───────────────┘    │
│  └─────────────┘                                           │
└─────────────────────────────────────────────────────────────┘
                              │
                              ↓
┌─────────────────────────────────────────────────────────────┐
│                   框架层（Eino 框架）                          │
│  ┌───────────────────────────────────────────────────────┐  │
│  │  adk.Runner：接收消息列表，调用 ChatModel，返回回复     │  │
│  └───────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

**流程图：**

```
┌─────────────────────────────────────────┐
│  用户输入                                 │
└─────────────────────────────────────────┘
                   ↓
        ┌──────────────────────┐
        │  session.Append()    │
        │  保存用户消息         │
        └──────────────────────┘
                   ↓
        ┌──────────────────────┐
        │  session.GetMessages()│
        │  获取完整历史         │
        └──────────────────────┘
                   ↓
        ┌──────────────────────┐
        │  runner.Run(history) │
        │  Agent 处理消息       │
        └──────────────────────┘
                   ↓
        ┌──────────────────────┐
        │  收集助手回复         │
        └──────────────────────┘
                   ↓
        ┌──────────────────────┐
        │  session.Append()    │
        │  保存助手消息         │
        └──────────────────────┘
```

## 本章小结

**框架层 vs 业务层：**
- **Eino 框架层**：提供 `adk.Runner`、`schema.Message` 等基础抽象，不关心消息如何存储
- **业务层（本章实现）**：Memory/Session/Store 是业务层概念，用于管理对话历史的存储

**业务层概念：**
- **Memory**：对话历史的持久化存储，支持跨进程恢复
- **Session**：一次完整的对话会话，包含 ID、创建时间、消息列表
- **Store**：管理多个 Session 的存储，支持创建、获取、列表、删除
- **JSONL 格式**：简单的文件格式，易于读写和扩展

**业务层与框架层的交互：**
- 业务层负责存储消息，通过 `session.GetMessages()` 获取消息列表
- 将消息列表传递给框架层的 `runner.Run(ctx, history)` 进行处理
- 收集框架层返回的回复，再由业务层保存到存储中

> **💡 提示**：本章的实现只是众多存储方案中的一种简单示例。在实际项目中，你可以根据业务需求选择数据库、Redis、云存储等方案，甚至可以实现更复杂的功能如会话过期清理、搜索、分享等。

## 扩展思考：业务层存储方案的选择

本章提供的 JSONL 文件存储方案适合简单的单机应用。在实际业务中，你可能需要考虑其他存储方案：

**其他存储实现：**
- 数据库存储（MySQL、PostgreSQL、MongoDB）
- Redis 存储（支持分布式）
- 云存储（S3、OSS）

**高级功能：**
- 会话过期清理
- 会话搜索
- 会话导出/导入
- 会话分享
