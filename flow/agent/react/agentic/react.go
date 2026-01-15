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
	"io"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

type state struct {
	Messages                 []*schema.AgenticMessage
	ReturnDirectlyToolCallID string
}

func init() {
	schema.RegisterName[*state]("_eino_agentic_react_state")
}

const (
	nodeKeyTools = "tools"
	nodeKeyModel = "agentic_model"
)

// AgentConfig is the config for ReAct agent.
type AgentConfig struct {
	// Model is the agentic model to be used for handling user messages with tool calling capability.
	Model model.AgenticModel

	// ToolsConfig is the config for tools node.
	ToolsConfig compose.ToolsNodeConfig

	// MaxStep.
	// default 12 of steps in pregel (node num + 10).
	MaxStep int `json:"max_step"`

	// Tools that will make agent return directly when the tool is called.
	// When multiple tools are called and more than one tool is in the return directly list, only the first one will be returned.
	ToolReturnDirectly map[string]struct{}
}

// Agent is the ReAct agent.
type Agent struct {
	runnable         compose.Runnable[[]*schema.AgenticMessage, *schema.AgenticMessage]
	graph            *compose.Graph[[]*schema.AgenticMessage, *schema.AgenticMessage]
	graphAddNodeOpts []compose.GraphAddNodeOpt
}

// NewAgent creates a ReAct agent that feeds tool response into next round of Chat Model generation.
func NewAgent(ctx context.Context, config *AgentConfig) (_ *Agent, err error) {
	var (
		toolsNode *compose.AgenticToolsNode
		toolInfos []*schema.ToolInfo
	)

	if toolInfos, err = genToolInfos(ctx, config.ToolsConfig); err != nil {
		return nil, err
	}

	agenticModel, err := config.Model.WithTools(toolInfos)
	if err != nil {
		return nil, err
	}

	if toolsNode, err = compose.NewAgenticToolsNode(ctx, &config.ToolsConfig); err != nil {
		return nil, err
	}

	graph := compose.NewGraph[[]*schema.AgenticMessage, *schema.AgenticMessage](compose.WithGenLocalState(func(ctx context.Context) *state {
		return &state{Messages: make([]*schema.AgenticMessage, 0, config.MaxStep+1)}
	}))

	modelPreHandle := func(ctx context.Context, input []*schema.AgenticMessage, state *state) ([]*schema.AgenticMessage, error) {
		state.Messages = append(state.Messages, input...)

		modifiedInput := make([]*schema.AgenticMessage, len(state.Messages))
		copy(modifiedInput, state.Messages)

		return modifiedInput, nil
	}

	_ = graph.AddAgenticModelNode(nodeKeyModel, agenticModel,
		compose.WithStatePreHandler(modelPreHandle),
		compose.WithNodeName("News Assistant"),
	)

	_ = graph.AddEdge(compose.START, nodeKeyModel)

	toolsNodePreHandle := func(ctx context.Context, input *schema.AgenticMessage, state *state) (*schema.AgenticMessage, error) {
		if input == nil {
			return state.Messages[len(state.Messages)-1], nil // used for rerun interrupt resume
		}
		state.Messages = append(state.Messages, input)
		state.ReturnDirectlyToolCallID = getReturnDirectlyToolCallID(input, config.ToolReturnDirectly)
		return input, nil
	}
	_ = graph.AddAgenticToolsNode(nodeKeyTools, toolsNode,
		compose.WithStatePreHandler(toolsNodePreHandle),
		compose.WithNodeName("Tools"),
	)

	modelPostBranchCondition := func(ctx context.Context, sr *schema.StreamReader[*schema.AgenticMessage]) (endNode string, err error) {
		if isToolCall, err := firstChunkStreamToolCallChecker(ctx, sr); err != nil {
			return "", err
		} else if isToolCall {
			return nodeKeyTools, nil
		}
		return compose.END, nil
	}

	_ = graph.AddBranch(nodeKeyModel, compose.NewStreamGraphBranch(modelPostBranchCondition,
		map[string]bool{
			nodeKeyTools: true,
			compose.END:  true,
		},
	))

	if err = buildReturnDirectly(graph); err != nil {
		return nil, err
	}

	compileOpts := []compose.GraphCompileOption{
		compose.WithMaxRunSteps(config.MaxStep),
		compose.WithNodeTriggerMode(compose.AnyPredecessor),
	}
	runnable, err := graph.Compile(ctx, compileOpts...)
	if err != nil {
		return nil, err
	}

	return &Agent{
		runnable:         runnable,
		graph:            graph,
		graphAddNodeOpts: []compose.GraphAddNodeOpt{compose.WithGraphCompileOptions(compileOpts...)},
	}, nil
}

