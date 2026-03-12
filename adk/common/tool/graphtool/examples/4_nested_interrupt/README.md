# Example 4: Nested Interrupts (Outer Approval + Inner Risk Check)

This example demonstrates nested interrupt handling where an `InvokableApprovableTool` wraps an `InvokableGraphTool` that contains its own internal interrupt for risk approval.

## What This Example Shows

- Two-level interrupt/resume flow
- Outer interrupt: Tool-level approval via `InvokableApprovableTool`
- Inner interrupt: Workflow-level risk check via `compose.StatefulInterrupt`
- Proper interrupt state isolation between layers
- Sequential approval handling

## Architecture

```
User Request
      │
      ▼
┌─────────────────────────────────────────────┐
│         InvokableApprovableTool             │
│  ┌───────────────────────────────────────┐  │
│  │      InvokableGraphTool               │  │
│  │  ┌─────────────────────────────────┐  │  │
│  │  │         Workflow                │  │  │
│  │  │                                 │  │  │
│  │  │  validate → risk_check_execute  │  │  │
│  │  │              ↓                  │  │  │
│  │  │     [INNER INTERRUPT]           │  │  │  ← If amount > $1000
│  │  │     (risk approval)             │  │  │
│  │  └─────────────────────────────────┘  │  │
│  └───────────────────────────────────────┘  │
│                    ↓                        │
│           [OUTER INTERRUPT]                 │  ← Always (tool approval)
│           (tool approval)                   │
└─────────────────────────────────────────────┘
```

## Interrupt Flow

```
1. User: "Transfer $1500 from A001 to B002"
         │
         ▼
2. Agent calls transfer_funds tool
         │
         ▼
3. OUTER INTERRUPT (InvokableApprovableTool)
   "tool 'transfer_funds' interrupted... waiting for approval"
         │
         ▼
4. User approves (Y)
         │
         ▼
5. Workflow executes: validate → risk_check_and_execute
         │
         ▼
6. INNER INTERRUPT (amount > $1000)
   "High-value transfer of $1500 requires risk team approval"
         │
         ▼
7. User approves (Y)
         │
         ▼
8. Transfer completes
```

## Key Components

### Inner Interrupt (Risk Check)

```go
workflow.AddLambdaNode("risk_check_and_execute", compose.InvokableLambda(func(ctx context.Context, validation *validationResult) (*TransferOutput, error) {
    // Check if resuming from interrupt
    wasInterrupted, _, storedValidation := compose.GetInterruptState[*validationResult](ctx)
    
    if wasInterrupted {
        isTarget, hasData, data := compose.GetResumeContext[*InternalApprovalResult](ctx)
        if isTarget && hasData {
            if data.Approved {
                // Execute transfer
            }
            // Rejected
        }
        // Re-interrupt if not target
    }
    
    // First run - check if high-value
    if validation.Amount > 1000 {
        return nil, compose.StatefulInterrupt(ctx, &InternalApprovalInfo{
            Step:    "risk_check",
            Message: fmt.Sprintf("High-value transfer of $%.2f requires risk team approval", validation.Amount),
        }, validation)
    }
    
    // Low-value - execute directly
}))
```

### Type Registration for Interrupts

```go
func init() {
    schema.Register[*InternalApprovalInfo]()
    schema.Register[*InternalApprovalResult]()
    schema.Register[*validationResult]()  // For interrupt state
}
```

### Handling Multiple Interrupts

```go
interruptCount := 0
for {
    // ... process events ...
    
    if lastEvent.Action.Interrupted != nil {
        interruptCount++
        
        var resumeData any
        if interruptCount == 1 {
            // First interrupt is outer (tool approval)
            resumeData = &tool2.ApprovalResult{Approved: true}
        } else {
            // Second interrupt is inner (risk approval)
            resumeData = &InternalApprovalResult{Approved: true, Comment: "Risk approved"}
        }
        
        iter, _ = runner.ResumeWithParams(ctx, checkpointID, &adk.ResumeParams{
            Targets: map[string]any{
                interruptID: resumeData,
            },
        })
    }
}
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
=== Nested Interrupt Test ===

This example tests:
1. InvokableApprovableTool wraps InvokableGraphTool
2. The inner workflow has its own interrupt (risk check)
3. Both interrupts should work independently

User Query: Transfer $1500 from account A001 to account B002

[Agent calls transfer_funds tool]

--- Interrupt #1 detected ---
Interrupt ID: xxx
[Tool approval interrupt]

Your decision (Y/N): Y

--- Resuming (interrupt #1) ---

  [Workflow] Validating transfer...
  [Workflow] Performing risk check...
  [Workflow] High-value transfer detected, triggering INTERNAL interrupt...

--- Interrupt #2 detected ---
Interrupt ID: yyy
[Risk approval interrupt]

Your decision (Y/N): Y

--- Resuming (interrupt #2) ---

  [Workflow] Resuming from interrupt...
  [Workflow] Risk team approved with comment: Risk approved by manager
  [Workflow] Executing transfer...

[Agent returns transfer confirmation]

=== Test Complete (Total interrupts: 2) ===
```

## Key Takeaways

1. **Distinct Interrupt State Types**: Outer (`string`) and inner (`*graphToolInterruptState`) use different types, preventing conflicts
2. **Sequential Approval**: Each interrupt must be resolved before the next can occur
3. **State Preservation**: `StatefulInterrupt` preserves data needed for resume
4. **Type Registration**: All interrupt info/result types must be registered with `schema.Register`
5. **Interrupt Identification**: Use `interruptID` from the event to target the correct interrupt when resuming
