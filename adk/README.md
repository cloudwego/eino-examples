# Eino ADK Examples

This directory provides examples for Eino ADK:

- Agent
  - `helloworld`: simple hello-world chat agent.
  - `agent/ralph-loop`: autonomous iteration loop with filesystem-backed work state and verification gates.
  - `intro`
    - `chatmodel`: example about using `ChatModelAgent` with interrupt.
    - `custom`: shows how to implement an agent which meets the definition of ADK.
    - `workflow`: examples about using `Loop` / `Parallel` / `Sequential` agent.
    - `session`: shows how to pass data and state across agents by using session.
    - `transfer`: shows transfer ability by using ChatModelAgent.
    - `agent_with_summarization`: shows how to add conversation summarization to long-running agents.
    - `http-sse-service`: exposes ADK Runner as an HTTP SSE service.
  - `human-in-the-loop`
    - `1_approval`: sensitive-operation approval.
    - `2_review-and-edit`: review and edit tool-call arguments.
    - `3_feedback-loop`: iterative writer/reviewer feedback loop.
    - `4_follow-up`: ask follow-up questions when required information is missing.
    - `5_supervisor`: supervisor pattern with approval.
    - `6_plan-execute-replan`: plan-execute-replan with review/edit.
    - `7_deep-agents`: Deep Agents with follow-up.
    - `8_supervisor-plan-execute`: nested supervisor and plan-execute agents with interrupt.
  - `multiagent`
    - `plan-execute-replan`: basic example of plan-execute-replan agent.
    - `supervisor`: basic example of supervisor agent.
    - `layered-supervisor`: another example of supervisor agent, which set a supervisor agent as sub-agent of another supervisor agent.
    - `integration-project-manager`: another example of using supervisor agent.
    - `deep`: Deep Agents example.
    - `integration-excel-agent`: ADK-integrated Excel Agent.
  - `agentic`
    - `research_assistant`: example of using `AgenticModel` with ADK typed agent, server-side search, local tools, native filesystem middleware, streaming events, and `AgenticMessage` output.
    - `retry_max_output_tokens`: retries when model output is truncated by `max_output_tokens`.
  - `cancel`
    - `graceful-exit`: cancel and resume nested agents safely from terminal signals.
  - `middlewares`
    - `skill`: loads Agent skills from filesystem.
    - `dynamictool/toolsearch`: dynamically retrieves and injects relevant tools from a large tool set.
  - `common`
    - `tool/graphtool`: wraps Graph/Chain/Workflow as Agent tools.
    - `model`, `prints`, `store`, `trace`: shared helpers used by examples.


Additionally, you can enable [coze-loop](https://github.com/coze-dev/coze-loop) trace for examples, see .example.env for keys. 
