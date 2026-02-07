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

package graphtool

import (
	"context"
	"fmt"
	"io"
	"reflect"

	"github.com/bytedance/sonic"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

type Compilable[I, O any] interface {
	Compile(ctx context.Context, opts ...compose.GraphCompileOption) (compose.Runnable[I, O], error)
}

type InvokableGraphTool[I, O any] struct {
	compilable     Compilable[I, O]
	compileOptions []compose.GraphCompileOption
	tInfo          *schema.ToolInfo
}

func NewInvokableGraphTool[I, O any](compilable Compilable[I, O],
	name, desc string,
	opts ...compose.GraphCompileOption,
) (*InvokableGraphTool[I, O], error) {
	tInfo, err := utils.GoStruct2ToolInfo[I](name, desc)
	if err != nil {
		return nil, err
	}

	return &InvokableGraphTool[I, O]{
		compilable:     compilable,
		compileOptions: opts,
		tInfo:          tInfo,
	}, nil
}

type graphToolOptions struct {
	composeOpts []compose.Option
}

func WithGraphToolOption(opts ...compose.Option) tool.Option {
	return tool.WrapImplSpecificOptFn(func(opt *graphToolOptions) {
		opt.composeOpts = opts
	})
}

type graphToolInterruptState struct {
	Data      []byte
	ToolInput string
}

func init() {
	schema.RegisterName[*graphToolInterruptState]("_eino_graph_tool_interrupt_state")
}

func (g *InvokableGraphTool[I, O]) InvokableRun(ctx context.Context, input string,
	opts ...tool.Option) (output string, err error) {
	var (
		checkpointStore *graphToolStore
		inputParams     I
		originOutput    O
		runnable        compose.Runnable[I, O]
	)

	callOpts := tool.GetImplSpecificOptions(&graphToolOptions{}, opts...).composeOpts
	callOpts = append(callOpts, compose.WithCheckPointID(graphToolCheckPointID))

	wasInterrupted, hasState, state := tool.GetInterruptState[*graphToolInterruptState](ctx)
	if wasInterrupted && hasState {
		input = state.ToolInput

		checkpointStore = newResumeStore(state.Data)
		compileOptions := make([]compose.GraphCompileOption, len(g.compileOptions)+1)
		copy(compileOptions, g.compileOptions)
		compileOptions[len(g.compileOptions)] = compose.WithCheckPointStore(checkpointStore)

		if runnable, err = g.compilable.Compile(ctx, compileOptions...); err != nil {
			return "", err
		}
	} else {
		checkpointStore = newEmptyStore()

		compileOptions := make([]compose.GraphCompileOption, len(g.compileOptions)+1)
		copy(compileOptions, g.compileOptions)
		compileOptions[len(g.compileOptions)] = compose.WithCheckPointStore(checkpointStore)

		if runnable, err = g.compilable.Compile(ctx, compileOptions...); err != nil {
			return "", err
		}
	}

	inputParams = NewInstance[I]()
	if err = sonic.UnmarshalString(input, &inputParams); err != nil {
		return "", err
	}

	originOutput, err = runnable.Invoke(ctx, inputParams, callOpts...)
	if err != nil {
		_, ok := compose.ExtractInterruptInfo(err)
		if !ok {
			return "", err
		}
		interruptErr := err
		data, existed, getErr := checkpointStore.Get(ctx, graphToolCheckPointID)
		if getErr != nil {
			return "", getErr
		}
		if !existed {
			return "", fmt.Errorf("interrupt has happened, but checkpoint not exist in store")
		}

		return "", tool.CompositeInterrupt(ctx, "graph tool interrupt", &graphToolInterruptState{
			Data:      data,
			ToolInput: input,
		}, interruptErr)
	}

	return sonic.MarshalString(originOutput)
}

func (g *InvokableGraphTool[I, O]) Info(_ context.Context) (*schema.ToolInfo, error) {
	return g.tInfo, nil
}

type StreamableGraphTool[I, O any] struct {
	compilable     Compilable[I, O]
	compileOptions []compose.GraphCompileOption
	tInfo          *schema.ToolInfo
}

func NewStreamableGraphTool[I, O any](compilable Compilable[I, O],
	name, desc string,
	opts ...compose.GraphCompileOption,
) (*StreamableGraphTool[I, O], error) {
	tInfo, err := utils.GoStruct2ToolInfo[I](name, desc)
	if err != nil {
		return nil, err
	}

	return &StreamableGraphTool[I, O]{
		compilable:     compilable,
		compileOptions: opts,
		tInfo:          tInfo,
	}, nil
}

func (g *StreamableGraphTool[I, O]) Info(_ context.Context) (*schema.ToolInfo, error) {
	return g.tInfo, nil
}

func (g *StreamableGraphTool[I, O]) StreamableRun(ctx context.Context, input string,
	opts ...tool.Option) (*schema.StreamReader[string], error) {
	var (
		checkpointStore *graphToolStore
		inputParams     I
		runnable        compose.Runnable[I, O]
		err             error
	)

	callOpts := tool.GetImplSpecificOptions(&graphToolOptions{}, opts...).composeOpts
	callOpts = append(callOpts, compose.WithCheckPointID(graphToolCheckPointID))

	wasInterrupted, hasState, state := tool.GetInterruptState[*graphToolInterruptState](ctx)
	if wasInterrupted && hasState {
		input = state.ToolInput

		checkpointStore = newResumeStore(state.Data)
		compileOptions := make([]compose.GraphCompileOption, len(g.compileOptions)+1)
		copy(compileOptions, g.compileOptions)
		compileOptions[len(g.compileOptions)] = compose.WithCheckPointStore(checkpointStore)

		if runnable, err = g.compilable.Compile(ctx, compileOptions...); err != nil {
			return nil, err
		}
	} else {
		checkpointStore = newEmptyStore()

		compileOptions := make([]compose.GraphCompileOption, len(g.compileOptions)+1)
		copy(compileOptions, g.compileOptions)
		compileOptions[len(g.compileOptions)] = compose.WithCheckPointStore(checkpointStore)

		if runnable, err = g.compilable.Compile(ctx, compileOptions...); err != nil {
			return nil, err
		}
	}

	inputParams = NewInstance[I]()
	if err = sonic.UnmarshalString(input, &inputParams); err != nil {
		return nil, err
	}

	sr, sw := schema.Pipe[string](1)

	go func() {
		defer sw.Close()

		outputStream, err := runnable.Stream(ctx, inputParams, callOpts...)
		if err != nil {
			_, ok := compose.ExtractInterruptInfo(err)
			if !ok {
				sw.Send("", err)
				return
			}
			interruptErr := err
			data, existed, getErr := checkpointStore.Get(ctx, graphToolCheckPointID)
			if getErr != nil {
				sw.Send("", getErr)
				return
			}
			if !existed {
				sw.Send("", fmt.Errorf("interrupt has happened, but checkpoint not exist in store"))
				return
			}

			sw.Send("", tool.CompositeInterrupt(ctx, "graph tool interrupt", &graphToolInterruptState{
				Data:      data,
				ToolInput: input,
			}, interruptErr))
			return
		}

		defer outputStream.Close()

		for {
			chunk, err := outputStream.Recv()
			if err != nil {
				if err == io.EOF {
					break
				}
				_, ok := compose.ExtractInterruptInfo(err)
				if !ok {
					sw.Send("", err)
					return
				}
				interruptErr := err
				data, existed, getErr := checkpointStore.Get(ctx, graphToolCheckPointID)
				if getErr != nil {
					sw.Send("", getErr)
					return
				}
				if !existed {
					sw.Send("", fmt.Errorf("interrupt has happened, but checkpoint not exist in store"))
					return
				}

				sw.Send("", tool.CompositeInterrupt(ctx, "graph tool interrupt", &graphToolInterruptState{
					Data:      data,
					ToolInput: input,
				}, interruptErr))
				return
			}

			chunkStr, err := sonic.MarshalString(chunk)
			if err != nil {
				sw.Send("", err)
				return
			}
			if closed := sw.Send(chunkStr, nil); closed {
				return
			}
		}
	}()

	return sr, nil
}

const graphToolCheckPointID = "graph_tool_checkpoint_id"

func newEmptyStore() *graphToolStore {
	return &graphToolStore{}
}

func newResumeStore(data []byte) *graphToolStore {
	return &graphToolStore{
		Data:  data,
		Valid: true,
	}
}

type graphToolStore struct {
	Data  []byte
	Valid bool
}

func (m *graphToolStore) Get(_ context.Context, _ string) ([]byte, bool, error) {
	if m.Valid {
		return m.Data, true, nil
	}
	return nil, false, nil
}

func (m *graphToolStore) Set(_ context.Context, _ string, checkPoint []byte) error {
	m.Data = checkPoint
	m.Valid = true
	return nil
}

func NewInstance[T any]() T {
	typ := TypeOf[T]()

	switch typ.Kind() {
	case reflect.Map:
		return reflect.MakeMap(typ).Interface().(T)
	case reflect.Slice, reflect.Array:
		return reflect.MakeSlice(typ, 0, 0).Interface().(T)
	case reflect.Ptr:
		typ = typ.Elem()
		origin := reflect.New(typ)
		inst := origin

		for typ.Kind() == reflect.Ptr {
			typ = typ.Elem()
			inst = inst.Elem()
			inst.Set(reflect.New(typ))
		}

		return origin.Interface().(T)
	default:
		var t T
		return t
	}
}

func TypeOf[T any]() reflect.Type {
	return reflect.TypeOf((*T)(nil)).Elem()
}
