# AIO Sandbox Filesystem Middleware Example

This example demonstrates how to use the Deep Agent with [AIO Sandbox](https://github.com/agent-infra/sandbox) 

You can access AIO Sandbox through [Volcano Engine veFaaS Sandbox](https://www.volcengine.com/docs/6662/1802770) to quickly get a secure isolated code execution environment.

This example demonstrates how to implement a custom `filesystem.Backend` and use it with `filesystem.NewMiddleware` to provide file system tools to an agent running in a remote AIO Sandbox environment.

## Overview

The `filesystem.Backend` interface allows you to plug in any file system implementation. This example shows how to:

1. Implement the `filesystem.Backend` interface using AIO Sandbox SDK
2. Create a filesystem middleware with the custom backend
3. Use the middleware with a deep agent

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
│         (implements filesystem.Backend + ShellBackend)           │
├─────────────────────────────────────────────────────────────────┤
│                   AIO Sandbox SDK                                │
│              (Remote sandbox environment)                        │
└─────────────────────────────────────────────────────────────────┘
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
| Execute (ShellBackend)    | Shell.ExecCommand             |

## Prerequisites

Set the following environment variables:

```bash
# AIO Sandbox configuration (without trailing slash)
AIO_SANDBOX_BASE_URL=https://xxx.apigateway-cn-beijing.volceapi.com

# Model configuration (choose one provider)

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

### Basic Usage

```go
// Create custom filesystem backend
backend, err := NewAIOSandboxBackend(ctx, &AIOSandboxBackendConfig{
    BaseURL: "https://xxx.apigateway-cn-beijing.volceapi.com",
    WorkDir: "/tmp",
})

// Create filesystem middleware with custom backend
fsMW, err := filesystem.NewMiddleware(ctx, &filesystem.Config{
    Backend: backend,
})

// Use with agent
agent, err := deep.New(ctx, &deep.Config{
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
    Read(ctx context.Context, req *ReadRequest) (string, error)
    GrepRaw(ctx context.Context, req *GrepRequest) ([]GrepMatch, error)
    GlobInfo(ctx context.Context, req *GlobInfoRequest) ([]FileInfo, error)
    Write(ctx context.Context, req *WriteRequest) error
    Edit(ctx context.Context, req *EditRequest) error
}
```

```go
type ShellBackend interface {
    Backend
    Execute(ctx context.Context, input *ExecuteRequest) (*ExecuteResponse, error)
}
```

## Comparison with CommandLine Approach

This example uses the **Middleware approach** with `filesystem.Backend`:
- Automatically registers 7 tools: `ls`, `read_file`, `write_file`, `edit_file`, `glob`, `grep`, `execute`
- Includes built-in large result offloading
- Clean interface-based design

The [deep-aiosandbox](../../multiagent/deep-aiosandbox) example uses the **Tool approach** with `commandline.Operator`:
- Manually registers tools: `bash`, `read_file`, `edit_file`, `tree`
- More flexible for custom tool combinations
- Requires more setup code
