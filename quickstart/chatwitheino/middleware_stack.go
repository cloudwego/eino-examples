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
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	localbk "github.com/cloudwego/eino-ext/adk/backend/local"
	"github.com/cloudwego/eino/adk"
	adkfs "github.com/cloudwego/eino/adk/filesystem"
	"github.com/cloudwego/eino/adk/middlewares/agentsmd"
	"github.com/cloudwego/eino/adk/middlewares/dynamictool/toolsearch"
	"github.com/cloudwego/eino/adk/middlewares/patchtoolcalls"
	"github.com/cloudwego/eino/adk/middlewares/plantask"
	"github.com/cloudwego/eino/adk/middlewares/reduction"
	"github.com/cloudwego/eino/adk/middlewares/skill"
	"github.com/cloudwego/eino/adk/middlewares/summarization"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
)

const (
	envMiddlewares           = "CHATWITHEINO_MIDDLEWARES"
	envAgentsMDFiles         = "CHATWITHEINO_AGENTS_MD_FILES"
	envAgentsMDMaxBytes      = "CHATWITHEINO_AGENTS_MD_MAX_BYTES"
	envPlantaskDir           = "CHATWITHEINO_PLANTASK_DIR"
	envReductionDir          = "CHATWITHEINO_REDUCTION_DIR"
	envReductionMaxChars     = "CHATWITHEINO_REDUCTION_MAX_CHARS"
	envReductionMaxTokens    = "CHATWITHEINO_REDUCTION_MAX_TOKENS"
	envSummarizationTokens   = "CHATWITHEINO_SUMMARY_TOKENS"
	envSummarizationMessages = "CHATWITHEINO_SUMMARY_MESSAGES"
	envSummarizationEvents   = "CHATWITHEINO_SUMMARY_EVENTS"
	envSkillsDir             = "EINO_EXT_SKILLS_DIR"
)

var defaultMiddlewareNames = []string{
	"patchtoolcalls",
	"reduction",
	"summarization",
	"plantask",
	"agentsmd",
	"skill",
	"toolsearch",
}

type middlewareStack[M adk.MessageType] struct {
	Handlers []adk.TypedChatModelAgentMiddleware[M]
	Tools    []tool.BaseTool
}

func buildMiddlewareStack[M adk.MessageType](
	ctx context.Context,
	cm model.BaseModel[M],
	backend *localbk.Local,
	ragTool tool.BaseTool,
) (*middlewareStack[M], error) {
	selected := resolveSelectedMiddlewares()
	stack := &middlewareStack[M]{
		Tools: []tool.BaseTool{ragTool},
	}
	var enabled []string

	if selected["patchtoolcalls"] {
		mw, err := patchtoolcalls.NewTyped[M](ctx, nil)
		if err != nil {
			return nil, fmt.Errorf("create patchtoolcalls middleware: %w", err)
		}
		stack.Handlers = append(stack.Handlers, mw)
		enabled = append(enabled, "patchtoolcalls")
	}

	if selected["reduction"] {
		mw, err := reduction.NewTyped[M](ctx, &reduction.TypedConfig[M]{
			Backend:           backend,
			RootDir:           absPath(envString(envReductionDir, "./data/middleware/reduction")),
			MaxLengthForTrunc: envInt(envReductionMaxChars, 50000),
			MaxTokensForClear: envInt64(envReductionMaxTokens, 160000),
		})
		if err != nil {
			return nil, fmt.Errorf("create reduction middleware: %w", err)
		}
		stack.Handlers = append(stack.Handlers, mw)
		enabled = append(enabled, "reduction")
	}

	if selected["summarization"] {
		mw, err := summarization.NewTyped[M](ctx, &summarization.TypedConfig[M]{
			Model: cm,
			Trigger: &summarization.TriggerCondition{
				ContextTokens:   envInt(envSummarizationTokens, 160000),
				ContextMessages: envInt(envSummarizationMessages, 0),
			},
			EmitInternalEvents: envBool(envSummarizationEvents, false),
		})
		if err != nil {
			return nil, fmt.Errorf("create summarization middleware: %w", err)
		}
		stack.Handlers = append(stack.Handlers, mw)
		enabled = append(enabled, "summarization")
	}

	if selected["plantask"] {
		mw, err := plantask.NewTyped[M](ctx, &plantask.Config{
			Backend: &plantaskLocalBackend{backend: backend},
			BaseDir: absPath(envString(envPlantaskDir, "./data/middleware/tasks")),
		})
		if err != nil {
			return nil, fmt.Errorf("create plantask middleware: %w", err)
		}
		stack.Handlers = append(stack.Handlers, mw)
		enabled = append(enabled, "plantask")
	}

	if selected["agentsmd"] {
		files := resolveAgentsMDFiles()
		if len(files) == 0 {
			log.Printf("agentsmd middleware skipped: no readable files; set %s", envAgentsMDFiles)
		} else {
			mw, err := agentsmd.NewTyped[M](ctx, &agentsmd.Config{
				Backend:             backend,
				AgentsMDFiles:       files,
				AllAgentsMDMaxBytes: envInt(envAgentsMDMaxBytes, 256*1024),
				OnLoadWarning:       func(path string, err error) { log.Printf("[agentsmd] %s: %v", path, err) },
			})
			if err != nil {
				return nil, fmt.Errorf("create agentsmd middleware: %w", err)
			}
			stack.Handlers = append(stack.Handlers, mw)
			enabled = append(enabled, "agentsmd")
		}
	}

	if selected["skill"] {
		if skillsDir, ok := resolveSkillsDir(); ok {
			skillBackend, err := skill.NewBackendFromFilesystem(ctx, &skill.BackendFromFilesystemConfig{
				Backend: backend,
				BaseDir: skillsDir,
			})
			if err != nil {
				return nil, fmt.Errorf("create skill backend: %w", err)
			}
			mw, err := skill.NewTyped[M](ctx, &skill.TypedConfig[M]{
				Backend: skillBackend,
			})
			if err != nil {
				return nil, fmt.Errorf("create skill middleware: %w", err)
			}
			stack.Handlers = append(stack.Handlers, mw)
			enabled = append(enabled, "skill")
		} else {
			log.Printf("skill middleware skipped: set %s to a skills directory", envSkillsDir)
		}
	}

	if selected["toolsearch"] {
		mw, err := toolsearch.NewTyped[M](ctx, &toolsearch.Config{
			DynamicTools: []tool.BaseTool{ragTool},
		})
		if err != nil {
			return nil, fmt.Errorf("create toolsearch middleware: %w", err)
		}
		stack.Handlers = append(stack.Handlers, mw)
		stack.Tools = nil
		enabled = append(enabled, "toolsearch")
	}

	if len(enabled) == 0 {
		log.Printf("chatwitheino middleware stack: none")
	} else {
		log.Printf("chatwitheino middleware stack: %s", strings.Join(enabled, ", "))
	}
	return stack, nil
}