func firstChunkStreamToolCallChecker(_ context.Context, sr *schema.StreamReader[*schema.AgenticMessage]) (bool, error) {
	defer sr.Close()

	for {
		msg, err := sr.Recv()
		if err == io.EOF {
			return false, nil
		}
		if err != nil {
			return false, err
		}

		for _, block := range msg.ContentBlocks {
			if block.Type == schema.ContentBlockTypeFunctionToolCall {
				return true, nil
			}
		}
	}
}

func buildReturnDirectly(graph *compose.Graph[[]*schema.AgenticMessage, *schema.AgenticMessage]) (err error) {
	directReturn := func(ctx context.Context, msgs *schema.StreamReader[[]*schema.AgenticMessage]) (*schema.StreamReader[*schema.AgenticMessage], error) {
		return schema.StreamReaderWithConvert(msgs, func(msgs []*schema.AgenticMessage) (*schema.AgenticMessage, error) {
			var msg *schema.AgenticMessage
			err = compose.ProcessState[*state](ctx, func(_ context.Context, state *state) error {
				for i := range msgs {
					msg_ := msgs[i]
					for _, block := range msg_.ContentBlocks {
						if block.Type == schema.ContentBlockTypeFunctionToolResult &&
							block.FunctionToolResult.CallID == state.ReturnDirectlyToolCallID {
							msg = msg_
							return nil
						}
					}
				}
				return nil
			})
			if err != nil {
				return nil, err
			}
			if msg == nil {
				return nil, schema.ErrNoValue
			}
			return msg, nil
		}), nil
	}

	nodeKeyDirectReturn := "direct_return"
	if err = graph.AddLambdaNode(nodeKeyDirectReturn, compose.TransformableLambda(directReturn)); err != nil {
		return err
	}

	// this branch checks if the tool called should return directly. It either leads to END or back to AgenticModel
	err = graph.AddBranch(nodeKeyTools, compose.NewStreamGraphBranch(func(ctx context.Context, msgsStream *schema.StreamReader[[]*schema.AgenticMessage]) (endNode string, err error) {
		msgsStream.Close()

		err = compose.ProcessState[*state](ctx, func(_ context.Context, state *state) error {
			if len(state.ReturnDirectlyToolCallID) > 0 {
				endNode = nodeKeyDirectReturn
			} else {
				endNode = nodeKeyModel
			}
			return nil
		})
		if err != nil {
			return "", err
		}
		return endNode, nil
	}, map[string]bool{nodeKeyModel: true, nodeKeyDirectReturn: true}))
	if err != nil {
		return err
	}

	return graph.AddEdge(nodeKeyDirectReturn, compose.END)
}

func genToolInfos(ctx context.Context, config compose.ToolsNodeConfig) ([]*schema.ToolInfo, error) {
	toolInfos := make([]*schema.ToolInfo, 0, len(config.Tools))
	for _, t := range config.Tools {
		tl, err := t.Info(ctx)
		if err != nil {
			return nil, err
		}

		toolInfos = append(toolInfos, tl)
	}

	return toolInfos, nil
}

func getReturnDirectlyToolCallID(input *schema.AgenticMessage, toolReturnDirectly map[string]struct{}) string {
	if len(toolReturnDirectly) == 0 {
		return ""
	}

	for _, block := range input.ContentBlocks {
		if block.Type != schema.ContentBlockTypeFunctionToolCall {
			continue
		}
		if _, ok := toolReturnDirectly[block.FunctionToolCall.Name]; ok {
			return block.FunctionToolCall.CallID
		}
	}

	return ""
}

type AgentOption struct {
	composeOptions []compose.Option
}

func GetComposeOptions(opts ...AgentOption) []compose.Option {
	var result []compose.Option
	for _, opt := range opts {
		result = append(result, opt.composeOptions...)
	}

	return result
}

func WithComposeOptions(opts ...compose.Option) AgentOption {
	return AgentOption{
		composeOptions: opts,
	}
}

// Generate generates a response from the agent.
func (r *Agent) Generate(ctx context.Context, input []*schema.AgenticMessage, opts ...AgentOption) (*schema.AgenticMessage, error) {
	return r.runnable.Invoke(ctx, input, GetComposeOptions(opts...)...)
}

// Stream calls the agent and returns a stream response.
func (r *Agent) Stream(ctx context.Context, input []*schema.AgenticMessage, opts ...AgentOption) (output *schema.StreamReader[*schema.AgenticMessage], err error) {
	return r.runnable.Stream(ctx, input, GetComposeOptions(opts...)...)
}

// ExportGraph exports the underlying graph from Agent, along with the []compose.GraphAddNodeOpt to be used when adding this graph to another graph.
func (r *Agent) ExportGraph() (compose.AnyGraph, []compose.GraphAddNodeOpt) {
	return r.graph, r.graphAddNodeOpts
}
