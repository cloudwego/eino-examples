# AIO Sandbox 文件系统中间件示例

本示例演示如何将 Deep Agent 与 [AIO Sandbox](https://github.com/agent-infra/sandbox) 集成

您可以访问 AIO Sandbox 通过 [Volcano Engine veFaaS Sandbox](https://www.volcengine.com/docs/6662/1802770) 快速获得一个安全隔离的代码执行环境。

本示例演示如何实现自定义 `filesystem.Backend`，并将其与 `filesystem.NewMiddleware` 配合使用，为运行在远程 AIO Sandbox 环境中的 Agent 提供文件系统工具。

## 概述

`filesystem.Backend` 接口允许你接入任意文件系统实现。本示例展示了如何：

1. 使用 AIO Sandbox SDK 实现 `filesystem.Backend` 接口
2. 使用自定义后端创建文件系统中间件
3. 将中间件与 Deep Agent 配合使用

## 架构

```
┌─────────────────────────────────────────────────────────────────┐
│                         Deep Agent                               │
├─────────────────────────────────────────────────────────────────┤
│                    Filesystem Middleware                         │
│  (自动注册: ls, read_file, write_file, edit_file,               │
│             glob, grep, execute 工具)                            │
├─────────────────────────────────────────────────────────────────┤
│                  AIOSandboxBackend                               │
│         (实现 filesystem.Backend + ShellBackend)                 │
├─────────────────────────────────────────────────────────────────┤
│                   AIO Sandbox SDK                                │
│                  (远程沙箱环境)                                   │
└─────────────────────────────────────────────────────────────────┘
```

## 后端接口映射

| filesystem.Backend 方法   | AIO Sandbox SDK API           |
|---------------------------|-------------------------------|
| LsInfo                    | File.ListPath                 |
| Read                      | File.ReadFile                 |
| Write                     | File.WriteFile                |
| Edit                      | File.ReplaceInFile            |
| GrepRaw                   | File.FindFiles + SearchInFile |
| GlobInfo                  | File.FindFiles                |
| Execute (ShellBackend)    | Shell.ExecCommand             |

## 前置条件

设置以下环境变量：

```bash
# AIO Sandbox 配置（不含尾部斜杠）
AIO_SANDBOX_BASE_URL=https://xxx.apigateway-cn-beijing.volceapi.com

# 模型配置（选择其中一种方式）

# 方式一：OpenAI 兼容接口
OPENAI_API_KEY=your-api-key
OPENAI_MODEL=gpt-4
OPENAI_BASE_URL=https://api.openai.com/v1

# 方式二：Ark（设置 MODEL_TYPE=ark）
# MODEL_TYPE=ark
# ARK_API_KEY=your-ark-api-key
# ARK_MODEL=your-model-name
# ARK_BASE_URL=https://ark.cn-beijing.volces.com/api/v3
```

## 使用方法

### 基本用法

```go
// 创建自定义文件系统后端
backend, err := NewAIOSandboxBackend(ctx, &AIOSandboxBackendConfig{
  BaseURL: "https://xxx.apigateway-cn-beijing.volceapi.com",
  WorkDir: "/tmp",
})

// 使用自定义后端创建文件系统中间件
fsMW, err := filesystem.NewMiddleware(ctx, &filesystem.Config{
    Backend: backend,
})

// 与 Agent 配合使用
agent, err := deep.New(ctx, &deep.Config{
    ChatModel:   chatModel,
    Middlewares: []adk.AgentMiddleware{fsMW},
})
```


## 运行示例

```bash
cd adk/middleware/aiosandbox-filesystem
go run .
```

## 实现自定义后端

要实现自定义文件系统后端，需要实现 `filesystem.Backend` 接口：

```go
type Backend interface {
    LsInfo(ctx context.Context, req *LsInfoRequest) ([]FileInfo, error)
    Read(ctx context.Context, req *ReadRequest) (string, error)
    GrepRaw(ctx context.Context, req *GrepRequest) ([]GrepMatch, error)
    GlobInfo(ctx context.Context, req *GlobInfoRequest) ([]FileInfo, error)
    Write(ctx context.Context, req *WriteRequest) error
    Edit(ctx context.Context, req *EditRequest) error
}
```

实现 `ShellBackend` 以提供 `execute` 工具：

```go
type ShellBackend interface {
    Backend
    Execute(ctx context.Context, input *ExecuteRequest) (*ExecuteResponse, error)
}
```

## 与 CommandLine 方式的对比

本示例使用 **中间件方式** 配合 `filesystem.Backend`：
- 自动注册 7 个工具：`ls`、`read_file`、`write_file`、`edit_file`、`glob`、`grep`、`execute`
- 内置大结果分流处理
- 简洁的基于接口的设计

[deep-aiosandbox](../multiagent/deep-aiosandbox) 示例使用 **工具方式** 配合 `commandline.Operator`：
- 手动注册工具：`bash`、`read_file`、`edit_file`、`tree`
- 对自定义工具组合更灵活
- 需要更多配置代码
