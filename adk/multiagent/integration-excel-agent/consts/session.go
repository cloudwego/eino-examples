package consts

import (
	"context"

	"github.com/cloudwego/eino/adk"
)

const (
	FilePathSessionKey            = "file_path_session_key"
	UserAllPreviewFilesSessionKey = "user_all_preview_files_session_key"
	WorkDirSessionKey             = "work_dir_session_key"
	PathUrlMapSessionKey          = "path_url_map_session_key"
)

func GetSessionValue[T any](ctx context.Context, key string) (T, bool) {
	v, ok := adk.GetSessionValue(ctx, key)
	if !ok {
		var zero T
		return zero, false
	}
	t, ok := v.(T)
	if !ok {
		var zero T
		return zero, false
	}

	return t, true
}
