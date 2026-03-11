/*
 * Copyright 2026 CloudWeGo Authors
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

// Package rag provides an answer_from_document tool backed by a compose.Workflow.
//
// The workflow uses field mapping to share the user's question across non-adjacent
// nodes (score, answer) without threading it through every intermediate output type:
//
//	START{FilePath, Question}
//	  │ (data via WithNoDirectDependency)──────────────────────────────────────────┐
//	  ▼                                                                            │ Question
//	[load]  os.ReadFile → []*schema.Document                                       │
//	  ▼                                                                            │
//	[chunk] paragraph splitter → []*schema.Document                               │
//	  ▼  Chunks ─────────────────────────────────────────────────────────────► [score]
//	                                                                               │ []scoredChunk
//	                                                                               ▼
//	                                                                           [filter]  top-k
//	                                                                               │ TopK (may be empty)
//	                                                                               ▼
//	                                                                           [answer] ◄─ Question (START)
//	                                                                    (synthesize or not_found inline)
//	                                                                               │
//	                                                                              END
//
// The [score] node wraps a BatchNode whose inner workflow scores each chunk with
// a ChatModel call in parallel (MaxConcurrency=5).
package rag

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"github.com/cloudwego/eino-examples/adk/common/tool/graphtool"
	"github.com/cloudwego/eino-examples/compose/batch/batch"
)

// Input is the tool call argument struct. Its JSON tags are used by utils.InferTool
// to generate the tool's parameter schema automatically.
type Input struct {
	FilePath string `json:"file_path" jsonschema:"description=Absolute path to the uploaded document file"`
	Question string `json:"question"  jsonschema:"description=The question to answer from the document"`
}

// Output is the structured result returned by the tool.
type Output struct {
	Answer  string   `json:"answer"`
	Sources []string `json:"sources"` // key excerpts used to produce the answer
}

// scoreTask is the per-chunk input fed into the inner BatchNode workflow.
type scoreTask struct {
	Text     string
	Question string
}

// scoredChunk is the per-chunk result produced by the inner BatchNode workflow.
type scoredChunk struct {
	Text    string
	Score   int    // 0–10 relevance to the question
	Excerpt string // most relevant sentence or phrase from this chunk
}

// scoreIn is the input to the outer "score" Lambda node.
// It is assembled by field mapping from two sources:
//   - Chunks: full output of "chunk" node ([]*schema.Document)
//   - Question: Question field of START (Input)
type scoreIn struct {
	Chunks   []*schema.Document
	Question string
}

// synthIn is the input to the "synthesize" Lambda node.
// It is assembled by field mapping from two sources:
//   - TopK: full output of "filter" node ([]scoredChunk)
//   - Question: Question field of START (Input)
type synthIn struct {
	TopK     []scoredChunk
	Question string
}

// BuildTool constructs the answer_from_document tool backed by the RAG workflow.
// It uses graphtool.NewInvokableGraphTool, which compiles the workflow per invocation
// and supports interrupt/resume via a built-in checkpoint store.
func BuildTool(ctx context.Context, cm model.BaseChatModel) (tool.BaseTool, error) {
	wf := buildWorkflow(cm)
	return graphtool.NewInvokableGraphTool[Input, Output](
		wf,
		"answer_from_document",
		"Search a large uploaded document for content relevant to a question and synthesize a "+
			"cited answer from the most relevant passages. "+
			"Use this instead of read_file when the document may be too large to fit in context.",
	)
}

// buildWorkflow constructs the RAG compose.Workflow (uncompiled).
// graphtool.NewInvokableGraphTool compiles it per invocation.
func buildWorkflow(cm model.BaseChatModel) *compose.Workflow[Input, Output] {
	scoreWF := newScoreWorkflow(cm)
	scorer := batch.NewBatchNode(&batch.NodeConfig[scoreTask, scoredChunk]{
		Name:           "ChunkScorer",
		InnerTask:      scoreWF,
		MaxConcurrency: 5,
	})

	wf := compose.NewWorkflow[Input, Output]()

	// load: read file from disk, emit a single Document.
	wf.AddLambdaNode("load", compose.InvokableLambda(
		func(ctx context.Context, in Input) ([]*schema.Document, error) {
			data, err := os.ReadFile(in.FilePath)
			if err != nil {
				return nil, fmt.Errorf("read %q: %w", in.FilePath, err)
			}
			return []*schema.Document{{Content: string(data)}}, nil
		},
	)).AddInput(compose.START)

	// chunk: split each Document into ~800-char pieces.
	wf.AddLambdaNode("chunk", compose.InvokableLambda(
		func(ctx context.Context, docs []*schema.Document) ([]*schema.Document, error) {
			var out []*schema.Document
			for _, d := range docs {
				out = append(out, splitIntoChunks(d.Content, 800)...)
			}
			return out, nil
		},
	)).AddInput("load")

	// score: score each chunk against the question in parallel via BatchNode.
	// Chunks comes from "chunk"; Question comes directly from START.
	// Both use WithNoDirectDependency because the execution order is already
	// established by the direct edges START→load→chunk→score.
	wf.AddLambdaNode("score", compose.InvokableLambda(
		func(ctx context.Context, in scoreIn) ([]scoredChunk, error) {
			tasks := make([]scoreTask, len(in.Chunks))
			for i, c := range in.Chunks {
				tasks[i] = scoreTask{Text: c.Content, Question: in.Question}
			}
			return scorer.Invoke(ctx, tasks)
		},
	)).
		AddInputWithOptions("chunk",
			[]*compose.FieldMapping{compose.ToField("Chunks")},
			compose.WithNoDirectDependency()).
		AddInputWithOptions(compose.START,
			[]*compose.FieldMapping{compose.MapFields("Question", "Question")},
			compose.WithNoDirectDependency())

	// filter: sort descending by score, keep up to top-3 chunks with score ≥ 3.
	wf.AddLambdaNode("filter", compose.InvokableLambda(
		func(ctx context.Context, scored []scoredChunk) ([]scoredChunk, error) {
			sort.Slice(scored, func(i, j int) bool {
				return scored[i].Score > scored[j].Score
			})
			const maxK = 3
			var top []scoredChunk
			for _, c := range scored {
				if c.Score < 3 {
					break
				}
				top = append(top, c)
				if len(top) == maxK {
					break
				}
			}
			return top, nil
		},
	)).AddInput("score")

	// answer: synthesize a response from top-k chunks, or return a not-found message if empty.
	// TopK comes from "filter"; Question comes directly from START.
	// Both use WithNoDirectDependency: "filter" governs execution order via its direct edge.
	wf.AddLambdaNode("answer", compose.InvokableLambda(
		func(ctx context.Context, in synthIn) (Output, error) {
			if len(in.TopK) == 0 {
				return Output{
					Answer: fmt.Sprintf("No relevant content found in the document for: %q", in.Question),
				}, nil
			}
			return synthesize(ctx, cm, in)
		},
	)).
		AddInputWithOptions("filter",
			[]*compose.FieldMapping{compose.ToField("TopK")},
			compose.WithNoDirectDependency()).
		AddInputWithOptions(compose.START,
			[]*compose.FieldMapping{compose.MapFields("Question", "Question")},
			compose.WithNoDirectDependency())

	// END receives output from answer.
	wf.End().
		AddInput("answer")

	return wf
}

// newScoreWorkflow builds the single-node inner workflow used by each BatchNode task.
// It is intentionally trivial: BatchNode provides the parallelism, not the inner graph.
func newScoreWorkflow(cm model.BaseChatModel) *compose.Workflow[scoreTask, scoredChunk] {
	wf := compose.NewWorkflow[scoreTask, scoredChunk]()
	wf.AddLambdaNode("score_chunk", compose.InvokableLambda(
		func(ctx context.Context, t scoreTask) (scoredChunk, error) {
			return scoreOneChunk(ctx, cm, t)
		},
	)).AddInput(compose.START)
	wf.End().AddInput("score_chunk")
	return wf
}

// scoreOneChunk asks the model to rate the relevance of a single chunk (0–10)
// and extract the most relevant excerpt. Parse errors are treated as score 0
// so a bad JSON response never aborts the pipeline.
func scoreOneChunk(ctx context.Context, cm model.BaseChatModel, t scoreTask) (scoredChunk, error) {
	prompt := fmt.Sprintf(`Rate how relevant the following text chunk is to the question.

Question: %s

Chunk:
%s

Reply with JSON only — no explanation, no markdown fences:
{"score": <0-10>, "excerpt": "<most relevant sentence or phrase, empty string if score is 0>"}

Score guide: 0=completely irrelevant, 3=tangentially related, 7=clearly relevant, 10=directly answers the question.`,
		t.Question, t.Text)

	resp, err := cm.Generate(ctx, []*schema.Message{schema.UserMessage(prompt)})
	if err != nil {
		// treat model error as irrelevant rather than aborting the batch
		return scoredChunk{Text: t.Text, Score: 0}, nil
	}

	content := strings.TrimSpace(resp.Content)
	// strip optional markdown code block wrapper
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var sr struct {
		Score   int    `json:"score"`
		Excerpt string `json:"excerpt"`
	}
	if err := json.Unmarshal([]byte(content), &sr); err != nil {
		return scoredChunk{Text: t.Text, Score: 0}, nil
	}
	return scoredChunk{Text: t.Text, Score: sr.Score, Excerpt: sr.Excerpt}, nil
}

// synthesize builds a prompt from the top-k chunks and generates a cited answer.
func synthesize(ctx context.Context, cm model.BaseChatModel, in synthIn) (Output, error) {
	var sb strings.Builder
	sb.WriteString("Answer the following question using only the provided document excerpts.\n\n")
	sb.WriteString("Question: ")
	sb.WriteString(in.Question)
	sb.WriteString("\n\nDocument excerpts:\n")

	sources := make([]string, len(in.TopK))
	for i, c := range in.TopK {
		excerpt := c.Excerpt
		if excerpt == "" {
			excerpt = c.Text
		}
		sources[i] = excerpt
		fmt.Fprintf(&sb, "[%d] %s\n\n", i+1, excerpt)
	}
	sb.WriteString("Provide a clear, concise answer. Cite excerpt numbers like [1] when referencing sources.")

	resp, err := cm.Generate(ctx, []*schema.Message{schema.UserMessage(sb.String())})
	if err != nil {
		return Output{}, fmt.Errorf("synthesize: %w", err)
	}
	return Output{Answer: resp.Content, Sources: sources}, nil
}

// splitIntoChunks splits text into chunks of at most chunkSize characters,
// breaking on paragraph boundaries (\n\n) where possible, then on newlines.
func splitIntoChunks(text string, chunkSize int) []*schema.Document {
	var chunks []*schema.Document
	var buf strings.Builder

	flush := func() {
		s := strings.TrimSpace(buf.String())
		if s != "" {
			chunks = append(chunks, &schema.Document{Content: s})
		}
		buf.Reset()
	}

	for _, para := range strings.Split(text, "\n\n") {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}
		if buf.Len()+len(para)+2 > chunkSize && buf.Len() > 0 {
			flush()
		}
		// paragraph itself exceeds chunkSize: split by line
		if len(para) > chunkSize {
			for _, line := range strings.Split(para, "\n") {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				if buf.Len()+len(line)+1 > chunkSize && buf.Len() > 0 {
					flush()
				}
				if buf.Len() > 0 {
					buf.WriteByte('\n')
				}
				buf.WriteString(line)
			}
		} else {
			if buf.Len() > 0 {
				buf.WriteString("\n\n")
			}
			buf.WriteString(para)
		}
	}
	flush()
	return chunks
}
