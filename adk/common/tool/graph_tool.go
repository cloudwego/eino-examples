package tool

import (
	"context"
	"fmt"
	"reflect"

	"github.com/bytedance/sonic"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

type InvokableGraphTool[I, O any] struct {
	graph          compose.Graph[I, O]
	compileOptions []compose.GraphCompileOption
	tInfo          *schema.ToolInfo
}

func NewInvokableGraphTool[I, O any](graph compose.Graph[I, O],
	name, desc string,
	opts ...compose.GraphCompileOption,
) (*InvokableGraphTool[I, O], error) {
	tInfo, err := utils.GoStruct2ToolInfo[I](name, desc)
	if err != nil {
		return nil, err
	}

	return &InvokableGraphTool[I, O]{
		graph:          graph,
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

func (g *InvokableGraphTool[I, O]) InvokableRun(ctx context.Context, input string,
	opts ...tool.Option) (output string, err error) {
	var (
		checkpointStore *graphToolStore
		inputParams     I
		originOutput    O
		runnable        compose.Runnable[I, O]
	)

	compileOptions := make([]compose.GraphCompileOption, len(g.compileOptions)+1)
	copy(compileOptions, g.compileOptions)
	compileOptions[len(g.compileOptions)] = compose.WithCheckPointStore(checkpointStore)

	callOpts := tool.GetImplSpecificOptions(&graphToolOptions{}, opts...).composeOpts
	callOpts = append(callOpts, compose.WithCheckPointID(graphToolCheckPointID))

	wasInterrupted, hasState, state := compose.GetInterruptState[[]byte](ctx)
	if !wasInterrupted {
		checkpointStore = newEmptyStore()

		if runnable, err = g.graph.Compile(ctx, compileOptions...); err != nil {
			return "", err
		}

		inputParams = NewInstance[I]()
		if err = sonic.UnmarshalString(input, &inputParams); err != nil {
			return "", err
		}
	} else {
		if !hasState {
			return "", fmt.Errorf("graph tool interrupt has happened, but cannot find interrupt state")
		}

		checkpointStore = newResumeStore(state)
		if runnable, err = g.graph.Compile(ctx, compileOptions...); err != nil {
			return "", err
		}
	}

	originOutput, err = runnable.Invoke(ctx, inputParams, callOpts...)
	if err != nil {
		_, ok := compose.ExtractInterruptInfo(err)
		if !ok {
			return "", err
		}
		data, existed, err := checkpointStore.Get(ctx, graphToolCheckPointID)
		if err != nil {
			return "", err
		}
		if !existed {
			return "", fmt.Errorf("interrupt has happened, but checkpoint not exist in store")
		}

		return "", compose.CompositeInterrupt(ctx, "graph tool interrupt", data,
			err)
	}

	return sonic.MarshalString(originOutput)
}

func (g *InvokableGraphTool[I, O]) Info(_ context.Context) (*schema.ToolInfo, error) {
	return g.tInfo, nil
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
