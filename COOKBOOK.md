# Eino Examples Cookbook

本文档为 eino-examples 项目的示例索引，帮助开发者快速找到所需的示例代码。

**GitHub 仓库**: https://github.com/cloudwego/eino-examples

---

## 📦 ADK (Agent Development Kit)

### Hello World
| 目录 | 名称 | 说明 |
|------|------|------|
| [adk/helloworld](https://github.com/cloudwego/eino-examples/tree/main/adk/helloworld) | Hello World Agent | 最简单的 Agent 示例，展示如何创建一个基础的对话 Agent |

### 入门示例 (Intro)
| 目录 | 名称 | 说明 |
|------|------|------|
| [adk/intro/chatmodel](https://github.com/cloudwego/eino-examples/tree/main/adk/intro/chatmodel) | ChatModel Agent | 展示如何使用 ChatModelAgent 并配合 Interrupt 机制 |
| [adk/intro/custom](https://github.com/cloudwego/eino-examples/tree/main/adk/intro/custom) | 自定义 Agent | 展示如何实现符合 ADK 定义的自定义 Agent |
| [adk/intro/workflow/loop](https://github.com/cloudwego/eino-examples/tree/main/adk/intro/workflow/loop) | Loop Agent | 展示如何使用 LoopAgent 实现循环反思模式 |
| [adk/intro/workflow/parallel](https://github.com/cloudwego/eino-examples/tree/main/adk/intro/workflow/parallel) | Parallel Agent | 展示如何使用 ParallelAgent 实现并行执行 |
| [adk/intro/workflow/sequential](https://github.com/cloudwego/eino-examples/tree/main/adk/intro/workflow/sequential) | Sequential Agent | 展示如何使用 SequentialAgent 实现顺序执行 |
| [adk/intro/session](https://github.com/cloudwego/eino-examples/tree/main/adk/intro/session) | Session 管理 | 展示如何通过 Session 在多个 Agent 之间传递数据和状态 |
| [adk/intro/transfer](https://github.com/cloudwego/eino-examples/tree/main/adk/intro/transfer) | Agent 转移 | 展示 ChatModelAgent 的 Transfer 能力，实现 Agent 间的任务转移 |
| [adk/intro/agent_with_summarization](https://github.com/cloudwego/eino-examples/tree/main/adk/intro/agent_with_summarization) | 带摘要的 Agent | 展示如何为 Agent 添加对话摘要功能 |
| [adk/intro/http-sse-service](https://github.com/cloudwego/eino-examples/tree/main/adk/intro/http-sse-service) | HTTP SSE 服务 | 展示如何将 ADK Runner 暴露为支持 Server-Sent Events 的 HTTP 服务 |

### Human-in-the-Loop (人机协作)
| 目录 | 名称 | 说明 |
|------|------|------|
| [adk/human-in-the-loop/1_approval](https://github.com/cloudwego/eino-examples/tree/main/adk/human-in-the-loop/1_approval) | 审批模式 | 展示敏感操作前的人工审批机制，Agent 执行前需用户确认 |
| [adk/human-in-the-loop/2_review-and-edit](https://github.com/cloudwego/eino-examples/tree/main/adk/human-in-the-loop/2_review-and-edit) | 审核编辑模式 | 展示工具调用参数的人工审核和编辑，支持修改、批准或拒绝 |
| [adk/human-in-the-loop/3_feedback-loop](https://github.com/cloudwego/eino-examples/tree/main/adk/human-in-the-loop/3_feedback-loop) | 反馈循环模式 | 多 Agent 协作，Writer 生成内容，Reviewer 收集人工反馈，支持迭代优化 |
| [adk/human-in-the-loop/4_follow-up](https://github.com/cloudwego/eino-examples/tree/main/adk/human-in-the-loop/4_follow-up) | 追问模式 | 智能识别信息缺失，通过多轮追问收集用户需求，完成复杂任务规划 |
| [adk/human-in-the-loop/5_supervisor](https://github.com/cloudwego/eino-examples/tree/main/adk/human-in-the-loop/5_supervisor) | Supervisor + 审批 | Supervisor 多 Agent 模式结合审批机制，敏感操作需人工确认 |
| [adk/human-in-the-loop/6_plan-execute-replan](https://github.com/cloudwego/eino-examples/tree/main/adk/human-in-the-loop/6_plan-execute-replan) | 计划执行重规划 + 审核编辑 | Plan-Execute-Replan 模式结合参数审核编辑，支持预订参数修改 |
| [adk/human-in-the-loop/7_deep-agents](https://github.com/cloudwego/eino-examples/tree/main/adk/human-in-the-loop/7_deep-agents) | Deep Agents + 追问 | Deep Agents 模式结合追问机制，在分析前主动收集用户偏好 |
| [adk/human-in-the-loop/8_supervisor-plan-execute](https://github.com/cloudwego/eino-examples/tree/main/adk/human-in-the-loop/8_supervisor-plan-execute) | 嵌套多 Agent + 审批 | Supervisor 嵌套 Plan-Execute-Replan 子 Agent，支持深层嵌套中断 |

### Multi-Agent (多 Agent 协作)
| 目录 | 名称 | 说明 |
|------|------|------|
| [adk/multiagent/supervisor](https://github.com/cloudwego/eino-examples/tree/main/adk/multiagent/supervisor) | Supervisor Agent | 基础的 Supervisor 多 Agent 模式，协调多个子 Agent 完成任务 |
| [adk/multiagent/layered-supervisor](https://github.com/cloudwego/eino-examples/tree/main/adk/multiagent/layered-supervisor) | 分层 Supervisor | 多层 Supervisor 嵌套，一个 Supervisor 作为另一个的子 Agent |
| [adk/multiagent/plan-execute-replan](https://github.com/cloudwego/eino-examples/tree/main/adk/multiagent/plan-execute-replan) | Plan-Execute-Replan | 计划-执行-重规划模式，支持动态调整执行计划 |
| [adk/multiagent/integration-project-manager](https://github.com/cloudwego/eino-examples/tree/main/adk/multiagent/integration-project-manager) | 项目管理器 | 使用 Supervisor 模式的项目管理示例，包含 Coder、Researcher、Reviewer |
| [adk/multiagent/deep](https://github.com/cloudwego/eino-examples/tree/main/adk/multiagent/deep) | Deep Agents (Excel Agent) | 智能 Excel 助手，分步骤理解和处理 Excel 文件，支持 Python 代码执行 |
| [adk/multiagent/integration-excel-agent](https://github.com/cloudwego/eino-examples/tree/main/adk/multiagent/integration-excel-agent) | Excel Agent (ADK 集成版) | ADK 集成版 Excel Agent，包含 Planner、Executor、Replanner、Reporter |

### GraphTool (图工具)
| 目录 | 名称 | 说明 |
|------|------|------|
| [adk/common/tool/graphtool](https://github.com/cloudwego/eino-examples/tree/main/adk/common/tool/graphtool) | GraphTool 包 | 将 Graph/Chain/Workflow 封装为 Agent 工具的工具包 |
| [adk/common/tool/graphtool/examples/1_chain_summarize](https://github.com/cloudwego/eino-examples/tree/main/adk/common/tool/graphtool/examples/1_chain_summarize) | Chain 文档摘要 | 使用 compose.Chain 实现文档摘要工具 |
| [adk/common/tool/graphtool/examples/2_graph_research](https://github.com/cloudwego/eino-examples/tree/main/adk/common/tool/graphtool/examples/2_graph_research) | Graph 多源研究 | 使用 compose.Graph 实现并行多源搜索和流式输出 |
| [adk/common/tool/graphtool/examples/3_workflow_order](https://github.com/cloudwego/eino-examples/tree/main/adk/common/tool/graphtool/examples/3_workflow_order) | Workflow 订单处理 | 使用 compose.Workflow 实现订单处理，结合审批机制 |
| [adk/common/tool/graphtool/examples/4_nested_interrupt](https://github.com/cloudwego/eino-examples/tree/main/adk/common/tool/graphtool/examples/4_nested_interrupt) | 嵌套中断 | 展示外层审批和内层风控的双层中断机制 |

---

## 🔗 Compose (编排)

### Chain (链式编排)
| 目录 | 名称 | 说明 |
|------|------|------|
| [compose/chain](https://github.com/cloudwego/eino-examples/tree/main/compose/chain) | Chain 基础示例 | 展示如何使用 compose.Chain 进行顺序编排，包含 Prompt + ChatModel |

### Graph (图编排)
| 目录 | 名称 | 说明 |
|------|------|------|
| [compose/graph/simple](https://github.com/cloudwego/eino-examples/tree/main/compose/graph/simple) | 简单 Graph | Graph 基础用法示例 |
| [compose/graph/state](https://github.com/cloudwego/eino-examples/tree/main/compose/graph/state) | State Graph | 带状态的 Graph 示例 |
| [compose/graph/tool_call_agent](https://github.com/cloudwego/eino-examples/tree/main/compose/graph/tool_call_agent) | Tool Call Agent | 使用 Graph 构建工具调用 Agent |
| [compose/graph/tool_call_once](https://github.com/cloudwego/eino-examples/tree/main/compose/graph/tool_call_once) | 单次工具调用 | 展示单次工具调用的 Graph 实现 |
| [compose/graph/two_model_chat](https://github.com/cloudwego/eino-examples/tree/main/compose/graph/two_model_chat) | 双模型对话 | 两个模型相互对话的 Graph 示例 |
| [compose/graph/async_node](https://github.com/cloudwego/eino-examples/tree/main/compose/graph/async_node) | 异步节点 | 展示异步 Lambda 节点，包含报告生成和实时转录场景 |
| [compose/graph/react_with_interrupt](https://github.com/cloudwego/eino-examples/tree/main/compose/graph/react_with_interrupt) | ReAct + 中断 | 票务预订场景，展示 Interrupt 和 Checkpoint 实践 |

### Workflow (工作流编排)
| 目录 | 名称 | 说明 |
|------|------|------|
| [compose/workflow/1_simple](https://github.com/cloudwego/eino-examples/tree/main/compose/workflow/1_simple) | 简单 Workflow | 最简单的 Workflow 示例，等价于 Graph |
| [compose/workflow/2_field_mapping](https://github.com/cloudwego/eino-examples/tree/main/compose/workflow/2_field_mapping) | 字段映射 | 展示 Workflow 的字段映射功能 |
| [compose/workflow/3_data_only](https://github.com/cloudwego/eino-examples/tree/main/compose/workflow/3_data_only) | 纯数据流 | 仅数据流的 Workflow 示例 |
| [compose/workflow/4_control_only_branch](https://github.com/cloudwego/eino-examples/tree/main/compose/workflow/4_control_only_branch) | 控制流分支 | 仅控制流的分支示例 |
| [compose/workflow/5_static_values](https://github.com/cloudwego/eino-examples/tree/main/compose/workflow/5_static_values) | 静态值 | 展示如何在 Workflow 中使用静态值 |
| [compose/workflow/6_stream_field_map](https://github.com/cloudwego/eino-examples/tree/main/compose/workflow/6_stream_field_map) | 流式字段映射 | 流式场景下的字段映射 |

### Batch (批处理)
| 目录 | 名称 | 说明 |
|------|------|------|
| [compose/batch](https://github.com/cloudwego/eino-examples/tree/main/compose/batch) | BatchNode | 批量处理组件，支持并发控制、中断恢复，适用于文档批量审核等场景 |

---

## 🌊 Flow (流程模块)

### ReAct Agent
| 目录 | 名称 | 说明 |
|------|------|------|
| [flow/agent/react](https://github.com/cloudwego/eino-examples/tree/main/flow/agent/react) | ReAct Agent | ReAct Agent 基础示例，餐厅推荐场景 |
| [flow/agent/react/memory_example](https://github.com/cloudwego/eino-examples/tree/main/flow/agent/react/memory_example) | 短期记忆 | ReAct Agent 的短期记忆实现，支持内存和 Redis 存储 |
| [flow/agent/react/dynamic_option_example](https://github.com/cloudwego/eino-examples/tree/main/flow/agent/react/dynamic_option_example) | 动态选项 | 运行时动态修改 Model Option，控制思考模式和工具选择 |
| [flow/agent/react/unknown_tool_handler_example](https://github.com/cloudwego/eino-examples/tree/main/flow/agent/react/unknown_tool_handler_example) | 未知工具处理 | 处理模型幻觉产生的未知工具调用，提高 Agent 鲁棒性 |

### Multi-Agent
| 目录 | 名称 | 说明 |
|------|------|------|
| [flow/agent/multiagent/host/journal](https://github.com/cloudwego/eino-examples/tree/main/flow/agent/multiagent/host/journal) | 日记助手 | Host Multi-Agent 示例，支持写日记、读日记、根据日记回答问题 |
| [flow/agent/multiagent/plan_execute](https://github.com/cloudwego/eino-examples/tree/main/flow/agent/multiagent/plan_execute) | Plan-Execute | 计划执行模式的 Multi-Agent 示例 |

### 完整应用示例
| 目录 | 名称 | 说明 |
|------|------|------|
| [flow/agent/manus](https://github.com/cloudwego/eino-examples/tree/main/flow/agent/manus) | Manus Agent | 基于 Eino 实现的 Manus Agent，参考 OpenManus 项目 |
| [flow/agent/deer-go](https://github.com/cloudwego/eino-examples/tree/main/flow/agent/deer-go) | Deer-Go | 参考 deer-flow 的 Go 语言实现，支持研究团队协作的状态图流转 |

---

## 🧩 Components (组件)

### Model (模型)
| 目录 | 名称 | 说明 |
|------|------|------|
| [components/model/abtest](https://github.com/cloudwego/eino-examples/tree/main/components/model/abtest) | A/B 测试路由 | 动态路由 ChatModel，支持 A/B 测试和模型切换 |
| [components/model/httptransport](https://github.com/cloudwego/eino-examples/tree/main/components/model/httptransport) | HTTP 传输日志 | cURL 风格的 HTTP 请求日志记录，支持流式响应和敏感信息脱敏 |

### Retriever (检索器)
| 目录 | 名称 | 说明 |
|------|------|------|
| [components/retriever/multiquery](https://github.com/cloudwego/eino-examples/tree/main/components/retriever/multiquery) | 多查询检索 | 使用 LLM 生成多个查询变体，提高检索召回率 |
| [components/retriever/router](https://github.com/cloudwego/eino-examples/tree/main/components/retriever/router) | 路由检索 | 根据查询内容动态路由到不同的检索器 |

### Tool (工具)
| 目录 | 名称 | 说明 |
|------|------|------|
| [components/tool/jsonschema](https://github.com/cloudwego/eino-examples/tree/main/components/tool/jsonschema) | JSON Schema 工具 | 展示如何使用 JSON Schema 定义工具参数 |
| [components/tool/mcptool/callresulthandler](https://github.com/cloudwego/eino-examples/tree/main/components/tool/mcptool/callresulthandler) | MCP 工具结果处理 | 展示 MCP 工具调用结果的自定义处理 |
| [components/tool/middlewares/errorremover](https://github.com/cloudwego/eino-examples/tree/main/components/tool/middlewares/errorremover) | 错误移除中间件 | 工具调用错误处理中间件，将错误转换为友好提示 |
| [components/tool/middlewares/jsonfix](https://github.com/cloudwego/eino-examples/tree/main/components/tool/middlewares/jsonfix) | JSON 修复中间件 | 修复 LLM 生成的格式错误 JSON 参数 |

### Document (文档)
| 目录 | 名称 | 说明 |
|------|------|------|
| [components/document/parser/customparser](https://github.com/cloudwego/eino-examples/tree/main/components/document/parser/customparser) | 自定义解析器 | 展示如何实现自定义文档解析器 |
| [components/document/parser/extparser](https://github.com/cloudwego/eino-examples/tree/main/components/document/parser/extparser) | 扩展解析器 | 使用扩展解析器处理 HTML 等格式 |
| [components/document/parser/textparser](https://github.com/cloudwego/eino-examples/tree/main/components/document/parser/textparser) | 文本解析器 | 基础文本文档解析器示例 |

### Prompt (提示词)
| 目录 | 名称 | 说明 |
|------|------|------|
| [components/prompt/chat_prompt](https://github.com/cloudwego/eino-examples/tree/main/components/prompt/chat_prompt) | Chat Prompt | 展示如何使用 Chat Prompt 模板 |

### Lambda
| 目录 | 名称 | 说明 |
|------|------|------|
| [components/lambda](https://github.com/cloudwego/eino-examples/tree/main/components/lambda) | Lambda 组件 | Lambda 函数组件的使用示例 |

---

## 🚀 QuickStart (快速开始)

| 目录 | 名称 | 说明 |
|------|------|------|
| [quickstart/chat](https://github.com/cloudwego/eino-examples/tree/main/quickstart/chat) | Chat 快速开始 | 最基础的 LLM 对话示例，包含模板、生成、流式输出 |
| [quickstart/eino_assistant](https://github.com/cloudwego/eino-examples/tree/main/quickstart/eino_assistant) | Eino 助手 | 完整的 RAG 应用示例，包含知识索引、Agent 服务、Web 界面 |
| [quickstart/todoagent](https://github.com/cloudwego/eino-examples/tree/main/quickstart/todoagent) | Todo Agent | 简单的 Todo 管理 Agent 示例 |

---

## 🛠️ DevOps (开发运维)

| 目录 | 名称 | 说明 |
|------|------|------|
| [devops/debug](https://github.com/cloudwego/eino-examples/tree/main/devops/debug) | 调试工具 | 展示如何使用 Eino 的调试功能，支持 Chain 和 Graph 调试 |
| [devops/visualize](https://github.com/cloudwego/eino-examples/tree/main/devops/visualize) | 可视化工具 | 将 Graph/Chain/Workflow 渲染为 Mermaid 图表 |

---

## 📚 相关资源

- **Eino 框架**: https://github.com/cloudwego/eino
- **Eino 扩展组件**: https://github.com/cloudwego/eino-ext
- **官方文档**: https://www.cloudwego.io/zh/docs/eino/
