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

package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/cloudwego/eino/compose"
)

// ============================================================
// State Graph 示例：多轮翻译助手
//
// 循环"翻译→审校"，直到质量达标或达到最大轮次。
// State 的核心价值：Branch 中根据 State 做有状态的决策（闭包变量做不到）
//
//	START → translate ──→ review ──→ Branch(质量达标?) ──→ END
//	          ↑                          │ 否
//	          └──────────────────────────┘
//
// State 访问方式：
//   - 方式1: WithStatePreHandler / WithStatePostHandler — 节点执行前/后自动传入
//   - 方式2: compose.ProcessState — Lambda 内部或 Branch 中手动获取
// ============================================================

type translateState struct {
	round   int      // 当前轮次
	history []string // 翻译历史
}

const (
	nodeTranslate = "translate"
	nodeReview    = "review"
)

// 演示【方式2】ProcessState：在 Lambda 内部主动获取 State
func newTranslateNode() *compose.Lambda {
	return compose.InvokableLambda(func(ctx context.Context, in string) (string, error) {
		var roundNum int
		var historyStr string

		compose.ProcessState[*translateState](ctx, func(ctx context.Context, s *translateState) error {
			roundNum = s.round
			if len(s.history) > 0 {
				historyStr = "\n\nPrevious translations:\n" + strings.Join(s.history, "\n")
			}
			return nil
		})

		return fmt.Sprintf("[Round %d translation]%s\n  %s", roundNum, historyStr, mockTranslate(in)), nil
	})
}

func newReviewNode() *compose.Lambda {
	return compose.InvokableLambda(func(ctx context.Context, in string) (string, error) {
		return mockReview(in), nil
	})
}

// 演示【方式1】WithStatePreHandler：节点执行前修改 State
func roundIncrement(ctx context.Context, in string, state *translateState) (string, error) {
	state.round++
	fmt.Printf("  [State] round incremented to %d\n", state.round)
	return in, nil
}

// 演示【方式1】WithStatePostHandler：节点执行后修改 State
func saveHistory(ctx context.Context, out string, state *translateState) (string, error) {
	state.history = append(state.history, out)
	fmt.Printf("  [State] history saved, total %d entries\n", len(state.history))
	return out, nil
}

// ★ State 最不可替代的场景：Branch 条件函数中通过 ProcessState 读取 State
// （闭包变量在 Branch 中不可访问，State 是唯一途径）
func qualityBranch(ctx context.Context, out string) (string, error) {
	var next string
	compose.ProcessState[*translateState](ctx, func(ctx context.Context, s *translateState) error {
		quality := mockQualityCheck(out)
		fmt.Printf("  [Branch] round=%d, quality=%d/10\n", s.round, quality)

		if quality >= 8 || s.round >= 3 {
			next = compose.END
			fmt.Printf("  [Branch] → END\n")
		} else {
			next = nodeTranslate
			fmt.Printf("  [Branch] → translate\n")
		}
		return nil
	})
	return next, nil
}

func buildGraph(ctx context.Context) (compose.Runnable[string, string], error) {
	genState := func(ctx context.Context) *translateState { return &translateState{} }

	g := compose.NewGraph[string, string](compose.WithGenLocalState(genState))

	g.AddLambdaNode(nodeTranslate, newTranslateNode(),
		compose.WithStatePreHandler(roundIncrement))

	g.AddLambdaNode(nodeReview, newReviewNode(),
		compose.WithStatePostHandler(saveHistory))

	g.AddBranch(nodeReview, compose.NewGraphBranch(
		qualityBranch,
		map[string]bool{compose.END: true, nodeTranslate: true},
	))

	g.AddEdge(compose.START, nodeTranslate)
	g.AddEdge(nodeTranslate, nodeReview)

	return g.Compile(ctx, compose.WithMaxRunSteps(10))
}

func mockTranslate(in string) string {
	return "快速的棕色狐狸跳过了懒狗"
}

func mockReview(in string) string {
	return in + " [reviewed]"
}

func mockQualityCheck(in string) int {
	reviewed := strings.Count(in, "[reviewed]")
	switch {
	case reviewed >= 3:
		return 9
	case reviewed >= 2:
		return 7
	default:
		return 5
	}
}

func main() {
	ctx := context.Background()

	run, err := buildGraph(ctx)
	if err != nil {
		fmt.Printf("Compile failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("=== Input: \"The quick brown fox jumps over the lazy dog\" ===")
	fmt.Println()

	result, err := run.Invoke(ctx, "The quick brown fox jumps over the lazy dog")
	if err != nil {
		fmt.Printf("Invoke failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n=== Final Result ===\n%s\n", result)
}
