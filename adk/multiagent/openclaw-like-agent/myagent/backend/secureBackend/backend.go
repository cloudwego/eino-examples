// Package secureBackend implements filesystem.Backend and filesystem.StreamingShell
// backed by the workspace-aware fileSystem abstraction from the tools package.
//
// Key properties:
//   - All file paths are sanitized (bare newlines stripped) before use.
//   - When Restrict=true the backend is confined to Workspace via os.Root (sandboxFs).
//   - Extra read-only paths outside Workspace may be whitelisted via AllowPaths regexps.
//   - GrepRaw delegates to the system `rg` binary (ripgrep must be in PATH).
//   - Execute / ExecuteStreaming run commands through /bin/sh; a ValidateCommand hook
//     lets callers deny dangerous commands before execution.
package secureBackend

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"sort"
	"strings"
	"time"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/cloudwego/eino-examples/adk/multiagent/openclaw-like-agent/myagent/backend/secureBackend/fileutil"
	"github.com/cloudwego/eino/adk/filesystem"
	"github.com/cloudwego/eino/schema"
)

// ─────────────────────────────────────────────
// Config
// ─────────────────────────────────────────────

// Config holds construction parameters for SecureBackend.
type Config struct {
	// Workspace is the root directory the backend is allowed to operate in.
	// Required.
	Workspace string

	// Restrict confines all file operations to Workspace when true.
	// Absolute paths outside Workspace are rejected unless they match AllowPaths.
	Restrict bool

	// AllowPaths is an optional list of compiled regexp patterns for paths outside
	// Workspace that are still permitted (e.g. shared read-only config dirs).
	// Ignored when Restrict=false.
	AllowPaths []*regexp.Regexp

	// ValidateCommand is an optional hook called before every shell command.
	// Return a non-nil error to reject the command.
	ValidateCommand func(cmd string) error

	// MaxReadLines caps the number of lines returned by Read.
	// Defaults to 2000 when ≤ 0.
	MaxReadLines int

	// ProtectedPaths is a list of absolute paths that must not be written or
	// edited via the backend.  Any Write/Edit targeting one of these paths will
	// be rejected with an "access denied: protected path" error.  Use this to
	// guard files like MEMORY.md and IDENTITY.md that the agent should only
	// mutate through dedicated store APIs.
	ProtectedPaths []string
}

// ─────────────────────────────────────────────
// SecureBackend
// ─────────────────────────────────────────────

// SecureBackend implements filesystem.Backend and filesystem.StreamingShell.
type SecureBackend struct {
	workspace       string
	fs              fileSystem
	allowPaths      []*regexp.Regexp
	validateCommand func(string) error
	maxReadLines    int
	protectedPaths  map[string]struct{}
}

// New creates a SecureBackend from cfg.
func New(cfg *Config) (*SecureBackend, error) {
	if cfg == nil {
		return nil, errors.New("secureBackend: config is required")
	}

	ws := strings.TrimSpace(cfg.Workspace)
	if ws == "" {
		return nil, errors.New("secureBackend: Workspace must not be empty")
	}

	absWs, err := filepath.Abs(ws)
	if err != nil {
		return nil, fmt.Errorf("secureBackend: cannot resolve Workspace: %w", err)
	}

	validateCmd := cfg.ValidateCommand
	if validateCmd == nil {
		validateCmd = func(string) error { return nil }
	}

	maxLines := cfg.MaxReadLines
	if maxLines <= 0 {
		maxLines = 2000
	}

	protected := make(map[string]struct{}, len(cfg.ProtectedPaths))
	for _, p := range cfg.ProtectedPaths {
		if abs, err := filepath.Abs(p); err == nil {
			protected[abs] = struct{}{}
		}
	}

	return &SecureBackend{
		workspace:       absWs,
		fs:              buildFs(absWs, cfg.Restrict, cfg.AllowPaths),
		allowPaths:      cfg.AllowPaths,
		validateCommand: validateCmd,
		maxReadLines:    maxLines,
		protectedPaths:  protected,
	}, nil
}

// Workspace returns the absolute workspace root.
func (b *SecureBackend) Workspace() string { return b.workspace }

// isProtected reports whether the resolved absolute path is in the protected
// paths set and must not be written or edited.
func (b *SecureBackend) isProtected(resolvedAbs string) bool {
	if len(b.protectedPaths) == 0 {
		return false
	}
	_, ok := b.protectedPaths[filepath.Clean(resolvedAbs)]
	return ok
}

