---
title: "ChatWithEino Quickstart 总览（Console → Web/A2UI）"
---

本篇文档用于作为 ChatWithEino Quickstart 的统一入口：用一条清晰的路径带你跑起来，并解释这个系列最终要交付什么（一个可扩展的端到端 Agent 应用骨架）。

## 这是什么

ChatWithEino 是一个基于 Eino 构建的学习型 Agent：它能读取源码/文档/示例，并通过对话帮助开发者理解 Eino 以及用 Eino 写代码。

这个 Quickstart 系列采用“渐进式搭建”的方式：

- 前期以 Console 为载体，逐步引入 ChatModel、Agent/Runner、Memory、Tool、Middleware、Callback、Interrupt/Resume、Graph Tool、Skill
- 最终把同一个 Agent 以 Web 形态交付出来，并用 A2UI 协议把事件流渲染成可增量更新的 UI

## 最短路径：先跑起来

在仓库根目录执行：

```bash
git clone https://github.com/cloudwego/eino-examples.git
cd eino-examples/quickstart/chatwitheino
```

### 1) 最小 Console（第一章）

准备模型配置（以 OpenAI 为例）：

```bash
export OPENAI_API_KEY="..."
export OPENAI_MODEL="gpt-4.1-mini"
```

运行：

```bash
go run ./cmd/ch01 -- "用一句话解释 Eino 的 Component 设计解决了什么问题？"
```

### 2) 最终 Web（A2UI）

```bash
go run .
```

启动后访问输出里的地址（默认 `http://localhost:8080`）。

### 3) （可选）开启 skills（第九章能力复用）

skills 用于把一组稳定的“知识/指令包”（`SKILL.md` + `reference/*.md`）注入到 Agent，让模型在需要时按需加载并调用。

```bash
go run ./scripts/sync_eino_ext_skills.go -src /path/to/eino-ext -dest ./skills/eino-ext -clean
EINO_EXT_SKILLS_DIR="$(pwd)/skills/eino-ext" go run .
```

说明：

- `./skills/` 目录默认被 `.gitignore` 忽略，避免把同步出来的 skills 误提交
- 如需验证 Skill 是否生效，可运行 [第九章示例](https://github.com/cloudwego/eino-examples/blob/main/quickstart/chatwitheino/cmd/ch09/main.go)

## 学习路线（章节导航）

| 章节 | 主题 | 入口 |
|------|------|------|
| 第一章 | ChatModel 与 Message（Console） | https://github.com/cloudwego/eino-examples/blob/main/quickstart/chatwitheino/docs/ch01_chatmodel_agent_console.md |
| 第二章 | Agent 与 Runner（Console 多轮） | https://github.com/cloudwego/eino-examples/blob/main/quickstart/chatwitheino/docs/ch02_chatmodel_agent_runner_console.md |
| 第三章 | Memory 与 Session（持久化对话） | https://github.com/cloudwego/eino-examples/blob/main/quickstart/chatwitheino/docs/ch03_memory_session_jsonl.md |
| 第四章 | Tool 与文件系统访问 | https://github.com/cloudwego/eino-examples/blob/main/quickstart/chatwitheino/docs/ch04_tool_backend_filesystem.md |
| 第五章 | Middleware（中间件模式） | https://github.com/cloudwego/eino-examples/blob/main/quickstart/chatwitheino/docs/ch05_middleware.md |
| 第六章 | Callback 与 Trace（可观测性） | https://github.com/cloudwego/eino-examples/blob/main/quickstart/chatwitheino/docs/ch06_callback.md |
| 第七章 | Interrupt/Resume（中断与恢复） | https://github.com/cloudwego/eino-examples/blob/main/quickstart/chatwitheino/docs/ch07_interrupt_resume.md |
| 第八章 | Graph Tool（复杂工作流） | https://github.com/cloudwego/eino-examples/blob/main/quickstart/chatwitheino/docs/ch08_graph_tool.md |
| 第九章 | Skill（Console） | https://github.com/cloudwego/eino-examples/blob/main/quickstart/chatwitheino/docs/ch09_skill.md |
| 最终章 | A2UI（Web） | https://github.com/cloudwego/eino-examples/blob/main/quickstart/chatwitheino/docs/ch10_a2ui.md |

## 最终交付：一个可扩展的端到端 Agent 应用骨架

你可以把这个 Quickstart 的最终产物理解为一套“可插拔的应用骨架”，它把 Eino 的关键能力连成闭环：

- 运行时：Runner 驱动执行，支持流式输出与事件模型
- 工具层：通过 Tool 接入文件系统/检索/工作流等能力
- 中间件：用 handler/middleware 承载重试、审批、错误处理等横切能力
- 人机协作：interrupt/resume + checkpoint 支持审批、补参、分支选择等交互式流程
- 确定性编排：compose（graph/chain/workflow）把复杂业务流程组织为可维护、可复用的执行图
- UI 交付：用 A2UI 把 Agent 的事件流映射为可增量渲染的 UI 组件树（SSE 推送）

其中 A2UI 的边界需要明确：它不是 Eino 框架本身的一部分，而是业务层的 UI 协议/渲染方案。本 Quickstart 用它来展示“Agent 能力如何以产品形态呈现给用户”，具体实现与协议细节以最终章为准。

## 下一步探索（从 Quickstart 到真实业务）

- 想系统理解 Eino 的组件抽象与用法：从第一章的 Component 入门开始，再按章节逐步补齐 Tool/Graph/Callback/Interrupt 等能力
- 想复用更大规模的知识与指令：对接 `eino-ext` 的 skills，并通过 Skill 中间件按需加载
- 想把 Agent 做成业务产品：参考最终章（A2UI/Web）把事件流、状态与交互打通，再替换为你自己的 UI 形态与协议
