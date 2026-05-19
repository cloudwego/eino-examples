---
title: "第四章:Tool 与文件系统访问"
---

本章目标:为 Agent 添加 Tool 能力,让 Agent 能够访问文件系统。

## 为什么需要 Tool

前三章我们实现的 Agent 只能对话,无法执行实际操作。

**Agent 的局限:**
- 只能生成文本回复
- 无法访问外部资源(文件、API、数据库等)
- 无法执行实际任务(计算、查询、修改等)

**Tool 的定位:**
- **Tool 是 Agent 的能力扩展**:让 Agent 能够执行具体操作
- **Tool 封装了具体实现**:Agent 不关心 Tool 内部如何工作,只关心输入输出
- **Tool 可组合**:一个 Agent 可以有多个 Tool,根据需要选择调用

**简单类比:**
- **Agent** = "智能助手"(能理解指令,但需要工具才能执行)
- **Tool** = "工具箱"(文件操作、网络请求、数据库查询等)

## 为什么需要文件系统能力

本示例是 ChatWithDoc(与文档对话),目标是帮助用户学习 Eino 框架并编写 Eino 代码。那么,最好的文档是什么?

**答案就是:Eino 仓库的代码本身。**

- **Code**: 源代码展示了框架的真实实现
- **Comment**: 代码注释提供了设计思路和使用说明
- **Examples**: 示例代码演示了最佳实践

通过文件系统访问能力,Agent 可以直接读取 Eino 源码、注释和示例,为用户提供最准确、最及时的技术支持。

## 关键概念

### Tool 接口

`Tool` 是 Eino 中定义可执行能力的接口:

```go
// BaseTool 提供工具的元信息,ChatModel 使用这些信息决定是否以及如何调用工具
type BaseTool interface {
    Info(ctx context.Context) (*schema.ToolInfo, error)
}

// InvokableTool 是可以被 ToolsNode 执行的工具
type InvokableTool interface {
    BaseTool
    // InvokableRun 执行工具,参数是 JSON 编码的字符串,返回字符串结果
    InvokableRun(ctx context.Context, argumentsInJSON string, opts ...Option) (string, error)
}

// StreamableTool 是 InvokableTool 的流式变体
type StreamableTool interface {
    BaseTool
    // StreamableRun 流式执行工具,返回 StreamReader
    StreamableRun(ctx context.Context, argumentsInJSON string, opts ...Option) (*schema.StreamReader[string], error)
}
```

**接口层次:**
- `BaseTool`:基础接口,只提供元信息
- `InvokableTool`:可执行工具(继承 BaseTool)
- `StreamableTool`:流式工具(继承 BaseTool)

### Backend 接口

`Backend` 是 Eino 中用于文件系统操作的抽象接口:

```go
type Backend interface {
    // 列出目录下的文件信息
    LsInfo(ctx context.Context, req *LsInfoRequest) ([]FileInfo, error)
    
    // 读取文件内容,支持按行偏移和限制
    Read(ctx context.Context, req *ReadRequest) (*FileContent, error)
    
    // 在文件中搜索匹配的内容
    GrepRaw(ctx context.Context, req *GrepRequest) ([]GrepMatch, error)
    
    // 根据 glob 模式匹配文件
    GlobInfo(ctx context.Context, req *GlobInfoRequest) ([]FileInfo, error)
    
    // 写入文件内容
    Write(ctx context.Context, req *WriteRequest) error
    
    // 编辑文件内容(字符串替换)
    Edit(ctx context.Context, req *EditRequest) error
}
```

### LocalBackend

`LocalBackend` 是 Backend 的本地文件系统实现,直接访问操作系统的文件系统:

```go
import localbk "github.com/cloudwego/eino-ext/adk/backend/local"

backend, err := localbk.NewBackend(ctx, &localbk.Config{})
```

**特点:**
- 直接访问本地文件系统,使用 Go 标准库实现
- 支持所有 Backend 接口方法
- 支持执行 shell 命令(ExecuteStreaming)
- 路径安全:要求使用绝对路径,防止目录遍历攻击
- 零配置:开箱即用,无需额外设置

