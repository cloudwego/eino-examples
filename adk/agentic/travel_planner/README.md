# ✨ ADK Agentic Travel Planner

这个示例展示如何在 Eino ADK 层使用 Ark Responses API / Agentic API。它不是一个完整旅游产品，而是一个可运行的能力展台：同一个旅行规划场景，逐步展示 `AgenticModel` 相比传统模型接口更适合构建真实 Agent 的地方。

> 一句话：Responses API 是构建 Agent loop 的 API 原语。对于 Ark server-side tools，它可以在服务端编排“思考、选工具、调用工具、读结果、继续推理、最终回答”；对于本地 Function tools，工具执行仍由 ADK / 应用侧负责，但同样会被表示成可观测、可续接的 `AgenticMessage` 轨迹。

## 🚀 Why Agentic / Responses API

传统 ChatModel 接口更像一次“文本补全 + 可选工具调用”：应用侧通常要自己维护循环、拼接历史、解析工具调用、回填工具结果、追踪引用来源和 usage。随着 Agent 变复杂，这些逻辑很容易散落在业务代码里。

Agentic / Responses API 的核心变化是：**response 不再只是最终文本，而是一次 Agent 执行的结构化载体**。模型可以先分析问题、选择工具、调用工具、读取工具结果、继续推理，最后再生成面向用户的答案。

这里需要区分两类工具执行边界：

- ✅ **Server-side tools**：例如 Ark `web_search`、`image_process`、`knowledge_search`、Doubao App、Remote MCP。模型选择工具后，工具调用、结果回填和后续推理可以由 Ark Responses API 在服务端编排，更接近“一次 response 内部的 Agent loop”。
- 🧰 **Local Function tools**：例如本示例 Step 1 的 `lookup_user_profile`、`estimate_trip_cost`。模型会产出结构化 tool call，ADK / 应用侧执行工具，再把 tool result 回填给模型；这时 Agent loop 是由 Responses API 语义和 ADK Runner 共同完成，而不是完全发生在 Ark 服务端内部。

一次 Agent 执行轨迹可能自然包含这些结构化步骤：

- 🧠 **Analyze / Deep Thinking**：服务端承载 reasoning / thinking，必要时返回为独立 reasoning block。
- 🛠️ **Function Tools**：产出本地工具调用，由 ADK / 应用侧执行，例如查用户画像、查预算约束、估算成本、打分。
- 🌐 **Server Tools**：调用 Ark 内置工具，例如 `web_search`、`image_process`、`knowledge_search`、Doubao App。
- 🔌 **Remote MCP Tools**：把远端 MCP 工具纳入同一条服务端工具链路，减少应用侧胶水代码。
- 🧾 **Structured Trace**：tool call/result、server tool call/result、assistant text、citation、usage 都成为结构化 block 或 meta。
- 🔁 **Stateful Continuation**：response 返回 `response_id`，下一轮用 `previous_response_id` 续接，不必反复重传完整历史。

这意味着业务代码不再只拿到一段文本，而是拿到一条可以展示、调试、恢复、续写的 Agent 执行轨迹。更准确地说：server-side tools 的循环主要由 Ark 托管；local tools 的循环由 Eino ADK 承接；两者都通过 `AgenticMessage` 统一观察。

## 🌉 Why Eino AgenticAPI

Eino 在框架层把 Ark Responses API 的 agentic 表达接进来，让它不只是一个底层 HTTP 能力，而是可以被 ADK、Runner、工具系统和事件流直接使用。

- 🧩 **`schema.AgenticMessage`**：把 reasoning、assistant text、function tool、server tool、MCP、usage、response meta 统一表达成结构化消息。
- 🤖 **`model.AgenticModel`**：让模型接口天然支持 Agentic execution trace，而不是把复杂执行过程压扁成一段 assistant 文本。
- 🧰 **ADK 友好**：通过 `adk.NewTypedChatModelAgent[*schema.AgenticMessage]`，仍然使用熟悉的 ADK Runner、ToolsConfig 和事件流。
- 🔭 **框架级可观测语义**：response id、previous response id、thinking、tool call/result、citation、cached tokens、错误状态等信息都沉淀在 `AgenticMessage` / meta / extension 中，上层可以直接记录、渲染、回放或调试。
- 🧱 **清晰的组件边界**：本地工具仍由 Eino ADK 管理，Ark server tools、stateful response、image process 等能力通过 `AgenticModel` 选项和结构化消息接入；应用可以渐进引入 Agentic 能力，而不需要推翻现有 Eino 组件体系。

