# HTTP SSE Service Example

This example demonstrates how to expose an `adk.Runner` as an HTTP service that returns Server-Sent Events (SSE). It shows how to properly handle different types of `adk.AgentEvent` outputs and convert them to SSE events.

## Overview

The example implements an HTTP endpoint that:
1. Accepts user queries via HTTP GET requests
2. Runs an ADK agent to process the query
3. Streams the agent's response back to the client using Server-Sent Events (SSE)

## Key Features

### Event Type Handling

The implementation handles all types of `adk.AgentEvent` outputs:

1. **Regular Messages** (`adk.Message`)
   - Single, non-streaming messages
   - Sent as a single SSE event with type `"message"`
   - Tool result messages (role = tool) sent with type `"tool_result"`

2. **Streaming Messages** (`adk.MessageStream`)
   - Streaming content from the agent
   - Each chunk is sent as a separate SSE event with type `"stream_chunk"`
   - Tool result chunks sent with type `"tool_result_chunk"`
   - Allows real-time display of agent responses

3. **Tool Calls**
   - Tool invocations by the agent
   - Sent as SSE events with type `"tool_calls"`
   - Includes tool name and arguments

4. **Agent Actions** (`adk.AgentAction`)
   - Transfer actions (routing to another agent)
   - Interrupt actions (human-in-the-loop)
   - Exit actions (agent completion)
   - Sent as SSE events with type `"action"`

5. **Errors**
   - Any errors during agent execution
   - Sent as SSE events with type `"error"`

## SSE Event Format

All SSE events are JSON-formatted with the following structure:

```json
{
  "type": "message|stream_chunk|tool_result|tool_result_chunk|tool_calls|action|error",
  "agent_name": "SSEAgent",
  "run_path": "SSEAgent",
  "content": "The actual content",
  "tool_calls": [...],
  "action_type": "transfer|interrupted|exit",
  "error": "error message if any"
}
```

### Event Types

- **`message`**: A complete, non-streaming message from the agent
- **`stream_chunk`**: A single chunk from a streaming response
- **`tool_result`**: A complete tool result message (role = tool)
- **`tool_result_chunk`**: A single chunk from a streaming tool result
- **`tool_calls`**: Tool invocations by the agent
- **`action`**: Agent actions (transfer, interrupt, exit)
- **`error`**: Error events

## Prerequisites

Make sure you have the required environment variables set:

```bash
# For OpenAI-compatible models
export OPENAI_API_KEY="your-api-key"
export OPENAI_MODEL="gpt-4"
export OPENAI_BASE_URL="https://api.openai.com/v1"

# Or for other providers (e.g., Ark/Volcengine)
export ARK_API_KEY="your-api-key"
export ARK_CHAT_MODEL="your-model"
```

See the `.example.env` file in the repository root for more configuration options.

## Running the Example

1. Navigate to the example directory:
```bash
cd adk/intro/http-sse-service
```

2. Run the server:
```bash
go run main.go
```

The server will start on `http://localhost:8080`.

## Usage Examples

### Using curl

Basic query:
```bash
curl -N 'http://localhost:8080/chat?query=tell me a short story'
```

The `-N` flag disables buffering, allowing you to see SSE events as they arrive.

### Example Response

```
data: {"type":"stream_chunk","agent_name":"SSEAgent","run_path":"SSEAgent","content":"Once"}

data: {"type":"stream_chunk","agent_name":"SSEAgent","run_path":"SSEAgent","content":" upon"}

data: {"type":"stream_chunk","agent_name":"SSEAgent","run_path":"SSEAgent","content":" a"}

data: {"type":"stream_chunk","agent_name":"SSEAgent","run_path":"SSEAgent","content":" time"}

...

data: {"type":"action","agent_name":"SSEAgent","run_path":"SSEAgent","action_type":"exit","content":"Agent execution completed"}
```

### Using JavaScript

```javascript
const eventSource = new EventSource('http://localhost:8080/chat?query=hello');

eventSource.onmessage = (event) => {
  const data = JSON.parse(event.data);
  
  switch(data.type) {
    case 'stream_chunk':
      console.log('Chunk:', data.content);
      break;
    case 'message':
      console.log('Message:', data.content);
      break;
    case 'tool_result':
      console.log('Tool Result:', data.content);
      break;
    case 'tool_result_chunk':
      console.log('Tool Result Chunk:', data.content);
      break;
    case 'tool_calls':
      console.log('Tool Calls:', data.tool_calls);
      break;
    case 'action':
      console.log('Action:', data.action_type, data.content);
      break;
    case 'error':
      console.error('Error:', data.error);
      break;
  }
};

eventSource.onerror = (error) => {
  console.error('SSE Error:', error);
  eventSource.close();
};
```