## 实现:使用 DeepAgent

本章使用 DeepAgent 预构建 Agent,它提供了 Backend 和 StreamingShell 的一级配置,可以方便地注册文件系统相关的工具。

### 从 ChatModelAgent 到 DeepAgent：何时需要切换？

前面章节一直使用 `ChatModelAgent`,它已经能处理多轮对话。但要访问文件系统，我们需要切换到 `DeepAgent`。

**ChatModelAgent vs DeepAgent 对比：**

| 能力 | ChatModelAgent | DeepAgent |
|------|----------------|-----------|
| 多轮对话 | ✅ | ✅ |
| 添加自定义 Tool | ✅ 手动注册每个 Tool | ✅ 手动注册或自动注册 |
| 文件系统访问（Backend） | ❌ 需手动创建并注册所有文件工具 | ✅ 一级配置，自动注册 |
| 命令执行（StreamingShell） | ❌ 需手动创建 | ✅ 一级配置，自动注册 |
| 内置任务管理 | ❌ | ✅ `write_todos` 工具 |
| 支持子 Agent | ❌ | ✅ |

**选择建议：**
- 纯对话场景（无外部访问）→ 用 `ChatModelAgent`
- 需要访问文件系统或执行命令 → 用 `DeepAgent`

### 为什么使用 DeepAgent?

相比直接使用 ChatModelAgent,DeepAgent 的优势:

1. **一级配置**: Backend 和 StreamingShell 是一级配置,直接传入即可
2. **自动注册工具**: 配置 Backend 后自动注册文件系统工具,无需手动创建
3. **内置任务管理**: 提供 `write_todos` 工具,支持任务规划和跟踪
4. **支持子 Agent**: 可以配置专门的子 Agent 处理特定任务
5. **更强大**: 集成了文件系统、命令执行等多种能力

### 代码实现

```go
import (
    localbk "github.com/cloudwego/eino-ext/adk/backend/local"
    "github.com/cloudwego/eino/adk/prebuilt/deep"
)

// 创建 LocalBackend
backend, err := localbk.NewBackend(ctx, &localbk.Config{})

// 创建 DeepAgent,自动注册文件系统工具
agent, err := deep.New(ctx, &deep.Config{
    Name:           "Ch04ToolAgent",
    Description:    "ChatWithDoc agent with filesystem access via LocalBackend.",
    ChatModel:      cm,
    Instruction:    agentInstruction,
    Backend:        backend,        // 提供文件系统操作能力
    StreamingShell: backend,        // 提供命令执行能力
    MaxIteration:   50,
})
```

### DeepAgent 自动注册的工具

当配置了 `Backend` 和 `StreamingShell` 后,DeepAgent 会自动注册以下工具:

- `read_file`: 读取文件内容
- `write_file`: 写入文件内容
- `edit_file`: 编辑文件内容
- `glob`: 根据 glob 模式查找文件
- `grep`: 在文件中搜索内容
- `execute`: 执行 shell 命令

## 代码位置

