package einoagent

import (
	"context"

	"github.com/cloudwego/eino-ext/components/retriever/redis"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
)

type EinoAgentBuildConfig struct {
	ChatTemplateKeyOfChatTemplate *ChatTemplateConfig
	ReactAgentKeyOfLambda         *react.AgentConfig
	RedisRetrieverKeyOfRetriever  *redis.RetrieverConfig
}

type BuildConfig struct {
	EinoAgent *EinoAgentBuildConfig
}

func BuildEinoAgent(ctx context.Context, config *BuildConfig) (r compose.Runnable[*UserMessage, *schema.Message], err error) {
	const (
		InputToQuery   = "InputToQuery"
		ChatTemplate   = "ChatTemplate"
		ReactAgent     = "ReactAgent"
		RedisRetriever = "RedisRetriever"
		InputToHistory = "InputToHistory"
	)
	g := compose.NewGraph[*UserMessage, *schema.Message]()
	_ = g.AddLambdaNode(InputToQuery, compose.InvokableLambdaWithOption(NewInputToQuery))
	chatTemplateKeyOfChatTemplate, err := NewChatTemplate(ctx, config.EinoAgent.ChatTemplateKeyOfChatTemplate)
	if err != nil {
		return nil, err
	}
	_ = g.AddChatTemplateNode(ChatTemplate, chatTemplateKeyOfChatTemplate)
	reactAgentKeyOfLambda, err := NewReactAgent(ctx, config.EinoAgent.ReactAgentKeyOfLambda)
	if err != nil {
		return nil, err
	}
	_ = g.AddLambdaNode(ReactAgent, reactAgentKeyOfLambda)
	redisRetrieverKeyOfRetriever, err := NewRedisRetriever(ctx, config.EinoAgent.RedisRetrieverKeyOfRetriever)
	if err != nil {
		return nil, err
	}
	_ = g.AddRetrieverNode(RedisRetriever, redisRetrieverKeyOfRetriever, compose.WithOutputKey("documents"))
	_ = g.AddLambdaNode(InputToHistory, compose.InvokableLambdaWithOption(NewInputToHistory))
	_ = g.AddEdge(compose.START, InputToQuery)
	_ = g.AddEdge(compose.START, InputToHistory)
	_ = g.AddEdge(ReactAgent, compose.END)
	_ = g.AddEdge(InputToQuery, RedisRetriever)
	_ = g.AddEdge(RedisRetriever, ChatTemplate)
	_ = g.AddEdge(InputToHistory, ChatTemplate)
	_ = g.AddEdge(ChatTemplate, ReactAgent)
	r, err = g.Compile(ctx, compose.WithGraphName("EinoAgent"), compose.WithNodeTriggerMode(compose.AllPredecessor))
	if err != nil {
		return nil, err
	}
	return r, err
}
