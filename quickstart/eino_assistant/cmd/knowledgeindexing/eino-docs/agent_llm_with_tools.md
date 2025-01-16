## **Agent 是什么**

Agent（智能代理）是一个能够感知环境并采取行动以实现特定目标的系统。在 AI 应用中，Agent 通过结合大语言模型的理解能力和预定义工具的执行能力，可以自主地完成复杂的任务。是未来 AI 应用到生活生产中主要的形态。

> 💡
> 本文中示例的代码片段详见：[eino-examples/quickstart/taskagent](https://github.com/cloudwego/eino-examples/blob/master/quickstart/taskagent/main.go)

## **Agent 的核心组成**

在 Eino 中，要实现 Agent 主要需要两个核心部分：ChatModel 和 Tool。

### **ChatModel**

ChatModel 是 Agent 的大脑，它通过强大的语言理解能力来处理用户的自然语言输入。当用户提出请求时，ChatModel 会深入理解用户的意图，分析任务需求，并决定是否需要调用特定的工具来完成任务。在需要使用工具时，它能够准确地选择合适的工具并生成正确的参数。不仅如此，ChatModel 还能将工具执行的结果转化为用户易于理解的自然语言回应，实现流畅的人机对话。

> 更详细的 ChatModel 的信息，可以参考： [Eino: ChatModel 使用说明](/zh/docs/eino/core_modules/components/chat_model_guide)

### **Tool**

Tool 是 Agent 的执行器，提供了具体的功能实现。每个 Tool 都有明确的功能定义和参数规范，使 ChatModel 能够准确地调用它们。Tool 可以实现各种功能，从简单的数据操作到复杂的外部服务调用都可以封装成 Tool。

> 更详细关于 Tool 和 ToolsNode 的信息，可参考： [Eino: ToolsNode 使用说明](/zh/docs/eino/core_modules/components/tools_node_guide)

## **Tool 的实现方式**

在 Eino 中，我们提供了多种方式来实现 Tool。下面通过一个待办事项（Task）管理系统的例子来说明。

### **方式一：使用 NewTool 构建**

这种方式适合简单的工具实现，通过定义工具信息和处理函数来创建 Tool：

```go
func getAddTaskTool() tool.InvokableTool {
    info := &schema.ToolInfo{
        Name: "add_task",
        Desc: "Add a task item",
        ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
            "content": {
                Desc:     "The content of the task item",
                Type:     schema.String,
                Required: true,
            },
            "started_at": {
                Desc: "The started time of the task item, in unix timestamp",
                Type: schema.Integer,
            },
            "deadline": {
                Desc: "The deadline of the task item, in unix timestamp",
                Type: schema.Integer,
            },
        }),
    }

    return utils.NewTool(info, AddTaskFunc)
}
```

这种方式虽然直观，但存在一个明显的缺点：需要在 ToolInfo 中手动定义参数信息（ParamsOneOf），和实际的参数结构（TaskAddParams）是分开定义的。这样不仅造成了代码的冗余，而且在参数发生变化时需要同时修改两处地方，容易导致不一致，维护起来也比较麻烦。

### **方式二：使用 InferTool 构建**

这种方式更加简洁，通过结构体的 tag 来定义参数信息，就能实现参数结构体和描述信息同源，无需维护两份信息：

```go
type TaskUpdateParams struct {
    ID        string  `json:"id" jsonschema:"description=id of the task"`
    Content   *string `json:"content,omitempty" jsonschema:"description=content of the task"`
    StartedAt *int64  `json:"started_at,omitempty" jsonschema:"description=start time in unix timestamp"`
    Deadline  *int64  `json:"deadline,omitempty" jsonschema:"description=deadline of the task in unix timestamp"`
    Done      *bool   `json:"done,omitempty" jsonschema:"description=done status"`
}

// 使用 InferTool 创建工具
updateTool, err := utils.InferTool("update_task", "Update a task item, eg: content,deadline...", UpdateTaskFunc)
```

### **方式三：实现 Tool 接口**

对于需要更多自定义逻辑的场景，可以通过实现 Tool 接口来创建：

```go
type ListTaskTool struct {}

func (lt *ListTaskTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
    return &schema.ToolInfo{
        Name: "list_task",
        Desc: "List all task items",
        ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
            "finished": {
                Desc:     "filter task items if finished",
                Type:     schema.Boolean,
                Required: false,
            },
        }),
    }, nil
}

func (lt *ListTaskTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
    // 具体的调用逻辑
}
```

### **方式四：使用官方封装的工具**

除了自己实现工具，我们还提供了许多开箱即用的工具。这些工具经过充分测试和优化，可以直接集成到你的 Agent 中。以 Google Search 工具为例：

```go
import (
    "github.com/bytedance/eino-ext/components/tool/googlesearch"
)

func main() {
    // 创建 Google Search 工具
    searchTool, err := googlesearch.NewGoogleSearchTool(ctx, &googlesearch.Config{
        APIKey:         os.Getenv("GOOGLE_API_KEY"),         // Google API Key
        SearchEngineID: os.Getenv("GOOGLE_SEARCH_ENGINE_ID"), // 自定义搜索引擎 ID
        Num:           5,      // 每次返回的结果数量
        Lang:          "zh-CN", // 搜索结果的语言
    })
    if err != nil {
        log.Fatal(err)
    }
}
```

使用 eino-ext 提供的工具不仅能避免重复开发的工作量，还能确保工具的稳定性和可靠性。这些工具都经过充分测试和持续维护，可以直接集成到项目中使用。

## **用 Chain 构建 Agent**

在构建 Agent 时，ToolsNode 是一个核心组件，它负责管理和执行工具调用。ToolsNode 可以集成多个工具，并提供统一的调用接口。它支持同步调用（Invoke）和流式调用（Stream）两种方式，能够灵活地处理不同类型的工具执行需求。

要创建一个 ToolsNode，你需要提供一个工具列表配置：

```go
func main() {
    conf := &compose.ToolsNodeConfig{
        Tools: []tool.BaseTool{tool1, tool2},  // 工具可以是 InvokableTool 或 StreamableTool
    }
    toolsNode, err := compose.NewToolNode(ctx, conf)    
}
```

下面是一个完整的 Agent 示例，它使用 OpenAI 的 ChatModel 并结合了上述的 Task 工具:

```go
func main() {
    // 初始化 tools
    taskTools := []tool.BaseTool{
        getAddTaskTool(),                                // 使用 NewTool 方式
        updateTool,                                     // 使用 InferTool 方式
        &ListTaskTool{},
        searchTool,                                 // 使用结构体实现方式, 此处未实现底层逻辑
    }

    // 创建并配置 ChatModel
    temp := float32(0.7)
    chatModel, err := openai.NewChatModel(context.Background(), &openai.ChatModelConfig{
        Model:       "gpt-4",
        APIKey:      os.Getenv("OPENAI_API_KEY"),
        Temperature: &temp,
    })
    if err != nil {
        log.Fatal(err)
    }

    // 获取工具信息, 用于绑定到 ChatModel
    toolInfos := make([]*schema.ToolInfo, 0, len(taskTools))
    for _, tool := range taskTools {
        info, err := tool.Info(ctx)
        if err != nil {
            log.Fatal(err)
        }
        toolInfos = append(toolInfos, info)
    }

    // 将 tools 绑定到 ChatModel
    err = chatModel.BindTools(toolInfos)
    if err != nil {
        log.Fatal(err)
    }


    // 创建 tools 节点
    taskToolsNode, err := compose.NewToolNode(context.Background(), &compose.ToolsNodeConfig{
        Tools: taskTools,
    })
    if err != nil {
        log.Fatal(err)
    }

    // 构建完整的处理链
    chain := compose.NewChain[*schema.Message, []*schema.Message]()
    chain.
        AppendChatModel(chatModel, compose.WithNodeName("chat_model")).
        AppendToolsNode(taskToolsNode, compose.WithNodeName("tools"))

    // 编译并运行 chain
    agent, err := chain.Compile(ctx)
    if err != nil {
        log.Fatal(err)
    }

    // 运行示例
    resp, err := agent.Invoke(context.Background(), &schema.Message{
        Content: "帮我创建一个明天下午3点截止的待办事项：准备Eino项目演示文稿",
    })
    if err != nil {
        log.Fatal(err)
    }

    // 输出结果
    for _, msg := range resp {
        fmt.Println(msg.Content)
    }
}
```

这个示例有一个假设，也就是 ChatModel 一定会做出 tool 调用的决策。实际上这个例子是 tool calling agent 的一个简化版本。更完整的 toolcalling agent 可以参考： [Tool Calling Agent](/zh/docs/eino/usage_guide/examples_collection/task_manager_implementation)

## **使用其他方式构建 Agent**

除了上述使用 Chain/Graph 构建的 agent 之外，Eino 还提供了常用的 Agent 模式的封装。

### **ReAct Agent**

ReAct（Reasoning + Acting）Agent 结合了推理和行动能力，通过思考-行动-观察的循环来解决复杂问题。它能够在执行任务时进行深入的推理，并根据观察结果调整策略，特别适合需要多步推理的复杂场景。

> 更详细的 react agent 可以参考： [Eino: React Agent 使用手册](/zh/docs/eino/core_modules/flow_integration_components/react_agent_manual)

### **Multi Agent**

Multi Agent 系统由多个协同工作的 Agent 组成，每个 Agent 都有其特定的职责和专长。通过 Agent 间的交互与协作，可以处理更复杂的任务，实现分工协作。这种方式特别适合需要多个专业领域知识结合的场景。

> 更详细的 multi agent 可以参考： [Eino Tutorial: Host Multi-Agent ](/zh/docs/eino/core_modules/flow_integration_components/multi_agent_hosting)

## **总结**

介绍了使用 Eino 框架构建 Agent 的基本方法。通过 Chain、Tool Calling 和 ReAct 等不同方式，我们可以根据实际需求灵活地构建 AI Agent。

Agent 是 AI 技术发展的重要方向。它不仅能够理解用户意图，还能主动采取行动，通过调用各种工具来完成复杂任务。随着大语言模型能力的不断提升，Agent 将在未来扮演越来越重要的角色，成为连接 AI 与现实世界的重要桥梁。我们期待 Eino 能为用户带来更强大、易用的 agent 构建方案，推动更多基于 Agent 的应用创新。

## **关联阅读**

- 快速开始
  - [实现一个最简 LLM 应用-ChatModel](/zh/docs/eino/quick_start/simple_llm_application)
  - [和幻觉说再见-RAG 召回再回答](/zh/docs/eino/quick_start/rag_retrieval_qa)
  - [复杂业务逻辑的利器-编排](/zh/docs/eino/quick_start/complex_business_logic_orchestration)
