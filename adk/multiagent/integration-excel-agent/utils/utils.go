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

package utils

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/cloudwego/eino-ext/components/tool/commandline"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
	"github.com/kaptinlin/jsonrepair"
)

type panicErr struct {
	info  any
	stack []byte
}

func (p *panicErr) Error() string {
	return fmt.Sprintf("panic error: %v, \nstack: %s", p.info, string(p.stack))
}

// NewPanicErr creates a new panic error.
// panicErr is a wrapper of panic info and stack trace.
// it implements the error interface, can print error message of info and stack trace.
func NewPanicErr(info any, stack []byte) error {
	return &panicErr{
		info:  info,
		stack: stack,
	}
}

func PtrOf[T any](v T) *T {
	return &v
}

func FormatInput(input []adk.Message) string {
	var sb strings.Builder
	for _, msg := range input {
		sb.WriteString(msg.Content)
		sb.WriteString("\n")
	}

	return sb.String()
}

type TaskGroup interface {
	Go(f func() error)
	Wait() error
}

func ToJSONString(v interface{}) string {
	str, _ := sonic.MarshalString(v)
	return str
}

func GetCurrentTime() string {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		// 出现错误时 fallback 到本地时间
		return time.Now().Format("2006-01-02 15:04:05 MST")
	}
	return time.Now().In(loc).Format("2006-01-02 15:04:05 MST")
}

func RepairJSON(input string) string {
	input = strings.TrimPrefix(input, "<|FunctionCallBegin|>")
	input = strings.TrimSuffix(input, "<|FunctionCallEnd|>")
	input = strings.TrimPrefix(input, "<think>")
	output, err := jsonrepair.JSONRepair(input)
	if err != nil {
		return input
	}

	return output
}

func GetSessionValue[T any](ctx context.Context, key string) (T, bool) {
	v, ok := adk.GetSessionValue(ctx, key)
	if !ok {
		var zero T
		return zero, false
	}
	t, ok := v.(T)
	if !ok {
		var zero T
		return zero, false
	}

	return t, true
}

func FormatExecutedSteps(in []planexecute.ExecutedStep) string {
	var sb strings.Builder
	for idx, m := range in {
		sb.WriteString(fmt.Sprintf("## %d. Step: %v\n  Result: %v\n\n", idx+1, m.Step, m.Result))
	}
	return sb.String()
}

func FormatCommandOutput(output *commandline.CommandOutput) string {
	return fmt.Sprintf("---\nstdout:%v\n---\nstderr:%v\n---", output.Stdout, output.Stderr)
}
