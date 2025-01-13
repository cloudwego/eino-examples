package open

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

type Open struct {
}

func (of *Open) ToEinoTool() (tool.InvokableTool, error) {
	return utils.InferTool("open", "open a file/dir/web url in the system by default application", of.Invoke)
}

func (of *Open) Invoke(ctx context.Context, req OpenReq) (res OpenRes, err error) {
	if req.URI == "" {
		res.Message = "uri is required"
		return res, nil
	}

	// if is file or dir, check if exists
	if isFilePath(req.URI) {
		if _, err := os.Stat(req.URI); os.IsNotExist(err) {
			res.Message = fmt.Sprintf("file not exists: %s", req.URI)
			return res, nil
		}
	}

	err = exec.Command("open", req.URI).Run()
	if err != nil {
		res.Message = fmt.Sprintf("failed to open %s: %s", req.URI, err.Error())
		return res, nil
	}

	res.Message = fmt.Sprintf("success, open %s", req.URI)
	return res, nil
}

type OpenReq struct {
	URI string `json:"uri" jsonschema:"description=The uri of the file/dir/web url to open"`
}

type OpenRes struct {
	Message string `json:"message" jsonschema:"description=The message of the operation"`
}

func isFilePath(path string) bool {
	return strings.HasPrefix(path, "file://") && !strings.Contains(path, "://")
}
