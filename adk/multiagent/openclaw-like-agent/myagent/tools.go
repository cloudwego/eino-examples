package myagent

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

type writeFileInput struct {
	Path    string `json:"path" jsonschema:"description=待写入文件路径"`
	Content string `json:"content" jsonschema:"description=完整文件内容"`
}

type appendFileInput struct {
	Path    string `json:"path" jsonschema:"description=待追加文件路径"`
	Content string `json:"content" jsonschema:"description=要追加的内容"`
}

type fileMutationOutput struct {
	Path   string `json:"path"`
	Status string `json:"status"`
}

func newWorkspaceTools(workspaceRoot string) ([]tool.BaseTool, error) {
	appendTool, err := utils.InferTool("append_file", fmt.Sprintf("追加写入 %s 内文件。", workspaceRoot), func(ctx context.Context, input appendFileInput) (fileMutationOutput, error) {
		return writeWorkspaceFile(workspaceRoot, input.Path, input.Content, true)
	})
	if err != nil {
		return nil, err
	}

	return []tool.BaseTool{appendTool}, nil
}

func writeWorkspaceFile(workspaceRoot, target, content string, appendMode bool) (fileMutationOutput, error) {
	if strings.TrimSpace(target) == "" {
		return fileMutationOutput{}, errors.New("path 不能为空")
	}
	absPath, err := resolveWorkspacePath(workspaceRoot, target)
	if err != nil {
		return fileMutationOutput{}, err
	}
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return fileMutationOutput{}, fmt.Errorf("创建目录失败: %w", err)
	}
	flag := os.O_CREATE | os.O_WRONLY
	status := "written"
	if appendMode {
		flag |= os.O_APPEND
		status = "appended"
	} else {
		flag |= os.O_TRUNC
	}
	file, err := os.OpenFile(absPath, flag, 0o644)
	if err != nil {
		return fileMutationOutput{}, fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()
	if _, err := file.WriteString(content); err != nil {
		return fileMutationOutput{}, fmt.Errorf("写入文件失败: %w", err)
	}
	return fileMutationOutput{Path: absPath, Status: status}, nil
}

func resolveWorkspacePath(workspaceRoot, target string) (string, error) {
	if strings.TrimSpace(target) == "" {
		return "", errors.New("path 不能为空")
	}
	var absPath string
	if filepath.IsAbs(target) {
		absPath = filepath.Clean(target)
	} else {
		absPath = filepath.Clean(filepath.Join(workspaceRoot, target))
	}
	rel, err := filepath.Rel(workspaceRoot, absPath)
	if err != nil {
		return "", fmt.Errorf("解析路径失败: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("路径越界: %s 不在 workspace 内", absPath)
	}
	return absPath, nil
}
