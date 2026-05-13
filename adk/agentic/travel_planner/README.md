# Agentic 旅行规划 🧳

这个示例展示如何在 Eino ADK 中使用 `agenticark.AgenticModel` 构建一个会自主循环执行的旅行规划 Agent。

它会为用户生成一份杭州周末两日旅行计划：先读取偏好和约束，再使用服务端 `web_search` 核验实时信息，随后经过自定义 middleware 检查预算和付款策略，最后把旅行计划写入本地文件。

## 为什么使用 Agentic API ✨

传统 Chat Completion 风格的 Agent 通常需要应用代码自己解析 tool call、执行工具、拼接 tool result，并反复请求模型。

Agentic/Responses API 会把这些过程结构化为 `reasoning`、`server_tool_call`、`function_tool_call`、`tool_result`、`assistant_gen_text` 和 `response_meta` 等内容。Eino 将它们保存在 `schema.AgenticMessage` 中，ADK 可以直接基于这些结构驱动 agent loop。

这个示例重点展示：

- 🧠 `thinking/reasoning`：观察模型如何规划下一步；
- 🔎 服务端 `web_search`：核验天气、展览、场馆开放等实时信息；
- 🛠️ 本地 function tools：读取用户偏好、行程约束，并评估候选体验；
- 📝 filesystem middleware：让 Agent 自己写入 `travel_plan.md`；
- 🛡️ 自定义 middleware：拒绝超预算或自动付款等不符合策略的动作。

## Agent 执行流程

```text
lookup_user_profile
  ↓
lookup_travel_constraints
  ↓
server tool: web_search
  ↓
evaluate_booking_option
  ↓
travelPolicyMiddleware -> policy_rejected
  ↓
write_file -> workspace/travel_plan.md
  ↓
assistant final answer
```

这里的关键点是：旅行计划不是单轮生成的，而是在 ADK runner 中由模型多轮思考、调用工具、读取结果后完成。

## 运行方式 ▶️

```bash
export ARK_API_KEY="your-ark-api-key"
export ARK_MODEL_ID="your-ark-model-id"

go run ./adk/agentic/travel_planner
```

可选配置：

```bash
export ARK_BASE_URL="your-ark-base-url"
```

运行完成后会生成：

```text
adk/agentic/travel_planner/workspace/travel_plan.md
```

## 代码结构 🧩

| 文件 | 作用 |
| --- | --- |
| `main.go` | 创建 runner 并启动 Agent |
| `model.go` | 创建 `agenticark.AgenticModel`，配置 thinking、reasoning 和 `web_search` |
| `agent.go` | 组装 ADK Agent、本地工具和 middleware |
| `prompt.go` | 定义 Agent 指令和用户请求 |
| `tools.go` | 定义本地 function tools |
| `middleware.go` | 定义预算和付款策略 middleware |
| `printer.go` | 打印 `AgenticMessage` 事件 |

## Middleware 示例 🛡️

`travelPolicyMiddleware` 使用 `WrapInvokableToolCall` 包装 `evaluate_booking_option`。当候选项超过预置单项预算，或模型尝试执行 `pay` / `purchase` 等付款动作时，middleware 会直接返回策略拒绝：

```json
{
  "status": "policy_rejected",
  "allowed": false,
  "intercepted_by": "travelPolicyMiddleware"
}
```

模型会继续 loop，并选择更低成本、无需付款的替代方案。这展示的是 middleware 的策略切面能力，而不是完整的 human-in-the-loop 中断恢复流程。
