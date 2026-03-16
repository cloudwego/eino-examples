---
title: "第九章：Skill（Console）"
---

本章目标：在第八章（RAG + Interrupt/Resume + Checkpoint）基础上，引入 `skill` 中间件，让 Agent 可以发现并加载一组可复用的技能文档（`SKILL.md`），并在需要时通过工具调用使用它们。

## 代码位置

- 入口代码：[cmd/ch09/main.go](https://github.com/cloudwego/eino-examples/blob/main/quickstart/chatwitheino/cmd/ch09/main.go)
- 同步脚本：[scripts/sync_eino_ext_skills.go](https://github.com/cloudwego/eino-examples/blob/main/quickstart/chatwitheino/scripts/sync_eino_ext_skills.go)

## 前置条件

- 与第一章一致：需要配置一个可用的 ChatModel（OpenAI 或 Ark）
- 准备好 `eino-ext` PR 提供的 skills（`eino-guide` / `eino-component` / `eino-compose` / `eino-agent`）

为什么是这四个？

ChatWithEino 的定位是“帮用户学习 Eino 框架、并尝试用 AI 辅助写 Eino 代码”。这四个 skills 正好覆盖了这个目标所需的关键知识面：

- `eino-guide`：学习入口与导航（从哪里开始、怎么快速跑起来）
- `eino-component`：Component 接口与各类实现参考（Model/Embedding/Retriever/Tool/Callback 等）
- `eino-compose`：编排与确定性工作流参考（Graph/Chain/Workflow 等）
- `eino-agent`：ADK/Agent 相关参考（Agent、Runner、Middleware、Filesystem、Human-in-the-loop 等）

skills 的来源可以是：

- `eino-ext` 仓库本地路径（脚本会自动读取 `<src>/skills/...`）
- 或你已安装 skills 的目录（目录下能看到上述四个子目录）

## 从 Graph Tool 到 Skill：为什么需要“技能文档”

第八章解决的是“复杂工作流如何做成一个可调用的 Tool”（Graph Tool）。但你在构建一个面向框架学习/开发辅助的 Agent 时，还会遇到另一类问题：**如何把一组稳定、可复用的知识与指令注入到 Agent 里，并让它在运行时按需加载？**

这就是 Skill 的定位：

- **Tool** 更像“动作/能力”：读文件、跑 workflow、调用外部系统
- **Skill** 更像“可复用的知识/指令包”：用一组 markdown（`SKILL.md` + `reference/*.md`）描述“如何做某类事”

简单类比：

- **Tool** = “能做什么”（函数/接口）
- **Skill** = “怎么做”（可复用的说明书/操作手册）

## 运行

在 `quickstart/chatwitheino` 目录下执行：

### 1) 同步 eino-ext skills 到本地目录

为了让 `skill` 中间件可以“发现”这些 skills，需要把它们放到一个统一目录下，并满足扫描约定：

- `EINO_EXT_SKILLS_DIR/<skillName>/SKILL.md`

同步命令（推荐）：

```bash
go run ./scripts/sync_eino_ext_skills.go -src /path/to/eino-ext -dest ./skills/eino-ext -clean
```

说明：

- `-src` 支持两种形式：
  - `eino-ext` 仓库根目录（脚本会自动读取 `<src>/skills/...`）
  - 你已安装 skills 的目录（目录下应包含 `eino-guide/`、`eino-component/` 等子目录）
- `-dest` 默认是 `./skills/eino-ext`（可以省略）

### 2) 启动 Chapter 9

```bash
EINO_EXT_SKILLS_DIR=/absolute/path/to/chatwitheino/skills/eino-ext go run ./cmd/ch09
```

输出示例（节选）：

```text
Skills dir: /.../skills/eino-ext
Enter your message (empty line to exit):
```

## 在 DeepAgent 中启用 Skill

本章的 “Skill 可被调用” 不是自动发生的，你需要在 Agent 构建时把 `skill` 中间件注册进去。核心就是三步：

1. 用本地 filesystem backend（本章用 `eino-ext/adk/backend/local`）提供文件读取/Glob 能力
2. 用 `skill.NewBackendFromFilesystem` 把 `EINO_EXT_SKILLS_DIR` 变成一个 Skill Backend
3. 用 `skill.NewMiddleware` 生成中间件，并把它塞进 DeepAgent 的 `Handlers`

**关键代码片段（**注意：这是简化后的代码片段，不能直接运行，完整代码请参考** [cmd/ch09/main.go](https://github.com/cloudwego/eino-examples/blob/main/quickstart/chatwitheino/cmd/ch09/main.go)）：**

```go
backend, _ := localbk.NewBackend(ctx, &localbk.Config{})

skillBackend, _ := skill.NewBackendFromFilesystem(ctx, &skill.BackendFromFilesystemConfig{
    Backend: backend,
    BaseDir: skillsDir, // = $EINO_EXT_SKILLS_DIR
})
skillMiddleware, _ := skill.NewMiddleware(ctx, &skill.Config{
    Backend: skillBackend,
})

agent, _ := deep.New(ctx, &deep.Config{
    ChatModel: cm,
    Backend: backend,
    StreamingShell: backend,
    Handlers: []adk.ChatModelAgentMiddleware{
        skillMiddleware,
        // ... 其他中间件，比如 approval/safeTool/retry 等
    },
})
```

补充说明：

- 本 quickstart 为了保证 “没配置 skills 也能跑”，在代码里对 `EINO_EXT_SKILLS_DIR` 做了存在性检查：目录存在才注册 `skillMiddleware`；否则跳过（此时仍可对话与使用 RAG 工具）。
- Skill 工具的入参是一个 JSON：`{"skill": "<skillName>"}`，例如 `{"skill":"eino-guide"}`。

## 快速验证（推荐）

启动后输入一条指令，明确要求模型调用 skill 工具（用于验证 skills 已被发现且可被加载）：

```text
Use the skill tool with skill="eino-guide" and tell me what the entry point is for getting started.
```

你应当能在控制台看到类似输出：

- `[tool result] Launching skill: eino-guide`
- Tool result 中包含 `Base directory for this skill: .../eino-guide`

## 你会看到什么

- 当模型调用 skill 工具时，控制台会打印：
  - `[tool call] ...`
  - `[tool result] ...`（对结果做了截断展示）
- 会话保存在 `SESSION_DIR`（默认 `./data/sessions`），支持恢复：
  - `go run ./cmd/ch09 --session <id>`
