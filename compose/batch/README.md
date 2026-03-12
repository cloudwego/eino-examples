# BatchNode Example

This example demonstrates how to build a **BatchNode** component that processes multiple inputs through a Graph or Workflow with configurable concurrency and interrupt/resume support.

## Overview

BatchNode is a reusable component that:
- Accepts `[]I` (slice of inputs) and returns `[]O` (slice of outputs)
- Runs a Graph or Workflow for each input item
- Supports configurable concurrency (sequential or parallel)
- Handles errors and interrupts from individual tasks
- Integrates with Eino's callback and checkpoint systems

## Business Scenario

**Document Review Pipeline**: A compliance team needs to review multiple documents. Each document goes through an automated review workflow, with high-priority documents requiring human approval before completion.

## Quick Start

```bash
cd compose/batch
go run .
```

## Project Structure

```
compose/batch/
├── batch/
│   ├── types.go    # Type definitions (NodeConfig, NodeInterruptState, etc.)
│   ├── options.go  # Batch invocation options (WithInnerOptions)
│   ├── store.go    # Internal checkpoint store for sub-tasks
│   └── node.go     # Core BatchNode implementation
├── main.go         # Example scenarios
└── README.md
```

## Key Concepts

### 1. Creating a BatchNode

```go
batchNode := batch.NewBatchNode(&batch.NodeConfig[ReviewRequest, ReviewResult]{
    Name:           "DocumentReviewer",
    InnerTask:      workflow,        // Graph or Workflow
    MaxConcurrency: 3,               // 0=sequential, >0=parallel limit
    InnerCompileOptions: []compose.GraphCompileOption{
        compose.WithGraphName("SingleDocReview"),
    },
})
```

### 2. Concurrency Control

| MaxConcurrency | Behavior |
|----------------|----------|
| `0` | Sequential: process one task at a time |
| `>0` | Concurrent: up to N parallel tasks (first task runs on main goroutine) |

### 3. Options

**Compile-time options** (in `NodeConfig.InnerCompileOptions`):
- Applied when compiling the inner Graph/Workflow
- Example: `compose.WithGraphName("...")`

**Request-time options** (via `batch.WithInnerOptions`):
- Applied to each inner task invocation
- Example: `compose.WithCallbacks(handler)`

```go
results, err := batchNode.Invoke(ctx, inputs,
    batch.WithInnerOptions(
        compose.WithCallbacks(progressHandler),
    ),
)
```

### 4. Error Handling

- **Normal errors**: BatchNode returns the first error encountered
- **Interrupt errors**: Collected and bundled via `compose.CompositeInterrupt`

### 5. Interrupt & Resume

BatchNode supports human-in-the-loop workflows:

```go
// In your inner workflow's lambda:
if needsHumanReview {
    wasInterrupted, _, _ := compose.GetInterruptState[any](ctx)
    if !wasInterrupted {
        // First run: interrupt for human review
        return Result{}, compose.Interrupt(ctx, map[string]string{
            "document_id": docID,
            "reason":      "Requires human approval",
        })
    }
    
    // Resume: get human decision
    isTarget, hasData, decision := compose.GetResumeContext[*Decision](ctx)
    if isTarget && hasData && decision != nil {
        return Result{Approved: decision.Approved}, nil
    }
}
```

Resume with approval decisions:

```go
// Extract interrupt contexts
info, _ := compose.ExtractInterruptInfo(err)

// Prepare resume data (keyed by interrupt ID)
resumeData := make(map[string]any)
for _, iCtx := range info.InterruptContexts {
    resumeData[iCtx.ID] = &Decision{Approved: true}
}

// Resume
resumeCtx := compose.BatchResumeWithData(ctx, resumeData)
results, err = runner.Invoke(resumeCtx, nil, compose.WithCheckPointID(checkpointID))
```

## Scenarios

### Scenario 1: Basic Sequential Processing
Process documents one at a time with `MaxConcurrency: 0`.

### Scenario 2: Concurrent Processing
Process multiple documents in parallel with `MaxConcurrency: 3`.

### Scenario 3: With Compile Options
Configure inner workflow at compile time using `InnerCompileOptions`.

### Scenario 4: With Invoke Options (Callbacks)
Add callbacks for monitoring using `callbacks.InitCallbacks`.

### Scenario 5: Normal Error Handling
Demonstrates how BatchNode handles errors from individual tasks.

### Scenario 6: Interrupt & Resume
Human-in-the-loop workflow:
1. High-priority documents interrupt for human review
2. Extract interrupt contexts with document IDs
3. Resume with approval decisions using `BatchResumeWithData`

### Scenario 7: Parent Graph with Reduce Node
- Integrate BatchNode in a larger pipeline
- Use `WithInnerOptions` for progress tracking callbacks
- Reduce pattern: aggregate batch results into a summary report

## Key APIs Used

