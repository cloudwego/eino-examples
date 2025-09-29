package main

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/cloudwego/eino-examples/adk/common/prints"
	"github.com/cloudwego/eino-examples/adk/multiagent/integration-excel-agent/agents/executor"
	"github.com/cloudwego/eino-examples/adk/multiagent/integration-excel-agent/agents/planner"
	"github.com/cloudwego/eino-examples/adk/multiagent/integration-excel-agent/agents/replanner"
	"github.com/cloudwego/eino-examples/adk/multiagent/integration-excel-agent/agents/report"
	"github.com/cloudwego/eino-examples/adk/multiagent/integration-excel-agent/generic"
	"github.com/cloudwego/eino-examples/adk/multiagent/integration-excel-agent/params"
	"github.com/cloudwego/eino-examples/adk/multiagent/integration-excel-agent/utils"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
)

func main() {
	// query := schema.UserMessage("统计附件文件中推荐的小说名称及推荐次数，凡是带有《》内容都是小说名称，形成表格，表头为小说名称和推荐次数，同名小说只列一行，推荐次数相加"),
	// query := schema.UserMessage("读取工作目录下 模拟出题.csv 中的表格内容，规范格式将题目、答案、解析、选项放在同一行，简答题只把答案写入解析即可"),
	query := schema.UserMessage("请帮我将 question.csv 表格中的第一列提取到一个新的 csv 中")

	ctx := context.Background()
	agent, err := newExcelAgent(ctx)
	if err != nil {
		log.Fatal(err)
	}

	uuid := uuid.New().String()
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
	})

	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	var inputFileDir, workdir string
	if env := os.Getenv("EXCEL_AGENT_INPUT_DIR"); env != "" {
		inputFileDir = env
	} else {
		inputFileDir = filepath.Join(wd, "adk/multiagent/integration-excel-agent/playground/input")
	}

	if env := os.Getenv("EXCEL_AGENT_WORK_DIR"); env != "" {
		workdir = filepath.Join(env, uuid)
	} else {
		workdir = filepath.Join(wd, "adk/multiagent/integration-excel-agent/playground", uuid)
	}

	if err = os.Mkdir(workdir, 0755); err != nil {
		log.Fatal(err)
	}

	if err = os.CopyFS(workdir, os.DirFS(inputFileDir)); err != nil {
		log.Fatal(err)
	}

	previews, err := generic.PreviewPath(workdir)
	if err != nil {
		log.Fatal(err)
	}

	ctx = params.InitContextParams(ctx)
	params.AppendContextParams(ctx, map[string]interface{}{
		params.FilePathSessionKey:            inputFileDir,
		params.WorkDirSessionKey:             workdir,
		params.UserAllPreviewFilesSessionKey: utils.ToJSONString(previews),
		params.TaskIDKey:                     uuid,
	})

	iter := runner.Run(ctx, []*schema.Message{query})

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			panic(event.Err)
		}
		prints.Event(event)
	}
}

func newExcelAgent(ctx context.Context) (adk.Agent, error) {
	operator := &LocalOperator{}

	p, err := planner.NewPlanner(ctx, operator)
	if err != nil {
		return nil, err
	}

	e, err := executor.NewExecutor(ctx, operator)
	if err != nil {
		return nil, err
	}

	rp, err := replanner.NewReplanner(ctx, operator)
	if err != nil {
		return nil, err
	}

	planExecuteAgent, err := planexecute.New(ctx, &planexecute.Config{
		Planner:       p,
		Executor:      e,
		Replanner:     rp,
		MaxIterations: 20,
	})
	if err != nil {
		return nil, err
	}

	reportAgent, err := report.NewReportAgent(ctx, operator)
	if err != nil {
		return nil, err
	}

	agent, err := adk.NewSequentialAgent(ctx, &adk.SequentialAgentConfig{
		Name:        "SequentialAgent",
		Description: "sequential agent",
		SubAgents: []adk.Agent{
			planExecuteAgent, reportAgent,
		},
	})
	if err != nil {
		return nil, err
	}

	return agent, nil
}
