---
title: "Message Kind：同时支持 Message 与 AgenticMessage"
---

本示例默认使用 `*schema.Message`，也可以在运行时切换到 `*schema.AgenticMessage`。两种消息类型由同一套泛型代码承载，业务代码通过 `MESSAGE_KIND` 选择具体实例：

```bash
# 默认：MESSAGE_KIND=message
go run .

# 使用 AgenticMessage
export MESSAGE_KIND=agentic
go run .
```

## 环境变量

| 变量 | 说明 |
|------|------|
| `MESSAGE_KIND` | `message` 或 `agentic`，默认 `message` |
| `SESSION_DIR` | `message` 会话目录，默认 `./data/sessions` |
| `SESSION_DIR_AGENTIC` | `agentic` 会话目录，默认 `./data/sessions_agentic` |
| `MODEL_TYPE` | `openai` 或 `ark`，默认 `openai` |
| `OPENAI_API_KEY` / `OPENAI_MODEL` / `OPENAI_BASE_URL` | OpenAI 模型配置 |
| `ARK_API_KEY` / `ARK_MODEL` | Ark 模型配置 |

`AgenticMessage` 模式会使用 eino-ext 官方的 agentic model 实现，例如 `agenticopenai` 或 `agenticark`；`Message` 模式继续使用普通 `openai` 或 `ark` ChatModel。

## 存储策略

两种消息类型不做互相转换，采用隔离存储：

- `message`：默认写入 `./data/sessions`
- `agentic`：默认写入 `./data/sessions_agentic`

每个 session 文件的 header 会记录 `message_kind`。旧的 `message` session 文件如果没有这个字段，仍按 `message` 读取；如果当前运行模式与文件 header 不一致，会拒绝加载或在列表中跳过。这样用户可以先用 `message` 跑，再用 `agentic` 跑同一个应用，而不会把两种 JSON 结构混在一起。

## 代码结构

泛型改造尽量复用 Eino 官方接口，不重复实现框架能力：

- `chatmodel.NewModel[M]`：chatwitheino 局部模型工厂，根据 `M` 和 `MODEL_TYPE` 创建 `model.BaseModel[M]`
- `msgops`：集中封装“创建用户消息、读取文本、读取 tool call/tool result”等两种消息类型的公共操作
- `mem.Store[M]` / `mem.Session[M]`：按消息类型保存和读取会话
- `rag.BuildTool[M]`：RAG tool 内部复用同一个泛型 ChatModel
- `a2ui.StreamToWriter[M]` / `StreamContinue[M]` / `RenderHistory[M]`：把 typed agent events 渲染为 A2UI
- `server.Server[M]`：Web/TurnLoop 层使用 `adk.TurnLoop[*ChatItem, M]`

使用的官方泛型 API 包括 `adk.NewTypedRunner[M]`、`adk.NewTypedChatModelAgent[M]`、`deep.NewTyped[M]`、`skill.NewTyped[M]`、`model.BaseModel[M]` 和 `adk.TurnLoop[T, M]`。

## 教程章节

`cmd/ch01` 到 `cmd/ch10` 都支持相同的运行时开关：

```bash
go run ./cmd/ch08
export MESSAGE_KIND=agentic
go run ./cmd/ch08
```

如果要恢复会话，请使用对应模式创建出的 session id，并让 `MESSAGE_KIND` 与创建时保持一致。由于默认目录已经隔离，普通运行时不需要手动切换目录。

切回默认 `message` 模式时，可以执行：

```bash
unset MESSAGE_KIND
```
