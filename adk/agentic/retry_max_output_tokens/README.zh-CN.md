# Retry Max Output Tokens

[English README](./README.md)

## 示例说明

这个示例展示：当模型因为输出 token 预算不足而被截断时，Eino ADK Agent 如何通过 retry 自动调大 `max_output_tokens`，避免把不完整输出误判成 final answer。

示例场景是旅游路线规划。Agent 会先调用一次本地 `travel_route` 工具，拿到杭州 3 天 2 晚路线，然后输出最终中文行程。第一次模型调用会故意使用较小的 `max_output_tokens`。如果模型返回“长度导致截断”的信号，ADK 的 `ModelRetryConfig` 会拒绝这次输出，并用更大的 token 预算重试。

## 因为max_output_tokens引发的截断

在 Responses API 里，输出预算不足导致的截断通常表现为：

```text
status=incomplete
incomplete_details.reason=max_output_tokens
```

这不应该被当成真正的 final answer。此时可见输出可能只是半截；对于 reasoning 模型，甚至可能因为预算消耗在 reasoning 阶段而没有任何可见输出。

## Retry 策略

这个示例只针对 Responses API token 限制导致的截断做 retry：

- `max_output_tokens`
- `max_tokens` 或 `length`：部分兼容 provider 可能会用不同 reason 字符串表达同类 token-limit 问题

每次 retry 会调大输出 token 预算：

```go
var retryMaxTokens = []int{4096, 8192, 16384}
```

具体实现里，retry decision 通过 `einoModel.WithMaxTokens(nextMaxTokens)` 把新的 token 预算传给下一次模型调用。

如果真实原因是上下文窗口超限，仅仅调大 `max_output_tokens` 可能不够。

## 运行方式

在仓库根目录执行：

```bash
cd /path/to/eino-examples
```

OpenAI Responses 兼容接口：

```bash
export AGENTIC_MODEL_PROVIDER="openai"
export OPENAI_API_KEY="your-openai-api-key"
export OPENAI_MODEL_ID="your-openai-model-id"
export OPENAI_BASE_URL="your-openai-compatible-base-url" # optional

go run ./adk/agentic/retry_max_output_tokens
```

Ark Responses 兼容接口：

```bash
export AGENTIC_MODEL_PROVIDER="ark"
export ARK_API_KEY="your-ark-api-key"
export ARK_MODEL_ID="your-ark-model-id"
export ARK_BASE_URL="your-ark-compatible-base-url" # optional

go run ./adk/agentic/retry_max_output_tokens
```

## 预期输出

当初始输出预算过小时，终端会看到类似日志：

```text
[retry] attempt=1 rejected status=incomplete reason=max_output_tokens -> max_output_tokens=4096
```

retry 成功后，Agent 会输出最终的中文旅游路线。

## 关键文件

- `main.go`：创建 agentic model、runner，并设置较小的初始 token 预算。
- `agent.go`：构建 ADK travel agent，并挂载 `ModelRetryConfig`。
- `retry.go`：识别 token-limit stop reason，并调大 `max_output_tokens` 重试。
- `tools.go`：定义本地 `travel_route` 工具。
