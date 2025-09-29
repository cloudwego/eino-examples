package agents

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/cloudwego/eino-examples/adk/multiagent/integration-excel-agent/generic"
	"github.com/cloudwego/eino-examples/adk/multiagent/integration-excel-agent/params"
	"github.com/cloudwego/eino-examples/adk/multiagent/integration-excel-agent/tools"
	"github.com/cloudwego/eino-examples/adk/multiagent/integration-excel-agent/utils"
	"github.com/cloudwego/eino/components"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/jinzhu/copier"
)

const defaultRealMaxStep = 60 // 在newMessageModifier中使用，提前结束ReAct Agent

func NewChatModelWithHistoryModifier(agentName string, model model.ToolCallingChatModel) model.ToolCallingChatModel {
	return &chatModelWithHistoryModifier{
		agentName:            agentName,
		ToolCallingChatModel: model,
	}
}

type chatModelWithHistoryModifier struct {
	agentName string
	model.ToolCallingChatModel
}

func (m *chatModelWithHistoryModifier) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	cm, err := m.ToolCallingChatModel.WithTools(tools)
	if err != nil {
		return nil, err
	}
	return NewChatModelWithHistoryModifier(m.agentName, cm), nil
}

func (m *chatModelWithHistoryModifier) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	input = modifyChatHistory(ctx, m.agentName, input)
	return m.ToolCallingChatModel.Generate(ctx, input, opts...)
}

func (m *chatModelWithHistoryModifier) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	input = modifyChatHistory(ctx, m.agentName, input)
	return m.ToolCallingChatModel.Stream(ctx, input, opts...)
}

func (m *chatModelWithHistoryModifier) IsCallbacksEnabled() bool {
	if cm, ok := m.ToolCallingChatModel.(components.Checker); ok {
		return cm.IsCallbacksEnabled()
	}
	return false
}

func (m *chatModelWithHistoryModifier) GetType() string {
	return "ark"
}

func modifyChatHistory(ctx context.Context, agentName string, input []*schema.Message) []*schema.Message {
	if len(input) <= 2 {
		return input
	}
	if input[0].Role != schema.System {
		return input
	}
	return newMessageModifier(ctx, agentName, input[0], nil, 0)(ctx, input[1:])
}

func newMessageModifier(ctx context.Context, agentName string, sp *schema.Message, spExtraFields []string, maxStep int) func(ctx context.Context, input []*schema.Message) []*schema.Message {
	return func(ctx context.Context, input []*schema.Message) []*schema.Message {
		res := make([]*schema.Message, 0, len(input))
		newSp := &schema.Message{}
		if err := copier.Copy(newSp, sp); err != nil {
			log.Fatalf("[NewReActAgentConfig] Copy failed, msg: %v", err)
		}

		res = append(res, modifySpByExtraFields(ctx, newSp, spExtraFields))
		if len(input) != 0 {
			// input 去重
			filterInput := make([]*schema.Message, 0, len(input))
			filterInput = append(filterInput, input[0])
			for i := 1; i < len(input); i++ {
				if input[i].Role == input[i-1].Role && input[i].Content == input[i-1].Content {
					continue
				} else {
					filterInput = append(filterInput, input[i])
				}
			}
			log.Printf("[newMessageModifier] input: %s, filterInput: %s", utils.ToJSONString(input), utils.ToJSONString(filterInput))

			// 去除 input 的第一个元素，第一个元素为上一个 Agent 的 Resp，有可能为 ToolCall 的结果
			if filterInput[0].Role == schema.Tool {
				res = append(res, filterInput[1:]...)
			} else {
				res = append(res, filterInput...)
			}
		}

		realMaxStep := defaultRealMaxStep
		if len(input) >= realMaxStep-1 {
			info, _ := tools.NewSubmitResultTool().Info(ctx)
			res = append(res, schema.UserMessage(fmt.Sprintf("当前迭代次数已达最大值，直接调用%s工具输出目前的执行结果", info.Name)))
		}

		// planner 兜底 fc 问题
		if agentName == AgentNamePlanner && len(res) != 0 {
			lastMsg := res[len(res)-1]
			_, isErr := isFunctionCallErrorFormat(ctx, lastMsg.Content)
			if isErr {
				res = append(res, schema.UserMessage("FunctionCall调用格式错误，必须使用`<|FunctionCallBegin|>`与`<|FunctionCallEnd|>`包裹"))
			}
		}

		up := ""
		for i, msg := range res {
			up += fmt.Sprintf("[%d][role:%s][content:%s]\n", i, msg.Role, msg.Content)
		}
		log.Printf("[MessageModifier] %s, UserPrompt: %s", agentName, up)

		return res

		// TODO: compress
		// output := newExecutorCheckMsgCompressProcess(ctx, res)
		// logs.CtxInfo(ctx, "[MessageModifier] agent: %s, Output: %s", agentName, utils.ToJSONString(output))
		// return output
	}
}

func modifySpByExtraFields(ctx context.Context, sp *schema.Message, spExtraFields []string) *schema.Message {
	sp.Content = fmt.Sprintf("%s\nCurrent time is：%s", sp.Content, utils.GetCurrentTime())
	if len(spExtraFields) == 0 {
		return sp
	}
	for _, field := range spExtraFields {
		value := params.MustGetBizContextParams(ctx, field)
		sp.Content = strings.ReplaceAll(sp.Content, fmt.Sprintf("{#%s#}", field), value)
	}
	// logs.CtxInfo(ctx, "[modifySpByExtraFields] replaced sp: %v", sp.Content)
	return sp
}

func isFunctionCallErrorFormat(ctx context.Context, content string) (*generic.FunctionCallParam, bool) {
	// 按四种情况清理前后缀
	content = strings.TrimSpace(content)
	if !strings.Contains(content, "<|FunctionCallBegin|>") &&
		!strings.Contains(content, "<|FunctionCallEnd|>") && !strings.Contains(content, "<think>") &&
		!strings.Contains(content, "</think>") {
		return nil, false
	}

	content = strings.TrimPrefix(content, "<|FunctionCallBegin|>")
	content = strings.TrimSuffix(content, "<|FunctionCallEnd|>")
	content = strings.TrimPrefix(content, "<think>")

	// 如果包含 </think>，只保留最后一个 </think> 之后的部分
	if idx := strings.LastIndex(content, "</think>"); idx != -1 {
		content = content[idx+len("</think>"):]
	}

	content = strings.TrimSpace(content)

	// 尝试解析 JSON
	var arr []*generic.FunctionCallParam
	if err := sonic.UnmarshalString(content, &arr); err != nil {
		return nil, false
	}
	if len(arr) == 0 {
		return nil, false
	}
	return arr[0], true
}