如果你的应用只是简单问答，传统 ChatModel 已经够用；如果你要做会搜索、会看图、会调用本地工具、会调用 Remote MCP、会续接状态、还能解释执行过程的 Agent，`AgenticModel` 会让能力边界更清楚，代码也更像 Agent 本身。

## 🧪 Project Scenario

本示例把这些能力放进一个连贯场景：用户周末去杭州看展、喝咖啡、轻松散步。Agent 先使用 ADK 本地工具做初版计划，再调用 Ark 服务端 `web_search` 查实时信息，随后使用 `image_process` 阅读景点或展馆平面图照片，定位入口、展区、咖啡区和出口并生成现场游览路线，最后用 `previous_response_id` 续接上下文并根据新约束调整方案。

## 🗺️ Demo Flow

本示例用四个 step 递进展示 AgenticArk 的能力。每个 step 都放在独立 Go 文件里，方便阅读和改造。

| Step | File | 展示重点 |
| --- | --- | --- |
| `plan` | `step_plan.go` | Ark deep thinking + ADK typed agent + 本地 Function tools |
| `search` | `step_search.go` | Ark server-side `web_search`、实时信息校验、citation |
| `visual` | `step_visual.go` | Ark server-side `image_process`、平面图理解、景点路线生成 |
| `stateful` | `step_stateful.go` | 先运行 Step 1 生成 session cache anchor，再用 `previous_response_id` 增量续接 |

### 🧠 Step 1: Local Tools + AgenticMessage

`plan` 阶段会显式开启 Ark 服务端 deep thinking，并设置 high reasoning effort，然后使用 ADK 本地工具读取用户画像、预算约束、费用估算和方案评分：

- `lookup_user_profile`
- `lookup_travel_policy`
- `estimate_trip_cost`
- `score_itinerary`

运行后可以看到清晰的工具链路：

```text
[response] id=... status=completed thinking=enabled
[reasoning]
...
[tool.call] lookup_user_profile args={}
[tool.result] lookup_user_profile ...
[tool.call] estimate_trip_cost args=...
[usage] input=... output=... total=... reasoning=...
```

这一步展示的是 Responses API 的结构化语义可以和 ADK 本地工具循环同时存在：Ark 服务端 reasoning / thinking 作为结构化 block 返回，本地业务工具仍然由 ADK 管理和执行。模型输入输出升级为 `AgenticMessage` 后，reasoning、tool call、tool result、usage 和 response id 都可以被统一观察和复用。代码也会为这一轮开启 session cache，让真实规划结果自然留下可续接的 response id；缓存机制本身集中放到 Step 4 讲解。

### 🌐 Step 2: Built-in Web Search

`search` 阶段把初版计划交给 Ark 服务端 `web_search`，让模型自己搜索并整合实时信息，例如天气、展馆开放时间、预约风险和排队备选。

相比把搜索做成应用侧本地工具，server-side `web_search` 的优势是：

- 模型可以围绕当前问题自动生成搜索查询。
- 搜索调用和最终回答都留在同一条 Responses API 轨迹中。
- 搜索引用会进入 text annotation，便于 UI 展示来源。
- ADK 侧不需要维护搜索 API、排序、摘要和引用拼装逻辑。

### 🖼️ Step 3: Image Process

`visual` 阶段模拟用户到达景点或展馆后，发来一张现场平面图、导览图或游客中心拍到的地图照片。Agent 会结合 Step 2 搜索得到的开放时间、天气、排队风险，把图片里的空间信息转化成一条可执行的游览路线。

这正好对应当前 `step_visual.go` 的实现方式：代码把用户图片作为 `schema.UserInputImage` 传给 AgenticArk，同时注册 Ark `image_process` server tool。模型可以按需执行：

- `point`：定位用户关心的位置，例如入口、当前所在点、咖啡区或出口。
- `grounding`：框选或定位图片中的目标区域，例如某个展区、游客服务中心或推荐路线段。
- `zoom`：放大小字、图例、楼层说明或拥挤区域标识。
- `rotate`：处理拍歪或方向不正确的平面图照片。

