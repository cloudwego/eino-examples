# Retry Max Output Tokens

[中文说明](./README.zh-CN.md)

## What This Example Shows

This example demonstrates how an Eino ADK agent can recover when a model output is cut off by the output token budget.

The agent plans a 3-day Hangzhou travel route. It calls the local `travel_route` tool once, then writes the final itinerary. The first model call uses a deliberately small `max_output_tokens` value. If the provider reports that the response was truncated, the ADK model retry config rejects that output and retries with a larger token budget.

## Truncation Caused by `max_output_tokens`

In the Responses API, truncation caused by insufficient output budget is usually represented as:

```text
status=incomplete
incomplete_details.reason=max_output_tokens
```

This is not a real final answer. The visible output may be partial, and for reasoning models there may be no visible output at all because the token budget was consumed during reasoning.

## Retry Strategy

This example only retries truncation caused by Responses API token limits:

- `max_output_tokens`
- `max_tokens` or `length`: some compatible providers may use different reason strings for the same token-limit condition

On each retry, it increases the output token budget:

```go
var retryMaxTokens = []int{4096, 8192, 16384}
```

The retry decision uses `einoModel.WithMaxTokens(nextMaxTokens)` to pass the larger budget to the next model call.

If the real issue is that the context window is exceeded, increasing `max_output_tokens` alone may not be enough.

## Run

From the repo root:

```bash
cd /path/to/eino-examples
```

OpenAI Responses compatible endpoint:

```bash
export AGENTIC_MODEL_PROVIDER="openai"
export OPENAI_API_KEY="your-openai-api-key"
export OPENAI_MODEL_ID="your-openai-model-id"
export OPENAI_BASE_URL="your-openai-compatible-base-url" # optional

go run ./adk/agentic/retry_max_output_tokens
```

Ark Responses compatible endpoint:

```bash
export AGENTIC_MODEL_PROVIDER="ark"
export ARK_API_KEY="your-ark-api-key"
export ARK_MODEL_ID="your-ark-model-id"
export ARK_BASE_URL="your-ark-compatible-base-url" # optional

go run ./adk/agentic/retry_max_output_tokens
```

## Expected Output

When the initial output budget is too small, the terminal shows retry logs similar to:

```text
[retry] attempt=1 rejected status=incomplete reason=max_output_tokens -> max_output_tokens=4096
```

After a successful retry, the agent prints the final Chinese travel itinerary.

## Key Files

- `main.go`: creates the agentic model, runner, and initial small token budget.
- `agent.go`: builds the ADK travel agent and attaches `ModelRetryConfig`.
- `retry.go`: detects token-limit stop reasons and increases `max_output_tokens`.
- `tools.go`: defines the single local `travel_route` tool used by the agent.
