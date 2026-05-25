---
title: "第一章：ChatModel 与 AgenticMessage（Console）"
---

## Eino 框架简介

**Eino 是什么？**

Eino 是一个 Go 语言实现的 AI 应用开发框架（Agent Development Kit），旨在帮助开发者快速构建可扩展、可维护的 AI 应用。

**Eino 解决什么问题？**

1. **模型抽象**：统一不同 LLM 提供商的接口（OpenAI、Ark、Claude 等），切换模型无需修改业务代码
2. **能力组合**：通过 Component 接口实现可替换、可组合的能力单元（对话、工具、检索等）
3. **编排框架**：提供 Agent、Graph、Chain 等编排抽象，支持复杂的多步骤 AI 工作流
4. **运行时支持**：内置流式输出、中断与恢复、状态管理、Callback 可观测性等能力

**Eino 的主要仓库：**
- **eino**（本仓库）：核心库，定义接口、编排抽象和 ADK
- **eino-ext**：扩展库，提供各类 Component 的具体实现（OpenAI、Ark、Milvus 等）
- **eino-examples**：示例代码库，包含本 quickstart 系列

---

## ChatWithEino：与 Eino 文档对话的智能助手

**ChatWithEino 是什么？**

ChatWithEino 是一个基于 Eino 框架构建的智能助手，能够帮助开发者学习 Eino 框架并编写 Eino 代码。它通过访问 Eino 仓库的源码、注释和示例，为用户提供最准确、最及时的技术支持。

**核心能力：**
- **对话交互**：理解用户关于 Eino 的问题，提供清晰的解答
- **代码访问**：直接读取 Eino 源码、注释和示例，基于真实实现回答问题
- **持久化会话**：支持多轮对话，记住上下文，可跨进程恢复会话
- **工具调用**：能够执行文件读取、代码搜索等操作

**技术架构：**
- **ChatModel**：与大语言模型通信（OpenAI、Ark、Claude 等）
- **Tool**：文件系统访问、代码搜索等能力扩展
- **Memory**：对话历史持久化存储
- **Agent**：统一的执行框架，协调各组件协同工作

## Quickstart 文档系列：从零构建 ChatWithEino

本系列文档通过循序渐进的方式，带你从最基础的 ChatModel 调用开始，逐步构建一个功能完整的 ChatWithEino Agent。

**学习路径：**

| 章节 | 主题 | 核心内容 | 能力提升 |
|------|------|----------|----------|
| **第一章** | ChatModel 与 AgenticMessage | 理解 Component 抽象，实现单次对话 | 基础对话能力 |
| **第二章** | Agent 与 Runner | 引入执行抽象，实现多轮对话 | 会话管理能力 |
| **第三章** | Memory 与 Session | 持久化对话历史，支持会话恢复 | 持久化能力 |
| **第四章** | Tool 与文件系统 | 添加文件访问能力，读取源码 | 工具调用能力 |
| **第五章** | Middleware | 中间件机制，统一处理横切关注点 | 扩展性增强 |
| **第六章** | Callback | 回调机制，监控 Agent 执行过程 | 可观测性 |
| **第七章** | Interrupt 与 Resume | 中断与恢复，支持长时间任务 | 可靠性增强 |
| **第八章** | Graph 与 Tool | 使用 Graph 编排复杂工作流 | 复杂编排能力 |
| **第九章** | Skill | 使用 Skill 中间件加载并复用技能文档 | 知识复用能力 |
| **最终章** | A2UI | Agent 到 UI 的集成方案 | 生产级应用 |

**为什么这样设计？**

每一章都在前一章的基础上增加一个核心能力，让你：
1. **理解每个组件的作用**：不是一次性展示所有功能，而是逐步引入
2. **看到架构演进过程**：从简单到复杂，理解为什么需要每个抽象
3. **掌握实际开发技能**：每章都有可运行的代码，可以动手实践

---

本章目标：理解 Eino 的 Component 抽象，用最小代码调用一次 ChatModel（支持流式输出），并掌握如何用 `schema.AgenticMessage` 组织模型输入和流式输出。

## 代码位置

