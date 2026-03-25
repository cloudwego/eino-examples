# Eino ADK Multiagent Example: OpenClaw-like Agent

## Description
`openclaw-like-agent` is a workspace-aware local agent example built on Eino ADK.

This example is a reproduction of the recent OpenClaw-style local agent experience using Eino ADK. It shows how to build an OpenClaw-inspired agent pattern with Eino primitives, while keeping the project structure consistent with the other examples in this repository.

Compared with the other `adk/multiagent` demos, this example is closer to a practical local assistant:

- It starts from a standard `main.go` entry and can be launched directly with `go run`.
- It keeps multi-turn conversation history by `session`.
- It creates and manages a dedicated `workspace`.
- It supports loading skills from multiple directories.
- It exposes filesystem and shell capabilities through a restricted backend.

The current implementation is organized as a CLI application, with the main logic living under the `myagent` package.

## Env
Before running it, configure the model-related environment variables:

```bash
# Required
export OPENAI_API_KEY=""
export OPENAI_MODEL=""

# Optional
export OPENAI_BASE_URL=""
export OPENAI_BY_AZURE="false"
```

## Quick Start
Start interactive chat mode:

```bash
go run ./adk/multiagent/openclaw-like-agent
```

Run one single-turn query:

```bash
go run ./adk/multiagent/openclaw-like-agent -- run -q "帮我总结当前 workspace 里有哪些文件"
```

Use a fixed workspace and session:

```bash
go run ./adk/multiagent/openclaw-like-agent -- \
  --workspace /tmp/myagent-demo \
  --session-id demo \
  chat
```

Example interactive output with sensitive information masked:

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

## CLI Commands
The root command defaults to interactive chat mode, and also provides several subcommands:

- `chat`: interactive chat mode.
- `run [-q query] [query]`: execute one turn only.
- `session list`: list all sessions in the current workspace.
- `session clear --session-id <id>`: clear conversation history and summary for one session.
- `session delete --session-id <id>`: delete the persisted files for one session.
- `skill list`: list all discovered skills.

Common flags:

- `--workspace`, `-w`: workspace directory. Default is `./myagent_workspace`.
- `--session-id`: session id. Auto-generated when omitted.
- `--instruction`: override the default system instruction.
- `--max-iterations`: max loop iterations for the agent.

## Interactive Commands
In `chat` mode, the following slash commands are supported:

- `/help`
- `/skills`
- `/session`
- `/history`
- `/clear`
- `/exit`

## Workspace Layout
When the agent starts for the first time, it will initialize the workspace automatically:

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

Key files and directories:

- `IDENTITY.md`: workspace-level identity override.
- `memory/MEMORY.md`: long-term memory.
- `sessions/`: persisted session metadata and message history.
- `skills/`: workspace-local skills.
- `artifacts/`: files generated during execution.
- `.claude/skills/`: optional skill directory under the workspace. It is supported by the loader, but not created automatically.

## Skill Loading Order
The agent currently loads skills in the following priority order:

1. `<workspace>/skills`
2. `<workspace>/.claude/skills`
3. `~/.claude/skills`
4. `$XDG_CONFIG_HOME/myagent/skills` or `~/.config/myagent/skills`
5. builtin skills directory, defaulting to `./skills`

You can inspect discovered skills with:

```bash
go run ./adk/multiagent/openclaw-like-agent -- skill list
```

## Notes
- The backend is workspace-restricted by default and protects key files such as `IDENTITY.md` and `MEMORY.md`.
- Session history is stored as JSONL under `sessions/`.
- This example has already been wired into the repository as a standard sub-application entry and can be built directly with:

```bash
go build ./adk/multiagent/openclaw-like-agent
```