// ─────────────────────────────────────────────
// filesystem.Backend – LsInfo
// ─────────────────────────────────────────────

func (b *SecureBackend) LsInfo(_ context.Context, req *filesystem.LsInfoRequest) ([]filesystem.FileInfo, error) {
	path := sanitizePath(req.Path)
	if path == "" {
		path = b.workspace
	}

	resolved, err := b.resolvePath(path)
	if err != nil {
		return nil, err
	}

	entries, err := b.fs.ReadDir(resolved)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("ls: %w", err)
	}

	files := make([]filesystem.FileInfo, 0, len(entries))
	for _, e := range entries {
		fi := filesystem.FileInfo{
			Path:  e.Name(),
			IsDir: e.IsDir(),
		}
		if info, err2 := e.Info(); err2 == nil {
			fi.Size = info.Size()
			fi.ModifiedAt = info.ModTime().UTC().Format(time.RFC3339)
		}
		files = append(files, fi)
	}
	return files, nil
}

// ─────────────────────────────────────────────
// filesystem.Backend – Read
// ─────────────────────────────────────────────

func (b *SecureBackend) Read(_ context.Context, req *filesystem.ReadRequest) (*filesystem.FileContent, error) {
	path := sanitizePath(req.FilePath)
	resolved, err := b.resolvePath(path)
	if err != nil {
		return nil, err
	}

	f, err := b.fs.Open(resolved)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}
	defer f.Close()

	offset := req.Offset
	if offset <= 0 {
		offset = 1
	}
	limit := req.Limit
	if limit <= 0 || limit > b.maxReadLines {
		limit = b.maxReadLines
	}

	reader := bufio.NewReader(f)
	var sb strings.Builder
	lineNum := 1
	linesRead := 0

	for {
		line, err := reader.ReadString('\n')
		if line != "" {
			if lineNum >= offset {
				sb.WriteString(line)
				linesRead++
				if linesRead >= limit {
					break
				}
			}
			lineNum++
		}
		if err != nil {
			if err != io.EOF {
				return nil, fmt.Errorf("read: error reading file: %w", err)
			}
			break
		}
	}

	return &filesystem.FileContent{Content: strings.TrimSuffix(sb.String(), "\n")}, nil
}

// ─────────────────────────────────────────────
// filesystem.Backend – GrepRaw  (delegates to rg)
// ─────────────────────────────────────────────

type rgJSON struct {
	Type string `json:"type"`
	Data struct {
		Path struct {
			Text string `json:"text"`
		} `json:"path"`
		LineNumber int `json:"line_number"`
		Lines      struct {
			Text string `json:"text"`
		} `json:"lines"`
	} `json:"data"`
}

func (b *SecureBackend) GrepRaw(ctx context.Context, req *filesystem.GrepRequest) ([]filesystem.GrepMatch, error) {
	if req.Pattern == "" {
		return nil, fmt.Errorf("grep: pattern is required")
	}

	searchPath := sanitizePath(req.Path)
	if searchPath == "" {
		searchPath = b.workspace
	} else {
		var err error
		searchPath, err = b.resolvePath(searchPath)
		if err != nil {
			return nil, fmt.Errorf("grep: %w", err)
		}
	}

	args := []string{"--json"}
	if req.CaseInsensitive {
		args = append(args, "-i")
	}
	if req.EnableMultiline {
		args = append(args, "-U", "--multiline-dotall")
	}
	if req.FileType != "" {
		args = append(args, "--type", req.FileType)
	} else if req.Glob != "" {
		args = append(args, "--glob", req.Glob)
	}
	if req.AfterLines > 0 {
		args = append(args, "-A", fmt.Sprintf("%d", req.AfterLines))
	}
	if req.BeforeLines > 0 {
		args = append(args, "-B", fmt.Sprintf("%d", req.BeforeLines))
	}
	args = append(args, "-e", req.Pattern, "--", searchPath)

	cmd := exec.CommandContext(ctx, "rg", args...)
	output, err := cmd.Output()
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return nil, fmt.Errorf("grep: ripgrep (rg) is not installed or not in PATH")
		}
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return []filesystem.GrepMatch{}, nil
		}
		return nil, fmt.Errorf("grep: ripgrep failed: %w", err)
	}

	var matches []filesystem.GrepMatch
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		var row rgJSON
		if jsonErr := json.Unmarshal([]byte(line), &row); jsonErr != nil {
			continue
		}
		if row.Type == "match" || row.Type == "context" {
			matches = append(matches, filesystem.GrepMatch{
				Path:    row.Data.Path.Text,
				Line:    row.Data.LineNumber,
				Content: strings.TrimRight(row.Data.Lines.Text, "\n"),
			})
		}
	}
	return matches, nil
}

