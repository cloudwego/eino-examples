# Example 3: Order Processing with compose.Workflow + Approval

This example demonstrates using `InvokableGraphTool` with `compose.Workflow` to create an order processing tool, wrapped with `InvokableApprovableTool` for human-in-the-loop approval.

## What This Example Shows

- Using `compose.Workflow` with field mapping for parallel branches
- Wrapping a workflow as an `InvokableTool`
- Human-in-the-loop approval via `InvokableApprovableTool`
- Interrupt/Resume flow with checkpoint persistence
- Parallel node execution with result aggregation

## Architecture

```
Order Input
      │
      ▼
┌─────────────────┐
│    validate     │  ← Validate order details
└────────┬────────┘
         │
    ┌────┴────┐
    │         │
    ▼         ▼
┌────────┐ ┌────────────┐
│calculate│ │lookup      │  ← Parallel execution
│_price   │ │_customer   │
└────┬───┘ └─────┬──────┘
     │           │
     └─────┬─────┘
           │
           ▼  (field mapping)
┌─────────────────────┐
│generate_confirmation│  ← Aggregate results
└─────────┬───────────┘
          │
          ▼
    Order Output
```

## Key Components

### Workflow with Parallel Branches

```go
workflow := compose.NewWorkflow[*OrderInput, *OrderOutput]()

workflow.AddLambdaNode("validate", ...).AddInput(compose.START)

// Parallel branches from validate
workflow.AddLambdaNode("calculate_price", ...).AddInput("validate")
workflow.AddLambdaNode("lookup_customer", ...).AddInput("validate")

// Merge with field mapping
workflow.AddLambdaNode("generate_confirmation", ...).
    AddInput("calculate_price", compose.ToField("Pricing")).
    AddInput("lookup_customer", compose.ToField("Customer"))

workflow.End().AddInput("generate_confirmation")
```

### Field Mapping for Aggregation

```go
type orderContext struct {
    Pricing  *pricingResult
    Customer *customerInfo
}

// The node receives aggregated input:
func(ctx context.Context, input *orderContext) (*OrderOutput, error) {
    // input.Pricing comes from calculate_price
    // input.Customer comes from lookup_customer
}
```

### Approval Wrapper

```go
innerTool, _ := graphtool.NewInvokableGraphTool[*OrderInput, *OrderOutput](
    workflow,
    "process_order",
    "Process a customer order...",
)

// Wrap with approval requirement
orderTool := tool2.InvokableApprovableTool{InvokableTool: innerTool}
```

### Interrupt/Resume Handling

```go
// Initial query triggers interrupt
iter := runner.Query(ctx, query, adk.WithCheckPointID(checkpointID))

// ... process events until interrupt ...

// Resume with approval
iter, _ = runner.ResumeWithParams(ctx, checkpointID, &adk.ResumeParams{
    Targets: map[string]any{
        interruptID: &tool2.ApprovalResult{Approved: true},
    },
})
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
=== Order Processing with Interrupt/Resume Example ===

This example demonstrates using InvokableGraphTool (with compose.Workflow)
wrapped with InvokableApprovableTool for human-in-the-loop approval.

User Query: Place an order for customer C001, product P100 (Laptop Pro), quantity 3

[Agent calls process_order tool]
[Tool interrupts for approval]

--- Order requires approval ---

Your decision (Y/N): Y

--- Resuming order processing ---

[Workflow executes: validate → calculate_price + lookup_customer → generate_confirmation]
[Agent returns order confirmation]

=== Order Processing Complete ===
```

## Key Takeaways

1. **Workflow for Parallel Branches**: `compose.Workflow` supports DAG-style execution with `AddInput` connections
2. **Field Mapping**: Use `compose.ToField("FieldName")` to aggregate multiple node outputs into a struct
3. **Approval Wrapper**: `InvokableApprovableTool` adds human approval without modifying the underlying tool
4. **Checkpoint Persistence**: Use `CheckPointStore` and `CheckPointID` for durable interrupt/resume
