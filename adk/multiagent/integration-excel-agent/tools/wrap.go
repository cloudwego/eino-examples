package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino-examples/adk/multiagent/integration-excel-agent/consts"
	"github.com/cloudwego/eino-examples/adk/multiagent/integration-excel-agent/utils"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	jsoniter "github.com/json-iterator/go"
)

type ToolRequestPreprocess func(ctx context.Context, baseTool tool.InvokableTool, toolArguments string) (string, error)

type ToolResponsePostprocess func(ctx context.Context, baseTool tool.InvokableTool, toolResponse, toolArguments string) (string, error)

func NewWrapTool(t tool.InvokableTool, preprocess []ToolRequestPreprocess, postprocess []ToolResponsePostprocess) tool.InvokableTool {
	return &wrapTool{
		baseTool:    t,
		preprocess:  preprocess,
		postprocess: postprocess,
	}
}

type wrapTool struct {
	baseTool    tool.InvokableTool
	preprocess  []ToolRequestPreprocess
	postprocess []ToolResponsePostprocess
}

func (w *wrapTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return w.baseTool.Info(ctx)
}

func (w *wrapTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	for _, pre := range w.preprocess {
		var err error
		argumentsInJSON, err = pre(ctx, w.baseTool, argumentsInJSON)
		if err != nil {
			log.Printf("[WrapTool.PreProcess] failed to process response: %v", err)
		}
	}

	resp, err := w.baseTool.InvokableRun(ctx, argumentsInJSON, opts...)
	if err != nil {
		return "", err
	}

	for _, post := range w.postprocess {
		resp, err = post(ctx, w.baseTool, resp, argumentsInJSON)
		if err != nil {
			log.Printf("[WrapTool.PostProcess] failed to process response: %v", err)
			return resp, err
		}
	}

	return resp, nil
}

func ToolRequestRepairJSON(ctx context.Context, baseTool tool.InvokableTool, toolArguments string) (string, error) {
	return utils.RepairJSON(toolArguments), nil
}

type runResult struct {
	Command    string            `json:"command"`
	StdOut     []*stdoutData     `json:"stdout"`
	StdErr     []*stderrData     `json:"stderr"`
	FileChange []*fileChangeData `json:"file_change"`
	ErrData    []*errData        `json:"err_data"`
}

type stdoutData struct {
	Stdout string `json:"stdout"`
}

type stderrData struct {
	Stderr string `json:"stderr"`
}

type errData struct {
	Err string `json:"err"`
}

type fileChangeData struct {
	FileType       string          `json:"file_type"`
	Path           string          `json:"path"`
	Type           string          `json:"type"`
	Uri            string          `json:"uri"`
	Url            string          `json:"url"`
	MultiMediaInfo *multiMediaInfo `json:"multi_media_info,omitempty"`
}

type multiMediaInfo struct {
	MediaType      string `json:"media_type"`
	AdditionalType string `json:"additional_type"`
	AdditionalInfo string `json:"additional_info"`
}

func FilePostProcess(ctx context.Context, baseTool tool.InvokableTool, toolResponse, toolArguments string) (string, error) {
	rawResult := runResult{}
	if err := json.Unmarshal([]byte(toolResponse), &rawResult); err != nil {
		return toolResponse, nil
	}
	var (
		imgs        []*fileChangeData
		path2UrlMap = make(map[string]string)
	)
	for _, d := range rawResult.FileChange {
		data := d
		if data.FileType == "file" && (data.Type == "create" || data.Type == "update") {
			if isImage(data.Uri) {
				imgs = append(imgs, data)
				path2UrlMap[data.Path] = data.Url
			}
		}
	}

	adk.AddSessionValue(ctx, consts.PathUrlMapSessionKey, path2UrlMap)

	type fileOutputFormat struct {
		FileType string `json:"Change subject (file/directory)"`
		Path     string `json:"File/directory relative path"`
		Type     string `json:"Change type (create/delete/update)"`
		Uri      string `json:"File URI"`
	}
	var (
		stdOut     []string
		fileChange []fileOutputFormat
		stdErr     []string
	)
	for _, item := range rawResult.StdOut {
		if item != nil {
			stdOut = append(stdOut, item.Stdout)
		}
	}
	for _, item := range rawResult.FileChange {
		if item != nil {
			fileChange = append(fileChange, fileOutputFormat{
				FileType: item.FileType,
				Path:     item.Path,
				Type:     item.Type,
				Uri:      item.Uri,
			})
		}
	}
	for _, item := range rawResult.StdErr {
		if item != nil {
			stdErr = append(stdErr, item.Stderr)
		}
	}
	for _, item := range rawResult.ErrData {
		if item != nil {
			stdErr = append(stdErr, item.Err)
		}
	}

	var output string
	if len(fileChange) > 0 {
		fcText, _ := jsoniter.MarshalToString(fileChange)
		output += "fileChange: \n" + fcText + "\n"
	}
	if len(stdErr) > 0 {
		output += "shell command stderr and warnings:" + strings.Join(stdErr, "\n") + "\n"
	}
	if len(rawResult.StdOut) > 0 {
		output += "shell command stdout: " + strings.Join(stdOut, "\n") + "\n"
	}

	return output, nil
}

func EditFilePostProcess(ctx context.Context, baseTool tool.InvokableTool, toolResponse, toolArguments string) (string, error) {
	return fmt.Sprintf("Write file: %s success!", toolResponse), nil
}

func isImage(uri string) bool {
	ext := filepath.Ext(uri)
	for _, e := range []string{".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp", ".tiff", ".heic"} {
		if ext == e {
			return true
		}
	}
	return false
}