// ─────────────────────────────────────────────
// filesystem.Backend – GlobInfo
// ─────────────────────────────────────────────

func (b *SecureBackend) GlobInfo(ctx context.Context, req *filesystem.GlobInfoRequest) ([]filesystem.FileInfo, error) {
	basePath := sanitizePath(req.Path)
	if basePath == "" {
		basePath = b.workspace
	} else {
		var err error
		basePath, err = b.resolvePath(basePath)
		if err != nil {
			return nil, fmt.Errorf("glob: %w", err)
		}
	}

	var relPaths []string
	err := filepath.WalkDir(basePath, func(p string, d fs.DirEntry, err error) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if err != nil {
			if os.IsPermission(err) {
				return filepath.SkipDir
			}
			return err
		}
		rel, relErr := filepath.Rel(basePath, p)
		if relErr != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if rel == "." {
			return nil
		}
		matched, _ := doublestar.Match(req.Pattern, rel)
		if matched {
			relPaths = append(relPaths, rel)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("glob: walk failed: %w", err)
	}

	sort.Strings(relPaths)

	files := make([]filesystem.FileInfo, 0, len(relPaths))
	for _, rel := range relPaths {
		files = append(files, filesystem.FileInfo{Path: rel})
	}
	return files, nil
}

// ─────────────────────────────────────────────
// filesystem.Backend – Write
// ─────────────────────────────────────────────

func (b *SecureBackend) Write(_ context.Context, req *filesystem.WriteRequest) error {
	path := sanitizePath(req.FilePath)
	resolved, err := b.resolveWritePath(path)
	if err != nil {
		return fmt.Errorf("write: %w", err)
	}
	if b.isProtected(resolved) {
		return fmt.Errorf("write: access denied: %s is a protected path and cannot be overwritten directly", req.FilePath)
	}

	if err := os.MkdirAll(filepath.Dir(resolved), 0o755); err != nil {
		return fmt.Errorf("write: cannot create parent directories: %w", err)
	}

	return fileutil.WriteFileAtomic(resolved, []byte(req.Content), 0o644)
}

// ─────────────────────────────────────────────
// filesystem.Backend – Edit
// ─────────────────────────────────────────────

func (b *SecureBackend) Edit(_ context.Context, req *filesystem.EditRequest) error {
	if req.OldString == "" {
		return fmt.Errorf("edit: OldString must not be empty")
	}
	if req.OldString == req.NewString {
		return fmt.Errorf("edit: NewString must differ from OldString")
	}

	path := sanitizePath(req.FilePath)
	resolved, err := b.resolveWritePath(path)
	if err != nil {
		return fmt.Errorf("edit: %w", err)
	}
	if b.isProtected(resolved) {
		return fmt.Errorf("edit: access denied: %s is a protected path and cannot be edited directly", req.FilePath)
	}

	raw, err := b.fs.ReadFile(resolved)
	if err != nil {
		return fmt.Errorf("edit: %w", err)
	}

	text := string(raw)
	count := strings.Count(text, req.OldString)
	if count == 0 {
		return fmt.Errorf("edit: OldString not found in file")
	}
	if count > 1 && !req.ReplaceAll {
		return fmt.Errorf("edit: OldString appears %d times; set ReplaceAll=true to replace all", count)
	}

	var newText string
	if req.ReplaceAll {
		newText = strings.ReplaceAll(text, req.OldString, req.NewString)
	} else {
		newText = strings.Replace(text, req.OldString, req.NewString, 1)
	}

	return fileutil.WriteFileAtomic(resolved, []byte(newText), 0o644)
}

// ─────────────────────────────────────────────
// filesystem.StreamingShell – ExecuteStreaming
// ─────────────────────────────────────────────

func (b *SecureBackend) ExecuteStreaming(ctx context.Context, input *filesystem.ExecuteRequest) (*schema.StreamReader[*filesystem.ExecuteResponse], error) {
	if input.Command == "" {
		return nil, fmt.Errorf("execute: command is required")
	}
	if err := b.validateCommand(input.Command); err != nil {
		return nil, fmt.Errorf("execute: command rejected: %w", err)
	}

	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", input.Command)
	cmd.Dir = b.workspace

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("execute: stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		_ = stdout.Close()
		return nil, fmt.Errorf("execute: stderr pipe: %w", err)
	}

	sr, w := schema.Pipe[*filesystem.ExecuteResponse](100)

	if err := cmd.Start(); err != nil {
		_ = stdout.Close()
		_ = stderr.Close()
		go func() { defer w.Close(); w.Send(nil, fmt.Errorf("execute: failed to start: %w", err)) }()
		return sr, nil
	}

	if input.RunInBackendGround {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					_ = cmd.Process.Kill()
				}
				_ = stdout.Close()
				_ = stderr.Close()
			}()
			done := make(chan struct{})
			go func() {
				drainConcurrently(stdout, stderr)
				_ = cmd.Wait()
				close(done)
			}()
			select {
			case <-done:
			case <-ctx.Done():
				_ = cmd.Process.Kill()
			}
		}()
		go func() {
			defer w.Close()
			w.Send(&filesystem.ExecuteResponse{Output: "command started in background\n", ExitCode: new(int)}, nil)
		}()
		return sr, nil
	}

	go streamOutput(ctx, cmd, stdout, stderr, w)
	return sr, nil
}

