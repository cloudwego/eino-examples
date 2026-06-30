/*
 * Copyright 2026 CloudWeGo Authors
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

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/cloudwego/eino/adk/middlewares/filesystem"
	"github.com/cloudwego/eino/adk/prebuilt/team"
)

// ---------------------------------------------------------------------------
// File-based Backend that persists data to the filesystem
// ---------------------------------------------------------------------------

type fileBackend struct {
	baseDir       string
	mu            sync.RWMutex
	seededInboxes map[string]bool
}

func newFileBackend(baseDir string) *fileBackend {
	if baseDir == "" {
		baseDir = os.TempDir()
	}
	return &fileBackend{
		baseDir:       baseDir,
		seededInboxes: make(map[string]bool),
	}
}

func (b *fileBackend) LsInfo(_ context.Context, req *team.LsInfoRequest) ([]team.FileInfo, error) {
	entries, err := os.ReadDir(req.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read dir %s: %w", req.Path, err)
	}

	var result []team.FileInfo
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		result = append(result, team.FileInfo{
			Path:       filepath.Join(req.Path, entry.Name()),
			IsDir:      entry.IsDir(),
			Size:       info.Size(),
			ModifiedAt: info.ModTime().UTC().Format(time.RFC3339),
		})
	}
	return result, nil
}

func (b *fileBackend) Read(_ context.Context, req *team.ReadRequest) (*filesystem.FileContent, error) {
	data, err := os.ReadFile(req.FilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", req.FilePath)
		}
		return nil, fmt.Errorf("read file %s: %w", req.FilePath, err)
	}
	return &filesystem.FileContent{Content: string(data)}, nil
}

func (b *fileBackend) Write(_ context.Context, req *team.WriteRequest) error {
	dir := filepath.Dir(req.FilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create dir %s: %w", dir, err)
	}

	content := req.Content

	// Atomic replace: write to a temp file in the same directory, fsync, then
	// rename over the target. This satisfies the team.Backend durability contract
	// so a crash mid-write can never leave a truncated config.json or inbox.
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(req.FilePath)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file for %s: %w", req.FilePath, err)
	}
	tmpName := tmp.Name()
	// Best-effort cleanup if we bail out before the rename succeeds.
	defer func() { _ = os.Remove(tmpName) }()

	if _, err := tmp.WriteString(content); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temp file for %s: %w", req.FilePath, err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync temp file for %s: %w", req.FilePath, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file for %s: %w", req.FilePath, err)
	}
	if err := os.Chmod(tmpName, 0644); err != nil {
		return fmt.Errorf("chmod temp file for %s: %w", req.FilePath, err)
	}
	if err := os.Rename(tmpName, req.FilePath); err != nil {
		return fmt.Errorf("rename temp file to %s: %w", req.FilePath, err)
	}
	return nil
}

func (b *fileBackend) Delete(_ context.Context, req *team.DeleteRequest) error {
	info, err := os.Stat(req.FilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat %s: %w", req.FilePath, err)
	}

	if info.IsDir() {
		if err := os.RemoveAll(req.FilePath); err != nil {
			return fmt.Errorf("remove dir %s: %w", req.FilePath, err)
		}
	} else {
		if err := os.Remove(req.FilePath); err != nil {
			return fmt.Errorf("remove file %s: %w", req.FilePath, err)
		}
	}
	return nil
}

func (b *fileBackend) Exists(_ context.Context, path string) (bool, error) {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("stat %s: %w", path, err)
	}
	return true, nil
}

func (b *fileBackend) Mkdir(_ context.Context, path string) error {
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("mkdir %s: %w", path, err)
	}
	return nil
}

func (b *fileBackend) FileExists(_ context.Context, path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("stat %s: %w", path, err)
	}
	return !info.IsDir(), nil
}
