package myagent

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/cloudwego/eino-examples/adk/multiagent/openclaw-like-agent/myagent/backend/secureBackend"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/filesystem"
	filesystemMiddleware "github.com/cloudwego/eino/adk/middlewares/filesystem"
	"github.com/cloudwego/eino/adk/middlewares/skill"
	"github.com/cloudwego/eino/adk/middlewares/summarization"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

var extSkillDir = []string{}

type runtime struct {
	workspace *workspaceRuntime
	sessionID string
	store     *jsonlSessionStore
	runner    *adk.Runner
}

func newRuntime(ctx context.Context, opts commandOptions) (*runtime, error) {
	workspacePath := strings.TrimSpace(opts.Workspace)
	if workspacePath == "" {
		cwd, err := filepath.Abs(".")
		if err != nil {
			return nil, fmt.Errorf("获取当前目录失败: %w", err)
		}
		workspacePath = filepath.Join(cwd, "myagent_workspace")
	}

	ws, err := ensureWorkspaceRuntime(workspacePath)
	if err != nil {
		return nil, err
	}

	sessionID := strings.TrimSpace(opts.SessionID)
	if sessionID == "" {
		sessionID = newSessionID()
	}
	if err := validateSessionID(sessionID); err != nil {
		return nil, err
	}

	store, err := newJSONLSessionStore(ws.sessionsDir)
	if err != nil {
		return nil, err
	}
	if err := store.EnsureSession(sessionID); err != nil {
		return nil, err
	}

	skillPaths := buildSkillPaths(ws.root, extSkillDir)
	allowPaths, err := buildReadAllowPathPatterns(skillPaths)
	if err != nil {
		return nil, err
	}

	backend, err := secureBackend.New(&secureBackend.Config{
		Workspace:  ws.root,
		Restrict:   true,
		AllowPaths: allowPaths,
		ProtectedPaths: []string{
			filepath.Join(ws.root, "memory", memoryFileName),
			filepath.Join(ws.root, identityFileName),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("创建 secure backend 失败: %w", err)
	}
	apiKey := os.Getenv("OPENAI_API_KEY")
	modelName := os.Getenv("OPENAI_MODEL")
	baseURL := os.Getenv("OPENAI_BASE_URL")
	byAzure := os.Getenv("OPENAI_BY_AZURE") == "true"
	maxTokens := 16000

	cm, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:    apiKey,
		Model:     modelName,
		BaseURL:   baseURL,
		MaxTokens: &maxTokens,
		ByAzure:   byAzure,
	})
	if err != nil {
		return nil, fmt.Errorf("创建 openai chat model 失败: %w", err)
	}

	handlers, err := loadSkillHandlers(ctx, backend, ws.root, skillPaths)
	if err != nil {
		return nil, err
	}

	contextBuilder := NewContextBuilder(ws.root)
	agentInstruction, err := contextBuilder.BuildInstruction(sessionID, strings.TrimSpace(opts.Instruction))
	if err != nil {
		return nil, err
	}

	tools, err := newWorkspaceTools(ws.root)
	if err != nil {
		return nil, err
	}

	maxIterations := opts.MaxIterations
	if maxIterations <= 0 {
		maxIterations = 50
	}

	// 构建 filesystem middleware（对应 deep.Config.Backend 注册 read/write/edit/glob/grep 工具）
	fsMw, err := filesystemMiddleware.New(ctx, &filesystemMiddleware.MiddlewareConfig{
		Backend: backend,
		Shell:   backend,
	})
	if err != nil {
		return nil, fmt.Errorf("创建 filesystem middleware 失败: %w", err)
	}

	// summary
	summaryMw, err := summarization.New(ctx, &summarization.Config{
		Model: cm,
		Trigger: &summarization.TriggerCondition{
			ContextTokens: 100000,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("创建 summary middleware 失败: %w", err)
	}

	// 组合所有 handlers：filesystem middleware 优先，再追加 skill handlers
	allHandlers := append([]adk.ChatModelAgentMiddleware{fsMw, summaryMw}, handlers...)

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "MyAgent",
		Description: "workspace/session aware local agent",
		Instruction: agentInstruction,
		Model:       cm,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: tools,
				// toolRetryMiddleware 先执行：对瞬态错误自动重试。
				// toolErrorRecoveryMiddleware 后执行：将最终错误转为模型可读的 result，避免 agent 崩溃。
				ToolCallMiddlewares: []compose.ToolMiddleware{toolRetryMiddleware(3), toolErrorRecoveryMiddleware()},
			},
			EmitInternalEvents: true,
		},
		MaxIterations: maxIterations,
		Handlers:      allHandlers,
	})
	if err != nil {
		return nil, fmt.Errorf("创建 chat model agent 失败: %w", err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
	})

	return &runtime{
		workspace: ws,
		sessionID: sessionID,
		store:     store,
		runner:    runner,
	}, nil
}