// Execute runs a command synchronously and returns its output.
func (b *SecureBackend) Execute(ctx context.Context, input *filesystem.ExecuteRequest) (*filesystem.ExecuteResponse, error) {
	if input.Command == "" {
		return nil, fmt.Errorf("execute: command is required")
	}
	if err := b.validateCommand(input.Command); err != nil {
		return nil, fmt.Errorf("execute: command rejected: %w", err)
	}

	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", input.Command)
	cmd.Dir = b.workspace

	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	exitCode := 0
	if runErr := cmd.Run(); runErr != nil {
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			exitCode = exitErr.ExitCode()
			parts := []string{fmt.Sprintf("command exited with non-zero code %d", exitCode)}
			if s := outBuf.String(); s != "" {
				parts = append(parts, "[stdout]:\n"+s)
			}
			if s := errBuf.String(); s != "" {
				parts = append(parts, "[stderr]:\n"+s)
			}
			return &filesystem.ExecuteResponse{Output: strings.Join(parts, "\n"), ExitCode: &exitCode}, nil
		}
		return nil, fmt.Errorf("execute: %w", runErr)
	}

	return &filesystem.ExecuteResponse{Output: outBuf.String(), ExitCode: &exitCode}, nil
}

// ─────────────────────────────────────────────
// Internal helpers
// ─────────────────────────────────────────────

// resolvePath converts a raw path (absolute or relative) to a clean absolute
// path and, when the backend is in restricted mode, verifies it stays within
// the workspace (or is whitelisted).
//
// Path sanitization (newlines, surrounding whitespace) is the caller's
// responsibility and must be done before calling this method.
func (b *SecureBackend) resolvePath(path string) (string, error) {
	return b.resolvePathForRead(path)
}

func (b *SecureBackend) resolvePathForRead(path string) (string, error) {
	var abs string
	if filepath.IsAbs(path) {
		abs = filepath.Clean(path)
	} else {
		abs = filepath.Clean(filepath.Join(b.workspace, path))
	}
	if b.isWithinWorkspace(abs) || b.matchesAllowPath(abs) {
		return abs, nil
	}
	return "", fmt.Errorf("access denied: path escapes workspace: %s", abs)
}

func (b *SecureBackend) resolveWritePath(path string) (string, error) {
	var abs string
	if filepath.IsAbs(path) {
		abs = filepath.Clean(path)
	} else {
		abs = filepath.Clean(filepath.Join(b.workspace, path))
	}
	if b.isWithinWorkspace(abs) {
		return abs, nil
	}
	return "", fmt.Errorf("access denied: path escapes workspace: %s", abs)
}

