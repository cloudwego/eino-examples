package myagent

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	identityFileName = "IDENTITY.md"
	memoryFileName   = "MEMORY.md"
)

var errNotExist = errors.New("not exist")

type fileInfo interface {
	IsDir() bool
}

type workspaceRuntime struct {
	root         string
	memoryDir    string
	skillsDir    string
	sessionsDir  string
	artifactsDir string
	logsDir      string
}

func ensureWorkspaceRuntime(root string) (*workspaceRuntime, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, errors.New("workspace 不能为空")
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("解析 workspace 路径失败: %w", err)
	}

	ws := &workspaceRuntime{
		root:         absRoot,
		memoryDir:    filepath.Join(absRoot, "memory"),
		skillsDir:    filepath.Join(absRoot, "skills"),
		sessionsDir:  filepath.Join(absRoot, "sessions"),
		artifactsDir: filepath.Join(absRoot, "artifacts"),
		logsDir:      filepath.Join(absRoot, "logs"),
	}

	for _, dir := range []string{ws.root, ws.memoryDir, ws.skillsDir, ws.sessionsDir, ws.artifactsDir, ws.logsDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("创建 workspace 目录失败 %s: %w", dir, err)
		}
	}

	if err := ensureFileWithDefault(filepath.Join(ws.root, identityFileName), defaultIdentityMarkdown()); err != nil {
		return nil, err
	}
	if err := ensureFileWithDefault(filepath.Join(ws.memoryDir, memoryFileName), defaultMemoryMarkdown()); err != nil {
		return nil, err
	}

	return ws, nil
}

func defaultIdentityMarkdown() string {
	return "# Identity\n\nYou are MyAgent, a concise engineering assistant that works step by step.\n"
}

func defaultMemoryMarkdown() string {
	return `# Long-term Memory

This file stores important information that should persist across sessions.

## User Information

(Important facts about user)

## Preferences

(User preferences learned over time)
...
`
}

func ensureFileWithDefault(path, content string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("检查文件失败 %s: %w", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("写入默认文件失败 %s: %w", path, err)
	}
	return nil
}

func readOptionalFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("读取文件失败 %s: %w", path, err)
	}
	return string(data), nil
}

// skillInfo holds metadata about a single discovered skill.
type skillInfo struct {
	Name    string
	Dir     string
	Summary string
}

// skillDirEntry groups skills found under one logical source directory.
type skillDirEntry struct {
	Label  string
	Path   string
	Skills []skillInfo
}

// listAllSkills returns all skills across every known source directory in
// priority order: workspace/skills, workspace/.claude/skills,
// ~/.claude/skills, global config skills, builtin skills.
func listAllSkills(loader *SkillsLoader) ([]skillDirEntry, error) {
	sources := []struct {
		label string
		path  string
	}{
		{"Workspace Skills", loader.workspaceSkillsDir},
		{"Workspace .claude Skills", loader.dotClaudeWorkspaceSkillsDir},
		{"Home .claude Skills", loader.dotClaudeHomeSkillsDir},
		{"Global Skills", loader.globalSkillsDir},
		{"Builtin Skills", loader.builtinSkillsDir},
	}

	var result []skillDirEntry
	for _, src := range sources {
		skills, err := collectSkillInfos(src.path)
		if err != nil {
			return nil, err
		}
		result = append(result, skillDirEntry{
			Label:  src.label,
			Path:   src.path,
			Skills: skills,
		})
	}
	return result, nil
}

func collectSkillInfos(skillsDir string) ([]skillInfo, error) {
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("读取 skills 目录失败: %w", err)
	}
	var infos []skillInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillPath := filepath.Join(skillsDir, entry.Name(), "SKILL.md")
		content, err := readOptionalFile(skillPath)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(content) == "" {
			continue
		}
		infos = append(infos, skillInfo{
			Name:    entry.Name(),
			Dir:     filepath.Join(skillsDir, entry.Name()),
			Summary: firstNonEmptyLine(content),
		})
	}
	return infos, nil
}

func collectSkillSummary(skillsDir string) (string, error) {
	infos, err := collectSkillInfos(skillsDir)
	if err != nil {
		return "", err
	}
	lines := make([]string, 0, len(infos))
	for _, info := range infos {
		lines = append(lines, fmt.Sprintf("- %s: %s", info.Name, trimForDisplay(info.Summary, 120)))
	}
	return strings.Join(lines, "\n"), nil
}

func osStat(path string) (fileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errNotExist
		}
		return nil, err
	}
	return info, nil
}
