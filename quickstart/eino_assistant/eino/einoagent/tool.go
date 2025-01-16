package eino_agent

import (
	"context"

	"github.com/cloudwego/eino-ext/components/tool/duckduckgo"
	"github.com/cloudwego/eino-ext/components/tool/duckduckgo/ddgsearch"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

func defaultDuckDuckGoToolConfig(ctx context.Context) (*duckduckgo.Config, error) {
	config := &duckduckgo.Config{
		DDGConfig: &ddgsearch.Config{
			Headers: map[string]string{}}}
	return config, nil
}

func NewDuckDuckGoTool(ctx context.Context, config *duckduckgo.Config) (bt tool.BaseTool, err error) {
	if config == nil {
		config, err = defaultDuckDuckGoToolConfig(ctx)
		if err != nil {
			return nil, err
		}
	}
	bt, err = duckduckgo.NewTool(ctx, config)
	if err != nil {
		return nil, err
	}
	return bt, nil
}

type TaskToolImpl struct {
	config *TaskToolConfig
}

type TaskToolConfig struct {
}

func defaultTaskToolConfig(ctx context.Context) (*TaskToolConfig, error) {
	config := &TaskToolConfig{}
	return config, nil
}

func NewTaskTool(ctx context.Context, config *TaskToolConfig) (bt tool.BaseTool, err error) {
	if config == nil {
		config, err = defaultTaskToolConfig(ctx)
		if err != nil {
			return nil, err
		}
	}
	bt = &TaskToolImpl{config: config}
	return bt, nil
}

func (impl *TaskToolImpl) Info(ctx context.Context) (*schema.ToolInfo, error) {
	panic("implement me")
}

func (impl *TaskToolImpl) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	panic("implement me")
}

type EinoToolImpl struct {
	config *EinoToolConfig
}

type EinoToolConfig struct {
}

func defaultEinoToolConfig(ctx context.Context) (*EinoToolConfig, error) {
	config := &EinoToolConfig{}
	return config, nil
}

func NewEinoTool(ctx context.Context, config *EinoToolConfig) (bt tool.BaseTool, err error) {
	if config == nil {
		config, err = defaultEinoToolConfig(ctx)
		if err != nil {
			return nil, err
		}
	}
	bt = &EinoToolImpl{config: config}
	return bt, nil
}

func (impl *EinoToolImpl) Info(ctx context.Context) (*schema.ToolInfo, error) {
	panic("implement me")
}

func (impl *EinoToolImpl) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	panic("implement me")
}

type OpenURIToolImpl struct {
	config *OpenURIToolConfig
}

type OpenURIToolConfig struct {
}

func defaultOpenURIToolConfig(ctx context.Context) (*OpenURIToolConfig, error) {
	config := &OpenURIToolConfig{}
	return config, nil
}

func NewOpenURITool(ctx context.Context, config *OpenURIToolConfig) (bt tool.BaseTool, err error) {
	if config == nil {
		config, err = defaultOpenURIToolConfig(ctx)
		if err != nil {
			return nil, err
		}
	}
	bt = &OpenURIToolImpl{config: config}
	return bt, nil
}

func (impl *OpenURIToolImpl) Info(ctx context.Context) (*schema.ToolInfo, error) {
	panic("implement me")
}

func (impl *OpenURIToolImpl) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	panic("implement me")
}

type GitCloneToolImpl struct {
	config *GitCloneToolConfig
}

type GitCloneToolConfig struct {
}

func defaultGitCloneToolConfig(ctx context.Context) (*GitCloneToolConfig, error) {
	config := &GitCloneToolConfig{}
	return config, nil
}

func NewGitCloneTool(ctx context.Context, config *GitCloneToolConfig) (bt tool.BaseTool, err error) {
	if config == nil {
		config, err = defaultGitCloneToolConfig(ctx)
		if err != nil {
			return nil, err
		}
	}
	bt = &GitCloneToolImpl{config: config}
	return bt, nil
}

func (impl *GitCloneToolImpl) Info(ctx context.Context) (*schema.ToolInfo, error) {
	panic("implement me")
}

func (impl *GitCloneToolImpl) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	panic("implement me")
}