func (b *SecureBackend) isWithinWorkspace(abs string) bool {
	rel, err := filepath.Rel(b.workspace, abs)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

func (b *SecureBackend) matchesAllowPath(abs string) bool {
	if len(b.allowPaths) == 0 {
		return false
	}
	abs = filepath.Clean(abs)
	for _, pattern := range b.allowPaths {
		if pattern.MatchString(abs) {
			return true
		}
	}
	return false
}

// sanitizePath removes embedded newlines and surrounding whitespace from a
// path string so that LLM-generated paths with bare \n do not cause failures.
func sanitizePath(path string) string {
	path = strings.ReplaceAll(path, "\r\n", "")
	path = strings.ReplaceAll(path, "\r", "")
	path = strings.ReplaceAll(path, "\n", "")
	return strings.TrimSpace(path)
}

// ─────────────────────────────────────────────
// fileSystem abstraction  (mirrors tools package)
// ─────────────────────────────────────────────

// fileSystem is the minimal interface SecureBackend needs for file I/O.
// It intentionally mirrors the private interface in tools/filesystem.go so both
// packages can evolve independently while sharing the same design pattern.
type fileSystem interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte) error
	ReadDir(path string) ([]os.DirEntry, error)
	Open(path string) (fs.File, error)
}

func buildFs(workspace string, restrict bool, patterns []*regexp.Regexp) fileSystem {
	if !restrict {
		return &hostFs{}
	}
	sb := &sandboxFs{workspace: workspace}
	if len(patterns) > 0 {
		return &whitelistFs{sandbox: sb, patterns: patterns}
	}
	return sb
}

// ── hostFs ──────────────────────────────────

type hostFs struct{}

func (h *hostFs) ReadFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %w", err)
		}
		if os.IsPermission(err) {
			return nil, fmt.Errorf("access denied: %w", err)
		}
		return nil, err
	}
	return data, nil
}

func (h *hostFs) WriteFile(path string, data []byte) error {
	return fileutil.WriteFileAtomic(path, data, 0o644)
}

func (h *hostFs) ReadDir(path string) ([]os.DirEntry, error) {
	return os.ReadDir(path)
}

func (h *hostFs) Open(path string) (fs.File, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %w", err)
		}
		if os.IsPermission(err) {
			return nil, fmt.Errorf("access denied: %w", err)
		}
		return nil, err
	}
	return f, nil
}

// ── sandboxFs ───────────────────────────────

// sandboxFs confines all operations to workspace using os.Root.
type sandboxFs struct {
	workspace string
}

func (s *sandboxFs) safeRelPath(path string) (string, error) {
	rel := filepath.Clean(path)
	if filepath.IsAbs(rel) {
		var err error
		rel, err = filepath.Rel(s.workspace, rel)
		if err != nil {
			return "", fmt.Errorf("cannot relativize path: %w", err)
		}
	}
	if !filepath.IsLocal(rel) {
		return "", fmt.Errorf("access denied: path escapes workspace: %s", path)
	}
	return rel, nil
}

func (s *sandboxFs) withRoot(path string, fn func(*os.Root, string) error) error {
	rel, err := s.safeRelPath(path)
	if err != nil {
		return err
	}
	root, err := os.OpenRoot(s.workspace)
	if err != nil {
		return fmt.Errorf("cannot open workspace root: %w", err)
	}
	defer root.Close()
	return fn(root, rel)
}

func (s *sandboxFs) ReadFile(path string) ([]byte, error) {
	var data []byte
	err := s.withRoot(path, func(root *os.Root, rel string) error {
		f, err := root.Open(rel)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("file not found: %w", err)
			}
			return fmt.Errorf("access denied: %w", err)
		}
		defer f.Close()
		data, err = io.ReadAll(f)
		return err
	})
	return data, err
}

func (s *sandboxFs) WriteFile(path string, data []byte) error {
	rel, err := s.safeRelPath(path)
	if err != nil {
		return err
	}
	absTarget := filepath.Join(s.workspace, rel)

	if dir := filepath.Dir(absTarget); dir != s.workspace {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("cannot create directories: %w", err)
		}
	}

	tmp := absTarget + fmt.Sprintf(".tmp-%d-%d", os.Getpid(), time.Now().UnixNano())
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return fmt.Errorf("cannot create temp file: %w", err)
	}
	if _, err = f.Write(data); err != nil {
		f.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err = f.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err = os.Rename(tmp, absTarget); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func (s *sandboxFs) ReadDir(path string) ([]os.DirEntry, error) {
	var entries []os.DirEntry
	err := s.withRoot(path, func(root *os.Root, rel string) error {
		var err error
		entries, err = fs.ReadDir(root.FS(), rel)
		return err
	})
	return entries, err
}

func (s *sandboxFs) Open(path string) (fs.File, error) {
	rel, err := s.safeRelPath(path)
	if err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(s.workspace)
	if err != nil {
		return nil, fmt.Errorf("cannot open workspace root: %w", err)
	}
	// root.Close() is intentionally deferred to the caller via the returned file.
	// We wrap the file so that closing it also closes the root handle.
	f, err := root.Open(rel)
	if err != nil {
		root.Close()
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %w", err)
		}
		return nil, fmt.Errorf("access denied: %w", err)
	}
	return &rootedFile{File: f, root: root}, nil
}

