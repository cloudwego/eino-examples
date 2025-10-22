package generic

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
)

type SubmitResult struct {
	IsSuccess *bool               `json:"is_success,omitempty"`
	Result    string              `json:"result,omitempty"`
	Files     []*SubmitResultFile `json:"files,omitempty"`
}

type SubmitResultFile struct {
	Path string `json:"path,omitempty"`
	Desc string `json:"desc,omitempty"`
}

func (s *SubmitResult) String() string {
	res := fmt.Sprintf("### 执行结果\n%s", s.Result)
	if len(s.Files) > 0 {
		res += "\n#### 中间产物"
	}
	for _, f := range s.Files {
		res += fmt.Sprintf("\n- 描述：%s, 路径：%s", f.Desc, f.Path)
	}
	return res
}

func ListDir(dir string) ([]*SubmitResultFile, error) {
	var resp []*SubmitResultFile

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if strings.HasPrefix(d.Name(), ".") {
			return nil
		}
		if d.IsDir() {
			next := filepath.Join(dir, d.Name())
			nextResp, err := ListDir(next)
			if err != nil {
				return err
			}
			resp = append(resp, nextResp...)
			return nil
		}
		resp = append(resp, &SubmitResultFile{
			Path: filepath.Join(filepath.Dir(dir), d.Name()),
		})
		return nil
	})

	if err != nil {
		return nil, err
	}

	return resp, nil
}
