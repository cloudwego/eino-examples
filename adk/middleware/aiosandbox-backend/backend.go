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
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

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

	baseURL := fmt.Sprintf("%s://%s%s", parsedURL.Scheme, parsedURL.Host, parsedURL.Path)

	opts := []option.RequestOption{
		option.WithBaseURL(baseURL),
	}

	if len(parsedURL.RawQuery) > 0 {
		opts = append(opts, option.WithQueryParameters(parsedURL.Query()))
	}

	if cfg.Token != "" {
		authHeader := http.Header{}
		authHeader.Set("Authorization", "Bearer "+cfg.Token)
		opts = append(opts, option.WithHTTPHeader(authHeader))
	}

	c := client.NewClient(opts...)

	return &AIOSandboxBackend{
		config: &cfg,
		client: c,
	}, nil
}

// LsInfo lists file information under the given path.
func (b *AIOSandboxBackend) LsInfo(ctx context.Context, req *filesystem.LsInfoRequest) ([]filesystem.FileInfo, error) {
	path := req.Path
	if path == "" {
		path = b.config.WorkDir
	}

	resp, err := b.client.File.ListPath(ctx, &sandboxsdk.FileListRequest{
		Path:      path,
		Recursive: sandboxsdk.Bool(false),
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
		result = append(result, filesystem.FileInfo{
			Path: f.GetPath(),
		})
	}

	return result, nil
}

// Read reads file content with support for line-based offset and limit.
func (b *AIOSandboxBackend) Read(ctx context.Context, req *filesystem.ReadRequest) (string, error) {
	filePath := b.resolvePath(req.FilePath)

	sdkReq := &sandboxsdk.FileReadRequest{
		File: filePath,
	}

	if req.Offset > 0 {
		sdkReq.StartLine = sandboxsdk.Int(req.Offset)
	}
	if req.Limit > 0 {
		endLine := req.Offset + req.Limit
		sdkReq.EndLine = sandboxsdk.Int(endLine)
	}

	resp, err := b.client.File.ReadFile(ctx, sdkReq)
	if err != nil {
		return "", fmt.Errorf("read file failed: %w", err)
	}

	data := resp.GetData()
	if data == nil {
		return "", fmt.Errorf("empty response data")
	}

	return data.GetContent(), nil
}

// escapeShellArg escapes single quotes for shell arguments: ' -> '\''
func escapeShellArg(s string) string {
	return strings.ReplaceAll(s, "'", "'\\''")
}

// GrepRaw searches for content matching the specified pattern in files.
// Uses ripgrep (rg) for better performance.
func (b *AIOSandboxBackend) GrepRaw(ctx context.Context, req *filesystem.GrepRequest) ([]filesystem.GrepMatch, error) {
	if req.Pattern == "" {
		return nil, nil
	}

	searchPath := req.Path
	if searchPath == "" {
		searchPath = b.config.WorkDir
	}

	// Build rg command
	// -n: show line numbers
	// -F: treat pattern as literal string
	// --no-heading: show file path on each line
	// --null: use \0 as separator to handle : in filenames
	// Output format: filename\0linenum\0content
	pattern := escapeShellArg(req.Pattern)
	searchPath = escapeShellArg(searchPath)

	var cmd string
	if req.Glob != "" {
		glob := escapeShellArg(req.Glob)
		cmd = fmt.Sprintf("rg -n -F --no-heading --null -g '%s' '%s' '%s' 2>/dev/null || true",
			glob, pattern, searchPath)
	} else {
		cmd = fmt.Sprintf("rg -n -F --no-heading --null '%s' '%s' 2>/dev/null || true",
			pattern, searchPath)
	}

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

	// Parse rg --null output: filename\0linenum\0content
	var matches []filesystem.GrepMatch
	lines := strings.Split(*data.Output, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\x00", 3)
		if len(parts) < 3 {
			continue
		}
		lineNum := 0
		fmt.Sscanf(parts[1], "%d", &lineNum)
		matches = append(matches, filesystem.GrepMatch{
			Path:    parts[0],
			Line:    lineNum,
			Content: parts[2],
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
	filePath := b.resolvePath(req.FilePath)

	_, err := b.client.File.WriteFile(ctx, &sandboxsdk.FileWriteRequest{
		File:    filePath,
		Content: req.Content,
	})
	if err != nil {
		return fmt.Errorf("write file failed: %w", err)
	}

	return nil
}

// Edit replaces string occurrences in a file.
func (b *AIOSandboxBackend) Edit(ctx context.Context, req *filesystem.EditRequest) error {
	if req.OldString == "" {
		return fmt.Errorf("old_string cannot be empty")
	}
	if req.OldString == req.NewString {
		return fmt.Errorf("old_string and new_string must be different")
	}

	filePath := b.resolvePath(req.FilePath)

	readResp, err := b.client.File.ReadFile(ctx, &sandboxsdk.FileReadRequest{
		File: filePath,
	})
	if err != nil {
		return fmt.Errorf("read file failed: %w", err)
	}

	data := readResp.GetData()
	if data == nil {
		return fmt.Errorf("empty response data")
	}

	content := data.GetContent()
	count := strings.Count(content, req.OldString)

	if count == 0 {
		return fmt.Errorf("old_string not found in file")
	}

	if !req.ReplaceAll && count > 1 {
		return fmt.Errorf("old_string appears %d times, use replace_all=true to replace all occurrences", count)
	}

	_, err = b.client.File.ReplaceInFile(ctx, &sandboxsdk.FileReplaceRequest{
		File:   filePath,
		OldStr: req.OldString,
		NewStr: req.NewString,
	})
	if err != nil {
		return fmt.Errorf("replace in file failed: %w", err)
	}

	return nil
}

func (b *AIOSandboxBackend) resolvePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(b.config.WorkDir, path)
}

// Execute runs a shell command in the sandbox.
// This implements the filesystem.ShellBackend interface.
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

// Compile-time check that AIOSandboxBackend implements filesystem.ShellBackend interface
var _ filesystem.ShellBackend = (*AIOSandboxBackend)(nil)