// rootedFile wraps fs.File and closes the associated os.Root on Close.
type rootedFile struct {
	fs.File
	root *os.Root
}

func (r *rootedFile) Close() error {
	err := r.File.Close()
	_ = r.root.Close()
	return err
}

// ── whitelistFs ─────────────────────────────

// whitelistFs extends sandboxFs with an allowlist of paths outside workspace.
type whitelistFs struct {
	sandbox  *sandboxFs
	patterns []*regexp.Regexp
}

func (w *whitelistFs) matches(path string) bool {
	abs := filepath.Clean(path)
	if !filepath.IsAbs(abs) {
		return false
	}
	for _, p := range w.patterns {
		if p.MatchString(abs) {
			return true
		}
	}
	return false
}

func (w *whitelistFs) ReadFile(path string) ([]byte, error) {
	if w.matches(path) {
		return (&hostFs{}).ReadFile(path)
	}
	return w.sandbox.ReadFile(path)
}

func (w *whitelistFs) WriteFile(path string, data []byte) error {
	if w.matches(path) {
		return fmt.Errorf("access denied: write outside workspace is not allowed: %s", path)
	}
	return w.sandbox.WriteFile(path, data)
}

func (w *whitelistFs) ReadDir(path string) ([]os.DirEntry, error) {
	if w.matches(path) {
		return (&hostFs{}).ReadDir(path)
	}
	return w.sandbox.ReadDir(path)
}

func (w *whitelistFs) Open(path string) (fs.File, error) {
	if w.matches(path) {
		return (&hostFs{}).Open(path)
	}
	return w.sandbox.Open(path)
}

// ─────────────────────────────────────────────
// Shell streaming helpers
// ─────────────────────────────────────────────

func streamOutput(ctx context.Context, cmd *exec.Cmd, stdout, stderr io.ReadCloser, w *schema.StreamWriter[*filesystem.ExecuteResponse]) {
	defer func() {
		if r := recover(); r != nil {
			w.Send(nil, &panicErr{info: r, stack: debug.Stack()})
			return
		}
		w.Close()
	}()

	stderrBytes := make(chan []byte, 1)
	stderrErr := make(chan error, 1)
	go func() {
		data, err := io.ReadAll(stderr)
		stderrBytes <- data
		stderrErr <- err
	}()

	reader := bufio.NewReader(stdout)
	hasOutput := false
	for {
		line, err := reader.ReadString('\n')
		if line != "" {
			hasOutput = true
			select {
			case <-ctx.Done():
				_ = cmd.Process.Kill()
				w.Send(nil, ctx.Err())
				return
			default:
				w.Send(&filesystem.ExecuteResponse{Output: line}, nil)
			}
		}
		if err != nil {
			if err != io.EOF {
				w.Send(nil, fmt.Errorf("execute: read stdout: %w", err))
				return
			}
			break
		}
	}

	if err := <-stderrErr; err != nil {
		w.Send(nil, fmt.Errorf("execute: read stderr: %w", err))
		return
	}
	errData := <-stderrBytes

	if err := cmd.Wait(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			code := exitErr.ExitCode()
			parts := []string{fmt.Sprintf("command exited with non-zero code %d", code)}
			if s := strings.TrimSpace(string(errData)); s != "" {
				parts = append(parts, "[stderr]:\n"+s)
			}
			w.Send(&filesystem.ExecuteResponse{Output: strings.Join(parts, "\n"), ExitCode: &code}, nil)
			return
		}
		w.Send(nil, fmt.Errorf("execute: command failed: %w", err))
		return
	}

	if !hasOutput {
		w.Send(&filesystem.ExecuteResponse{ExitCode: new(int)}, nil)
	}
}

func drainConcurrently(a, b io.Reader) {
	done := make(chan struct{}, 2)
	go func() { _, _ = io.Copy(io.Discard, a); done <- struct{}{} }()
	go func() { _, _ = io.Copy(io.Discard, b); done <- struct{}{} }()
	<-done
	<-done
}

type panicErr struct {
	info  any
	stack []byte
}

func (p *panicErr) Error() string {
	return fmt.Sprintf("panic: %v\nstack:\n%s", p.info, p.stack)
}