func (rt *runtime) RunTurn(ctx context.Context, userInput string, out io.Writer) error {
	userInput = strings.TrimSpace(userInput)
	if userInput == "" {
		return errors.New("message 不能为空")
	}

	history, err := rt.store.GetHistory(rt.sessionID)
	if err != nil {
		return err
	}

	if err := rt.store.AddMessage(rt.sessionID, schema.UserMessage(userInput)); err != nil {
		return err
	}

	runMessages := append(cloneMessages(history), schema.UserMessage(userInput))
	printRunHeader(out, rt, userInput)

	events := rt.runner.Run(ctx, runMessages)
	recorder := newTurnRecorder()
	for {
		event, ok := events.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			return fmt.Errorf("agent run 失败: %w", event.Err)
		}
		if err := renderAgentEvent(out, event, recorder); err != nil {
			return err
		}
	}

	messages := recorder.Messages()
	if len(messages) == 0 {
		return errors.New("模型未返回任何消息")
	}

	if err := rt.store.AddMessages(rt.sessionID, messages...); err != nil {
		return err
	}
	if err := rt.store.TouchSummary(rt.sessionID, recorder.AssistantReply()); err != nil {
		return err
	}

	fmt.Fprintf(out, "\n[run.completed] session=%s saved_messages=%d at=%s\n",
		rt.sessionID,
		1+len(messages),
		time.Now().Format("15:04:05"),
	)
	return nil
}

func loadSkillHandlers(ctx context.Context, backend filesystem.Backend, resolveRoot string, paths []string) ([]adk.ChatModelAgentMiddleware, error) {
	seen := make(map[string]struct{}, len(paths))
	handlers := make([]adk.ChatModelAgentMiddleware, 0, len(paths))
	for _, rawPath := range paths {
		skillPath := strings.TrimSpace(rawPath)
		if skillPath == "" {
			continue
		}
		if !filepath.IsAbs(skillPath) {
			skillPath = filepath.Join(resolveRoot, skillPath)
		}
		skillPath, err := filepath.Abs(skillPath)
		if err != nil {
			return nil, fmt.Errorf("解析 skill 路径失败 %q: %w", rawPath, err)
		}
		if _, ok := seen[skillPath]; ok {
			continue
		}
		seen[skillPath] = struct{}{}

		info, err := osStat(skillPath)
		if err != nil {
			if errors.Is(err, errNotExist) {
				continue
			}
			return nil, fmt.Errorf("检查 skill 路径失败 %q: %w", skillPath, err)
		}
		if !info.IsDir() {
			continue
		}

		skillBackend, err := skill.NewBackendFromFilesystem(ctx, &skill.BackendFromFilesystemConfig{
			Backend: backend,
			BaseDir: skillPath,
		})
		if err != nil {
			return nil, fmt.Errorf("创建 skill backend 失败 %q: %w", skillPath, err)
		}
		middleware, err := skill.NewMiddleware(ctx, &skill.Config{
			Backend: skillBackend,
		})
		if err != nil {
			return nil, fmt.Errorf("创建 skill middleware 失败 %q: %w", skillPath, err)
		}
		handlers = append(handlers, middleware)
	}
	return handlers, nil
}