这一步比“识别一张现场照片”更有代表性：用户真正需要的不是图片描述，而是“我应该怎么走”。模型会先利用 grounding/zoom 等工具读懂平面图，再生成类似“入口 -> 低排队展区 -> 咖啡休息 -> 主展区 -> 出口”的路线，并把天气、排队、预算和节奏偏好一起纳入决策。

示例默认启用 `image_process`，使用与其他 step 相同的 `ARK_MODEL_ID`，并在代码里自动添加 Ark beta header。用户只需要准备一个支持视觉输入和 `image_process` 的 Ark endpoint，就可以直接运行 `--step=visual` 或 `--step=all`。

### 🔁 Step 4: Stateful Continuation

`stateful` 阶段集中展示缓存和续接。Step 1 的本地工具规划结果会作为 session cache anchor 缓存起来；Step 4 不再重传完整规划历史，而是显式设置 `previous_response_id`，只发送“实时搜索摘要、平面图视觉摘要、下雨、预算收紧、下午固定咖啡、少排队”等增量信息。

这展示了 Agentic API 的一个关键优势：长任务不一定要把所有历史 messages 重新塞回 prompt。真实工作流通常是先完成一次有价值的 Agent 执行并留下 stateful anchor，再在后续步骤用 `previous_response_id` 续接它。这样比在 Step 4 临时造一个缓存上下文更连贯，也更接近实际 Agent 应用。

单独运行 `--step=stateful` 时，示例会先自动运行 Step 1 创建真实的 response id，再运行 Step 4；为了聚焦缓存续接，它不会额外执行 `web_search` 和 `image_process`，而是使用紧凑 fallback 摘要补齐这两部分上下文。

## ▶️ Run

```bash
export ARK_API_KEY="<your ark api key>"
export ARK_MODEL_ID="<your ark endpoint id>"

go run ./adk/agentic/travel_planner --step=all
```

也可以单独运行某个阶段：

```bash
go run ./adk/agentic/travel_planner --step=plan
go run ./adk/agentic/travel_planner --step=search
go run ./adk/agentic/travel_planner --step=visual
go run ./adk/agentic/travel_planner --step=stateful
```

如果控制台里配置的是自定义 Endpoint ID，把 `ARK_MODEL_ID` 设置为对应 Endpoint ID 即可。

## 🖼️ Image Process

`image_process` 需要当前 `ARK_MODEL_ID` 指向支持视觉输入和图片处理工具的 Ark endpoint。示例不需要额外配置视觉模型、图片 URL 或开关；它会使用同一个模型，并自动添加：

```text
ark-beta-image-process: true
```

默认图片是一张公开的景点平面图，便于 example 直接运行。

`image_process` 当前不支持 `tool_choice` 强制选择，也不支持 `max_tool_calls`，因此示例保留为自动工具选择。

## 🔎 What To Notice

运行示例时，建议重点观察这些输出：

- `[response] id=... previous=...`：Responses API 的状态链路。
- `[tool.call]` / `[tool.result]`：ADK 本地工具调用。
- `[server_tool.call] web_search`：Ark 服务端联网搜索。
- `[server_tool.result] image_process action=grounding|zoom`：Ark 服务端对平面图进行区域定位或局部放大。
- `[usage] ... cached=...`：session cache 命中和 token 使用。
- `[citation.N] ...`：可展示给用户的引用来源。

这些信息如果用传统文本接口实现，通常需要应用侧额外记录和对齐；`AgenticModel` 会把它们作为 Agent 执行轨迹的一部分自然带回来。

## 📁 Files

- `main.go`：按 `--step` 调度完整 demo。
- `agent.go`：创建 `adk.NewTypedChatModelAgent[*schema.AgenticMessage]` 和 typed runner。
- `config.go`：创建 `agenticark.Model`，配置 session cache 和 `previous_response_id`。
- `tools.go`：本地 ADK 工具定义。
- `timeline.go`：打印 AgenticMessage 执行时间线。
- `step_plan.go`：本地工具规划。
- `step_search.go`：Ark `web_search`。
- `step_visual.go`：Ark `image_process`。
- `step_stateful.go`：stateful continuation。
