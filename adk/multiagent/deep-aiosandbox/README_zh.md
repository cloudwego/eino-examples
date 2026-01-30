# Deep Agent 集成 AIO Sandbox

[English](./README.md)

本示例演示如何将 Deep Agent 与 [AIO Sandbox](https://github.com/agent-infra/sandbox) 集成，实现远程沙箱执行。所有文件操作和命令执行都在安全的远程沙箱环境中进行，而非本地机器。

## 什么是 AIO Sandbox？

AIO Sandbox 是一个安全的远程代码执行环境，支持：
- Shell 命令执行
- Python 代码运行
- 文件读写操作
- Jupyter Notebook
- 浏览器自动化

你可以通过 [火山引擎 veFaaS 接入 AIO Sandbox](https://www.volcengine.com/docs/6662/1802770)，快速获得一个安全隔离的代码执行环境。

## 架构

```
┌─────────────────────────────────────────────────────────────┐
│                        Deep Agent                            │
│  ┌─────────────────┐    ┌─────────────────┐                 │
│  │   ExcelAgent    │    │  WebSearchAgent │                 │
│  │    (顶层代理)    │    │    (子代理)     │                 │
│  └────────┬────────┘    └─────────────────┘                 │
│           │                                                  │
│           ▼                                                  │
│  ┌─────────────────┐                                        │
│  │   CodeAgent     │  ← Python/Bash 执行                    │
│  │    (子代理)      │                                        │
│  └────────┬────────┘                                        │
└───────────┼─────────────────────────────────────────────────┘
            │
            ▼
┌─────────────────────────────────────────────────────────────┐
│                     AIO Sandbox                              │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │
│  │    Bash     │  │    文件     │  │   Python    │         │
│  │    命令     │  │    操作     │  │    执行     │         │
│  └─────────────┘  └─────────────┘  └─────────────┘         │
│                                                              │
│                   远程沙箱执行环境                            │
└─────────────────────────────────────────────────────────────┘
```

## 特性

- **远程执行**：所有命令在安全的云端沙箱中运行
- **会话持久化**：环境变量和状态在多次命令调用间保持
- **文件操作**：在沙箱中读取、写入和管理文件
- **Excel 处理**：使用 pandas/openpyxl 创建、读取和操作 Excel 文件
- **Python 执行**：在沙箱环境中运行 Python 脚本

## 前置条件

1. AIO Sandbox API 端点访问权限
2. 火山方舟 API（推荐）或 OpenAI 兼容的 LLM API

## 配置

复制示例环境文件并填入你的凭证：

```bash
cp .example.env .env
```

编辑 `.env` 配置：

```bash
# AIO Sandbox（必需）
AIO_SANDBOX_BASE_URL=https://xxxx.apigateway-cn-beijing.volceapi.com

# LLM - 火山方舟 API（推荐）
ARK_API_KEY=your-ark-api-key
ARK_MODEL=your-model-endpoint
ARK_BASE_URL=https://ark.cn-beijing.volces.com/api/v3

# 或使用 OpenAI 兼容 API
# OPENAI_API_KEY=your-api-key
# OPENAI_MODEL=gpt-4
# OPENAI_BASE_URL=https://api.openai.com/v1
```

## 使用方法

### 运行示例

```bash
export AIO_SANDBOX_BASE_URL="https://xxxx.apigateway-cn-beijing.volceapi.com"
export ARK_API_KEY="your-ark-api-key"
export ARK_MODEL="your-model-endpoint"
export ARK_BASE_URL="https://ark.cn-beijing.volces.com/api/v3"

go run .
```

### 示例查询

示例在 `main.go` 中包含多个查询选项：

**1. 简单测试**
```go
query := schema.UserMessage("请帮我执行 echo 'Hello from AIO Sandbox' 并创建一个 test.txt 文件写入当前时间")
```

**2. Excel 处理**
```go
query := schema.UserMessage(`请帮我完成以下任务：
1. 创建一个名为 sales.xlsx 的 Excel 文件，包含以下销售数据：
   - 列：产品名称、销售数量、单价、销售额
   - 数据：苹果/100/5/500, 香蕉/200/3/600, 橙子/150/4/600, 葡萄/80/8/640
2. 计算总销售额并添加到最后一行
3. 读取文件内容并展示`)
```

## 示例输出

```
AIO Sandbox connected, Session ID: 577ab099-e73d-4d6a-93da-7182e3c98086
Task ID: f7a76387-fb9d-4c54-ab4c-191353405eee
Work Directory: /tmp/f7a76387-fb9d-4c54-ab4c-191353405eee

=== Files in work directory ===
-rw-r--r-- 1 gem gem  5159 Jan 28 22:14 sales.xlsx
-rw-r--r-- 1 gem gem  1229 Jan 28 22:14 *.py
-rw-r--r-- 1 gem gem   844 Jan 28 22:14 sales_summary.txt
```

## 与本地 Deep Agent 的主要区别

| 方面 | 本地版 (deep/) | AIO Sandbox 版 (deep-aiosandbox/) |
|------|---------------|----------------------------------|
| Operator | `LocalOperator` | `aiosandbox.AIOSandbox` |
| 执行位置 | 本地机器 | 远程沙箱 |
| 文件访问 | 本地文件系统 | 沙箱文件系统 |
| 安全性 | 完全本地访问 | 隔离环境 |
| 依赖 | 本地 Python/工具 | 沙箱预装 |

## 代码改动

主要改动是将 `LocalOperator` 替换为 `AIOSandbox`：

```go
// 之前（本地执行）
operator := &LocalOperator{}

// 之后（远程沙箱执行）
sandbox, err := aiosandbox.NewAIOSandbox(ctx, &aiosandbox.Config{
    BaseURL:     os.Getenv("AIO_SANDBOX_BASE_URL"),
    Token:       os.Getenv("AIO_SANDBOX_TOKEN"),
    WorkDir:     "/tmp",
    Timeout:     120,
    KeepSession: true,
})

// 其余代码完全不变！
ca, err := agents.NewCodeAgent(ctx, sandbox)
```

## 故障排查

### "no model config" 错误
确保已设置 `OPENAI_API_KEY` 或 `ARK_API_KEY` 环境变量。

### 连接超时
检查 `AIO_SANDBOX_BASE_URL` 是否正确且可访问。

### 沙箱中权限被拒绝
使用 `/tmp` 作为工作目录，而不是 `/workspace`。

## 相关链接

- [AIO Sandbox](https://github.com/agent-infra/sandbox)
- [AIO Sandbox Go SDK](https://github.com/agent-infra/sandbox-sdk-go)
- [火山引擎 veFaaS](https://www.volcengine.com/docs/6662/1802770)
- [本地 Deep Agent 示例](../deep/)
- [Eino 框架](https://github.com/cloudwego/eino)
