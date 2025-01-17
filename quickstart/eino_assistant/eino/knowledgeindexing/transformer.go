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

package knowledgeindexing

import (
	"context"

	"github.com/cloudwego/eino-ext/components/document/transformer/splitter/markdown"
	"github.com/cloudwego/eino/components/document"
)

func defaultMarkdownSplitterConfig(ctx context.Context) (*markdown.HeaderConfig, error) {
	config := &markdown.HeaderConfig{
		TrimHeaders: true}
	return config, nil
}

func NewMarkdownSplitter(ctx context.Context, config *markdown.HeaderConfig) (tfr document.Transformer, err error) {
	if config == nil {
		config, err = defaultMarkdownSplitterConfig(ctx)
		if err != nil {
			return nil, err
		}
	}
	tfr, err = markdown.NewHeaderSplitter(ctx, config)
	if err != nil {
		return nil, err
	}
	return tfr, nil
}
