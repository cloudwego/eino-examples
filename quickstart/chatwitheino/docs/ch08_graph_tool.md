---
title: "第八章：Graph Tool（复杂工作流）"
---

本章目标：理解 Graph Tool 的概念，实现大文件的并行 chunk 召回，引入 compose 包构建复杂工作流。

## 代码位置

- 入口代码：[cmd/ch08/main.go](https://github.com/cloudwego/eino-examples/blob/main/quickstart/chatwitheino/cmd/ch08/main.go)
- RAG 实现：[rag/rag.go](https://github.com/cloudwego/eino-examples/blob/main/quickstart/chatwitheino/rag/rag.go)

## 前置条件

与第一章一致：需要配置一个可用的 ChatModel（OpenAI 或 Ark）。

## 运行

在 `examples/quickstart/chatwitheino` 目录下执行：

```bash
# 设置项目根目录
export PROJECT_ROOT=/path/to/your/project

go run ./cmd/ch08
```

输出示例：

```text
you> 请帮我分析 RFC6455 文档中关于 WebSocket 握手的部分
[assistant] 我来帮你分析文档...
[tool call] answer_from_document(file_path: "rfc6455.txt", question: "WebSocket 握手过程")
[tool result] 找到 3 个相关片段，正在生成答案...
[assistant] 根据 RFC6455 文档，WebSocket 握手过程如下...
```

## 从简单 Tool 到 Graph Tool：为什么需要复杂工作流

第四章我们创建了简单的 Tool，每个 Tool 执行单一任务。但实际场景中，很多任务需要多个步骤协同完成。

**简单 Tool 的局限：**
- 单一职责：每个 Tool 只做一件事
- 无法并行：多个独立任务无法同时执行
- 难以复用：复杂逻辑难以拆分和组合

**重要说明：本章只是展示 compose/graph/workflow 能力的一角。**

从更大的视角看，Eino 的 `compose` 包提供了非常通用、确定性的编排能力：你可以把任何需要“确定性业务流程”的系统，用 `compose` 的 Graph/Chain/Workflow 组织成可执行的流水线，并且它能够**原生编排 Eino 的所有 component**（如 ChatModel、Prompt、Tools、Retriever、Embedding、Indexer 等），同时具备完整的 **callback** 体系，以及 **interrupt/resume + checkpoint** 支持。

**Graph Tool 的定位：**
- **Graph Tool 是 compose 工作流的 Tool 化封装**：把 `compose.Graph / compose.Chain / compose.Workflow` 这类可编译的编排产物，包装成一个 Agent 可调用的 Tool
- **支持并行/分支/组合**：由 compose 提供（并行、分支、字段映射、子图等），Graph Tool 只是把它们暴露为 Tool 入口
- **支持状态管理与持久化**：节点间传递数据、以及通过 checkpoint 保存/恢复运行状态
- **可中断恢复**：既支持工作流内部的中断（节点里触发 interrupt），也支持工具层面的中断包装（嵌套 interrupt 场景）

**简单类比：**
- **简单 Tool** = "单步操作"（读取文件）
- **Graph Tool** = "流水线"（读取 → 分块 → 评分 → 筛选 → 生成答案）

## 关键概念

### compose.Workflow

`compose.Workflow` 是 Eino 中构建工作流的核心组件：

```go
wf := compose.NewWorkflow[Input, Output]()

// 添加节点
wf.AddLambdaNode("load", loadFunc).AddInput(compose.START)
wf.AddLambdaNode("chunk", chunkFunc).AddInput("load")
wf.AddLambdaNode("score", scoreFunc).AddInput("chunk")
wf.AddLambdaNode("answer", answerFunc).AddInput("score")

// 连接到结束节点
wf.End().AddInput("answer")
```

**核心概念：**
- **Node**：工作流中的处理单元
- **Edge**：节点间的数据流向
- **START**：工作流的入口
- **END**：工作流的出口

### BatchNode

`BatchNode` 用于并行处理多个任务：

```go
scorer := batch.NewBatchNode(&batch.NodeConfig[Task, Result]{
    Name:           "ChunkScorer",
    InnerTask:      scoreOneChunk,  // 单个任务的处理函数
    MaxConcurrency: 5,              // 最大并发数
})
```

**工作原理：**
1. 接收任务列表作为输入
2. 并行执行每个任务（受 MaxConcurrency 限制）
3. 收集所有结果返回

### FieldMapping

`FieldMapping` 用于跨节点传递数据：

```go
wf.AddLambdaNode("answer", answerFunc).
    AddInputWithOptions("filter",  // 从 filter 节点获取数据
        []*compose.FieldMapping{compose.ToField("TopK")},
        compose.WithNoDirectDependency()).
    AddInputWithOptions(compose.START,  // 从 START 节点获取数据
        []*compose.FieldMapping{compose.MapFields("Question", "Question")},
        compose.WithNoDirectDependency())
```

**为什么需要 FieldMapping？**
- 非相邻节点间传递数据
- 多个数据源合并到同一节点
- 数据字段重命名

## Graph Tool 的实现

### 1. 定义输入输出结构

```go
type Input struct {
    FilePath string `json:"file_path" jsonschema:"description=Absolute path to the document"`
    Question string `json:"question"  jsonschema:"description=The question to answer"`
}

type Output struct {
    Answer  string   `json:"answer"`
    Sources []string `json:"sources"`
}
```

### 2. 构建工作流

```go
func buildWorkflow(cm model.BaseChatModel) *compose.Workflow[Input, Output] {
    wf := compose.NewWorkflow[Input, Output]()

    // load: 读取文件
    wf.AddLambdaNode("load", compose.InvokableLambda(
        func(ctx context.Context, in Input) ([]*schema.Document, error) {
            data, err := os.ReadFile(in.FilePath)
            if err != nil {
                return nil, err
            }
            return []*schema.Document{{Content: string(data)}}, nil
        },
    )).AddInput(compose.START)

    // chunk: 分块
    wf.AddLambdaNode("chunk", compose.InvokableLambda(
        func(ctx context.Context, docs []*schema.Document) ([]*schema.Document, error) {
            var out []*schema.Document
            for _, d := range docs {
                out = append(out, splitIntoChunks(d.Content, 800)...)
            }
            return out, nil
        },
    )).AddInput("load")

    // score: 并行评分
    scorer := batch.NewBatchNode(&batch.NodeConfig[scoreTask, scoredChunk]{
        Name:           "ChunkScorer",
        InnerTask:      newScoreWorkflow(cm),
        MaxConcurrency: 5,
    })

    wf.AddLambdaNode("score", compose.InvokableLambda(
        func(ctx context.Context, in scoreIn) ([]scoredChunk, error) {
            tasks := make([]scoreTask, len(in.Chunks))
            for i, c := range in.Chunks {
                tasks[i] = scoreTask{Text: c.Content, Question: in.Question}
            }
            return scorer.Invoke(ctx, tasks)
        },
    )).
        AddInputWithOptions("chunk", []*compose.FieldMapping{compose.ToField("Chunks")}, compose.WithNoDirectDependency()).
        AddInputWithOptions(compose.START, []*compose.FieldMapping{compose.MapFields("Question", "Question")}, compose.WithNoDirectDependency())

    // filter: 筛选 top-k
    wf.AddLambdaNode("filter", compose.InvokableLambda(
        func(ctx context.Context, scored []scoredChunk) ([]scoredChunk, error) {
            sort.Slice(scored, func(i, j int) bool {
                return scored[i].Score > scored[j].Score
            })
            // 返回 top-3
            if len(scored) > 3 {
                scored = scored[:3]
            }
            return scored, nil
        },
    )).AddInput("score")

    // answer: 生成答案
    wf.AddLambdaNode("answer", compose.InvokableLambda(
        func(ctx context.Context, in synthIn) (Output, error) {
            return synthesize(ctx, cm, in)
        },
    )).
        AddInputWithOptions("filter", []*compose.FieldMapping{compose.ToField("TopK")}, compose.WithNoDirectDependency()).
        AddInputWithOptions(compose.START, []*compose.FieldMapping{compose.MapFields("Question", "Question")}, compose.WithNoDirectDependency())

    wf.End().AddInput("answer")

    return wf
}
```

### 3. 封装为 Tool

```go
func BuildTool(ctx context.Context, cm model.BaseChatModel) (tool.BaseTool, error) {
    wf := buildWorkflow(cm)
    return graphtool.NewInvokableGraphTool[Input, Output](
        wf,
        "answer_from_document",
        "Search a large document for relevant content and synthesize an answer.",
    )
}
```

**关键代码片段（**注意：这是简化后的代码片段，不能直接运行，完整代码请参考** [rag/rag.go](https://github.com/cloudwego/eino-examples/blob/main/quickstart/chatwitheino/rag/rag.go)）：

```go
// 构建工作流
wf := compose.NewWorkflow[Input, Output]()

// 添加节点
wf.AddLambdaNode("load", loadFunc).AddInput(compose.START)
wf.AddLambdaNode("chunk", chunkFunc).AddInput("load")
wf.AddLambdaNode("score", scoreFunc).
    AddInputWithOptions("chunk", []*compose.FieldMapping{compose.ToField("Chunks")}, compose.WithNoDirectDependency()).
    AddInputWithOptions(compose.START, []*compose.FieldMapping{compose.MapFields("Question", "Question")}, compose.WithNoDirectDependency())

// 封装为 Tool
return graphtool.NewInvokableGraphTool[Input, Output](wf, "answer_from_document", "...")
```

## Graph Tool 执行流程

```
┌─────────────────────────────────────────┐
│  输入：file_path, question               │
└─────────────────────────────────────────┘
                   ↓
        ┌──────────────────────┐
        │  load: 读取文件       │
        │  输出: []*Document    │
        └──────────────────────┘
                   ↓
        ┌──────────────────────┐
        │  chunk: 分块          │
        │  输出: []*Document    │
        └──────────────────────┘
                   ↓
        ┌──────────────────────┐
        │  score: 并行评分      │
        │  (MaxConcurrency=5)  │
        │  输出: []scoredChunk  │
        └──────────────────────┘
                   ↓
        ┌──────────────────────┐
        │  filter: 筛选 top-k   │
        │  输出: []scoredChunk  │
        └──────────────────────┘
                   ↓
        ┌──────────────────────┐
        │  answer: 生成答案     │
        │  输出: Output         │
        └──────────────────────┘
                   ↓
        ┌──────────────────────┐
        │  返回结果             │
        │  {answer, sources}    │
        └──────────────────────┘
```

## 本章小结

- **Graph Tool**：将复杂工作流封装为 Tool，支持多步骤协同
- **compose.Workflow**：构建工作流的核心组件
- **BatchNode**：并行处理多个任务
- **FieldMapping**：跨节点传递数据
- **可中断恢复**：Graph Tool 支持 Checkpoint 机制

## 扩展思考

**其他 Graph Tool 应用：**
- 多文档 RAG：并行处理多个文档
- 多模型协作：不同模型处理不同任务
- 复杂决策树：根据条件选择不同分支

**性能优化：**
- 调整 MaxConcurrency 控制并发
- 使用缓存避免重复计算
- 流式输出提升用户体验
