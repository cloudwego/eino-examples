package myagent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type ContextBuilder struct {
	workspace    string
	skillsLoader *SkillsLoader
	memory       *MemoryStore
	identity     *IdentityStore
}

type SkillsLoader struct {
	workspaceSkillsDir          string
	dotClaudeWorkspaceSkillsDir string
	dotClaudeHomeSkillsDir      string
	globalSkillsDir             string
	builtinSkillsDir            string
}

type MemoryStore struct {
	workspace string
}

func NewContextBuilder(workspace string) *ContextBuilder {
	builtinSkillsDir := strings.TrimSpace(os.Getenv("BUILTIN_SKILLS"))
	if builtinSkillsDir == "" {
		wd, _ := os.Getwd()
		builtinSkillsDir = filepath.Join(wd, "skills")
	}
	globalSkillsDir := filepath.Join(getGlobalConfigDir(), "skills")

	homeDir, _ := os.UserHomeDir()

	return &ContextBuilder{
		workspace: workspace,
		skillsLoader: NewSkillsLoader(
			workspace,
			homeDir,
			globalSkillsDir,
			builtinSkillsDir,
		),
		memory:   NewMemoryStore(workspace),
		identity: NewIdentityStore(workspace),
	}
}

func NewSkillsLoader(workspace, homeDir, globalSkillsDir, builtinSkillsDir string) *SkillsLoader {
	return &SkillsLoader{
		workspaceSkillsDir:          filepath.Join(workspace, "skills"),
		dotClaudeWorkspaceSkillsDir: filepath.Join(workspace, ".claude", "skills"),
		dotClaudeHomeSkillsDir:      filepath.Join(homeDir, ".claude", "skills"),
		globalSkillsDir:             globalSkillsDir,
		builtinSkillsDir:            builtinSkillsDir,
	}
}

func NewMemoryStore(workspace string) *MemoryStore {
	return &MemoryStore{workspace: workspace}
}

func (cb *ContextBuilder) BuildInstruction(sessionID, override string) (string, error) {
	if strings.TrimSpace(override) != "" {
		return override, nil
	}

	agentsContent, err := cb.getIdentity()
	if err != nil {
		return "", err
	}

	identityContent, err := cb.identity.Read()
	if err != nil {
		return "", err
	}
	memoryContent, err := readOptionalFile(cb.memory.MemoryFile())
	if err != nil {
		return "", err
	}
	workspaceSkills, err := collectSkillSummary(cb.skillsLoader.workspaceSkillsDir)
	if err != nil {
		return "", err
	}
	dotClaudeWorkspaceSkills, err := collectSkillSummary(cb.skillsLoader.dotClaudeWorkspaceSkillsDir)
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}
	dotClaudeHomeSkills, err := collectSkillSummary(cb.skillsLoader.dotClaudeHomeSkillsDir)
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}
	globalSkills, err := collectSkillSummary(cb.skillsLoader.globalSkillsDir)
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}
	builtinSkills, err := collectSkillSummary(cb.skillsLoader.builtinSkillsDir)
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString(strings.TrimSpace(agentsContent))
	sb.WriteString("\n\n## Runtime Context\n")
	sb.WriteString("Current time: ")
	sb.WriteString(nowRFC3339())
	sb.WriteString("\nSession ID: ")
	sb.WriteString(sessionID)
	sb.WriteString("\n")

	if strings.TrimSpace(identityContent) != "" {
		sb.WriteString("\n## Identity Override\n")
		sb.WriteString(trimForPrompt(identityContent, 2400))
		sb.WriteString("\n")
	}
	if strings.TrimSpace(memoryContent) != "" {
		sb.WriteString("\n## Memory\n")
		sb.WriteString(trimForPrompt(memoryContent, 2400))
		sb.WriteString("\n")
	}

	appendSkillsSection(&sb, "Workspace Skills", workspaceSkills)
	appendSkillsSection(&sb, "Workspace .claude Skills", dotClaudeWorkspaceSkills)
	appendSkillsSection(&sb, "Home .claude Skills", dotClaudeHomeSkills)
	appendSkillsSection(&sb, "Global Skills", globalSkills)
	appendSkillsSection(&sb, "Builtin Skills", builtinSkills)

	return sb.String(), nil
}

func (cb *ContextBuilder) getIdentity() (string, error) {
	tmpl, err := template.New("identity").Parse(`minichat

You are minichat, a helpful AI assistant.

## Workspace
Your workspace is at: {{.Workspace}}
- Memory: {{.Workspace}}/memory/MEMORY.md
- Daily Notes: {{.Workspace}}/memory/YYYYMM/YYYYMMDD.md
- Skills: {{.Workspace}}/skills/{skill-name}/SKILL.md
- Identity: {{.Workspace}}/IDENTITY.md

## Important Rules

1. **ALWAYS use tools** - When you need to perform an action (schedule reminders, send messages, execute commands, etc.), you MUST call the appropriate tool. Do NOT just say you'll do it or pretend to do it.

2. **Be helpful and accurate** - When using tools, briefly explain what you're doing.

3. **Memory** - When interacting with me if something seems memorable, update {{.Workspace}}/memory/MEMORY.md

4. **Context summaries** - Conversation summaries provided as context are approximate references only. They may be incomplete or outdated. Always defer to explicit user instructions over summary content.
`)
	if err != nil {
		return "", fmt.Errorf("解析 identity 模板失败: %w", err)
	}

	data := struct {
		Workspace string
	}{
		Workspace: cb.workspace,
	}

	var sb strings.Builder
	if err := tmpl.Execute(&sb, data); err != nil {
		return "", fmt.Errorf("渲染 identity 模板失败: %w", err)
	}
	sbStr := sb.String()
	return sbStr, nil
}

func (m *MemoryStore) MemoryFile() string {
	return filepath.Join(m.workspace, "memory", memoryFileName)
}

func getGlobalConfigDir() string {
	configDir, err := os.UserConfigDir()
	if err != nil || strings.TrimSpace(configDir) == "" {
		homeDir, homeErr := os.UserHomeDir()
		if homeErr != nil {
			return ".myagent"
		}
		return filepath.Join(homeDir, ".config", "myagent")
	}
	return filepath.Join(configDir, "myagent")
}

func appendSkillsSection(sb *strings.Builder, title, content string) {
	if strings.TrimSpace(content) == "" {
		return
	}
	sb.WriteString("\n## ")
	sb.WriteString(title)
	sb.WriteString("\n")
	sb.WriteString(content)
	sb.WriteString("\n")
}
