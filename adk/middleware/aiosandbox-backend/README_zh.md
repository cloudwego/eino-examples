# AIO Sandbox 文件系统中间件示例

本示例演示如何将 Deep Agent 与 [AIO Sandbox](https://github.com/agent-infra/sandbox) 集成

您可以通过 [火山引擎 veFaaS Sandbox](https://www.volcengine.com/docs/6662/1802770) 快速获得一个安全隔离的代码执行环境。

本示例演示如何实现自定义 `filesystem.Backend`，并将其与 `filesystem.NewMiddleware` 配合使用，为运行在远程 AIO Sandbox 环境中的 Agent 提供文件系统工具。

## 概述

`filesystem.Backend` 接口允许你接入任意文件系统实现。本示例展示了如何：

1. 使用 AIO Sandbox SDK 实现 `filesystem.Backend` 接口（数据面）
2. 通过 veFaaS API 管理沙箱生命周期（控制面）
3. 使用自定义后端创建文件系统中间件
4. 将中间件与 Deep Agent 配合使用

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
│         (实现 filesystem.Backend + Shell)                        │
├──────────────────────┬──────────────────────────────────────────┤
│  控制面              │  数据面                                    │
│  veFaaS SDK          │  AIO Sandbox SDK                          │
│  (创建/销毁/         │  (文件/Shell 操作，通过                     │
│   查询沙箱)          │   faasInstanceName 查询参数)               │
└──────────────────────┴──────────────────────────────────────────┘
```

## 后端接口映射

| filesystem.Backend 方法   | AIO Sandbox SDK API           |
|---------------------------|-------------------------------|
| LsInfo                    | File.ListPath                 |
| Read                      | File.ReadFile                 |
| Write                     | File.WriteFile                |
| Edit                      | File.ReplaceInFile            |
| GrepRaw                   | Ripgrep                       |
| GlobInfo                  | File.FindFiles                |
| Execute (Shell)           | Shell.ExecCommand             |

## 前置条件

示例支持两种模式：

### 模式一：直连（使用已有沙箱）

```bash
# 数据面 URL，通过 faasInstanceName 查询参数指定沙箱实例
AIO_SANDBOX_BASE_URL=https://xxx.apigateway-cn-beijing.volceapi.com?faasInstanceName=your-instance-name

# 可选：Bearer Token 认证
# AIO_SANDBOX_TOKEN=your-token
```

### 模式二：托管（自动创建/销毁沙箱）

```bash
# 火山引擎凭证，用于沙箱生命周期管理
VOLC_ACCESSKEY=your-access-key
VOLC_SECRETKEY=your-secret-key
VEFAAS_FUNCTION_ID=your-function-id
VEFAAS_GATEWAY_URL=https://xxx.apigateway-cn-beijing.volceapi.com
```

### 模型配置

```bash
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

### 直连模式

```go
// 通过数据面 URL 使用已有沙箱
backend, err := NewAIOSandboxBackend(ctx, &AIOSandboxBackendConfig{
    BaseURL: "https://xxx.apigateway-cn-beijing.volceapi.com?faasInstanceName=your-instance-name",
    WorkDir: "/tmp",
})
```

### 托管模式

```go
// 通过控制面创建沙箱
mgr, _ := NewSandboxManager(&SandboxManagerConfig{
    AccessKey:  "your-ak",
    SecretKey:  "your-sk",
    FunctionID: "your-function-id",
})

sandboxID, _ := mgr.CreateSandbox()
defer mgr.KillSandbox(sandboxID)

// 连接数据面
baseURL := mgr.DataPlaneBaseURL("https://xxx.apigateway-cn-beijing.volceapi.com", sandboxID)
backend, err := NewAIOSandboxBackend(ctx, &AIOSandboxBackendConfig{
    BaseURL: baseURL,
    WorkDir: "/tmp",
})
```

### 与 Agent 配合使用

```go
fsMW, _ := filesystem.NewMiddleware(ctx, &filesystem.Config{
    Backend: backend,
    Shell:   backend,
})

agent, _ := deep.New(ctx, &deep.Config{
    ChatModel:   chatModel,
    Middlewares: []adk.AgentMiddleware{fsMW},
})
```

## 运行示例

```bash
cd adk/middleware/aiosandbox-backend
go run .
```

## 实现自定义后端

要实现自定义文件系统后端，需要实现 `filesystem.Backend` 接口：

```go
type Backend interface {
    LsInfo(ctx context.Context, req *LsInfoRequest) ([]FileInfo, error)
    Read(ctx context.Context, req *ReadRequest) (*FileContent, error)
    GrepRaw(ctx context.Context, req *GrepRequest) ([]GrepMatch, error)
    GlobInfo(ctx context.Context, req *GlobInfoRequest) ([]FileInfo, error)
    Write(ctx context.Context, req *WriteRequest) error
    Edit(ctx context.Context, req *EditRequest) error
}
```

可选实现 `filesystem.Shell` 接口以提供 `execute` 工具：

```go
type Shell interface {
    Execute(ctx context.Context, input *ExecuteRequest) (*ExecuteResponse, error)
}
```
