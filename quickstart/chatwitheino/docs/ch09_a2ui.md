---
title: "第九章：A2UI 协议（流式 UI 组件）"
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

- 入口代码：[main.go](https://github.com/cloudwego/eino/blob/main/examples/quickstart/chatwitheino/main.go)
- A2UI 实现：[a2ui/streamer.go](https://github.com/cloudwego/eino/blob/main/examples/quickstart/chatwitheino/a2ui/streamer.go)

## 前置条件

与第一章一致：需要配置一个可用的 ChatModel（OpenAI 或 Ark）

## 运行

在 `examples/quickstart/chatwitheino` 目录下执行：

```bash
go run .
```

输出示例：

```text
starting server on http://localhost:8080
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

### A2UI 组件

A2UI 定义了一系列 UI 组件类型：

```go
type ComponentType string

const (
    ComponentText      ComponentType = "text"       // 文本
    ComponentMarkdown  ComponentType = "markdown"   // Markdown
    ComponentCode      ComponentType = "code"       // 代码块
    ComponentImage     ComponentType = "image"      // 图片
    ComponentTable     ComponentType = "table"      // 表格
    ComponentCard      ComponentType = "card"       // 卡片
    ComponentButton    ComponentType = "button"     // 按钮
    ComponentForm      ComponentType = "form"       // 表单
    ComponentProgress  ComponentType = "progress"   // 进度条
    ComponentDivider   ComponentType = "divider"    // 分隔线
)
```

### A2UI 消息

每条 A2UI 消息包含：

```go
type Message struct {
    ID        string        // 消息 ID
    Role      string        // user / assistant
    Components []Component  // UI 组件列表
    Timestamp time.Time     // 时间戳
}
```

### A2UI 流式输出

A2UI 支持流式输出组件：

```go
type StreamMessage struct {
    Type      string      // add / update / delete
    Index     int         // 组件索引
    Component Component   // 组件内容
}
```

**流式更新类型：**
- `add`：添加新组件
- `update`：更新已有组件
- `delete`：删除组件

## A2UI 的实现

### 1. 创建 A2UI Streamer

```go
streamer := a2ui.NewStreamer()
```

### 2. 添加组件

```go
// 添加文本组件
streamer.AddText("正在处理您的请求...")

// 添加进度条
streamer.AddProgress(0, 100, "加载中")

// 更新进度
streamer.UpdateProgress(0, 50, "处理中")

// 添加代码块
streamer.AddCode("go", `fmt.Println("Hello, World!")`)

// 添加表格
streamer.AddTable([][]string{
    {"Name", "Age", "City"},
    {"Alice", "30", "New York"},
    {"Bob", "25", "London"},
})
```

### 3. 流式输出

```go
// 获取流式消息
stream := streamer.Stream()

for {
    msg, ok := stream.Next()
    if !ok {
        break
    }
    // 发送到前端
    sendToClient(msg)
}
```

**关键代码片段（**注意：这是简化后的代码片段，不能直接运行，完整代码请参考** [cmd/ch09/main.go](https://github.com/cloudwego/eino/blob/main/examples/quickstart/chatwitheino/cmd/ch09/main.go)）：

```go
// 创建 A2UI Streamer
streamer := a2ui.NewStreamer()

// Agent 执行过程中添加组件
streamer.AddText("我来帮你分析这个文件...")

// 调用 Tool
streamer.AddProgress(0, 0, "读取文件")
result, err := tool.Run(ctx, args)
streamer.UpdateProgress(0, 100, "完成")

// 显示结果
streamer.AddCode("json", result)

// 流式输出
stream := streamer.Stream()
for {
    msg, ok := stream.Next()
    if !ok {
        break
    }
    wsConn.WriteJSON(msg)
}
```

## A2UI 与 Agent 的集成

### 在 Agent 中使用 A2UI

```go
func buildAgent(ctx context.Context) (adk.Agent, error) {
    return deep.New(ctx, &deep.Config{
        Name:        "A2UIAgent",
        Description: "Agent with A2UI streaming output",
        ChatModel:   cm,
        Backend:     backend,
        // 配置 A2UI Streamer
        StreamingShell: backend,
    })
}
```

### 在 Runner 中使用 A2UI

```go
runner := adk.NewRunner(ctx, adk.RunnerConfig{
    Agent:           agent,
    EnableStreaming: true,
})

// 执行 Agent
events := runner.Run(ctx, history)

// 将事件转换为 A2UI 组件
streamer := a2ui.NewStreamer()
for {
    event, ok := events.Next()
    if !ok {
        break
    }
    if event.Output != nil && event.Output.MessageOutput != nil {
        // 添加文本组件
        streamer.AddText(event.Output.MessageOutput.Message.Content)
    }
}
```

## A2UI 流式渲染流程

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
        │  A2UI: UpdateProgress│
        │  进度: 100%          │
        └──────────────────────┘
                   ↓
        ┌──────────────────────┐
        │  显示结果             │
        │  A2UI: AddCode       │
        │  代码块               │
        └──────────────────────┘
```

## 前端集成

### WebSocket 连接

```javascript
const ws = new WebSocket('ws://localhost:8080/ws');

ws.onmessage = (event) => {
    const msg = JSON.parse(event.data);
    renderComponent(msg);
};

function renderComponent(msg) {
    const { type, index, component } = msg;
    
    switch (component.type) {
        case 'text':
            renderText(component.content);
            break;
        case 'code':
            renderCode(component.language, component.content);
            break;
        case 'progress':
            renderProgress(component.value, component.max, component.label);
            break;
        // ...
    }
}
```

## 本章小结

- **A2UI**：Agent 到 UI 的协议，定义了 Agent 输出如何映射到 UI 组件
- **组件类型**：文本、Markdown、代码、图片、表格、卡片、按钮、表单、进度条等
- **流式输出**：支持实时添加、更新、删除组件
- **声明式**：Agent 只需声明"显示什么"，UI 负责渲染
- **前端集成**：通过 WebSocket 实现实时通信

## 系列收尾：这个 Quickstart Agent 的完整愿景

到本章为止，我们用一个可以实际运行的 Agent 串起了 Eino 的核心能力。你可以把它理解为一个可扩展的“端到端 Agent 应用骨架”：

- 运行时：Runner 驱动执行，支持流式输出与事件模型
- 工具层：Filesystem / Shell 等 Tool 能力接入，工具错误可被安全处理
- 中间件：可插拔的 middleware/handler，用于错误处理、重试、审批等横切能力
- 可观测：callbacks/trace 能力把关键链路打通，便于调试与线上观测
- 人机协作：interrupt/resume + checkpoint 支持审批、补参、分支选择等交互式流程
- 确定性编排：compose（graph/chain/workflow）把复杂业务流程组织为可维护、可复用的执行图
- 业务交付：像 A2UI 这样的 UI 集成，属于业务层自由选择的一环，用来把 Agent 能力以合适的产品形态呈现给用户

你可以在这个骨架上逐步替换/扩展任意环节：模型、工具、存储、工作流、前端渲染协议，而不需要推倒重来。

## 扩展思考

**其他组件类型：**
- 图表组件（折线图、柱状图、饼图）
- 地图组件
- 时间线组件
- 树形组件
- 标签页组件

**高级功能：**
- 组件交互（点击、拖拽、输入）
- 条件渲染
- 组件动画
- 响应式布局