// isTransientToolError reports whether err is a transient failure that is safe
// to retry automatically without involving the model (e.g. network blips,
// temporary I/O errors).  Permanent errors such as "access denied" or "path
// escapes workspace" must NOT be retried — they need the model to correct the
// parameters instead.
func isTransientToolError(err error) bool {
	if err == nil {
		return false
	}
	// Context cancellation / deadline: never retry — the caller gave up.
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	// Network-level transient conditions.
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	msg := err.Error()
	transientPhrases := []string{
		"connection reset",
		"connection refused",
		"EOF",
		"broken pipe",
		"i/o timeout",
		"temporary failure",
		"try again",
		"resource temporarily unavailable",
	}
	lower := strings.ToLower(msg)
	for _, phrase := range transientPhrases {
		if strings.Contains(lower, phrase) {
			return true
		}
	}
	return false
}

// toolRetryMiddleware automatically retries a tool call up to maxAttempts times
// when a transient error is detected.  Between retries it applies an
// exponential back-off (100 ms → 200 ms → 400 ms …) so as not to hammer a
// temporarily unavailable resource.  Non-transient errors are passed through
// immediately so that toolErrorRecoveryMiddleware (which runs after this one)
// can surface them to the model as a structured [tool_error] result.
func toolRetryMiddleware(maxAttempts int) compose.ToolMiddleware {
	if maxAttempts <= 0 {
		maxAttempts = 3
	}

	backoff := func(attempt int) time.Duration {
		d := 100 * time.Millisecond
		for i := 0; i < attempt; i++ {
			d *= 2
		}
		if d > 2*time.Second {
			d = 2 * time.Second
		}
		return d
	}

	wrap := func(next compose.InvokableToolEndpoint) compose.InvokableToolEndpoint {
		return func(ctx context.Context, input *compose.ToolInput) (*compose.ToolOutput, error) {
			var (
				out *compose.ToolOutput
				err error
			)
			for attempt := 0; attempt < maxAttempts; attempt++ {
				out, err = next(ctx, input)
				if err == nil {
					return out, nil
				}
				if !isTransientToolError(err) {
					return nil, err
				}
				log.Printf("tool call transient error (attempt %d/%d): tool=%s error=%s",
					attempt+1, maxAttempts, input.Name, err.Error())
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(backoff(attempt)):
				}
			}
			return nil, fmt.Errorf("tool %s failed after %d attempts: %w", input.Name, maxAttempts, err)
		}
	}

	wrapStream := func(next compose.StreamableToolEndpoint) compose.StreamableToolEndpoint {
		return func(ctx context.Context, input *compose.ToolInput) (*compose.StreamToolOutput, error) {
			var (
				out *compose.StreamToolOutput
				err error
			)
			for attempt := 0; attempt < maxAttempts; attempt++ {
				out, err = next(ctx, input)
				if err == nil {
					return out, nil
				}
				if !isTransientToolError(err) {
					return nil, err
				}
				log.Printf("streaming tool call transient error (attempt %d/%d): tool=%s error=%s",
					attempt+1, maxAttempts, input.Name, err.Error())
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(backoff(attempt)):
				}
			}
			return nil, fmt.Errorf("tool %s failed after %d attempts: %w", input.Name, maxAttempts, err)
		}
	}

	return compose.ToolMiddleware{
		Invokable:  wrap,
		Streamable: wrapStream,
	}
}