| API | Purpose |
|-----|---------|
| `compose.NewWorkflow` | Create inner workflow |
| `compose.AppendAddressSegment` | Create unique address for each sub-task |
| `compose.GetInterruptState` | Check if resuming from interrupt |
| `compose.GetResumeContext` | Get resume data for this component |
| `compose.Interrupt` | Interrupt execution for human input |
| `compose.CompositeInterrupt` | Bundle multiple interrupt errors |
| `compose.ExtractInterruptInfo` | Extract interrupt contexts from error |
| `compose.BatchResumeWithData` | Resume with data for multiple targets |
| `compose.WithCheckPointStore` | Enable checkpoint persistence |
| `compose.WithCheckPointID` | Identify checkpoint for resume |
| `callbacks.EnsureRunInfo` | Setup callback context |
| `callbacks.OnStart/OnEnd/OnError` | Trigger callbacks |
| `schema.RegisterName` | Register types for serialization |

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      BatchNode                               │
│  ┌─────────────────────────────────────────────────────┐    │
│  │  Input: []ReviewRequest                              │    │
│  └─────────────────────────────────────────────────────┘    │
│                           │                                  │
│                           ▼                                  │
│  ┌─────────────────────────────────────────────────────┐    │
│  │  Concurrency Control (MaxConcurrency)               │    │
│  │  - Sequential (0): one at a time                    │    │
│  │  - Concurrent (>0): parallel with semaphore         │    │
│  └─────────────────────────────────────────────────────┘    │
│                           │                                  │
│         ┌─────────────────┼─────────────────┐               │
│         ▼                 ▼                 ▼               │
│  ┌─────────────┐   ┌─────────────┐   ┌─────────────┐       │
│  │ Inner Task  │   │ Inner Task  │   │ Inner Task  │       │
│  │ (index: 0)  │   │ (index: 1)  │   │ (index: 2)  │       │
│  │             │   │             │   │             │       │
│  │ Workflow/   │   │ Workflow/   │   │ Workflow/   │       │
│  │ Graph       │   │ Graph       │   │ Graph       │       │
│  └─────────────┘   └─────────────┘   └─────────────┘       │
│         │                 │                 │               │
│         ▼                 ▼                 ▼               │
│  ┌─────────────────────────────────────────────────────┐    │
│  │  Result Collection                                   │    │
│  │  - Success: store in outputs[index]                 │    │
│  │  - Error: return first error                        │    │
│  │  - Interrupt: collect for CompositeInterrupt        │    │
│  └─────────────────────────────────────────────────────┘    │
│                           │                                  │
│                           ▼                                  │
│  ┌─────────────────────────────────────────────────────┐    │
│  │  Output: []ReviewResult                              │    │
│  └─────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
```

## Interrupt & Resume Flow

```
First Invocation:
┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐
│  DOC-001 │    │  DOC-002 │    │  DOC-003 │    │  DOC-004 │
│  (high)  │    │ (medium) │    │  (high)  │    │  (low)   │
└────┬─────┘    └────┬─────┘    └────┬─────┘    └────┬─────┘
     │               │               │               │
     ▼               ▼               ▼               ▼
 INTERRUPT       COMPLETE        INTERRUPT       COMPLETE
     │               │               │               │
     └───────────────┴───────────────┴───────────────┘
                           │
                           ▼
              ┌────────────────────────┐
              │   CompositeInterrupt   │
              │   - InterruptContexts  │
              │   - NodeInterruptState │
              └────────────────────────┘

Resume with Approval:
┌──────────┐                    ┌──────────┐
│  DOC-001 │                    │  DOC-003 │
│  (high)  │                    │  (high)  │
└────┬─────┘                    └────┬─────┘
     │                               │
     ▼                               ▼
 GetResumeContext               GetResumeContext
 → Decision{Approved: true}     → Decision{Approved: true}
     │                               │
     ▼                               ▼
 COMPLETE                        COMPLETE
     │                               │
     └───────────────┬───────────────┘
                     │
                     ▼
         ┌─────────────────────┐
         │   Final Results     │
         │   DOC-001: ✓        │
         │   DOC-002: ✓        │
         │   DOC-003: ✓        │
         │   DOC-004: ✓        │
         └─────────────────────┘
```

## Sample Output

```
=== Document Review Pipeline Example ===

--- Scenario 6: Interrupt & Resume ---
First invocation (will interrupt for high priority docs):
    Document DOC-001 requires human review (high priority)
    Document DOC-003 requires human review (high priority)

  Interrupt detected! Found 2 interrupt context(s):
    1. ID=fd49cbc4-deca-4f02-bdf9-02f921c0c1f5
       Address=runnable:InterruptResumeDemo;node:batch_review;batch_process:0;...
       DocumentID=DOC-001, Reason=High priority document requires human approval
    2. ID=af4a3f99-2414-4d6c-9c06-b9b4b1786044
       Address=runnable:InterruptResumeDemo;node:batch_review;batch_process:2;...
       DocumentID=DOC-003, Reason=High priority document requires human approval

  Resuming with approval decisions...
    Document DOC-001 resumed with decision: approved=true
    Document DOC-003 resumed with decision: approved=true

  Final results after resume:
    - DOC-001: approved=true, comments=Human review: Approved by supervisor
    - DOC-002: approved=true, comments=Auto-approved (non-high priority)
    - DOC-003: approved=true, comments=Human review: Approved by supervisor
    - DOC-004: approved=true, comments=Auto-approved (non-high priority)
```

## Learn More

- [Eino Documentation](https://github.com/cloudwego/eino)
- [Compose Package](https://github.com/cloudwego/eino/tree/main/compose)
- [Human-in-the-Loop Examples](../graph/react_with_interrupt/)
