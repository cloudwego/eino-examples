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
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/cloudwego/eino-examples/adk/multiagent/deep/params"
	"github.com/cloudwego/eino-ext/components/tool/commandline"
)

type LocalOperator struct{}

func (l *LocalOperator) ReadFile(ctx context.Context, path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return err.Error(), nil
	}
	return string(b), nil
}

func (l *LocalOperator) WriteFile(ctx context.Context, path string, content string) error {
	return os.WriteFile(path, []byte(content), 0666)
}

func (l *LocalOperator) IsDirectory(ctx context.Context, path string) (bool, error) {
	return true, nil
}

func (l *LocalOperator) Exists(ctx context.Context, path string) (bool, error) {
	return true, nil
}

func (l *LocalOperator) RunCommand(ctx context.Context, command []string) (*commandline.CommandOutput, error) {
	wd, ok := params.GetTypedContextParams[string](ctx, params.WorkDirSessionKey)
	if !ok {
		return nil, fmt.Errorf("work dir not found")
	}

	var shellCmd []string
	switch runtime.GOOS {
	case "windows":
		shellCmd = append([]string{"cmd.exe", "/C"}, command...)
	default:
		shellCmd = []string{"/bin/sh", "-c", strings.Join(command, " ")}
	}

	cmd := exec.CommandContext(ctx, shellCmd[0], shellCmd[1:]...)
	cmd.Dir = wd

	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	cmd.Stdout = outBuf
	cmd.Stderr = errBuf
	err := cmd.Run()
	if err != nil {
		err = fmt.Errorf("internal error:\ncommand: %v\n\nerr: %v\n\nexec error: %v", cmd.String(), err, errBuf.String())
		return nil, err
	}
	return &commandline.CommandOutput{
		Stdout: outBuf.String(),
		Stderr: errBuf.String(),
	}, nil
}
