# Deep Agent with AIO Sandbox

[中文文档](./README_zh.md)

This example demonstrates how to use the Deep Agent with [AIO Sandbox](https://github.com/agent-infra/sandbox) for remote sandboxed execution. Instead of running commands locally, all file operations and command executions happen in a secure remote sandbox environment.

## What is AIO Sandbox?

AIO Sandbox is a secure remote code execution environment that supports:
- Shell command execution
- Python code running
- File read/write operations
- Jupyter Notebook
- Browser automation

You can access AIO Sandbox through [Volcano Engine veFaaS](https://www.volcengine.com/docs/6662/1802770) to quickly get a secure isolated code execution environment.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        Deep Agent                            │
│  ┌─────────────────┐    ┌─────────────────┐                 │
│  │   ExcelAgent    │    │  WebSearchAgent │                 │
│  │   (Top-level)   │    │   (Sub-agent)   │                 │
│  └────────┬────────┘    └─────────────────┘                 │
│           │                                                  │
│           ▼                                                  │
│  ┌─────────────────┐                                        │
│  │   CodeAgent     │  ← Python/Bash execution               │
│  │   (Sub-agent)   │                                        │
│  └────────┬────────┘                                        │
└───────────┼─────────────────────────────────────────────────┘
            │
            ▼
┌─────────────────────────────────────────────────────────────┐
│                     AIO Sandbox                              │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │
│  │    Bash     │  │    File     │  │   Python    │         │
│  │   Commands  │  │  Operations │  │  Execution  │         │
│  └─────────────┘  └─────────────┘  └─────────────┘         │
│                                                              │
│              Remote Sandboxed Environment                    │
└─────────────────────────────────────────────────────────────┘
```

## Features

- **Remote Execution**: All commands run in a secure cloud sandbox
- **Session Persistence**: Environment variables and state persist across commands
- **File Operations**: Read, write, and manage files in the sandbox
- **Excel Processing**: Create, read, and manipulate Excel files using pandas/openpyxl
- **Python Execution**: Run Python scripts in the sandbox environment

## Prerequisites

1. Access to an AIO Sandbox API endpoint
2. Volcano Ark API (recommended) or OpenAI-compatible LLM API

## Configuration

Copy the example environment file and fill in your credentials:

```bash
cp .example.env .env
```

Edit `.env` with your configuration:

```bash
# AIO Sandbox (Required)
AIO_SANDBOX_BASE_URL=https://xxxx.apigateway-cn-beijing.volceapi.com

# LLM - Volcano Ark API (Recommended)
ARK_API_KEY=your-ark-api-key
ARK_MODEL=your-model-endpoint
ARK_BASE_URL=https://ark.cn-beijing.volces.com/api/v3

# Or use OpenAI Compatible API
# OPENAI_API_KEY=your-api-key
# OPENAI_MODEL=gpt-4
# OPENAI_BASE_URL=https://api.openai.com/v1
```

## Usage

### Run the example

```bash
export AIO_SANDBOX_BASE_URL="https://xxxx.apigateway-cn-beijing.volceapi.com"
export ARK_API_KEY="your-ark-api-key"
export ARK_MODEL="your-model-endpoint"
export ARK_BASE_URL="https://ark.cn-beijing.volces.com/api/v3"

go run .
```

### Example Queries

The example includes several query options in `main.go`:

**1. Simple Test**
```go
query := schema.UserMessage("Execute echo 'Hello from AIO Sandbox' and create a test.txt file with current time")
```

**2. Excel Processing**
```go
query := schema.UserMessage(`Please complete the following tasks:
1. Create an Excel file named sales.xlsx with the following sales data:
   - Columns: Product Name, Quantity, Unit Price, Sales Amount
   - Data: Apple/100/5/500, Banana/200/3/600, Orange/150/4/600, Grape/80/8/640
2. Calculate total sales and add to the last row
3. Read and display file contents`)
```

## Sample Output

```
AIO Sandbox connected, Session ID: 577ab099-e73d-4d6a-93da-7182e3c98086
Task ID: f7a76387-fb9d-4c54-ab4c-191353405eee
Work Directory: /tmp/f7a76387-fb9d-4c54-ab4c-191353405eee

=== Files in work directory ===
-rw-r--r-- 1 gem gem  5159 Jan 28 22:14 sales.xlsx
-rw-r--r-- 1 gem gem  1229 Jan 28 22:14 *.py
-rw-r--r-- 1 gem gem   844 Jan 28 22:14 sales_summary.txt
```

## Key Differences from Local Deep Agent

| Aspect | Local (deep/) | AIO Sandbox (deep-aiosandbox/) |
|--------|---------------|-------------------------------|
| Operator | `LocalOperator` | `aiosandbox.AIOSandbox` |
| Execution | Local machine | Remote sandbox |
| File Access | Local filesystem | Sandbox filesystem |
| Security | Full local access | Isolated environment |
| Dependencies | Local Python/tools | Pre-installed in sandbox |

## Code Changes

The main change is replacing `LocalOperator` with `AIOSandbox`:

```go
// Before (local execution)
operator := &LocalOperator{}

// After (remote sandbox execution)
sandbox, err := aiosandbox.NewAIOSandbox(ctx, &aiosandbox.Config{
    BaseURL:     os.Getenv("AIO_SANDBOX_BASE_URL"),
    Token:       os.Getenv("AIO_SANDBOX_TOKEN"),
    WorkDir:     "/tmp",
    Timeout:     120,
    KeepSession: true,
})

// The rest of the code remains the same!
ca, err := agents.NewCodeAgent(ctx, sandbox)
```

## Troubleshooting

### "no model config" error
Make sure you have set either `OPENAI_API_KEY` or `ARK_API_KEY` environment variable.

### Connection timeout
Check if your `AIO_SANDBOX_BASE_URL` is correct and accessible.

### Permission denied in sandbox
Use `/tmp` as the working directory instead of `/workspace`.

## Related

- [AIO Sandbox](https://github.com/agent-infra/sandbox)
- [AIO Sandbox Go SDK](https://github.com/agent-infra/sandbox-sdk-go)
- [Volcano Engine veFaaS](https://www.volcengine.com/docs/6662/1802770)
- [Local Deep Agent Example](../deep/)
- [Eino Framework](https://github.com/cloudwego/eino)
