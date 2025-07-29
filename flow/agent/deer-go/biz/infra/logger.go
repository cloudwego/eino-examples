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

package infra

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/RanFeng/ilog"
	"github.com/cloudwego/eino/callbacks"
	ecmodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/cloudwego/hertz/pkg/protocol/sse"

	"github.com/cloudwego/eino-examples/flow/agent/deer-go/biz/model"
	"github.com/cloudwego/eino-examples/flow/agent/deer-go/biz/util"
)

type LoggerCallback struct {
	callbacks.HandlerBuilder // 可以用 callbacks.HandlerBuilder 来辅助实现 callback

	ID  string
	SSE *sse.Writer
	Out chan string
}

func (cb *LoggerCallback) pushF(ctx context.Context, event string, data *model.ChatResp) error {
	dataByte, err := json.Marshal(data)
	if err != nil {
		ilog.EventError(ctx, err, "json_marshal_error", "data", data)
		return err
	}
	if cb.SSE != nil {
		err = cb.SSE.WriteEvent("", event, dataByte)
	}
	if cb.Out != nil {
		cb.Out <- data.Content
	}
	return nil
}

func (cb *LoggerCallback) pushMsg(ctx context.Context, msgID string, msg *schema.Message) error {
	if msg == nil {
		return nil
	}

	agentName := ""
	_ = compose.ProcessState[*model.State](ctx, func(_ context.Context, state *model.State) error {
		agentName = state.Goto
		return nil
	})

	fr := ""
	if msg.ResponseMeta != nil {
		fr = msg.ResponseMeta.FinishReason
	}
	data := &model.ChatResp{
		ThreadID:      cb.ID,
		Agent:         agentName,
		ID:            msgID,
		Role:          "assistant",
		Content:       msg.Content,
		FinishReason:  fr,
		MessageChunks: msg.Content,
	}

	if msg.Role == schema.Tool {
		data.ToolCallID = msg.ToolCallID
		return cb.pushF(ctx, "tool_call_result", data)
	}

	if len(msg.ToolCalls) > 0 {
		event := "tool_call_chunks"
		if len(msg.ToolCalls) != 1 {
			ilog.EventWarn(ctx, "sse_tool_calls", "raw", msg)
			return nil
		}

		ts := []model.ToolResp{}
		tcs := []model.ToolChunkResp{}
		fn := msg.ToolCalls[0].Function.Name
		if len(fn) > 0 {
			event = "tool_calls"
			if strings.HasSuffix(fn, "search") {
				fn = "web_search"
			}
			ts = append(ts, model.ToolResp{
				Name: fn,
				Args: map[string]interface{}{},
				Type: "tool_call",
				ID:   msg.ToolCalls[0].ID,
			})
		}
		tcs = append(tcs, model.ToolChunkResp{
			Name: fn,
			Args: msg.ToolCalls[0].Function.Arguments,
			Type: "tool_call_chunk",
			ID:   msg.ToolCalls[0].ID,
		})
		data.ToolCalls = ts
		data.ToolCallChunks = tcs
		return cb.pushF(ctx, event, data)
	}
	return cb.pushF(ctx, "message_chunk", data)
}

func (cb *LoggerCallback) OnStart(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
	if inputStr, ok := input.(string); ok {
		if cb.Out != nil {
			cb.Out <- "\n==================\n"
			cb.Out <- fmt.Sprintf(" [OnStart] %s ", inputStr)
			cb.Out <- "\n==================\n"
		}
	}
	return ctx
}

func (cb *LoggerCallback) OnEnd(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
	//fmt.Println("=========[OnEnd]=========", info.Name, "|", info.Component, "|", info.Type)
	//outputStr, _ := json.MarshalIndent(output, "", "  ") // nolint: byted_s_returned_err_check
	//if len(outputStr) > 200 {
	//	outputStr = outputStr[:200]
	//}
	//fmt.Println(string(outputStr))
	return ctx
}

func (cb *LoggerCallback) OnError(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
	fmt.Println("=========[OnError]=========")
	fmt.Println(err)
	return ctx
}

func (cb *LoggerCallback) OnEndWithStreamOutput(ctx context.Context, info *callbacks.RunInfo,
	output *schema.StreamReader[callbacks.CallbackOutput]) context.Context {
	msgID := util.RandStr(20)
	go func() {
		defer output.Close() // remember to close the stream in defer
		defer func() {
			if err := recover(); err != nil {
				ilog.EventFatal(ctx, "[OnEndStream]panic_recover", "msgID", msgID, "err", err)
			}
		}()
		for {
			frame, err := output.Recv()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				ilog.EventError(ctx, err, "[OnEndStream] recv_error")
				return
			}

			switch v := frame.(type) {
			case *schema.Message:
				_ = cb.pushMsg(ctx, msgID, v)
			case *ecmodel.CallbackOutput:
				_ = cb.pushMsg(ctx, msgID, v.Message)
			case []*schema.Message:
				for _, m := range v {
					_ = cb.pushMsg(ctx, msgID, m)
				}
			//case string:
			//	ilog.EventInfo(ctx, "frame_type", "type", "str", "v", v)
			default:
				//ilog.EventInfo(ctx, "frame_type", "type", "unknown", "v", v)
			}
		}

	}()
	return ctx
}

func (cb *LoggerCallback) OnStartWithStreamInput(ctx context.Context, info *callbacks.RunInfo,
	input *schema.StreamReader[callbacks.CallbackInput]) context.Context {
	defer input.Close()
	return ctx
}