### Using Python

```python
import requests
import json

url = 'http://localhost:8080/chat?query=hello'

with requests.get(url, stream=True) as response:
    for line in response.iter_lines():
        if line:
            line = line.decode('utf-8')
            if line.startswith('data: '):
                data = json.loads(line[6:])
                
                if data['type'] == 'stream_chunk':
                    print(data['content'], end='', flush=True)
                elif data['type'] == 'message':
                    print(data['content'])
                elif data['type'] == 'tool_result':
                    print(f"\n[Tool Result] {data['content']}")
                elif data['type'] == 'tool_result_chunk':
                    print(data['content'], end='', flush=True)
                elif data['type'] == 'tool_calls':
                    print(f"\n[Tool Calls] {data['tool_calls']}")
                elif data['type'] == 'action':
                    print(f"\n[{data['action_type']}] {data['content']}")
                elif data['type'] == 'error':
                    print(f"\nError: {data['error']}")
```

## Implementation Details

### Agent Configuration

The example uses a simple `ChatModelAgent` configured with:
- **Name**: "SSEAgent"
- **Description**: "An agent that responds via Server-Sent Events"
- **Instruction**: Basic helpful assistant prompt
- **Model**: Uses the common model helper from `adk/common/model`

### Runner Configuration

The `adk.Runner` is configured with:
- **EnableStreaming**: `true` - Essential for streaming responses
- **Agent**: The configured ChatModelAgent

### Event Processing Flow

1. HTTP request arrives with a `query` parameter
2. `runner.Query()` is called to start agent execution
3. For each `AgentEvent` from the iterator:
   - Check for errors → send error SSE event
   - Check for message output:
     - If `Message` (non-streaming) → send single SSE event
     - If `MessageStream` (streaming) → iterate and send chunk events
   - Check for actions → send action SSE events
4. Connection closes when iterator completes

### Streaming Message Handling

When handling `MessageStream`:
1. Iterate through all chunks using `stream.Recv()`
2. Send each content chunk as a separate SSE event
3. Collect tool call chunks and concatenate them
4. Send concatenated tool calls as separate events

This ensures that:
- Content streams in real-time
- Tool calls are properly assembled from chunks
- The stream is fully consumed

## Architecture

```
┌─────────────┐
│ HTTP Client │
└──────┬──────┘
       │ GET /chat?query=...
       ▼
┌─────────────────┐
│  HTTP Handler   │
└────────┬────────┘
         │ runner.Query()
         ▼
┌─────────────────┐
│   adk.Runner    │
└────────┬────────┘
         │ AgentEvent Iterator
         ▼
┌─────────────────────────┐
│ Event Processing Logic  │
│  - Message              │
│  - MessageStream        │
│  - Action               │
│  - Error                │
└────────┬────────────────┘
         │ SSE Events
         ▼
┌─────────────────┐
│   SSE Stream    │
└────────┬────────┘
         │ data: {...}
         ▼
┌─────────────┐
│ HTTP Client │
└─────────────┘
```

## Extending the Example

### Adding Tool Support

To add tools to the agent:

```go
func createAgent(ctx context.Context) (adk.Agent, error) {
    myTool, err := utils.InferTool(
        "my_tool",
        "description",
        func(ctx context.Context, input MyInput) (string, error) {
            // tool implementation
            return "result", nil
        },
    )
    if err != nil {
        return nil, err
    }

    return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
        Name:        "SSEAgent",
        Description: "An agent that responds via Server-Sent Events",
        Instruction: `You are a helpful assistant with tools.`,
        Model:       model.NewChatModel(),
        ToolsConfig: adk.ToolsConfig{
            ToolsNodeConfig: compose.ToolsNodeConfig{
                Tools: []tool.BaseTool{myTool},
            },
        },
    })
}
```

### Adding Authentication

Add middleware to verify API keys or tokens:

```go
func authMiddleware() app.HandlerFunc {
    return func(ctx context.Context, c *app.RequestContext) {
        apiKey := c.GetHeader("X-API-Key")
        if string(apiKey) != "expected-key" {
            c.JSON(consts.StatusUnauthorized, map[string]string{
                "error": "unauthorized",
            })
            c.Abort()
            return
        }