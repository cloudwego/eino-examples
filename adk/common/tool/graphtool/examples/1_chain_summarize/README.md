# Example 1: Document Summarization with compose.Chain

This example demonstrates using `InvokableGraphTool` with `compose.Chain` to create a document summarization tool.

## What This Example Shows

- Using `compose.Chain` for sequential processing
- Wrapping a chain as an `InvokableTool`
- Multi-step LLM processing (extract key points → generate summary)
- Integrating with `ChatModelAgent`

## Architecture

```
Input Document
      │
      ▼
┌─────────────────┐
│ Extract Key     │  ← ChatModel extracts key points
│ Points          │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Generate        │  ← ChatModel creates coherent summary
│ Summary         │
└────────┬────────┘
         │
         ▼
   Output Summary
```

## Key Components

### Input/Output Types

```go
type SummarizeInput struct {
    Document string `json:"document"`
    MaxWords int    `json:"max_words"`
}

type SummarizeOutput struct {
    Summary   string   `json:"summary"`
    KeyPoints []string `json:"key_points"`
    WordCount int      `json:"word_count"`
}
```

### Chain Construction

```go
fullChain := compose.NewChain[*SummarizeInput, *SummarizeOutput]()
fullChain.
    AppendLambda(/* transform input */).
    AppendChatTemplate(extractKeyPointsPrompt).
    AppendChatModel(cm).
    AppendLambda(/* transform for next step */).
    AppendChatTemplate(condenseSummaryPrompt).
    AppendChatModel(cm).
    AppendLambda(/* format output */)
```

### Tool Creation

```go
tool, err := graphtool.NewInvokableGraphTool[*SummarizeInput, *SummarizeOutput](
    fullChain,
    "summarize_document",
    "Summarize a document by extracting key points and creating a coherent summary.",
)
```

## Running the Example

```bash
# Set your OpenAI API key
export OPENAI_API_KEY=your-api-key

# Run the example
go run main.go
```

## Expected Output

```
=== Document Summarization Example ===

[Agent calls summarize_document tool]
[Chain executes: extract key points → generate summary]
[Agent returns formatted summary to user]
```

## Key Takeaways

1. **Chain for Sequential Processing**: `compose.Chain` is ideal for linear pipelines where each step's output feeds the next
2. **Type Safety**: Generic types `[I, O]` ensure compile-time type checking
3. **Prompt Templates**: Use `prompt.FromMessages` with `schema.FString` for dynamic prompts
4. **Tool Integration**: The wrapped chain appears as a single tool to the agent
