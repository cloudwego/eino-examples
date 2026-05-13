# Eino ADK Examples

This directory provides examples for Eino ADK:

- Agent
  - `helloworld`: simple hello-world chat agent.
  - `intro`
    - `chatmodel`: example about using `ChatModelAgent` with interrupt.
    - `custom`: shows how to implement an agent which meets the definition of ADK.
    - `workflow`: examples about using `Loop` / `Parallel` / `Sequential` agent.
    - `session`: shows how to pass data and state across agents by using session.
    - `transfer`: shows transfer ability by using ChatModelAgent.
  - `multiagent`
    - `plan-execute-replan`: basic example of plan-execute-replan agent.
    - `supervisor`: basic example of supervisor agent.
    - `layered-supervisor`: another example of supervisor agent, which set a supervisor agent as sub-agent of another supervisor agent.
    - `integration-project-manager`: another example of using supervisor agent.
  - `agentic`
    - `travel_planner`: example of using `agenticark.AgenticModel` with ADK agent loop, server-side web search, local tools, filesystem middleware, and custom policy middleware.
  - `common`: utils. 


Additionally, you can enable [coze-loop](https://github.com/coze-dev/coze-loop) trace for examples, see .example.env for keys. 
