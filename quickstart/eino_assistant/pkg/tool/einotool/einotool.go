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

package einotool

import (
	"context"
	"embed"
	"os"
	"path/filepath"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

//go:embed templates/*
var templateFS embed.FS

const desc = `eino tool can get eino project info, 
action:
- get_example_project: get the example project url, path of eino-examples
- get_github_repo: get the github repo url, e.g. eino, eino-ext, eino-examples
- get_doc_url: get the doc url of eino website
- init_template: init the eino project template, to create files from template
`

type EinoAssistantToolImpl struct {
	config *EinoAssistantToolConfig
}

type EinoAssistantToolConfig struct {
	BaseDir string
}

func defaultEinoAssistantToolConfig(ctx context.Context) (*EinoAssistantToolConfig, error) {
	config := &EinoAssistantToolConfig{
		BaseDir: "./data/eino",
	}
	return config, nil
}

func NewEinoAssistantTool(ctx context.Context, config *EinoAssistantToolConfig) (tn tool.BaseTool, err error) {
	if config == nil {
		config, err = defaultEinoAssistantToolConfig(ctx)
		if err != nil {
			return nil, err
		}
	}
	t := &EinoAssistantToolImpl{config: config}
	tn, err = t.ToEinoTool()
	if err != nil {
		return nil, err
	}
	return tn, nil
}

var (
	EinoRepo = map[string]string{
		"eino":          "https://github.com/cloudwego/eino",
		"eino-ext":      "https://github.com/cloudwego/eino-ext",
		"eino-examples": "https://github.com/cloudwego/eino-examples",
	}

	EinoDoc = map[string]string{
		"eino_index": "https://www.cloudwego.io/zh/docs/eino/",
		"quickstart": "https://www.cloudwego.io/zh/docs/eino/quick_start/",
		"graph":      "https://www.cloudwego.io/zh/docs/eino/core_modules/chain_and_graph_orchestration/",
		"agent":      "https://www.cloudwego.io/zh/docs/eino/core_modules/flow_integration_components/",
		"components": "https://www.cloudwego.io/zh/docs/eino/core_modules/components/",
		"integrate":  "https://www.cloudwego.io/zh/docs/eino/ecosystem_integration/",
	}

	EinoExample = map[string][]string{
		"agent":      {"https://github.com/cloudwego/eino-examples/tree/main/flow/agent/react"},
		"components": {"https://github.com/cloudwego/eino-examples/tree/main/components"},
		"graph":      {"https://github.com/cloudwego/eino-examples/tree/main/compose/graph/tool_call_agent.go"},
		"quickstart": {"https://github.com/cloudwego/eino-examples/tree/main/quickstart"},
	}

	Template = map[string][]string{
		"react_agent": {"react_agent/main.go"},
		"simple_llm":  {"simple_llm/main.go"},
		"http_agent":  {"http_agent/main.go", "http_agent/README.md", "http_agent/client/main.go"},
	}
)

func (e *EinoAssistantToolImpl) ToEinoTool() (tool.BaseTool, error) {
	return utils.InferTool("eino_tool", desc, e.Invoke)
}

func (e *EinoAssistantToolImpl) Invoke(ctx context.Context, req *EinoToolRequest) (res *EinoToolResponse, err error) {
	res = &EinoToolResponse{}

	switch req.Action {
	case EinoToolActionGetExampleProject:
		exampleURL := EinoExample[req.ExampleType]
		if len(exampleURL) == 0 {
			res.Error = "invalid example type, can be one of: agent, components, graph, quickstart. example repo is " + EinoRepo["eino-examples"]
			return
		}
		res.Message = exampleURL[0]
	case EinoToolActionGetGithubRepo:
		repoURL := EinoRepo[req.RepoType]
		if repoURL == "" {
			res.Error = "invalid repo type, can be one of: eino, eino-ext, eino-examples. eino repo url is " + EinoRepo["eino"]
			return
		}
		res.Message = repoURL
	case EinoToolActionGetDocURL:
		docURL := EinoDoc[req.DocType]
		if docURL == "" {
			res.Error = "invalid doc type, can be one of: eino_index, quickstart, graph, agent, components, integrate. eino doc url is " + EinoDoc["eino_index"]
			return
		}
		res.Message = docURL
	case EinoToolActionInitTemplate:
		templateURL := Template[req.TemplateType]
		if len(templateURL) == 0 {
			res.Error = "invalid template type, can be one of: react_agent, simple_llm, http_agent"
			return res, nil
		}

		baseDir := e.config.BaseDir
		for _, file := range templateURL {
			// Read template file
			content, err := templateFS.ReadFile(filepath.Join("templates", file))
			if err != nil {
				res.Error = "failed to read template file: " + err.Error()
				return res, nil
			}

			// Create target directory
			targetPath := filepath.Join(baseDir, file)
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				res.Error = "failed to create directory: " + err.Error()
				return res, nil
			}

			// Write file
			if err := os.WriteFile(targetPath, content, 0644); err != nil {
				res.Error = "failed to write file: " + err.Error()
				return res, nil
			}
		}
		absPath, err := filepath.Abs(filepath.Join(baseDir, req.TemplateType))
		if err != nil {
			absPath = filepath.Join(baseDir, req.TemplateType)
		}
		res.Message = "success, init template, path is: " + absPath
		return res, nil
	default:
		res.Error = "invalid action, can be one of: get_example_project, get_github_repo, get_doc_url"
	}

	return res, nil
}

type EinoToolAction string

const (
	EinoToolActionGetExampleProject EinoToolAction = "get_example_project" // 获取示例项目
	EinoToolActionGetGithubRepo     EinoToolAction = "get_github_repo"     // 获取 github 仓库
	EinoToolActionGetDocURL         EinoToolAction = "get_doc_url"         // 获取文档地址
	EinoToolActionInitTemplate      EinoToolAction = "init_template"       // 初始化项目模板
)

type EinoToolRequest struct {
	Action       EinoToolAction `json:"action" jsonschema:"description='The action of the request',enum=get_example_project,enum=get_github_repo,enum=get_doc_url,enum=init_template"`
	ExampleType  string         `json:"example_type,omitempty" jsonschema:"description='The type of the example project, only for action: get_example_project',enum=agent,enum=components,enum=graph,enum=quickstart"`
	RepoType     string         `json:"repo_type,omitempty" jsonschema:"description='The type of the repo, only for action: get_github_repo',enum=eino,enum=eino-ext,enum=eino-examples"`
	DocType      string         `json:"doc_type,omitempty" jsonschema:"description='The type of the doc, only for action: get_doc_url',enum=eino_index,enum=quickstart,enum=graph,enum=agent,enum=components,enum=integrate"`
	TemplateType string         `json:"template_type,omitempty" jsonschema:"description='The template of the project, only for action: init_template',enum=react_agent,enum=simple_llm,enum=http_agent"`
}

type EinoToolResponse struct {
	Message string `json:"message" jsonschema:"description=The message of the response"`
	Error   string `json:"error" jsonschema:"description=The error of the response"`
}
