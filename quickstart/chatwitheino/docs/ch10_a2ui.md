---
title: "第十章：A2UI 协议（流式 UI 组件）"
---

本章目标：实现 A2UI 协议，将 Agent 的输出渲染为流式 UI 组件。

## 重要说明：A2UI 的边界

A2UI 并不属于 Eino 框架本身的范畴，它是一个业务层的 UI 协议/渲染方案。本章把 A2UI 集成进前面章节逐步构建出来的 Agent，是为了提供一个端到端、可落地的完整示例：从模型调用、工具调用、工作流编排，到最终把结果以更友好的 UI 方式呈现出来。

在真实业务场景中，你完全可以根据产品形态选择不同的 UI 形式，例如：

- Web / App：自定义组件、表格、卡片、图表等
- IM/办公套件：消息卡片、交互式表单
- 命令行：纯文本或 TUI（终端 UI）

Eino 更关注“可组合的智能执行与编排能力”，至于“如何呈现给用户”，属于业务层可以自由扩展的一环。

## 代码位置

- 入口代码（Runner 版）：[cmd/ch10/main.go](https://github.com/cloudwego/eino-examples/blob/main/quickstart/chatwitheino/cmd/ch10/main.go)
- A2UI 子集实现：[a2ui/types.go](https://github.com/cloudwego/eino-examples/blob/main/quickstart/chatwitheino/a2ui/types.go)
- A2UI 事件流转换：[a2ui/streamer.go](https://github.com/cloudwego/eino-examples/blob/main/quickstart/chatwitheino/a2ui/streamer.go)
- 前端页面：[static/index.html](https://github.com/cloudwego/eino-examples/blob/main/quickstart/chatwitheino/static/index.html)

## 前置条件

与第一章一致：需要配置一个可用的 ChatModel（OpenAI 或 Ark）

## 运行

在 `quickstart/chatwitheino` 目录下执行：

```bash
go run ./cmd/ch10/
```

输出示例：

```text
starting server on http://localhost:8080
```

### （可选）启用 ch09 的 skills 能力

最终 Web 版使用的 Agent 构建逻辑与 Chapter 9 对齐：当 `EINO_EXT_SKILLS_DIR` 指向一个合法 skills 目录时，会自动注册 `skill` 中间件，模型就能按需调用 `skill` 工具加载 `eino-guide` / `eino-component` / `eino-compose` / `eino-agent`。

```bash
go run ./scripts/sync_eino_ext_skills.go -src /path/to/eino-ext -dest ./skills/eino-ext -clean
EINO_EXT_SKILLS_DIR="$(pwd)/skills/eino-ext" go run ./cmd/ch10/
```

## 从文本到 UI：为什么需要 A2UI

前八章我们实现的 Agent 只输出文本，但现代 AI 应用需要更丰富的交互。

**纯文本输出的局限：**
- 无法展示结构化数据（表格、列表、卡片等）
- 无法实时更新（进度条、状态变化等）
- 无法嵌入交互元素（按钮、表单、链接等）
- 无法支持多媒体（图片、视频、音频等）

**A2UI 的定位：**
- **A2UI 是 Agent 到 UI 的协议**：定义了 Agent 输出如何映射到 UI 组件
- **A2UI 支持流式渲染**：组件可以实时更新，无需等待完整响应
- **A2UI 是声明式的**：Agent 只需声明"显示什么"，UI 负责渲染

**简单类比：**
- **纯文本输出** = "终端命令行"（只能显示文本）
- **A2UI** = "Web 应用"（可以显示任何 UI 组件）

## 关键概念

### A2UI v0.8 子集（本示例的边界）

本 quickstart 并没有实现一个“完整的 A2UI 标准库”，而是实现了一个 **A2UI v0.8 的子集**：目标是把 Agent 的事件流，以稳定、可增量渲染的 UI 组件树方式推给浏览器。

当前实现的 A2UI 消息类型与组件类型，以 [a2ui/types.go](https://github.com/cloudwego/eino-examples/blob/main/quickstart/chatwitheino/a2ui/types.go) 为准。

### A2UI 消息：BeginRendering / SurfaceUpdate / DataModelUpdate / InterruptRequest

每一行 SSE（`data: {...}`）承载一个 A2UI Message，Message 是一个“信封结构”，每次只会出现一个字段：

**关键代码片段（**注意：这是简化后的代码片段，不能直接运行，完整代码请参考** [a2ui/types.go](https://github.com/cloudwego/eino-examples/blob/main/quickstart/chatwitheino/a2ui/types.go)）：**

```go
type Message struct {
    BeginRendering   *BeginRenderingMsg
    SurfaceUpdate    *SurfaceUpdateMsg
    DataModelUpdate  *DataModelUpdateMsg
    DeleteSurface    *DeleteSurfaceMsg
    InterruptRequest *InterruptRequestMsg
}
```

其中：

- `BeginRendering`：告诉前端“开始渲染一个 surface（会话）”，并指定根节点 ID
- `SurfaceUpdate`：新增/更新一批组件（组件是一个树，用 `id` 互相引用）
- `DataModelUpdate`：更新 data bindings（用于把流式文本增量更新到某个 Text 组件）
- `InterruptRequest`：当 Agent 触发 interrupt（例如审批）时，通知前端展示批准/拒绝入口

### A2UI 组件：Text / Column / Card / Row

本示例 UI 组件只实现了 4 种（见 [a2ui/types.go](https://github.com/cloudwego/eino-examples/blob/main/quickstart/chatwitheino/a2ui/types.go)）：

- `Text`：文本渲染（支持 `usageHint` 区分 caption/body/title）；当 `dataKey` 存在时，文本来自 `DataModelUpdate`
- `Column` / `Row`：布局（children 是组件 ID 列表）
- `Card`：卡片容器（children 是组件 ID 列表）

## A2UI 的实现：把 AgentEvent 转成 A2UI SSE

最终 Web 版的核心链路是：

- 后端运行 Agent，得到 `*adk.AsyncIterator[*adk.AgentEvent]`
- 把事件流转换为 A2UI JSONL/SSE 流输出给浏览器（见 [a2ui/streamer.go](https://github.com/cloudwego/eino-examples/blob/main/quickstart/chatwitheino/a2ui/streamer.go)）
- 前端解析 SSE 的 `data:` 行并渲染组件树（见 [static/index.html](https://github.com/cloudwego/eino-examples/blob/main/quickstart/chatwitheino/static/index.html)）

### 服务端路由（高层）

与 A2UI 相关的关键接口（见 [cmd/ch10/main.go](https://github.com/cloudwego/eino-examples/blob/main/quickstart/chatwitheino/cmd/ch10/main.go)）：

- `GET /`：返回前端页面 `static/index.html`
- `POST /sessions/:id/chat`：返回 SSE 流（A2UI messages），把 Agent 运行结果边跑边渲染到 UI
- `GET /sessions/:id/render`：返回 JSONL（A2UI messages），用于“选中会话时回放历史”
- `POST /sessions/:id/approve`：处理 interrupt 的批准/拒绝并继续返回 SSE 流

### 事件流转换（高层）

服务端把 `Runner.Run(...)` 的事件流交给 `a2ui.StreamToWriter(...)`，后者负责：

- 对 user/assistant/tool 的输出做拆分
- 把 tool call / tool result 渲染成 “chip 卡片”
- 把 assistant 的流式 token 做成 `DataModelUpdate`，实现“边生成边渲染”
- 遇到 interrupt 时发送 `InterruptRequest`，并暂停等待人类批准

## 前端集成：fetch + SSE（不是 WebSocket）

- 前端通过 `fetch('/sessions/:id/chat')` 发起请求，然后从 `res.body` 读取流式字节，按行切分并解析 `data: {...}` 的 JSON（见 [static/index.html](https://github.com/cloudwego/eino-examples/blob/main/quickstart/chatwitheino/static/index.html)）。

**关键代码片段（**注意：这是简化后的代码片段，不能直接运行，完整代码请参考** [static/index.html](https://github.com/cloudwego/eino-examples/blob/main/quickstart/chatwitheino/static/index.html)）：**

```javascript
const res = await fetch(`/sessions/${id}/chat`, {
  method: 'POST',
  headers: {'Content-Type': 'application/json'},
  body: JSON.stringify({message}),
});

const reader = res.body.getReader();
const decoder = new TextDecoder();
let buffer = '';
while (true) {
  const {done, value} = await reader.read();
  if (done) break;
  buffer += decoder.decode(value, {stream: true});
  const lines = buffer.split('\n');
  buffer = lines.pop();
  for (const line of lines) {
    const trimmed = line.trim();
    if (trimmed.startsWith('data:')) {
      const jsonStr = trimmed.slice(5).trimStart();
      processA2UIMessage(JSON.parse(jsonStr));
    }
  }
}
```

## A2UI 流式渲染流程（概览）

```
┌─────────────────────────────────────────┐
│  用户：分析这个文件                       │
└─────────────────────────────────────────┘
                   ↓
        ┌──────────────────────┐
        │  Agent 开始处理       │
        │  A2UI: AddText       │
        │  "正在分析..."         │
        └──────────────────────┘
                   ↓
        ┌──────────────────────┐
        │  调用 Tool           │
        │  A2UI: AddProgress   │
        │  进度: 0%            │
        └──────────────────────┘
                   ↓
        ┌──────────────────────┐
        │  Tool 执行中          │
        │  A2UI: UpdateProgress│
        │  进度: 50%           │
        └──────────────────────┘
                   ↓
        ┌──────────────────────┐
        │  Tool 完成            │
        │  A2UI: tool result    │
        └──────────────────────┘
                   ↓
        ┌──────────────────────┐
        │  显示结果             │
        │  A2UI: DataModelUpdate│
        │  （流式更新 assistant）│
        └──────────────────────┘
```

## 本章小结

- **A2UI**：Agent 到 UI 的协议，定义了 Agent 输出如何映射到 UI 组件
- **子集实现**：本示例只实现了 Text/Column/Card/Row 与 data binding
- **流式输出**：后端以 SSE 推送 A2UI JSONL，前端增量渲染组件树
- **事件到 UI**：把 `AgentEvent` 转为 `tool call / tool result / assistant stream` 的可视化输出

## 下一步

本章的 `cmd/ch10` 使用 `adk.Runner` 实现了完整的 Web 应用。但 Runner 是"一次性"模型——如果用户在 Agent 回答到一半时发出新问题，Runner 没有内置机制来取消当前执行并切换到新输入。

下一章将引入 `adk.TurnLoop`，为 Agent 增加 **抢占（Preempt）** 和 **中止（Abort）** 能力。
