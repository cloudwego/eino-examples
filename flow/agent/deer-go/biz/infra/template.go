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

package infra

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

// GetPromptTemplate 加载并返回一个提示模板
func GetPromptTemplate(ctx context.Context, promptName string) (string, error) {
	// 获取当前文件所在目录
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("获取当前工作目录失败: %w", err)
	}

	// 构建模板文件路径
	templatePath := filepath.Join(dir, "biz", "prompts", fmt.Sprintf("%s.md", promptName))

	// 读取模板文件内容
	content, err := ioutil.ReadFile(templatePath)
	if err != nil {
		return "", fmt.Errorf("读取模板文件 %s 失败: %w", promptName, err)
	}

	return string(content), nil
}
