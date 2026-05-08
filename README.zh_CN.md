# Eino Examples

[English](README.md) | 中文

## 概述

本仓库包含了 Eino 框架的示例和演示代码，提供了实用的示例来帮助开发者更好地理解和使用 Eino 的功能。

## 仓库结构

### 📦 ADK (Agent Development Kit)

| 目录 | 名称 | 说明 |
|------|------|------|
| [adk/helloworld](./adk/helloworld) | Hello World Agent | 最简单的 Agent 示例，展示如何创建一个基础的对话 Agent |
| [adk/intro/chatmodel](./adk/intro/chatmodel) | ChatModel Agent | 展示如何使用 ChatModelAgent 并配合 Interrupt 机制 |
| [adk/intro/custom](./adk/intro/custom) | 自定义 Agent | 展示如何实现符合 ADK 定义的自定义 Agent |
| [adk/intro/workflow](./adk/intro/workflow) | Workflow Agents | Loop、Parallel、Sequential Agent 模式 |
| [adk/intro/session](./adk/intro/session) | Session 管理 | 展示如何通过 Session 在多个 Agent 之间传递数据和状态 |
| [adk/intro/transfer](./adk/intro/transfer) | Agent 转移 | 展示 ChatModelAgent 的 Transfer 能力，实现 Agent 间的任务转移 |
| [adk/intro/http-sse-service](./adk/intro/http-sse-service) | HTTP SSE 服务 | 展示如何将 ADK Runner 暴露为支持 Server-Sent Events 的 HTTP 服务 |
| [adk/human-in-the-loop](./adk/human-in-the-loop) | 人机协作 | 8 个示例：审批、审核编辑、反馈循环、追问、Supervisor 等模式 |
| [adk/multiagent](./adk/multiagent) | 多 Agent 协作 | Supervisor、Plan-Execute-Replan、Deep Agents、Excel Agent 示例 |
| [adk/common/tool/graphtool](./adk/common/tool/graphtool) | GraphTool | 将 Graph/Chain/Workflow 封装为 Agent 工具 |

### 🔗 Compose (编排)

| 目录 | 名称 | 说明 |
|------|------|------|
| [compose/chain](./compose/chain) | Chain | 使用 compose.Chain 进行顺序编排，包含 Prompt + ChatModel |
| [compose/graph](./compose/graph) | Graph | 图编排示例：状态图、单次工具调用、异步节点、中断机制 |
| [compose/workflow](./compose/workflow) | Workflow | 工作流示例：字段映射、纯数据流、纯控制流、静态值、流式处理 |
| [compose/batch](./compose/batch) | BatchNode | 批量处理组件，支持并发控制和中断恢复 |

### 🌊 Flow (流程模块)

| 目录 | 名称 | 说明 |
|------|------|------|
| [flow/agent/react](./flow/agent/react) | ReAct Agent | ReAct Agent，包含记忆、动态选项、未知工具处理 |
| [flow/agent/multiagent](./flow/agent/multiagent) | Multi-Agent | Host Multi-Agent（日记助手）、Plan-Execute 模式 |
| [flow/agent/manus](./flow/agent/manus) | Manus Agent | 基于 Eino 实现的 Manus Agent，参考 OpenManus 项目 |
| [flow/agent/deer-go](./flow/agent/deer-go) | Deer-Go | 参考 deer-flow 的 Go 语言实现，支持研究团队协作 |

### 🧩 Components (组件)

| 目录 | 名称 | 说明 |
|------|------|------|
| [components/model](./components/model) | Model | A/B 测试路由、cURL 风格的 HTTP 传输日志 |
| [components/retriever](./components/retriever) | Retriever | 多查询检索、路由检索 |
| [components/tool](./components/tool) | Tool | JSON Schema 工具、MCP 工具、中间件（错误移除、JSON 修复） |
| [components/document](./components/document) | Document | 自定义解析器、扩展解析器、文本解析器 |
| [components/prompt](./components/prompt) | Prompt | Chat Prompt 模板示例 |
| [components/lambda](./components/lambda) | Lambda | Lambda 函数组件示例 |

### 🚀 QuickStart (快速开始)

| 目录 | 名称 | 说明 |
|------|------|------|
| [quickstart/chat](./quickstart/chat) | Chat 快速开始 | 最基础的 LLM 对话示例，包含模板、生成、流式输出 |
| [quickstart/eino_assistant](./quickstart/eino_assistant) | Eino 助手 | 完整的 RAG 应用示例，包含知识索引、Agent 服务、Web 界面 |
| [quickstart/todoagent](./quickstart/todoagent) | Todo Agent | 简单的 Todo 管理 Agent 示例 |

### 🛠️ DevOps (开发运维)

| 目录 | 名称 | 说明 |
|------|------|------|
| [devops/debug](./devops/debug) | 调试工具 | 展示如何使用 Eino 的调试功能，支持 Chain 和 Graph 调试 |
| [devops/visualize](./devops/visualize) | 可视化工具 | 将 Graph/Chain/Workflow 渲染为 Mermaid 图表 |

## 详细文档

每个示例的详细说明请参考 [COOKBOOK.md](./COOKBOOK.md)。

## 相关资源

- **Eino 框架**: https://github.com/cloudwego/eino
- **Eino 扩展组件**: https://github.com/cloudwego/eino-ext
- **官方文档**: https://www.cloudwego.io/zh/docs/eino/

## 安全

如果你在该项目中发现潜在的安全问题，或你认为可能发现了安全问题，请通过我们的[安全中心](https://security.bytedance.com/src)或[漏洞报告邮箱](sec@bytedance.com)通知字节跳动安全团队。

请**不要**创建公开的 GitHub Issue。

## 开源许可证

本项目依据 [Apache-2.0 许可证](LICENSE-APACHE) 授权。
