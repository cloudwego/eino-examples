# AIO Sandbox Filesystem Middleware Example

This example demonstrates how to use the Deep Agent with [AIO Sandbox](https://github.com/agent-infra/sandbox)

You can access AIO Sandbox through [Volcano Engine veFaaS Sandbox](https://www.volcengine.com/docs/6662/1802770) to quickly get a secure isolated code execution environment.

This example demonstrates how to implement a custom `filesystem.Backend` and use it with `filesystem.NewMiddleware` to provide file system tools to an agent running in a remote AIO Sandbox environment.

## Overview

The `filesystem.Backend` interface allows you to plug in any file system implementation. This example shows how to:

1. Implement the `filesystem.Backend` interface using AIO Sandbox SDK (data plane)
2. Manage sandbox lifecycle via veFaaS API (control plane)
3. Create a filesystem middleware with the custom backend
4. Use the middleware with a deep agent

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         Deep Agent                               │
├─────────────────────────────────────────────────────────────────┤
│                    Filesystem Middleware                         │
│  (Auto-registers: ls, read_file, write_file, edit_file,        │
│                   glob, grep, execute tools)                     │
├─────────────────────────────────────────────────────────────────┤
│                  AIOSandboxBackend                               │
│         (implements filesystem.Backend + Shell)                  │
├──────────────────────┬──────────────────────────────────────────┤
│  Control Plane       │  Data Plane                               │
│  veFaaS SDK          │  AIO Sandbox SDK                          │
│  (Create/Kill/       │  (File/Shell operations via               │
│   Describe sandbox)  │   faasInstanceName query param)           │
└──────────────────────┴──────────────────────────────────────────┘
```

## Backend Interface Mapping

| filesystem.Backend Method | AIO Sandbox SDK API           |
|---------------------------|-------------------------------|
| LsInfo                    | File.ListPath                 |
| Read                      | File.ReadFile                 |
| Write                     | File.WriteFile                |
| Edit                      | File.ReplaceInFile            |
| GrepRaw                   | Ripgrep                       |
| GlobInfo                  | File.FindFiles                |
| Execute (Shell)           | Shell.ExecCommand             |

## Prerequisites

The example supports two modes:

### Mode 1: Direct (use existing sandbox)

```bash
# Data plane URL with faasInstanceName query parameter
AIO_SANDBOX_BASE_URL=https://xxx.apigateway-cn-beijing.volceapi.com?faasInstanceName=your-instance-name

# Optional: Bearer token authentication
# AIO_SANDBOX_TOKEN=your-token
```

### Mode 2: Managed (auto create/kill sandbox)

```bash
# Volcengine credentials for sandbox lifecycle management
VOLC_ACCESSKEY=your-access-key
VOLC_SECRETKEY=your-secret-key
VEFAAS_FUNCTION_ID=your-function-id
VEFAAS_GATEWAY_URL=https://xxx.apigateway-cn-beijing.volceapi.com
```

### Model configuration

```bash
# Option 1: OpenAI compatible
OPENAI_API_KEY=your-api-key
OPENAI_MODEL=gpt-4
OPENAI_BASE_URL=https://api.openai.com/v1

# Option 2: Ark (set MODEL_TYPE=ark)
# MODEL_TYPE=ark
# ARK_API_KEY=your-ark-api-key
# ARK_MODEL=your-model-name
# ARK_BASE_URL=https://ark.cn-beijing.volces.com/api/v3
```

## Usage

### Direct mode

```go
// Use an existing sandbox via data plane URL
backend, err := NewAIOSandboxBackend(ctx, &AIOSandboxBackendConfig{
    BaseURL: "https://xxx.apigateway-cn-beijing.volceapi.com?faasInstanceName=your-instance-name",
    WorkDir: "/tmp",
})
```

### Managed mode

```go
// Create sandbox via control plane
mgr, _ := NewSandboxManager(&SandboxManagerConfig{
    AccessKey:  "your-ak",
    SecretKey:  "your-sk",
    FunctionID: "your-function-id",
})

sandboxID, _ := mgr.CreateSandbox()
defer mgr.KillSandbox(sandboxID)

// Connect data plane
baseURL := mgr.DataPlaneBaseURL("https://xxx.apigateway-cn-beijing.volceapi.com", sandboxID)
backend, err := NewAIOSandboxBackend(ctx, &AIOSandboxBackendConfig{
    BaseURL: baseURL,
    WorkDir: "/tmp",
})
```

### Use with agent

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

## Run the Example

```bash
cd adk/middleware/aiosandbox-backend
go run .
```

## Implementing Your Own Backend

To implement a custom filesystem backend, implement the `filesystem.Backend` interface:

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

Optionally implement the `filesystem.Shell` interface to provide the `execute` tool:

```go
type Shell interface {
    Execute(ctx context.Context, input *ExecuteRequest) (*ExecuteResponse, error)
}
```
