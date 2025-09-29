package params

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino-examples/adk/multiagent/integration-excel-agent/generic"
	"github.com/cloudwego/eino-examples/adk/multiagent/integration-excel-agent/utils"
)

func SetWorkDir(ctx context.Context, workDir string) {
	AppendContextParams(ctx, CustomBizKey, map[string]interface{}{
		"WorkDir": workDir,
	})
}

func GetWorkDir(ctx context.Context) string {
	return MustGetContextParams[string](ctx, CustomBizKey, "WorkDir")
}

func SetPlan(ctx context.Context, plan []*generic.FullPlan) {
	AppendContextParams(ctx, CustomBizKey, map[string]interface{}{
		"Plan": plan,
	})
}

func GetPlan(ctx context.Context) []*generic.FullPlan {
	return MustGetContextParams[[]*generic.FullPlan](ctx, CustomBizKey, "Plan")
}

func SetPlanMD(ctx context.Context, planMD string) {
	AppendContextParams(ctx, CustomBizKey, map[string]interface{}{
		"PlanMD": planMD,
	})
}

func GetPlanMD(ctx context.Context) string {
	return MustGetContextParams[string](ctx, CustomBizKey, "PlanMD")
}

func SetCurrentStep(ctx context.Context, currentStep int) {
	AppendContextParams(ctx, CustomBizKey, map[string]interface{}{
		"CurrentStep": currentStep,
	})
}

func GetCurrentStep(ctx context.Context) int {
	return MustGetContextParams[int](ctx, CustomBizKey, "CurrentStep")
}

func SetCurrentTaskID(ctx context.Context, currentTaskID int) {
	AppendContextParams(ctx, CustomBizKey, map[string]interface{}{
		"CurrentTaskID": currentTaskID,
	})
}

func GetCurrentTaskID(ctx context.Context) int {
	return MustGetContextParams[int](ctx, CustomBizKey, "CurrentTaskID")
}

func SetCurrentPlan(ctx context.Context, currentPlan *generic.FullPlan) {
	AppendContextParams(ctx, CustomBizKey, map[string]interface{}{
		"CurrentPlan": currentPlan,
	})
}

func GetCurrentPlan(ctx context.Context) *generic.FullPlan {
	return MustGetContextParams[*generic.FullPlan](ctx, CustomBizKey, "CurrentPlan")
}

func MustGetBizContextParams(ctx context.Context, key string) string {
	switch key {
	case CtxParamPathUrlMapKey:
		return GetFormattedPathUriContextParams(ctx)
	default:
		v, ok := GetContextParams(ctx, CustomBizKey, key)
		if !ok {
			return ""
		}
		switch v := v.(type) {
		case string:
			return v
		default:
			return utils.ToJSONString(v)
		}
	}
}

const CtxParamPathUrlMapKey = "PathUrlMap"

func GetPathUriContextParams(ctx context.Context) map[string]string {
	m, ok := GetContextParams(ctx, CustomBizKey, CtxParamPathUrlMapKey)
	if !ok {
		return nil
	}
	pathUrlMap, ok := m.(map[string]string)
	if !ok {
		return nil
	}
	return pathUrlMap
}

func GetFormattedPathUriContextParams(ctx context.Context) string {
	pathUrlMap := GetPathUriContextParams(ctx)
	var output string
	for k, v := range pathUrlMap {
		output += fmt.Sprintf("图片名称:%s,url:%s\n", k, v)
	}
	return output
}

func ResetPlanAndStep(ctx context.Context, plan []*generic.FullPlan, currentStep int) {
	SetPlan(ctx, plan)
	SetPlanMD(ctx, generic.FullPlan2String(plan))
	SetCurrentStep(ctx, currentStep)
	if currentStep >= len(plan) {
		return
	}
	SetCurrentPlan(ctx, plan[currentStep])
	SetCurrentTaskID(ctx, plan[currentStep].TaskID)
}

func SetCozeSpaceTaskID(ctx context.Context, cozeSpaceTaskID string) {
	AppendContextParams(ctx, CustomBizKey, map[string]interface{}{
		"CozeSpaceTaskID": cozeSpaceTaskID,
	})
}

func GetCozeSpaceTaskID(ctx context.Context) string {
	return MustGetContextParams[string](ctx, CustomBizKey, "CozeSpaceTaskID")
}

func SetCurrentAgentResult(ctx context.Context, submitResult *generic.SubmitResult) {
	AppendContextParams(ctx, CustomBizKey, map[string]interface{}{
		"CurrentAgentResult": submitResult,
	})
}
