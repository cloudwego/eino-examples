# Eino Examples

English | [中文](README.zh_CN.md)

## Overview

This repository contains examples and demonstrations for using the Eino framework. It provides practical examples to help developers better understand and utilize Eino's features.

## Repository Structure

### 📦 ADK (Agent Development Kit)

| Directory | Name | Description |
|-----------|------|-------------|
| [adk/helloworld](./adk/helloworld) | Hello World Agent | The simplest Agent example, showing how to create a basic conversational Agent |
| [adk/intro/chatmodel](./adk/intro/chatmodel) | ChatModel Agent | Demonstrates using ChatModelAgent with Interrupt mechanism |
| [adk/intro/custom](./adk/intro/custom) | Custom Agent | Shows how to implement a custom Agent conforming to ADK definition |
| [adk/intro/workflow](./adk/intro/workflow) | Workflow Agents | Loop, Parallel, and Sequential Agent patterns |
| [adk/intro/session](./adk/intro/session) | Session Management | Passing data and state across Agents using Session |
| [adk/intro/transfer](./adk/intro/transfer) | Agent Transfer | ChatModelAgent's Transfer capability for task handoff between Agents |
| [adk/intro/http-sse-service](./adk/intro/http-sse-service) | HTTP SSE Service | Exposing ADK Runner as an HTTP service with Server-Sent Events |
| [adk/human-in-the-loop](./adk/human-in-the-loop) | Human-in-the-Loop | 8 examples: Approval, Review-Edit, Feedback Loop, Follow-up, Supervisor patterns |
| [adk/multiagent](./adk/multiagent) | Multi-Agent | Supervisor, Plan-Execute-Replan, Deep Agents, Excel Agent examples |
| [adk/common/tool/graphtool](./adk/common/tool/graphtool) | GraphTool | Wrapping Graph/Chain/Workflow as Agent tools |

### 🔗 Compose (Orchestration)

| Directory | Name | Description |
|-----------|------|-------------|
| [compose/chain](./compose/chain) | Chain | Sequential orchestration with compose.Chain, including Prompt + ChatModel |
| [compose/graph](./compose/graph) | Graph | Graph orchestration examples: state graph, tool call agent, async nodes, interrupt |
| [compose/workflow](./compose/workflow) | Workflow | Workflow examples: field mapping, data-only, control-only, static values, streaming |
| [compose/batch](./compose/batch) | BatchNode | Batch processing component with concurrency control and interrupt/resume support |

### 🌊 Flow

| Directory | Name | Description |
|-----------|------|-------------|
| [flow/agent/react](./flow/agent/react) | ReAct Agent | ReAct Agent with memory, dynamic options, unknown tool handler |
| [flow/agent/multiagent](./flow/agent/multiagent) | Multi-Agent | Host multi-agent (Journal Assistant), Plan-Execute patterns |
| [flow/agent/manus](./flow/agent/manus) | Manus Agent | Manus Agent implementation inspired by OpenManus |
| [flow/agent/deer-go](./flow/agent/deer-go) | Deer-Go | Go implementation based on deer-flow, supporting research team collaboration |

### 🧩 Components

| Directory | Name | Description |
|-----------|------|-------------|
| [components/model](./components/model) | Model | A/B test routing, HTTP transport logging with cURL-style output |
| [components/retriever](./components/retriever) | Retriever | Multi-query retriever, router retriever |
| [components/tool](./components/tool) | Tool | JSON Schema tools, MCP tools, middlewares (error remover, JSON fix) |
| [components/document](./components/document) | Document | Custom parser, extension parser, text parser |
| [components/prompt](./components/prompt) | Prompt | Chat prompt template examples |
| [components/lambda](./components/lambda) | Lambda | Lambda function component examples |

### 🚀 QuickStart

| Directory | Name | Description |
|-----------|------|-------------|
| [quickstart/chat](./quickstart/chat) | Chat QuickStart | Basic LLM chat example with template, generate, and streaming |
| [quickstart/eino_assistant](./quickstart/eino_assistant) | Eino Assistant | Complete RAG application with knowledge indexing, Agent service, and Web UI |
| [quickstart/todoagent](./quickstart/todoagent) | Todo Agent | Simple Todo management Agent example |

### 🛠️ DevOps

| Directory | Name | Description |
|-----------|------|-------------|
| [devops/debug](./devops/debug) | Debug Tools | Eino debugging features for Chain and Graph |
| [devops/visualize](./devops/visualize) | Visualization | Rendering Graph/Chain/Workflow as Mermaid diagrams |

## Documentation

For detailed documentation of each example, see [COOKBOOK.md](./COOKBOOK.md).

## Related Resources

- **Eino Framework**: https://github.com/cloudwego/eino
- **Eino Extensions**: https://github.com/cloudwego/eino-ext
- **Official Documentation**: https://www.cloudwego.io/docs/eino/

## Security

If you discover a potential security issue in this project, or think you may
have discovered a security issue, we ask that you notify Bytedance Security via
our [security center](https://security.bytedance.com/src) or [vulnerability reporting email](sec@bytedance.com).

Please do **not** create a public GitHub issue.

## License

This project is licensed under the [Apache-2.0 License](LICENSE-APACHE).
