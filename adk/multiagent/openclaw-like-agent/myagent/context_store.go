package myagent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type IdentityStore struct {
	workspace string
}

func NewIdentityStore(workspace string) *IdentityStore {
	return &IdentityStore{workspace: workspace}
}

func (m *MemoryStore) Read() (string, error) {
	return readOptionalFile(m.MemoryFile())
}

func (m *MemoryStore) Write(content string) error {
	path := m.MemoryFile()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("创建 memory 目录失败: %w", err)
	}
	if err := os.WriteFile(path, []byte(strings.TrimSpace(content)+"\n"), 0o644); err != nil {
		return fmt.Errorf("写入 MEMORY.md 失败: %w", err)
	}
	return nil
}

func (i *IdentityStore) IdentityFile() string {
	return filepath.Join(i.workspace, identityFileName)
}

func (i *IdentityStore) Read() (string, error) {
	return readOptionalFile(i.IdentityFile())
}

func (i *IdentityStore) Write(content string) error {
	path := i.IdentityFile()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("创建 identity 目录失败: %w", err)
	}
	if err := os.WriteFile(path, []byte(strings.TrimSpace(content)+"\n"), 0o644); err != nil {
		return fmt.Errorf("写入 IDENTITY.md 失败: %w", err)
	}
	return nil
}

func isProtectedContextPath(workspaceRoot, target string) bool {
	absPath := target
	if !filepath.IsAbs(absPath) {
		absPath = filepath.Join(workspaceRoot, target)
	}
	absPath = filepath.Clean(absPath)

	protected := []string{
		filepath.Join(workspaceRoot, "memory", memoryFileName),
		filepath.Join(workspaceRoot, identityFileName),
	}
	for _, path := range protected {
		if absPath == filepath.Clean(path) {
			return true
		}
	}
	return false
}