- 入口代码：[cmd/ch01/main.go](https://github.com/cloudwego/eino-examples/blob/main/quickstart/chatwitheino/cmd/ch01/main.go)

## 为什么需要 Component 接口

Eino 定义了一组 Component 接口（`ChatModel`、`Tool`、`Retriever`、`Loader` 等），每个接口描述一类可替换的能力：

```go
type BaseModel[M any] interface {
    Generate(ctx context.Context, input []M, opts ...Option) (M, error)
    Stream(ctx context.Context, input []M, opts ...Option) (*schema.StreamReader[M], error)
}

type AgenticModel = BaseModel[*schema.AgenticMessage]
```

**接口带来的好处：**

1. **实现可替换**：`eino-ext` 提供了 OpenAI、Ark、Claude、Ollama 等多种实现，业务代码只依赖接口，切换模型只需改构造逻辑。
2. **编排可组合**：Agent、Graph、Chain 等编排层只依赖 Component 接口，不关心具体实现。你可以把 OpenAI 换成 Ark，编排代码无需改动。
3. **测试可 Mock**：接口天然支持 mock，单元测试不需要真实调用模型。

本章只涉及 `ChatModel`，后续章节会逐步引入 `Tool`、`Retriever` 等 Component。

本示例代码默认使用 `model.AgenticModel`，也就是 `model.BaseModel[*schema.AgenticMessage]`。这样后续章节可以在同一套消息结构里表达文本、reasoning、工具调用、工具结果等内容。

## schema.AgenticMessage：对话的基本单位

`AgenticMessage` 是本 Quickstart 使用的对话数据结构：

在一次模型调用中，模型可能会返回多个有序事件，例如先输出 `reasoning`，再调用 server tool，随后继续 `reasoning`，接着调用 function tool。`AgenticMessage` 会用 `ContentBlock` 按顺序保存这些结构化事件。

```go
type AgenticMessage struct {
    Role          AgenticRoleType
    ContentBlocks []*ContentBlock
    ResponseMeta  *AgenticResponseMeta
    Extra         map[string]any
}

type ContentBlock struct {
    Type               ContentBlockType
    Reasoning          *Reasoning
    UserInputText      *UserInputText
    AssistantGenText   *AssistantGenText
    FunctionToolCall   *FunctionToolCall
    FunctionToolResult *FunctionToolResult
    // ...
}
```

常用构造函数：

```go
schema.SystemAgenticMessage("You are a helpful assistant.")
schema.UserAgenticMessage("What is the weather today?")

&schema.AgenticMessage{
    Role: schema.AgenticRoleTypeAssistant,
    ContentBlocks: []*schema.ContentBlock{
        schema.NewContentBlock(&schema.AssistantGenText{Text: "I don't know."}),
    },
}
```

**角色语义：**
- `system`：系统指令，通常放在消息列表最前面
- `user`：用户输入
- `assistant`：模型回复
- 工具调用和工具结果通过 `function_tool_call` / `function_tool_result` content block 表达（后续章节涉及）

## 前置条件

### 获取代码

```bash
git clone https://github.com/cloudwego/eino-examples.git
cd eino-examples/quickstart/chatwitheino
```

- Go 版本：Go 1.21+（见 `go.mod`）
- 一个可调用的 ChatModel（默认使用 OpenAI；也支持 Ark）

### 方式 A：OpenAI（默认）

```bash
export OPENAI_API_KEY="..."
export OPENAI_MODEL="gpt-4.1-mini"  # OpenAI 2025 年新模型，也可用 gpt-4o、gpt-4o-mini 等
# 可选：
# OPENAI_BASE_URL（代理或兼容服务）
# OPENAI_BY_AZURE=true（使用 Azure OpenAI）
```

### 方式 B：Ark

```bash
export MODEL_TYPE="ark"
export ARK_API_KEY="..."
export ARK_MODEL="..."
# 可选：ARK_BASE_URL
```

## 运行

在 `examples/quickstart/chatwitheino` 目录下执行：

```bash
go run ./cmd/ch01 -- "用一句话解释 Eino 的 Component 设计解决了什么问题？"
```

输出示例（流式逐步打印）：

```text
[assistant] Eino 的 Component 设计通过定义统一接口...
```

## 入口代码做了什么

按执行顺序：

1. **创建 ChatModel**：根据 `MODEL_TYPE` 环境变量选择 OpenAI 或 Ark 的 agentic model
2. **构造输入 messages**：通过 `msgops.NewSystem[M]` / `msgops.NewUser[M]` 创建 `AgenticMessage`
3. **调用 Stream**：使用 `model.BaseModel[M].Stream()`，返回 `StreamReader[M]`
4. **打印结果**：迭代 `StreamReader` 逐帧打印 assistant 回复

关键代码片段（**注意：这是简化后的代码片段，不能直接运行，完整代码请参考** [cmd/ch01/main.go](https://github.com/cloudwego/eino-examples/blob/main/quickstart/chatwitheino/cmd/ch01/main.go)）：

```go
func runTyped[M adk.MessageType](ctx context.Context, instruction, query string) {
    cm, err := chatmodel.NewModel[M](ctx)
    if err != nil {
        log.Fatal(err)
    }

    messages := []M{
        msgops.NewSystem[M](instruction),
        msgops.NewUser[M](query),
    }

    stream, err := cm.Stream(ctx, messages)
    if err != nil {
        log.Fatal(err)
    }
    defer stream.Close()

    for {
        frame, err := stream.Recv()
        if errors.Is(err, io.EOF) {
            break
        }
        if err != nil {
            log.Fatal(err)
        }
        fmt.Print(msgops.AssistantDeltaText(frame))
    }
}
```

## 本章小结

- **Component 接口**：定义可替换、可组合、可测试的能力边界
- **AgenticMessage**：对话数据的基本单位，通过角色和 content block 区分语义
- **ChatModel**：最基础的 Component，提供 `Generate` 和 `Stream` 两个核心方法
- **实现选择**：通过环境变量或配置切换 OpenAI/Ark 等不同实现，业务代码无需改动