- 入口代码：[cmd/ch04/main.go](https://github.com/cloudwego/eino-examples/blob/main/quickstart/chatwitheino/cmd/ch04/main.go)

## 前置条件

与第一章一致:需要配置一个可用的 ChatModel(OpenAI 或 Ark)。

本章还需要设置 `PROJECT_ROOT`（可选，见下方运行说明）。

## 运行

在 `examples/quickstart/chatwitheino` 目录下执行:

```bash
# 可选：设置 Eino 核心库的根目录路径
# 未设置时，Agent 默认使用当前工作目录（即 chatwitheino 目录）作为根目录
# 若要让 Agent 能检索完整的 Eino 代码库，建议指向 eino 核心库根目录
export PROJECT_ROOT=/path/to/eino

# 验证路径是否正确（应该能看到 adk、components、compose 等目录）
ls $PROJECT_ROOT

go run ./cmd/ch04
```

**`PROJECT_ROOT` 说明：**

- **不设置时**：`PROJECT_ROOT` 默认为当前工作目录（`chatwitheino` 所在目录），Agent 只能访问本示例项目的文件。这对于快速试验已足够。
- **设置后**：指向 Eino 核心库根目录，Agent 可以检索 Eino 框架的完整代码库（核心库、扩展库、示例库）。这是 ChatWithEino 的完整使用场景。

**推荐的三仓库目录结构（如要完整体验）：**

```
eino/                    # PROJECT_ROOT（Eino 核心库）
├── adk/
├── components/
├── compose/
├── ext/                 # eino-ext（扩展组件，如 OpenAI、Ark 等实现）
├── examples/            # eino-examples（本仓库，本示例所在位置）
│   └── quickstart/
│       └── chatwitheino/
└── ...
```

可以使用 `dev_setup.sh` 脚本自动设置上述目录结构：

```bash
# 在 eino 根目录运行，自动克隆扩展库和示例库到正确位置
bash scripts/dev_setup.sh
```

输出示例:

```text
you> 列出当前目录的文件
[assistant] 我来帮你列出当前目录的文件...
[tool call] glob(pattern: "*")
[tool result] 找到 5 个文件:
- main.go
- go.mod
- go.sum
- README.md
- cmd/

you> 读取 main.go 文件的内容
[assistant] 我来读取 main.go 文件...
[tool call] read_file(file_path: "main.go")
[tool result] 文件内容如下:
...
```

**注意:** 如果在运行过程中遇到 Tool 报错导致 Agent 中断,请不要 panic,这是正常现象。Tool 报错是常见的情况,例如参数错误、文件不存在等。如何优雅地处理 Tool 错误,我们将在下一章详细介绍。

## Tool 调用流程

当 Agent 需要调用 Tool 时:

```
┌─────────────────────────────────────────┐
│  用户:列出当前目录的文件                 │
└─────────────────────────────────────────┘
                   ↓
        ┌──────────────────────┐
        │  Agent 分析意图       │
        │  决定调用 glob 工具   │
        └──────────────────────┘
                   ↓
        ┌──────────────────────┐
        │  生成 Tool Call       │
        │  {"pattern": "*"}     │
        └──────────────────────┘
                   ↓
        ┌──────────────────────┐
        │  执行 Tool            │
        │  glob("*")            │
        └──────────────────────┘
                   ↓
        ┌──────────────────────┐
        │  返回 Tool Result     │
        │  {"files": [...]}     │
        └──────────────────────┘
                   ↓
        ┌──────────────────────┐
        │  Agent 生成回复       │
        │  "找到 5 个文件..."    │
        └──────────────────────┘
```

## 本章小结

- **Tool**:Agent 的能力扩展,让 Agent 能够执行具体操作
- **Backend**:文件系统操作的抽象接口,提供统一的文件操作能力
- **LocalBackend**:Backend 的本地文件系统实现,直接访问操作系统文件系统
- **DeepAgent**:预构建的高级 Agent,提供 Backend 和 StreamingShell 的一级配置
- **自动注册工具**:配置 Backend 后自动注册文件系统工具
- **Tool 调用流程**:Agent 分析意图 → 生成 Tool Call → 执行 Tool → 返回结果 → 生成回复

## 扩展思考

**其他 Tool 类型:**
- HTTP Tool:调用外部 API
- Database Tool:查询数据库
- Calculator Tool:执行计算
- Code Executor Tool:运行代码

**其他 Backend 实现:**
- 可以基于 Backend 接口实现其他存储后端
- 例如:云存储、数据库存储等
- LocalBackend 已经提供了完整的文件系统操作能力

**自定义 Tool 创建:**

如果需要创建自定义 Tool,可以使用 `utils.InferTool` 从函数自动推断。详见:
- [Tool 接口文档](https://github.com/cloudwego/eino/tree/main/components/tool)
- [Tool 创建示例](https://github.com/cloudwego/eino-examples/tree/main/components/tool)
