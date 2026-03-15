/*
 * Copyright 2025 CloudWeGo Authors
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
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	sandboxsdk "github.com/agent-infra/sandbox-sdk-go"
	"github.com/agent-infra/sandbox-sdk-go/client"
	"github.com/agent-infra/sandbox-sdk-go/option"
	"github.com/cloudwego/eino/adk/filesystem"
)

// AIOSandboxBackendConfig is the configuration for AIO Sandbox backend.
type AIOSandboxBackendConfig struct {
	// BaseURL is the AIO Sandbox API endpoint. Required.
	BaseURL string

	// Token is the authentication token for AIO Sandbox API. Optional.
	Token string

	// Headers is additional HTTP headers to include in requests. Optional.
	Headers map[string]string

	// WorkDir is the working directory inside the sandbox. Default: "/tmp"
	WorkDir string
}

func (c *AIOSandboxBackendConfig) setDefaults() {
	if c.WorkDir == "" {
		c.WorkDir = "/tmp"
	}
}

// AIOSandboxBackend implements filesystem.Backend interface using AIO Sandbox API.
type AIOSandboxBackend struct {
	config *AIOSandboxBackendConfig
	client *client.Client
}

// NewAIOSandboxBackend creates a new AIO Sandbox backend that implements filesystem.Backend.
func NewAIOSandboxBackend(ctx context.Context, config *AIOSandboxBackendConfig) (*AIOSandboxBackend, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}
	if config.BaseURL == "" {
		return nil, fmt.Errorf("BaseURL is required")
	}

	cfg := *config
	cfg.setDefaults()

	parsedURL, err := url.Parse(cfg.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid BaseURL: %w", err)
	}

	if parsedURL.Scheme == "" || parsedURL.Host == "" {
		return nil, fmt.Errorf("invalid BaseURL: scheme and host are required")
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, fmt.Errorf("invalid BaseURL: only http and https schemes are supported")
	}

	baseURL := fmt.Sprintf("%s://%s%s", parsedURL.Scheme, parsedURL.Host, strings.TrimRight(parsedURL.Path, "/"))

	opts := []option.RequestOption{
		option.WithBaseURL(baseURL),
	}

	if len(parsedURL.RawQuery) > 0 {
		opts = append(opts, option.WithQueryParameters(parsedURL.Query()))
	}

	customHeader := http.Header{}
	if cfg.Token != "" {
		customHeader.Set("Authorization", "Bearer "+cfg.Token)
	}
	for k, v := range cfg.Headers {
		customHeader.Set(k, v)
	}
	if len(customHeader) > 0 {
		opts = append(opts, option.WithHTTPHeader(customHeader))
	}

	c := client.NewClient(opts...)

	return &AIOSandboxBackend{
		config: &cfg,
		client: c,
	}, nil
}

// LsInfo lists file information under the given path.
func (b *AIOSandboxBackend) LsInfo(ctx context.Context, req *filesystem.LsInfoRequest) ([]filesystem.FileInfo, error) {
	p := req.Path
	if p == "" {
		p = b.config.WorkDir
	}

	resp, err := b.client.File.ListPath(ctx, &sandboxsdk.FileListRequest{
		Path:        p,
		Recursive:   sandboxsdk.Bool(false),
		IncludeSize: sandboxsdk.Bool(true),
	})
	if err != nil {
		return nil, fmt.Errorf("list path failed: %w", err)
	}

	data := resp.GetData()
	if data == nil {
		return nil, fmt.Errorf("empty response data")
	}

	files := data.GetFiles()
	result := make([]filesystem.FileInfo, 0, len(files))
	for _, f := range files {
		info := filesystem.FileInfo{
			Path:  f.GetPath(),
			IsDir: f.GetIsDirectory(),
		}
		if f.Size != nil {
			info.Size = int64(*f.Size)
		}
		if f.ModifiedTime != nil {
			info.ModifiedAt = unixToISO8601(*f.ModifiedTime)
		}
		result = append(result, info)
	}

	return result, nil
}

// Read reads file content with support for line-based offset and limit.
func (b *AIOSandboxBackend) Read(ctx context.Context, req *filesystem.ReadRequest) (*filesystem.FileContent, error) {
	sdkReq := &sandboxsdk.FileReadRequest{
		File: req.FilePath,
	}

	// eino Offset is 1-based, SDK StartLine is 0-based
	if req.Offset > 0 {
		sdkReq.StartLine = sandboxsdk.Int(req.Offset - 1)
	}
	if req.Limit > 0 {
		offset := req.Offset
		if offset < 1 {
			offset = 1
		}
		sdkReq.EndLine = sandboxsdk.Int(offset - 1 + req.Limit)
	}

	resp, err := b.client.File.ReadFile(ctx, sdkReq)
	if err != nil {
		return nil, fmt.Errorf("read file failed: %w", err)
	}

	data := resp.GetData()
	if data == nil {
		return nil, fmt.Errorf("empty response data")
	}

	return &filesystem.FileContent{Content: data.GetContent()}, nil
}

// escapeShellArg escapes single quotes for shell arguments: ' -> '\''
func escapeShellArg(s string) string {
	return strings.ReplaceAll(s, "'", "'\\''")
}

// rgJSONMatch represents the ripgrep JSON output for a match.
type rgJSONMatch struct {
	Type string `json:"type"`
	Data struct {
		Path struct {
			Text string `json:"text"`
		} `json:"path"`
		Lines struct {
			Text string `json:"text"`
		} `json:"lines"`
		LineNumber int `json:"line_number"`
	} `json:"data"`
}

var validFileType = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// buildGrepCommand builds ripgrep command from GrepRequest.
func buildGrepCommand(req *filesystem.GrepRequest, searchPath string) string {
	var args []string
	args = append(args, "rg", "--json")

	if req.CaseInsensitive {
		args = append(args, "-i")
	}
	if req.EnableMultiline {
		args = append(args, "-U", "--multiline-dotall")
	}
	if req.AfterLines > 0 {
		args = append(args, fmt.Sprintf("-A %d", req.AfterLines))
	}
	if req.BeforeLines > 0 {
		args = append(args, fmt.Sprintf("-B %d", req.BeforeLines))
	}
	if req.Glob != "" {
		args = append(args, fmt.Sprintf("-g '%s'", escapeShellArg(req.Glob)))
	}
	if req.FileType != "" && validFileType.MatchString(req.FileType) {
		args = append(args, fmt.Sprintf("-t %s", req.FileType))
	}

	args = append(args, fmt.Sprintf("'%s'", escapeShellArg(req.Pattern)))
	args = append(args, fmt.Sprintf("'%s'", escapeShellArg(searchPath)))

	return strings.Join(args, " ") + " 2>/dev/null || true"
}

// GrepRaw searches for content matching the specified pattern in files.
// Uses ripgrep (rg) with JSON output for reliable parsing.
func (b *AIOSandboxBackend) GrepRaw(ctx context.Context, req *filesystem.GrepRequest) ([]filesystem.GrepMatch, error) {
	if req.Pattern == "" {
		return nil, nil
	}

	searchPath := req.Path
	if searchPath == "" {
		searchPath = b.config.WorkDir
	}

	cmd := buildGrepCommand(req, searchPath)

	resp, err := b.client.Shell.ExecCommand(ctx, &sandboxsdk.ShellExecRequest{
		Command: cmd,
		ExecDir: &b.config.WorkDir,
	})
	if err != nil {
		return nil, fmt.Errorf("rg command failed: %w", err)
	}

	data := resp.GetData()
	if data == nil || data.Output == nil || *data.Output == "" {
		return nil, nil
	}

	// Parse rg --json output: each line is a JSON object
	var matches []filesystem.GrepMatch
	lines := strings.Split(*data.Output, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		var m rgJSONMatch
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			continue
		}
		if m.Type != "match" {
			continue
		}
		matches = append(matches, filesystem.GrepMatch{
			Path:    m.Data.Path.Text,
			Line:    m.Data.LineNumber,
			Content: strings.TrimSuffix(m.Data.Lines.Text, "\n"),
		})
	}

	return matches, nil
}

// GlobInfo returns file information matching the glob pattern.
func (b *AIOSandboxBackend) GlobInfo(ctx context.Context, req *filesystem.GlobInfoRequest) ([]filesystem.FileInfo, error) {
	searchPath := req.Path
	if searchPath == "" {
		searchPath = b.config.WorkDir
	}

	resp, err := b.client.File.FindFiles(ctx, &sandboxsdk.FileFindRequest{
		Path: searchPath,
		Glob: req.Pattern,
	})
	if err != nil {
		return nil, fmt.Errorf("find files failed: %w", err)
	}

	data := resp.GetData()
	if data == nil {
		return nil, fmt.Errorf("empty response data")
	}

	files := data.GetFiles()
	result := make([]filesystem.FileInfo, 0, len(files))
	for _, f := range files {
		result = append(result, filesystem.FileInfo{
			Path: f,
		})
	}

	return result, nil
}

// Write creates or updates file content.
func (b *AIOSandboxBackend) Write(ctx context.Context, req *filesystem.WriteRequest) error {
	_, err := b.client.File.WriteFile(ctx, &sandboxsdk.FileWriteRequest{
		File:    req.FilePath,
		Content: req.Content,
	})
	if err != nil {
		return fmt.Errorf("write file failed: %w", err)
	}

	return nil
}

// Edit replaces string occurrences in a file using the sandbox's str_replace_editor.
func (b *AIOSandboxBackend) Edit(ctx context.Context, req *filesystem.EditRequest) error {
	if req.OldString == "" {
		return fmt.Errorf("old_string cannot be empty")
	}
	if req.OldString == req.NewString {
		return fmt.Errorf("old_string and new_string must be different")
	}

	editorReq := &sandboxsdk.StrReplaceEditorRequest{
		Command: sandboxsdk.CommandStrReplace,
		Path:    req.FilePath,
		OldStr:  &req.OldString,
		NewStr:  &req.NewString,
	}

	if req.ReplaceAll {
		mode := sandboxsdk.StrReplaceEditorRequestReplaceModeAll
		editorReq.ReplaceMode = &mode
	}

	_, err := b.client.File.StrReplaceEditor(ctx, editorReq)
	if err != nil {
		return fmt.Errorf("edit file failed: %w", err)
	}

	return nil
}

// unixToISO8601 converts a unix timestamp string to ISO 8601 format.
// Falls back to returning the original string if parsing fails.
func unixToISO8601(s string) string {
	ts, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return s
	}
	return time.Unix(ts, 0).UTC().Format(time.RFC3339)
}

// Execute runs a shell command in the sandbox.
// This implements the filesystem.Shell interface.
func (b *AIOSandboxBackend) Execute(ctx context.Context, req *filesystem.ExecuteRequest) (*filesystem.ExecuteResponse, error) {
	sdkReq := &sandboxsdk.ShellExecRequest{
		Command: req.Command,
		ExecDir: &b.config.WorkDir,
	}

	resp, err := b.client.Shell.ExecCommand(ctx, sdkReq)
	if err != nil {
		return nil, fmt.Errorf("execute command failed: %w", err)
	}

	data := resp.GetData()
	if data == nil {
		return nil, fmt.Errorf("empty response data")
	}

	var output string
	if data.Output != nil {
		output = *data.Output
	}

	result := &filesystem.ExecuteResponse{
		Output:   output,
		ExitCode: data.ExitCode,
	}

	return result, nil
}

// Compile-time check that AIOSandboxBackend implements filesystem.Backend interface
var _ filesystem.Backend = (*AIOSandboxBackend)(nil)

// Compile-time check that AIOSandboxBackend implements filesystem.Shell interface
var _ filesystem.Shell = (*AIOSandboxBackend)(nil)
