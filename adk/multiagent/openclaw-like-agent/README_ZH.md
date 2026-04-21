# Eino ADK 多智能体示例：OpenClaw-like Agent

## 简介
`openclaw-like-agent` 是一个基于 Eino ADK 构建的、本地运行的 workspace 感知型 Agent 示例。

这个示例可以理解为基于 Eino ADK 对近期很火的 OpenClaw 风格本地 Agent 体验做的一次复刻。它展示了如何用 Eino 提供的 Agent、工具、session 和 workspace 能力，搭建出一个 OpenClaw 风格的本地 Agent 形态，同时保持与本仓库其他示例一致的项目结构。

和其他 `adk/multiagent` 示例相比，这个例子更接近一个可实际使用的本地助手，也更贴近近期 OpenClaw 风格本地 Agent 的使用方式：

- 通过标准 `main.go` 入口启动，可直接使用 `go run` 运行。
- 支持基于 `session` 的多轮对话历史持久化。
- 自动创建并管理独立的 `workspace`。
- 支持从多个目录加载 skills。
- 通过受限 backend 暴露文件系统与 shell 能力。

当前实现是一个 CLI 应用，核心逻辑位于 `myagent` 包中。

## 环境变量
运行前请先配置模型相关环境变量：

```bash
# 必填
export OPENAI_API_KEY=""
export OPENAI_MODEL=""

# 可选
export OPENAI_BASE_URL=""
export OPENAI_BY_AZURE="false"
```

## 快速开始
启动交互式聊天模式：

```bash
go run ./adk/multiagent/openclaw-like-agent
```

执行单轮请求：

```bash
go run ./adk/multiagent/openclaw-like-agent -- run -q "帮我总结当前 workspace 里有哪些文件"
```

指定固定的 workspace 和 session：

```bash
go run ./adk/multiagent/openclaw-like-agent -- \
  --workspace /tmp/myagent-demo \
  --session-id demo \
  chat
```

脱敏后的交互示例：

```text
$ go run main.go
MyAgent TUI
workspace: /path/to/eino-examples/adk/multiagent/openclaw-like-agent/myagent_workspace
session:   sess_xxxxxxxxxxxxxxxx
commands:  /help /session /history /clear /exit

myagent> 你好

[00:15:09] user> 你好
[run.started] session=sess_xxxxxxxxxxxxxxxx workspace=/path/to/eino-examples/adk/multiagent/openclaw-like-agent/myagent_workspace
[assistant] 你好！很高兴见到你。有什么我可以帮你的吗？

[run.completed] session=sess_xxxxxxxxxxxxxxxx saved_messages=2 at=00:15:12

myagent>
```

## CLI 命令
根命令默认进入交互式聊天模式，同时也提供以下子命令：

- `chat`：进入交互式聊天模式。
- `run [-q query] [query]`：执行单轮请求。
- `session list`：列出当前 workspace 下的全部 session。
- `session clear --session-id <id>`：清空某个 session 的历史和摘要。
- `session delete --session-id <id>`：删除某个 session 的持久化文件。
- `skill list`：列出当前可发现的所有 skills。

常用参数：

- `--workspace`, `-w`：指定 workspace 目录，默认是 `./myagent_workspace`。
- `--session-id`：指定 session id；不传时自动生成。
- `--instruction`：覆盖默认 system instruction。
- `--max-iterations`：Agent loop 的最大迭代次数。

## 交互模式命令
在 `chat` 模式下，支持以下 slash commands：

- `/help`
- `/skills`
- `/session`
- `/history`
- `/clear`
- `/exit`

## Workspace 目录结构
首次启动时，Agent 会自动初始化 workspace：

```text
myagent_workspace/
├── IDENTITY.md
├── artifacts/
├── logs/
├── memory/
│   └── MEMORY.md
├── sessions/
└── skills/
```

主要目录和文件说明：

- `IDENTITY.md`：workspace 级别的 identity 覆盖文件。
- `memory/MEMORY.md`：长期记忆文件。
- `sessions/`：session 元数据与消息历史持久化目录。
- `skills/`：workspace 本地 skills 目录。
- `artifacts/`：执行过程中产生的产物文件目录。
- `.claude/skills/`：workspace 下可选的 skill 目录，loader 支持读取，但不会自动创建。

## Skill 加载顺序
当前 Agent 按如下优先级加载 skills：

1. `<workspace>/skills`
2. `<workspace>/.claude/skills`
3. `~/.claude/skills`
4. `$XDG_CONFIG_HOME/myagent/skills` 或 `~/.config/myagent/skills`
5. 内置 skills 目录，默认是 `./skills`

你可以用下面的命令查看最终发现的 skills：

```bash
go run ./adk/multiagent/openclaw-like-agent -- skill list
```

## 说明
- backend 默认限制在 workspace 范围内运行，并保护 `IDENTITY.md`、`MEMORY.md` 等关键文件。
- session 历史会以 JSONL 形式持久化到 `sessions/` 目录。
- 这个示例已经作为标准子应用接入仓库，可以直接编译：

```bash
go build ./adk/multiagent/openclaw-like-agent
```
