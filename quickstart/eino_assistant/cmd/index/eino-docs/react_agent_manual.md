---
Description: ""
date: "2025-01-07"
lastmod: ""
tags: []
title: 'Eino: React Agent 使用手册'
weight: 0
---

# 简介

Eino React Agent 是实现了 [React 逻辑](https://react-lm.github.io/)的智能体框架，用户可以用来快速灵活地构建并调用 React Agent.

> 💡
> 代码实现详见：[实现代码目录](https://github.com/cloudwego/eino/tree/main/flow/agent/react)

## 节点拓扑&数据流图

react agent 底层使用 `compose.StateGraph` 作为编排方案，仅有 2 个节点: ChatModel、Tools，中间运行过程中的所有历史消息都会放入 state 中，在将所有历史消息传递给 ChatModel 之前，会 copy 消息交由 MessageModifier 进行处理，处理的结果再传递给 ChatModel。直到 ChatModel 返回的消息中不再有 tool call，则返回最终消息。

![](/img/eino/react_agent_graph.png)

## 初始化

提供了 ReactAgent 初始化函数，必填参数为 Model 和 ToolsConfig，选填参数为 MessageModifier 和 MaxStep.

```bash
go get github.com/cloudwego/eino-ext/components/model/openai@latest
go get github.com/cloudwego/eino@latest
```

```go
import (
    "github.com/cloudwego/eino-ext/components/model/openai"
    
    "github.com/cloudwego/eino/components/model"
    "github.com/cloudwego/eino/components/tool"
    "github.com/cloudwego/eino/compose"
    "github.com/cloudwego/eino/flow/agent/react"
    "github.com/cloudwego/eino/schema"
)

func main() {
    // 先初始化所需的 chatModel
    toolableChatModel, err := openai.NewChatModel(...)
    
    // 初始化所需的 tools
    tools := compose.ToolsNodeConfig{
        InvokableTools:  []tool.InvokableTool{mytool},
        StreamableTools: []tool.StreamableTool{myStreamTool},
    }
    
    // 创建 agent
    agent, err := react.NewAgent(ctx, react.AgentConfig{
        Model: toolableChatModel,
        ToolsConfig: tools,
        ...
    }
}
```

### Model

model 接收一个 ChatModel，在 agent 内部，会调用 BindTools 接口，定义为:

```go
type ChatModel interface {
    Generate(ctx context.Context, input []*schema.Message, opts ...Option) (*schema.Message, error)
    Stream(ctx context.Context, input []*schema.Message, opts ...Option) (
        *schema.StreamReader[*schema.Message], error)
        
    BindTools(tools []*schema.ToolInfo) error
}
```

目前，eino 提供了 openai 和 ark 的实现，只要底层模型支持 tool call 即可。

```bash
go get github.com/cloudwego/eino-ext/components/model/openai@latest
go get github.com/cloudwego/eino-ext/components/model/ark@latest
```

```go
import (
    "github.com/cloudwego/eino-ext/components/model/openai"
    "github.com/cloudwego/eino-ext/components/model/ark"
)

func openaiExample() {
    chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
        BaseURL: "https://search.bytedance.net/gpt/openapi/online/multimodal/crawl",
        Key:     os.Getenv("OPENAI_ACCESS_KEY"),
        ByAzure: true,
        Model:   "{{model name which support tool call}}",
    })

    agent, err := react.NewAgent(ctx, react.AgentConfig{
        Model: chatModel,
        ToolsConfig: ...,
    })
}

func arkExample() {
    arkModel, err := ark.NewChatModel(context.Background(), ark.ChatModelConfig{
        APIKey: os.Getenv("ARK_API_KEY"),
        Model:  os.Getenv("ARK_MODEL"),
    })

    agent, err := react.NewAgent(ctx, react.AgentConfig{
        Model: arkModel,
        ToolsConfig: ...,
    })
}
```

### ToolsConfig

toolsConfig 类型为 `compose.ToolsNodeConfig`, 在 eino 中，若要构建一个 Tool 节点，则需要提供 Tool 的信息，以及调用 Tool 的接口，tool 的接口定义如下:

```go
type InvokableRun func(ctx context.Context, arguments string, opts ...Option) (content string, err error)
type StreamableRun func(ctx context.Context, arguments string, opts ...Option) (content *schema.StreamReader[string], err error)

type BaseTool interface {
    Info() *schema.ToolInfo
}

// InvokableTool the tool for ChatModel intent recognition and ToolsNode execution.
type InvokableTool interface {
    BaseTool
    Run() InvokableRun
}

// StreamableTool the stream tool for ChatModel intent recognition and ToolsNode execution.
type StreamableTool interface {
    BaseTool
    Run() StreamableRun
}
```

用户可以根据 tool 的接口定义自行实现所需的 tool，同时框架也提供了更简便的构建 tool 的方法：

```go
import (
    "context"
    
    "github.com/cloudwego/eino/components/tool/utils"
    "github.com/cloudwego/eino/compose"
    "github.com/cloudwego/eino/schema"
)

func main() {
    // 提供 tool 的信息
    toolInfo := &schema.ToolInfo{
        Name: "xxx",
        Desc: "description for tool, it's important for chatmodel choice which tool to use",
        Params: map[string]*schema.ParameterInfo{
            "param01": {
                Type: "string",
                Desc: "xxxx", // import for chatmodel generate params
            },
            "param01": {
                Type: "string",
                Desc: "xxxx",
            },
        },
    }
    
    // 提供 tool 的调用方法
    // 需满足 type InvokeFunc[T, D any] func(ctx context.Context, input T) (output D, err error)
    toolInvokeFunc := func(ctx context.Context, in string) (out string, err error)
    
    // 构建 tool
    invokeTool := utils.NewTool(toolInfo, toolInvokeFunc)
    
    // stream tool 同理
    // utils.NewStreamTool
    
    toolConfig := &compose.ToolsNodeConfig{
        InvokableTools:  []tool.InvokableTool{invokeTool},
    }

}
```

### MessageModifier

MessageModifier 会在每次把所有历史消息传递给 ChatModel 之前执行，其定义为：

```go
// modify the input messages before the model is called.
type MessageModifier func(ctx context.Context, input []*schema.Message) []*schema.Message
```

框架提供了一个简便的 PersonaModifier，用于在消息列表的最头部增加一个代表 agent 个性的 system message，使用如下:

```go
import (
    "github.com/cloudwego/eino/flow/agent/react"
    "github.com/cloudwego/eino/schema"
)

func main() {
    persona := `你是一个 golang 开发专家.`
    
    agent, err := react.NewAgent(ctx, react.AgentConfig{
        Model: toolableChatModel,
        ToolsConfig: tools,
        
        // MessageModifier
        MessageModifier: react.NewPersonaModifier(persona),
    }
    
    agent.Generate(ctx, []*schema.Message{{Role: schame.Human, Content: "写一个 hello world 的代码"}}
    // 实际到 ChatModel 的 input 为
    // []*schema.Message{
    //    {Role: schema.System, Content: "你是一个 golang 开发专家."},
    //    {Role: schema.Human, Content: "写一个 hello world 的代码"}
    //}
}
```

### MaxStep

指定 Agent 最大运行步长，每次从一个节点转移到下一个节点为一步，默认值为 12。

由于 Agent 中一次循环为 ChatModel + Tools，即为 2 步，因此默认值 12 最多可运行 6 个循环。但由于最后一步必须为 ChatModel 返回 (因为 ChatModel 结束后判断无须运行 tool 才能返回最终结果)，因此最多运行 5 次 tool。

同理，若希望最多可运行 10 个循环 (10 次 ChatModel + 9 次 Tools)，则需要设置 MaxStep 为 20。若希望最多运行 20 个循环，则 MaxStep 需为 40。

```go
func main() {
    agent, err := react.NewAgent(ctx, react.AgentConfig{
        Model: toolableChatModel,
        ToolsConfig: tools,
        MaxStep: 20,
    }
}
```

## 调用

### Generate

```go
import (
    "context"

    "github.com/cloudwego/eino/flow/agent/react"
    "github.com/cloudwego/eino/schema"
)

func main() {
    agent, err := react.NewAgent(...)

    var outMessage *schema.Message
    outMessage, err = agent.Generate(ctx, []*schema.Message{
        {
            Role:    schema.Human,
            Content: "写一个 golang 的 hello world 程序",
        },
    })
}
```

### Stream

```go
import (
    "context"
    "fmt"
    
    "github.com/cloudwego/eino/flow/agent/react"
    "github.com/cloudwego/eino/schema"
)

func main() {
    agent, err := react.NewAgent(...)

    var msgReader *schema.StreamReader[*schema.Message]
    msgReader, err = agent.Stream(ctx, []*schema.Message{
        {
            Role:    schema.Human,
            Content: "写一个 golang 的 hello world 程序",
        },
    })
    
    for {
        // msg type is *schema.Message
        msg, err := msgReader.Recv()
        if err != nil {
            if errors.Is(err, io.EOF) {
                // finish
                break
            }
            // error
            log.Printf("failed to recv: %v\n", err)
            return
        }

        fmt.Print(msg.Content)
    }
}
```

### WithCallbacks

Callback 是在 Agent 运行时特定时机执行的回调，传递了一些运行时信息，定义为：

```go
type AgentCallback interface {
    OnChatModelStart(ctx context.Context, input *model.CallbackInput)
    OnChatModelEnd(ctx context.Context, output *model.CallbackOutput)
    OnChatModelEndStream(ctx context.Context, output *schema.StreamReader[*model.CallbackOutput])

    OnToolStart(ctx context.Context, input string)
    OnToolEnd(ctx context.Context, output string)
    OnToolEndStream(ctx context.Context, output *schema.StreamReader[string])

    OnError(ctx context.Context, err error)
}
```

框架提供了空的 BaseCallback 来辅助用户实现接口：

```go
import "github.com/cloudwego/eino/flow/agent/react"

// type BaseCallback struct{}
// func (cb *BaseCallback) OnChatModelStart(ctx context.Context, input *model.CallbackInput) {}
// func (cb *BaseCallback) OnChatModelEnd(ctx context.Context, output *model.CallbackOutput) {}
// func (cb *BaseCallback) OnChatModelEndStream(ctx context.Context, output *schema.StreamReader[*model.CallbackOutput]) {}

// func (cb *BaseCallback) OnToolStart(ctx context.Context, input string)                            {}
// func (cb *BaseCallback) OnToolEnd(ctx context.Context, output string)                             {}
// func (cb *BaseCallback) OnToolEndStream(ctx context.Context, output *schema.StreamReader[string]) {}

// func (cb *BaseCallback) OnError(ctx context.Context, err error) {}

type MyCallback struct{
    *react.BaseCallback
}

// 重载需要的方法……
func (m *MyCallback) OnChatModelEnd(ctx context.Context, output *model.CallbackOutput) {
  // some logic
}

func main() {
    agent, err := react.NewAgent(...)
    if err != nil {...}

    agent.Generate(ctx, []*schema.Message{...}, react.WithCallbacks(&MyCallback{})
}
```

## Agent In Graph/Chain

目前 agent 不是一级的 component 编排到 graph 中，可作为 Lambda 编排 Agent:

```go
import (
    "context"

    "github.com/cloudwego/eino/components/model"
    "github.com/cloudwego/eino/components/model/openai"
    "github.com/cloudwego/eino/components/tool"
    "github.com/cloudwego/eino/compose"
    "github.com/cloudwego/eino/flow/agent/react"
    "github.com/cloudwego/eino/schema"
)

func main() {
    // 创建一个 chain
    chain := compose.NewChain[[]*schema.Message, string]()
    
    // 创建 agent
    agent, err := react.NewAgent(...)
    
    // 把 agent 变成一个 Lambda
    agentLambda, err := compose.AnyLambda(agent.Generate, agent.Stream, nil, nil)
    
    // 把 agentLambda 加入到 chain 的第一个节点
    chain.AppendLambda(agentLambda)
    
    // other
    chain.AppendLambda(...).AppendXXX(...)
    runnable, err := chain.Compile()
    
    // 调用时可传入 LambdaOption，用于传递 agent 的 calloption，例如 callback
    res, err := r.Invoke(ctx, []*schema.Message{{Role: schema.Human, Content: "hello"}},
            compose.WithLambdaOption(agent.WithCallbacks(&MyCallback{})))
    
}
```

## Demo

### 基本信息

简介：这是一个拥有两个 tool (query_restaurants 和 query_dishes ) 的 `美食推荐官`

地址：[eino-examples/flow/agent/react](https://github.com/cloudwego/eino-examples/tree/main/flow/agent/react)

使用方式:

1. clone eino-examples repo，并 cd 到根目录
2. 提供一个 `OPENAI_API_KEY`: `export OPENAI_API_KEY=xxxxxxx`
3. 运行 demo: `go run flow/agent/react/react.go`

### 运行过程解释

- 模拟用户输入了 `我在海淀区，给我推荐一些菜，需要有口味辣一点的菜，至少推荐有 2 家餐厅`
- agent 运行第一个节点 `ChatModel`，大模型判断出需要做一次 ToolCall 调用来查询餐厅，并且给出的参数为：

```json
"function": {
    "name": "query_restaurants",
    "arguments": "{\"location\":\"海淀区\",\"topn\":2}"
}
```

- 进入 `Tools` 节点，调用 查询餐厅 的 tool，并且得到结果，结果返回了 2 家海淀区的餐厅信息:

```json
[{"id":"1001","name":"跳不动的E世界5F餐厅","place":"中关村E世界 5F, 左转进入","desc":"","score":3},{"id":"1002","name":"跳动的E世界地下餐厅","place":"中关村E世界-1F","desc":"","score":5}]
```

- 得到 tool 的结果后，此时对话的 history 中包含了 tool 的结果，再次运行 `ChatModel`，大模型判断出需要再次调用另一个 ToolCall，用来查询餐厅有哪些菜品，注意，由于有两家餐厅，因此大模型返回了 2 个 ToolCall，如下：

```json
"Message": {
    "role": "ai",
    "content": "",
    "tool_calls": [ // <= 这里有 2 个 tool call
      {
        "index": 1,
        "id": "call_wV7zA3vGGJBhuN7r9guhhAfF",
        "function": {
          "name": "query_dishes",
          "arguments": "{\"restaurant_id\": \"1002\", \"topn\": 5}"
        }
      },
      {
        "index": 0,
        "id": "call_UOsp0jRtzEbfxixNjP5501MF",
        "function": {
          "name": "query_dishes",
          "arguments": "{\"restaurant_id\": \"1001\", \"topn\": 5}"
        }
      }
    ]
  }
```

- 再次进入到 `Tools` 节点，由于有 2 个 tool call，Tools 节点内部并发执行这两个调用，并且均加入到对话的 history 中，从 callback 的调试日志中可以看到结果如下：

```json
=========[OnToolStart]=========
{"restaurant_id": "1001", "topn": 5}
=========[OnToolEnd]=========
[{"name":"红烧肉","desc":"一块红烧肉","price":20,"score":8},{"name":"清泉牛肉","desc":"很多的水煮牛肉","price":50,"score":8},{"name":"清炒小南瓜","desc":"炒的糊糊的南瓜","price":5,"score":5},{"name":"韩式辣白菜","desc":"这可是开过光的辣白菜，好吃得很","price":20,"score":9},{"name":"酸辣土豆丝","desc":"酸酸辣辣的土豆丝","price":10,"score":9}]
=========[OnToolStart]=========
{"restaurant_id": "1002", "topn": 5}
=========[OnToolEnd]=========
[{"name":"红烧排骨","desc":"一块一块的排骨","price":43,"score":7},{"name":"大刀回锅肉","desc":"经典的回锅肉, 肉很大","price":40,"score":8},{"name":"火辣辣的吻","desc":"凉拌猪嘴，口味辣而不腻","price":60,"score":9},{"name":"辣椒拌皮蛋","desc":"擂椒皮蛋，下饭的神器","price":15,"score":8}]
```

- 得到所有 tool call 返回的结果后，再次进入 `ChatModel` 节点，这次大模型发现已经拥有了回答用户提问的所有信息，因此整合信息后输出结论，由于调用时使用的 `Stream` 方法，因此流式返回的大模型结果。

## 关联阅读

- [Eino Tutorial: Host Multi-Agent ](/zh/docs/eino/core_modules/flow_integration_components/multi_agent_hosting)
