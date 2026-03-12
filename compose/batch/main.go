/*
 * Copyright 2025 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// This example demonstrates the BatchNode component for processing multiple inputs
// through a Graph or Workflow with configurable concurrency and interrupt/resume support.
//
// Business Scenario: Document Review Pipeline
// A compliance team needs to review multiple documents. Each document goes through
// an automated review workflow, with high-priority documents requiring human approval.
//
// Scenarios covered:
//  1. Basic Sequential Processing - Process documents one at a time
//  2. Concurrent Processing - Process multiple documents in parallel
//  3. Compile Options - Configure inner workflow at compile time
//  4. Invoke Options (Callbacks) - Add callbacks for monitoring
//  5. Error Handling - Handle errors from individual tasks
//  6. Interrupt & Resume - Human-in-the-loop for high-priority documents
//  7. Parent Graph with Reduce - Integrate BatchNode in a larger pipeline
package main

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"github.com/cloudwego/eino-examples/compose/batch/batch"
)

func init() {
	// Register types for serialization (required for interrupt/resume checkpoint)
	schema.RegisterName[ReviewRequest]("batch_example.ReviewRequest")
	schema.RegisterName[ReviewResult]("batch_example.ReviewResult")
	schema.RegisterName[*ApprovalDecision]("batch_example.ApprovalDecision")
	schema.RegisterName[[]ReviewRequest]("batch_example.ReviewRequestSlice")
	schema.RegisterName[[]ReviewResult]("batch_example.ReviewResultSlice")
}

// ReviewRequest represents a document to be reviewed
type ReviewRequest struct {
	DocumentID string
	Content    string
	Priority   string // "high", "medium", "low"
}

// ReviewResult contains the outcome of a document review
type ReviewResult struct {
	DocumentID string
	Approved   bool
	Score      float64
	Comments   string
	ReviewedAt time.Time
}

// ReviewReport aggregates results from batch processing
type ReviewReport struct {
	TotalDocuments   int
	ApprovedCount    int
	RejectedCount    int
	AverageScore     float64
	HighPriorityPass int
	Results          []ReviewResult
	GeneratedAt      time.Time
}

// ApprovalDecision is the human decision for interrupted documents
type ApprovalDecision struct {
	Approved bool
	Comments string
}

// BatchReviewInput wraps documents with batch metadata
type BatchReviewInput struct {
	Documents []ReviewRequest
	BatchName string
}

func main() {
	ctx := context.Background()

	fmt.Println("=== Document Review Pipeline Example ===")
	fmt.Println()

	fmt.Println("--- Scenario 1: Basic Sequential Processing ---")
	runBasicSequential(ctx)
	fmt.Println()

	fmt.Println("--- Scenario 2: Concurrent Processing ---")
	runConcurrent(ctx)
	fmt.Println()

	fmt.Println("--- Scenario 3: With Compile Options ---")
	runWithCompileOptions(ctx)
	fmt.Println()

	fmt.Println("--- Scenario 4: With Invoke Options (Callbacks) ---")
	runWithInvokeOptions(ctx)
	fmt.Println()

	fmt.Println("--- Scenario 5: Normal Error Handling ---")
	runWithError(ctx)
	fmt.Println()

	fmt.Println("--- Scenario 6: Interrupt & Resume ---")
	runInterruptAndResume(ctx)
	fmt.Println()

	fmt.Println("--- Scenario 7: Parent Graph with Reduce Node ---")
	runParentGraphWithReduce(ctx)
	fmt.Println()

	fmt.Println("=== All Scenarios Completed ===")
}

// createSampleDocuments generates test documents with rotating priorities
func createSampleDocuments(count int) []ReviewRequest {
	priorities := []string{"high", "medium", "low"}
	docs := make([]ReviewRequest, count)
	for i := 0; i < count; i++ {
		docs[i] = ReviewRequest{
			DocumentID: fmt.Sprintf("DOC-%03d", i+1),
			Content:    fmt.Sprintf("Document content for review #%d. This is a sample compliance document.", i+1),
			Priority:   priorities[i%len(priorities)],
		}
	}
	return docs
}

// createSimpleReviewWorkflow creates a basic document review workflow
// that simulates automated review with random scoring
func createSimpleReviewWorkflow() *compose.Workflow[ReviewRequest, ReviewResult] {
	workflow := compose.NewWorkflow[ReviewRequest, ReviewResult]()

	workflow.AddLambdaNode("analyze", compose.InvokableLambda(func(ctx context.Context, req ReviewRequest) (ReviewResult, error) {
		time.Sleep(50 * time.Millisecond) // Simulate processing time

		score := 0.5 + rand.Float64()*0.5
		approved := score >= 0.7

		return ReviewResult{
			DocumentID: req.DocumentID,
			Approved:   approved,
			Score:      score,
			Comments:   fmt.Sprintf("Auto-reviewed document %s with priority %s", req.DocumentID, req.Priority),
			ReviewedAt: time.Now(),
		}, nil
	})).AddInput(compose.START)

	workflow.End().AddInput("analyze")

	return workflow
}

// Scenario 1: Basic Sequential Processing
// Demonstrates: MaxConcurrency=0 for sequential execution
func runBasicSequential(ctx context.Context) {
	docs := createSampleDocuments(3)
	workflow := createSimpleReviewWorkflow()

	batchNode := batch.NewBatchNode(&batch.NodeConfig[ReviewRequest, ReviewResult]{
		Name:           "SequentialReviewer",
		InnerTask:      workflow,
		MaxConcurrency: 0, // Sequential: process one at a time
	})

	start := time.Now()
	results, err := batchNode.Invoke(ctx, docs)
	elapsed := time.Since(start)

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Processed %d documents sequentially in %v\n", len(results), elapsed)
	for _, r := range results {
		fmt.Printf("  - %s: approved=%v, score=%.2f\n", r.DocumentID, r.Approved, r.Score)
	}
}

// Scenario 2: Concurrent Processing
// Demonstrates: MaxConcurrency>0 for parallel execution with limit
func runConcurrent(ctx context.Context) {
	docs := createSampleDocuments(5)
	workflow := createSimpleReviewWorkflow()

	batchNode := batch.NewBatchNode(&batch.NodeConfig[ReviewRequest, ReviewResult]{
		Name:           "ConcurrentReviewer",
		InnerTask:      workflow,
		MaxConcurrency: 3, // Concurrent: up to 3 parallel tasks
	})

	start := time.Now()
	results, err := batchNode.Invoke(ctx, docs)
	elapsed := time.Since(start)

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Processed %d documents concurrently (max 3) in %v\n", len(results), elapsed)
	for _, r := range results {
		fmt.Printf("  - %s: approved=%v, score=%.2f\n", r.DocumentID, r.Approved, r.Score)
	}
}

// Scenario 3: With Compile Options
// Demonstrates: InnerCompileOptions for configuring inner workflow at compile time
func runWithCompileOptions(ctx context.Context) {
	docs := createSampleDocuments(3)
	workflow := createSimpleReviewWorkflow()

	batchNode := batch.NewBatchNode(&batch.NodeConfig[ReviewRequest, ReviewResult]{
		Name:           "NamedReviewer",
		InnerTask:      workflow,
		MaxConcurrency: 2,
		InnerCompileOptions: []compose.GraphCompileOption{
			compose.WithGraphName("SingleDocumentReviewWorkflow"), // Name for debugging/tracing
		},
	})

	results, err := batchNode.Invoke(ctx, docs)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Processed %d documents with named inner workflow\n", len(results))
	for _, r := range results {
		fmt.Printf("  - %s: approved=%v\n", r.DocumentID, r.Approved)
	}
}

// Scenario 4: With Invoke Options (Callbacks)
// Demonstrates: Callbacks for batch-level monitoring via context
func runWithInvokeOptions(ctx context.Context) {
	docs := createSampleDocuments(3)
	workflow := createSimpleReviewWorkflow()

	batchNode := batch.NewBatchNode(&batch.NodeConfig[ReviewRequest, ReviewResult]{
		Name:           "CallbackReviewer",
		InnerTask:      workflow,
		MaxConcurrency: 0,
	})

	// Create callback handler for monitoring
	handler := callbacks.NewHandlerBuilder().
		OnStartFn(func(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
			fmt.Printf("  [Callback] OnStart: %s/%s\n", info.Component, info.Name)
			return ctx
		}).
		OnEndFn(func(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
			fmt.Printf("  [Callback] OnEnd: %s/%s\n", info.Component, info.Name)
			return ctx
		}).
		OnErrorFn(func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
			fmt.Printf("  [Callback] OnError: %s/%s - %v\n", info.Component, info.Name, err)
			return ctx
		}).
		Build()

	// Initialize callbacks in context
	ctxWithCallback := callbacks.InitCallbacks(ctx, nil, handler)

	results, err := batchNode.Invoke(ctxWithCallback, docs)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Processed %d documents with callbacks\n", len(results))
}

// Scenario 5: Normal Error Handling
// Demonstrates: How BatchNode handles errors from individual tasks
func runWithError(ctx context.Context) {
	workflow := compose.NewWorkflow[ReviewRequest, ReviewResult]()

	workflow.AddLambdaNode("analyze", compose.InvokableLambda(func(ctx context.Context, req ReviewRequest) (ReviewResult, error) {
		// Simulate validation failure for specific document
		if req.DocumentID == "DOC-002" {
			return ReviewResult{}, fmt.Errorf("validation failed for document %s: content too short", req.DocumentID)
		}
		return ReviewResult{
			DocumentID: req.DocumentID,
			Approved:   true,
			Score:      0.9,
			Comments:   "Passed validation",
			ReviewedAt: time.Now(),
		}, nil
	})).AddInput(compose.START)

	workflow.End().AddInput("analyze")

	docs := createSampleDocuments(3)
	batchNode := batch.NewBatchNode(&batch.NodeConfig[ReviewRequest, ReviewResult]{
		Name:           "ErrorHandlingReviewer",
		InnerTask:      workflow,
		MaxConcurrency: 0,
	})

	results, err := batchNode.Invoke(ctx, docs)
	if err != nil {
		fmt.Printf("Expected error occurred: %v\n", err)
		return
	}

	fmt.Printf("Results: %v\n", results)
}

// Scenario 6: Interrupt & Resume
// Demonstrates: Human-in-the-loop workflow using compose.Interrupt and compose.BatchResumeWithData
//
// Flow:
//  1. First invocation: High-priority documents interrupt for human review
//  2. Extract interrupt contexts with document IDs
//  3. Resume with approval decisions using BatchResumeWithData
//  4. Interrupted tasks complete with human decisions
func runInterruptAndResume(ctx context.Context) {
	innerWorkflow := compose.NewWorkflow[ReviewRequest, ReviewResult]()

	innerWorkflow.AddLambdaNode("analyze", compose.InvokableLambda(func(ctx context.Context, req ReviewRequest) (ReviewResult, error) {
		if req.Priority == "high" {
			// Check if this is a resume from previous interrupt
			wasInterrupted, _, _ := compose.GetInterruptState[any](ctx)
			if !wasInterrupted {
				// First run: interrupt for human review
				fmt.Printf("    Document %s requires human review (high priority)\n", req.DocumentID)
				return ReviewResult{}, compose.Interrupt(ctx, map[string]string{
					"document_id": req.DocumentID,
					"reason":      "High priority document requires human approval",
				})
			}

			// Resume: check if we have approval decision
			isResumeTarget, hasData, decision := compose.GetResumeContext[*ApprovalDecision](ctx)
			if isResumeTarget && hasData && decision != nil {
				fmt.Printf("    Document %s resumed with decision: approved=%v\n", req.DocumentID, decision.Approved)
				return ReviewResult{
					DocumentID: req.DocumentID,
					Approved:   decision.Approved,
					Score:      1.0,
					Comments:   fmt.Sprintf("Human review: %s", decision.Comments),
					ReviewedAt: time.Now(),
				}, nil
			}

			// Still waiting for decision
			return ReviewResult{}, compose.Interrupt(ctx, map[string]string{
				"document_id": req.DocumentID,
				"reason":      "Still waiting for human approval",
			})
		}

		// Non-high priority: auto-approve
		return ReviewResult{
			DocumentID: req.DocumentID,
			Approved:   true,
			Score:      0.85,
			Comments:   "Auto-approved (non-high priority)",
			ReviewedAt: time.Now(),
		}, nil
	})).AddInput(compose.START)

	innerWorkflow.End().AddInput("analyze")

	batchNode := batch.NewBatchNode(&batch.NodeConfig[ReviewRequest, ReviewResult]{
		Name:           "InterruptReviewer",
		InnerTask:      innerWorkflow,
		MaxConcurrency: 0,
	})

	// Wrap BatchNode in a parent graph for proper interrupt handling
	parentGraph := compose.NewGraph[[]ReviewRequest, []ReviewResult]()
	_ = parentGraph.AddLambdaNode("batch_review", compose.InvokableLambda(func(ctx context.Context, inputs []ReviewRequest) ([]ReviewResult, error) {
		return batchNode.Invoke(ctx, inputs)
	}))
	_ = parentGraph.AddEdge(compose.START, "batch_review")
	_ = parentGraph.AddEdge("batch_review", compose.END)

	// Compile with checkpoint store for state persistence
	store := newMemoryCheckpointStore()
	runner, err := parentGraph.Compile(ctx,
		compose.WithGraphName("InterruptResumeDemo"),
		compose.WithCheckPointStore(store),
	)
	if err != nil {
		fmt.Printf("Failed to compile graph: %v\n", err)
		return
	}

	docs := []ReviewRequest{
		{DocumentID: "DOC-001", Content: "Content 1", Priority: "high"},
		{DocumentID: "DOC-002", Content: "Content 2", Priority: "medium"},
		{DocumentID: "DOC-003", Content: "Content 3", Priority: "high"},
		{DocumentID: "DOC-004", Content: "Content 4", Priority: "low"},
	}

	checkpointID := "interrupt-resume-demo-001"

	// Step 1: First invocation - high priority docs will interrupt
	fmt.Println("First invocation (will interrupt for high priority docs):")
	results, err := runner.Invoke(ctx, docs, compose.WithCheckPointID(checkpointID))

	if err != nil {
		// Step 2: Extract interrupt info
		info, infoOk := compose.ExtractInterruptInfo(err)
		if infoOk && len(info.InterruptContexts) > 0 {
			fmt.Printf("\n  Interrupt detected! Found %d interrupt context(s):\n", len(info.InterruptContexts))

			// Step 3: Prepare resume data with approval decisions
			resumeData := make(map[string]any)
			for i, iCtx := range info.InterruptContexts {
				infoMap, _ := iCtx.Info.(map[string]string)
				docID := infoMap["document_id"]
				fmt.Printf("    %d. ID=%s\n", i+1, iCtx.ID)
				fmt.Printf("       Address=%v\n", iCtx.Address)
				fmt.Printf("       DocumentID=%s, Reason=%s\n", docID, infoMap["reason"])
				resumeData[iCtx.ID] = &ApprovalDecision{
					Approved: true,
					Comments: fmt.Sprintf("Approved by supervisor for %s", docID),
				}
			}

			// Step 4: Resume with approval decisions
			fmt.Println("\n  Resuming with approval decisions...")
			resumeCtx := compose.BatchResumeWithData(ctx, resumeData)
			results, err = runner.Invoke(resumeCtx, nil, compose.WithCheckPointID(checkpointID))
			if err != nil {
				fmt.Printf("  Resume error: %v\n", err)
				return
			}

			fmt.Println("\n  Final results after resume:")
			for _, r := range results {
				fmt.Printf("    - %s: approved=%v, comments=%s\n", r.DocumentID, r.Approved, r.Comments)
			}
			return
		}
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println("\nFinal results:")
	for _, r := range results {
		fmt.Printf("  - %s: approved=%v, comments=%s\n", r.DocumentID, r.Approved, r.Comments)
	}
}

// memoryCheckpointStore is a simple in-memory checkpoint store for demos
type memoryCheckpointStore struct {
	mu   sync.RWMutex
	data map[string][]byte
}

func newMemoryCheckpointStore() *memoryCheckpointStore {
	return &memoryCheckpointStore{
		data: make(map[string][]byte),
	}
}

func (m *memoryCheckpointStore) Get(_ context.Context, id string) ([]byte, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	data, ok := m.data[id]
	return data, ok, nil
}

func (m *memoryCheckpointStore) Set(_ context.Context, id string, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[id] = data
	return nil
}

// Scenario 7: Parent Graph with Reduce Node
// Demonstrates:
//   - BatchNode as a node in a larger pipeline
//   - WithInnerOptions for passing runtime options (callbacks) to inner tasks
//   - Reduce pattern: aggregate batch results into a summary report
func runParentGraphWithReduce(ctx context.Context) {
	innerWorkflow := createSimpleReviewWorkflow()

	batchNode := batch.NewBatchNode(&batch.NodeConfig[ReviewRequest, ReviewResult]{
		Name:           "BatchDocumentReviewer",
		InnerTask:      innerWorkflow,
		MaxConcurrency: 3,
		InnerCompileOptions: []compose.GraphCompileOption{
			compose.WithGraphName("SingleDocReview"),
		},
	})

	parentGraph := compose.NewGraph[BatchReviewInput, ReviewReport]()

	// Node 1: Preprocess - extract documents from batch input
	_ = parentGraph.AddLambdaNode("preprocess", compose.InvokableLambda(func(ctx context.Context, input BatchReviewInput) ([]ReviewRequest, error) {
		fmt.Printf("  Preprocessing batch '%s' with %d documents\n", input.BatchName, len(input.Documents))
		return input.Documents, nil
	}))

	// Node 2: Batch Review - process all documents with progress tracking
	_ = parentGraph.AddLambdaNode("batch_review", compose.InvokableLambda(func(ctx context.Context, inputs []ReviewRequest) ([]ReviewResult, error) {
		fmt.Printf("  Starting batch review of %d documents\n", len(inputs))

		// Progress tracking with atomic counter (thread-safe for concurrent processing)
		var completedCount int32
		totalCount := int32(len(inputs))

		// Create callback for progress tracking
		reviewProgressHandler := callbacks.NewHandlerBuilder().
			OnEndFn(func(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
				if info.Component == "Workflow" {
					current := atomic.AddInt32(&completedCount, 1)
					fmt.Printf("    [Progress] %d/%d documents reviewed\n", current, totalCount)
				}
				return ctx
			}).
			Build()

		// Use WithInnerOptions to pass callbacks to each inner task
		return batchNode.Invoke(ctx, inputs,
			batch.WithInnerOptions(compose.WithCallbacks(reviewProgressHandler)),
		)
	}))

	// Node 3: Reduce - aggregate results into a report
	_ = parentGraph.AddLambdaNode("reduce", compose.InvokableLambda(func(ctx context.Context, results []ReviewResult) (ReviewReport, error) {
		fmt.Printf("  Reducing %d results into report\n", len(results))

		report := ReviewReport{
			TotalDocuments: len(results),
			Results:        results,
			GeneratedAt:    time.Now(),
		}

		var totalScore float64
		for _, r := range results {
			totalScore += r.Score
			if r.Approved {
				report.ApprovedCount++
			} else {
				report.RejectedCount++
			}
		}

		if len(results) > 0 {
			report.AverageScore = totalScore / float64(len(results))
		}

		return report, nil
	}))

	// Connect nodes: preprocess -> batch_review -> reduce
	_ = parentGraph.AddEdge(compose.START, "preprocess")
	_ = parentGraph.AddEdge("preprocess", "batch_review")
	_ = parentGraph.AddEdge("batch_review", "reduce")
	_ = parentGraph.AddEdge("reduce", compose.END)

	runner, err := parentGraph.Compile(ctx, compose.WithGraphName("DocumentReviewPipeline"))
	if err != nil {
		fmt.Printf("Failed to compile parent graph: %v\n", err)
		return
	}

	input := BatchReviewInput{
		Documents: createSampleDocuments(5),
		BatchName: "Q4-Compliance-Review",
	}

	fmt.Println("Running document review pipeline:")
	report, err := runner.Invoke(ctx, input)
	if err != nil {
		fmt.Printf("Pipeline error: %v\n", err)
		return
	}

	fmt.Println("\n=== Review Report ===")
	fmt.Printf("  Total Documents:  %d\n", report.TotalDocuments)
	fmt.Printf("  Approved:         %d\n", report.ApprovedCount)
	fmt.Printf("  Rejected:         %d\n", report.RejectedCount)
	fmt.Printf("  Average Score:    %.2f\n", report.AverageScore)
	fmt.Printf("  Generated At:     %s\n", report.GeneratedAt.Format(time.RFC3339))
	fmt.Println("\n  Individual Results:")
	for _, r := range report.Results {
		status := "✓"
		if !r.Approved {
			status = "✗"
		}
		fmt.Printf("    %s %s (score: %.2f)\n", status, r.DocumentID, r.Score)
	}
}