func resolveSelectedMiddlewares() map[string]bool {
	raw := strings.TrimSpace(os.Getenv(envMiddlewares))
	if raw == "" || strings.EqualFold(raw, "all") {
		return middlewareNameSet(defaultMiddlewareNames...)
	}
	if strings.EqualFold(raw, "none") {
		return map[string]bool{}
	}

	selected := map[string]bool{}
	for _, name := range strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';' || r == ' ' || r == '\t' || r == '\n'
	}) {
		name = normalizeMiddlewareName(name)
		if name == "" {
			continue
		}
		if name == "all" {
			for k := range middlewareNameSet(defaultMiddlewareNames...) {
				selected[k] = true
			}
			continue
		}
		if !middlewareNameSet(defaultMiddlewareNames...)[name] {
			log.Printf("unknown middleware %q in %s; ignoring", name, envMiddlewares)
			continue
		}
		selected[name] = true
	}
	return selected
}

func normalizeMiddlewareName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.ReplaceAll(name, "_", "")
	name = strings.ReplaceAll(name, "-", "")
	switch name {
	case "patch", "patchtoolcall":
		return "patchtoolcalls"
	case "summary", "summarize", "summarisation", "summarization":
		return "summarization"
	case "reduce", "toolreduction":
		return "reduction"
	case "plan", "task", "plantask":
		return "plantask"
	case "agents", "agentsmd", "agentsmarkdown":
		return "agentsmd"
	case "skills":
		return "skill"
	case "toolsearch", "dynamictool", "dynamictoolsearch":
		return "toolsearch"
	default:
		return name
	}
}

func middlewareNameSet(names ...string) map[string]bool {
	set := make(map[string]bool, len(names))
	for _, name := range names {
		set[name] = true
	}
	return set
}

func resolveSkillsDir() (string, bool) {
	skillsDir := strings.TrimSpace(os.Getenv(envSkillsDir))
	if skillsDir == "" {
		return "", false
	}
	skillsDir = absPath(skillsDir)
	fi, err := os.Stat(skillsDir)
	if err != nil || !fi.IsDir() {
		return "", false
	}
	return skillsDir, true
}

func resolveAgentsMDFiles() []string {
	raw := strings.TrimSpace(os.Getenv(envAgentsMDFiles))
	if raw == "" {
		raw = "./AGENTS.md"
	}

	var files []string
	for _, candidate := range strings.Split(raw, ",") {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		path := absPath(candidate)
		if fi, err := os.Stat(path); err == nil && !fi.IsDir() {
			files = append(files, path)
		} else if os.Getenv(envAgentsMDFiles) != "" {
			log.Printf("agentsmd file skipped: %s", path)
		}
	}
	return files
}

func envString(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func envInt(name string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		log.Printf("invalid %s=%q; using %d", name, value, fallback)
		return fallback
	}
	return parsed
}

func envInt64(name string, fallback int64) int64 {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		log.Printf("invalid %s=%q; using %d", name, value, fallback)
		return fallback
	}
	return parsed
}

func envBool(name string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		log.Printf("invalid %s=%q; using %t", name, value, fallback)
		return fallback
	}
	return parsed
}

func absPath(path string) string {
	if abs, err := filepath.Abs(path); err == nil {
		return abs
	}
	return path
}

type plantaskLocalBackend struct {
	backend *localbk.Local
}

func (b *plantaskLocalBackend) LsInfo(ctx context.Context, req *plantask.LsInfoRequest) ([]plantask.FileInfo, error) {
	files, err := b.backend.LsInfo(ctx, req)
	if err != nil {
		return nil, err
	}
	if req == nil || strings.TrimSpace(req.Path) == "" {
		return files, nil
	}
	for i := range files {
		if files[i].Path != "" && !filepath.IsAbs(files[i].Path) {
			files[i].Path = filepath.Join(req.Path, files[i].Path)
		}
	}
	return files, nil
}

func (b *plantaskLocalBackend) Read(ctx context.Context, req *plantask.ReadRequest) (*adkfs.FileContent, error) {
	return b.backend.Read(ctx, req)
}

func (b *plantaskLocalBackend) Write(ctx context.Context, req *plantask.WriteRequest) error {
	return b.backend.Write(ctx, req)
}

func (b *plantaskLocalBackend) Delete(_ context.Context, req *plantask.DeleteRequest) error {
	if req == nil || strings.TrimSpace(req.FilePath) == "" {
		return fmt.Errorf("file path is required")
	}
	if err := os.Remove(filepath.Clean(req.FilePath)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete task file: %w", err)
	}
	return nil
}