// toolErrorRecoveryMiddleware converts tool execution errors into a structured
// error message that the model can read and reason about, instead of letting
// the error propagate up and kill the entire agent run.
//
// Without this middleware, any tool error (e.g. "access denied: path escapes
// workspace") causes the ToolNode to return an error to the graph, which
// terminates the run immediately with a cryptic stack trace shown to the user.
//
// With this middleware, the error is captured and returned to the model as a
// tool result string. The model can then explain the failure, suggest
// alternatives, or ask the user for clarification — exactly like a real
// terminal would print an error and keep the shell alive.
func toolErrorRecoveryMiddleware() compose.ToolMiddleware {
	wrap := func(next compose.InvokableToolEndpoint) compose.InvokableToolEndpoint {
		return func(ctx context.Context, input *compose.ToolInput) (*compose.ToolOutput, error) {
			out, err := next(ctx, input)
			if err == nil {
				return out, nil
			}
			// Convert the error into a model-readable tool result.
			// Prefix with [tool_error] so the model can distinguish it from
			// normal output and respond appropriately.
			msg := fmt.Sprintf("[tool_error] tool=%s error=%s", input.Name, err.Error())
			log.Printf("tool call error recovered as result: tool=%s error=%s", input.Name, err.Error())
			return &compose.ToolOutput{Result: msg}, nil
		}
	}

	// Apply the same recovery logic to streaming tools.
	wrapStream := func(next compose.StreamableToolEndpoint) compose.StreamableToolEndpoint {
		return func(ctx context.Context, input *compose.ToolInput) (*compose.StreamToolOutput, error) {
			out, err := next(ctx, input)
			if err == nil {
				return out, nil
			}
			msg := fmt.Sprintf("[tool_error] tool=%s error=%s", input.Name, err.Error())
			log.Printf("streaming tool call error recovered as result: tool=%s error=%s", input.Name, err.Error())
			sr, sw := schema.Pipe[string](1)
			sw.Send(msg, nil)
			sw.Close()
			return &compose.StreamToolOutput{Result: sr}, nil
		}
	}

	return compose.ToolMiddleware{
		Invokable:  wrap,
		Streamable: wrapStream,
	}
}

// buildSkillPaths assembles the full ordered list of skill directories to load.
// Priority (front = highest): workspace/.claude/skills, ~/.claude/skills, then
// any paths from config. Duplicates are removed while preserving order.
func buildSkillPaths(workspaceRoot string, configPaths []string) []string {
	homeDir, _ := os.UserHomeDir()

	candidates := []string{
		filepath.Join(workspaceRoot, ".claude", "skills"),
		filepath.Join(homeDir, ".claude", "skills"),
	}
	candidates = append(candidates, configPaths...)

	seen := make(map[string]struct{}, len(candidates))
	result := make([]string, 0, len(candidates))
	for _, p := range candidates {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		abs, err := filepath.Abs(p)
		if err != nil {
			continue
		}
		if _, ok := seen[abs]; ok {
			continue
		}
		seen[abs] = struct{}{}
		result = append(result, abs)
	}
	return result
}

func buildReadAllowPathPatterns(paths []string) ([]*regexp.Regexp, error) {
	seen := make(map[string]struct{}, len(paths))
	patterns := make([]*regexp.Regexp, 0, len(paths))
	sep := regexp.QuoteMeta(string(filepath.Separator))

	for _, rawPath := range paths {
		path := strings.TrimSpace(rawPath)
		if path == "" {
			continue
		}
		absPath, err := filepath.Abs(path)
		if err != nil {
			return nil, fmt.Errorf("解析白名单路径失败 %q: %w", rawPath, err)
		}
		absPath = filepath.Clean(absPath)
		if _, ok := seen[absPath]; ok {
			continue
		}
		seen[absPath] = struct{}{}

		expr := "^" + regexp.QuoteMeta(absPath) + "(?:$|" + sep + ")"
		pattern, err := regexp.Compile(expr)
		if err != nil {
			return nil, fmt.Errorf("编译白名单路径失败 %q: %w", absPath, err)
		}
		patterns = append(patterns, pattern)
	}

	return patterns, nil
}

func cloneMessages(messages []*schema.Message) []*schema.Message {
	if len(messages) == 0 {
		return nil
	}
	cloned := make([]*schema.Message, 0, len(messages))
	for _, message := range messages {
		if message == nil {
			continue
		}
		cp := *message
		if len(message.ToolCalls) > 0 {
			cp.ToolCalls = append([]schema.ToolCall(nil), message.ToolCalls...)
		}
		cloned = append(cloned, &cp)
	}
	return cloned
}
